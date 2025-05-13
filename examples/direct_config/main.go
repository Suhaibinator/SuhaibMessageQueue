package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Suhaibinator/SuhaibMessageQueue/client"
	"github.com/Suhaibinator/SuhaibMessageQueue/config"
)

func main() {
	// Example of configuring mTLS through environment variables
	// os.Setenv(config.ENV_CLIENT_CERT, "/path/to/client.crt")
	// os.Setenv(config.ENV_CLIENT_KEY, "/path/to/client.key")
	// os.Setenv(config.ENV_CA_CERT, "/path/to/ca.crt")
	// os.Setenv(config.ENV_ENABLE_MTLS, "true")

	// Example of configuring mTLS directly
	enableMTLS := false // Set to true to enable mTLS

	// Sample paths - replace with actual paths when using mTLS
	clientCertPath := "/path/to/client.crt"
	clientKeyPath := "/path/to/client.key"
	caCertPath := "/path/to/ca.crt"

	// Create a TLSConfig if mTLS is enabled
	var tlsConfig *config.TLSConfig
	if enableMTLS {
		// Check if certificate files exist
		if _, err := os.Stat(clientCertPath); os.IsNotExist(err) {
			log.Fatalf("Client certificate file not found: %s", clientCertPath)
		}
		if _, err := os.Stat(clientKeyPath); os.IsNotExist(err) {
			log.Fatalf("Client key file not found: %s", clientKeyPath)
		}
		if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
			log.Fatalf("CA certificate file not found: %s", caCertPath)
		}

		tlsConfig = &config.TLSConfig{
			CertFile: clientCertPath,
			KeyFile:  clientKeyPath,
			CAFile:   caCertPath,
		}
	}

	// Create a client with optional mTLS
	// If tlsConfig is nil, it will use an insecure connection
	smqClient, err := client.NewClient("localhost", "8097", tlsConfig)
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
	message := []byte("Hello, secure world!")
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
}
