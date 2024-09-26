# Variables
BINARY_NAME := mcduck-wallet
REMOTE_SERVER := mcduck
REMOTE_PATH := ~/mcduck-wallet
LDFLAGS := '-linkmode external -extldflags "-static" -s -w'

# Phony targets
.PHONY: all build transfer deploy clean test run help

# Default target
all: build transfer clean

# Build the binary
build:
	@echo "Generating html templates.."
	templ generate ./...
	@echo "Building binary..."
	@mkdir -p bin
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-cc \
	go build -ldflags $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

# Transfer the binary to the remote server
transfer: build
	@echo "Transferring the binary to the remote server..."
	command scp ./bin/$(BINARY_NAME) $(REMOTE_SERVER):$(REMOTE_PATH)/bin/$(BINARY_NAME).tmp

# Deploy the application
deploy: transfer
	@echo "Restarting the application on the remote server..."
	command ssh $(REMOTE_SERVER) 'pkill $(BINARY_NAME) || true; cd $(REMOTE_PATH) && mv bin/$(BINARY_NAME){.tmp,}; nohup ./bin/$(BINARY_NAME) > /dev/null 2>&1 &'
	@echo "Deployment complete."

# Clean up
clean:
	@echo "Cleaning up..."
	rm -rf bin

# Run tests
test:
	@echo "Running tests..."
	go test ./... -count=1

# Run the application locally
run:
	@echo "Running the app..."
	go run main.go

# Print help information
help:
	@echo "Available targets:"
	@echo "  all        - Build, transfer, and clean (default)"
	@echo "  build      - Build the binary"
	@echo "  transfer   - Transfer the binary to the remote server"
	@echo "  deploy     - Deploy the application to the remote server"
	@echo "  clean      - Remove the bin directory"
	@echo "  test       - Run the test suite"
	@echo "  run        - Run the application locally"
	@echo "  help       - Print this help information"

