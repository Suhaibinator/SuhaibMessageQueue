package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/errors"
)

type Topic struct {
	maxOffset                     int64
	name                          string
	db                            *sql.DB
	dbMux                         *sync.RWMutex
	getLatestOffsetStmt           *sql.Stmt
	getEarliestOffsetStmt         *sql.Stmt
	getLatestMessageStmt          *sql.Stmt
	getEarliestMessageStmt        *sql.Stmt
	getMessageAtOffsetStmt        *sql.Stmt
	deleteMessagesUntilOffsetStmt *sql.Stmt
	addMessageStmt                *sql.Stmt
	messagesChannel               chan []byte
	resultChannel                 chan error
}

func (d *DBDriver) newTopic(db *sql.DB, topic string) (*Topic, error) {
	// Assume topic name is valid

	getLatestOffset, err := db.Prepare("SELECT MAX(offset) FROM " + topic)
	if err != nil {
		return nil, err
	}

	getEarliestOffset, err := db.Prepare("SELECT MIN(offset) FROM " + topic)
	if err != nil {
		return nil, err
	}

	getLatestMessage, err := db.Prepare("SELECT data, offset FROM " + topic + " ORDER BY offset DESC LIMIT 1")
	if err != nil {
		return nil, err
	}

	getEarliestMessage, err := db.Prepare("SELECT data, offset, time FROM " + topic + " ORDER BY offset ASC LIMIT 1")
	if err != nil {
		return nil, err
	}

	getMessageAtOffset, err := db.Prepare("SELECT data FROM " + topic + " WHERE offset = ?")
	if err != nil {
		return nil, err
	}

	deleteMessagesUntilOffset, err := db.Prepare("DELETE FROM " + topic + " WHERE offset <= ?")
	if err != nil {
		return nil, err
	}

	addMessage, err := db.Prepare("INSERT INTO " + topic + " (data) VALUES (?)")
	if err != nil {
		return nil, err
	}

	messagesChannel := make(chan []byte)
	resultChannel := make(chan error, 1) // Buffered channel to avoid blocking

	var maxOffset sql.NullInt64
	err = getLatestOffset.QueryRow().Scan(&maxOffset)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the topic is empty
			maxOffset.Int64 = 0
			maxOffset.Valid = true // Explicitly set Valid to true when assigning a value
		} else {
			return nil, fmt.Errorf("error retrieving from topic: %v", err)
		}
	}
	if !maxOffset.Valid {
		maxOffset.Int64 = 0
	}

	// Use maxOffset.Int64 where an int64 is required, checking maxOffset.Valid if necessary

	result := &Topic{
		name:                          topic,
		maxOffset:                     maxOffset.Int64,
		db:                            db,
		dbMux:                         d.dbMux,
		getLatestOffsetStmt:           getLatestOffset,
		getEarliestOffsetStmt:         getEarliestOffset,
		getLatestMessageStmt:          getLatestMessage,
		getEarliestMessageStmt:        getEarliestMessage,
		getMessageAtOffsetStmt:        getMessageAtOffset,
		deleteMessagesUntilOffsetStmt: deleteMessagesUntilOffset,
		addMessageStmt:                addMessage,
		messagesChannel:               messagesChannel,
		resultChannel:                 resultChannel,
	}

	go func(result *Topic) {
		for message := range messagesChannel { // This gracefully exits the loop if the channel is closed.
			d.dbMux.Lock()
			sqlResult, err := addMessage.Exec(message)
			if err != nil {
				log.Printf("Error adding message to topic %s: %v", topic, err)
				result.resultChannel <- err
				d.dbMux.Unlock()
				continue
			}
			maxOffset, err := sqlResult.LastInsertId()
			if err == nil && maxOffset > result.maxOffset { // Corrected condition check order.
				result.maxOffset = maxOffset
			}
			d.dbMux.Unlock()
			if err != nil {
				log.Printf("Error retrieving last insert ID for topic %s: %v", topic, err)
				result.resultChannel <- err
			} else {
				result.resultChannel <- nil // Signal success
			}
		}
	}(result)

	return result, nil
}

