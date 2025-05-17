package database

import (
	"testing"

	"github.com/Suhaibinator/SuhaibMessageQueue/errors"
)

func TestNewDBDriver(t *testing.T) {
	// Use in-memory SQLite database for testing
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Verify driver was created successfully
	if driver == nil {
		t.Fatal("Expected driver to be non-nil")
	}

	// Verify topics map was initialized
	if driver.topics == nil {
		t.Fatal("Expected topics map to be initialized")
	}

	// Verify mutexes were initialized
	if driver.topicsMux == nil {
		t.Fatal("Expected topicsMux to be initialized")
	}
	if driver.dbMux == nil {
		t.Fatal("Expected dbMux to be initialized")
	}
}

func TestCreateTopic(t *testing.T) {
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Test creating a valid topic
	err = driver.CreateTopic("test_topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Verify topic was created
	if _, ok := driver.topics["test_topic"]; !ok {
		t.Fatal("Expected topic to be created")
	}

	// Test creating a topic with invalid name
	err = driver.CreateTopic("invalid topic name")
	if err == nil {
		t.Fatal("Expected error when creating topic with invalid name")
	}

	// Test creating a topic that already exists
	err = driver.CreateTopic("test_topic")
	// This should not return an error as the CreateTopic method uses CREATE TABLE IF NOT EXISTS
	if err != nil {
		t.Fatalf("Unexpected error when creating existing topic: %v", err)
	}
}

func TestAddMessageToTopic(t *testing.T) {
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

	// Test adding a message to the topic
	message := []byte("test message")
	err = driver.AddMessageToTopic("test_topic", message)
	if err != nil {
		t.Fatalf("Failed to add message to topic: %v", err)
	}

	// Test adding a message to a non-existent topic
	err = driver.AddMessageToTopic("non_existent_topic", message)
	if err == nil {
		t.Fatal("Expected error when adding message to non-existent topic")
	}
}

func TestGetMessageAtOffset(t *testing.T) {
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

	// Add a message to the topic
	message := []byte("test message")
	err = driver.AddMessageToTopic("test_topic", message)
	if err != nil {
		t.Fatalf("Failed to add message to topic: %v", err)
	}

	// Test getting the message at offset 1
	retrievedMessage, err := driver.GetMessageAtOffset("test_topic", 1)
	if err != nil {
		t.Fatalf("Failed to get message at offset: %v", err)
	}

	// Verify the message is correct
	if string(retrievedMessage) != string(message) {
		t.Errorf("Expected message '%s', got '%s'", string(message), string(retrievedMessage))
	}

	// Test getting a message at an offset that doesn't exist
	_, err = driver.GetMessageAtOffset("test_topic", 2)
	if err != errors.ErrOffsetGreaterThanLatest {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrOffsetGreaterThanLatest, err)
	}

	// Test getting a message from a non-existent topic
	_, err = driver.GetMessageAtOffset("non_existent_topic", 1)
	if err == nil {
		t.Fatal("Expected error when getting message from non-existent topic")
	}
}

func TestGetEarliestOffset(t *testing.T) {
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

	// Test getting earliest offset from empty topic
	_, err = driver.GetEarliestOffset("test_topic")
	if err != errors.ErrTopicIsEmpty {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrTopicIsEmpty, err)
	}

	// Add a message to the topic
	message := []byte("test message")
	err = driver.AddMessageToTopic("test_topic", message)
	if err != nil {
		t.Fatalf("Failed to add message to topic: %v", err)
	}

	// Test getting earliest offset
	offset, err := driver.GetEarliestOffset("test_topic")
	if err != nil {
		t.Fatalf("Failed to get earliest offset: %v", err)
	}

	// Verify the offset is correct
	if offset != 1 {
		t.Errorf("Expected offset 1, got %d", offset)
	}

	// Test getting earliest offset from a non-existent topic
	_, err = driver.GetEarliestOffset("non_existent_topic")
	if err == nil {
		t.Fatal("Expected error when getting earliest offset from non-existent topic")
	}
}

func TestGetLatestOffset(t *testing.T) {
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

	// Test getting latest offset from empty topic
	_, err = driver.GetLatestOffset("test_topic")
	if err != errors.ErrTopicIsEmpty {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrTopicIsEmpty, err)
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		err = driver.AddMessageToTopic("test_topic", []byte(msg))
		if err != nil {
			t.Fatalf("Failed to add message to topic: %v", err)
		}
	}

	// Test getting latest offset
	offset, err := driver.GetLatestOffset("test_topic")
	if err != nil {
		t.Fatalf("Failed to get latest offset: %v", err)
	}

	// Verify the offset is correct
	if offset != 3 {
		t.Errorf("Expected offset 3, got %d", offset)
	}

	// Test getting latest offset from a non-existent topic
	_, err = driver.GetLatestOffset("non_existent_topic")
	if err == nil {
		t.Fatal("Expected error when getting latest offset from non-existent topic")
	}
}

