# Development Guide

Complete guide for developers working on Snippy Backend.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Debugging](#debugging)
- [Code Guidelines](#code-guidelines)
- [Common Tasks](#common-tasks)
- [Troubleshooting](#troubleshooting)

---

## Getting Started

### Prerequisites

- **Go**: Version 1.24 or higher
- **Docker & Docker Compose**: For PostgreSQL
- **Git**: For version control
- **cURL or Postman**: For API testing

### Initial Setup

1. **Clone the repository:**

   ```bash
   git clone https://github.com/jheysaaz/snippy-backend.git
   cd snippy-backend
   ```

2. **Install dependencies:**

   ```bash
   go mod download
   ```

3. **Set up environment variables:**

   ```bash
   cp .env.example .env
   ```

   Edit `.env` if needed (defaults work with docker-compose):

   ```env
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=postgres
   DB_NAME=snippy
   PORT=8080
   ```

4. **Start PostgreSQL:**

   ```bash
   docker-compose up -d
   ```

5. **Run the application:**

   ```bash
   go run .
   ```

6. **Verify it's working:**
   ```bash
   curl http://localhost:8080/api/v1/health
   # Should return: {"status":"ok"}
   ```

---

## Development Workflow

### Running the Server

**Standard Run:**

```bash
go run .
```

**With Auto-Reload (using air):**

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with auto-reload
air
```

**Build and Run:**

```bash
go build -o snippy-backend
./snippy-backend
```

### Project Structure

```
snippy-backend/
├── main.go                 # Entry point, router setup
├── database.go            # Database connection & schema
├── models.go              # Data structures
├── handlers.go            # Snippet endpoints
├── user_handlers.go       # User & auth endpoints
├── middleware.go          # CORS & other middleware
├── *_test.go              # Test files
├── go.mod / go.sum        # Dependencies
├── .env.example           # Environment template
├── docker-compose.yml     # PostgreSQL setup
└── docs/                  # Documentation
```

### Making Changes

1. **Create a feature branch:**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the code guidelines

3. **Run tests:**

   ```bash
   go test -v ./...
   ```

4. **Format code:**

   ```bash
   go fmt ./...
   ```

5. **Check for issues:**

   ```bash
   go vet ./...
   ```

6. **Commit and push:**
   ```bash
   git add .
   git commit -m "feat: your feature description"
   git push origin feature/your-feature-name
   ```

---

## Testing

### Running Tests

**All tests:**

```bash
go test -v ./...
```

**Specific file:**

```bash
go test -v -run TestSnippetScanFunction
```

**With coverage:**

```bash
go test -v -cover ./...
```

**Generate coverage report:**

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Categories

#### 1. Unit Tests (`models_test.go`)

Tests data models and validation without database:

```go
func TestSnippetScanFunction(t *testing.T) {
    // Test scanning logic
}
```

#### 2. Integration Tests

Tests requiring PostgreSQL (auto-skipped if DB unavailable):

**Database Tests (`database_test.go`):**

```bash
# Start PostgreSQL first
docker-compose up -d

# Run tests
go test -v -run TestDatabase
```

**Handler Tests (`handlers_test.go`):**

```bash
go test -v -run TestCreateSnippet
```

### Writing Tests

**Unit Test Pattern:**

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    interface{}
        expected interface{}
    }{
        {"case 1", input1, expected1},
        {"case 2", input2, expected2},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := functionUnderTest(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

**Integration Test Pattern:**

```go
func TestEndpoint(t *testing.T) {
    // Skip if no DB
    if db == nil {
        t.Skip("Skipping: Cannot connect to PostgreSQL")
    }

    // Setup
    router := setupRouter()

    // Make request
    req := httptest.NewRequest("GET", "/api/v1/snippets", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // Assert
    if w.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", w.Code)
    }
}
```

---

## Debugging

### Enable Debug Logging

**Gin Debug Mode:**

```go
// In main.go
gin.SetMode(gin.DebugMode) // Shows detailed request logs
```

**Database Query Logging:**
Add after query execution:

```go
rows, err := db.Query(query, args...)
fmt.Printf("Query: %s\nArgs: %v\n", query, args)
```

### Using Delve Debugger

**Install:**

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**Debug:**

```bash
dlv debug
```

**Set breakpoints in code:**

```go
import "runtime/debug"

func someFunction() {
    debug.PrintStack() // Print stack trace
    // Your code
}
```

### Database Debugging

**Connect to PostgreSQL:**

```bash
docker-compose exec postgres psql -U postgres -d snippy
```

**Useful queries:**

```sql
-- Check current connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'snippy';

-- View all snippets
SELECT * FROM snippets ORDER BY created_at DESC LIMIT 10;

-- Check indexes
\di

-- View table structure
\d snippets

-- Enable query timing
\timing on
```

### API Testing

**Using cURL:**

```bash
# Create snippet
curl -X POST http://localhost:8080/api/v1/snippets \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","category":"test","shortcut":"tst","content":"test"}'

# Get snippets
curl http://localhost:8080/api/v1/snippets

# With pretty print
curl http://localhost:8080/api/v1/snippets | jq
```

**Using httpie:**

```bash
# Install
brew install httpie

# Usage
http POST localhost:8080/api/v1/snippets title="Test" category="test" shortcut="tst" content="test"
```

---

## Code Guidelines

### Go Style

Follow [Effective Go](https://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

### Naming Conventions

**Files:**

```
lowercase_with_underscores.go
user_handlers.go
database_test.go
```

**Functions:**

```go
// Public (exported)
func CreateSnippet() {}

// Private (unexported)
func validateInput() {}
```

**Variables:**

```go
// Descriptive names
userID := "123"
snippetCount := 10

// Avoid single letters except in loops
for i := 0; i < len(items); i++ {}
```

### Error Handling

**Always check errors:**

```go
// ❌ Bad
result, _ := someFunction()

// ✅ Good
result, err := someFunction()
if err != nil {
    return err
}
```

**Return early:**

```go
// ✅ Good
func process(input string) error {
    if input == "" {
        return errors.New("input required")
    }

    // Main logic here
    return nil
}
```

### Database Queries

**Use parameterized queries:**

```go
// ✅ Good
query := "SELECT * FROM snippets WHERE category = $1"
rows, err := db.Query(query, category)

// ❌ Bad - SQL injection risk
query := fmt.Sprintf("SELECT * FROM snippets WHERE category = '%s'", category)
```

**Multi-line for readability:**

```go
query := `
    SELECT id, title, description, category, shortcut, content, tags, user_id, created_at, updated_at
    FROM snippets
    WHERE category = $1
    ORDER BY created_at DESC
`
```

### JSON Handling

**Use struct tags:**

```go
type Snippet struct {
    ID       int64  `json:"id"`
    Title    string `json:"title"`
    Password string `json:"password,omitempty"` // Omit if empty
}
```

### Comments

**Public functions:**

```go
// CreateSnippet creates a new snippet in the database.
// It validates the input and returns the created snippet with generated ID.
func CreateSnippet(c *gin.Context) {
    // Implementation
}
```

**Complex logic:**

```go
// Build dynamic query based on filters
// Priority: category > tag > search
if category != "" {
    query += " AND category = $" + strconv.Itoa(argPos)
    args = append(args, category)
    argPos++
}
```

---

## Common Tasks

### Adding a New Endpoint

1. **Define handler function:**

   ```go
   // In handlers.go or user_handlers.go
   func newEndpoint(c *gin.Context) {
       // Implementation
       c.JSON(http.StatusOK, gin.H{"message": "success"})
   }
   ```

2. **Register route:**

   ```go
   // In main.go
   api := router.Group("/api/v1")
   api.GET("/new-endpoint", newEndpoint)
   ```

3. **Add tests:**

   ```go
   // In handlers_test.go
   func TestNewEndpoint(t *testing.T) {
       // Test implementation
   }
   ```

4. **Update documentation:**
   - Add to `docs/API.md`

### Adding a Database Field

1. **Update model:**

   ```go
   // In models.go
   type Snippet struct {
       // ... existing fields
       NewField string `json:"new_field"`
   }
   ```

2. **Update schema:**

   ```go
   // In database.go
   ALTER TABLE snippets ADD COLUMN new_field VARCHAR(255);
   ```

3. **Update scanning:**

   ```go
   // In scanSnippet function
   err := row.Scan(
       &snippet.ID,
       // ... existing fields
       &snippet.NewField,
   )
   ```

4. **Update queries:**

   - Add to SELECT statements
   - Add to INSERT statements
   - Add to UPDATE statements

5. **Update tests:**
   - Update test data
   - Update expected values

### Adding a New Filter

1. **Extract query parameter:**

   ```go
   func getSnippets(c *gin.Context) {
       newFilter := c.Query("new_filter")
   ```

2. **Add to query builder:**

   ```go
   if newFilter != "" {
       query += " AND new_field = $" + strconv.Itoa(argPos)
       args = append(args, newFilter)
       argPos++
   }
   ```

3. **Update documentation:**
   - Add to API docs
   - Add example usage

---

## Troubleshooting

### Database Connection Issues

**Error: "Cannot connect to PostgreSQL"**

1. Check if PostgreSQL is running:

   ```bash
   docker-compose ps
   ```

2. Restart PostgreSQL:

   ```bash
   docker-compose restart
   ```

3. Check logs:

   ```bash
   docker-compose logs postgres
   ```

4. Verify connection string in `.env`

### Port Already in Use

**Error: "bind: address already in use"**

1. Find process using port 8080:

   ```bash
   lsof -i :8080
   ```

2. Kill the process:

   ```bash
   kill -9 <PID>
   ```

3. Or change port in `.env`:
   ```env
   PORT=8081
   ```

### Test Failures

**Tests skip with "Cannot connect to PostgreSQL"**

This is expected if PostgreSQL isn't running. Start it:

```bash
docker-compose up -d
```

**Tests fail with schema errors**

Reset database:

```bash
docker-compose down -v
docker-compose up -d
go run .  # Recreates schema
```

### Build Issues

**Error: "package not found"**

```bash
go mod tidy
go mod download
```

**Error: "undefined: function"**

Check imports and ensure all files are in same package.

### Runtime Errors

**Panic: "assignment to entry in nil map"**

Initialize maps before use:

```go
// ❌ Bad
var m map[string]string
m["key"] = "value" // Panic!

// ✅ Good
m := make(map[string]string)
m["key"] = "value"
```

**Panic: "invalid memory address or nil pointer dereference"**

Check for nil before dereferencing:

```go
if pointer != nil {
    value := *pointer
}
```

---

## Performance Tips

### Database

1. **Use connection pooling** (already configured)
2. **Add indexes** for frequently queried columns
3. **Use EXPLAIN ANALYZE** to check query performance:
   ```sql
   EXPLAIN ANALYZE SELECT * FROM snippets WHERE category = 'git';
   ```

### Application

1. **Pre-allocate slices:**

   ```go
   // ✅ Good - avoids reallocation
   snippets := make([]*Snippet, 0, 10)

   // ❌ Less efficient
   var snippets []*Snippet
   ```

2. **Avoid unnecessary allocations:**

   ```go
   // ✅ Good
   query := strings.Builder{}
   query.WriteString("SELECT * FROM ")

   // ❌ Bad - creates many temporary strings
   query := "SELECT * " + "FROM " + table
   ```

3. **Use benchmarks:**
   ```go
   func BenchmarkFunction(b *testing.B) {
       for i := 0; i < b.N; i++ {
           functionUnderTest()
       }
   }
   ```

---

## Useful Commands

```bash
# Run with race detector
go run -race .

# Run benchmarks
go test -bench=. -benchmem

# Check for vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Generate documentation
go doc -all > API_DOCS.txt

# Profile CPU usage
go test -cpuprofile cpu.prof
go tool pprof cpu.prof

# Profile memory usage
go test -memprofile mem.prof
go tool pprof mem.prof
```

---

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

---

## Getting Help

- **Issues**: Check existing issues on GitHub
- **Documentation**: Read the docs in the `/docs` folder
- **Community**: Ask questions in discussions
- **Code**: Look at existing patterns in the codebase

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Write/update tests
5. Update documentation
6. Submit a pull request

See `CONTRIBUTING.md` for detailed guidelines (if available).
