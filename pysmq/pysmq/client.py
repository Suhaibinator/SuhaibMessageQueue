"""
Core client implementation for the SMQ Python client.
"""
import os
import ssl
from typing import Iterator, Iterable, Tuple, Callable, Optional, Any, Union

import grpc

from .proto import smq_pb2, smq_pb2_grpc
from .config import ClientTLSConfig, ClientConfig, SMQ_SPECIAL_OFFSET_HEARTBEAT
from .exceptions import (
    SMQException,
    SMQConnectionError,
    SMQAuthenticationError,
    SMQProduceError,
    SMQConsumeError,
    SMQTimeoutError,
    SMQTopicCreationError,
    map_grpc_error,
)


class Client:
    """
    Client for SuhaibMessageQueue.
    
    This client handles communication with the SMQ server, including connection
    management, message production, consumption, topic management, and offset
    management.
    """
    
    def __init__(
        self, 
        host: str, 
        port: int, 
        tls_config: Optional[ClientTLSConfig] = None,
        max_send_message_size_mb: int = ClientConfig.DEFAULT_MAX_MESSAGE_SIZE_MB,
        max_receive_message_size_mb: int = ClientConfig.DEFAULT_MAX_MESSAGE_SIZE_MB,
    ):
        """
        Initialize a new SMQ client.
        
        Args:
            host: The hostname or IP address of the SMQ server.
            port: The port number of the SMQ server.
            tls_config: Optional TLS configuration for secure connections.
                If provided, establishes a secure connection using mTLS.
                If None, establishes an insecure connection.
            max_send_message_size_mb: Maximum size of messages to send, in MB.
            max_receive_message_size_mb: Maximum size of messages to receive, in MB.
        
        Raises:
            SMQConnectionError: If there's an issue establishing the connection.
            SMQAuthenticationError: If there's an issue with TLS authentication.
        """
        self._address = f"{host}:{port}"
        self._channel = None
        self._stub = None
        
        # Convert message sizes from MB to bytes
        max_send_size = max_send_message_size_mb * 1024 * 1024
        max_receive_size = max_receive_message_size_mb * 1024 * 1024
        
        # Set up channel options
        options = [
            ('grpc.max_send_message_length', max_send_size),
            ('grpc.max_receive_message_length', max_receive_size),
        ]
        
        try:
            if tls_config:
                # Load client certificate and key for mTLS
                with open(tls_config.cert_file, 'rb') as f:
                    client_cert = f.read()
                
                with open(tls_config.key_file, 'rb') as f:
                    client_key = f.read()
                
                # Load CA certificate for server verification
                with open(tls_config.ca_file, 'rb') as f:
                    ca_cert = f.read()
                
                # Create SSL credentials
                credentials = grpc.ssl_channel_credentials(
                    root_certificates=ca_cert,
                    private_key=client_key,
                    certificate_chain=client_cert,
                )
                
                # Create secure channel
                self._channel = grpc.secure_channel(
                    self._address,
                    credentials,
                    options=options,
                )
            else:
                # Create insecure channel
                self._channel = grpc.insecure_channel(
                    self._address,
                    options=options,
                )
            
            # Create the gRPC stub
            self._stub = smq_pb2_grpc.SuhaibMessageQueueStub(self._channel)
            
        except (OSError, IOError) as e:
            # Handle file I/O errors (e.g., certificate files not found)
            raise SMQAuthenticationError(f"Error loading TLS certificates: {str(e)}", e)
        except grpc.RpcError as e:
            # Handle gRPC connection errors
            raise map_grpc_error(e, SMQConnectionError)
    
    def __enter__(self):
        """Enable the use of 'with' statement for better resource management."""
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Close the client when exiting the 'with' block."""
        self.close()
    
    def close(self):
        """
        Close the connection to the SMQ server.
        
        This method should be called when the client is no longer needed to
        release resources. It's automatically called when using the client
        with a 'with' statement.
        """
        if self._channel:
            self._channel.close()
            self._channel = None
            self._stub = None
    
    def connect(self, timeout: Optional[float] = None) -> None:
        """
        Establish a connection to the SMQ server.
        
        This is a lightweight method that simply verifies the connection.
        The actual connection is established when the client is created.
        
        Args:
            timeout: Optional timeout in seconds. If None, uses the default timeout.
        
        Raises:
            SMQConnectionError: If there's an issue with the connection.
        """
        try:
            self._stub.Connect(
                smq_pb2.ConnectRequest(),
                timeout=timeout,
            )
        except grpc.RpcError as e:
            raise map_grpc_error(e, SMQConnectionError)
    
    def create_topic(self, topic: str, timeout: Optional[float] = ClientConfig.DEFAULT_TOPIC_TIMEOUT) -> None:
        """
        Create a new topic on the SMQ server.
        
        Args:
            topic: The name of the topic to create.
            timeout: Optional timeout in seconds.
        
        Raises:
            SMQTopicCreationError: If there's an issue creating the topic.
        """
        try:
            self._stub.CreateTopic(
                smq_pb2.CreateTopicRequest(topic=topic),
                timeout=timeout,
            )
        except grpc.RpcError as e:
            raise map_grpc_error(e, SMQTopicCreationError)
    
    def get_latest_offset(self, topic: str, timeout: Optional[float] = ClientConfig.DEFAULT_TIMEOUT) -> int:
        """
        Get the latest (highest) offset for a topic.
        
        Args:
            topic: The name of the topic.
            timeout: Optional timeout in seconds.
        
        Returns:
            The latest offset as an integer.
        
        Raises:
            SMQOffsetError: If there's an issue getting the offset.
        """
        try:
            response = self._stub.GetLatestOffset(
                smq_pb2.GetLatestOffsetRequest(topic=topic),
                timeout=timeout,
            )
            return response.offset
        except grpc.RpcError as e:
            raise map_grpc_error(e)
    
    def get_earliest_offset(self, topic: str, timeout: Optional[float] = ClientConfig.DEFAULT_TIMEOUT) -> int:
        """
        Get the earliest (lowest) available offset for a topic.
        
        Args:
            topic: The name of the topic.
            timeout: Optional timeout in seconds.
        
        Returns:
            The earliest offset as an integer.
        
        Raises:
            SMQOffsetError: If there's an issue getting the offset.
        """
        try:
            response = self._stub.GetEarliestOffset(
                smq_pb2.GetEarliestOffsetRequest(topic=topic),
                timeout=timeout,
            )
            return response.offset
        except grpc.RpcError as e:
            raise map_grpc_error(e)
    
    def produce(self, topic: str, message: bytes, timeout: Optional[float] = ClientConfig.DEFAULT_TIMEOUT) -> int:
        """
        Produce a single message to a topic.
        
        Args:
            topic: The name of the topic.
            message: The message content as bytes.
            timeout: Optional timeout in seconds.
        
        Returns:
            The offset of the produced message.
        
        Raises:
            SMQProduceError: If there's an issue producing the message.
        """
        try:
            response = self._stub.Produce(
                smq_pb2.ProduceRequest(topic=topic, message=message),
                timeout=timeout,
            )
            return response.offset
        except grpc.RpcError as e:
            raise map_grpc_error(e, SMQProduceError)
    
    def stream_produce(self, topic: str, messages: Iterable[bytes]) -> int:
        """
        Produce a stream of messages to a topic.
        
        Args:
            topic: The name of the topic.
            messages: An iterable of messages, each as bytes.
        
        Returns:
            The offset of the last produced message.
        
        Raises:
            SMQProduceError: If there's an issue producing messages.
        """
        try:
            # Start a streaming RPC
            stream = self._stub.StreamProduce()
            
            # Send messages
            for message in messages:
                stream.write(smq_pb2.ProduceRequest(topic=topic, message=message))
            
            # Close the stream and get the response
            stream.done_writing()
            response = stream.recv()
            return response.offset
        except grpc.RpcError as e:
            raise map_grpc_error(e, SMQProduceError)
    
    def consume(self, topic: str, offset: int, timeout: Optional[float] = ClientConfig.DEFAULT_TIMEOUT) -> Tuple[bytes, int]:
        """
        Consume a single message from a topic at a specific offset.
        
        Args:
            topic: The name of the topic.
            offset: The offset to consume from.
            timeout: Optional timeout in seconds.
        
        Returns:
            A tuple of (message_bytes, offset).
        
        Raises:
            SMQConsumeError: If there's an issue consuming the message.
        """
        try:
            response = self._stub.Consume(
                smq_pb2.ConsumeRequest(topic=topic, offset=offset),
                timeout=timeout,
            )
            return response.message, response.offset
        except grpc.RpcError as e:
            raise map_grpc_error(e, SMQConsumeError)
    
    def consume_earliest(self, topic: str, timeout: Optional[float] = ClientConfig.DEFAULT_TIMEOUT) -> Tuple[bytes, int]:
        """
        Consume the earliest available message from a topic.
        
        Args:
            topic: The name of the topic.
            timeout: Optional timeout in seconds.
        
        Returns:
            A tuple of (message_bytes, offset).
        
        Raises:
            SMQConsumeError: If there's an issue consuming the message.
        """
        try:
            # Get the earliest offset
            earliest_offset = self.get_earliest_offset(topic, timeout)
            
            # Consume the message at that offset
            return self.consume(topic, earliest_offset, timeout)
        except SMQException as e:
            raise SMQConsumeError(f"Failed to consume earliest message: {str(e)}", e.original_error)
    
    def consume_latest(self, topic: str, timeout: Optional[float] = ClientConfig.DEFAULT_TIMEOUT) -> Tuple[bytes, int]:
        """
        Consume the latest message from a topic.
        
        Args:
            topic: The name of the topic.
            timeout: Optional timeout in seconds.
        
        Returns:
            A tuple of (message_bytes, offset).
        
        Raises:
            SMQConsumeError: If there's an issue consuming the message.
        """
        try:
            # Get the latest offset
            latest_offset = self.get_latest_offset(topic, timeout)
            
            # Consume the message at that offset
            return self.consume(topic, latest_offset, timeout)
        except SMQException as e:
            raise SMQConsumeError(f"Failed to consume latest message: {str(e)}", e.original_error)
    
    def stream_consume(self, topic: str, start_offset: int) -> Iterator[Tuple[bytes, int]]:
        """
        Consume a stream of messages from a topic starting at a specific offset.
        
        This method returns an iterator that yields message bytes and offsets
        as they arrive. It handles heartbeat messages automatically.
        
        Args:
            topic: The name of the topic.
            start_offset: The offset to start consuming from.
        
        Returns:
            An iterator of (message_bytes, offset) tuples.
        
        Raises:
            SMQConsumeError: If there's an issue consuming messages.
        """
        try:
            # Start a streaming RPC
            stream_request = smq_pb2.ConsumeRequest(topic=topic, offset=start_offset)
            stream = self._stub.StreamConsume(stream_request)
            
            # Iterate over the stream, skipping heartbeats
            for response in stream:
                # Skip heartbeat messages
                if response.offset == SMQ_SPECIAL_OFFSET_HEARTBEAT:
                    continue
                
                yield response.message, response.offset
        except grpc.RpcError as e:
            raise map_grpc_error(e, SMQConsumeError)
    
    def delete_until_offset(self, topic: str, offset: int, timeout: Optional[float] = ClientConfig.DEFAULT_DELETE_TIMEOUT) -> None:
        """
        Delete all messages in a topic up to and including the specified offset.
        
        Args:
            topic: The name of the topic.
            offset: The offset up to which messages should be deleted.
            timeout: Optional timeout in seconds.
        
        Raises:
            SMQException: If there's an issue deleting messages.
        """
        try:
            self._stub.DeleteUntilOffset(
                smq_pb2.DeleteUntilOffsetRequest(topic=topic, offset=offset),
                timeout=timeout,
            )
        except grpc.RpcError as e:
            raise map_grpc_error(e)
    
    def bulk_retrieve(
        self, 
        topic: str, 
        start_offset: int, 
        limit: int, 
        timeout: Optional[float] = ClientConfig.DEFAULT_TIMEOUT
    ) -> smq_pb2.BulkRetrieveResponse:
        """
        Retrieve a batch of messages from a topic starting at a specific offset.
        
        Args:
            topic: The name of the topic.
            start_offset: The offset to start retrieving from.
            limit: The maximum number of messages to retrieve.
            timeout: Optional timeout in seconds.
        
        Returns:
            A BulkRetrieveResponse containing messages, count, and next_offset.
        
        Raises:
            SMQConsumeError: If there's an issue retrieving messages.
        """
        try:
            response = self._stub.BulkRetrieve(
                smq_pb2.BulkRetrieveRequest(
                    topic=topic,
                    start_offset=start_offset,
                    limit=limit,
                ),
                timeout=timeout,
            )
            return response
        except grpc.RpcError as e:
            raise map_grpc_error(e, SMQConsumeError)
