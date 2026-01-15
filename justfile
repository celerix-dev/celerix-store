set shell := ["sh", "-c"]

# Variables
binary := "celerix-stored"
image := "celerix/stored"
version := "1.0.0"
port := "7001"

# Default command: show available recipes
default:
    @just --list

# Build the static binary for Distroless/Linux
build:
    @echo "Building static binary..."
    mkdir -p bin
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/{{binary}} ./cmd/celerix-stored/main.go

# Run the store locally with the dev port
run: build
    @echo "Starting {{binary}} on port {{port}}..."
    CELERIX_STORE_PORT={{port}} ./bin/{{binary}}

# Clean build artifacts and temp files
clean:
    @echo "Cleaning up..."
    rm -rf bin/
    rm -f data/*.tmp

# Run a quick terminal health check
[confirm]
test:
    @echo "Running TCP Health Check..."
    @echo "PING" | nc localhost {{port}}
    @echo "LIST_PERSONAS" | nc localhost {{port}}
