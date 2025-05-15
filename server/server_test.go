package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/config"
	"github.com/Suhaibinator/SuhaibMessageQueue/errors"
	pb "github.com/Suhaibinator/SuhaibMessageQueue/proto"
	"github.com/Suhaibinator/SuhaibMessageQueue/server/database"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// Ensure MockDBDriver implements database.DBDriverInterface
var _ database.DBDriverInterface = (*MockDBDriver)(nil)

// MockDBDriver is a mock implementation of the database driver for testing
type MockDBDriver struct {
	topics                              map[string]bool
	messages                            map[string]map[int64][]byte
	latestOffsets                       map[string]int64
	createTopicFunc                     func(string) error
	addMessageToTopicFunc               func(string, []byte) error
	getMessageAtOffsetFunc              func(string, int64) ([]byte, error)
	getEarliestOffsetFunc               func(string) (int64, error)
	getLatestOffsetFunc                 func(string) (int64, error)
	getEarliestMessageFunc              func(string) ([]byte, int64, error)
	getLatestMessageFunc                func(string) ([]byte, int64, error)
	deleteMessagesUntilOffsetFunc       func(string, int64) error
	getMessagesAfterOffsetWithLimitFunc func(topic string, startOffset int64, limit int) ([]database.Message, int64, error)
}

func NewMockDBDriver() *MockDBDriver {
	return &MockDBDriver{
		topics:        make(map[string]bool),
		messages:      make(map[string]map[int64][]byte),
		latestOffsets: make(map[string]int64),
	}
}

func (m *MockDBDriver) CreateTopic(topic string) error {
	if m.createTopicFunc != nil {
		return m.createTopicFunc(topic)
	}
	if _, ok := m.topics[topic]; ok {
		return errors.ErrTopicAlreadyExists
	}
	m.topics[topic] = true
	m.messages[topic] = make(map[int64][]byte)
	m.latestOffsets[topic] = 0
	return nil
}

func (m *MockDBDriver) AddMessageToTopic(topic string, data []byte) error {
	if m.addMessageToTopicFunc != nil {
		return m.addMessageToTopicFunc(topic, data)
	}
	if _, ok := m.topics[topic]; !ok {
		return errors.ErrTopicNotFound
	}
	m.latestOffsets[topic]++
	m.messages[topic][m.latestOffsets[topic]] = data
	return nil
}

func (m *MockDBDriver) GetMessageAtOffset(topic string, offset int64) ([]byte, error) {
	if m.getMessageAtOffsetFunc != nil {
		return m.getMessageAtOffsetFunc(topic, offset)
	}
	if _, ok := m.topics[topic]; !ok {
		return nil, errors.ErrTopicNotFound
	}
	if offset > m.latestOffsets[topic] {
		return nil, errors.ErrOffsetGreaterThanLatest
	}
	if message, ok := m.messages[topic][offset]; ok {
		return message, nil
	}
	return nil, errors.ErrNotMessageAtOffset
}

func (m *MockDBDriver) GetEarliestOffset(topic string) (int64, error) {
	if m.getEarliestOffsetFunc != nil {
		return m.getEarliestOffsetFunc(topic)
	}
	if _, ok := m.topics[topic]; !ok {
		return 0, errors.ErrTopicNotFound
	}
	if m.latestOffsets[topic] == 0 {
		return 0, errors.ErrTopicIsEmpty
	}
	// Find the earliest offset
	var earliestOffset int64 = -1
	for offset := range m.messages[topic] {
		if earliestOffset == -1 || offset < earliestOffset {
			earliestOffset = offset
		}
	}
	return earliestOffset, nil
}

func (m *MockDBDriver) GetLatestOffset(topic string) (int64, error) {
	if m.getLatestOffsetFunc != nil {
		return m.getLatestOffsetFunc(topic)
	}
	if _, ok := m.topics[topic]; !ok {
		return 0, errors.ErrTopicNotFound
	}
	if m.latestOffsets[topic] == 0 {
		return 0, errors.ErrTopicIsEmpty
	}
	return m.latestOffsets[topic], nil
}

