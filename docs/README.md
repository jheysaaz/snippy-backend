# Documentation Index

Welcome to the Snippy Backend documentation! This guide will help you find the information you need.

## 📚 Documentation Structure

### For Users

- **[README.md](../README.md)** - Quick start guide and overview
- **[API.md](./API.md)** - Complete API reference with examples

### For Developers

- **[DEVELOPMENT.md](./DEVELOPMENT.md)** - Development setup and workflow
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - System architecture and design
- **[DATABASE.md](./DATABASE.md)** - Database schema and optimizations
- **[AUTHENTICATION.md](./AUTHENTICATION.md)** - JWT authentication & authorization
- **[SECURITY.md](./SECURITY.md)** - Security audit & best practices
- **[GITHUB_ACTIONS_DEPLOYMENT.md](./GITHUB_ACTIONS_DEPLOYMENT.md)** - Deployment with Docker Compose & systemd

---

## Quick Links

### Getting Started

- [Installation Guide](../README.md#setup)
- [Environment Setup](./DEVELOPMENT.md#getting-started)
- [First API Request](./API.md#authentication)

### API Reference

- [Authentication Endpoints](./API.md#authentication)
- [User Management](./API.md#users)
- [Snippet Operations](./API.md#snippets)
- [Error Responses](./API.md#error-responses)

### Development

- [Running Tests](./DEVELOPMENT.md#testing)
- [Debugging Tips](./DEVELOPMENT.md#debugging)
- [Code Guidelines](./DEVELOPMENT.md#code-guidelines)
- [Common Tasks](./DEVELOPMENT.md#common-tasks)

### Architecture

- [System Overview](./ARCHITECTURE.md#architecture-overview)
- [Component Details](./ARCHITECTURE.md#component-details)
- [Request Flow](./ARCHITECTURE.md#request-flow)
- [Security](./ARCHITECTURE.md#security-architecture)

### Database

- [Schema Definition](./DATABASE.md#schema)
- [Performance Optimization](./DATABASE.md#performance-optimizations)
- [Backup & Restore](./DATABASE.md#backup-and-restore)
- [Monitoring](./DATABASE.md#monitoring-queries)

---

## What's Snippy Backend?

Snippy Backend is a high-performance REST API built with Go and PostgreSQL for managing code snippets and shortcuts. It's designed to power Chrome extensions and web applications with:

- ✨ **Fast & Efficient** - Optimized PostgreSQL queries with strategic indexing
- 🔐 **Secure** - Argon2id password hashing and parameterized queries
- 🎯 **Feature-Rich** - Full-text search, tag filtering, and categorization
- 🚀 **Production-Ready** - Connection pooling, error handling, and comprehensive tests

---

## Key Features

### Snippet Management

- Create, read, update, and delete snippets
- Organize by categories (git, docker, kubernetes, etc.)
- Quick access via shortcuts
- Tag-based organization
- Full-text search in titles and descriptions

### User Management

- User registration and authentication
- UUID-based user IDs
- Secure password hashing (Argon2id)
- User-specific snippet collections

### Performance

- PostgreSQL connection pooling (25 max, 5 idle)
- B-tree indexes on frequently queried fields
- GIN indexes for array operations and full-text search
- Optimized query execution with prepared statements

---

## API Overview

### Base URL

```
http://localhost:8080/api/v1
```

### Core Endpoints

**Authentication:**

- `POST /auth/login` - User login

**Users:**

- `POST /users` - Register new user
- `GET /users` - List all users
- `GET /users/:id` - Get user details
- `PUT /users/:id` - Update user
- `DELETE /users/:id` - Delete user

**Snippets:**

- `POST /snippets` - Create snippet
- `GET /snippets` - List snippets (with filters)
- `GET /snippets/:id` - Get snippet details
- `PUT /snippets/:id` - Update snippet
- `DELETE /snippets/:id` - Delete snippet

**Health:**

- `GET /health` - API health check

For detailed endpoint documentation, see [API.md](./API.md).

---

## Technology Stack

- **Language**: Go 1.24+
- **Web Framework**: Gin v1.11.0
- **Database**: PostgreSQL 16-alpine
- **Database Driver**: lib/pq v1.10.9
- **Password Hashing**: Argon2id (x/crypto)
- **Environment**: Docker Compose for development

---

## Documentation Pages

### [API.md](./API.md)

Complete API reference with:

- Request/response formats
- Authentication examples
- All endpoints documented
- Error handling
- cURL examples for every endpoint

### [DEVELOPMENT.md](./DEVELOPMENT.md)

Developer guide covering:

- Setup instructions
- Development workflow
- Testing strategies
- Debugging techniques
- Code guidelines
- Common tasks
- Troubleshooting

### [ARCHITECTURE.md](./ARCHITECTURE.md)

System architecture documentation:

- Layered architecture overview
- Component responsibilities
- Request/response flow
- Data flow diagrams
- Error handling patterns
- Performance considerations
- Security architecture

### [DATABASE.md](./DATABASE.md)

Database documentation including:

- Complete schema definitions
- Index strategies
- Performance optimizations
- Backup and restore procedures
- Monitoring queries
- Migration strategies

---

## Quick Start

1. **Clone and setup:**

   ```bash
   git clone https://github.com/jheysaaz/snippy-backend.git
   cd snippy-backend
   cp .env.example .env
   ```

2. **Start database:**

   ```bash
   docker-compose up -d
   ```

3. **Run server:**

   ```bash
   go mod download
   go run .
   ```

4. **Test it:**
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

For detailed setup instructions, see [README.md](../README.md) or [DEVELOPMENT.md](./DEVELOPMENT.md).

---

## Examples

### Create a User

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "securePass123"
  }'
```

### Create a Snippet

```bash
curl -X POST http://localhost:8080/api/v1/snippets \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Git Status",
    "description": "Quick git status command",
    "category": "git",
    "shortcut": "gst",
    "content": "git status -sb",
    "tags": ["git", "shortcuts"]
  }'
```

### Search Snippets

```bash
# By category
curl "http://localhost:8080/api/v1/snippets?category=git"

# By tag
curl "http://localhost:8080/api/v1/snippets?tag=shortcuts"

# Full-text search
curl "http://localhost:8080/api/v1/snippets?search=status"

# Combined
curl "http://localhost:8080/api/v1/snippets?category=git&tag=shortcuts&limit=10"
```

---

## Contributing

We welcome contributions! Here's how:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following our [code guidelines](./DEVELOPMENT.md#code-guidelines)
4. Write tests for your changes
5. Ensure all tests pass (`go test -v ./...`)
6. Update documentation if needed
7. Submit a pull request

See [DEVELOPMENT.md](./DEVELOPMENT.md) for detailed development guidelines.

---

## Support

### Need Help?

- **Documentation**: Start with this index to find relevant docs
- **Issues**: Check [existing issues](https://github.com/jheysaaz/snippy-backend/issues)
- **Questions**: Open a discussion on GitHub

### Found a Bug?

1. Check if it's already reported
2. Create a new issue with:
   - Clear description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (Go version, OS, etc.)

### Want a Feature?

1. Check if it's already requested
2. Open a feature request with:
   - Use case description
   - Proposed solution
   - Alternative approaches considered

---

## Changelog

For version history and release notes, see:

- [GitHub Releases](https://github.com/jheysaaz/snippy-backend/releases)
- [CHANGELOG.md](../CHANGELOG.md) (if available)

---

## Related Projects

- **Snippy Chrome Extension** - Frontend Chrome extension (link coming soon)
- **Snippy Web UI** - Web interface for managing snippets (link coming soon)

---

## Acknowledgments

Built with:

- [Go](https://golang.org/) - Programming language
- [Gin](https://gin-gonic.com/) - Web framework
- [PostgreSQL](https://www.postgresql.org/) - Database
- [Docker](https://www.docker.com/) - Containerization

---

**Happy coding! 🚀**

For questions or feedback, please open an issue on GitHub.
