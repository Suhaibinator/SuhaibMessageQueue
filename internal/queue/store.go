// Package queue defines the message queue's storage boundary.
package queue

// Message represents a message retrieved from the database, including its content and offset.
type Message struct {
	Message []byte
	Offset  int64
}

// Store defines the persistence operations required by the queue service.
// Implementations are responsible for serializing concurrent mutations.
type Store interface {
	CreateTopic(topic string) error
	AddMessage(topic string, data []byte) (int64, error)
	GetMessageAtOffset(topic string, offset int64) ([]byte, error)
	GetEarliestOffset(topic string) (int64, error)
	GetLatestOffset(topic string) (int64, error)
	GetEarliestMessageFromTopic(topic string) ([]byte, int64, error)
	GetLatestMessageFromTopic(topic string) ([]byte, int64, error)
	GetMessagesAfterOffsetWithLimit(topic string, startOffset int64, limit int) ([]Message, int64, error)
	DeleteMessagesUntilOffset(topic string, offset int64) error
	Close() error
}
