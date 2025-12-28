# Variables
BINARY_NAME := mcduck-wallet
LDFLAGS := '-linkmode external -extldflags "-static" -s -w'

# Phony targets
.PHONY: all build deploy update verify clean test run help

# Default target
all: update

# Build the binary
build:
	@echo "Generating html templates.."
	templ generate ./...
	@echo "Building binary..."
	@mkdir -p bin
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-cc \
	go build -ldflags $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

# First-time deployment (sets up user, firewall, systemd)
deploy:
	./scripts/deploy.sh

# Update existing deployment
update:
	./scripts/update.sh

# Verify deployment
verify:
	./scripts/verify.sh

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
	go run cmd/$(BINARY_NAME)/main.go

# Print help information
help:
	@echo "McDuck Wallet - Makefile targets"
	@echo ""
	@echo "Deployment:"
	@echo "  deploy     - First-time deployment (user, firewall, systemd)"
	@echo "  update     - Update existing deployment (default)"
	@echo "  verify     - Check deployment status and logs"
	@echo ""
	@echo "Development:"
	@echo "  build      - Build the binary"
	@echo "  test       - Run the test suite"
	@echo "  run        - Run the application locally"
	@echo "  clean      - Remove the bin directory"
	@echo ""
	@echo "Configuration:"
	@echo "  Copy .env.example to .env and configure before deploying"
