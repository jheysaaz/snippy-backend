# Snippy Backend

[![CI](https://github.com/jheysaaz/snippy-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/jheysaaz/snippy-backend/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jheysaaz/snippy-backend)](https://goreportcard.com/report/github.com/jheysaaz/snippy-backend)
[![codecov](https://codecov.io/gh/jheysaaz/snippy-backend/graph/badge.svg?token=WFc5JInjwY)](https://codecov.io/gh/jheysaaz/snippy-backend)

REST API for code snippet management built with Go, Gin, and PostgreSQL.

## Features

- User authentication with username/email login (Argon2id password hashing)
- UUID-based user IDs with PostgreSQL generation
- Full CRUD operations for code snippets
- Full-text search and filtering by language/tags
- Optimized PostgreSQL with connection pooling and strategic indexes
- CORS enabled for browser extension integration
- Auto-updating timestamps via database triggers

## Project Structure

```
app/
├── auth/           # Authentication logic and middleware
├── database/       # Database connection and initialization
├── handlers/       # HTTP request handlers
├── models/         # Data models and structures
└── middleware/     # Application middleware

tests/              # All test files
docs/               # Complete documentation
migrations/         # Database migrations
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
POST /api/v1/auth/register   # Register new user
POST /api/v1/auth/login      # Login with username/email
POST /api/v1/auth/refresh    # Refresh JWT token
POST /api/v1/auth/logout     # Logout
```

### Snippets

```
GET    /api/v1/snippets      # List user snippets (with search/filter)
POST   /api/v1/snippets      # Create snippet
GET    /api/v1/snippets/:id  # Get snippet by ID
PUT    /api/v1/snippets/:id  # Update snippet
DELETE /api/v1/snippets/:id  # Delete snippet
```

### Users

```
GET /api/v1/users/profile    # Get current user profile
PUT /api/v1/users/profile    # Update user profile
```

### Health

```
GET /api/v1/health          # Health check
```

## Documentation

- [API Reference](docs/API.md) - Complete API documentation with examples
- [Development Setup](docs/DEVELOPMENT.md) - Development environment setup
- [Database Schema](docs/DATABASE.md) - Database structure and optimizations
- [Authentication](docs/AUTHENTICATION.md) - JWT authentication system
- [Architecture](docs/ARCHITECTURE.md) - System design and structure
- [Security](docs/SECURITY.md) - Security features and best practices
- [Project Structure](STRUCTURE.md) - Detailed file organization

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

## Environment Variables

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable

# JWT
JWT_SECRET=your-secret-key

# Server
PORT=8080
GIN_MODE=release

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://example.com
```

## Deployment

The project includes GitHub Actions workflows for automated deployment:

- **CI Pipeline**: Tests, security scans, and build on every push
- **Deploy Pipeline**: Automated deployment on version tags

See `.github/workflows/` for workflow configurations.

## License

MIT License. See [LICENSE](LICENSE) for details.
