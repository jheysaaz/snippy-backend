# Architecture Documentation

Complete architectural overview of Snippy Backend.

## Project Structure

```
snippy-backend/
├── main.go                 # Application entry point
├── database.go            # Database connection and initialization
├── models.go              # Data models and scanning functions
├── handlers.go            # Snippet endpoint handlers
├── user_handlers.go       # User and auth endpoint handlers
├── middleware.go          # CORS middleware
├── database_test.go       # Database integration tests
├── handlers_test.go       # Handler integration tests
├── models_test.go         # Model unit tests
├── go.mod                 # Go module dependencies
├── go.sum                 # Go module checksums
├── .env.example           # Environment variables template
├── docker-compose.yml     # PostgreSQL container setup
├── README.md              # Project overview
└── docs/
    ├── API.md             # API reference
    ├── DATABASE.md        # Database documentation
    └── ARCHITECTURE.md    # This file
```

---

## Architecture Overview

Snippy Backend follows a **layered architecture** pattern with clear separation of concerns:

```
┌─────────────────────────────────────────────┐
│          HTTP Layer (Gin Router)            │
│  - CORS Middleware                          │
│  - Route Registration                       │
│  - Request/Response Handling                │
└─────────────────┬───────────────────────────┘
                  │
┌─────────────────▼───────────────────────────┐
│         Handler Layer                       │
│  - Request Validation                       │
│  - Business Logic                           │
│  - Response Formatting                      │
└─────────────────┬───────────────────────────┘
                  │
┌─────────────────▼───────────────────────────┐
│         Model Layer                         │
│  - Data Structures                          │
│  - Data Validation                          │
│  - Row Scanning Functions                   │
└─────────────────┬───────────────────────────┘
                  │
┌─────────────────▼───────────────────────────┐
│         Database Layer                      │
│  - PostgreSQL Connection                    │
│  - Connection Pooling                       │
│  - Schema Initialization                    │
│  - Raw SQL Queries                          │
└─────────────────────────────────────────────┘
```

---

## Component Details

### 1. Main Application (`main.go`)

**Responsibilities:**

- Application bootstrap
- Environment variable loading
- Database initialization
- Router setup
- Server startup

**Key Functions:**

```go
func main()
```

**Flow:**

1. Load environment variables from `.env`
2. Initialize database connection
3. Set up Gin router with CORS middleware
4. Register all routes (health, auth, users, snippets)
5. Start HTTP server on configured port

---

### 2. Database Layer (`database.go`)

**Responsibilities:**

- Database connection management
- Connection pooling configuration
- Schema initialization
- Table and index creation

**Key Functions:**

```go
func InitDB() (*sql.DB, error)
```

**Features:**

- **Connection Pooling**: 25 max connections, 5 idle, 5-minute timeout
- **UUID Extension**: Automatically enables `uuid-ossp`
- **Auto-Schema Creation**: Creates tables, indexes, and triggers on startup
- **Error Handling**: Graceful handling of connection failures

**Connection String Format:**

```
postgres://user:password@host:port/dbname?sslmode=disable
```

---

### 3. Model Layer (`models.go`)

**Responsibilities:**

- Define data structures
- Provide row scanning utilities
- Handle data transformation

**Data Structures:**

#### User

```go
type User struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Password  string    `json:"password,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

#### Snippet

```go
type Snippet struct {
    ID          int64     `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Category    string    `json:"category"`
    Shortcut    string    `json:"shortcut"`
    Content     string    `json:"content"`
    Tags        []string  `json:"tags"`
    UserID      *string   `json:"user_id"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

#### Request Models

```go
type CreateSnippetRequest struct {
    Title       string   `json:"title"`
    Description string   `json:"description"`
    Category    string   `json:"category"`
    Shortcut    string   `json:"shortcut"`
    Content     string   `json:"content"`
    Tags        []string `json:"tags"`
    UserID      *string  `json:"userId"`
}