func (m *MockDBDriver) GetEarliestMessageFromTopic(topic string) ([]byte, int64, error) {
	if m.getEarliestMessageFunc != nil {
		return m.getEarliestMessageFunc(topic)
	}
	if _, ok := m.topics[topic]; !ok {
		return nil, 0, errors.ErrTopicNotFound
	}
	if m.latestOffsets[topic] == 0 {
		return nil, 0, errors.ErrTopicIsEmpty
	}
	// Find the earliest offset
	var earliestOffset int64 = -1
	for offset := range m.messages[topic] {
		if earliestOffset == -1 || offset < earliestOffset {
			earliestOffset = offset
		}
	}
	return m.messages[topic][earliestOffset], earliestOffset, nil
}

func (m *MockDBDriver) GetLatestMessageFromTopic(topic string) ([]byte, int64, error) {
	if m.getLatestMessageFunc != nil {
		return m.getLatestMessageFunc(topic)
	}
	if _, ok := m.topics[topic]; !ok {
		return nil, 0, errors.ErrTopicNotFound
	}
	if m.latestOffsets[topic] == 0 {
		return nil, 0, errors.ErrTopicIsEmpty
	}
	return m.messages[topic][m.latestOffsets[topic]], m.latestOffsets[topic], nil
}

func (m *MockDBDriver) DeleteMessagesUntilOffset(topic string, offset int64) error {
	if m.deleteMessagesUntilOffsetFunc != nil {
		return m.deleteMessagesUntilOffsetFunc(topic, offset)
	}
	if _, ok := m.topics[topic]; !ok {
		return errors.ErrTopicNotFound
	}
	for o := range m.messages[topic] {
		if o <= offset {
			delete(m.messages[topic], o)
		}
	}
	return nil
}

func (m *MockDBDriver) Close() {
	// No-op for mock
}

func (m *MockDBDriver) Debug() {
	// No-op for mock
}

func (m *MockDBDriver) GetMessagesAfterOffsetWithLimit(topic string, startOffset int64, limit int) ([]database.Message, int64, error) {
	if m.getMessagesAfterOffsetWithLimitFunc != nil {
		return m.getMessagesAfterOffsetWithLimitFunc(topic, startOffset, limit)
	}

	if _, ok := m.topics[topic]; !ok {
		return nil, -1, errors.ErrTopicNotFound
	}

	var resultMessages []database.Message
	var lastOffsetSeen int64 = -1
	count := 0

	// Iterate in order of offsets to correctly apply limit
	// This requires getting all offsets and sorting them, or iterating and sorting if performance is an issue.
	// For a mock, iterating through the map and then sorting keys is acceptable.
	var offsets []int64
	for off := range m.messages[topic] {
		offsets = append(offsets, off)
	}
	// Sort offsets: Go maps don't guarantee order.
	// A simple sort for testing:
	for i := 0; i < len(offsets); i++ {
		for j := i + 1; j < len(offsets); j++ {
			if offsets[i] > offsets[j] {
				offsets[i], offsets[j] = offsets[j], offsets[i]
			}
		}
	}

	for _, currentOffset := range offsets {
		if currentOffset > startOffset {
			if limit <= 0 || count < limit {
				msgData := m.messages[topic][currentOffset]
				resultMessages = append(resultMessages, database.Message{Message: msgData, Offset: currentOffset})
				lastOffsetSeen = currentOffset
				count++
			} else {
				break // Reached limit
			}
		}
	}

	// Determine nextOffset based on whether messages were found.
	// If messages were found, nextOffset is the offset after the last retrieved message.
	// If no messages were found, nextOffset remains the original startOffset.
	// This aligns with the simplified logic in topic_operations.go.
	nextOffset := startOffset
	if len(resultMessages) > 0 {
		nextOffset = lastOffsetSeen + 1
	}

	return resultMessages, nextOffset, nil
}

