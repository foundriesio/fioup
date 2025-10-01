BINARY_NAME := fioup
BUILD_DIR := bin
MAIN := ./cmd/fioup
LINTER = golangci-lint
TAGS = disable_pkcs11
DEBS_DIR ?= $(BUILD_DIR)

.PHONY: all build clean test format manpages bash-completion

all: build manpages bash-completion

build:
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) -tags $(TAGS) $(MAIN)

manpages:
	@go run -tags disable_pkcs11,disable_main $(MAIN) manpages $(BUILD_DIR)/man

bash-completion:
	@go run -tags disable_pkcs11,disable_main $(MAIN) bash-completion $(BUILD_DIR)/bash-completion

deb:
	docker build --output $(DEBS_DIR) -f debian/Dockerfile .

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

