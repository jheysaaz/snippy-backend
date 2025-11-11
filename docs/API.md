# API Documentation

Complete API reference for Snippy Backend.

## Base URL

```
http://localhost:8080/api/v1
```

## Table of Contents

- [Authentication](#authentication)
- [Users](#users)
- [Snippets](#snippets)
- [Error Responses](#error-responses)

---

## Authentication

### Login

Authenticate a user and receive user information.

**Endpoint:** `POST /auth/login`

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Success Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "johndoe",
  "email": "user@example.com",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:00Z"
}
```

**Error Responses:**

- `400 Bad Request` - Invalid credentials
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securePassword123"
  }'
```

---

## Users

### Create User (Register)

Register a new user account.

**Endpoint:** `POST /users`

**Request Body:**

```json
{
  "username": "johndoe",
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Validation Rules:**

- `username`: Required, 3-50 characters, alphanumeric with underscores
- `email`: Required, valid email format
- `password`: Required, minimum 8 characters

**Success Response (201 Created):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "johndoe",
  "email": "user@example.com",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:00Z"
}
```

**Error Responses:**

- `400 Bad Request` - Validation error or duplicate username/email
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "user@example.com",
    "password": "securePassword123"
  }'
```

---

### Get All Users

Retrieve a list of all users.

**Endpoint:** `GET /users`

**Query Parameters:** None

**Success Response (200 OK):**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "email": "user@example.com",
    "created_at": "2025-11-10T10:30:00Z",
    "updated_at": "2025-11-10T10:30:00Z"
  }
]
```

**Example:**

```bash
curl http://localhost:8080/api/v1/users
```

---

### Get User by ID

Retrieve a specific user by their UUID.

**Endpoint:** `GET /users/:id`

**URL Parameters:**

- `id` (UUID) - User ID

**Success Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "johndoe",
  "email": "user@example.com",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:00Z"
}
```

**Error Responses:**

- `404 Not Found` - User not found

**Example:**

```bash
curl http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000
```

---

### Get User by Username

Retrieve a user by their username.

**Endpoint:** `GET /users/username/:username`

**URL Parameters:**

- `username` (string) - Username

**Success Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "johndoe",
  "email": "user@example.com",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:00Z"
}
```

**Error Responses:**

- `404 Not Found` - User not found

**Example:**

```bash
curl http://localhost:8080/api/v1/users/username/johndoe
```

---

### Update User

Update user information.

**Endpoint:** `PUT /users/:id`

**URL Parameters:**

- `id` (UUID) - User ID

**Request Body:**

```json
{
  "username": "newusername",
  "email": "newemail@example.com",
  "password": "newPassword123"
}
```

**Note:** All fields are optional. Only provided fields will be updated.

**Success Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "newusername",
  "email": "newemail@example.com",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T11:00:00Z"
}
```

**Error Responses:**

- `400 Bad Request` - No valid fields provided or duplicate username/email
- `404 Not Found` - User not found
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X PUT http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newusername"
  }'
```

---

### Delete User

Delete a user account.

**Endpoint:** `DELETE /users/:id`

**URL Parameters:**

- `id` (UUID) - User ID

**Success Response (204 No Content)**

**Error Responses:**

- `404 Not Found` - User not found
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X DELETE http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000
```

---

### Get User's Snippets

Retrieve all snippets created by a specific user.

**Endpoint:** `GET /users/:id/snippets`

**URL Parameters:**

- `id` (UUID) - User ID

**Success Response (200 OK):**

```json
[
  {
    "id": 1,
    "title": "Git Status Shortcut",
    "description": "Quick command to check git status",
    "category": "git",
    "shortcut": "gst",
    "content": "git status -sb",
    "tags": ["git", "shortcuts"],
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-11-10T10:30:00Z",
    "updated_at": "2025-11-10T10:30:00Z"
  }
]
```

**Error Responses:**

- `404 Not Found` - User not found
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000/snippets
```

---

## Snippets

### Create Snippet

Create a new code snippet.

**Endpoint:** `POST /snippets`

**Request Body:**

