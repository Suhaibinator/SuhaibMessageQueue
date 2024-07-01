package errors

import "errors"

const (
	// ErrTopicNotFound is returned when a topic is not found in the database
	ErrTopicNotFoundMessage = "topic not found"
	// ErrTopicAlreadyExists is returned when a topic already exists in the database
	ErrTopicAlreadyExistsMessage = "topic already exists"
	// ErrOffsetGreaterThanLatest is returned when an offset is greater than the latest offset in the topic
	ErrOffsetGreaterThanLatestMessage = "offset is greater than the latest offset"
	// ErrNotMessageAtOffset is returned when there is no message at the given offset
	ErrNotMessageAtOffsetMessage = "no message at offset"
	// ErrTopicIsEmpty is returned when a topic is empty
	ErrTopicIsEmptyMessage = "topic is empty"
	// ErrDeletingTopic is returned when a topic cannot be deleted
	ErrDeletingTopicMessage = "error deleting topic"
)

var (
	// ErrTopicNotFound is returned when a topic is not found in the database
	ErrTopicNotFound = errors.New(ErrTopicNotFoundMessage)
	// ErrTopicAlreadyExists is returned when a topic already exists in the database
	ErrTopicAlreadyExists = errors.New(ErrTopicAlreadyExistsMessage)
	// ErrOffsetGreaterThanLatest is returned when an offset is greater than the latest offset in the topic
	ErrOffsetGreaterThanLatest = errors.New(ErrOffsetGreaterThanLatestMessage)
	// ErrNotMessageAtOffset is returned when there is no message at the given offset
	ErrNotMessageAtOffset = errors.New(ErrNotMessageAtOffsetMessage)
	// ErrTopicIsEmpty is returned when a topic is empty
	ErrTopicIsEmpty = errors.New(ErrTopicIsEmptyMessage)
	// ErrDeletingTopic is returned when a topic cannot be deleted
	ErrDeletingTopic = errors.New(ErrDeletingTopicMessage)
)
