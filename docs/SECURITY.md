# Security Documentation

## 🔒 Security Audit & Implementation

This document outlines all security measures implemented in Snippy Backend and recommendations for production deployment.

---

## Security Checklist

### ✅ Implemented Security Measures

| Category             | Measure                       | Status         | Details                                  |
| -------------------- | ----------------------------- | -------------- | ---------------------------------------- |
| **Authentication**   | Argon2id password hashing     | ✅ Implemented | 64MB memory, 4 threads, 1 iteration      |
| **Authentication**   | JWT tokens with expiration    | ✅ Implemented | 24-hour expiration                       |
| **Authentication**   | JWT algorithm validation      | ✅ Implemented | Prevents algorithm substitution attacks  |
| **Authentication**   | Rate limiting on login        | ✅ Implemented | 5 requests/minute per IP                 |
| **Authentication**   | Rate limiting on registration | ✅ Implemented | 5 requests/minute per IP                 |
| **Authorization**    | User ownership validation     | ✅ Implemented | Users can only modify own data           |
| **Authorization**    | JWT-based access control      | ✅ Implemented | Protected endpoints require valid tokens |
| **Input Validation** | Request body validation       | ✅ Implemented | Type checking, required fields           |
| **Input Validation** | String length limits          | ✅ Implemented | Prevents memory exhaustion               |
| **Input Validation** | Email format validation       | ✅ Implemented | RFC 5322 compliant                       |
| **Input Validation** | URL format validation         | ✅ Implemented | For avatar URLs                          |
| **Input Validation** | Alphanumeric username         | ✅ Implemented | Prevents special character exploits      |
| **Rate Limiting**    | Global API rate limit         | ✅ Implemented | 100 requests/minute per IP               |
| **Rate Limiting**    | Strict auth rate limit        | ✅ Implemented | 5 requests/minute for login/register     |
| **Database**         | Parameterized queries         | ✅ Implemented | Prevents SQL injection                   |
| **Database**         | User scoping in queries       | ✅ Implemented | Returns only authorized data             |
| **Database**         | Connection pooling            | ✅ Implemented | Prevents connection exhaustion           |
| **Configuration**    | Secrets in environment        | ✅ Implemented | No secrets in code                       |
| **Configuration**    | Configurable CORS             | ✅ Implemented | Production-ready CORS control            |
| **Errors**           | Generic error messages        | ✅ Implemented | No internal details leaked               |
| **Errors**           | No password/token logging     | ✅ Implemented | Sensitive data not logged                |
| **DoS Protection**   | Request body size limit       | ✅ Implemented | 10MB maximum                             |
| **DoS Protection**   | Rate limiting cleanup         | ✅ Implemented | Auto-cleanup of old rate limiters        |

### ⚠️ Production Recommendations

| Category          | Recommendation       | Priority    | Details                                   |
| ----------------- | -------------------- | ----------- | ----------------------------------------- |
| **Transport**     | HTTPS only           | 🔴 Critical | Never use HTTP in production              |
| **Transport**     | TLS 1.2+             | 🔴 Critical | Disable older SSL/TLS versions            |
| **Configuration** | Strong JWT secret    | 🔴 Critical | Use 32+ character random string           |
| **Configuration** | Restrict CORS        | 🔴 Critical | Set specific allowed origins              |
| **Configuration** | SSL mode for DB      | 🟡 High     | Set `sslmode=require`                     |
| **Monitoring**    | Request logging      | 🟡 High     | Log all API requests (not sensitive data) |
| **Monitoring**    | Error tracking       | 🟡 High     | Use Sentry or similar                     |
| **Monitoring**    | Performance metrics  | 🟢 Medium   | Use Prometheus or similar                 |
| **Hardening**     | Helmet-style headers | 🟡 High     | Add security headers                      |
| **Hardening**     | CSRF protection      | 🟢 Medium   | For web-based clients                     |

---

## Detailed Security Measures

### 1. Password Hashing (Argon2id)

**Implementation:**

```go
// Argon2 parameters
const (
    argon2Time    = 1
    argon2Memory  = 64 * 1024 // 64 MB
    argon2Threads = 4
    argon2KeyLen  = 32
    argon2SaltLen = 16
)
```

**Why Argon2id?**

- Winner of Password Hashing Competition (PHC)
- Memory-hard algorithm (resists GPU/ASIC attacks)
- Configurable time, memory, and parallelism
- Better than bcrypt, scrypt, or PBKDF2

**Security Level:**

- ✅ Excellent - Industry standard
- ✅ Resistant to brute force
- ✅ Resistant to rainbow tables
- ✅ Resistant to timing attacks (constant-time comparison)

---

### 2. JWT Token Security

**Implementation:**

