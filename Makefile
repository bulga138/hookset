# Makefile for Hookset Tool (Cross-Platform)

BINARY_NAME=hookset
MAIN_PKG=./cmd/hookset/main.go

# --- Cross-Platform OS Detection & Command Setup ---
ifeq ($(OS),Windows_NT)
    # Windows Settings
    NULL_DEV = nul
    # On Windows, 'date' is interactive. We use git to get the date instead.
    # We use 'cmd /c' to ensure built-ins work if needed.
    GET_DATE = git log -1 --format=%%cd 2>$(NULL_DEV)
    RM_CMD = if exist $(BINARY_NAME).exe del /F /Q $(BINARY_NAME).exe && if exist hookset.log del /F /Q hookset.log
    EXE_EXT = .exe
else
    # Linux/Unix Settings
    NULL_DEV = /dev/null
    GET_DATE = date +%Y-%m-%dT%H:%M:%S%z
    RM_CMD = rm -f $(BINARY_NAME) hookset.log
    EXE_EXT =
endif

# --- Version Info ---
# We use git for the date to ensure cross-platform compatibility without external tools like 'date' on Windows
VERSION ?= $(shell git describe --tags --always --dirty 2>$(NULL_DEV) || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>$(NULL_DEV) || echo none)
BUILD_TIME ?= $(shell $(GET_DATE) || echo unknown)

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building hookset..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit:  $(COMMIT)"
	@go build -ldflags "-X github.com/bulga138/hookset/internal/version.Version=$(VERSION) -X github.com/bulga138/hookset/internal/version.Commit=$(COMMIT) -X 'github.com/bulga138/hookset/internal/version.BuildTime=$(BUILD_TIME)'" -o $(BINARY_NAME)$(EXE_EXT) $(MAIN_PKG)
	@echo "Build complete: $(BINARY_NAME)$(EXE_EXT)"

# Build with debug info (development)
.PHONY: build-dev
build-dev:
	@echo "Building development version..."
	@go build -o $(BINARY_NAME)$(EXE_EXT) $(MAIN_PKG)
	@echo "Development build complete"

# Release build
.PHONY: release
release:
	@echo "Building release version..."
	@go build -ldflags "-X github.com/bulga138/hookset/internal/version.Version=$(TAG) -X github.com/bulga138/hookset/internal/version.Commit=$(shell git rev-parse --short HEAD) -X 'github.com/bulga138/hookset/internal/version.BuildTime=$(shell $(GET_DATE))'" -o $(BINARY_NAME)-$(TAG)$(EXE_EXT) $(MAIN_PKG)
	@echo "Release build complete"

# Run the editor
.PHONY: run
run:
	@go run $(MAIN_PKG)

# Run with version info
.PHONY: run-version
run-version:
	@go run -ldflags "-X github.com/bulga138/hookset/internal/version.Version=$(VERSION) -X github.com/bulga138/hookset/internal/version.Commit=$(COMMIT) -X 'github.com/bulga138/hookset/internal/version.BuildTime=$(BUILD_TIME)'" $(MAIN_PKG) --version

# Run with file
.PHONY: run-file
run-file:
	@go run $(MAIN_PKG) $(FILE)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test ./...

# Generate per-archive SHA-256 files after goreleaser (run: make goreleaser-checksums)
# This produces dist/hookset_<version>_<os>_<arch>.tar.gz.sha256 files consumed
# by the install script for independent per-download verification.
.PHONY: goreleaser-checksums
goreleaser-checksums:
	@echo "Generating per-archive SHA-256 files in dist/..."
	@for f in dist/*.tar.gz dist/*.zip; do \
	  [ -f "$$f" ] || continue; \
	  if command -v sha256sum >/dev/null 2>&1; then \
	    sha256sum "$$f" | awk '{print $$1}' > "$$f.sha256"; \
	  else \
	    shasum -a 256 "$$f" | awk '{print $$1}' > "$$f.sha256"; \
	  fi; \
	  echo "  wrote $$f.sha256"; \
	done
	@echo "Done."

# Clean up
.PHONY: clean
clean:
	@echo "Cleaning up..."
	@$(RM_CMD)
	@echo "Cleanup complete."