type UpdateSnippetRequest struct {
    Title       *string  `json:"title"`
    Description *string  `json:"description"`
    Category    *string  `json:"category"`
    Shortcut    *string  `json:"shortcut"`
    Content     *string  `json:"content"`
    Tags        []string `json:"tags"`
}
```

**Key Functions:**

```go
func scanSnippet(row scanner) (*Snippet, error)
func scanUser(row scanner) (*User, error)
```

---

### 4. Handler Layer

#### Snippet Handlers (`handlers.go`)

**Endpoints:**

- `GET /api/v1/health` - Health check
- `GET /api/v1/snippets` - List snippets with filtering
- `GET /api/v1/snippets/:id` - Get single snippet
- `POST /api/v1/snippets` - Create snippet
- `PUT /api/v1/snippets/:id` - Update snippet
- `DELETE /api/v1/snippets/:id` - Delete snippet

**Key Features:**

- Dynamic query building for filters
- Full-text search support
- Tag-based filtering
- Category filtering
- Result limiting (max 100)
- Pre-allocated slices for performance

**Example Handler Pattern:**

```go
func getSnippets(c *gin.Context) {
    // 1. Extract query parameters
    category := c.Query("category")
    tag := c.Query("tag")
    search := c.Query("search")

    // 2. Build dynamic SQL query
    query := "SELECT ... FROM snippets WHERE 1=1"
    args := []interface{}{}

    // 3. Add filters
    if category != "" {
        query += " AND category = $1"
        args = append(args, category)
    }

    // 4. Execute query
    rows, err := db.Query(query, args...)

    // 5. Scan results
    snippets := make([]*Snippet, 0, 10)
    for rows.Next() {
        snippet, _ := scanSnippet(rows)
        snippets = append(snippets, snippet)
    }

    // 6. Return response
    c.JSON(http.StatusOK, snippets)
}
```

#### User Handlers (`user_handlers.go`)

**Endpoints:**

- `POST /api/v1/auth/login` - User login
- `GET /api/v1/users` - List all users
- `GET /api/v1/users/:id` - Get user by ID
- `GET /api/v1/users/username/:username` - Get user by username
- `POST /api/v1/users` - Create user (register)
- `PUT /api/v1/users/:id` - Update user
- `DELETE /api/v1/users/:id` - Delete user
- `GET /api/v1/users/:id/snippets` - Get user's snippets

**Security Features:**

- Argon2id password hashing
- Password field omitted from responses
- Input validation
- SQL injection prevention

**Password Hashing Parameters:**

```go
memory:      64 * 1024  // 64 MB
iterations:  1
parallelism: 4
saltLength:  16
keyLength:   32
```

---

### 5. Middleware Layer (`middleware.go`)

**CORS Middleware:**

```go
func CORSMiddleware() gin.HandlerFunc
```

**Configuration:**

- Origins: `*` (all origins allowed)
- Methods: `GET, POST, PUT, DELETE, OPTIONS`
- Headers: `Content-Type, Content-Length, Accept-Encoding, Authorization, Accept, Origin`
- Max Age: 12 hours

**Purpose:** Enable browser extensions and web applications to interact with the API

---

## Request Flow

### Example: Creating a Snippet

```
1. Client sends POST request
   ↓
2. CORS middleware processes request
   ↓
3. Gin router matches route → createSnippet handler
   ↓
4. Handler validates request body
   ↓
5. Handler builds SQL INSERT query
   ↓
6. Database executes query
   ↓
7. Handler scans returned row
   ↓
8. Handler formats response
   ↓
9. Gin sends JSON response to client
```

### Example: Filtered Search

```
1. Client sends GET /api/v1/snippets?category=git&search=status
   ↓
2. CORS middleware processes request
   ↓
3. Gin router matches route → getSnippets handler
   ↓
4. Handler extracts query parameters
   ↓
5. Handler builds dynamic SQL with filters
   ↓
6. Database executes query with full-text search
   ↓
7. Handler scans multiple rows
   ↓
8. Handler returns JSON array
   ↓
9. Client receives filtered results
```

---

## Data Flow

### User Registration Flow

```
Client Request
    ↓
[Validation]
    ↓
[Argon2id Hashing]
    ↓
[UUID Generation (DB)]
    ↓
[Insert User]
    ↓
[Return User (no password)]
    ↓
Client Response
```

### Snippet Creation Flow

```
Client Request
    ↓
[Validation]
    ↓
[Check User Exists (if userId provided)]
    ↓
[Insert Snippet]
    ↓
[Auto-generate ID (DB)]
    ↓
[Set Timestamps (DB)]
    ↓
[Return Snippet]
    ↓
