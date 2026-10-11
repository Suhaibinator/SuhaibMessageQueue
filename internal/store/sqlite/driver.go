// Package sqlite provides the SQLite-backed queue store.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/internal/queue"
	_ "github.com/mattn/go-sqlite3"
)

// Synchronous values accepted by Options.Synchronous.
const (
	SynchronousNormal = "NORMAL"
	SynchronousFull   = "FULL"
)

const defaultBusyTimeout = 5 * time.Second

// Options tunes how the SQLite database is opened. The zero value matches the
// historical server behavior.
type Options struct {
	// Synchronous is the SQLite synchronous pragma applied to every pooled
	// connection: SynchronousNormal (default) or SynchronousFull.
	Synchronous string
	// BusyTimeout bounds how long SQLite waits on a lock held by another
	// connection or process. Zero uses five seconds.
	BusyTimeout time.Duration
}

type DBDriver struct {
	db        *sql.DB
	topics    map[string]*Topic
	topicsMux *sync.RWMutex
	dbMux     *sync.RWMutex
	closed    bool // guarded by dbMux
	closeOnce sync.Once
	closeErr  error
}

// NewDBDriver opens dbPath with default Options.
func NewDBDriver(dbPath string) (*DBDriver, error) {
	return Open(dbPath, Options{})
}

// Open opens or creates the SQLite database at dbPath and loads its topics.
func Open(dbPath string, opts Options) (*DBDriver, error) {
	dsn, err := buildDSN(dbPath, opts)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	initialized := false
	defer func() {
		if !initialized {
			_ = db.Close()
		}
	}()
	if isMemoryDB(dbPath) {
		// Every pooled connection to a private in-memory database sees a
		// different, empty database. Pin the pool to one connection.
		db.SetMaxOpenConns(1)
	}

	// Journal mode, synchronous, auto_vacuum, and busy_timeout are applied to
	// every pooled connection through the DSN. cache_size is a tuning hint.
	if _, err = db.Exec(`PRAGMA cache_size=100000;`); err != nil {
		return nil, err
	}

	result := DBDriver{
		db:        db,
		topics:    make(map[string]*Topic),
		topicsMux: &sync.RWMutex{},
		dbMux:     &sync.RWMutex{},
	}

	names, err := topicTables(db)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		topic, err := result.newTopic(db, name)
		if err != nil {
			return nil, err
		}
		result.topics[strings.TrimPrefix(name, "Topic_")] = topic
	}

	initialized = true
	return &result, nil
}

func buildDSN(dbPath string, opts Options) (string, error) {
	if dbPath == "" {
		return "", fmt.Errorf("database path must not be empty")
	}
	synchronous := strings.ToUpper(opts.Synchronous)
	switch synchronous {
	case "":
		synchronous = SynchronousNormal
	case SynchronousNormal, SynchronousFull:
	default:
		return "", fmt.Errorf("unsupported synchronous mode %q", opts.Synchronous)
	}
	busyTimeout := opts.BusyTimeout
	if busyTimeout <= 0 {
		busyTimeout = defaultBusyTimeout
	}

	params := url.Values{}
	params.Set("_journal_mode", "WAL")
	params.Set("_synchronous", synchronous)
	params.Set("_auto_vacuum", "FULL")
	params.Set("_busy_timeout", fmt.Sprint(busyTimeout.Milliseconds()))
	separator := "?"
	if strings.Contains(dbPath, "?") {
		separator = "&"
	}
	return dbPath + separator + params.Encode(), nil
}

func isMemoryDB(dbPath string) bool {
	return dbPath == ":memory:" || strings.HasPrefix(dbPath, ":memory:?") ||
		strings.HasPrefix(dbPath, "file::memory:") || strings.Contains(dbPath, "mode=memory")
}

func topicTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if strings.HasPrefix(name, "Topic_") {
			names = append(names, name)
		}
	}
	return names, rows.Err()
}

var validTopic = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func validTopicName(topic string) bool {
	return len(topic) <= 35 && validTopic.MatchString(topic)
}

