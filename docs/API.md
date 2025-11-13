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

Authenticate a user and receive access token and refresh token.

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
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "email": "user@example.com",
    "created_at": "2025-11-10T10:30:00Z",
    "updated_at": "2025-11-10T10:30:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "abc123def456...",
  "expires_in": 900
}
```

**Response Fields:**
- `access_token`: Short-lived JWT token (15 minutes) for API requests
- `refresh_token`: Long-lived token (30 days) for obtaining new access tokens
- `expires_in`: Access token lifetime in seconds (900 = 15 minutes)

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

### Refresh Access Token

Get a new access token using a refresh token.

**Endpoint:** `POST /auth/refresh`

**Request Body:**

```json
{
  "refresh_token": "abc123def456..."
}
```

**Success Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Error Responses:**

- `400 Bad Request` - Invalid or missing refresh token
- `401 Unauthorized` - Token expired or revoked
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "abc123def456..."
  }'
```

---

### Logout (Single Device)

Revoke the current refresh token.

**Endpoint:** `POST /auth/logout`

**Request Body:**

```json
{
  "refresh_token": "abc123def456..."
}
```

**Success Response (200 OK):**

```json
{
  "message": "Logged out successfully"
}
```

**Error Responses:**

- `400 Bad Request` - Invalid or missing refresh token
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "abc123def456..."
  }'
```

---

### Logout All Devices

Revoke all refresh tokens for the authenticated user.

**Endpoint:** `POST /auth/logout-all`

**Authentication Required:** Yes (Bearer token)

**Success Response (200 OK):**

```json
{
  "message": "Logged out from all devices successfully"
}
```

**Error Responses:**

- `401 Unauthorized` - Missing or invalid access token
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout-all \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

---

## Users

### Create User (Register)

Register a new user account.

**Endpoint:** `POST /auth/register`

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
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "user@example.com",
    "password": "securePassword123"
  }'
```

---

### Get All Users

Retrieve a list of all users. **Requires authentication.**

**Endpoint:** `GET /users`

**Headers:**
- `Authorization: Bearer <access_token>`

**Success Response (200 OK):**

```json
{
  "users": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "johndoe",
      "email": "user@example.com",
      "created_at": "2025-11-10T10:30:00Z",
      "updated_at": "2025-11-10T10:30:00Z"
    }
  ],
  "count": 1
}
```

**Example:**

```bash
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <access_token>"
```

---

### Get Current User Profile

Retrieve the authenticated user's profile. **Requires authentication.**

**Endpoint:** `GET /users/profile`

**Headers:**
- `Authorization: Bearer <access_token>`

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

**Example:**

```bash
curl http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer <access_token>"
```

---

### Get User by ID

Retrieve a specific user by their UUID. **Requires authentication.**

**Endpoint:** `GET /users/:id`

**Headers:**
- `Authorization: Bearer <access_token>`

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
curl http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <access_token>"
```

---

### Update Current User Profile

Update the authenticated user's profile. **Requires authentication.**

**Endpoint:** `PUT /users/profile`

**Headers:**
- `Authorization: Bearer <access_token>`

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
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X PUT http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newusername"
  }'
```

---

### Update User by ID

Update a user by their ID. **Requires authentication.**

**Endpoint:** `PUT /users/:id`

**Headers:**
- `Authorization: Bearer <access_token>`

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
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newusername"
  }'
```

---

### Delete User

Delete a user account. **Requires authentication.**

**Endpoint:** `DELETE /users/:id`

**Headers:**
- `Authorization: Bearer <access_token>`

**URL Parameters:**

- `id` (UUID) - User ID

**Success Response (200 OK):**

```json
{
  "message": "User deleted successfully"
}
```

**Error Responses:**

- `404 Not Found` - User not found
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X DELETE http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <access_token>"
```

---

## Snippets

### Create Snippet

Create a new code snippet. **Requires authentication.** The snippet is automatically assigned to the authenticated user.

**Endpoint:** `POST /snippets`

**Headers:**
- `Authorization: Bearer <access_token>`

**Request Body:**

```json
{
  "label": "Git Status Shortcut",
  "shortcut": "gst",
  "content": "git status -sb",
  "tags": ["git", "shortcuts"]
}
```

**Validation Rules:**

- `label`: Required
- `shortcut`: Optional, no spaces allowed
- `content`: Required
- `tags`: Optional array of strings

**Success Response (201 Created):**

```json
{
  "id": 1,
  "label": "Git Status Shortcut",
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
- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/snippets \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "Git Status Shortcut",
    "shortcut": "gst",
    "content": "git status -sb",
    "tags": ["git", "shortcuts"]
  }'
```

---

### Get User Snippets

Retrieve all snippets for the authenticated user with optional filtering. **Requires authentication.**

**Endpoint:** `GET /snippets`

**Headers:**
- `Authorization: Bearer <access_token>`

**Query Parameters:**

- `tag` (string) - Filter by tag
- `search` (string) - Full-text search in label and content
- `limit` (integer) - Limit results (default: all)

**Success Response (200 OK):**

```json
{
  "snippets": [
    {
      "id": 1,
      "label": "Git Status Shortcut",
      "shortcut": "gst",
      "content": "git status -sb",
      "tags": ["git", "shortcuts"],
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "created_at": "2025-11-10T10:30:00Z",
      "updated_at": "2025-11-10T10:30:00Z"
    }
  ],
  "count": 1
}
```

**Examples:**

```bash
# Get all user snippets
curl http://localhost:8080/api/v1/snippets \
  -H "Authorization: Bearer <access_token>"

# Filter by tag
curl http://localhost:8080/api/v1/snippets?tag=git \
  -H "Authorization: Bearer <access_token>"

# Search snippets
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

**Headers:**
- `Authorization: Bearer <access_token>`

**URL Parameters:**

- `id` (integer) - Snippet ID

**Success Response (200 OK):**

```json
{
  "id": 1,
  "label": "Git Status Shortcut",
  "shortcut": "gst",
  "content": "git status -sb",
  "tags": ["git", "shortcuts"],
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:00Z"
}
```

**Error Responses:**

- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - Snippet not found

**Example:**

```bash
curl http://localhost:8080/api/v1/snippets/1 \
  -H "Authorization: Bearer <access_token>"
```

---

### Update Snippet

Update an existing snippet. **Requires authentication.** Only the owner can update their snippets.

**Endpoint:** `PUT /snippets/:id`

**Headers:**
- `Authorization: Bearer <access_token>`

**URL Parameters:**

- `id` (integer) - Snippet ID

**Request Body:**

```json
{
  "label": "Updated Git Status",
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
  "label": "Updated Git Status",
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
- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Not authorized to update this snippet
- `404 Not Found` - Snippet not found
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X PUT http://localhost:8080/api/v1/snippets/1 \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "Updated Git Status",
    "content": "git status -sb --show-stash"
  }'
```

---

### Delete Snippet

Delete a snippet. **Requires authentication.** Only the owner can delete their snippets.

**Endpoint:** `DELETE /snippets/:id`

**Headers:**
- `Authorization: Bearer <access_token>`

**URL Parameters:**

- `id` (integer) - Snippet ID

**Success Response (200 OK):**

```json
{
  "message": "Snippet deleted successfully"
}
```

**Error Responses:**

- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Not authorized to delete this snippet
- `404 Not Found` - Snippet not found
- `500 Internal Server Error` - Server error

**Example:**

```bash
curl -X DELETE http://localhost:8080/api/v1/snippets/1 \
  -H "Authorization: Bearer <access_token>"
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
