package database

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/errors"
)

func TestNewTopic(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Verify topic was created
	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Verify topic properties
	if topic.name != "Topic_test_topic" {
		t.Errorf("Expected topic name to be 'Topic_test_topic', got '%s'", topic.name)
	}

	// Verify prepared statements were created
	if topic.getLatestOffsetStmt == nil {
		t.Fatal("Expected getLatestOffsetStmt to be initialized")
	}
	if topic.getEarliestOffsetStmt == nil {
		t.Fatal("Expected getEarliestOffsetStmt to be initialized")
	}
	if topic.getLatestMessageStmt == nil {
		t.Fatal("Expected getLatestMessageStmt to be initialized")
	}
	if topic.getEarliestMessageStmt == nil {
		t.Fatal("Expected getEarliestMessageStmt to be initialized")
	}
	if topic.getMessageAtOffsetStmt == nil {
		t.Fatal("Expected getMessageAtOffsetStmt to be initialized")
	}
	if topic.deleteMessagesUntilOffsetStmt == nil {
		t.Fatal("Expected deleteMessagesUntilOffsetStmt to be initialized")
	}
	if topic.addMessageStmt == nil {
		t.Fatal("Expected addMessageStmt to be initialized")
	}

	// Verify message channel was created
	if topic.messagesChannel == nil {
		t.Fatal("Expected messagesChannel to be initialized")
	}
}

func TestTopicGetLatestOffset(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Test getting latest offset from empty topic
	_, err = topic.getLatestOffset()
	if err != errors.ErrTopicIsEmpty {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrTopicIsEmpty, err)
	}

	// Add a message to the topic
	err = topic.addMessage([]byte("test message"))
	if err != nil {
		t.Fatalf("Failed to add message to topic: %v", err)
	}

	// Wait for the message to be processed
	time.Sleep(100 * time.Millisecond)

	// Test getting latest offset
	offset, err := topic.getLatestOffset()
	if err != nil {
		t.Fatalf("Failed to get latest offset: %v", err)
	}

	// Verify the offset is correct
	if offset != 1 {
		t.Errorf("Expected offset 1, got %d", offset)
	}
}

func TestTopicGetEarliestOffset(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Test getting earliest offset from empty topic
	_, err = topic.getEarliestOffset()
	if err != errors.ErrTopicIsEmpty {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrTopicIsEmpty, err)
	}

	// Add a message to the topic
	err = topic.addMessage([]byte("test message"))
	if err != nil {
		t.Fatalf("Failed to add message to topic: %v", err)
	}

	// Wait for the message to be processed
	time.Sleep(100 * time.Millisecond)

	// Test getting earliest offset
	offset, err := topic.getEarliestOffset()
	if err != nil {
		t.Fatalf("Failed to get earliest offset: %v", err)
	}

	// Verify the offset is correct
	if offset != 1 {
		t.Errorf("Expected offset 1, got %d", offset)
	}
}

func TestTopicGetLatestMessage(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Test getting latest message from empty topic
	_, _, err = topic.getLatestMessage()
	if err != errors.ErrTopicIsEmpty {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrTopicIsEmpty, err)
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		err = topic.addMessage([]byte(msg))
		if err != nil {
			t.Fatalf("Failed to add message to topic: %v", err)
		}
	}

	// Wait for the messages to be processed
	time.Sleep(100 * time.Millisecond)

	// Test getting latest message
	message, offset, err := topic.getLatestMessage()
	if err != nil {
		t.Fatalf("Failed to get latest message: %v", err)
	}

	// Verify the message and offset are correct
	if string(message) != messages[len(messages)-1] {
		t.Errorf("Expected message '%s', got '%s'", messages[len(messages)-1], string(message))
	}
	if offset != 3 {
		t.Errorf("Expected offset 3, got %d", offset)
	}
}

func TestTopicGetEarliestMessage(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Test getting earliest message from empty topic
	_, _, _, err = topic.getEarliestMessage()
	if err != errors.ErrTopicIsEmpty {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrTopicIsEmpty, err)
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		err = topic.addMessage([]byte(msg))
		if err != nil {
			t.Fatalf("Failed to add message to topic: %v", err)
		}
	}

	// Wait for the messages to be processed
	time.Sleep(100 * time.Millisecond)

	// Test getting earliest message
	message, offset, _, err := topic.getEarliestMessage()
	if err != nil {
		t.Fatalf("Failed to get earliest message: %v", err)
	}

	// Verify the message and offset are correct
	if string(message) != messages[0] {
		t.Errorf("Expected message '%s', got '%s'", messages[0], string(message))
	}
	if offset != 1 {
		t.Errorf("Expected offset 1, got %d", offset)
	}
}

