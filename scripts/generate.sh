#!/bin/sh
set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PYTHON_BIN=${PYTHON_BIN:-python3}

cd "$REPO_ROOT"

if ! command -v buf >/dev/null 2>&1; then
    echo "error: buf is required for deterministic protobuf generation" >&2
    exit 1
fi

buf generate

"$PYTHON_BIN" -m grpc_tools.protoc \
    --proto_path=api/smq/v1 \
    --python_out=python/src/pysmq/_generated \
    --grpc_python_out=python/src/pysmq/_generated \
    api/smq/v1/smq.proto
"$PYTHON_BIN" scripts/fix_python_proto_import.py
