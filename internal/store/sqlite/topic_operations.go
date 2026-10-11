package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/internal/queue"
)

// Topic holds the prepared statements for one topic table. Every operation
// runs under the driver's dbMux: writes take the write lock, so offsets are
// assigned and committed in order, and reads share the read lock.
type Topic struct {
	maxOffset                     int64 // guarded by dbMux
	name                          string
	db                            *sql.DB
	dbMux                         *sync.RWMutex
	closed                        bool // guarded by dbMux
	getLatestOffsetStmt           *sql.Stmt
	getEarliestOffsetStmt         *sql.Stmt
	getLatestMessageStmt          *sql.Stmt
	getEarliestMessageStmt        *sql.Stmt
	getMessageAtOffsetStmt        *sql.Stmt
	deleteMessagesUntilOffsetStmt *sql.Stmt
	addMessageStmt                *sql.Stmt
}

func (d *DBDriver) newTopic(db *sql.DB, topic string) (result *Topic, err error) {
	// Assume topic name is valid
	var stmts []*sql.Stmt
	prepare := func(query string) *sql.Stmt {
		if err != nil {
			return nil
		}
		var stmt *sql.Stmt
		stmt, err = db.Prepare(query)
		if err == nil {
			stmts = append(stmts, stmt)
		}
		return stmt
	}
	defer func() {
		if err != nil {
			for _, stmt := range stmts {
				_ = stmt.Close()
			}
		}
	}()

	getLatestOffset := prepare("SELECT MAX(offset) FROM " + topic)
	getEarliestOffset := prepare("SELECT MIN(offset) FROM " + topic)
	getLatestMessage := prepare("SELECT data, offset FROM " + topic + " ORDER BY offset DESC LIMIT 1")
	getEarliestMessage := prepare("SELECT data, offset, time FROM " + topic + " ORDER BY offset ASC LIMIT 1")
	getMessageAtOffset := prepare("SELECT data FROM " + topic + " WHERE offset = ?")
	deleteMessagesUntilOffset := prepare("DELETE FROM " + topic + " WHERE offset <= ?")
	addMessage := prepare("INSERT INTO " + topic + " (data) VALUES (?)")
	if err != nil {
		return nil, err
	}

	var maxOffset sql.NullInt64
	if err = getLatestOffset.QueryRow().Scan(&maxOffset); err != nil {
		return nil, fmt.Errorf("error retrieving from topic: %v", err)
	}

	return &Topic{
		name:                          topic,
		maxOffset:                     maxOffset.Int64, // zero when empty
		db:                            db,
		dbMux:                         d.dbMux,
		getLatestOffsetStmt:           getLatestOffset,
		getEarliestOffsetStmt:         getEarliestOffset,
		getLatestMessageStmt:          getLatestMessage,
		getEarliestMessageStmt:        getEarliestMessage,
		getMessageAtOffsetStmt:        getMessageAtOffset,
		deleteMessagesUntilOffsetStmt: deleteMessagesUntilOffset,
		addMessageStmt:                addMessage,
	}, nil
}

// rlock acquires the shared lock and fails if the topic has been closed.
func (t *Topic) rlock() error {
	t.dbMux.RLock()
	if t.closed {
		t.dbMux.RUnlock()
		return queue.ErrStoreClosed
	}
	return nil
}

// lock acquires the exclusive lock and fails if the topic has been closed.
func (t *Topic) lock() error {
	t.dbMux.Lock()
	if t.closed {
		t.dbMux.Unlock()
		return queue.ErrStoreClosed
	}
	return nil
}

