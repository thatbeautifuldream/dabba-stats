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

# Main paths
BACKEND_PATH=./backend
FRONTEND_PATH=./frontend

.PHONY: all build clean run test deps tidy dev frontend-build backend-build

all: clean build

# Build both frontend and backend
build: frontend-build backend-build

# Build frontend
frontend-build:
	@echo "Building frontend..."
	@cd $(FRONTEND_PATH) && pnpm install && pnpm run build

# Build backend
backend-build:
	@echo "Building backend..."
	@mkdir -p $(BUILD_DIR)
	@cd $(BACKEND_PATH) && $(GOBUILD) -o ../$(BUILD_DIR)/$(BINARY_NAME) -v

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@cd $(FRONTEND_PATH) && rm -rf dist
	@$(GOCLEAN)

# Run the application (builds frontend first)
run: frontend-build
	@echo "Running..."
	@cd $(BACKEND_PATH) && $(GORUN) main.go

# Run tests
test:
	@echo "Running tests..."
	@cd $(BACKEND_PATH) && $(GOTEST) -v ./...
	@cd $(FRONTEND_PATH) && pnpm test

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@cd $(BACKEND_PATH) && $(GOGET) ./...
	@cd $(FRONTEND_PATH) && pnpm install

# Tidy up dependencies
tidy:
	@echo "Tidying up modules..."
	@cd $(BACKEND_PATH) && $(GOMOD) tidy
	@cd $(FRONTEND_PATH) && pnpm install

# Development with hot reload (requires air)
dev:
	@echo "Starting development server..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	@cd $(FRONTEND_PATH) && pnpm install && pnpm run dev & 
	@cd $(BACKEND_PATH) && air