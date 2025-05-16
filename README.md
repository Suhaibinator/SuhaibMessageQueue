# Suhaib Message Queue (SMQ)

Suhaib Message Queue (SMQ) is an ultra-lightweight, efficient messaging queue system developed in Go. It is designed for high adaptability and resilience, perfect for integrating with Go applications, deploying in Docker containers, or operating as a standalone service.

## Features

- **Lightweight and Efficient**: Built specifically in Go to ensure minimal overhead.
- **Flexible Deployment Options**: Run as a Docker container, integrate within Go applications, or operate as a standalone executable.
- **Reliable Persistence**: Uses SQLite to store messages on disk with a write-ahead log and rollback journal, ensuring data durability across restarts and failures.
- **gRPC Integration**: Utilizes gRPC for efficient, scalable communication between distributed services.
- **Client Packages**: Includes both a Go client package and a Python client package (pysmq on PyPI) to facilitate easy integration and communication across different languages.

## Why Choose SMQ Over Apache Kafka or RabbitMQ?
SMQ is particularly advantageous in environments where simplicity, minimal overhead, and integration ease are paramount over the comprehensive feature sets of larger systems like Apache Kafka or RabbitMQ. It is especially well-suited for:

- **Small to Medium-Sized Applications**: Ideal for applications where the overhead of larger message brokers is impractical.
- **Microservices Architectures**: Especially those developed in Go, benefiting from native integration and performance optimization.
- **Rapid Prototyping**: Offers quick setup and easy configuration, making it suitable for development environments and testing new ideas.
- **IoT and Edge Computing**: Its minimalistic design fits well in resource-constrained environments such as IoT devices and edge computing scenarios.


## Getting Started

### Prerequisites

- Go 1.22 or later
- Docker (optional for Docker deployment)

### Installing

To install SMQ, clone the repository and build the application using Go:

```bash
git clone https://github.com/Suhaibinator/SuhaibMessageQueue.git
cd SuhaibMessageQueue
go build
```

### Running SMQ

To run SMQ, execute the built binary:

```bash
./smq
```

SMQ will start running on port 8097 by default. However, you can customize the behavior of SMQ using command-line flags and environment variables. The priority for these settings is as follows: command-line flags > environment variables > default values.

### Using Docker

```bash
docker run suhaibinator/smq:latest-amd64
```

or

```bash
docker run suhaibinator/smq:latest-arm64
```

### Command-Line Flags

- `--dbpath`: Specifies the path to the SQLite database file. Default: `./dbtest.db`.
- `--port`: Specifies the port to listen on. Default: `8097`.
- `--enable-mtls`: Enables client-side mTLS authentication. (boolean, e.g., `--enable-mtls=true`)
- `--client-cert`: Path to the client's certificate file for mTLS.
- `--client-key`: Path to the client's private key file for mTLS.
- `--ca-cert`: Path to the CA certificate file for the client to verify the server.
- `--server-enable-mtls`: Enables server-side mTLS authentication. (boolean, e.g., `--server-enable-mtls=true`)
- `--server-cert`: Path to the server's certificate file for mTLS.
- `--server-key`: Path to the server's private key file for mTLS.
- `--server-ca-cert`: Path to the CA certificate file for the server to verify clients.

If command-line flags are provided, they take the highest priority and override any other settings.

### Environment Variables

- `DB_PATH`: Overrides the default path to the SQLite database file (unless `--dbpath` is used).
- `PORT`: Overrides the default port to listen on (unless `--port` is used).
- `ENABLE_MTLS`: Enables client-side mTLS (`true` or `false`).
- `CLIENT_CERT`: Path to client certificate file.
- `CLIENT_KEY`: Path to client key file.
- `CA_CERT`: Path to CA certificate for client to verify server.
- `SERVER_ENABLE_MTLS`: Enables server-side mTLS (`true` or `false`).
- `SERVER_CERT_FILE`: Path to server certificate file.
- `SERVER_KEY_FILE`: Path to server key file.
- `SERVER_CA_CERT_FILE`: Path to CA certificate for server to verify clients.

Environment variables take priority over default values but are overridden by command-line flags if they are set.

To use an environment variable, export it before running the SMQ binary. For example:

```bash
export DB_PATH=/path/to/your/database.db
export PORT=9000
./smq
```


## Usage

### mTLS Configuration

SMQ supports mutual TLS (mTLS) for securing communication between the client and server. You can configure mTLS for the client, the server, or both.

**Client-Side mTLS:**
When client-side mTLS is enabled, the client will present its certificate to the server, and the server will verify it. The client will also verify the server's certificate using the provided CA certificate.
- Enable with `ENABLE_MTLS=true` (env) or `--enable-mtls` (flag).
- Provide `CLIENT_CERT`, `CLIENT_KEY`, and `CA_CERT` (env) or `--client-cert`, `--client-key`, `--ca-cert` (flags).

