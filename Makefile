BINARY_NAME=mkduck-wallet

.PHONY: build run
test:
	go test ./... -count=1

build:
	@echo "Building the app..."
	mkdir -p bin
	go build -o bin/$(BINARY_NAME) main.go

run:
	 @echo "Running the app..."
	sudo -E ./bin/$(BINARY_NAME)
