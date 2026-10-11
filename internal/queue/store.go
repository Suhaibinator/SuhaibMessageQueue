// Package queue defines the message queue's storage boundary.
package queue

import "time"

// Message represents a message retrieved from the database, including its content and offset.
type Message struct {
	Message []byte
	Offset  int64
}

// TopicStats summarizes the messages currently retained by a topic.
type TopicStats struct {
	// Count is the number of retained messages.
	Count int64
	// Bytes is the sum of retained payload sizes.
	Bytes int64
	// EarliestOffset and LatestOffset are zero when the topic is empty.
	EarliestOffset int64
	LatestOffset   int64
	// OldestTime is the insertion time of the earliest retained message, with
	// second precision in UTC. It is the zero time when the topic is empty.
	OldestTime time.Time
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
