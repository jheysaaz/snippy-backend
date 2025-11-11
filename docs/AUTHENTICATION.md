# Authentication & Authorization Guide

Complete guide for authentication and authorization in Snippy Backend.

## Overview

Snippy Backend implements **JWT (JSON Web Token)** based authentication with **Refresh Token** support for secure, long-lasting sessions:

- ✅ Short-lived access tokens (15 minutes) for API requests
- ✅ Long-lived refresh tokens (30 days) for persistent sessions
- ✅ Users can only modify their own data
- ✅ Snippets are owned by users
- ✅ Automatic token cleanup to prevent database bloat
- ✅ Protected endpoints require authentication

---

## Authentication Flow

### Initial Login (First Time)
```
1. User registers → POST /api/v1/users
2. User logs in → POST /api/v1/auth/login 
   → Receives: accessToken (15 min) + refreshToken (30 days)
3. Client stores both tokens securely
```

### Normal API Usage
```
1. Client uses accessToken for API requests
   → Authorization: Bearer <accessToken>
2. Server validates token and extracts user info
3. Server checks ownership before allowing modifications
```

### When Access Token Expires
```
1. API returns 401 Unauthorized (token expired)
2. Client automatically calls POST /api/v1/auth/refresh
   → Sends: refreshToken
   → Receives: new accessToken (15 min)
3. Client retries original request with new accessToken
4. User never notices (seamless experience)
```

### When Refresh Token Expires (After 30 Days)
```
1. Refresh endpoint returns 401 (refresh token expired)
2. Client redirects user to login page
3. User logs in again
4. Cycle repeats
```

---

## Token Details

### Access Token
- **Lifetime**: 15 minutes
- **Purpose**: Authorize API requests
- **Storage**: Memory/local storage (frontend)
- **Contains**: UserID, Username, Email, Expiration

### Refresh Token
- **Lifetime**: 30 days
- **Purpose**: Get new access tokens without re-login
- **Storage**: Secure storage (httpOnly cookie or secure storage)
- **Database**: Stored in `refresh_tokens` table with metadata

---

## API Changes

### Public Endpoints (No Authentication Required)

- `GET /api/v1/health` - Health check
- `POST /api/v1/users` - User registration
- `POST /api/v1/auth/login` - User login (returns both tokens)
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout (revoke refresh token)
- `GET /api/v1/snippets` - View all snippets
- `GET /api/v1/snippets/:id` - View a single snippet

### Protected Endpoints (Authentication Required)

**Auth Management:**
- `POST /api/v1/auth/logout-all` - Logout from all devices (revoke all tokens)

**User Management:**

- `GET /api/v1/users` - List all users (requires auth)
- `GET /api/v1/users/:id` - Get user details (requires auth)
- `GET /api/v1/users/username/:username` - Get user by username (requires auth)
- `PUT /api/v1/users/:id` - Update user (can only update own profile)
- `DELETE /api/v1/users/:id` - Delete user (can only delete own account)
- `GET /api/v1/users/:id/snippets` - Get user's snippets (requires auth)

**Snippet Management:**
- `POST /api/v1/snippets` - Create snippet (requires auth, automatically assigned to user)
- `PUT /api/v1/snippets/:id` - Update snippet (can only update own snippets)
- `DELETE /api/v1/snippets/:id` - Delete snippet (can only delete own snippets)

---

## Usage Examples

### 1. Register a New User

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "securePassword123"
  }'
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "johndoe",
  "email": "john@example.com",
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:00Z"
}
```

---

### 2. Login and Get Tokens

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securePassword123"
  }'
```

**Response:**

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "email": "john@example.com",
    "created_at": "2025-11-10T10:30:00Z",
    "updated_at": "2025-11-10T10:30:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiNTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAwIiwidXNlcm5hbWUiOiJqb2huZG9lIiwiZW1haWwiOiJqb2huQGV4YW1wbGUuY29tIiwiZXhwIjoxNzMxMjYyMjAwfQ.signature",
  "refresh_token": "abc123def456ghi789jkl012mno345pqr678stu901vwx234yz",
  "expires_in": 900
}
```

**Important:** Save both tokens!
- Use the `access_token` for API requests (valid for 15 minutes)
- Use the `refresh_token` to get new access tokens (valid for 30 days)
- `expires_in` is in seconds (900 = 15 minutes)

---

### 3. Create a Snippet (Authenticated)

```bash
curl -X POST http://localhost:8080/api/v1/snippets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "title": "Git Status Shortcut",
    "description": "Quick git status command",
    "category": "git",
    "shortcut": "gst",
    "content": "git status -sb",
    "tags": ["git", "shortcuts"]
  }'
