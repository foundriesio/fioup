BINARY_NAME := fioup
BUILD_DIR := bin
MAIN := ./cmd/fioup
LINTER = golangci-lint
TAGS = disable_pkcs11
DEBS_DIR ?= $(BUILD_DIR)

PKG := main
VERSION := $(shell git describe --tags --always --dirty)
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X '$(PKG).Version=$(VERSION)' -X '$(PKG).Commit=$(COMMIT)' -X '$(PKG).Date=$(DATE)'

.PHONY: all build clean test format manpages bash-completion

all: build manpages bash-completion

build:
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) -tags $(TAGS) $(MAIN)

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
	@go test -tags $(TAGS) ./...

test-e2e-single-command:
	pytest test/e2e/e2e-test.py --maxfail=1 -vv -k 'test_incremental_updates[False-False-True'

test-e2e-granular:
	pytest test/e2e/e2e-test.py --maxfail=1 -vv -k 'test_incremental_updates[False-True-False'

test-e2e-daemon:
	pytest test/e2e/e2e-test.py --maxfail=1 -vv -k 'test_fioup_daemon'

test-e2e: test-e2e-granular test-e2e-single-command

clean:
	@rm -rf $(BUILD_DIR)

