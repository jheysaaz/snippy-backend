.PHONY: help test test-coverage security format lint build clean all

# Default target
help:
	@echo "Available targets:"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make security       - Run security scans (gosec + govulncheck)"
	@echo "  make format         - Format code with gofmt"
	@echo "  make format-check   - Check if code is formatted"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make build          - Build the application"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make all            - Run all checks (format, lint, security, test, build)"

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run security scans
security: security-gosec security-govulncheck

security-gosec:
	@echo "Running gosec security scanner..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec -fmt=json -out=gosec-report.json ./...
	gosec ./...

security-govulncheck:
	@echo "Running govulncheck..."
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# Format code
format:
	@echo "Formatting code..."
	gofmt -s -w .
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w .

# Check if code is formatted
format-check:
	@echo "Checking code format..."
	@test -z "$$(gofmt -s -l . | tee /dev/stderr)" || (echo "Please run 'make format' to format your code" && exit 1)

# Run linter
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.61.0)
	golangci-lint run --timeout=5m

# Build application
build:
	@echo "Building application..."
	go build -v -o snippy-backend .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f snippy-backend
	rm -f coverage.out coverage.html
	rm -f gosec-report.json
	go clean

# Run all checks
all: format-check lint security test build
	@echo "All checks passed!"
