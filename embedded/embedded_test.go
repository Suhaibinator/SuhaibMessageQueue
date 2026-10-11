package embedded_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	smqv1 "github.com/Suhaibinator/SuhaibMessageQueue/api/smq/v1"
	"github.com/Suhaibinator/SuhaibMessageQueue/embedded"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func openQueue(t *testing.T, path string) *embedded.Queue {
	t.Helper()
	if path == "" {
		path = filepath.Join(t.TempDir(), "queue.db")
	}
	q, err := embedded.Open(path, embedded.Options{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = q.Close() })
	return q
}

func mustCreateTopic(t *testing.T, q *embedded.Queue, topic string) {
	t.Helper()
	if err := q.CreateTopic(context.Background(), topic); err != nil {
		t.Fatalf("CreateTopic(%q) error = %v", topic, err)
	}
}

func mustPublish(t *testing.T, q *embedded.Queue, topic string, payloads ...string) []int64 {
	t.Helper()
	offsets := make([]int64, 0, len(payloads))
	for _, payload := range payloads {
		offset, err := q.Publish(context.Background(), topic, []byte(payload))
		if err != nil {
			t.Fatalf("Publish(%q) error = %v", payload, err)
		}
		offsets = append(offsets, offset)
	}
	return offsets
}

func mustRead(t *testing.T, q *embedded.Queue, topic string, from int64, opts embedded.ReadOptions) []embedded.Message {
	t.Helper()
	messages, err := q.Read(context.Background(), topic, from, opts)
	if err != nil {
		t.Fatalf("Read(%d, %+v) error = %v", from, opts, err)
	}
	return messages
}

func offsetsOf(messages []embedded.Message) []int64 {
	offsets := make([]int64, len(messages))
	for i, msg := range messages {
		offsets[i] = msg.Offset
	}
	return offsets
}

