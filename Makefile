BINARY_NAME := fioup
BUILD_DIR := bin
MAIN := ./cmd/fioup
LINTER = golangci-lint

.PHONY: all build clean test format

all: build

build:
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN)

format:
	@go fmt ./...

check: format
	$(LINTER) run

test:
	@go test ./...

clean:
	@rm -rf $(BUILD_DIR)

