# =============================================================================
# Recall — Developer Makefile
# =============================================================================
# Auto-discovers the Go binary so `make` works even when ~/go/bin is not in
# the shell PATH (common in non-login shell environments used by AI agents,
# CI runners without profile sourcing, and sudo contexts).
# =============================================================================

BINARY_NAME  := recall
BUILD_DIR    := bin
VERSION      := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE   := $(shell date -u '+%Y-%m-%d')
LDFLAGS      := -ldflags "-X github.com/Om-Rohilla/recall/cmd.Version=$(VERSION) \
                           -X github.com/Om-Rohilla/recall/cmd.BuildDate=$(BUILD_DATE)"
BUILD_TAGS   := -tags sqlite_fts5
PKG          := ./...

# ---------------------------------------------------------------------------
# Auto-discover Go — checks PATH first, then common install locations.
# Override with: make test GO=/path/to/go
# ---------------------------------------------------------------------------
GO ?= $(shell \
  command -v go 2>/dev/null || \
  ls $(HOME)/go/bin/go 2>/dev/null || \
  ls /usr/local/go/bin/go 2>/dev/null || \
  ls /usr/lib/go/bin/go 2>/dev/null || \
  echo "go-not-found" \
)

_check_go:
	@if [ "$(GO)" = "go-not-found" ]; then \
	  echo ""; \
	  echo "  ✗ Go toolchain not found. Install Go from https://go.dev/dl/"; \
	  echo "    or set: make <target> GO=/path/to/go"; \
	  echo ""; \
	  exit 1; \
	fi
	@$(GO) version > /dev/null

.PHONY: all build clean install run fmt vet lint snapshot release-dry-run \
        test test-fast test-race test-security test-capture test-vault \
        test-encrypt coverage _check_go

# ---------------------------------------------------------------------------
# Default
# ---------------------------------------------------------------------------
all: vet build

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build: _check_go
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(BUILD_TAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "  ✓ Built $(BUILD_DIR)/$(BINARY_NAME) ($(VERSION))"

# ---------------------------------------------------------------------------
# Run (dev helper)
# ---------------------------------------------------------------------------
run: _check_go
	$(GO) run $(BUILD_TAGS) . $(ARGS)

# ---------------------------------------------------------------------------
# Install to GOPATH/bin (or ~/go/bin as fallback)
# ---------------------------------------------------------------------------
install: build
	@DEST="$$($(GO) env GOPATH)/bin"; \
	 mkdir -p "$$DEST" && \
	 cp $(BUILD_DIR)/$(BINARY_NAME) "$$DEST/$(BINARY_NAME)" && \
	 echo "  ✓ Installed to $$DEST/$(BINARY_NAME)"

# ---------------------------------------------------------------------------
# Code quality
# ---------------------------------------------------------------------------
fmt: _check_go
	@$(GO) fmt $(PKG)
	@echo "  ✓ Formatted"

vet: _check_go
	@$(GO) vet $(PKG)
	@echo "  ✓ Vet clean"

lint: vet
	@if command -v staticcheck > /dev/null 2>&1; then \
	  staticcheck $(PKG); \
	  echo "  ✓ Staticcheck clean"; \
	else \
	  echo "  ⚠  staticcheck not installed (optional). Install with:"; \
	  echo "     $(GO) install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

# ---------------------------------------------------------------------------
# Tests — tiered for speed and focus
# ---------------------------------------------------------------------------

## test: full suite with race detector (authoritative — use in CI)
test: _check_go
	@echo "Running full test suite with -race …"
	$(GO) test $(BUILD_TAGS) $(PKG) -count=1 -race -timeout 300s

## test-fast: full suite WITHOUT race detector (~5× faster, good for rapid iteration)
test-fast: _check_go
	@echo "Running fast test suite (no race detector) …"
	$(GO) test $(BUILD_TAGS) $(PKG) -count=1 -timeout 120s

## test-security: secret filtering + encrypt/decrypt tests only (< 2 s)
test-security: _check_go
	@echo "Running security-specific tests …"
	$(GO) test $(BUILD_TAGS) ./tests/ -run "TestSecretFilter|TestEncryptDecrypt|TestArgon2|TestDifferent|TestTampered|TestWrongPassword|TestGenerateSalt|TestPackUnpack|TestFTS" \
	      -count=1 -v -timeout 30s

## test-capture: capture package unit tests with race (< 5 s)
test-capture: _check_go
	@echo "Running capture package tests (race) …"
	$(GO) test $(BUILD_TAGS) ./internal/capture/ -count=1 -race -v -timeout 30s

## test-vault: vault package tests only (< 20 s)
test-vault: _check_go
	@echo "Running vault package tests …"
	$(GO) test $(BUILD_TAGS) ./internal/vault/ -count=1 -race -timeout 60s

## test-encrypt: encryption roundtrip tests only (< 1 s)
test-encrypt: _check_go
	@echo "Running encrypt/decrypt tests …"
	$(GO) test $(BUILD_TAGS) ./tests/ -run "TestEncrypt" -count=1 -v -timeout 15s

## test-race: alias for the authoritative race-enabled run
test-race: test

# ---------------------------------------------------------------------------
# Coverage report
# ---------------------------------------------------------------------------
coverage: _check_go
	@echo "Generating coverage report …"
	$(GO) test $(BUILD_TAGS) $(PKG) -coverprofile=coverage.out -timeout 300s
	$(GO) tool cover -html=coverage.out
	@echo "  ✓ Coverage report opened in browser"

# ---------------------------------------------------------------------------
# Release helpers
# ---------------------------------------------------------------------------
snapshot:
	goreleaser release --snapshot --clean

release-dry-run:
	goreleaser check

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------
clean:
	rm -rf $(BUILD_DIR) coverage.out
	$(GO) clean -testcache
	@echo "  ✓ Clean"

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
help:
	@echo ""
	@echo "  Recall Developer Makefile"
	@echo ""
	@echo "  Build"
	@echo "    make build          Build binary to $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "    make install        Build and install to \$$(GOPATH)/bin"
	@echo "    make run ARGS=...   Run in dev mode"
	@echo ""
	@echo "  Code quality"
	@echo "    make fmt            Auto-format all Go files"
	@echo "    make vet            Run go vet"
	@echo "    make lint           Run vet + staticcheck"
	@echo ""
	@echo "  Tests (tiered by speed)"
	@echo "    make test-security  Secret filtering + crypto only  (~2 s)"
	@echo "    make test-capture   capture/ package with race      (~5 s)"
	@echo "    make test-fast      Full suite, no race detector    (~30 s)"
	@echo "    make test-vault     vault/ package with race        (~20 s)"
	@echo "    make test           Full suite with -race           (~3 min)"
	@echo ""
	@echo "  Other"
	@echo "    make coverage       HTML coverage report"
	@echo "    make clean          Remove build artefacts"
	@echo ""
	@echo "  Override Go path: make <target> GO=/path/to/go"
	@echo ""