Client Response
```

---

## Error Handling Strategy

### HTTP Status Codes

| Code | Use Case                      |
| ---- | ----------------------------- |
| 200  | Successful GET/PUT request    |
| 201  | Successfully created resource |
| 204  | Successfully deleted resource |
| 400  | Invalid request data          |
| 404  | Resource not found            |
| 500  | Server/database error         |

### Error Response Format

```json
{
  "error": "Descriptive error message"
}
```

### Error Handling Patterns

1. **Validation Errors**: Return 400 with specific message
2. **Not Found**: Return 404 with "Resource not found"
3. **Database Errors**: Return 500 with generic message (don't leak details)
4. **Constraint Violations**: Return 400 with user-friendly message

---

## Performance Considerations

### 1. Database Optimizations

- **Connection Pooling**: Reuse connections instead of creating new ones
- **Indexes**: Strategic B-tree and GIN indexes for fast queries
- **Prepared Statements**: Allow PostgreSQL to cache query plans
- **Pre-allocated Slices**: Reduce memory allocations in hot paths

### 2. Query Optimizations

- **Full-Text Search**: Use GIN index instead of LIKE queries
- **Array Operations**: Use GIN index for tag filtering
- **Limited Results**: Cap at 100 to prevent memory issues
- **Selective Columns**: Only SELECT needed columns

### 3. Application Optimizations

- **Minimal Allocations**: Pre-allocate slices with capacity
- **Direct Scanning**: Scan rows directly into structs
- **Early Returns**: Fail fast on validation errors
- **Efficient JSON**: Use Gin's optimized JSON encoder

---

## Security Architecture

### 1. Authentication

- **Password Hashing**: Argon2id (memory-hard algorithm)
- **No Plain Text**: Passwords never stored in plain text
- **Salt**: Unique salt per password

### 2. Authorization

- **Currently**: No authorization layer (all operations allowed)
- **Future**: Consider JWT tokens or session management

### 3. Database Security

- **Parameterized Queries**: Prevent SQL injection
- **Connection String**: Keep credentials in environment variables
- **Least Privilege**: Use dedicated database user with minimal permissions

### 4. API Security

- **CORS**: Configured for extension use
- **Input Validation**: Validate all user inputs
- **Error Messages**: Don't leak sensitive information

---

## Testing Strategy

### Unit Tests (`models_test.go`)

- Test data models
- Test validation logic
- Mock database interactions

### Integration Tests (`handlers_test.go`, `database_test.go`)

- Test full request/response cycle
- Test database operations
- Skip when PostgreSQL unavailable

### Test Coverage Areas

- ✅ Model scanning functions
- ✅ Request validation
- ✅ Health endpoint
- ✅ Database schema integrity
- ✅ Database triggers

---

## Deployment Architecture

### Development

```
Developer Machine
    ↓
[Docker Compose]
    ├── PostgreSQL (port 5432)
    └── Local Go Server (port 8080)
```

### Production (Recommended)

```
Internet
    ↓
[Load Balancer / Reverse Proxy]
    ↓
[Multiple Go Server Instances]
    ↓
[PostgreSQL Primary]
    ├── Replica 1
    └── Replica 2
```

---

## Environment Configuration

### Required Variables

- `DB_HOST`: PostgreSQL host
- `DB_PORT`: PostgreSQL port
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password
- `DB_NAME`: Database name

### Optional Variables

- `PORT`: HTTP server port (default: 8080)
- `DB_SSLMODE`: SSL mode (default: disable)

---

## Extension Points

### Adding New Endpoints

1. Define handler function in appropriate file
2. Register route in `main.go`
3. Add tests
4. Update API documentation

### Adding New Models

1. Define struct in `models.go`
2. Create scanning function
3. Add validation logic
4. Update database schema

### Adding Middleware

1. Create middleware function in `middleware.go`
2. Apply to router or specific routes in `main.go`
3. Test behavior

---

## Future Improvements

### Short Term

- [ ] JWT-based authentication
- [ ] Rate limiting middleware
- [ ] Request logging
- [ ] Metrics/monitoring endpoints

### Medium Term

- [ ] Role-based access control (RBAC)
- [ ] API versioning strategy
- [ ] GraphQL endpoint
- [ ] Caching layer (Redis)

### Long Term

- [ ] Microservices architecture
- [ ] Event-driven updates
- [ ] Real-time WebSocket support
- [ ] Multi-tenancy support

---

## Dependencies

### Core Dependencies

- **Gin**: v1.11.0 - HTTP web framework
- **lib/pq**: v1.10.9 - PostgreSQL driver
- **godotenv**: v1.5.1 - Environment variable loading
- **Argon2**: x/crypto - Password hashing

### Why These Choices?

1. **Gin**: Fast, well-documented, middleware support
2. **lib/pq**: Pure Go, stable, feature-complete
3. **Argon2**: Industry-standard password hashing
4. **No ORM**: Direct SQL for performance and control

---

## Code Style & Conventions

### Naming

- **Files**: lowercase with underscores (e.g., `user_handlers.go`)
- **Functions**: camelCase, public start with uppercase
- **Variables**: camelCase
- **Constants**: UPPER_SNAKE_CASE

### Error Handling

```go
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Message"})
    return
}
```

### JSON Tags

```go
type Model struct {
    Field string `json:"field_name"`
}
```

### SQL Queries

- Use parameterized queries (`$1`, `$2`)
- Multi-line strings for readability
- Clear column selection

---

## Monitoring & Observability

### Health Check

```bash
curl http://localhost:8080/api/v1/health
```

### Database Connection Check

```go
if err := db.Ping(); err != nil {
    // Handle connection issue
}
```

### Recommended Tools

- **Prometheus**: Metrics collection
- **Grafana**: Visualization
- **Loki**: Log aggregation
- **Jaeger**: Distributed tracing

---

## License

MIT License - See LICENSE file for details
