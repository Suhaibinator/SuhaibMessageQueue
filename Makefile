PYTHON_BIN ?= python3

.PHONY: build check check-generated fmt generate lint proto-check python-build python-test test

build:
	go build ./cmd/smq

generate:
	PYTHON_BIN=$(PYTHON_BIN) ./scripts/generate.sh

check-generated: generate
	git diff --exit-code -- api/smq/v1 python/src/pysmq/_generated

fmt:
	gofmt -w $$(find api client cmd config embedded errors examples internal proto -name '*.go' -type f)
	cd python && $(PYTHON_BIN) -m ruff format src tests ../examples/python ../scripts/fix_python_proto_import.py

lint:
	go vet ./...
	cd python && $(PYTHON_BIN) -m ruff check src tests ../examples/python ../scripts/fix_python_proto_import.py
	cd python && $(PYTHON_BIN) -m ruff format --check src tests ../examples/python ../scripts/fix_python_proto_import.py

proto-check:
	buf lint
	buf breaking --against '.git#branch=origin/main'

test:
	go test -race -shuffle=on ./...

python-test:
	cd python && $(PYTHON_BIN) -m pytest --cov=pysmq --cov-report=term-missing

python-build:
	cd python && $(PYTHON_BIN) -m build

check: lint test python-test check-generated
