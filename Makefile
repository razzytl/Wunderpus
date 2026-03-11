.PHONY: all build install uninstall clean help test lint vet deps update-deps check run docker-build docker-run generate

# Build variables
BINARY_NAME=wonderpus
BUILD_DIR=build
CMD_DIR=cmd/wonderpus
MAIN_GO=$(CMD_DIR)/main.go

# Version
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
GO_VERSION=$(shell go version | awk '{print $$3}')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME) -s -w"

# Go variables
GO?=CGO_ENABLED=0 go
GOFLAGS?=-v

# Golangci-lint
GOLANGCI_LINT?=golangci-lint

# Installation
INSTALL_PREFIX?=$(HOME)/.local
INSTALL_BIN_DIR=$(INSTALL_PREFIX)/bin
INSTALL_TMP_SUFFIX=.new

# Workspace and Skills
WONDERPUS_HOME?=$(HOME)/.wonderpus
WORKSPACE_DIR?=$(WONDERPUS_HOME)/workspace
WORKSPACE_SKILLS_DIR=$(WORKSPACE_DIR)/skills
BUILTIN_SKILLS_DIR=$(CURDIR)/skills

# OS detection
UNAME_S:=$(shell uname -s)
UNAME_M:=$(shell uname -m)

# Platform-specific settings
ifeq ($(UNAME_S),Linux)
	PLATFORM=linux
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),aarch64)
		ARCH=arm64
	else ifeq ($(UNAME_M),armv81)
		ARCH=arm64
	else
		ARCH=$(UNAME_M)
	endif
else ifeq ($(UNAME_S),Darwin)
	PLATFORM=darwin
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),arm64)
		ARCH=arm64
	else
		ARCH=$(UNAME_M)
	endif
else
	PLATFORM=$(UNAME_S)
	ARCH=$(UNAME_M)
endif

BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)-$(PLATFORM)-$(ARCH)

# Default target
all: build

## build: Build the wonderpus binary for current platform
build:
	@echo "Building $(BINARY_NAME) for $(PLATFORM)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./$(CMD_DIR)
	@echo "Build complete: $(BINARY_PATH)"
	@ln -sf $(BINARY_NAME)-$(PLATFORM)-$(ARCH) $(BUILD_DIR)/$(BINARY_NAME)

## build-all: Build wonderpus for all platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "All builds complete"

## build-linux-arm: Build for Linux ARMv7 (e.g. Raspberry Pi)
build-linux-arm:
	@echo "Building for linux/arm (GOARM=7)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm"

## build-linux-arm64: Build for Linux ARM64 (e.g. Raspberry Pi 64-bit)
build-linux-arm64:
	@echo "Building for linux/arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

## build-pi-zero: Build for Raspberry Pi Zero 2 W (32-bit and 64-bit)
build-pi-zero: build-linux-arm build-linux-arm64
	@echo "Pi Zero 2 W builds: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm (32-bit), $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 (64-bit)"

## build-whatsapp-native: Build with WhatsApp native support (larger binary)
build-whatsapp-native:
	@echo "Building $(BINARY_NAME) with WhatsApp native for $(PLATFORM)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -tags whatsapp_native $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build -tags whatsapp_native $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build -tags whatsapp_native $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build -tags whatsapp_native $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build -tags whatsapp_native $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "WhatsApp native builds complete"

## install: Install wonderpus to system
install: build
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(INSTALL_BIN_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)$(INSTALL_TMP_SUFFIX)
	@chmod +x $(INSTALL_BIN_DIR)/$(BINARY_NAME)$(INSTALL_TMP_SUFFIX)
	@mv -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)$(INSTALL_TMP_SUFFIX) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Installed binary to $(INSTALL_BIN_DIR)/$(BINARY_NAME)"

## uninstall: Remove wonderpus from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Removed binary from $(INSTALL_BIN_DIR)/$(BINARY_NAME)"

## uninstall-all: Remove wonderpus and all data
uninstall-all:
	@echo "Removing workspace and skills..."
	@rm -rf $(WONDERPUS_HOME)
	@echo "Removed workspace: $(WONDERPUS_HOME)"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## vet: Run go vet for static analysis
vet:
	$(GO) vet ./...

## test: Test Go code
test:
	$(GO) test ./...

## fmt: Format Go code
fmt:
	$(GO) fmt ./...

## lint: Run linters
lint:
	$(GOLANGCI_LINT) run ./...

## fix: Fix linting issues
fix:
	$(GOLANGCI_LINT) run --fix

## deps: Download dependencies
deps:
	$(GO) mod download
	$(GO) mod verify

## update-deps: Update dependencies
update-deps:
	$(GO) get -u ./...
	$(GO) mod tidy

## check: Run vet, fmt, and verify dependencies
check: deps vet fmt test

## generate: Run go generate for code generation
generate:
	@echo "Running go generate..."
	$(GO) generate ./...
	@echo "Generate complete"

## run: Build and run wonderpus
run: build
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## docker-build: Build Docker image (minimal Alpine-based)
docker-build:
	@echo "Building minimal Docker image..."
	docker build -t wonderpus:latest .
	@echo "Docker image built: wonderpus:latest"

## docker-build-multi: Build multi-platform Docker images
docker-build-multi:
	@echo "Building multi-platform Docker images..."
	docker buildx build --platform linux/amd64,linux/arm64 -t wonderpus:latest .
	@echo "Multi-platform Docker images built"

## docker-run: Run wonderpus in Docker
docker-run:
	docker run -d --name wonderpus \
		-p 8080:8080 \
		-p 9090:9090 \
		-v $(HOME)/.wonderpus:/data \
		wonderpus:latest

## docker-stop: Stop wonderpus Docker container
docker-stop:
	docker stop wonderpus || true
	docker rm wonderpus || true

## docker-clean: Clean Docker images
docker-clean:
	docker rmi wonderpus:latest || true

## help: Show this help message
help:
	@echo "wonderpus Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sort | awk -F': ' '{printf "  %-20s %s\n", substr($$1, 4), $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build for current platform"
	@echo "  make build-all          # Build for all platforms"
	@echo "  make build-pi-zero     # Build for Raspberry Pi"
	@echo "  make install            # Install to ~/.local/bin"
	@echo "  make docker-build       # Build Docker image"
	@echo ""
	@echo "Environment Variables:"
	@echo "  INSTALL_PREFIX          # Installation prefix (default: ~/.local)"
	@echo "  WORKSPACE_DIR           # Workspace directory (default: ~/.wonderpus/workspace)"
	@echo "  VERSION                # Version string (default: git describe)"
	@echo ""
	@echo "Current Configuration:"
	@echo "  Platform: $(PLATFORM)/$(ARCH)"
	@echo "  Binary: $(BINARY_PATH)"
	@echo "  Install Prefix: $(INSTALL_PREFIX)"
	@echo "  Workspace: $(WORKSPACE_DIR)"
