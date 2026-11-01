export PATH := /opt/homebrew/bin:/usr/local/bin:/usr/local/go/bin:$(PATH)
PATH := /opt/homebrew/bin:/usr/local/bin:/usr/local/go/bin:$(PATH)

.PHONY: all build ui-build cross-build check test lint fmt-check ci clean release-check

BIN_DIR := bin
BINARY := $(BIN_DIR)/soundcheck
RELEASE_DIR := $(BIN_DIR)/release
VERSION ?= $(shell PATH="$(PATH)" git describe --tags --always --dirty 2>/dev/null || echo "v2.0.1")

GOPATH ?= $(shell PATH="$(PATH)" go env GOPATH 2>/dev/null || echo $(HOME)/go)
GOLANGCI_LINT ?= $(shell PATH="$(PATH)" which golangci-lint 2>/dev/null || echo $(GOPATH)/bin/golangci-lint)

all: build

ui-build:
	@if [ -d "ui" ]; then \
		echo "==> Building UI assets..."; \
		npm --prefix ui run build; \
	fi

build: ui-build
	@mkdir -p $(BIN_DIR)
	@if [ -d "cmd/soundcheck" ]; then \
		go build -trimpath -ldflags "-s -w -X main.version=$(VERSION) -X github.com/soundcheck/soundcheck/internal/transport/cli.Version=$(VERSION)" -o $(BINARY) ./cmd/soundcheck; \
	else \
		go build -trimpath ./...; \
	fi

cross-build: ui-build
	@mkdir -p $(RELEASE_DIR)
	@echo "==> Cross-compiling for 6 release targets..."
	@for target in "darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64" "windows/amd64" "windows/arm64"; do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="$(RELEASE_DIR)/soundcheck_$${os}_$${arch}$$ext"; \
		echo "--> Building $$os/$$arch -> $$out"; \
		CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" go build -trimpath \
			-ldflags "-s -w -X main.version=$(VERSION) -X github.com/soundcheck/soundcheck/internal/transport/cli.Version=$(VERSION)" \
			-o "$$out" ./cmd/soundcheck || exit 1; \
	done
	@echo "==> Release binaries built in $(RELEASE_DIR):"
	@ls -lh $(RELEASE_DIR)

release-check: build
	@./scripts/build.sh $(VERSION)

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
