// Package embedded runs an SMQ queue inside another Go process.
//
// A Queue opens the same SQLite database format as the smq server without
// starting a network listener. It suits local durable buffers such as an
// outbox: publish events, read them in batches from an offset, forward them,
// and delete through the acknowledged offset.
//
// All methods are safe for concurrent use. Writes are serialized, so offsets
// within a topic are assigned in commit order. Offsets start at 1, increase
// monotonically, and are never reused, even after deletion or a restart.
//
// Only one process may open a database at a time. Opening the same file from
// a second process, including a standalone smq server, is unsupported.
package embedded

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/internal/queue"
	"github.com/Suhaibinator/SuhaibMessageQueue/internal/server"
	"github.com/Suhaibinator/SuhaibMessageQueue/internal/store/sqlite"
	"google.golang.org/grpc"
)

// Errors returned by Queue. They are the same values used by the server and
// the deprecated top-level errors package, so errors.Is works across them.
var (
	ErrTopicNotFound = queue.ErrTopicNotFound
	ErrInvalidTopic  = queue.ErrInvalidTopic
	ErrTopicEmpty    = queue.ErrTopicIsEmpty
	ErrClosed        = queue.ErrStoreClosed
)

// DefaultReadLimit is the batch size used when ReadOptions.Limit is zero.
const DefaultReadLimit = 100

// Synchronous selects SQLite's durability level. The database always uses
// write-ahead logging (WAL).
type Synchronous int

const (
	// SyncNormal is the default. Committed writes survive a process crash,
	// but the most recent commits can be lost if the machine loses power or
	// the operating system crashes.
	SyncNormal Synchronous = iota
	// SyncFull fsyncs the WAL on every commit, so committed writes also
	// survive power loss, at a higher per-commit cost.
	SyncFull
)

// Options configures Open. The zero value is ready to use.
type Options struct {
	Synchronous Synchronous
	// BusyTimeout bounds how long SQLite waits for a file lock. Zero uses
	// five seconds.
	BusyTimeout time.Duration
}

// Message is a stored message and its offset.
type Message struct {
	Offset int64
	Data   []byte
}

// ReadOptions bounds a Read batch.
type ReadOptions struct {
	// Limit is the maximum number of messages. Zero uses DefaultReadLimit.
	Limit int
	// MaxBytes, when positive, caps the total payload size of the batch. The
	// first available message is always returned, even if it is larger, so a
	// single oversized message cannot stall a reader.
	MaxBytes int64
}

// TopicStats summarizes the messages currently retained by a topic.
type TopicStats struct {
	// Count is the number of retained messages.
	Count int64
	// Bytes is the sum of retained payload sizes.
	Bytes int64
	// EarliestOffset and LatestOffset are zero when the topic is empty.
	EarliestOffset int64
	LatestOffset   int64
	// OldestTime is when the earliest retained message was stored, with
	// second precision in UTC. It is the zero time when the topic is empty.
	OldestTime time.Time
}

// Queue is an open embedded queue.
type Queue struct {
	store  *sqlite.DBDriver
	closed atomic.Bool
}