func equalOffsets(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPublishAndReadBack(t *testing.T) {
	q := openQueue(t, "")
	mustCreateTopic(t, q, "events")

	offsets := mustPublish(t, q, "events", "a", "b", "c")
	if !equalOffsets(offsets, []int64{1, 2, 3}) {
		t.Fatalf("offsets = %v, want [1 2 3]", offsets)
	}

	messages := mustRead(t, q, "events", 1, embedded.ReadOptions{})
	if got := offsetsOf(messages); !equalOffsets(got, []int64{1, 2, 3}) {
		t.Fatalf("read offsets = %v, want [1 2 3]", got)
	}
	for i, want := range []string{"a", "b", "c"} {
		if string(messages[i].Data) != want {
			t.Errorf("message %d = %q, want %q", i, messages[i].Data, want)
		}
	}

	// fromOffset is inclusive.
	messages = mustRead(t, q, "events", 2, embedded.ReadOptions{Limit: 1})
	if len(messages) != 1 || messages[0].Offset != 2 || string(messages[0].Data) != "b" {
		t.Fatalf("Read(2, limit 1) = %+v, want offset 2 %q", messages, "b")
	}

	if got := mustRead(t, q, "events", 4, embedded.ReadOptions{}); len(got) != 0 {
		t.Fatalf("Read past latest = %+v, want empty", got)
	}

	earliest, err := q.EarliestOffset(context.Background(), "events")
	if err != nil || earliest != 1 {
		t.Fatalf("EarliestOffset() = %d, %v; want 1", earliest, err)
	}
	latest, err := q.LatestOffset(context.Background(), "events")
	if err != nil || latest != 3 {
		t.Fatalf("LatestOffset() = %d, %v; want 3", latest, err)
	}
}

func TestPublishBatch(t *testing.T) {
	q := openQueue(t, "")
	mustCreateTopic(t, q, "events")
	ctx := context.Background()

	offsets, err := q.PublishBatch(ctx, "events", [][]byte{[]byte("a"), []byte("b"), []byte("c")})
	if err != nil {
		t.Fatalf("PublishBatch() error = %v", err)
	}
	if !equalOffsets(offsets, []int64{1, 2, 3}) {
		t.Fatalf("offsets = %v, want [1 2 3]", offsets)
	}

	offsets, err = q.PublishBatch(ctx, "events", nil)
	if err != nil || offsets != nil {
		t.Fatalf("PublishBatch(nil) = %v, %v; want nil, nil", offsets, err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := q.PublishBatch(canceled, "events", [][]byte{[]byte("x"), []byte("y")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("PublishBatch(canceled) error = %v, want context.Canceled", err)
	}
	if got := mustRead(t, q, "events", 1, embedded.ReadOptions{}); len(got) != 3 {
		t.Fatalf("stored %d messages after canceled batch, want 3", len(got))
	}
}

func TestTopics(t *testing.T) {
	q := openQueue(t, "")
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := q.CreateTopic(ctx, "events"); err != nil {
			t.Fatalf("CreateTopic() call %d error = %v", i+1, err)
		}
	}
	mustCreateTopic(t, q, "alerts")
	if got := q.Topics(); len(got) != 2 || got[0] != "alerts" || got[1] != "events" {
		t.Fatalf("Topics() = %v, want [alerts events]", got)
	}

	if err := q.CreateTopic(ctx, "bad topic"); !errors.Is(err, embedded.ErrInvalidTopic) {
		t.Fatalf("CreateTopic(invalid) error = %v, want ErrInvalidTopic", err)
	}
	if _, err := q.Publish(ctx, "missing", []byte("x")); !errors.Is(err, embedded.ErrTopicNotFound) {
		t.Fatalf("Publish(missing) error = %v, want ErrTopicNotFound", err)
	}
	if _, err := q.Read(ctx, "missing", 1, embedded.ReadOptions{}); !errors.Is(err, embedded.ErrTopicNotFound) {
		t.Fatalf("Read(missing) error = %v, want ErrTopicNotFound", err)
	}
	if _, err := q.EarliestOffset(ctx, "events"); !errors.Is(err, embedded.ErrTopicEmpty) {
		t.Fatalf("EarliestOffset(empty) error = %v, want ErrTopicEmpty", err)
	}
}

func TestReadLimits(t *testing.T) {
	q := openQueue(t, "")
	mustCreateTopic(t, q, "events")
	for i := 1; i <= 5; i++ {
		mustPublish(t, q, "events", fmt.Sprintf("payload-%02d", i)) // 10 bytes each
	}

	tests := []struct {
		name string
		from int64
		opts embedded.ReadOptions
		want []int64
	}{
		{name: "limit", from: 1, opts: embedded.ReadOptions{Limit: 2}, want: []int64{1, 2}},
		{name: "limit from middle", from: 3, opts: embedded.ReadOptions{Limit: 10}, want: []int64{3, 4, 5}},
		{name: "max bytes", from: 1, opts: embedded.ReadOptions{MaxBytes: 25}, want: []int64{1, 2}},
		{name: "max bytes exact fit", from: 2, opts: embedded.ReadOptions{MaxBytes: 30}, want: []int64{2, 3, 4}},
		{name: "first message always returned", from: 1, opts: embedded.ReadOptions{MaxBytes: 1}, want: []int64{1}},
		{name: "limit tighter than bytes", from: 1, opts: embedded.ReadOptions{Limit: 1, MaxBytes: 1000}, want: []int64{1}},
		{name: "before earliest", from: -10, opts: embedded.ReadOptions{Limit: 1}, want: []int64{1}},
		{name: "past latest with max bytes", from: 6, opts: embedded.ReadOptions{MaxBytes: 100}, want: []int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := offsetsOf(mustRead(t, q, "events", tt.from, tt.opts))
			if !equalOffsets(got, tt.want) {
				t.Fatalf("offsets = %v, want %v", got, tt.want)
			}
		})
	}

	if _, err := q.Read(context.Background(), "events", 1, embedded.ReadOptions{Limit: -1}); err == nil {
		t.Fatal("Read(limit -1) succeeded, want error")
	}
}

func TestReadDefaultLimit(t *testing.T) {
	q := openQueue(t, "")
	mustCreateTopic(t, q, "events")
	batch := make([][]byte, embedded.DefaultReadLimit+5)
	for i := range batch {
		batch[i] = []byte{byte(i)}
	}
	if _, err := q.PublishBatch(context.Background(), "events", batch); err != nil {
		t.Fatalf("PublishBatch() error = %v", err)
	}
	if got := mustRead(t, q, "events", 1, embedded.ReadOptions{}); len(got) != embedded.DefaultReadLimit {
		t.Fatalf("Read() returned %d messages, want %d", len(got), embedded.DefaultReadLimit)
	}
}

func TestDeleteUntilOffset(t *testing.T) {
	q := openQueue(t, "")
	ctx := context.Background()
	mustCreateTopic(t, q, "events")
	mustPublish(t, q, "events", "1", "2", "3", "4", "5")

	if err := q.DeleteUntilOffset(ctx, "events", 3); err != nil {
		t.Fatalf("DeleteUntilOffset(3) error = %v", err)
	}
	if got := offsetsOf(mustRead(t, q, "events", 1, embedded.ReadOptions{})); !equalOffsets(got, []int64{4, 5}) {
		t.Fatalf("offsets after delete = %v, want [4 5]", got)
	}
	if earliest, err := q.EarliestOffset(ctx, "events"); err != nil || earliest != 4 {
		t.Fatalf("EarliestOffset() = %d, %v; want 4", earliest, err)
	}

	// Repeating an acknowledged delete is harmless.
	if err := q.DeleteUntilOffset(ctx, "events", 3); err != nil {
		t.Fatalf("repeated DeleteUntilOffset(3) error = %v", err)
	}

	if err := q.DeleteUntilOffset(ctx, "events", 100); err != nil {
		t.Fatalf("DeleteUntilOffset(100) error = %v", err)
	}
	if _, err := q.LatestOffset(ctx, "events"); !errors.Is(err, embedded.ErrTopicEmpty) {
		t.Fatalf("LatestOffset(empty) error = %v, want ErrTopicEmpty", err)
	}

	// Offsets are never reused after deletion.
	if offsets := mustPublish(t, q, "events", "6"); offsets[0] != 6 {
		t.Fatalf("offset after delete = %d, want 6", offsets[0])
	}
}

func TestReopenKeepsData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.db")
	ctx := context.Background()

	q, err := embedded.Open(path, embedded.Options{Synchronous: embedded.SyncFull})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	mustCreateTopic(t, q, "events")
	mustPublish(t, q, "events", "a", "b", "c")
	if err := q.DeleteUntilOffset(ctx, "events", 1); err != nil {
		t.Fatalf("DeleteUntilOffset() error = %v", err)
	}
	if err := q.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	q = openQueue(t, path)
	if got := q.Topics(); len(got) != 1 || got[0] != "events" {
		t.Fatalf("Topics() after reopen = %v, want [events]", got)
	}
	messages := mustRead(t, q, "events", 1, embedded.ReadOptions{})
	if got := offsetsOf(messages); !equalOffsets(got, []int64{2, 3}) {
		t.Fatalf("offsets after reopen = %v, want [2 3]", got)
	}
	if string(messages[0].Data) != "b" || string(messages[1].Data) != "c" {
		t.Fatalf("data after reopen = %q, %q", messages[0].Data, messages[1].Data)
	}

	// Delete everything, reopen, and confirm offsets continue.
	if err := q.DeleteUntilOffset(ctx, "events", 3); err != nil {
		t.Fatalf("DeleteUntilOffset() error = %v", err)
	}
	if err := q.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	q = openQueue(t, path)
	if offsets := mustPublish(t, q, "events", "d"); offsets[0] != 4 {
		t.Fatalf("offset after emptying and reopening = %d, want 4", offsets[0])
	}
}

func TestConcurrentPublishers(t *testing.T) {
	q := openQueue(t, "")
	mustCreateTopic(t, q, "events")
	ctx := context.Background()

	const writers = 8
	const perWriter = 50
	type published struct {
		offset int64
		data   []byte
	}
	results := make([][]published, writers)
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWriter; {
				// Alternate single publishes with small batches.
				if i%10 == 0 && i+3 <= perWriter {
					batch := [][]byte{
						[]byte(fmt.Sprintf("w%d-%d", w, i)),
						[]byte(fmt.Sprintf("w%d-%d", w, i+1)),
						[]byte(fmt.Sprintf("w%d-%d", w, i+2)),
					}
					offsets, err := q.PublishBatch(ctx, "events", batch)
					if err != nil {
						errs <- err
						return
					}
					for j := range batch {
						results[w] = append(results[w], published{offsets[j], batch[j]})
					}
					i += 3
					continue
				}
				data := []byte(fmt.Sprintf("w%d-%d", w, i))
				offset, err := q.Publish(ctx, "events", data)
				if err != nil {
					errs <- err
					return
				}
				results[w] = append(results[w], published{offset, data})
				i++
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("publish error = %v", err)
	}

	stored := make(map[int64][]byte)
	for _, msg := range mustRead(t, q, "events", 1, embedded.ReadOptions{Limit: writers * perWriter * 2}) {
		stored[msg.Offset] = msg.Data
	}
	if len(stored) != writers*perWriter {
		t.Fatalf("stored %d messages, want %d", len(stored), writers*perWriter)
	}
	seen := make(map[int64]bool)
	for w, list := range results {
		var previous int64
		for _, p := range list {
			if seen[p.offset] {
				t.Fatalf("offset %d returned twice", p.offset)
			}
			seen[p.offset] = true
			if p.offset <= previous {
				t.Fatalf("writer %d offsets not increasing: %d after %d", w, p.offset, previous)
			}
			previous = p.offset
			if !bytes.Equal(stored[p.offset], p.data) {
				t.Fatalf("offset %d holds %q, but Publish returned it for %q", p.offset, stored[p.offset], p.data)
			}
		}
	}
}

func TestStats(t *testing.T) {
	q := openQueue(t, "")
	ctx := context.Background()
	mustCreateTopic(t, q, "events")

	stats, err := q.Stats(ctx, "events")
	if err != nil {
		t.Fatalf("Stats(empty) error = %v", err)
	}
	if stats != (embedded.TopicStats{}) {
		t.Fatalf("Stats(empty) = %+v, want zero", stats)
	}

	before := time.Now().UTC().Add(-2 * time.Second)
	mustPublish(t, q, "events", "abc", "de", "f")
	if err := q.DeleteUntilOffset(ctx, "events", 1); err != nil {
		t.Fatalf("DeleteUntilOffset() error = %v", err)
	}
	stats, err = q.Stats(ctx, "events")
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Count != 2 || stats.Bytes != 3 || stats.EarliestOffset != 2 || stats.LatestOffset != 3 {
		t.Fatalf("Stats() = %+v, want count 2, bytes 3, offsets 2-3", stats)
	}
	if stats.OldestTime.Before(before) || stats.OldestTime.After(time.Now().Add(2*time.Second)) {
		t.Fatalf("Stats().OldestTime = %v, want about now", stats.OldestTime)
	}
	if _, err := q.Stats(ctx, "missing"); !errors.Is(err, embedded.ErrTopicNotFound) {
		t.Fatalf("Stats(missing) error = %v, want ErrTopicNotFound", err)
	}
}

func TestClose(t *testing.T) {
	q, err := embedded.Open(filepath.Join(t.TempDir(), "queue.db"), embedded.Options{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	mustCreateTopic(t, q, "events")
	ctx := context.Background()

	// Publishers racing Close must fail cleanly rather than panic.
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if _, err := q.Publish(ctx, "events", []byte("x")); err != nil {
					if !errors.Is(err, embedded.ErrClosed) {
						t.Errorf("Publish() during Close error = %v, want ErrClosed", err)
					}
					return
				}
			}
		}()
	}
	time.Sleep(20 * time.Millisecond)
	if err := q.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	wg.Wait()

	if err := q.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if err := q.CreateTopic(ctx, "events"); !errors.Is(err, embedded.ErrClosed) {
		t.Fatalf("CreateTopic() after Close error = %v, want ErrClosed", err)
	}
	if _, err := q.Read(ctx, "events", 1, embedded.ReadOptions{}); !errors.Is(err, embedded.ErrClosed) {
		t.Fatalf("Read() after Close error = %v, want ErrClosed", err)
	}
	if _, err := q.Stats(ctx, "events"); !errors.Is(err, embedded.ErrClosed) {
		t.Fatalf("Stats() after Close error = %v, want ErrClosed", err)
	}
}

