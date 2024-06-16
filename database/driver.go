package database

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type DBDriver struct {
	db        *sql.DB
	topics    map[string]*Topic
	topicsMux *sync.RWMutex
	dbMux     *sync.RWMutex
}

func NewDBDriver() (*DBDriver, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := DBDriver{
		db:        db,
		topics:    make(map[string]*Topic),
		topicsMux: &sync.RWMutex{},
		dbMux:     &sync.RWMutex{},
	}

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		if strings.HasPrefix(name, "!_") {
			topic, err := result.newTopic(db, strings.TrimPrefix(name, "!_"))
			if err != nil {
				return nil, err
			}
			result.topics[strings.TrimPrefix(name, "!_")] = topic
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &result, nil
}

var validTopic = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func validTopicName(topic string) bool {
	return len(topic) <= 35 && validTopic.MatchString(topic)
}

func (d *DBDriver) CreateTopic(topic string) error {
	if !validTopicName(topic) {
		return fmt.Errorf("invalid topic name: %s", topic)
	}

	d.dbMux.Lock()
	defer d.dbMux.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`CREATE TABLE %s (
		offset INTEGER PRIMARY KEY AUTOINCREMENT,
		time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		data BLOB
	)`, "!_"+topic)

	_, err = tx.Exec(query)
	if err != nil {
		tx.Rollback()
		return err
	}

	newTopic, err := d.newTopic(d.db, topic)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	d.topicsMux.Lock()
	d.topics[topic] = newTopic
	d.topicsMux.Unlock()

	return nil
}

func (d *DBDriver) DeleteTopic(topic string) error {
	if !validTopicName(topic) {
		return fmt.Errorf("invalid topic name: %s", topic)
	}

	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return fmt.Errorf("topic %s does not exist", topic)
	}
	err := topicObj.closeTopic()
	if err != nil {
		return err
	}

	d.topicsMux.Lock()
	delete(d.topics, topic)
	d.topicsMux.Unlock()

	d.dbMux.Lock()
	defer d.dbMux.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DROP TABLE %s", "!_"+topic)

	_, err = tx.Exec(query)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (d *DBDriver) AddMessageToTopic(topic string, data []byte) error {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return fmt.Errorf("topic %s does not exist", topic)
	}

	return topicObj.addMessage(data)
}

func (d *DBDriver) GetEarliestMessageFromTopic(topic string) ([]byte, int64, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return nil, -1, fmt.Errorf("topic %s does not exist", topic)
	}

	message, offset, _, err := topicObj.getEarliestMessage()
	return message, offset, err
}
