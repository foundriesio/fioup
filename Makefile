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
	docker build --progress=plain --output $(DEBS_DIR) -f debian/Dockerfile .

format:
	@go fmt ./...

check: format
	$(LINTER) run

test:
	@go test ./...

test-e2e-single-command:
	pytest test/e2e/e2e-test.py --maxfail=1 -vv -k 'test_incremental_updates[False-False-True'

test-e2e-granular:
	pytest test/e2e/e2e-test.py --maxfail=1 -vv -k 'test_incremental_updates[False-True-False'

test-e2e: test-e2e-granular test-e2e-single-command

clean:
	@rm -rf $(BUILD_DIR)

