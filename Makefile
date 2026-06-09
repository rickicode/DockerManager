BINARY_NAME=docker-manager
GO_BUILD=go build
GO_TEST=go test
GO_FLAGS=-ldflags="-s -w"
INSTALL_DIR=/usr/local/bin
REPO_URL=https://github.com/rickicode/DockerManager

.PHONY: all build run install test clean

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	$(GO_BUILD) $(GO_FLAGS) -o $(BINARY_NAME) .
	@echo "Done: ./$(BINARY_NAME) ($$(ls -lh $(BINARY_NAME) | awk '{print $$5}'))"

run: build
	@echo "Starting $(BINARY_NAME)..."
	./$(BINARY_NAME)

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@if [ ! -w "$(INSTALL_DIR)" ]; then \
		echo "Requires sudo to install to $(INSTALL_DIR)"; \
		sudo cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME); \
		sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME); \
	else \
		cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME); \
		chmod +x $(INSTALL_DIR)/$(BINARY_NAME); \
	fi
	@echo "Installed! Run: $(BINARY_NAME) --port 8080"

test:
	@echo "Running tests..."
	$(GO_TEST) -v ./...

test-short:
	@echo "Running tests (short)..."
	$(GO_TEST) ./...

clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	@echo "Done"

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Linting..."
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

version:
	@echo "$(BINARY_NAME) v1.0.0"

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build        Build the binary"
	@echo "  run          Build and run"
	@echo "  install      Build and install to $(INSTALL_DIR)"
	@echo "  test         Run all tests with verbose output"
	@echo "  test-short   Run all tests (no verbose)"
	@echo "  clean        Remove build artifacts"
	@echo "  fmt          Format Go source code"
	@echo "  lint         Run linter (if installed)"
	@echo "  version      Show version"
	@echo "  help         Show this help"