// Setup a gRPC server and client for testing
func setupTest(t *testing.T) (*Server, pb.SuhaibMessageQueueClient, func()) {
	// Create a buffer connection
	lis := bufconn.Listen(1024 * 1024)

	// Create a mock database driver
	mockDriver := NewMockDBDriver()

	// Create a server with the mock driver
	s := &Server{
		driver:     mockDriver,
		grpcServer: grpc.NewServer(),
		port:       "8097",
	}

	// Register the server
	pb.RegisterSuhaibMessageQueueServer(s.grpcServer, s)

	// Start the server
	errCh := make(chan error, 1)
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("failed to serve: %v", err)
		}
	}()

	// Check for any immediate errors
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	default:
		// No error, continue
	}

	// Create a client connection
	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	// Create a client
	client := pb.NewSuhaibMessageQueueClient(conn)

	// Return a cleanup function
	cleanup := func() {
		s.grpcServer.Stop()
		conn.Close()
	}

	return s, client, cleanup
}

func TestCreateTopic(t *testing.T) {
	s, client, cleanup := setupTest(t)
	defer cleanup()

	// Test creating a topic
	_, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Verify topic was created
	mockDriver := s.driver.(*MockDBDriver)
	if !mockDriver.topics["test_topic"] {
		t.Fatal("Expected topic to be created")
	}

	// Test creating a topic that already exists
	mockDriver.createTopicFunc = func(topic string) error {
		return errors.ErrTopicAlreadyExists
	}
	_, err = client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err == nil {
		t.Fatal("Expected error when creating topic that already exists")
	}
}

func TestProduce(t *testing.T) {
	s, client, cleanup := setupTest(t)
	defer cleanup()

	// Create a topic
	_, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Test producing a message
	message := []byte("test message")
	_, err = client.Produce(context.Background(), &pb.ProduceRequest{Topic: "test_topic", Message: message})
	if err != nil {
		t.Fatalf("Failed to produce message: %v", err)
	}

	// Verify message was added
	mockDriver := s.driver.(*MockDBDriver)
	if mockDriver.latestOffsets["test_topic"] != 1 {
		t.Errorf("Expected latest offset to be 1, got %d", mockDriver.latestOffsets["test_topic"])
	}
	if string(mockDriver.messages["test_topic"][1]) != string(message) {
		t.Errorf("Expected message '%s', got '%s'", string(message), string(mockDriver.messages["test_topic"][1]))
	}

	// Test producing a message to a non-existent topic
	mockDriver.addMessageToTopicFunc = func(topic string, data []byte) error {
		return errors.ErrTopicNotFound
	}
	_, err = client.Produce(context.Background(), &pb.ProduceRequest{Topic: "non_existent_topic", Message: message})
	if err == nil {
		t.Fatal("Expected error when producing message to non-existent topic")
	}
}

func TestConsume(t *testing.T) {
	s, client, cleanup := setupTest(t)
	defer cleanup()

	// Create a topic
	_, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Add a message to the topic
	message := []byte("test message")
	mockDriver := s.driver.(*MockDBDriver)
	mockDriver.AddMessageToTopic("test_topic", message)

	// Test consuming the message
	response, err := client.Consume(context.Background(), &pb.ConsumeRequest{Topic: "test_topic", Offset: 1})
	if err != nil {
		t.Fatalf("Failed to consume message: %v", err)
	}

	// Verify the message is correct
	if string(response.Message) != string(message) {
		t.Errorf("Expected message '%s', got '%s'", string(message), string(response.Message))
	}
	if response.Offset != 1 {
		t.Errorf("Expected offset 1, got %d", response.Offset)
	}

	// Test consuming a message at an offset that doesn't exist
	mockDriver.getMessageAtOffsetFunc = func(topic string, offset int64) ([]byte, error) {
		return nil, errors.ErrOffsetGreaterThanLatest
	}
	_, err = client.Consume(context.Background(), &pb.ConsumeRequest{Topic: "test_topic", Offset: 2})
	if err == nil {
		t.Fatal("Expected error when consuming message at non-existent offset")
	}

	// Test consuming a message from a non-existent topic
	mockDriver.getMessageAtOffsetFunc = func(topic string, offset int64) ([]byte, error) {
		return nil, errors.ErrTopicNotFound
	}
	_, err = client.Consume(context.Background(), &pb.ConsumeRequest{Topic: "non_existent_topic", Offset: 1})
	if err == nil {
		t.Fatal("Expected error when consuming message from non-existent topic")
	}
}