// CreateTopic creates topic if it does not exist. It is idempotent.
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
	if d.closed {
		return queue.ErrStoreClosed
	}
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

	// Lock order matches CreateTopic: dbMux, then topicsMux.
	d.dbMux.Lock()
	defer d.dbMux.Unlock()
	if d.closed {
		return queue.ErrStoreClosed
	}

	d.topicsMux.Lock()
	topicObj, ok := d.topics[topic]
	if ok {
		delete(d.topics, topic)
	}
	d.topicsMux.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}
	if err := topicObj.closeTopicLocked(); err != nil {
		return err
	}

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

	return tx.Commit()
}

// Topics returns the names of all topics in lexical order.
func (d *DBDriver) Topics() []string {
	d.topicsMux.RLock()
	defer d.topicsMux.RUnlock()
	names := make([]string, 0, len(d.topics))
	for name := range d.topics {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (d *DBDriver) topic(topic string) (*Topic, error) {
	d.topicsMux.RLock()
	topicObj, ok := d.topics[topic]
	d.topicsMux.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", queue.ErrTopicNotFound, topic)
	}
	return topicObj, nil
}

func (d *DBDriver) AddMessageToTopic(topic string, data []byte) error {
	_, err := d.AddMessage(topic, data)
	return err
}

// AddMessage persists data and returns its assigned topic offset.
func (d *DBDriver) AddMessage(topic string, data []byte) (int64, error) {
	offsets, err := d.AddMessages(context.Background(), topic, [][]byte{data})
	if err != nil {
		return 0, err
	}
	return offsets[0], nil
}

// AddMessages persists batch atomically, in order, and returns the offset
// assigned to each message. Either every message is committed or none is.
func (d *DBDriver) AddMessages(ctx context.Context, topic string, batch [][]byte) ([]int64, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return nil, err
	}
	return topicObj.addMessages(ctx, batch)
}

// ReadMessages returns up to limit messages with offset >= fromOffset in
// offset order. When maxBytes is positive, the batch stops before the payload
// total would exceed it, but always includes at least one message.
func (d *DBDriver) ReadMessages(ctx context.Context, topic string, fromOffset int64, limit int, maxBytes int64) ([]queue.Message, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return nil, err
	}
	return topicObj.readMessages(ctx, fromOffset, limit, maxBytes)
}

// TopicStats summarizes the messages currently retained by topic.
func (d *DBDriver) TopicStats(ctx context.Context, topic string) (queue.TopicStats, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return queue.TopicStats{}, err
	}
	return topicObj.stats(ctx)
}

func (d *DBDriver) GetEarliestMessageFromTopic(topic string) ([]byte, int64, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return nil, -1, err
	}

	message, offset, _, err := topicObj.getEarliestMessage()
	return message, offset, err
}

func (d *DBDriver) GetEarliestOffset(topic string) (int64, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return -1, err
	}

	return topicObj.getEarliestOffset()
}

func (d *DBDriver) GetLatestMessageFromTopic(topic string) ([]byte, int64, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return nil, -1, err
	}

	return topicObj.getLatestMessage()
}

func (d *DBDriver) GetLatestOffset(topic string) (int64, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return -1, err
	}

	return topicObj.getLatestOffset()
}

func (d *DBDriver) GetMessageAtOffset(topic string, offset int64) ([]byte, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return nil, err
	}

	return topicObj.getMessageAtOffset(offset)
}

func (d *DBDriver) DeleteMessagesUntilOffset(topic string, offset int64) error {
	topicObj, err := d.topic(topic)
	if err != nil {
		return err
	}

	return topicObj.deleteMessagesUntilOffset(offset)
}

func (d *DBDriver) GetMessagesAfterOffsetWithLimit(topic string, startOffset int64, limit int) ([]queue.Message, int64, error) {
	topicObj, err := d.topic(topic)
	if err != nil {
		return nil, -1, err
	}

	return topicObj.getMessagesAfterOffsetWithLimit(startOffset, limit)
}

// Ensure DBDriver implements the queue storage contract.
var _ queue.Store = (*DBDriver)(nil)

// Close waits for in-flight operations, then closes every topic and the
// database. Later operations return queue.ErrStoreClosed.
func (d *DBDriver) Close() error {
	d.closeOnce.Do(func() {
		d.dbMux.Lock()
		defer d.dbMux.Unlock()
		d.closed = true
		d.topicsMux.RLock()
		for _, topic := range d.topics {
			if err := topic.closeTopicLocked(); err != nil && d.closeErr == nil {
				d.closeErr = err
			}
		}
		d.topicsMux.RUnlock()
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
