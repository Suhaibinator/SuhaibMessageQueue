#!/usr/bin/env python3
"""
Example showing how to consume messages from SMQ using the Python client.
"""
import json
import time
from typing import Tuple

from pysmq.client import Client
from pysmq.config import ClientTLSConfig


def process_message(message_bytes: bytes, offset: int) -> None:
    """
    Process a received message.
    
    Args:
        message_bytes: The message content as bytes.
        offset: The offset of the message.
    """
    try:
        # Decode the message from bytes to a Python dictionary
        message = json.loads(message_bytes.decode('utf-8'))
        print(f"Received message at offset {offset}:")
        print(f"  ID: {message.get('id')}")
        print(f"  Timestamp: {message.get('timestamp')}")
        print(f"  Content: {message.get('content')}")
        print("-" * 40)
    except json.JSONDecodeError:
        print(f"Received non-JSON message at offset {offset}: {message_bytes}")
    except Exception as e:
        print(f"Error processing message at offset {offset}: {e}")


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
        
        # Specify the topic to consume from
        topic = "example-topic"
        
        # Demonstration 1: Consume the latest message
        print("\n=== Consuming latest message ===")
        try:
            message_bytes, offset = client.consume_latest(topic)
            process_message(message_bytes, offset)
        except Exception as e:
            print(f"Error consuming latest message: {e}")
        
        # Demonstration 2: Consume the earliest message
        print("\n=== Consuming earliest message ===")
        try:
            message_bytes, offset = client.consume_earliest(topic)
            process_message(message_bytes, offset)
        except Exception as e:
            print(f"Error consuming earliest message: {e}")
        
        # Demonstration 3: Bulk retrieve messages
        print("\n=== Bulk retrieving messages ===")
        try:
            # Get the earliest offset
            start_offset = client.get_earliest_offset(topic)
            # Retrieve up to 10 messages
            response = client.bulk_retrieve(topic, start_offset, 10)
            
            print(f"Retrieved {response.count} messages, next offset: {response.next_offset}")
            for message in response.messages:
                process_message(message.message, message.offset)
        except Exception as e:
            print(f"Error bulk retrieving messages: {e}")
        
        # Demonstration 4: Stream consumption (continuous)
        print("\n=== Starting stream consumption (will run for 30 seconds) ===")
        try:
            # Get the latest offset to start consuming from
            start_offset = client.get_latest_offset(topic)
            print(f"Starting consumption from offset {start_offset}")
            
            # Set an end time for the demo (30 seconds from now)
            end_time = time.time() + 30
            
            def stream_consumer():
                """Generator that yields True until the end time is reached."""
                while time.time() < end_time:
                    yield True
                return
            
            # Start consuming messages
            print("Waiting for new messages... (Ctrl+C to stop)")
            consumer = stream_consumer()
            
            for message_bytes, offset in client.stream_consume(topic, start_offset):
                process_message(message_bytes, offset)
                
                # Check if we should stop
                try:
                    next(consumer)
                except StopIteration:
                    print("Time limit reached, stopping consumption")
                    break
                
                # If you want to stop based on some condition,
                # you can add a break statement here
        
        except KeyboardInterrupt:
            print("Interrupted by user, stopping consumption")
        except Exception as e:
            print(f"Error in stream consumption: {e}")
        
        print("\nSubscribe example completed!")


if __name__ == "__main__":
    main()
