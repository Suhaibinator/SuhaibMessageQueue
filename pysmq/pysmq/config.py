"""
Configuration and constants for the SMQ Python client.
"""
from dataclasses import dataclass
from typing import Optional


# Special offset value for heartbeat messages
SMQ_SPECIAL_OFFSET_HEARTBEAT = -1


@dataclass
class ClientTLSConfig:
    """
    Configuration for mTLS (mutual TLS) authentication.
    
    Attributes:
        cert_file: Path to client's certificate file (PEM format)
        key_file: Path to client's private key file (PEM format)
        ca_file: Path to CA's certificate file (PEM format) to verify the server
    """
    cert_file: str
    key_file: str
    ca_file: str


class ClientConfig:
    """
    Default configuration values for the SMQ client.
    """
    # Default timeout for unary RPC calls (in seconds)
    DEFAULT_TIMEOUT = 100.0
    
    # Default timeout for topic creation (in seconds)
    DEFAULT_TOPIC_TIMEOUT = 1.0
    
    # Default timeout for deletion operations (in seconds)
    DEFAULT_DELETE_TIMEOUT = 1.0
    
    # Default max message size in MB
    DEFAULT_MAX_MESSAGE_SIZE_MB = 1024