func (t *Topic) getLatestOffset() (int64, error) {
	var nullableOffset sql.NullInt64 // Use sql.NullInt64 to handle NULL values
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getLatestOffsetStmt.QueryRow().Scan(&nullableOffset)
	if err != nil {
		return -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	if !nullableOffset.Valid {
		// This means the topic is empty or the offset is NULL
		return 0, errors.ErrTopicIsEmpty
	}
	return nullableOffset.Int64, nil
}

func (t *Topic) getEarliestOffset() (int64, error) {
	var nullableOffset sql.NullInt64 // Use sql.NullInt64 to handle NULL values
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getEarliestOffsetStmt.QueryRow().Scan(&nullableOffset)
	if err != nil {
		return -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	if !nullableOffset.Valid {
		// This means the topic is empty or the offset is NULL
		return 0, errors.ErrTopicIsEmpty
	}
	return nullableOffset.Int64, nil
}

func (t *Topic) getLatestMessage() ([]byte, int64, error) {
	var data []byte
	var offset int64
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getLatestMessageStmt.QueryRow().Scan(&data, &offset)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the topic is empty
			return nil, 0, errors.ErrTopicIsEmpty
		}
		return nil, -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return data, offset, nil
}

func (t *Topic) getEarliestMessage() ([]byte, int64, time.Time, error) {
	var data []byte
	var offset int64
	var time time.Time
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getEarliestMessageStmt.QueryRow().Scan(&data, &offset, &time)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the topic is empty
			return nil, 0, time, errors.ErrTopicIsEmpty
		}
		return nil, -1, time, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return data, offset, time, nil
}

func (t *Topic) getMessageAtOffset(offset int64) ([]byte, error) {
	if offset > t.maxOffset {
		return nil, errors.ErrOffsetGreaterThanLatest
	}

	var data []byte
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getMessageAtOffsetStmt.QueryRow(offset).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the offset does not exist
			return nil, errors.ErrNotMessageAtOffset
		}
		return nil, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return data, nil
}

func (t *Topic) deleteMessagesUntilOffset(offset int64) error {
	t.dbMux.Lock()
	defer t.dbMux.Unlock()
	_, err := t.deleteMessagesUntilOffsetStmt.Exec(offset)
	if err != nil {
		return errors.ErrDeletingTopic
	}
	return nil
}

func (t *Topic) closeTopic() error {
	// Assume topic name is valid

	for _, stmt := range []*sql.Stmt{
		t.getLatestOffsetStmt,
		t.getEarliestOffsetStmt,
		t.getLatestMessageStmt,
		t.getEarliestMessageStmt,
		t.getMessageAtOffsetStmt,
		t.deleteMessagesUntilOffsetStmt,
	} {
		err := stmt.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Topic) addMessage(data []byte) error {
	t.messagesChannel <- data
	// Wait for the result
	err := <-t.resultChannel
	return err
}

func (t *Topic) getMessagesAfterOffsetWithLimit(startOffset int64, limit int) ([]Message, int64, error) {
	var messages []Message
	var nextOffset int64 = startOffset // Default nextOffset if no messages are found

	// If limit is 0 or less, retrieve all messages (SQLite uses -1 for no limit)
	effectiveLimit := limit
	if limit <= 0 {
		effectiveLimit = -1
	}

	query := fmt.Sprintf("SELECT data, offset FROM %s WHERE offset > ? ORDER BY offset ASC LIMIT ?", t.name)

	t.dbMux.RLock()
	defer t.dbMux.RUnlock()

	rows, err := t.db.Query(query, startOffset, effectiveLimit)
	if err != nil {
		return nil, startOffset, fmt.Errorf("error querying messages from topic %s: %v", t.name, err)
	}
	defer rows.Close()

	lastOffset := startOffset
	foundMessages := false

	for rows.Next() {
		var msg Message
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
