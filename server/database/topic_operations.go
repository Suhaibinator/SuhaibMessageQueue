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

	var maxOffset int64
	err = getLatestOffset.QueryRow().Scan(&maxOffset)
	if err != nil {
		if err == sql.ErrNoRows || err.Error() == "sql: Scan error on column index 0, name \"MAX(offset)\": converting NULL to int64 is unsupported" {
			// No rows were returned - this means the topic is empty
			maxOffset = 0
		} else {
			return nil, fmt.Errorf("error retrieving from topic: %v", err)
		}
	}

	result := &Topic{
		name:                          topic,
		maxOffset:                     maxOffset,
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
	}

	go func(result *Topic) {
		for message := range messagesChannel { // This gracefully exits the loop if the channel is closed.
			d.dbMux.Lock()
			sqlResult, err := addMessage.Exec(message)
			if err != nil {
				log.Printf("Error adding message to topic %s: %v", topic, err)
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
			}
		}
	}(result)

	return result, nil
}

func (t *Topic) getLatestOffset() (int64, error) {
	var offset int64
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getLatestOffsetStmt.QueryRow().Scan(&offset)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the topic is empty
			return 0, errors.ErrTopicIsEmpty
		}
		return -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return offset, nil
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
	return nil
}
