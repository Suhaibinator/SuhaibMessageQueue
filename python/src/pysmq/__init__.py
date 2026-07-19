"""Python client for SuhaibMessageQueue."""

from importlib.metadata import PackageNotFoundError, version

from .client import Client
from .config import SMQ_SPECIAL_OFFSET_HEARTBEAT, ClientConfig, ClientTLSConfig
from .exceptions import (
    SMQAuthenticationError,
    SMQConnectionError,
    SMQConsumeError,
    SMQException,
    SMQOffsetError,
    SMQProduceError,
    SMQTimeoutError,
    SMQTopicCreationError,
)

try:
    __version__ = version("pysmq")
except PackageNotFoundError:
    __version__ = "0+unknown"

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
