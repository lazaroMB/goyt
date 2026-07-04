.PHONY: all build run clean tidy compile-all release-cli

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Default target
all: build

# Build the GoYT binary for local system
build:
	@echo "Building GoYT..."
	go build -ldflags "$(LDFLAGS)" -o goyt ./cmd/goyt/...

# Build and run the GoYT player
run: build
	@echo "Running GoYT..."
	./goyt

# Clean built binaries and packaging files
clean:
	@echo "Cleaning binaries..."
	rm -f goyt
	rm -rf dist/

# Run go mod tidy
tidy:
	@echo "Tidying go modules..."
	go mod tidy

# Compile for all targets (Linux, macOS Intel/M1+, Windows)
compile-all: clean
	@echo "Compiling for multiple platforms..."
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/goyt-linux-amd64 ./cmd/goyt/...
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/goyt-darwin-amd64 ./cmd/goyt/...
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/goyt-darwin-arm64 ./cmd/goyt/...
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/goyt-windows-amd64.exe ./cmd/goyt/...

# Release to GitHub using gh CLI
release-cli: compile-all
	@echo "Creating GitHub release for $(VERSION)..."
	gh release create $(VERSION) dist/* --generate-notes --title "Release $(VERSION)"

