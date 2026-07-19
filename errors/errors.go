// Package errors preserves the v1 public error values.
//
// Deprecated: new server code uses the internal queue domain errors.
package errors

import "github.com/Suhaibinator/SuhaibMessageQueue/internal/queue"

const (
	// ErrTopicNotFound is returned when a topic is not found in the database
	ErrTopicNotFoundMessage = queue.ErrTopicNotFoundMessage
	// ErrTopicAlreadyExists is returned when a topic already exists in the database
	ErrTopicAlreadyExistsMessage = queue.ErrTopicAlreadyExistsMessage
	// ErrOffsetGreaterThanLatest is returned when an offset is greater than the latest offset in the topic
	ErrOffsetGreaterThanLatestMessage = queue.ErrOffsetGreaterThanLatestMessage
	// ErrNotMessageAtOffset is returned when there is no message at the given offset
	ErrNotMessageAtOffsetMessage = queue.ErrNotMessageAtOffsetMessage
	// ErrTopicIsEmpty is returned when a topic is empty
	ErrTopicIsEmptyMessage = queue.ErrTopicIsEmptyMessage
	// ErrDeletingTopic is returned when a topic cannot be deleted
	ErrDeletingTopicMessage = queue.ErrDeletingTopicMessage
	// ErrRemoteWritesDisabled is returned when remote writes are disabled
	ErrRemoteWritesDisabledMessage = queue.ErrRemoteWritesDisabledMessage
)

var (
	// ErrTopicNotFound is returned when a topic is not found in the database
	ErrTopicNotFound = queue.ErrTopicNotFound
	// ErrTopicAlreadyExists is returned when a topic already exists in the database
	ErrTopicAlreadyExists = queue.ErrTopicAlreadyExists
	// ErrOffsetGreaterThanLatest is returned when an offset is greater than the latest offset in the topic
	ErrOffsetGreaterThanLatest = queue.ErrOffsetGreaterThanLatest
	// ErrNotMessageAtOffset is returned when there is no message at the given offset
	ErrNotMessageAtOffset = queue.ErrNotMessageAtOffset
	// ErrTopicIsEmpty is returned when a topic is empty
	ErrTopicIsEmpty = queue.ErrTopicIsEmpty
	// ErrDeletingTopic is returned when a topic cannot be deleted
	ErrDeletingTopic = queue.ErrDeletingTopic
	// ErrRemoteWritesDisabled is returned when remote write operations are not allowed
	ErrRemoteWritesDisabled = queue.ErrRemoteWritesDisabled
)
