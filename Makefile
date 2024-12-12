# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=system-monitor

# Build directory
BUILD_DIR=build

# Main entry point
MAIN_PATH=./backend

.PHONY: all build clean run test deps tidy

all: clean build

# Build the application
build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	@cd $(MAIN_PATH) && $(GOBUILD) -o ../$(BUILD_DIR)/$(BINARY_NAME) -v

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@$(GOCLEAN)

# Run the application
run:
	@echo "Running..."
	@cd $(MAIN_PATH) && $(GORUN) main.go

# Run tests
test:
	@echo "Running tests..."
	@cd $(MAIN_PATH) && $(GOTEST) -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@cd $(MAIN_PATH) && $(GOGET) ./...

# Tidy up dependencies
tidy:
	@echo "Tidying up modules..."
	@cd $(MAIN_PATH) && $(GOMOD) tidy

# Development with hot reload (requires air)
dev:
	@echo "Starting development server..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	@cd $(MAIN_PATH) && air 