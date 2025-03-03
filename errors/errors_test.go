package errors

import (
	"testing"
)

func TestErrorConstants(t *testing.T) {
	// Test that error constants are defined correctly
	testCases := []struct {
		err     error
		message string
	}{
		{ErrTopicNotFound, ErrTopicNotFoundMessage},
		{ErrTopicAlreadyExists, ErrTopicAlreadyExistsMessage},
		{ErrOffsetGreaterThanLatest, ErrOffsetGreaterThanLatestMessage},
		{ErrNotMessageAtOffset, ErrNotMessageAtOffsetMessage},
		{ErrTopicIsEmpty, ErrTopicIsEmptyMessage},
		{ErrDeletingTopic, ErrDeletingTopicMessage},
	}

	for _, tc := range testCases {
		if tc.err.Error() != tc.message {
			t.Errorf("Expected error message '%s', got '%s'", tc.message, tc.err.Error())
		}
	}
}

func TestErrorMessages(t *testing.T) {
	// Test that error messages are as expected
	if ErrTopicNotFoundMessage != "topic not found" {
		t.Errorf("Expected ErrTopicNotFoundMessage to be 'topic not found', got '%s'", ErrTopicNotFoundMessage)
	}

	if ErrTopicAlreadyExistsMessage != "topic already exists" {
		t.Errorf("Expected ErrTopicAlreadyExistsMessage to be 'topic already exists', got '%s'", ErrTopicAlreadyExistsMessage)
	}

	if ErrOffsetGreaterThanLatestMessage != "offset is greater than the latest offset" {
		t.Errorf("Expected ErrOffsetGreaterThanLatestMessage to be 'offset is greater than the latest offset', got '%s'", ErrOffsetGreaterThanLatestMessage)
	}

	if ErrNotMessageAtOffsetMessage != "no message at offset" {
		t.Errorf("Expected ErrNotMessageAtOffsetMessage to be 'no message at offset', got '%s'", ErrNotMessageAtOffsetMessage)
	}

	if ErrTopicIsEmptyMessage != "topic is empty" {
		t.Errorf("Expected ErrTopicIsEmptyMessage to be 'topic is empty', got '%s'", ErrTopicIsEmptyMessage)
	}

	if ErrDeletingTopicMessage != "error deleting topic" {
		t.Errorf("Expected ErrDeletingTopicMessage to be 'error deleting topic', got '%s'", ErrDeletingTopicMessage)
	}
}
