.PHONY: build test lint coverage docker clean help

# Variables
BINARY_NAME=outbound-lb
BUILD_DIR=bin
MAIN_PATH=./cmd/outbound-lb
COVERAGE_FILE=coverage.out

# Build flags
LDFLAGS=-ldflags "-s -w"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-linux: ## Build for Linux amd64
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)

test: ## Run tests
	go test -race -v ./...

test-short: ## Run tests (short mode)
	go test -short ./...

coverage: ## Run tests with coverage
	go test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	go tool cover -func=$(COVERAGE_FILE)
	@echo ""
	@echo "Total coverage:"
	@go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}'

coverage-html: coverage ## Generate HTML coverage report
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter
	golangci-lint run ./...

lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix ./...

fmt: ## Format code
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	go vet ./...

mod-tidy: ## Tidy go modules
	go mod tidy

docker: ## Build Docker image
	docker build -t $(BINARY_NAME):latest -f deployments/docker/Dockerfile .

docker-compose-up: ## Start with docker-compose
	docker-compose -f deployments/docker-compose.yml up -d

docker-compose-down: ## Stop docker-compose
	docker-compose -f deployments/docker-compose.yml down

run: build ## Build and run locally
	./$(BUILD_DIR)/$(BINARY_NAME) --ips "127.0.0.1" --log-level debug --log-format text

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) coverage.html

install: build ## Install binary to GOPATH/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

all: lint test build ## Run lint, test, and build
