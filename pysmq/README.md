# PySMQ: Python Client for SuhaibMessageQueue

This package provides a Python client for interacting with [SuhaibMessageQueue](https://github.com/Suhaibinator/SuhaibMessageQueue), a simple and efficient message queue service.

## Installation

```bash
# From PyPI (once published)
pip install pysmq

# From source
git clone https://github.com/Suhaibinator/SuhaibMessageQueue.git
cd SuhaibMessageQueue
pip install -e ./pysmq
```

## Dependencies

- Python 3.7+
- grpcio
- protobuf

## Quick Start

### Creating a client and publishing messages

```python
from pysmq.client import Client

# Create a client
with Client(host="localhost", port=8097) as client:
    # Create a topic
    client.create_topic("my-topic")
    
    # Publish a message
    offset = client.produce("my-topic", b"Hello, SMQ!")
    print(f"Message published at offset {offset}")
```

### Consuming messages

```python
from pysmq.client import Client

# Create a client
with Client(host="localhost", port=8097) as client:
    # Consume the latest message
    message, offset = client.consume_latest("my-topic")
    print(f"Latest message: {message.decode('utf-8')}, offset: {offset}")
    
    # Consume the earliest message
    message, offset = client.consume_earliest("my-topic")
    print(f"Earliest message: {message.decode('utf-8')}, offset: {offset}")
    
    # Consume a specific message
    message, offset = client.consume("my-topic", 5)  # offset 5
    print(f"Message at offset 5: {message.decode('utf-8')}")
```

### Streaming messages

```python
from pysmq.client import Client

# Create a client
with Client(host="localhost", port=8097) as client:
    # Stream produce messages
    messages = [b"Message 1", b"Message 2", b"Message 3"]
    last_offset = client.stream_produce("my-topic", messages)
    print(f"Last message published at offset {last_offset}")
    
    # Stream consume messages (continuous)
    for message, offset in client.stream_consume("my-topic", 0):
        print(f"Received message: {message.decode('utf-8')}, offset: {offset}")
        # Break the loop when needed
        if offset >= 10:
            break
```

### Secure Connection (mTLS)

```python
from pysmq.client import Client
from pysmq.config import ClientTLSConfig

# Create TLS configuration
tls_config = ClientTLSConfig(
    cert_file="/path/to/client.crt",
    key_file="/path/to/client.key",
    ca_file="/path/to/ca.crt"
)

# Create a secure client
with Client(host="localhost", port=8097, tls_config=tls_config) as client:
    # Use the client as normal
    client.create_topic("secure-topic")
    offset = client.produce("secure-topic", b"Secure message")
    print(f"Secure message published at offset {offset}")
```

## API Overview

### Client

The `Client` class provides methods for interacting with the SMQ server.

```python
# Constructor
client = Client(
    host="localhost",
    port=8097,
    tls_config=None,  # Optional: ClientTLSConfig for secure connections
    max_send_message_size_mb=1024,  # Optional: Maximum size of messages to send
    max_receive_message_size_mb=1024  # Optional: Maximum size of messages to receive
)
```

### Topic Management

```python
# Create a topic
client.create_topic(topic)
```

### Message Production

```python
# Produce a single message
offset = client.produce(topic, message_bytes)

# Produce a stream of messages
last_offset = client.stream_produce(topic, messages_iterable)
```

### Message Consumption

```python
# Consume a single message at a specific offset
message, offset = client.consume(topic, offset)

# Consume the earliest available message
message, offset = client.consume_earliest(topic)

# Consume the latest message
message, offset = client.consume_latest(topic)

# Stream consume messages (returns an iterator)
for message, offset in client.stream_consume(topic, start_offset):
    # Process each message
    pass
```

### Offset Management

```python
# Get the latest offset for a topic
latest_offset = client.get_latest_offset(topic)

# Get the earliest available offset for a topic
earliest_offset = client.get_earliest_offset(topic)

# Delete all messages up to and including the specified offset
client.delete_until_offset(topic, offset)
```

### Bulk Operations

```python
# Retrieve a batch of messages
response = client.bulk_retrieve(topic, start_offset, limit)
for message in response.messages:
    # Process each message
    message_bytes = message.message
    message_offset = message.offset
```

### Connection Management

```python
# Connect to the server (optional, happens automatically on first operation)
client.connect()

# Close the connection (or use 'with' statement)
client.close()
```

## Exception Handling

The client throws specific exceptions for different types of errors:

```python
from pysmq.exceptions import (
    SMQException,              # Base exception for all SMQ errors
    SMQConnectionError,        # Connection issues
    SMQAuthenticationError,    # TLS/authentication issues
    SMQProduceError,           # Issues with producing messages
    SMQConsumeError,           # Issues with consuming messages
    SMQTopicCreationError,     # Issues with creating topics
    SMQOffsetError,            # Issues with offsets
    SMQTimeoutError,           # Timeout errors
)

try:
    client.produce("my-topic", b"Hello")
except SMQProduceError as e:
    print(f"Failed to produce message: {e}")
    # Access the original error if needed
    original_error = e.original_error
```

## Configuration

The client supports various configuration options:

```python
from pysmq.config import ClientTLSConfig, ClientConfig

# Default timeout values can be accessed or modified
ClientConfig.DEFAULT_TIMEOUT = 60.0  # Default timeout for most operations
ClientConfig.DEFAULT_TOPIC_TIMEOUT = 2.0  # Default timeout for topic creation
ClientConfig.DEFAULT_DELETE_TIMEOUT = 2.0  # Default timeout for deletion operations
```

## Examples

For more detailed examples, see the [examples](examples/) directory:

- [Basic Publishing](examples/publish_example.py)
- [Message Consumption](examples/subscribe_example.py)
- [Secure Connection](examples/secure_connection_example.py)

## License

This project is licensed under the MIT License - see the LICENSE file for details.