// Open opens or creates the SQLite database at path. It does not start a
// listener. Close the Queue when done.
func Open(path string, opts Options) (*Queue, error) {
	var synchronous string
	switch opts.Synchronous {
	case SyncNormal:
		synchronous = sqlite.SynchronousNormal
	case SyncFull:
		synchronous = sqlite.SynchronousFull
	default:
		return nil, fmt.Errorf("smq embedded: unknown synchronous mode %d", opts.Synchronous)
	}
	store, err := sqlite.Open(path, sqlite.Options{
		Synchronous: synchronous,
		BusyTimeout: opts.BusyTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("smq embedded: open %s: %w", path, err)
	}
	return &Queue{store: store}, nil
}

func (q *Queue) check(ctx context.Context) error {
	if q.closed.Load() {
		return ErrClosed
	}
	return ctx.Err()
}

// CreateTopic creates topic if it does not already exist. It is idempotent.
// Topic names are 1-35 ASCII letters, digits, or underscores.
func (q *Queue) CreateTopic(ctx context.Context, topic string) error {
	if err := q.check(ctx); err != nil {
		return err
	}
	return q.store.CreateTopic(topic)
}

// Topics returns the existing topic names in lexical order.
func (q *Queue) Topics() []string {
	if q.closed.Load() {
		return nil
	}
	return q.store.Topics()
}

// Publish durably appends data to topic and returns its offset.
func (q *Queue) Publish(ctx context.Context, topic string, data []byte) (int64, error) {
	offsets, err := q.PublishBatch(ctx, topic, [][]byte{data})
	if err != nil {
		return 0, err
	}
	return offsets[0], nil
}

// PublishBatch appends batch to topic in one transaction and returns the
// offset of each message in order. Either every message is stored or none
// is. The offsets are contiguous unless the transaction fails.
func (q *Queue) PublishBatch(ctx context.Context, topic string, batch [][]byte) ([]int64, error) {
	if err := q.check(ctx); err != nil {
		return nil, err
	}
	if len(batch) == 0 {
		return nil, nil
	}
	return q.store.AddMessages(ctx, topic, batch)
}

// Read returns messages with offset >= fromOffset in offset order, bounded by
// opts. An empty result means no message at or after fromOffset exists yet.
// If fromOffset precedes the earliest retained message, reading starts at the
// earliest one. Continue from the last returned offset plus one.
func (q *Queue) Read(ctx context.Context, topic string, fromOffset int64, opts ReadOptions) ([]Message, error) {
	if err := q.check(ctx); err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit == 0 {
		limit = DefaultReadLimit
	}
	if limit < 0 {
		return nil, fmt.Errorf("smq embedded: read limit must not be negative: %d", limit)
	}
	stored, err := q.store.ReadMessages(ctx, topic, fromOffset, limit, opts.MaxBytes)
	if err != nil {
		return nil, err
	}
	messages := make([]Message, len(stored))
	for i, msg := range stored {
		messages[i] = Message{Offset: msg.Offset, Data: msg.Message}
	}
	return messages, nil
}

// EarliestOffset returns the offset of the oldest retained message, or
// ErrTopicEmpty.
func (q *Queue) EarliestOffset(ctx context.Context, topic string) (int64, error) {
	if err := q.check(ctx); err != nil {
		return 0, err
	}
	offset, err := q.store.GetEarliestOffset(topic)
	return emptyOffset(offset, err)
}

// LatestOffset returns the offset of the newest retained message, or
// ErrTopicEmpty.
func (q *Queue) LatestOffset(ctx context.Context, topic string) (int64, error) {
	if err := q.check(ctx); err != nil {
		return 0, err
	}
	offset, err := q.store.GetLatestOffset(topic)
	return emptyOffset(offset, err)
}

func emptyOffset(offset int64, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	return offset, nil
}

// DeleteUntilOffset deletes every message in topic with offset <= offset.
// Deleting an already deleted or not yet written range is not an error.
func (q *Queue) DeleteUntilOffset(ctx context.Context, topic string, offset int64) error {
	if err := q.check(ctx); err != nil {
		return err
	}
	return q.store.DeleteMessagesUntilOffset(topic, offset)
}

// Stats summarizes the messages retained by topic. It scans the topic's index
// and payload lengths, so its cost grows with the number of retained messages.
func (q *Queue) Stats(ctx context.Context, topic string) (TopicStats, error) {
	if err := q.check(ctx); err != nil {
		return TopicStats{}, err
	}
	stats, err := q.store.TopicStats(ctx, topic)
	if err != nil {
		return TopicStats{}, err
	}
	return TopicStats(stats), nil
}

// GRPCOptions configures RegisterGRPC.
type GRPCOptions struct {
	// AllowRemoteWrites permits remote clients to create topics, publish,
	// and delete. When false, remote clients can only read.
	AllowRemoteWrites bool
}

// RegisterGRPC registers the standard SMQ gRPC service, backed by this queue,
// on registrar. Existing SMQ clients can then use the embedded queue. The
// caller owns the gRPC server and listener and must stop the server before
// closing the Queue; calls that arrive after Close fail with Unavailable.
func (q *Queue) RegisterGRPC(registrar grpc.ServiceRegistrar, opts GRPCOptions) {
	server.Register(registrar, q.store, opts.AllowRemoteWrites)
}

// Close waits for in-flight operations and closes the database. Later calls
// return ErrClosed. Close is idempotent.
func (q *Queue) Close() error {
	q.closed.Store(true)
	return q.store.Close()
}