**Server-Side mTLS:**
When server-side mTLS is enabled, the server will require clients to present a certificate and will verify it against the server's CA certificate. The server will also present its own certificate to clients.
- Enable with `SERVER_ENABLE_MTLS=true` (env) or `--server-enable-mtls` (flag).
- Provide `SERVER_CERT_FILE`, `SERVER_KEY_FILE`, and `SERVER_CA_CERT_FILE` (env) or `--server-cert`, `--server-key`, `--server-ca-cert` (flags).

**Generating Certificates (Example using OpenSSL):**

You'll typically need a Certificate Authority (CA), a server certificate/key signed by the CA, and a client certificate/key signed by the CA.

1.  **Create CA:**
    ```bash
    openssl genrsa -out ca.key 4096
    openssl req -new -x509 -days 365 -key ca.key -out ca.crt -subj "/CN=My SMQ CA"
    ```

2.  **Create Server Certificate:**
    ```bash
    openssl genrsa -out server.key 4096
    openssl req -new -key server.key -out server.csr -subj "/CN=localhost" # Use your server's hostname
    openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt
    ```

3.  **Create Client Certificate:**
    ```bash
    openssl genrsa -out client.key 4096
    openssl req -new -key client.key -out client.csr -subj "/CN=smqclient"
    openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt
    ```

Place these files in appropriate locations and configure SMQ using the flags or environment variables described above. For example, to run the server with mTLS:
```bash
export SERVER_ENABLE_MTLS=true
export SERVER_CERT_FILE=./server.crt
export SERVER_KEY_FILE=./server.key
export SERVER_CA_CERT_FILE=./ca.crt # This CA will be used to verify client certs
./smq
```
And for the client (in your client application's configuration):
```go
// In your client application
config.EnableMTLS = true
config.ClientCert = "./client.crt"
config.ClientKey = "./client.key"
config.CACert = "./ca.crt" // This CA will be used to verify the server cert
// ... then initialize your client
```

Here's an example of how to use the SMQ client package to create a topic, produce a message, and consume messages:

```go
package main

import (
    "github.com/yourusername/smq/client"
    "log"
    "time"
)

func main() {
    // Create a new client
    smqClient, err := client.NewClient("localhost", "8097")
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer smqClient.Close()

    // Create a topic
    topicName := "exampleTopic"
    if err := smqClient.CreateTopic(topicName); err != nil {
        log.Fatalf("Failed to create topic: %v", err)
    }

    // Produce a message to the topic
    message := []byte("Hello, SMQ!")
    if err := smqClient.Produce(topicName, message); err != nil {
        log.Fatalf("Failed to send message: %v", err)
    }

    // Consume the earliest message from the topic
    msg, offset, err := smqClient.ConsumeEarliest(topicName)
    if err != nil {
        log.Fatalf("Failed to consume message: %v", err)
    }
    log.Printf("Received message at offset %d: %s", offset, string(msg))
}
```

## Python Client Usage

SMQ also provides a Python client package (`pysmq`) available on PyPI, which allows Python applications to interact with SMQ servers.

### Installing the Python Client

Install the Python client using pip:

```bash
pip install pysmq
```

### Basic Usage Examples

#### Publishing Messages

Here's an example of how to publish messages to an SMQ server using the Python client:

```python
import json
import time
from pysmq.client import Client

# Create a client
with Client(host="localhost", port=8097) as client:
    # Connect to the server
    client.connect()
    
    # Create a topic
    topic = "example-topic"
    client.create_topic(topic)
    
    # Publish a message
    message = {
        "id": 1,
        "timestamp": time.time(),
        "content": "Hello from Python!",
    }
    message_bytes = json.dumps(message).encode('utf-8')
    
    # Produce the message
    offset = client.produce(topic, message_bytes)
    print(f"Message published at offset {offset}")
```

#### Consuming Messages

Here's an example of how to consume messages:

```python
import json
from pysmq.client import Client

# Create a client
with Client(host="localhost", port=8097) as client:
    # Connect to the server
    client.connect()
    
    topic = "example-topic"
    
    # Consume the latest message
    message_bytes, offset = client.consume_latest(topic)
    
    # Process the message
    message = json.loads(message_bytes.decode('utf-8'))
    print(f"Received message at offset {offset}:")
    print(f"Content: {message.get('content')}")
    
    # Stream consumption example
    print("Starting stream consumption...")
    for message_bytes, offset in client.stream_consume(topic, start_offset=0):
        message = json.loads(message_bytes.decode('utf-8'))
        print(f"Received message at offset {offset}: {message.get('content')}")
        # Break condition (example: stop after processing 10 messages)
        if offset >= 10:
            break
```

#### Secure Connection with mTLS

The Python client also supports secure connections using mTLS:

```python
from pysmq.client import Client
from pysmq.config import ClientTLSConfig

# Create a TLS configuration
tls_config = ClientTLSConfig(
    cert_file="/path/to/client.crt",
    key_file="/path/to/client.key",
    ca_file="/path/to/ca.crt",
)

# Create a secure client
with Client(host="localhost", port=8097, tls_config=tls_config) as client:
    # Connect securely
    client.connect()
    print("Connected securely!")
    
    # Now you can use the client as in the previous examples
```

For more detailed examples, refer to the Python client documentation and example files in the `pysmq/examples` directory.