func TestGetEarliestOffset(t *testing.T) {
	s, client, cleanup := setupTest(t)
	defer cleanup()

	// Create a topic
	_, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Add a message to the topic
	message := []byte("test message")
	mockDriver := s.driver.(*MockDBDriver)
	mockDriver.AddMessageToTopic("test_topic", message)

	// Test getting the earliest offset
	response, err := client.GetEarliestOffset(context.Background(), &pb.GetEarliestOffsetRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to get earliest offset: %v", err)
	}

	// Verify the offset is correct
	if response.Offset != 1 {
		t.Errorf("Expected offset 1, got %d", response.Offset)
	}

	// Test getting the earliest offset from an empty topic
	mockDriver.getEarliestOffsetFunc = func(topic string) (int64, error) {
		return 0, errors.ErrTopicIsEmpty
	}
	_, err = client.GetEarliestOffset(context.Background(), &pb.GetEarliestOffsetRequest{Topic: "empty_topic"})
	if err == nil {
		t.Fatal("Expected error when getting earliest offset from empty topic")
	}

	// Test getting the earliest offset from a non-existent topic
	mockDriver.getEarliestOffsetFunc = func(topic string) (int64, error) {
		return 0, errors.ErrTopicNotFound
	}
	_, err = client.GetEarliestOffset(context.Background(), &pb.GetEarliestOffsetRequest{Topic: "non_existent_topic"})
	if err == nil {
		t.Fatal("Expected error when getting earliest offset from non-existent topic")
	}
}

func TestGetLatestOffset(t *testing.T) {
	s, client, cleanup := setupTest(t)
	defer cleanup()

	// Create a topic
	_, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3"}
	mockDriver := s.driver.(*MockDBDriver)
	for _, msg := range messages {
		mockDriver.AddMessageToTopic("test_topic", []byte(msg))
	}

	// Test getting the latest offset
	response, err := client.GetLatestOffset(context.Background(), &pb.GetLatestOffsetRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to get latest offset: %v", err)
	}

	// Verify the offset is correct
	if response.Offset != 3 {
		t.Errorf("Expected offset 3, got %d", response.Offset)
	}

	// Test getting the latest offset from an empty topic
	mockDriver.getLatestOffsetFunc = func(topic string) (int64, error) {
		return 0, errors.ErrTopicIsEmpty
	}
	_, err = client.GetLatestOffset(context.Background(), &pb.GetLatestOffsetRequest{Topic: "empty_topic"})
	if err == nil {
		t.Fatal("Expected error when getting latest offset from empty topic")
	}

	// Test getting the latest offset from a non-existent topic
	mockDriver.getLatestOffsetFunc = func(topic string) (int64, error) {
		return 0, errors.ErrTopicNotFound
	}
	_, err = client.GetLatestOffset(context.Background(), &pb.GetLatestOffsetRequest{Topic: "non_existent_topic"})
	if err == nil {
		t.Fatal("Expected error when getting latest offset from non-existent topic")
	}
}

