# Client Migration Guide

This guide covers upgrading from SMQ v1.0.7 or earlier to the first release that
contains the versioned API and packaging layout. The protobuf wire contract and SQLite
data format are unchanged, so clients and servers can be upgraded independently.

## At a glance

| Client or integration | Required action |
| --- | --- |
| Go client using `client.NewClient` | None; the existing constructor remains supported |
| Go code importing the old `proto` package | None immediately; migrate to `api/smq/v1` when convenient |
| Python client using public `pysmq` imports | Use Python 3.9+ and reinstall the package |
| Python code importing `pysmq.proto` directly | Change the import to `pysmq._generated`, or preferably use the public client API |
| Tooling that reads `smq.proto` from this repository | Use `api/smq/v1/smq.proto` |
| Raw gRPC clients | No regeneration is required; handle the status and offset behavior described below |

There is no database migration, topic recreation, or message replay step.

## Go clients

### Existing high-level clients

The original constructor still works:

```go
c, err := client.NewClient("localhost", "8097", nil)
if err != nil {
	log.Fatal(err)
}
defer c.Close()
```

`config.ClientTLSConfig` is now a deprecated alias of `client.TLSConfig`, so existing
mTLS calls also continue to compile.

For new code, use `client.New` and explicit options:

```go
package main

import (
	"log"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/client"
)

func main() {
	c, err := client.New(client.Options{
		Address:           "localhost",
		Port:              "8097",
		RequestTimeout:    30 * time.Second,
		MaxSendMsgSize:    16 * 1024 * 1024,
		MaxReceiveMsgSize: 16 * 1024 * 1024,
		TLS: &client.TLSConfig{
			CertFile: "client.crt",
			KeyFile:  "client.key",
			CAFile:   "ca.crt",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Printf("close SMQ client: %v", err)
		}
	}()
}
```

`Address` and `Port` are separate values, message-size options are bytes, and non-positive
timeout or size values use the defaults. If TLS is configured, the certificate, key, and
CA paths must all be present. TLS 1.2 is the minimum supported version.

`Client.Close` now returns an error. Ordinary calls such as `defer c.Close()` remain valid,
but code that assigns `Close` to a `func()` or requires an interface with `Close()` must
change that signature to `Close() error`.

### Generated protobuf imports

The preferred generated-code import moved from:

```go
import pb "github.com/Suhaibinator/SuhaibMessageQueue/proto"
```

to:

```go
import smqv1 "github.com/Suhaibinator/SuhaibMessageQueue/api/smq/v1"
```

The old package remains as a deprecated compatibility layer and aliases the v1 messages,
clients, servers, method constants, and service descriptor. This means an immediate import
rewrite is optional. New code should use `smqv1` so a future API version can coexist with v1.

## Python clients

### Runtime and installation

Python 3.9 or newer is now required. Upgrade the installed distribution normally:

```bash
python -m pip install --upgrade pysmq
```

For an editable checkout, the project path changed:

```bash
# Before
python -m pip install -e './pysmq[dev]'

# After
python -m pip install -e './python[dev]'
```

Runtime dependencies are constrained to compatible gRPC and protobuf major versions:
`grpcio>=1.71,<2` and `protobuf>=5.29,<7`.

### Public API imports

Public imports are unchanged:

```python
from pysmq import Client, ClientTLSConfig

with Client("localhost", 8097) as client:
    client.connect()
```

Imports from `pysmq.client`, `pysmq.config`, and `pysmq.exceptions` also remain available.

Generated stubs are an internal implementation detail, but code that used them directly
must update:

```python
# Before
from pysmq.proto import smq_pb2, smq_pb2_grpc

# After
from pysmq._generated import smq_pb2, smq_pb2_grpc
```

Prefer the public `Client` class unless an RPC is unavailable there.

`Client.stream_produce` accepts a normal synchronous iterable of `bytes`. No async
generator is required:

```python
last_offset = client.stream_produce("events", [b"one", b"two"])
```

## Behavior changes visible to all clients

### Produced offsets

`ProduceResponse.offset` and the response from `StreamProduce` now contain the actual
SQLite offset. Earlier servers returned the protobuf default (`0`) even after a successful
insert. Remove any client logic that treats a successful zero offset as authoritative or
as the first persisted message. SQLite topic offsets begin at `1`.

The Go high-level `Client.Produce` still returns only an error; this change affects the
Python client and raw generated clients that read `ProduceResponse`.

### Canonical gRPC status codes

Server errors now use stable gRPC status codes:

| Condition | Status code |
| --- | --- |
| Invalid topic name | `INVALID_ARGUMENT` |
| Missing topic or message | `NOT_FOUND` |
| Requested offset is beyond the latest offset | `OUT_OF_RANGE` |
| Topic is empty | `FAILED_PRECONDITION` |
| Remote writes are disabled | `PERMISSION_DENIED` |
| Request canceled or deadline exceeded | Matching context status |
| Unexpected storage failure | `INTERNAL` |

Clients should branch on status codes or the public Python exception classes instead of
matching error text.

### Topic names and mTLS

Topic names must contain 1–35 ASCII letters, digits, or underscores. Hyphens and spaces
are not accepted.

When server mTLS is enabled, startup now fails if the certificate, key, or client CA is
missing or invalid. Check certificate paths before upgrading and ensure both client and
server support TLS 1.2 or newer.

## Recommended rollout

1. Verify client runtimes: Go 1.25+ for source builds and Python 3.9+ for PySMQ.
2. Check topic names and remove code that expects a successful produced offset of zero.
3. Validate mTLS certificates and update any error handling to use gRPC status codes.
4. Deploy the server. Existing wire-compatible clients can be upgraded separately.
5. Upgrade client dependencies and adopt the versioned Go import when convenient.
6. Run a smoke test that creates a topic, produces a message, and consumes the returned
   offset.

Rollback does not require restoring the database. If application code has started relying
on canonical status codes or real produced offsets, account for the older server behavior
before rolling the server back.