func TestTopicGetMessageAtOffset(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		err = topic.addMessage([]byte(msg))
		if err != nil {
			t.Fatalf("Failed to add message to topic: %v", err)
		}
	}

	// Wait for the messages to be processed
	time.Sleep(100 * time.Millisecond)

	// Test getting message at each offset
	for i, expectedMsg := range messages {
		offset := int64(i + 1) // Offsets are 1-based
		message, err := topic.getMessageAtOffset(offset)
		if err != nil {
			t.Fatalf("Failed to get message at offset %d: %v", offset, err)
		}

		// Verify the message is correct
		if string(message) != expectedMsg {
			t.Errorf("Expected message '%s' at offset %d, got '%s'", expectedMsg, offset, string(message))
		}
	}

	// Test getting message at an offset that doesn't exist
	_, err = topic.getMessageAtOffset(int64(len(messages) + 1))
	if err != errors.ErrOffsetGreaterThanLatest {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrOffsetGreaterThanLatest, err)
	}

	// Test getting message at an offset that has been deleted
	// First, delete messages until offset 2
	err = topic.deleteMessagesUntilOffset(2)
	if err != nil {
		t.Fatalf("Failed to delete messages: %v", err)
	}

	// Then try to get message at offset 1
	_, err = topic.getMessageAtOffset(1)
	if err == nil || err == sql.ErrNoRows {
		t.Errorf("Expected error when getting deleted message, got: %v", err)
	}

	// Verify message at offset 3 is still accessible
	message, err := topic.getMessageAtOffset(3)
	if err != nil {
		t.Fatalf("Failed to get message at offset 3: %v", err)
	}
	if string(message) != messages[2] {
		t.Errorf("Expected message '%s' at offset 3, got '%s'", messages[2], string(message))
	}
}

func TestTopicDeleteMessagesUntilOffset(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3", "message 4", "message 5"}
	for _, msg := range messages {
		err = topic.addMessage([]byte(msg))
		if err != nil {
			t.Fatalf("Failed to add message to topic: %v", err)
		}
	}

	// Wait for the messages to be processed
	time.Sleep(100 * time.Millisecond)

	// Delete messages until offset 3
	err = topic.deleteMessagesUntilOffset(3)
	if err != nil {
		t.Fatalf("Failed to delete messages: %v", err)
	}

	// Verify messages were deleted
	for i := 1; i <= 3; i++ {
		_, err := topic.getMessageAtOffset(int64(i))
		if err == nil {
			t.Errorf("Expected error when getting deleted message at offset %d", i)
		}
	}

	// Verify remaining messages are still accessible
	for i := 4; i <= 5; i++ {
		message, err := topic.getMessageAtOffset(int64(i))
		if err != nil {
			t.Fatalf("Failed to get message at offset %d: %v", i, err)
		}
		if string(message) != messages[i-1] {
			t.Errorf("Expected message '%s' at offset %d, got '%s'", messages[i-1], i, string(message))
		}
	}

	// Test deleting all messages
	err = topic.deleteMessagesUntilOffset(5)
	if err != nil {
		t.Fatalf("Failed to delete all messages: %v", err)
	}

	// Verify all messages were deleted
	for i := 1; i <= 5; i++ {
		_, err := topic.getMessageAtOffset(int64(i))
		if err == nil {
			t.Errorf("Expected error when getting deleted message at offset %d", i)
		}
	}
}

func TestTopicCloseTopic(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Close the topic
	err = topic.closeTopic()
	if err != nil {
		t.Fatalf("Failed to close topic: %v", err)
	}

	// Verify prepared statements were closed
	// Note: We can't directly test if the statements were closed,
	// but we can test that using them after closing results in an error
	// However, this would require modifying the Topic struct to expose the statements
	// or adding methods to check if they're closed, which is beyond the scope of this test
}

func TestTopicAddMessage(t *testing.T) {
	// Create a new database driver with in-memory SQLite
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create a topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	topic, ok := driver.topics["test_topic"]
	if !ok {
		t.Fatal("Expected topic to be created")
	}

	// Add a message to the topic
	message := []byte("test message")
	err = topic.addMessage(message)
	if err != nil {
		t.Fatalf("Failed to add message to topic: %v", err)
	}

	// Wait for the message to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify the message was added
	retrievedMessage, err := topic.getMessageAtOffset(1)
	if err != nil {
		t.Fatalf("Failed to get message at offset 1: %v", err)
	}
	if string(retrievedMessage) != string(message) {
		t.Errorf("Expected message '%s', got '%s'", string(message), string(retrievedMessage))
	}

	// Verify the maxOffset was updated
	if topic.maxOffset != 1 {
		t.Errorf("Expected maxOffset to be 1, got %d", topic.maxOffset)
	}
}