func TestOpenRejectsInvalidOptions(t *testing.T) {
	if _, err := embedded.Open(filepath.Join(t.TempDir(), "queue.db"), embedded.Options{Synchronous: 99}); err == nil {
		t.Fatal("Open(invalid synchronous) succeeded, want error")
	}
	if _, err := embedded.Open("", embedded.Options{}); err == nil {
		t.Fatal("Open(empty path) succeeded, want error")
	}
}

func TestInMemoryQueue(t *testing.T) {
	q := openQueue(t, ":memory:")
	mustCreateTopic(t, q, "events")
	if _, err := q.PublishBatch(context.Background(), "events", [][]byte{[]byte("a"), []byte("b")}); err != nil {
		t.Fatalf("PublishBatch() error = %v", err)
	}
	if got := mustRead(t, q, "events", 1, embedded.ReadOptions{MaxBytes: 10}); len(got) != 2 {
		t.Fatalf("Read() returned %d messages, want 2", len(got))
	}
}

func TestRegisterGRPC(t *testing.T) {
	for _, tt := range []struct {
		name        string
		allowWrites bool
	}{
		{name: "read only", allowWrites: false},
		{name: "remote writes", allowWrites: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			q := openQueue(t, "")
			mustCreateTopic(t, q, "events")
			mustPublish(t, q, "events", "local")

			listener := bufconn.Listen(1 << 20)
			grpcServer := grpc.NewServer()
			q.RegisterGRPC(grpcServer, embedded.GRPCOptions{AllowRemoteWrites: tt.allowWrites})
			go func() { _ = grpcServer.Serve(listener) }()
			t.Cleanup(grpcServer.Stop)

			conn, err := grpc.NewClient("passthrough:///bufnet",
				grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
					return listener.DialContext(ctx)
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				t.Fatalf("grpc.NewClient() error = %v", err)
			}
			t.Cleanup(func() { _ = conn.Close() })
			client := smqv1.NewSuhaibMessageQueueClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			bulk, err := client.BulkRetrieve(ctx, &smqv1.BulkRetrieveRequest{Topic: "events", StartOffset: 0, Limit: 10})
			if err != nil {
				t.Fatalf("BulkRetrieve() error = %v", err)
			}
			if len(bulk.Messages) != 1 || string(bulk.Messages[0].Message) != "local" {
				t.Fatalf("BulkRetrieve() = %+v, want the locally published message", bulk.Messages)
			}

			produced, err := client.Produce(ctx, &smqv1.ProduceRequest{Topic: "events", Message: []byte("remote")})
			if !tt.allowWrites {
				if status.Code(err) != codes.PermissionDenied {
					t.Fatalf("Produce() error = %v, want PermissionDenied", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Produce() error = %v", err)
			}
			messages := mustRead(t, q, "events", produced.Offset, embedded.ReadOptions{})
			if len(messages) != 1 || string(messages[0].Data) != "remote" {
				t.Fatalf("Read() of remote message = %+v", messages)
			}
		})
	}
}