func (t *Topic) getLatestOffset() (int64, error) {
	var nullableOffset sql.NullInt64 // Use sql.NullInt64 to handle NULL values
	if err := t.rlock(); err != nil {
		return -1, err
	}
	defer t.dbMux.RUnlock()
	err := t.getLatestOffsetStmt.QueryRow().Scan(&nullableOffset)
	if err != nil {
		return -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	if !nullableOffset.Valid {
		// This means the topic is empty or the offset is NULL
		return 0, queue.ErrTopicIsEmpty
	}
	return nullableOffset.Int64, nil
}

func (t *Topic) getEarliestOffset() (int64, error) {
	var nullableOffset sql.NullInt64 // Use sql.NullInt64 to handle NULL values
	if err := t.rlock(); err != nil {
		return -1, err
	}
	defer t.dbMux.RUnlock()
	err := t.getEarliestOffsetStmt.QueryRow().Scan(&nullableOffset)
	if err != nil {
		return -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	if !nullableOffset.Valid {
		// This means the topic is empty or the offset is NULL
		return 0, queue.ErrTopicIsEmpty
	}
	return nullableOffset.Int64, nil
}

func (t *Topic) getLatestMessage() ([]byte, int64, error) {
	var data []byte
	var offset int64
	if err := t.rlock(); err != nil {
		return nil, -1, err
	}
	defer t.dbMux.RUnlock()
	err := t.getLatestMessageStmt.QueryRow().Scan(&data, &offset)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the topic is empty
			return nil, 0, queue.ErrTopicIsEmpty
		}
		return nil, -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return data, offset, nil
}

func (t *Topic) getEarliestMessage() ([]byte, int64, time.Time, error) {
	var data []byte
	var offset int64
	var time time.Time
	if err := t.rlock(); err != nil {
		return nil, -1, time, err
	}
	defer t.dbMux.RUnlock()
	err := t.getEarliestMessageStmt.QueryRow().Scan(&data, &offset, &time)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the topic is empty
			return nil, 0, time, queue.ErrTopicIsEmpty
		}
		return nil, -1, time, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return data, offset, time, nil
}

func (t *Topic) getMessageAtOffset(offset int64) ([]byte, error) {
	if err := t.rlock(); err != nil {
		return nil, err
	}
	defer t.dbMux.RUnlock()

	if offset > t.maxOffset {
		return nil, queue.ErrOffsetGreaterThanLatest
	}

	var data []byte
	err := t.getMessageAtOffsetStmt.QueryRow(offset).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the offset does not exist
			return nil, queue.ErrNotMessageAtOffset
		}
		return nil, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return data, nil
}

func (t *Topic) deleteMessagesUntilOffset(offset int64) error {
	if err := t.lock(); err != nil {
		return err
	}
	defer t.dbMux.Unlock()
	_, err := t.deleteMessagesUntilOffsetStmt.Exec(offset)
	if err != nil {
		return queue.ErrDeletingTopic
	}
	return nil
}

// closeTopicLocked closes the topic's statements. The caller must hold the
// dbMux write lock so no operation is using them.
func (t *Topic) closeTopicLocked() error {
	if t.closed {
		return nil
	}
	t.closed = true
	var closeErr error
	for _, stmt := range []*sql.Stmt{
		t.getLatestOffsetStmt,
		t.getEarliestOffsetStmt,
		t.getLatestMessageStmt,
		t.getEarliestMessageStmt,
		t.getMessageAtOffsetStmt,
		t.deleteMessagesUntilOffsetStmt,
		t.addMessageStmt,
	} {
		if err := stmt.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func (t *Topic) closeTopic() error {
	t.dbMux.Lock()
	defer t.dbMux.Unlock()
	return t.closeTopicLocked()
}

func (t *Topic) addMessage(data []byte) error {
	_, err := t.addMessageWithOffset(data)
	return err
}

func (t *Topic) addMessageWithOffset(data []byte) (int64, error) {
	offsets, err := t.addMessages(context.Background(), [][]byte{data})
	if err != nil {
		return 0, err
	}
	return offsets[0], nil
}

// addMessages inserts batch in one transaction and returns the assigned
// offsets. Holding the write lock across the commit keeps offsets ordered and
// makes a message visible to readers only after it is durable.
func (t *Topic) addMessages(ctx context.Context, batch [][]byte) ([]int64, error) {
	if len(batch) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := t.lock(); err != nil {
		return nil, err
	}
	defer t.dbMux.Unlock()

	offsets := make([]int64, 0, len(batch))
	if len(batch) == 1 {
		// A single statement is its own implicit transaction.
		offset, err := insertMessage(ctx, t.addMessageStmt, batch[0])
		if err != nil {
			return nil, fmt.Errorf("add message to topic %s: %w", t.name, err)
		}
		offsets = append(offsets, offset)
	} else {
		tx, err := t.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("begin batch for topic %s: %w", t.name, err)
		}
		stmt := tx.StmtContext(ctx, t.addMessageStmt)
		for _, data := range batch {
			offset, err := insertMessage(ctx, stmt, data)
			if err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("add message to topic %s: %w", t.name, err)
			}
			offsets = append(offsets, offset)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit batch for topic %s: %w", t.name, err)
		}
	}

	if last := offsets[len(offsets)-1]; last > t.maxOffset {
		t.maxOffset = last
	}
	return offsets, nil
}

