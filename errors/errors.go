package errors

const (
	// ErrTopicNotFound is returned when a topic is not found in the database
	ErrTopicNotFound = "topic not found"
	// ErrTopicAlreadyExists is returned when a topic already exists in the database
	ErrTopicAlreadyExists = "topic already exists"
	// ErrOffsetGreaterThanLatest is returned when an offset is greater than the latest offset in the topic
	ErrOffsetGreaterThanLatest = "offset is greater than the latest offset"
)