```go
// Token expiration: 24 hours
expirationTime := time.Now().Add(24 * time.Hour)

// Algorithm validation (prevents algorithm substitution)
if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
}
```

**Protection Against:**

- ✅ Algorithm substitution attacks (e.g., "none" algorithm)
- ✅ Token reuse after expiration
- ✅ Token tampering (HMAC signature)

**Token Contents:**

```json
{
  "user_id": "uuid",
  "username": "john",
  "email": "john@example.com",
  "exp": 1731262200,
  "iat": 1731175800
}
```

---

### 3. Rate Limiting

**General API Rate Limit:**

- **Limit:** 100 requests per minute per IP
- **Burst:** 100 requests
- **Cleanup:** Auto-cleanup after 3 minutes of inactivity

**Strict Rate Limit (Auth endpoints):**

- **Limit:** 5 requests per minute per IP
- **Burst:** 5 requests
- **Endpoints:** Login, Registration

**Implementation:**

```go
// Rate limiter with per-IP tracking
type RateLimiter struct {
    visitors map[string]*visitor
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}
```

**Protection Against:**

- ✅ Brute force login attacks
- ✅ Account enumeration
- ✅ DDoS attacks
- ✅ Spam registrations

---

### 4. Input Validation

**String Length Limits:**

```go
Title:       max 255 characters
Description: max 5,000 characters
Category:    max 100 characters
Shortcut:    max 50 characters
Content:     max 100,000 characters (100KB)
Tags:        max 20 tags, each max 50 characters
Username:    min 3, max 50 characters, alphanumeric only
Email:       max 255 characters, valid email format
Password:    min 8, max 128 characters
```

**Validation Rules:**

```go
binding:"required,min=3,max=50,alphanum"  // Username
binding:"required,email,max=255"          // Email
binding:"required,min=8,max=128"          // Password
binding:"max=100000"                      // Content (100KB)
binding:"max=20,dive,max=50"              // Tags
```

**Protection Against:**

- ✅ Memory exhaustion DoS
- ✅ Database overflow
- ✅ Special character exploits
- ✅ SQL injection (via parameterized queries)

---

### 5. CORS Configuration

**Development:**

```bash
CORS_ALLOWED_ORIGINS=*
```

**Production:**

```bash
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://extension-id.chromiumapp.org
```

**Implementation:**

```go
allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
if allowedOrigins == "" {
    allowedOrigins = "*"
    log.Println("WARNING: CORS set to allow all origins (*)")
}
```

**Security:**

- ✅ Configurable per environment
- ✅ Warnings for insecure defaults
- ✅ Supports multiple origins

---

### 6. Request Body Size Limiting

**Implementation:**

```go
router.MaxMultipartMemory = 10 << 20 // 10 MB
```

**Protection Against:**

- ✅ Memory exhaustion attacks
- ✅ Large file upload DoS
- ✅ Bandwidth exhaustion

---

### 7. Database Security

**Parameterized Queries:**

```go
query := "SELECT * FROM snippets WHERE user_id = $1"
db.Query(query, userID)  // ✅ Safe from SQL injection
```

**User Scoping:**

```go
// Check ownership before modification
var snippetUserID sql.NullString
checkQuery := `SELECT user_id FROM snippets WHERE id = $1`
err = db.QueryRow(checkQuery, id).Scan(&snippetUserID)

if snippetUserID.String != authUserID {
    return 403 Forbidden
}
```

**Connection Security:**

- ✅ Connection pooling (prevents exhaustion)
- ✅ Idle timeout (5 minutes)
- ✅ Max connections (25)
- ✅ SSL mode configurable

---

### 8. Error Handling

**Generic Messages:**

```go
// ✅ Good - Doesn't reveal which field is wrong
c.JSON(401, gin.H{"error": "Invalid email or password"})

// ❌ Bad - Reveals user exists
c.JSON(401, gin.H{"error": "Invalid password"})
```

**No Sensitive Data Logging:**

```go
// ✅ Safe logging
log.Println("Login attempt for user:", email)

// ❌ Unsafe - logs sensitive data
log.Println("Password:", password)  // NEVER DO THIS
```

**No Stack Traces in Production:**

```go
// ✅ Production mode
gin.SetMode(gin.ReleaseMode)

// ❌ Development mode in production
gin.SetMode(gin.DebugMode)  // Don't use in production
```

---

## Production Deployment Checklist

### Critical (Must Do)

- [ ] **Generate strong JWT secret** (32+ characters)

  ```bash
  openssl rand -base64 64 > jwt_secret.txt
  ```

- [ ] **Set specific CORS origins**

  ```bash
  CORS_ALLOWED_ORIGINS=https://your-production-domain.com
  ```

- [ ] **Enable HTTPS** (use reverse proxy like Nginx/Caddy)

  ```nginx
  server {
      listen 443 ssl http2;
      ssl_certificate /path/to/cert.pem;
      ssl_certificate_key /path/to/key.pem;
  }
  ```

