# Suhaib Message Queue

Suhaib Message Queue (SMQ) is a small persistent message broker written in Go. It stores
topics in SQLite and exposes publish, consume, streaming, retention, and bulk-retrieval
operations over gRPC. Go and Python clients are maintained in this repository.

Upgrading an existing integration? See the [client migration guide](MIGRATION.md).

## Requirements

- Go 1.25 or newer. The module pins the development toolchain in `go.mod`.
- A C compiler, because `github.com/mattn/go-sqlite3` uses CGO.
- Docker, optionally.
- Python 3.9 or newer when developing the Python client.

## Run the server

```bash
go run ./cmd/smq
```

The server listens on port `8097` and writes to `./dbtest.db` by default. Command-line
flags override environment variables.

| Flag | Environment | Default | Purpose |
| --- | --- | --- | --- |
| `--dbpath` | `DB_PATH` | `./dbtest.db` | SQLite database path |
| `--port` | `PORT` | `8097` | gRPC port |
| `--allow-remote-writes` | `ALLOW_REMOTE_WRITES` | `true` | Permit create, publish, and delete operations |
| `--server-enable-mtls` | `SERVER_ENABLE_MTLS` | `false` | Require mutual TLS |
| `--server-cert` | `SERVER_CERT_FILE` | — | Server certificate |
| `--server-key` | `SERVER_KEY_FILE` | — | Server private key |
| `--server-ca-cert` | `SERVER_CA_CERT_FILE` | — | CA used to verify clients |
| `--max-recv-message-size` | — | 1 GiB | Maximum received gRPC message size |
| `--max-send-message-size` | — | 1 GiB | Maximum sent gRPC message size |

If mTLS is enabled, all three certificate paths are required and startup fails if any
credential cannot be loaded. The server also implements the standard gRPC health service.
Topic names must contain 1–35 ASCII letters, digits, or underscores.

## Docker

Build and run locally:

```bash
docker build -t smq .
docker run --rm -p 8097:8097 -v smq-data:/data smq
```

Tagged releases publish a single `linux/amd64` and `linux/arm64` manifest:

```bash
docker run --rm -p 8097:8097 -v smq-data:/data ghcr.io/suhaibinator/smq:latest
```

The image runs as a non-root user and stores its database at `/data/smq.db`.

## Go client

```go
package main

import (
	"log"

	"github.com/Suhaibinator/SuhaibMessageQueue/client"
)

func main() {
	c, err := client.NewClient("localhost", "8097", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err := c.CreateTopic("events"); err != nil {
		log.Fatal(err)
	}
	if err := c.Produce("events", []byte("hello")); err != nil {
		log.Fatal(err)
	}
}
```

For explicit timeouts, message limits, or mTLS, use `client.New(client.Options{...})`.
The old `config.ClientTLSConfig` name remains a deprecated alias for compatibility.

## Embedding SMQ in a Go program

The `embedded` package opens an SMQ database inside your process without starting a
listener. It suits a durable local buffer, such as an outbox that forwards events
upstream and deletes them after acknowledgement.

```go
q, err := embedded.Open("/var/lib/app/outbox.db", embedded.Options{})
if err != nil {
	log.Fatal(err)
}
defer q.Close()

ctx := context.Background()
if err := q.CreateTopic(ctx, "events"); err != nil { // idempotent
	log.Fatal(err)
}
offset, err := q.Publish(ctx, "events", payload)
offsets, err := q.PublishBatch(ctx, "events", payloads) // one transaction

batch, err := q.Read(ctx, "events", fromOffset, embedded.ReadOptions{Limit: 500, MaxBytes: 4 << 20})
// ...forward batch; after the upstream acknowledges the last offset:
err = q.DeleteUntilOffset(ctx, "events", batch[len(batch)-1].Offset)

stats, err := q.Stats(ctx, "events") // Count, Bytes, EarliestOffset, LatestOffset, OldestTime
```

Behavior to rely on:

- Offsets start at 1, increase monotonically, and are never reused, even after deletion
  or a restart. `Read` is inclusive of `fromOffset`; continue from the last offset plus one.
- `Publish` and `PublishBatch` return after the commit. A batch is all-or-nothing.
- `ReadOptions.MaxBytes` caps a batch's payload total, but the first message is always
  returned so an oversized message cannot stall a reader.
- The database uses WAL. `SyncNormal` (default) survives process crashes; the newest
  commits can be lost on power loss. `SyncFull` fsyncs every commit.
- A message can be at most 1,000,000,000 bytes, SQLite's default blob limit.
- Only one process may open a database file at a time.

To also expose the standard gRPC API to existing SMQ clients, register it on your own
`grpc.Server`. Remote writes are opt-in:

```go
grpcServer := grpc.NewServer()
q.RegisterGRPC(grpcServer, embedded.GRPCOptions{AllowRemoteWrites: false})
```

See [`embedded/example_test.go`](embedded/example_test.go) for a complete forwarding loop.

## Python client

Install the published client:

```bash
python -m pip install pysmq
```

```python
from pysmq import Client

with Client(host="localhost", port=8097) as client:
    client.connect()
    client.create_topic("events")
    offset = client.produce("events", b"hello")
    print(offset)
```

See [`python/README.md`](python/README.md) for the complete Python API.

## Development

Common operations are exposed through the root Makefile:

```bash
make build
make test
make generate PYTHON_BIN=python3
make proto-check
```

Install Python development dependencies before running Python checks:

```bash
python -m pip install -e './python[dev]'
make python-test PYTHON_BIN=python
make python-build PYTHON_BIN=python
```

Protobuf generation requires `buf`, `protoc-gen-go`, `protoc-gen-go-grpc`, and
`grpcio-tools`. The schema is the source of truth at `api/smq/v1/smq.proto`. Generated
Go and Python files are committed, and CI rejects generation drift and incompatible
wire/JSON schema changes.

## Repository layout

```text
api/smq/v1/             Versioned protobuf API and generated Go bindings
client/                 Public Go client
embedded/               Public in-process queue (no listener)
cmd/smq/                Server executable
internal/config/        Server process configuration
internal/queue/         Storage contract and domain errors
internal/server/        gRPC transport
internal/store/sqlite/  SQLite implementation
python/                 Independently packaged Python distribution
proto/                  Deprecated Go v1 compatibility aliases
examples/               Go and Python examples
```

Server and container releases use `vX.Y.Z` tags. Python releases use
`pysmq-vX.Y.Z` tags so the two artifacts can evolve independently.

## License

MIT. See [`LICENSE`](LICENSE).
