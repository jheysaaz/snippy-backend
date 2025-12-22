.PHONY: help test test-coverage test-db-up test-db-down test-db-logs test-with-db test-clean security format format-check lint build build-linux clean all up down logs ssl-init ssl-renew ssl-status

GOCMD := go
GOTEST := $(GOCMD) test -v -race
DC := docker compose

.DEFAULT_GOAL := help
help: ## Show available targets
	@awk -F':.*##' '/^[a-zA-Z0-9_.-]+:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# =============================================================================
# Testing
# =============================================================================
test: ## Run tests
	@if [ -f .env.test ]; then set -a; . ./.env.test; set +a; fi; $(GOTEST) ./...

test-coverage: ## Run tests with coverage
	@if [ -f .env.test ]; then set -a; . ./.env.test; set +a; fi; \
	$(GOTEST) -coverprofile=coverage.out -covermode=atomic ./...; \
	go tool cover -html=coverage.out -o coverage.html

test-db-up: ## Start test database
	$(DC) -f docker compose.test.yml up -d

test-db-down: ## Stop test database
	$(DC) -f docker compose.test.yml down

test-db-logs: ## Show test database logs
	$(DC) -f docker compose.test.yml logs -f

test-with-db: test-db-up ## Run tests with database
	@DATABASE_URL="postgres://test_user:test_password@localhost:5433/snippy_test?sslmode=disable" \
	JWT_SECRET="test-jwt-secret-key-for-testing-only" $(GOTEST) ./...

test-clean: test-db-down ## Clean test database volumes
	docker volume prune -f

# =============================================================================
# Code Quality
# =============================================================================
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

# =============================================================================
# Build
# =============================================================================
build: ## Build for current OS
	$(GOCMD) build -v -o snippy-api .

build-linux: ## Build for Linux (Docker)
	GOOS=linux GOARCH=amd64 $(GOCMD) build -v -o snippy-api .

clean: ## Clean build artifacts
	rm -f snippy-api coverage.out coverage.html gosec-report.json
	go clean

all: format-check lint security test build ## Run all checks

# =============================================================================
# Docker
# =============================================================================
up: ssl-init build-linux ## Start all containers
	$(DC) up -d

down: ## Stop all containers
	$(DC) down

logs: ## Show container logs
	$(DC) logs -f

restart: ## Restart all containers
	$(DC) restart

ps: ## Show container status
	$(DC) ps

clean-docker: ## Remove containers and volumes (WARNING: deletes data!)
	$(DC) down -v

# =============================================================================
# SSL / Let's Encrypt
# =============================================================================
ssl-init: ## Initialize SSL certs (self-signed local, Let's Encrypt production)
	@docker volume create snippy-backend_certbot_certs >/dev/null 2>&1 || true
	@docker run --rm \
		-v snippy-backend_certbot_certs:/etc/letsencrypt \
		-v $(PWD)/scripts/init-ssl.sh:/init-ssl.sh:ro \
		alpine sh -c "apk add --no-cache openssl bash >/dev/null && bash /init-ssl.sh '$${DOMAIN:-}' '$${CERTBOT_EMAIL:-}' '$${CERTBOT_STAGING:-false}'"

ssl-renew: ## Renew Let's Encrypt certificates
	$(DC) --profile ssl run --rm certbot renew
	$(DC) exec nginx nginx -s reload

ssl-status: ## Show certificate info
	@docker run --rm -v snippy-backend_certbot_certs:/etc/letsencrypt alpine \
		sh -c "apk add --no-cache openssl >/dev/null 2>&1; cat /etc/letsencrypt/live/cert/fullchain.pem 2>/dev/null | openssl x509 -noout -subject -dates || echo 'No certificate found'"

ssl-clean: ## Remove SSL certificates volume
	docker volume rm snippy-backend_certbot_certs 2>/dev/null || true

