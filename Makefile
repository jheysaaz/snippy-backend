.PHONY: help test test-coverage test-db-up test-db-down test-db-logs test-with-db test-clean security format lint build clean all

# Default target
help:
	@echo "Available targets:"
	@echo "  make test           - Run tests (skips DB tests if no connection)"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-db-up     - Start test database container"
	@echo "  make test-db-down   - Stop test database container"
	@echo "  make test-db-logs   - Show test database logs"
	@echo "  make test-with-db   - Start DB and run all tests"
	@echo "  make test-clean     - Stop DB and clean up volumes"
	@echo "  make security       - Run security scans (gosec + govulncheck)"
	@echo "  make format         - Format code with gofmt"
	@echo "  make format-check   - Check if code is formatted"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make build          - Build the application"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make all            - Run all checks (format, lint, security, test, build)"

# Run tests (will skip DB tests if no connection)
test:
	@echo "Running tests..."
	@if [ -f .env.test ]; then \
		set -a; . ./.env.test; set +a; go test -v -race ./...; \
	else \
		go test -v -race ./...; \
	fi

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@if [ -f .env.test ]; then \
		set -a; . ./.env.test; set +a; go test -v -race -coverprofile=coverage.out -covermode=atomic ./...; \
	else \
		go test -v -race -coverprofile=coverage.out -covermode=atomic ./...; \
	fi
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Start test database
test-db-up:
	@echo "Starting test database..."
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for database to be ready..."
	@sleep 3
	@echo "Test database is ready on port 5433"

# Stop test database
test-db-down:
	@echo "Stopping test database..."
	docker-compose -f docker-compose.test.yml down

# Show test database logs
test-db-logs:
	@echo "Showing test database logs..."
	docker-compose -f docker-compose.test.yml logs -f

# Run tests with database
test-with-db: test-db-up
	@echo "Waiting for database to be fully ready..."
	@sleep 2
	@echo "Running tests with database..."
	@DATABASE_URL="postgres://test_user:test_password@localhost:5433/snippy_test?sslmode=disable" \
	 JWT_SECRET="test-jwt-secret-key-for-testing-only" \
	 go test -v -race ./...
	@echo ""
	@echo "Tests completed. To stop the database, run: make test-db-down"

# Clean test database and volumes
test-clean: test-db-down
	@echo "Cleaning up test database..."
	@docker volume prune -f
	@echo "Cleanup completed"

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
	go build -v -o snippy-api .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f snippy-api
	rm -f coverage.out coverage.html
	rm -f gosec-report.json
	go clean

# Run all checks
all: format-check lint security test build
	@echo "All checks passed!"
