"""
Python client for SuhaibMessageQueue.

This package provides a Python client for interacting with SuhaibMessageQueue,
a simple and efficient message queue service.
"""

from .client import Client
from .config import ClientTLSConfig, ClientConfig, SMQ_SPECIAL_OFFSET_HEARTBEAT
from .exceptions import (
    SMQException,
    SMQConnectionError,
    SMQAuthenticationError,
    SMQProduceError,
    SMQConsumeError,
    SMQTopicCreationError,
    SMQOffsetError,
    SMQTimeoutError,
)

# Package information
__version__ = "0.1.0"  # Will be overwritten by setuptools_scm
__author__ = "Suhaib"
__all__ = [
    "Client",
    "ClientTLSConfig",
    "ClientConfig",
    "SMQ_SPECIAL_OFFSET_HEARTBEAT",
    "SMQException",
    "SMQConnectionError",
    "SMQAuthenticationError",
    "SMQProduceError",
    "SMQConsumeError",
    "SMQTopicCreationError",
    "SMQOffsetError",
    "SMQTimeoutError",
]
