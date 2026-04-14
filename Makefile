# Vectora Unified Makefile
# Unifies build.ps1 logic and existing test suite

# Project variables
BINARY_NAME := vectora
VERSION := 0.1.0
BIN_DIR := bin
CMD_PATH := ./cmd/core
STORAGE_PATH := $(shell pwd)/internal/storage/db

# Go settings
GO := go
GO_FLAGS := -s -w
MOD_FLAG := $(shell if [ -d "vendor" ]; then echo "-mod=vendor"; fi)

# Colors
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
NC     := \033[0m

# Detect OS and Arch
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)

# CGO settings
export CGO_ENABLED := 1
export CGO_CFLAGS := -I$(STORAGE_PATH)

.PHONY: all build clean test help windows linux darwin k8s hash release

help:
	@echo "$(GREEN)Vectora Build & Test System$(NC)"
	@echo "============================"
	@echo "$(YELLOW)Build Commands:$(NC)"
	@echo "  make build          - Build for host platform ($(OS)/$(ARCH))"
	@echo "  make windows        - Build for Windows (amd64)"
	@echo "  make linux          - Build for Linux (amd64)"
	@echo "  make darwin         - Build for macOS (amd64)"
	@echo "  make k8s            - Build static binary for Kubernetes"
	@echo "  make clean          - Remove build artifacts"
	@echo ""
	@echo "$(YELLOW)Test Commands:$(NC)"
	@echo "  make test           - Run all tests"
	@echo "  make test-local      - Run local test suite"
	@echo "  make test-coverage   - Generate coverage report"
	@echo ""
	@echo "$(YELLOW)Release Commands:$(NC)"
	@echo "  make release        - Build all platforms and package"
	@echo "  make hash           - Generate build hashes"

all: clean build test

# --- Build Targets ---

build: pre-build build-host post-build

pre-build:
	@echo "$(YELLOW)[PRE-CHECK] Stopping active Vectora processes...$(NC)"
	@pkill -f $(BINARY_NAME) || true
	@mkdir -p $(BIN_DIR)

build-host:
	@echo "$(YELLOW)[PHASE 1] Compiling for host ($(OS)/$(ARCH))...$(NC)"
	$(GO) build $(MOD_FLAG) -ldflags="$(GO_FLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "$(GREEN)OK$(NC) - Produced $(BIN_DIR)/$(BINARY_NAME)"

windows: pre-build
	@echo "$(YELLOW)[PHASE 1] Compiling for Windows (amd64)...$(NC)"
	GOOS=windows GOARCH=amd64 $(GO) build $(MOD_FLAG) -ldflags="$(GO_FLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "$(GREEN)OK$(NC) - Produced $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe"

linux: pre-build
	@echo "$(YELLOW)[PHASE 1] Compiling for Linux (amd64)...$(NC)"
	GOOS=linux GOARCH=amd64 $(GO) build $(MOD_FLAG) -ldflags="$(GO_FLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "$(GREEN)OK$(NC) - Produced $(BIN_DIR)/$(BINARY_NAME)-linux-amd64"

darwin: pre-build
	@echo "$(YELLOW)[PHASE 1] Compiling for macOS (amd64)...$(NC)"
	GOOS=darwin GOARCH=amd64 $(GO) build $(MOD_FLAG) -ldflags="$(GO_FLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	@echo "$(GREEN)OK$(NC) - Produced $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64"

k8s: pre-build
	@echo "$(YELLOW)[K8S] Building static binary for Linux...$(NC)"
	GOOS=linux GOARCH=amd64 $(GO) build $(MOD_FLAG) -ldflags="$(GO_FLAGS) -extldflags '-static'" -o $(BIN_DIR)/$(BINARY_NAME)-static $(CMD_PATH)
	@echo "$(GREEN)OK$(NC) - Produced $(BIN_DIR)/$(BINARY_NAME)-static"

post-build:
	@cp $(BIN_DIR)/$(BINARY_NAME)* $(BIN_DIR)/$(BINARY_NAME) 2>/dev/null || true

# --- Tooling Targets ---

clean:
	@echo "$(YELLOW)[CLEAN] Removing bin/ and go.sum artifacts...$(NC)"
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Cleanup complete$(NC)"

vendor:
	@echo "$(YELLOW)[VENDOR] Updating dependencies...$(NC)"
	$(GO) mod tidy
	$(GO) mod vendor
	@echo "$(GREEN)Vendoring complete$(NC)"

hash:
	@echo "$(YELLOW)[HASH] Generating SHA256 hashes...$(NC)"
	@find $(BIN_DIR) -type f -exec sha256sum {} + > $(BIN_DIR)/checksums.txt
	@cat $(BIN_DIR)/checksums.txt

# --- Test Targets (ported from old Makefile) ---

test:
	@cd tests && $(GO) run .

test-local: test

test-coverage:
	@$(GO) test -cover -coverprofile=coverage.out -covermode=atomic ./...
	@$(GO) tool cover -html=coverage.out -o $(BIN_DIR)/coverage.html
	@echo "$(GREEN)Coverage report saved to $(BIN_DIR)/coverage.html$(NC)"

# --- Release Packaging ---

release: clean windows linux darwin hash
	@echo "$(YELLOW)[RELEASE] Packaging binaries...$(NC)"
	@cd $(BIN_DIR) && tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BIN_DIR) && zip -q $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "$(GREEN)Release packages ready in $(BIN_DIR)/$(NC)"
