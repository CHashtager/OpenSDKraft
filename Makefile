BINARY_NAME=sdkraft
VERSION=1.0.0
BUILD_DIR=build
MAIN_PATH=cmd/main.go

.PHONY: all build clean test lint

all: clean lint test build

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

install:
	@echo "Installing..."
	@go install $(MAIN_PATH)

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	@go test -v ./...

lint:
	@echo "Running linter..."
	@golangci-lint run

# Development helpers
dev-deps:
	@echo "Installing development dependencies..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create default template directory
init:
	@echo "Creating template directory..."
	@mkdir -p templates
	@cp -r internal/templates/* templates/

# Run example
example:
	@echo "Running example..."
	@$(BUILD_DIR)/$(BINARY_NAME) -c examples/config.yaml examples/petstore.yaml