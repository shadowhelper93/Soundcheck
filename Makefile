.PHONY: all build check test lint fmt-check ci clean

BIN_DIR := bin
BINARY := $(BIN_DIR)/soundcheck

GOPATH ?= $(shell go env GOPATH)
GOLANGCI_LINT ?= $(shell which golangci-lint 2>/dev/null || echo $(GOPATH)/bin/golangci-lint)

all: build

build:
	@mkdir -p $(BIN_DIR)
	@if [ -d "cmd/soundcheck" ]; then \
		go build -trimpath -ldflags "-s -w" -o $(BINARY) ./cmd/soundcheck; \
	else \
		go build -trimpath ./...; \
	fi

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files: run 'go fmt ./...'" && gofmt -l . && exit 1)

test:
	go test -v ./...

lint:
	$(GOLANGCI_LINT) run ./...

check: fmt-check lint test
	go vet ./...

ci: check
	go test -v -race ./...

clean:
	rm -rf $(BIN_DIR)
