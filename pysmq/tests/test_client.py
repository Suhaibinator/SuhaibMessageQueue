"""
Unit tests for the SMQ Python client.
"""
import unittest
from unittest.mock import Mock, patch, MagicMock
import grpc

from pysmq.client import Client
from pysmq.config import ClientTLSConfig
from pysmq.exceptions import SMQConnectionError, SMQProduceError, SMQConsumeError


class TestClient(unittest.TestCase):
    """Test cases for the SMQ Client."""
    
    def setUp(self):
        """Set up test fixtures."""
        # Create a mock for the gRPC stub
        self.mock_stub = Mock()
        
        # Patch the gRPC channel creation to return a mock
        self.channel_patcher = patch('grpc.insecure_channel')
        self.mock_channel = self.channel_patcher.start()
        self.mock_channel.return_value = Mock()
        
        # Patch the stub class to return our mock stub
        self.stub_patcher = patch('pysmq.proto.smq_pb2_grpc.SuhaibMessageQueueStub')
        self.mock_stub_class = self.stub_patcher.start()
        self.mock_stub_class.return_value = self.mock_stub
        
        # Create the client with the patched grpc channel
        self.client = Client(host="localhost", port=8097)
    
    def tearDown(self):
        """Tear down test fixtures."""
        self.channel_patcher.stop()
        self.stub_patcher.stop()
    
    def test_connect(self):
        """Test the connect method."""
        # Set up mock response
        self.mock_stub.Connect.return_value = Mock()
        
        # Call the method
        self.client.connect()
        
        # Assert the stub was called correctly
        self.mock_stub.Connect.assert_called_once()
    
    def test_connect_error(self):
        """Test the connect method with an error."""
        # Set up mock to raise an error
        error = grpc.RpcError()
        error.code = lambda: grpc.StatusCode.UNAVAILABLE
        error.details = lambda: "Connection refused"
        self.mock_stub.Connect.side_effect = error
        
        # Call the method and assert it raises the expected exception
        with self.assertRaises(SMQConnectionError):
            self.client.connect()
    
    def test_create_topic(self):
        """Test the create_topic method."""
        # Set up mock response
        self.mock_stub.CreateTopic.return_value = Mock()
        
        # Call the method
        self.client.create_topic("test-topic")
        
        # Assert the stub was called correctly
        self.mock_stub.CreateTopic.assert_called_once()
        args, kwargs = self.mock_stub.CreateTopic.call_args
        self.assertEqual(args[0].topic, "test-topic")
    
    def test_produce(self):
        """Test the produce method."""
        # Set up mock response
        mock_response = Mock()
        mock_response.offset = 42
        self.mock_stub.Produce.return_value = mock_response
        
        # Call the method
        offset = self.client.produce("test-topic", b"test-message")
        
        # Assert the stub was called correctly
        self.mock_stub.Produce.assert_called_once()
        args, kwargs = self.mock_stub.Produce.call_args
        self.assertEqual(args[0].topic, "test-topic")
        self.assertEqual(args[0].message, b"test-message")
        
        # Assert the return value
        self.assertEqual(offset, 42)
    
    def test_produce_error(self):
        """Test the produce method with an error."""
        # Set up mock to raise an error
        error = grpc.RpcError()
        error.code = lambda: grpc.StatusCode.INVALID_ARGUMENT
        error.details = lambda: "Invalid topic"
        self.mock_stub.Produce.side_effect = error
        
        # Call the method and assert it raises the expected exception
        with self.assertRaises(SMQProduceError):
            self.client.produce("test-topic", b"test-message")
    
    def test_consume(self):
        """Test the consume method."""
        # Set up mock response
        mock_response = Mock()
        mock_response.message = b"test-message"
        mock_response.offset = 42
        self.mock_stub.Consume.return_value = mock_response
        
        # Call the method
        message, offset = self.client.consume("test-topic", 42)
        
        # Assert the stub was called correctly
        self.mock_stub.Consume.assert_called_once()
        args, kwargs = self.mock_stub.Consume.call_args
        self.assertEqual(args[0].topic, "test-topic")
        self.assertEqual(args[0].offset, 42)
        
        # Assert the return values
        self.assertEqual(message, b"test-message")
        self.assertEqual(offset, 42)
    
    def test_consume_error(self):
        """Test the consume method with an error."""
        # Set up mock to raise an error
        error = grpc.RpcError()
        error.code = lambda: grpc.StatusCode.NOT_FOUND
        error.details = lambda: "Offset not found"
        self.mock_stub.Consume.side_effect = error
        
        # Call the method and assert it raises the expected exception
        with self.assertRaises(SMQConsumeError):
            self.client.consume("test-topic", 42)
    
    def test_close(self):
        """Test the close method."""
        # Call the method
        self.client.close()
        
        # Assert the channel was closed
        self.client._channel.close.assert_called_once()


if __name__ == "__main__":
    unittest.main()