- [ ] **Enable PostgreSQL SSL**

  ```bash
  DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require
  ```

- [ ] **Set Gin to release mode**

  ```bash
  GIN_MODE=release
  ```

- [ ] **Use strong database password**

- [ ] **Restrict database access** (firewall rules)

### High Priority (Should Do)

- [ ] **Add security headers** (middleware)

  ```go
  c.Header("X-Frame-Options", "DENY")
  c.Header("X-Content-Type-Options", "nosniff")
  c.Header("X-XSS-Protection", "1; mode=block")
  c.Header("Strict-Transport-Security", "max-age=31536000")
  ```

- [ ] **Set up request logging** (without sensitive data)

- [ ] **Configure error tracking** (Sentry, Rollbar, etc.)

- [ ] **Set up monitoring** (Prometheus, Grafana)

- [ ] **Regular security audits**

- [ ] **Dependency vulnerability scanning**
  ```bash
  go install golang.org/x/vuln/cmd/govulncheck@latest
  govulncheck ./...
  ```

### Medium Priority (Nice to Have)

- [ ] **Implement refresh tokens** (for better UX)

- [ ] **Add CSRF protection** (for web clients)

- [ ] **Implement account lockout** (after X failed login attempts)

- [ ] **Add email verification** (for new accounts)

- [ ] **Implement password reset** (via email)

- [ ] **Add 2FA support** (TOTP)

- [ ] **Set up automated backups**

- [ ] **Implement audit logging** (who did what when)

---

## Security Testing

### Manual Testing

**Test Rate Limiting:**

```bash
# Should be blocked after 5 attempts
for i in {1..10}; do
    curl -X POST http://localhost:8080/api/v1/auth/login \
      -H "Content-Type: application/json" \
      -d '{"email":"test@example.com","password":"wrong"}'
    sleep 1
done
```

**Test JWT Expiration:**

```bash
# Save token, wait 24+ hours, try to use it
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login ... | jq -r '.token')
# ... 24 hours later ...
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/users
# Should return 401 Unauthorized
```

**Test Ownership Protection:**

```bash
# User A creates snippet
TOKEN_A="..."
SNIPPET_ID=$(curl -X POST ... -H "Authorization: Bearer $TOKEN_A" ...)

# User B tries to modify it (should fail)
TOKEN_B="..."
curl -X PUT http://localhost:8080/api/v1/snippets/$SNIPPET_ID \
  -H "Authorization: Bearer $TOKEN_B" \
  -d '{"title":"Hacked"}'
# Should return 403 Forbidden
```

### Automated Testing

**Vulnerability Scanning:**

```bash
# Check for known vulnerabilities
govulncheck ./...

# Dependency audit
go list -json -m all | nancy sleuth
```

**Static Analysis:**

```bash
# Security-focused linting
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...
```

---

## Incident Response

### If JWT Secret is Compromised

1. **Immediately rotate JWT secret**

   ```bash
   JWT_SECRET=$(openssl rand -base64 64)
   ```

2. **All users must re-login** (old tokens invalid)

3. **Investigate how secret was leaked**

4. **Review access logs**

### If Database is Compromised

1. **Immediately change database password**

2. **Rotate all user passwords** (force password reset)

3. **Audit all database access**

4. **Review application logs**

5. **Notify affected users** (if data was accessed)

### If Rate Limiting is Bypassed

1. **Check for distributed attacks** (multiple IPs)

2. **Consider IP-based blocking** (firewall level)

3. **Implement additional rate limiting** (per account)

4. **Review rate limiter implementation**

---

## Security Contacts

For security vulnerabilities, please report to:

- **Email:** security@jheysonsaavedra.com
- **GitHub Security Advisories:** [Report a vulnerability](https://github.com/jheysaaz/snippy-backend/security/advisories/new)

**DO NOT** report security vulnerabilities via public GitHub issues.

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [Go Security Checklist](https://go.dev/security/)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
- [Argon2 Specification](https://github.com/P-H-C/phc-winner-argon2)

---

## Change Log

| Date       | Change                         | Reason                                 |
| ---------- | ------------------------------ | -------------------------------------- |
| 2025-11-10 | Added JWT algorithm validation | Prevent algorithm substitution attacks |
| 2025-11-10 | Implemented rate limiting      | Prevent brute force and DoS            |
| 2025-11-10 | Added input validation limits  | Prevent memory exhaustion              |
| 2025-11-10 | Made CORS configurable         | Production security                    |
| 2025-11-10 | Added request size limits      | Prevent DoS attacks                    |

---

**Last Updated:** 2025-11-10  
**Security Review Date:** 2025-11-10  
**Next Review Due:** 2026-02-10 (quarterly)
