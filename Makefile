.PHONY: help test test-coverage test-db-up test-db-down test-db-logs test-with-db test-clean security format format-check lint build clean all

GOCMD := go
GOTEST := $(GOCMD) test -v -race
DC := docker-compose
COMPOSE := -f docker-compose.test.yml

.DEFAULT_GOAL := help
help: ## Show available targets
	@awk -F':.*##' '/^[a-zA-Z0-9_.-]+:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

test: ## Run tests
	@if [ -f .env.test ]; then set -a; . ./.env.test; set +a; fi; $(GOTEST) ./...

test-coverage: ## Run tests with coverage
	@if [ -f .env.test ]; then set -a; . ./.env.test; set +a; fi; $(GOTEST) -coverprofile=coverage.out -covermode=atomic ./...; \
		go tool cover -html=coverage.out -o coverage.html

test-db-up: ## Start test database
	$(DC) $(COMPOSE) up -d

test-db-down: ## Stop test database
	$(DC) $(COMPOSE) down

test-db-logs: ## Show test database logs
	$(DC) $(COMPOSE) logs -f

test-with-db: test-db-up ## Run tests with database
	@DATABASE_URL="postgres://test_user:test_password@localhost:5433/snippy_test?sslmode=disable" \
	JWT_SECRET="test-jwt-secret-key-for-testing-only" $(GOTEST) ./...

test-clean: test-db-down ## Clean test database volumes
	docker volume prune -f

security: ## Run security scans (gosec + govulncheck)
	@command -v gosec >/dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	@command -v govulncheck >/dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	gosec ./... && govulncheck ./...

format: ## Format code
	@gofmt -s -w .
	@command -v goimports >/dev/null || go install golang.org/x/tools/cmd/goimports@latest
	goimports -w .

format-check: ## Check code format
	@test -z "$$(gofmt -s -l . | tee /dev/stderr)" || (echo "Please run 'make format'"; exit 1)

lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.61.0
	golangci-lint run --timeout=5m

build: ## Build the application
	$(GOCMD) build -v -o snippy-api .

clean: ## Clean build artifacts
	rm -f snippy-api coverage.out coverage.html gosec-report.json
	go clean

all: format-check lint security test build ## Run all checks
