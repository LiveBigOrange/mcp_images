DEFAULT_VERSION = 0.2.0
VERSION ?= $(or $(shell cat VERSION 2>/dev/null),$(DEFAULT_VERSION))
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

BINARY_NAME = mcp_images
CMD_PATH = ./cmd/mcp_images
BIN_DIR = bin

.PHONY: build build-windows build-linux build-darwin build-all test clean install run lint

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_PATH)

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)_windows_amd64.exe $(CMD_PATH)

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)_linux_amd64 $(CMD_PATH)

build-darwin:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)_darwin_arm64 $(CMD_PATH)

build-all: build-windows build-linux build-darwin

test:
	go test -v ./...

test-cover:
	go test -cover ./...

clean:
	rm -rf $(BIN_DIR)/

install: build
	cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

run:
	go run $(CMD_PATH)

lint:
	go vet ./...
