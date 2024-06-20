package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

type Topic struct {
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

	go func() {
		for {
			message := <-messagesChannel
			d.dbMux.Lock()
			_, err := addMessage.Exec(message)
			d.dbMux.Unlock()
			if err != nil {
				log.Println("Error adding message to topic", err)
			}
		}
	}()

	return &Topic{
		name:                          topic,
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
	}, nil
}

func (t *Topic) getLatestOffset() (int64, error) {
	var offset int64
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getLatestOffsetStmt.QueryRow().Scan(&offset)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the topic is empty
			return 0, nil
		}
		return -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return offset, nil
}

func (t *Topic) getEarliestOffset() (int64, error) {
	var offset int64
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getEarliestOffsetStmt.QueryRow().Scan(&offset)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the topic is empty
			return 0, nil
		}
		return -1, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return offset, nil
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
			return nil, 0, nil
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
			return nil, 0, time, nil
		}
		return nil, -1, time, fmt.Errorf("error retrieving from topic: %v", err)
	}
	return data, offset, time, nil
}

func (t *Topic) getMessageAtOffset(offset int64) ([]byte, error) {
	var data []byte
	t.dbMux.RLock()
	defer t.dbMux.RUnlock()
	err := t.getMessageAtOffsetStmt.QueryRow(offset).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - this means the offset does not exist
			return nil, nil
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
		return fmt.Errorf("error deleting from topic: %v", err)
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
