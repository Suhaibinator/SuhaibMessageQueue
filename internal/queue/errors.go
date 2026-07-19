package queue

import "errors"

const (
	ErrTopicNotFoundMessage                 = "topic not found"
	ErrTopicAlreadyExistsMessage            = "topic already exists"
	ErrOffsetGreaterThanLatestMessage       = "offset is greater than the latest offset"
	ErrNotMessageAtOffsetMessage            = "no message at offset"
	ErrTopicIsEmptyMessage                  = "topic is empty"
	ErrDeletingTopicMessage                 = "error deleting topic"
	ErrRemoteWritesDisabledMessage          = "remote writes disabled"
	ErrInvalidTopicMessage                  = "invalid topic"
	HeartbeatOffset                   int64 = -1
)

var (
	ErrTopicNotFound           = errors.New(ErrTopicNotFoundMessage)
	ErrTopicAlreadyExists      = errors.New(ErrTopicAlreadyExistsMessage)
	ErrOffsetGreaterThanLatest = errors.New(ErrOffsetGreaterThanLatestMessage)
	ErrNotMessageAtOffset      = errors.New(ErrNotMessageAtOffsetMessage)
	ErrTopicIsEmpty            = errors.New(ErrTopicIsEmptyMessage)
	ErrDeletingTopic           = errors.New(ErrDeletingTopicMessage)
	ErrRemoteWritesDisabled    = errors.New(ErrRemoteWritesDisabledMessage)
	ErrInvalidTopic            = errors.New(ErrInvalidTopicMessage)
)
