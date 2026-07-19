#!/usr/bin/env python3
"""
Publish messages to SMQ using the Python client.
"""

import json
import time

from pysmq.client import Client


def main():
    # Replace with actual server details
    host = "localhost"
    port = 8097

    # Create a client with default settings (insecure connection)
    with Client(host=host, port=port) as client:
        # Connect to the server (this is optional but a good test)
        print("Connecting to SMQ server...")
        client.connect()
        print("Connected!")

        # Create a topic if it doesn't exist
        topic = "example_topic"
        try:
            print(f"Creating topic '{topic}'...")
            client.create_topic(topic)
            print(f"Topic '{topic}' created successfully!")
        except Exception as e:
            # Topic may already exist, which is okay
            print(f"Note: {e}")

        # Publish a few messages
        for i in range(5):
            # Create a sample message (convert to bytes)
            message = {
                "id": i,
                "timestamp": time.time(),
                "content": f"Message #{i}",
            }
            message_bytes = json.dumps(message).encode("utf-8")

            # Publish the message
            print(f"Publishing message {i}...")
            offset = client.produce(topic, message_bytes)
            print(f"Message published at offset {offset}")

            # Small delay between messages for demonstration
            time.sleep(0.5)

        print("All messages published successfully!")


if __name__ == "__main__":
    main()
