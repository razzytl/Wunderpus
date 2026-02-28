.PHONY: build run test lint clean

BINARY=wonderpus
BUILD_DIR=bin

build:
	@echo "Building $(BINARY)..."
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) ./cmd/wonderpus

run: build
	@./$(BUILD_DIR)/$(BINARY)

test:
	@go test -v -race ./...

lint:
	@golangci-lint run ./...

clean:
	@rm -rf $(BUILD_DIR)