```

**Note:** The snippet is automatically assigned to the authenticated user. No need to provide `userId` in the request.

**Response:**

```json
{
  "id": 1,
  "title": "Git Status Shortcut",
  "description": "Quick git status command",
  "category": "git",
  "shortcut": "gst",
  "content": "git status -sb",
  "tags": ["git", "shortcuts"],
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-11-10T10:35:00Z",
  "updated_at": "2025-11-10T10:35:00Z"
}
```

---

### 4. Refresh Your Access Token

When your access token expires (after 15 minutes), use the refresh token to get a new one:

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "abc123def456ghi789jkl012mno345pqr678stu901vwx234yz"
  }'
```

**Response:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.NEW_TOKEN_HERE.signature",
  "expires_in": 900
}
```

**Note:** Continue using the same refresh token until it expires (30 days) or you logout.

---

### 5. Logout (Single Device)

Revoke your current refresh token (logout from current device):

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "abc123def456ghi789jkl012mno345pqr678stu901vwx234yz"
  }'
```

**Response:**

```json
{
  "message": "Logged out successfully"
}
```

---

### 6. Logout All Devices

Revoke all refresh tokens for your account (requires access token):

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout-all \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN_HERE"
```

**Response:**

```json
{
  "message": "Logged out from all devices successfully"
}
```

---

### 7. Update Your Own Snippet

```bash
curl -X PUT http://localhost:8080/api/v1/snippets/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "content": "git status -sb --show-stash"
  }'
```

**Success Response (200):**

```json
{
  "id": 1,
  "title": "Git Status Shortcut",
  "content": "git status -sb --show-stash",
  ...
}
```

**Error Response if not owner (403):**

```json
{
  "error": "You don't have permission to update this snippet"
}
```

---

### 8. Try to Update Someone Else's Snippet

```bash
curl -X PUT http://localhost:8080/api/v1/snippets/999 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "title": "Hacked!"
  }'
```

**Response (403 Forbidden):**

```json
{
  "error": "You don't have permission to update this snippet"
}
```

---

### 9. Update Your Profile

```bash
curl -X PUT http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "username": "john_updated"
  }'
```

**Success Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "john_updated",
  "email": "john@example.com",
  ...
}
```

---

### 10. Try to Update Someone Else's Profile

```bash
curl -X PUT http://localhost:8080/api/v1/users/OTHER_USER_ID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "username": "hacked"
  }'
```

**Response (403 Forbidden):**

```json
{
  "error": "You can only update your own profile"
}
```

---

### 11. View Snippets (Public, No Auth Required)

```bash
curl http://localhost:8080/api/v1/snippets
```

**Response:**

```json
[
  {
    "id": 1,
    "title": "Git Status Shortcut",
    ...
  },
  {
    "id": 2,
    "title": "Docker Up",
    ...
  }
]
```

---

## Authorization Rules

### Snippet Ownership

| Action             | Rule                    | HTTP Status on Violation |
| ------------------ | ----------------------- | ------------------------ |
| Create             | Must be authenticated   | 401 Unauthorized         |
| Read (single/list) | Public (no auth needed) | N/A                      |
| Update             | Must be owner           | 403 Forbidden            |
| Delete             | Must be owner           | 403 Forbidden            |

### User Profile

| Action          | Rule                        | HTTP Status on Violation |
| --------------- | --------------------------- | ------------------------ |
| Register        | Public                      | N/A                      |
| Login           | Public                      | N/A                      |
| View (any user) | Must be authenticated       | 401 Unauthorized         |
| Update          | Can only update own profile | 403 Forbidden            |
| Delete          | Can only delete own account | 403 Forbidden            |

---

## JWT Token Details

### Token Structure

```
Header.Payload.Signature
```

**Payload includes:**

- `user_id`: User's UUID
- `username`: Username
- `email`: Email address
- `exp`: Expiration timestamp (24 hours from issue)
- `iat`: Issued at timestamp

### Token Expiration

- **Default:** 24 hours
- **After expiration:** User must login again to get a new token
- **Recommendation:** Store token securely (localStorage/sessionStorage in web apps)

### Security Considerations

1. **Secret Key:** Change `JWT_SECRET` in production to a long, random string
2. **HTTPS:** Always use HTTPS in production to prevent token interception
3. **Token Storage:** Store tokens securely on the client side
4. **Token Refresh:** Consider implementing refresh tokens for better UX

---

## Error Responses

### 401 Unauthorized

**Reason:** No token provided or invalid token

```json
{
  "error": "Authorization header required"
}
```

```json
{
  "error": "Invalid or expired token"
}
```

**Solution:** Login again to get a new token

---

### 403 Forbidden

**Reason:** Token is valid, but user doesn't have permission

```json
{
  "error": "You don't have permission to update this snippet"
}
```

```json
{
  "error": "You can only update your own profile"
}
```

**Solution:** You can only modify your own resources

---

### 404 Not Found

**Reason:** Resource doesn't exist

```json
{
  "error": "Snippet not found"
}
```

**Solution:** Check the resource ID

---

## Environment Configuration

Add to your `.env` file:

```bash
# JWT Configuration (IMPORTANT: Change in production!)
JWT_SECRET=your-very-long-random-secret-key-at-least-32-characters
```

