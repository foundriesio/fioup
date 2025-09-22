BINARY_NAME := fioup
BUILD_DIR := bin
MAIN := ./cmd/fioup
LINTER = golangci-lint
TAGS = disable_pkcs11

.PHONY: all build clean test format

all: build

build:
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) -tags $(TAGS) $(MAIN)

format:
	@go fmt ./...

check: format
	$(LINTER) run

test:
	@go test ./...

test-e2e: build
	pytest test/e2e/e2e-test.py --maxfail=1 -vv -k 'test_incremental_updates[False-True-False or test_incremental_updates[False-False-True'

clean:
	@rm -rf $(BUILD_DIR)