func TestGetEarliestMessageFromTopic(t *testing.T) {
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

	// Test getting earliest message from empty topic
	_, _, err = driver.GetEarliestMessageFromTopic("test_topic")
	if err != errors.ErrTopicIsEmpty {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrTopicIsEmpty, err)
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		err = driver.AddMessageToTopic("test_topic", []byte(msg))
		if err != nil {
			t.Fatalf("Failed to add message to topic: %v", err)
		}
	}

	// Test getting earliest message
	message, offset, err := driver.GetEarliestMessageFromTopic("test_topic")
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

	// Test getting earliest message from a non-existent topic
	_, _, err = driver.GetEarliestMessageFromTopic("non_existent_topic")
	if err == nil {
		t.Fatal("Expected error when getting earliest message from non-existent topic")
	}
}

func TestGetLatestMessageFromTopic(t *testing.T) {
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

	// Test getting latest message from empty topic
	_, _, err = driver.GetLatestMessageFromTopic("test_topic")
	if err != errors.ErrTopicIsEmpty {
		t.Errorf("Expected error '%v', got '%v'", errors.ErrTopicIsEmpty, err)
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		err = driver.AddMessageToTopic("test_topic", []byte(msg))
		if err != nil {
			t.Fatalf("Failed to add message to topic: %v", err)
		}
	}

	// Test getting latest message
	message, offset, err := driver.GetLatestMessageFromTopic("test_topic")
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

	// Test getting latest message from a non-existent topic
	_, _, err = driver.GetLatestMessageFromTopic("non_existent_topic")
	if err == nil {
		t.Fatal("Expected error when getting latest message from non-existent topic")
	}
}

func TestDeleteMessagesUntilOffset(t *testing.T) {
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

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3", "message 4", "message 5"}
	for _, msg := range messages {
		err = driver.AddMessageToTopic("test_topic", []byte(msg))
		if err != nil {
			t.Fatalf("Failed to add message to topic: %v", err)
		}
	}

	// Delete messages until offset 3
	err = driver.DeleteMessagesUntilOffset("test_topic", 3)
	if err != nil {
		t.Fatalf("Failed to delete messages: %v", err)
	}

	// Verify messages were deleted
	message, err := driver.GetMessageAtOffset("test_topic", 1)
	if err == nil {
		t.Errorf("Expected error when getting deleted message, got message: %s", string(message))
	}

	message, err = driver.GetMessageAtOffset("test_topic", 2)
	if err == nil {
		t.Errorf("Expected error when getting deleted message, got message: %s", string(message))
	}

	message, err = driver.GetMessageAtOffset("test_topic", 3)
	if err == nil {
		t.Errorf("Expected error when getting deleted message, got message: %s", string(message))
	}

	// Verify remaining messages are still accessible
	message, err = driver.GetMessageAtOffset("test_topic", 4)
	if err != nil {
		t.Fatalf("Failed to get message at offset 4: %v", err)
	}
	if string(message) != messages[3] {
		t.Errorf("Expected message '%s', got '%s'", messages[3], string(message))
	}

	message, err = driver.GetMessageAtOffset("test_topic", 5)
	if err != nil {
		t.Fatalf("Failed to get message at offset 5: %v", err)
	}
	if string(message) != messages[4] {
		t.Errorf("Expected message '%s', got '%s'", messages[4], string(message))
	}

	// Test deleting messages from a non-existent topic
	err = driver.DeleteMessagesUntilOffset("non_existent_topic", 1)
	if err == nil {
		t.Fatal("Expected error when deleting messages from non-existent topic")
	}
}

func TestMultipleTopics(t *testing.T) {
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database driver: %v", err)
	}
	defer driver.Close()

	// Create multiple topics
	topics := []string{"topic1", "topic2", "topic3"}
	for _, topic := range topics {
		err = driver.CreateTopic(topic)
		if err != nil {
			t.Fatalf("Failed to create topic %s: %v", topic, err)
		}
	}

	// Add messages to each topic
	for _, topic := range topics {
		for j := 0; j < 3; j++ {
			message := []byte(topic + " message " + string(rune('A'+j)))
			err = driver.AddMessageToTopic(topic, message)
			if err != nil {
				t.Fatalf("Failed to add message to topic %s: %v", topic, err)
			}
		}
	}

	// Verify messages in each topic
	for _, topic := range topics {
		// Get latest offset
		offset, err := driver.GetLatestOffset(topic)
		if err != nil {
			t.Fatalf("Failed to get latest offset for topic %s: %v", topic, err)
		}
		if offset != 3 {
			t.Errorf("Expected latest offset 3 for topic %s, got %d", topic, offset)
		}

		// Get earliest message
		message, offset, err := driver.GetEarliestMessageFromTopic(topic)
		if err != nil {
			t.Fatalf("Failed to get earliest message for topic %s: %v", topic, err)
		}
		expectedMessage := topic + " message A"
		if string(message) != expectedMessage {
			t.Errorf("Expected earliest message '%s' for topic %s, got '%s'", expectedMessage, topic, string(message))
		}
		if offset != 1 {
			t.Errorf("Expected earliest offset 1 for topic %s, got %d", topic, offset)
		}

		// Get latest message
		message, offset, err = driver.GetLatestMessageFromTopic(topic)
		if err != nil {
			t.Fatalf("Failed to get latest message for topic %s: %v", topic, err)
		}
		expectedMessage = topic + " message C"
		if string(message) != expectedMessage {
			t.Errorf("Expected latest message '%s' for topic %s, got '%s'", expectedMessage, topic, string(message))
		}
		if offset != 3 {
			t.Errorf("Expected latest offset 3 for topic %s, got %d", topic, offset)
		}
	}
}
