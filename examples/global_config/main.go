package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Suhaibinator/SuhaibMessageQueue/client"
	"github.com/Suhaibinator/SuhaibMessageQueue/config"
)

func main() {
	// Set environment variables for mTLS configuration
	// You can also use command-line flags when starting the application
	// The example below shows how to configure it via environment variables

	// Uncomment to use mTLS in this example
	// os.Setenv(config.ENV_ENABLE_MTLS, "true")
	// os.Setenv(config.ENV_CLIENT_CERT, "/path/to/client.crt")
	// os.Setenv(config.ENV_CLIENT_KEY, "/path/to/client.key")
	// os.Setenv(config.ENV_CA_CERT, "/path/to/ca.crt")

	// You can also set these values in code if not using environment variables
	// config.EnableMTLS = true
	// config.ClientCert = "/path/to/client.crt"
	// config.ClientKey = "/path/to/client.key"
	// config.CACert = "/path/to/ca.crt"

	// Check if certificate files exist when mTLS is enabled
	if config.EnableMTLS {
		if _, err := os.Stat(config.ClientCert); os.IsNotExist(err) {
			log.Fatalf("Client certificate file not found: %s", config.ClientCert)
		}
		if _, err := os.Stat(config.ClientKey); os.IsNotExist(err) {
			log.Fatalf("Client key file not found: %s", config.ClientKey)
		}
		if _, err := os.Stat(config.CACert); os.IsNotExist(err) {
			log.Fatalf("CA certificate file not found: %s", config.CACert)
		}
	}

	// Get TLS config from global configuration
	tlsConfig := config.GetClientTLSConfig()

	// Create a client with optional mTLS
	// If tlsConfig is nil, it will use an insecure connection
	smqClient, err := client.NewClient("localhost", config.Port, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer smqClient.Close()

	// Now we can use the client as normal
	err = smqClient.CreateTopic("example-topic")
	if err != nil {
		log.Fatalf("Failed to create topic: %v", err)
	}

	// Produce a message
	message := []byte("Hello, configured secure world!")
	err = smqClient.Produce("example-topic", message)
	if err != nil {
		log.Fatalf("Failed to produce message: %v", err)
	}
	fmt.Println("Successfully produced message")

	// Consume the message
	receivedMsg, offset, err := smqClient.ConsumeLatest("example-topic")
	if err != nil {
		log.Fatalf("Failed to consume message: %v", err)
	}
	fmt.Printf("Received message: %s (offset: %d)\n", string(receivedMsg), offset)

	// Log connection security status
	if config.EnableMTLS {
		fmt.Println("Connection is secured with mTLS")
	} else {
		fmt.Println("Connection is using insecure transport")
	}
}
