# Suhaib Message Queue (SMQ)

Suhaib Message Queue (SMQ) is an ultra-lightweight, efficient messaging queue system developed in Go. It is designed for high adaptability and resilience, perfect for integrating with Go applications, deploying in Docker containers, or operating as a standalone service.

## Features

- **Lightweight and Efficient**: Built specifically in Go to ensure minimal overhead.
- **Flexible Deployment Options**: Run as a Docker container, integrate within Go applications, or operate as a standalone executable.
- **Reliable Persistence**: Uses SQLite to store messages on disk with a write-ahead log and rollback journal, ensuring data durability across restarts and failures.
- **gRPC Integration**: Utilizes gRPC for efficient, scalable communication between distributed services.
- **Client Package**: Includes a client package to facilitate easy integration and communication.

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
git clone https://github.com/yourusername/smq.git
cd smq
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
docker run suhaibinator/smq:latest
```

### Command-Line Flags

- `--dbpath`: Specifies the path to the SQLite database file. By default, it is set to `./dbtest.db`.
- `--port`: Specifies the port to listen on. By default, it is set to `8097`.

If command-line flags are provided, they take the highest priority and override any other settings.

### Environment Variables

- `DB_PATH`: Overrides the default path to the SQLite database file, unless the `--dbpath` flag is used.
- `PORT`: Overrides the default port to listen on, unless the `--port` flag is used.

Environment variables take priority over default values but are overridden by command-line flags if they are set.

To use an environment variable, export it before running the SMQ binary. For example:

```bash
export DB_PATH=/path/to/your/database.db
export PORT=9000
./smq
```


## Usage

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