func TestDeleteUntilOffset(t *testing.T) {
	s, client, cleanup := setupTest(t)
	defer cleanup()

	// Create a topic
	_, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3", "message 4", "message 5"}
	mockDriver := s.driver.(*MockDBDriver)
	for _, msg := range messages {
		mockDriver.AddMessageToTopic("test_topic", []byte(msg))
	}

	// Test deleting messages until offset 3
	_, err = client.DeleteUntilOffset(context.Background(), &pb.DeleteUntilOffsetRequest{Topic: "test_topic", Offset: 3})
	if err != nil {
		t.Fatalf("Failed to delete messages: %v", err)
	}

	// Verify messages were deleted
	for i := int64(1); i <= 3; i++ {
		if _, ok := mockDriver.messages["test_topic"][i]; ok {
			t.Errorf("Expected message at offset %d to be deleted", i)
		}
	}

	// Verify remaining messages are still accessible
	for i := int64(4); i <= 5; i++ {
		if _, ok := mockDriver.messages["test_topic"][i]; !ok {
			t.Errorf("Expected message at offset %d to be accessible", i)
		}
	}

	// Test deleting messages from a non-existent topic
	mockDriver.deleteMessagesUntilOffsetFunc = func(topic string, offset int64) error {
		return errors.ErrTopicNotFound
	}
	_, err = client.DeleteUntilOffset(context.Background(), &pb.DeleteUntilOffsetRequest{Topic: "non_existent_topic", Offset: 1})
	if err == nil {
		t.Fatal("Expected error when deleting messages from non-existent topic")
	}
}

func TestNewServer(t *testing.T) {
	// Test creating a new server
	s := NewServer("8097", ":memory:")

	// Verify server was created successfully
	if s == nil {
		t.Fatal("Expected server to be non-nil")
	}

	// Verify driver was created
	if s.driver == nil {
		t.Fatal("Expected driver to be non-nil")
	}

	// Verify gRPC server was created
	if s.grpcServer == nil {
		t.Fatal("Expected gRPC server to be non-nil")
	}

	// Verify port was set
	if s.port != "8097" {
		t.Errorf("Expected port to be '8097', got '%s'", s.port)
	}

	// Clean up
	s.Close()
}

func TestStreamConsume(t *testing.T) {
	s, client, cleanup := setupTest(t)
	defer cleanup()

	// Create a topic
	_, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Add messages to the topic
	messages := []string{"message 1", "message 2", "message 3"}
	mockDriver := s.driver.(*MockDBDriver)
	for _, msg := range messages {
		mockDriver.AddMessageToTopic("test_topic", []byte(msg))
	}

	// Test streaming messages
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.StreamConsume(ctx, &pb.ConsumeRequest{Topic: "test_topic", Offset: 1})
	if err != nil {
		t.Fatalf("Failed to start stream: %v", err)
	}

	// Receive messages
	receivedMessages := make([]string, 0)
	for {
		response, err := stream.Recv()
		if err != nil {
			break
		}
		if response.Offset == config.SPECIAL_OFFSET_HEARTBEAT {
			continue
		}
		receivedMessages = append(receivedMessages, string(response.Message))
	}

	// Verify all messages were received
	if len(receivedMessages) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(receivedMessages))
	}
	for i, msg := range messages {
		if i < len(receivedMessages) && receivedMessages[i] != msg {
			t.Errorf("Expected message '%s', got '%s'", msg, receivedMessages[i])
		}
	}
}

func TestStreamProducer(t *testing.T) {
	s, client, cleanup := setupTest(t)
	defer cleanup()

	// Create a topic
	_, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{Topic: "test_topic"})
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Test streaming messages
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.StreamProduce(ctx)
	if err != nil {
		t.Fatalf("Failed to start stream: %v", err)
	}

	// Send messages
	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		err = stream.Send(&pb.ProduceRequest{Topic: "test_topic", Message: []byte(msg)})
		if err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}
	}

	// Close the stream
	_, err = stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("Failed to close stream: %v", err)
	}

	// Verify messages were added
	mockDriver := s.driver.(*MockDBDriver)
	if mockDriver.latestOffsets["test_topic"] != int64(len(messages)) {
		t.Errorf("Expected latest offset to be %d, got %d", len(messages), mockDriver.latestOffsets["test_topic"])
	}
	for i, msg := range messages {
		offset := int64(i + 1)
		if string(mockDriver.messages["test_topic"][offset]) != msg {
			t.Errorf("Expected message '%s' at offset %d, got '%s'", msg, offset, string(mockDriver.messages["test_topic"][offset]))
		}
	}
}
