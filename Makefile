.PHONY: all build run clean tidy

# Default target
all: build

# Build the GoYT binary
build:
	@echo "Building GoYT..."
	go build -o goyt ./cmd/goyt/...

# Build and run the GoYT player
run: build
	@echo "Running GoYT..."
	./goyt

# Clean built binaries
clean:
	@echo "Cleaning binaries..."
	rm -f goyt

# Run go mod tidy
tidy:
	@echo "Tidying go modules..."
	go mod tidy
