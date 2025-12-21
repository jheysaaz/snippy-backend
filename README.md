# Snippy Backend

[![CI](https://github.com/jheysaaz/snippy-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/jheysaaz/snippy-backend/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jheysaaz/snippy-backend)](https://goreportcard.com/report/github.com/jheysaaz/snippy-backend)
[![codecov](https://codecov.io/gh/jheysaaz/snippy-backend/graph/badge.svg?token=WFc5JInjwY)](https://codecov.io/gh/jheysaaz/snippy-backend)

REST API for code snippet management built with Go, Gin, and PostgreSQL.

## Features

- **Authentication**: JWT with refresh tokens (HTTP-only cookies), Argon2id hashing
- **Snippets**: CRUD operations with version history and soft delete
- **Search**: Full-text search with language/tag filtering
- **Sessions**: User session tracking with activity monitoring
- **Sync**: Bandwidth-efficient sync endpoint for incremental updates
- **Retention**: Automatic cleanup of old data (30/60/90-day policies)
- **Database**: PostgreSQL with connection pooling, triggers, and CASCADE DELETE

## Project Structure

```
app/
├── auth/           # JWT authentication and middleware
├── database/       # PostgreSQL connection and schema
├── handlers/       # HTTP handlers and routes
├── models/         # Data models and database operations
└── middleware/     # Rate limiting and CORS

migrations/         # Database migrations (auto-applied)
tests/              # Test files
docs/               # Documentation
scripts/            # Deployment scripts
```

## Quick Start

```bash
# Start PostgreSQL
docker-compose up -d

# Run server (auto-creates schema)
go run .
```

Server runs on `http://localhost:8080`

## API Endpoints

### Authentication

```
POST   /api/v1/auth/register       # Register new user
POST   /api/v1/auth/login          # Login (sets refresh token cookie)
POST   /api/v1/auth/refresh        # Refresh access token
POST   /api/v1/auth/logout         # Logout (clears cookie)
GET    /api/v1/auth/availability   # Check username/email availability
GET    /api/v1/auth/sessions       # List active sessions
DELETE /api/v1/auth/sessions/:id   # Logout specific session
```

### Snippets

```
GET    /api/v1/snippets                      # List snippets (search, filter, pagination)
POST   /api/v1/snippets                      # Create snippet
GET    /api/v1/snippets/sync                 # Sync changes since timestamp
GET    /api/v1/snippets/:id                  # Get snippet
PUT    /api/v1/snippets/:id                  # Update snippet
DELETE /api/v1/snippets/:id                  # Soft delete snippet
GET    /api/v1/snippets/:id/history          # Get version history
POST   /api/v1/snippets/:id/history/:version # Restore version
```

### Users

```
GET    /api/v1/users/profile    # Get profile
PUT    /api/v1/users/profile    # Update profile
DELETE /api/v1/users/profile    # Soft delete account
```

### Health

```
GET /api/v1/health    # Health check
```

## Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./tests/...

# Run with hot reload (requires air)
air

# Format code
gofmt -s -w .

# Lint code
golangci-lint run
```

## Deployment

The project includes GitHub Actions workflows for automated deployment:

- **CI Pipeline**: Tests, security scans, and build on every push
- **Deploy Pipeline**: Automated deployment on version tags

See `.github/workflows/` for workflow configurations.

## License

MIT License. See [LICENSE](LICENSE) for details.
