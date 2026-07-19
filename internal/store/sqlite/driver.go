// Package sqlite provides the SQLite-backed queue store.
package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/Suhaibinator/SuhaibMessageQueue/internal/queue"
	_ "github.com/mattn/go-sqlite3"
)

type DBDriver struct {
	db        *sql.DB
	topics    map[string]*Topic
	topicsMux *sync.RWMutex
	dbMux     *sync.RWMutex
	closeOnce sync.Once
	closeErr  error
}

func NewDBDriver(dbPath string) (*DBDriver, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	initialized := false
	defer func() {
		if !initialized {
			_ = db.Close()
		}
	}()

	// Set PRAGMA settings

	_, err = db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA cache_size=100000;
		PRAGMA auto_vacuum=FULL;
		`) // Unit is pages, each page is 1024 bytes
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

		if strings.HasPrefix(name, "Topic_") {
			topic, err := result.newTopic(db, name)
			if err != nil {
				return nil, err
			}
			result.topics[strings.TrimPrefix(name, "Topic_")] = topic
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	initialized = true
	return &result, nil
}

var validTopic = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func validTopicName(topic string) bool {
	return len(topic) <= 35 && validTopic.MatchString(topic)
}

func (d *DBDriver) CreateTopic(topic string) error {
	if !validTopicName(topic) {
		return fmt.Errorf("%w: %q", queue.ErrInvalidTopic, topic)
	}
	d.topicsMux.RLock()
	_, exists := d.topics[topic]
	d.topicsMux.RUnlock()
	if exists {
		return nil
	}

	d.dbMux.Lock()
	defer d.dbMux.Unlock()
	d.topicsMux.RLock()
	_, exists = d.topics[topic]
	d.topicsMux.RUnlock()
	if exists {
		return nil
	}

	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		offset INTEGER PRIMARY KEY AUTOINCREMENT,
		time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		data BLOB
	)`, "Topic_"+topic)

	_, err := d.db.Exec(query)
	if err != nil {
		return err
	}

	newTopic, err := d.newTopic(d.db, "Topic_"+topic)
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
		return fmt.Errorf("%w: %q", queue.ErrInvalidTopic, topic)
	}

	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
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

	query := fmt.Sprintf("DROP TABLE %s", "Topic_"+topic)

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
	_, err := d.AddMessage(topic, data)
	return err
}

// AddMessage persists data and returns its assigned topic offset.
func (d *DBDriver) AddMessage(topic string, data []byte) (int64, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return 0, fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}

	return topicObj.addMessageWithOffset(data)
}

func (d *DBDriver) GetEarliestMessageFromTopic(topic string) ([]byte, int64, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return nil, -1, fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}

	message, offset, _, err := topicObj.getEarliestMessage()
	return message, offset, err
}

func (d *DBDriver) GetEarliestOffset(topic string) (int64, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return -1, fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}

	return topicObj.getEarliestOffset()
}

func (d *DBDriver) GetLatestMessageFromTopic(topic string) ([]byte, int64, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return nil, -1, fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}

	message, offset, err := topicObj.getLatestMessage()
	return message, offset, err
}

func (d *DBDriver) GetLatestOffset(topic string) (int64, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return -1, fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}

	return topicObj.getLatestOffset()
}

func (d *DBDriver) GetMessageAtOffset(topic string, offset int64) ([]byte, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}

	return topicObj.getMessageAtOffset(offset)
}

func (d *DBDriver) DeleteMessagesUntilOffset(topic string, offset int64) error {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}

	return topicObj.deleteMessagesUntilOffset(offset)
}

func (d *DBDriver) GetMessagesAfterOffsetWithLimit(topic string, startOffset int64, limit int) ([]queue.Message, int64, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return nil, -1, fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}

	return topicObj.getMessagesAfterOffsetWithLimit(startOffset, limit)
}

// Ensure DBDriver implements the queue storage contract.
var _ queue.Store = (*DBDriver)(nil)

func (d *DBDriver) Close() error {
	d.closeOnce.Do(func() {
		d.topicsMux.RLock()
		defer d.topicsMux.RUnlock()
		for _, topic := range d.topics {
			if err := topic.closeTopic(); err != nil && d.closeErr == nil {
				d.closeErr = err
			}
		}
		if err := d.db.Close(); err != nil && d.closeErr == nil {
			d.closeErr = err
		}
	})
	return d.closeErr
}

func (d *DBDriver) Debug() {

	d.Debug2()

	tables, err := d.db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		log.Fatalf("failed to get tables: %v", err)
	}
	defer tables.Close()

	for tables.Next() {
		var tableName string
		if err := tables.Scan(&tableName); err != nil {
			log.Fatalf("failed to scan table name: %v", err)
		}

		fmt.Printf("Table: %s\n", tableName)

		rows, err := d.db.Query("SELECT * FROM " + tableName)
		if err != nil {
			log.Fatalf("failed to get rows from table %s: %v", tableName, err)
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			log.Fatalf("failed to get columns from table %s: %v", tableName, err)
		}

		fmt.Println("Columns:", strings.Join(cols, ", "))

		for rows.Next() {
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := rows.Scan(columnPointers...); err != nil {
				log.Fatalf("failed to scan row: %v", err)
			}

			for i, colName := range cols {
				val := columnPointers[i].(*interface{})
				fmt.Printf("%s: %v\n", colName, *val)
			}

			fmt.Println()
		}

		if err := rows.Err(); err != nil {
			log.Fatalf("failed to iterate rows: %v", err)
		}
	}

	if err := tables.Err(); err != nil {
		log.Fatalf("failed to iterate tables: %v", err)
	}
}

func (d *DBDriver) Debug2() {

	fmt.Println("Debugging database2")

	debugTables, err := d.db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		log.Fatalf("Error querying sqlite_master: %v", err)
	}
	defer debugTables.Close()

	for debugTables.Next() {
		var debugTableName string
		if err := debugTables.Scan(&debugTableName); err != nil {
			log.Fatalf("Error scanning table name: %v", err)
		}
		fmt.Printf("Debug Table: %s\n", debugTableName)

		// Try a simple SELECT to see if the table is accessible
		testRows, testErr := d.db.Query("SELECT * FROM " + debugTableName + " LIMIT 1")
		if testErr != nil {
			fmt.Printf("Error accessing table %s: %v\n", debugTableName, testErr)
		} else {
			testRows.Close() // make sure to close the rows
		}
	}

	fmt.Println("Debugging database2 complete")
}