```json
{
  "title": "Git Status Shortcut",
  "description": "Quick command to check git status",
  "category": "git",
  "shortcut": "gst",
  "content": "git status -sb",
  "tags": ["git", "shortcuts"],
  "userId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Validation Rules:**

- `title`: Required
- `category`: Required
- `shortcut`: Required, no spaces allowed
- `content`: Required
- `description`: Optional
- `tags`: Optional array of strings
- `userId`: Optional UUID

**Success Response (201 Created):**

```json
{
  "id": 1,
  "title": "Git Status Shortcut",
  "description": "Quick command to check git status",
  "category": "git",
  "shortcut": "gst",
  "content": "git status -sb",
  "tags": ["git", "shortcuts"],
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:00Z"
}
```

**Error Responses:**

- `400 Bad Request` - Validation error
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/snippets \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Git Status Shortcut",
    "description": "Quick command to check git status",
    "category": "git",
    "shortcut": "gst",
    "content": "git status -sb",
    "tags": ["git", "shortcuts"],
    "userId": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

---

### Get All Snippets

Retrieve snippets with optional filtering.

**Endpoint:** `GET /snippets`

**Query Parameters:**

- `category` (string) - Filter by category (e.g., "git", "docker", "kubernetes")
- `tag` (string) - Filter by tag
- `search` (string) - Full-text search in title and description
- `limit` (integer) - Limit results (max 100, default: all)

**Success Response (200 OK):**

```json
[
  {
    "id": 1,
    "title": "Git Status Shortcut",
    "description": "Quick command to check git status",
    "category": "git",
    "shortcut": "gst",
    "content": "git status -sb",
    "tags": ["git", "shortcuts"],
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-11-10T10:30:00Z",
    "updated_at": "2025-11-10T10:30:00Z"
  }
]
```

**Examples:**

```bash
# Get all snippets
curl http://localhost:8080/api/v1/snippets

# Filter by category
curl http://localhost:8080/api/v1/snippets?category=git

# Filter by tag
curl http://localhost:8080/api/v1/snippets?tag=shortcuts

# Full-text search
curl http://localhost:8080/api/v1/snippets?search=status

# Combine filters
curl "http://localhost:8080/api/v1/snippets?category=git&tag=shortcuts&limit=10"
```

---

### Get Snippet by ID

Retrieve a specific snippet.

**Endpoint:** `GET /snippets/:id`

**URL Parameters:**

- `id` (integer) - Snippet ID

**Success Response (200 OK):**

```json
{
  "id": 1,
  "title": "Git Status Shortcut",
  "description": "Quick command to check git status",
  "category": "git",
  "shortcut": "gst",
  "content": "git status -sb",
  "tags": ["git", "shortcuts"],
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:00Z"
}
```

**Error Responses:**

- `404 Not Found` - Snippet not found

**Example:**

```bash
curl http://localhost:8080/api/v1/snippets/1
```

---

### Update Snippet

Update an existing snippet.

**Endpoint:** `PUT /snippets/:id`

**URL Parameters:**

- `id` (integer) - Snippet ID

**Request Body:**

```json
{
  "title": "Updated Git Status",
  "description": "Enhanced git status command",
  "category": "git",
  "shortcut": "gst",
  "content": "git status -sb --show-stash",
  "tags": ["git", "shortcuts", "enhanced"]
}
```

**Note:** All fields are optional. Only provided fields will be updated.

**Success Response (200 OK):**

```json
{
  "id": 1,
  "title": "Updated Git Status",
  "description": "Enhanced git status command",
  "category": "git",
  "shortcut": "gst",
  "content": "git status -sb --show-stash",
  "tags": ["git", "shortcuts", "enhanced"],
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T11:00:00Z"
}
```

**Error Responses:**

- `400 Bad Request` - No valid fields provided or validation error
- `404 Not Found` - Snippet not found
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X PUT http://localhost:8080/api/v1/snippets/1 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated Git Status",
    "content": "git status -sb --show-stash"
  }'
```

---

### Delete Snippet

Delete a snippet.

**Endpoint:** `DELETE /snippets/:id`

**URL Parameters:**

- `id` (integer) - Snippet ID

**Success Response (204 No Content)**

**Error Responses:**

- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X DELETE http://localhost:8080/api/v1/snippets/1
```

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

### Common HTTP Status Codes

- `200 OK` - Request succeeded
- `201 Created` - Resource created successfully
- `204 No Content` - Request succeeded with no content to return
- `400 Bad Request` - Invalid request data or validation error
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

### Error Examples

**Validation Error:**

```json
{
  "error": "Title is required"
}
```

**Not Found:**

```json
{
  "error": "Snippet not found"
}
```

**Duplicate Entry:**

```json
{
  "error": "Username already exists"
}
```

---

## Rate Limiting

Currently, there is no rate limiting implemented. This may be added in future versions.

## CORS

CORS is enabled for all origins (`*`) to support browser extensions and web applications.

## Security Notes

- Passwords are hashed using Argon2id with the following parameters:

  - Memory: 64 MB
  - Iterations: 1
  - Parallelism: 4 threads
  - Salt length: 16 bytes
  - Key length: 32 bytes

- User IDs are UUIDs (v4) generated by PostgreSQL
- All database queries use parameterized statements to prevent SQL injection
- Timestamps are automatically managed by database triggers
