package errors

import "fmt"

// ErrTopicNotFound is returned when a topic is not found in the database.
type ErrTopicNotFound struct {
	Message string
}

func (e *ErrTopicNotFound) Error() string {
	return fmt.Sprintf("topic not found: %s", e.Message)
}

// NewErrTopicNotFound creates a new ErrTopicNotFound error.
func NewErrTopicNotFound(message string) error {
	return &ErrTopicNotFound{Message: message}
}

// ErrTopicAlreadyExists is returned when a topic already exists in the database.
type ErrTopicAlreadyExists struct {
	Message string
}

func (e *ErrTopicAlreadyExists) Error() string {
	return fmt.Sprintf("topic already exists: %s", e.Message)
}

// NewErrTopicAlreadyExists creates a new ErrTopicAlreadyExists error.
func NewErrTopicAlreadyExists(message string) error {
	return &ErrTopicAlreadyExists{Message: message}
}

// ErrOffsetGreaterThanLatest is returned when an offset is greater than the latest offset in the topic.
type ErrOffsetGreaterThanLatest struct {
	Message string
}

func (e *ErrOffsetGreaterThanLatest) Error() string {
	return fmt.Sprintf("offset is greater than the latest offset: %s", e.Message)
}

// NewErrOffsetGreaterThanLatest creates a new ErrOffsetGreaterThanLatest error.
func NewErrOffsetGreaterThanLatest(message string) error {
	return &ErrOffsetGreaterThanLatest{Message: message}
}

type ErrorTopicEmpty struct {
	Message string
}

func (e *ErrorTopicEmpty) Error() string {
	return fmt.Sprintf("topic is empty: %s", e.Message)
}

func NewErrorTopicEmpty(message string) error {
	return &ErrorTopicEmpty{Message: message}
}