func insertMessage(ctx context.Context, stmt *sql.Stmt, data []byte) (int64, error) {
	result, err := stmt.ExecContext(ctx, data)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (t *Topic) readMessages(ctx context.Context, fromOffset int64, limit int, maxBytes int64) ([]queue.Message, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("read limit must be positive: %d", limit)
	}
	if err := t.rlock(); err != nil {
		return nil, err
	}
	defer t.dbMux.RUnlock()

	toOffset := int64(-1) // inclusive upper bound; -1 means bounded by limit only
	if maxBytes > 0 {
		var err error
		toOffset, err = t.offsetWithinBytes(ctx, fromOffset, limit, maxBytes)
		if err != nil {
			return nil, err
		}
		if toOffset < fromOffset {
			return nil, nil
		}
	}

	query := "SELECT data, offset FROM " + t.name + " WHERE offset >= ? AND (? < 0 OR offset <= ?) ORDER BY offset ASC LIMIT ?"
	rows, err := t.db.QueryContext(ctx, query, fromOffset, toOffset, toOffset, limit)
	if err != nil {
		return nil, fmt.Errorf("read messages from topic %s: %w", t.name, err)
	}
	defer rows.Close()

	messages := make([]queue.Message, 0, min(limit, 256))
	for rows.Next() {
		var msg queue.Message
		if err := rows.Scan(&msg.Message, &msg.Offset); err != nil {
			return nil, fmt.Errorf("scan message from topic %s: %w", t.name, err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read messages from topic %s: %w", t.name, err)
	}
	return messages, nil
}

// offsetWithinBytes returns the last offset of the longest prefix, starting at
// fromOffset and at most limit messages long, whose payloads fit in maxBytes.
// The first message is always included. It reads lengths, not payloads, and
// returns fromOffset-1 when no message exists.
func (t *Topic) offsetWithinBytes(ctx context.Context, fromOffset int64, limit int, maxBytes int64) (int64, error) {
	query := "SELECT offset, COALESCE(LENGTH(data), 0) FROM " + t.name + " WHERE offset >= ? ORDER BY offset ASC LIMIT ?"
	rows, err := t.db.QueryContext(ctx, query, fromOffset, limit)
	if err != nil {
		return 0, fmt.Errorf("size messages in topic %s: %w", t.name, err)
	}
	defer rows.Close()

	last := fromOffset - 1
	var total int64
	for first := true; rows.Next(); first = false {
		var offset, size int64
		if err := rows.Scan(&offset, &size); err != nil {
			return 0, fmt.Errorf("scan message size from topic %s: %w", t.name, err)
		}
		total += size
		if !first && total > maxBytes {
			break
		}
		last = offset
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("size messages in topic %s: %w", t.name, err)
	}
	return last, nil
}

func (t *Topic) stats(ctx context.Context) (queue.TopicStats, error) {
	if err := t.rlock(); err != nil {
		return queue.TopicStats{}, err
	}
	defer t.dbMux.RUnlock()

	var stats queue.TopicStats
	var earliest, latest sql.NullInt64
	query := "SELECT COUNT(*), COALESCE(SUM(LENGTH(data)), 0), MIN(offset), MAX(offset) FROM " + t.name
	if err := t.db.QueryRowContext(ctx, query).Scan(&stats.Count, &stats.Bytes, &earliest, &latest); err != nil {
		return queue.TopicStats{}, fmt.Errorf("stat topic %s: %w", t.name, err)
	}
	if stats.Count == 0 {
		return stats, nil
	}
	stats.EarliestOffset = earliest.Int64
	stats.LatestOffset = latest.Int64

	var oldest sql.NullTime
	query = "SELECT time FROM " + t.name + " WHERE offset = ?"
	if err := t.db.QueryRowContext(ctx, query, stats.EarliestOffset).Scan(&oldest); err != nil {
		return queue.TopicStats{}, fmt.Errorf("stat topic %s: %w", t.name, err)
	}
	stats.OldestTime = oldest.Time.UTC()
	return stats, nil
}

func (t *Topic) getMessagesAfterOffsetWithLimit(startOffset int64, limit int) ([]queue.Message, int64, error) {
	var messages []queue.Message
	var nextOffset int64 = startOffset // Default nextOffset if no messages are found

	// If limit is 0 or less, retrieve all messages (SQLite uses -1 for no limit)
	effectiveLimit := limit
	if limit <= 0 {
		effectiveLimit = -1
	}

	query := fmt.Sprintf("SELECT data, offset FROM %s WHERE offset > ? ORDER BY offset ASC LIMIT ?", t.name)

	if err := t.rlock(); err != nil {
		return nil, startOffset, err
	}
	defer t.dbMux.RUnlock()

	rows, err := t.db.Query(query, startOffset, effectiveLimit)
	if err != nil {
		return nil, startOffset, fmt.Errorf("error querying messages from topic %s: %v", t.name, err)
	}
	defer rows.Close()

	lastOffset := startOffset
	foundMessages := false

	for rows.Next() {
		var msg queue.Message
		var data []byte // Temporary variable to scan into
		if err := rows.Scan(&data, &msg.Offset); err != nil {
			return nil, startOffset, fmt.Errorf("error scanning message row from topic %s: %v", t.name, err)
		}
		msg.Message = data // Assign scanned data to Message struct
		messages = append(messages, msg)
		lastOffset = msg.Offset
		foundMessages = true
	}

	if err := rows.Err(); err != nil {
		return nil, startOffset, fmt.Errorf("error iterating message rows from topic %s: %v", t.name, err)
	}

	if foundMessages {
		nextOffset = lastOffset + 1
	} else {
		// If no messages were found, nextOffset could be the current latest offset,
		// or simply startOffset + 1 if we want to indicate to try again from the next logical point.
		// For now, let's return the latest known offset of the topic if no messages are found
		// to prevent clients from polling non-existent future offsets indefinitely if startOffset was already high.
		// However, if startOffset is already beyond the latest, then startOffset+1 is fine.
		// The server.go logic might handle this better.
		// Let's stick to lastOffset + 1 if found, otherwise, the initial nextOffset (startOffset) or latest topic offset.
		// For simplicity here, if no messages are found, nextOffset will remain startOffset,
		// or if we want to be more precise, it could be the current latest offset.
		// The current implementation in server.go will use the returned nextOffset.
		// If no messages are found, and startOffset was valid, nextOffset should ideally be startOffset + 1
		// or the latest offset if startOffset is already beyond it.
		// Let's refine: if no messages are found, nextOffset will be the greater of startOffset+1 or current latest offset.
		// For now, if no messages are found, nextOffset will be startOffset. The server can decide.
		// The simplest is: if messages are found, nextOffset = lastOffset + 1.
		// If no messages are found, the initial value of nextOffset (startOffset) is returned.
		// This might lead to re-querying the same startOffset if it's empty.
		// A better approach for "no messages found":
		// nextOffset = current latest offset in topic. If startOffset > latest, then nextOffset = startOffset.
		// Let's use: if foundMessages, nextOffset = lastOffset + 1. Else, nextOffset = startOffset (as initialized).
		// The server can then check if count is 0 and decide.
		// The prompt for BulkRetrieveResponse has `next_offset` as "Offset to use for the next BulkRetrieve request".
		// If no messages are returned, `next_offset` should ideally be the offset from which the *next* attempt should start.
		// If `start_offset` is already the latest, then `next_offset` could be `start_offset` or `start_offset + 1`.
		// Let's make it `startOffset + 1` if no messages are found and `startOffset < t.maxOffset`.
		// If `startOffset >= t.maxOffset` and no messages, then `nextOffset` is `t.maxOffset`.
		// This is getting complex for the DB layer.
		// The server layer (BulkRetrieve in server.go) is better suited to determine the final next_offset for the response.
		// So, this db method will return lastOffset + 1 if messages are found.
		// If no messages are found, it will return the original startOffset.
		// The server.go method will then adjust it.
		// Let's simplify: if messages are found, nextOffset = lastOffset + 1.
		// If no messages are found, nextOffset = startOffset.
		// The server will then set response.NextOffset. If count is 0, it might set it to request.StartOffset or latest.
		// The current server code: `NextOffset: nextOffsetFromDB`.
		// So, if db returns startOffset when no messages, server returns startOffset.
		// This seems reasonable. Client can then decide to wait or retry.
	}

	return messages, nextOffset, nil
}
