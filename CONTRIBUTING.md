# Contributing

## Setup

Install the Go toolchain declared in `go.mod`, a C compiler for SQLite, Python 3.9 or
newer, and the Python development dependencies:

```bash
python -m venv .venv
. .venv/bin/activate
python -m pip install -e './python[dev]'
```

Protobuf changes additionally require Buf, `protoc-gen-go`, and
`protoc-gen-go-grpc`.

## Before opening a pull request

```bash
make generate PYTHON_BIN=python
make lint PYTHON_BIN=python
make test
make python-test PYTHON_BIN=python
make python-build PYTHON_BIN=python
```

Commit generated bindings whenever the schema changes. Do not change or reuse existing
protobuf field numbers. Server/container tags use `vX.Y.Z`; Python package tags use
`pysmq-vX.Y.Z`.
