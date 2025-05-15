package database

// Message represents a message retrieved from the database, including its content and offset.
type Message struct {
	Message []byte
	Offset  int64
}

// DBDriverInterface defines the interface for database operations
type DBDriverInterface interface {
	CreateTopic(topic string) error
	AddMessageToTopic(topic string, data []byte) error
	GetMessageAtOffset(topic string, offset int64) ([]byte, error)
	GetEarliestOffset(topic string) (int64, error)
	GetLatestOffset(topic string) (int64, error)
	GetEarliestMessageFromTopic(topic string) ([]byte, int64, error)
	GetLatestMessageFromTopic(topic string) ([]byte, int64, error)
	GetMessagesAfterOffsetWithLimit(topic string, startOffset int64, limit int) ([]Message, int64, error)
	DeleteMessagesUntilOffset(topic string, offset int64) error
	Close()
	Debug()
}

// Ensure DBDriver implements DBDriverInterface
var _ DBDriverInterface = (*DBDriver)(nil)
