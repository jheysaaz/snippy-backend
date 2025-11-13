# Project Structure

## Root Level Files

- `main.go` - Application entry point and server setup
- `go.mod`, `go.sum` - Go module dependencies
- `Makefile` - Build and development commands
- `docker-compose.yml`, `docker-compose.test.yml` - Container orchestration
- `snippy-api.service` - Systemd service configuration
- `README.md` - Project overview and quick start guide
- `LICENSE` - MIT license

## Application Code (`app/`)

### Authentication (`app/auth/`)

- `auth.go` - Authentication logic and JWT functions
- `auth_middleware.go` - JWT middleware for protected routes

### Database (`app/database/`)

- `database.go` - Database connection and schema initialization
- `database_test.go` - Database function tests

### HTTP Handlers (`app/handlers/`)

- `handlers.go` - Snippet CRUD handlers
- `handlers_test.go` - Snippet handler tests
- `user_handlers.go` - User management handlers
- `user_handlers_test.go` - User handler tests

### Models (`app/models/`)

- `models.go` - Data structures and request/response models
- `models_test.go` - Model and validation tests
- `refresh_token.go` - JWT refresh token functionality

### Middleware (`app/middleware/`)

- `rate_limiter.go` - Rate limiting middleware

## Documentation (`docs/`)

- `API.md` - Complete API reference with examples
- `ARCHITECTURE.md` - System design and structure
- `AUTHENTICATION.md` - JWT authentication system
- `DATABASE.md` - Database schema and optimizations
- `DEVELOPMENT.md` - Development environment setup
- `SECURITY.md` - Security features and best practices

## Database (`migrations/`)

- `README.md` - Migration instructions
- `001_*.sql` - Database migration files

## Deployment (`scripts/`)

- `deploy.sh` - Production deployment script

## GitHub Actions (`.github/workflows/`)

- `ci.yml` - Continuous integration (reusable workflow)
- `deploy.yml` - Production deployment workflow

## Key Improvements

1. **Organized Structure**: Related files are grouped in logical directories
2. **Separation of Concerns**: Auth, database, handlers, models, and middleware are clearly separated
3. **Colocated Tests**: Test files are placed alongside the code they test for better maintainability
4. **Logical Grouping**: Similar functionality is co-located
5. **Scalable Architecture**: Easy to add new features in appropriate directories

## Benefits

- **Easier Navigation**: Developers can quickly find relevant code
- **Better Maintainability**: Related code and tests are grouped together
- **Cleaner Root**: Main directory is not cluttered with source files
- **Standard Structure**: Follows common Go project patterns with tests alongside source code
- **Test Discovery**: Tests are easily discoverable next to the code they test