**Generate a secure secret:**

```bash
# On Linux/Mac
openssl rand -base64 64

# Or use online generator
# https://www.grc.com/passwords.htm
```

---

## Testing with cURL

### Full Workflow Example

```bash
# 1. Register
USER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"test123"}')

echo "User registered: $USER_RESPONSE"

# 2. Login and extract token
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}')

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.token')
echo "Token: $TOKEN"

# 3. Create snippet
SNIPPET=$(curl -s -X POST http://localhost:8080/api/v1/snippets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Test","category":"test","shortcut":"tst","content":"test content","tags":["test"]}')

echo "Snippet created: $SNIPPET"

# 4. Get snippet ID
SNIPPET_ID=$(echo $SNIPPET | jq -r '.id')
echo "Snippet ID: $SNIPPET_ID"

# 5. Update snippet
curl -X PUT http://localhost:8080/api/v1/snippets/$SNIPPET_ID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"content":"updated content"}'

# 6. Delete snippet
curl -X DELETE http://localhost:8080/api/v1/snippets/$SNIPPET_ID \
  -H "Authorization: Bearer $TOKEN"
```

---

## Integration with Chrome Extension

### Storing Token in Extension

```javascript
// After login
chrome.storage.local.set({ authToken: token }, () => {
  console.log("Token stored");
});

// Before API requests
chrome.storage.local.get(["authToken"], (result) => {
  const token = result.authToken;

  fetch("http://localhost:8080/api/v1/snippets", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      title: "My Snippet",
      category: "test",
      shortcut: "ms",
      content: "snippet content",
      tags: ["tag1"],
    }),
  });
});
```

### Handling Token Expiration

```javascript
async function apiRequest(url, options = {}) {
  const { authToken } = await chrome.storage.local.get(["authToken"]);

  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      Authorization: `Bearer ${authToken}`,
    },
  });

  if (response.status === 401) {
    // Token expired - redirect to login
    chrome.runtime.sendMessage({ type: "TOKEN_EXPIRED" });
    throw new Error("Authentication required");
  }

  return response.json();
}
```

---

## Migration from Old Version

### Breaking Changes

1. **Snippet Creation:** No longer accepts `userId` in request body

   - **Old:** `{ ..., "userId": "some-uuid" }`
   - **New:** User ID extracted from JWT token

2. **Authentication Required:** Some endpoints now require authentication

   - **POST /snippets** - Now requires auth
   - **PUT /snippets/:id** - Now requires auth + ownership
   - **DELETE /snippets/:id** - Now requires auth + ownership
   - **PUT /users/:id** - Now requires auth + self-check
   - **DELETE /users/:id** - Now requires auth + self-check

3. **Login Response:** Now returns both user and token
   - **Old:** Returns only user object
   - **New:** Returns `{ user: {...}, token: "..." }`

### Migration Steps

1. Update frontend to store and use JWT tokens
2. Update API calls to include `Authorization: Bearer <token>` header
3. Remove `userId` from snippet creation requests
4. Handle 401/403 errors appropriately
5. Implement token refresh logic

---

## Security Best Practices

1. ✅ **Use HTTPS in production** - Prevents token interception
2. ✅ **Change JWT_SECRET** - Use a long, random secret key
3. ✅ **Validate tokens on every request** - Already implemented
4. ✅ **Check ownership before modifications** - Already implemented
5. ✅ **Use secure password hashing** - Using Argon2id
6. ✅ **Implement token expiration** - 24-hour tokens
7. ⏳ **Consider refresh tokens** - Future enhancement
8. ⏳ **Implement rate limiting** - Future enhancement
9. ⏳ **Add request logging** - Future enhancement

---

## Troubleshooting

### "Authorization header required"

**Problem:** Token not included in request

**Solution:**

```bash
# Add Authorization header
curl -H "Authorization: Bearer YOUR_TOKEN" ...
```

---

### "Invalid or expired token"

**Problem:** Token is invalid, malformed, or expired

**Solutions:**

1. Login again to get a new token
2. Check token format: `Bearer <token>`
3. Verify token hasn't expired (24h default)

---

### "You don't have permission to..."

**Problem:** Trying to modify someone else's resource

**Solution:** You can only modify your own snippets and profile

---

### Token expires too quickly

**Problem:** 24-hour expiration is too short

**Solution:** Modify in `auth.go`:

```go
expirationTime := time.Now().Add(7 * 24 * time.Hour) // 7 days
```

---

## Future Enhancements

- [ ] Refresh token mechanism
- [ ] Role-based access control (admin, user, viewer)
- [ ] OAuth2 integration (Google, GitHub)
- [ ] API rate limiting
- [ ] Token blacklisting for logout
- [ ] Multi-factor authentication (MFA)
- [ ] Session management dashboard

---

For more information, see:

- [API Documentation](./API.md)
- [Development Guide](./DEVELOPMENT.md)
- [Architecture Documentation](./ARCHITECTURE.md)
