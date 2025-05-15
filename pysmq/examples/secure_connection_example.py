#!/usr/bin/env python3
"""
Example showing how to establish a secure connection to the SMQ server using mTLS.
"""
import json
import time
import os

from pysmq.client import Client
from pysmq.config import ClientTLSConfig
from pysmq.exceptions import SMQAuthenticationError, SMQConnectionError


def main():
    # Replace with actual server details
    host = "localhost"
    port = 8097
    
    # Replace with actual paths to your TLS certificate files
    # These paths should point to your client certificate, client key, 
    # and CA certificate that signed the server's certificate
    cert_file = os.environ.get("CLIENT_CERT", "/path/to/client.crt")
    key_file = os.environ.get("CLIENT_KEY", "/path/to/client.key")
    ca_file = os.environ.get("CA_CERT", "/path/to/ca.crt")
    
    # Check if the certificate files exist
    for file_path, file_name in [
        (cert_file, "Client certificate"),
        (key_file, "Client key"),
        (ca_file, "CA certificate")
    ]:
        if not os.path.exists(file_path):
            print(f"Warning: {file_name} file not found at {file_path}")
            print("This example requires valid TLS certificates to work.")
            print("You can set the paths via environment variables:")
            print("  CLIENT_CERT: Path to client certificate file")
            print("  CLIENT_KEY: Path to client key file")
            print("  CA_CERT: Path to CA certificate file")
            return
    
    # Create the TLS configuration
    tls_config = ClientTLSConfig(
        cert_file=cert_file,
        key_file=key_file,
        ca_file=ca_file,
    )
    
    try:
        print("Creating secure client with mTLS...")
        # Create a client with mTLS configuration
        with Client(host=host, port=port, tls_config=tls_config) as client:
            # Connect to the server
            print("Connecting to SMQ server securely...")
            client.connect()
            print("Connected securely!")
            
            # Example: Create a topic
            topic = "secure-example-topic"
            try:
                print(f"Creating topic '{topic}'...")
                client.create_topic(topic)
                print(f"Topic '{topic}' created successfully!")
            except Exception as e:
                # Topic may already exist, which is okay
                print(f"Note: {e}")
            
            # Example: Publish a message
            message = {
                "id": 1,
                "timestamp": time.time(),
                "content": "Secure message",
                "secure": True,
            }
            message_bytes = json.dumps(message).encode('utf-8')
            
            print("Publishing a secure message...")
            offset = client.produce(topic, message_bytes)
            print(f"Message published securely at offset {offset}")
            
            # Example: Consume the message
            print("Consuming the message...")
            received_bytes, received_offset = client.consume(topic, offset)
            
            # Decode and print the message
            received_message = json.loads(received_bytes.decode('utf-8'))
            print(f"Received secure message at offset {received_offset}:")
            print(f"  ID: {received_message.get('id')}")
            print(f"  Timestamp: {received_message.get('timestamp')}")
            print(f"  Content: {received_message.get('content')}")
            print(f"  Secure: {received_message.get('secure')}")
            
            print("Secure connection example completed successfully!")
    
    except SMQAuthenticationError as e:
        print(f"TLS Authentication Error: {e}")
        print("Make sure your certificates are correctly configured and valid.")
    except SMQConnectionError as e:
        print(f"Connection Error: {e}")
        print("Make sure the server is running and configured for mTLS.")
    except Exception as e:
        print(f"Unexpected Error: {e}")


if __name__ == "__main__":
    main()
