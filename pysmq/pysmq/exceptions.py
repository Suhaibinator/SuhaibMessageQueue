"""
Custom exceptions for the SMQ Python client.
"""
import grpc
from typing import Optional


class SMQException(Exception):
    """Base exception class for all SMQ-related exceptions."""
    
    def __init__(self, message: str, original_error: Optional[Exception] = None):
        super().__init__(message)
        self.original_error = original_error


class SMQConnectionError(SMQException):
    """Exception raised when there's an issue with the connection to the SMQ server."""
    pass


class SMQAuthenticationError(SMQConnectionError):
    """Exception raised when there's an authentication issue (e.g., TLS-related)."""
    pass


class SMQProduceError(SMQException):
    """Exception raised when there's an issue with producing messages."""
    pass


class SMQConsumeError(SMQException):
    """Exception raised when there's an issue with consuming messages."""
    pass


class SMQTopicCreationError(SMQException):
    """Exception raised when there's an issue with creating a topic."""
    pass


class SMQOffsetError(SMQException):
    """Exception raised when there's an issue with offsets."""
    pass


class SMQTimeoutError(SMQException):
    """Exception raised when an operation times out."""
    pass


def map_grpc_error(error: grpc.RpcError, default_exception=SMQException) -> SMQException:
    """
    Map a gRPC error to an appropriate SMQ exception.
    
    Args:
        error: The gRPC error to map
        default_exception: The default exception class to use if no specific mapping is found
        
    Returns:
        An instance of the appropriate SMQ exception
    """
    code = error.code()
    details = error.details() if hasattr(error, 'details') else str(error)
    
    if code == grpc.StatusCode.DEADLINE_EXCEEDED:
        return SMQTimeoutError(f"Operation timed out: {details}", error)
    elif code == grpc.StatusCode.UNAVAILABLE:
        return SMQConnectionError(f"Server unavailable: {details}", error)
    elif code == grpc.StatusCode.UNAUTHENTICATED:
        return SMQAuthenticationError(f"Authentication failed: {details}", error)
    elif code == grpc.StatusCode.INVALID_ARGUMENT:
        # Attempt to categorize based on error message
        msg = str(error).lower()
        if 'topic' in msg and ('create' in msg or 'new' in msg):
            return SMQTopicCreationError(f"Failed to create topic: {details}", error)
        elif 'offset' in msg:
            return SMQOffsetError(f"Invalid offset: {details}", error)
        elif 'produce' in msg or 'message' in msg:
            return SMQProduceError(f"Failed to produce message: {details}", error)
        elif 'consume' in msg:
            return SMQConsumeError(f"Failed to consume message: {details}", error)
    
    # Default case
    return default_exception(f"SMQ operation failed: {details}", error)
