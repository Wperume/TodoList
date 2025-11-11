# Security Features

This document describes the security features implemented in the Todo List API and best practices for deployment.

## Current Security Protections

### 1. SQL Injection Protection ✅

**Status**: PROTECTED

The API is protected against SQL injection attacks through:
- **GORM ORM**: All database queries use parameterized statements
- **No raw SQL**: User input is never concatenated into SQL strings
- **Automatic escaping**: GORM handles all parameter escaping

Example:
```go
// Safe - GORM uses prepared statements
db.Where("name = ?", userInput).First(&list)
```

### 2. Cross-Site Scripting (XSS) Protection ✅

**Status**: PROTECTED

- **Input sanitization**: All user input is HTML-escaped before storage
- **XSS prevention headers**: X-XSS-Protection header enabled
- **Content Security Policy**: Strict CSP headers prevent script injection

Example attack prevented:
```json
{
  "name": "<script>alert('xss')</script>"
}
```
Stored as:
```json
{
  "name": "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
}
```

### 3. Denial of Service (DoS) Protection ✅

**Status**: PROTECTED

Multiple layers of DoS protection:
- **Rate limiting**: 60 requests/minute per IP (configurable)
- **Request size limits**: Maximum 1MB request body (configurable)
- **Server-side enforcement**: Cannot be bypassed by clients

### 4. Buffer Overflow Protection ✅

**Status**: PROTECTED

Go language provides built-in protection:
- **Bounds checking**: All array/slice access is bounds-checked
- **Memory safety**: Garbage collection prevents use-after-free bugs
- **Type safety**: No pointer arithmetic or uncontrolled casts

### 5. UUID Validation ✅

**Status**: PROTECTED

- **Format validation**: All UUID parameters are validated before database queries
- **Prevents injection**: Malformed UUIDs rejected before reaching database
- **Clear error messages**: Invalid UUIDs return HTTP 400 with descriptive error

### 6. Security Headers ✅

**Status**: PROTECTED

All responses include security headers:
- `X-Frame-Options: DENY` - Prevents clickjacking
- `X-Content-Type-Options: nosniff` - Prevents MIME sniffing
- `X-XSS-Protection: 1; mode=block` - Enables browser XSS filter
- `Content-Security-Policy` - Restricts resource loading
- `Referrer-Policy: no-referrer` - Prevents referrer leakage
- `Cache-Control: no-store` - Prevents caching of sensitive data

### 7. CORS Protection ✅

**Status**: CONFIGURABLE

Cross-Origin Resource Sharing controls which domains can access the API:
- **Whitelist mode**: Only specified origins allowed
- **Wildcard support**: Can allow all origins for development
- **Preflight handling**: OPTIONS requests properly handled
- **Credential control**: Configurable cookie/auth header support

### 8. Error Sanitization ✅

**Status**: PROTECTED

- **Generic errors**: Internal errors return generic messages to clients
- **Detailed logging**: Full error details logged server-side only
- **No information leakage**: Database errors, stack traces hidden from users

## Current Limitations

### 1. No Authentication ⚠️

**Status**: NOT IMPLEMENTED

**Impact**: Anyone can access all endpoints without credentials

**Mitigation**: Plan to implement JWT authentication (next step)

### 2. No Authorization ⚠️

**Status**: NOT IMPLEMENTED

**Impact**: No user-level access control

**Mitigation**: Will be implemented with authentication

### 3. HTTP Only (No HTTPS) ⚠️

**Status**: NOT ENFORCED

**Impact**: Data transmitted in plaintext over network

**Mitigation**: Should be deployed behind HTTPS proxy (nginx, load balancer)

**Recommendation**: Use Let's Encrypt for free SSL certificates

## Configuration

### Security Settings

```bash
# Request Size Limit
MAX_REQUEST_BODY_SIZE=1048576          # 1MB default

# XSS Protection
ENABLE_XSS_PROTECTION=true             # Enable HTML escaping

# Trusted Proxies (optional)
TRUSTED_PROXIES=192.168.1.1,10.0.0.1   # Comma-separated IPs
```

### CORS Settings

```bash
# Enable/Disable CORS
CORS_ENABLED=true

# Allowed Origins
CORS_ALLOWED_ORIGINS=*                 # Use specific domains in production!

# Allowed Methods
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH

# Allowed Headers
CORS_ALLOWED_HEADERS=Origin,Content-Type,Accept,Authorization,X-API-Key

# Exposed Headers
CORS_EXPOSE_HEADERS=Content-Length,Content-Type

# Allow Credentials
CORS_ALLOW_CREDENTIALS=false           # Set true if using cookies/auth

# Preflight Cache
CORS_MAX_AGE=3600                      # 1 hour
```

### Rate Limiting

```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MIN=60         # Per IP address
```

## Security Best Practices

### For Production Deployment

1. **Use HTTPS**
   ```
   - Deploy behind nginx or load balancer with SSL
   - Use Let's Encrypt for free certificates
   - Redirect HTTP to HTTPS
   ```

2. **Configure CORS Strictly**
   ```bash
   # DO NOT use wildcard in production!
   CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com
   ```

3. **Use Strong Rate Limits**
   ```bash
   RATE_LIMIT_REQUESTS_PER_MIN=30      # Stricter for production
   ```

4. **Enable All Security Features**
   ```bash
   ENABLE_XSS_PROTECTION=true
   MAX_REQUEST_BODY_SIZE=524288        # 512KB for production
   ```

5. **Monitor Logs**
   ```bash
   LOG_LEVEL=warn                      # Less verbose in production
   LOG_JSON_FORMAT=true                # For log aggregation
   ```

6. **Database Security**
   ```bash
   DB_SSLMODE=require                  # Require SSL for database
   DB_LOG_LEVEL=silent                 # Don't log SQL in production
   ```

### For Development

```bash
# Relaxed settings for development
CORS_ALLOWED_ORIGINS=*
RATE_LIMIT_ENABLED=false
LOG_LEVEL=debug
MAX_REQUEST_BODY_SIZE=10485760        # 10MB for testing
```

## Testing Security

### XSS Prevention Test

```bash
# Try to inject script
curl -X POST http://localhost:8080/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{"name":"<script>alert(\"xss\")</script>"}'

# Check that it's escaped in response
```

### Rate Limiting Test

```bash
# Send 70 requests quickly (exceeds 60/min limit)
for i in {1..70}; do
  curl http://localhost:8080/health
done

# Should see 429 errors after 60 requests
```

### Request Size Limit Test

```bash
# Try to send large request
dd if=/dev/zero bs=2M count=1 | \
  curl -X POST http://localhost:8080/api/v1/lists \
  -H "Content-Type: application/json" \
  --data-binary @-

# Should get 413 Request Entity Too Large
```

### UUID Validation Test

```bash
# Try invalid UUID
curl http://localhost:8080/api/v1/lists/not-a-uuid

# Should get 400 Bad Request with INVALID_UUID code
```

## Reporting Security Issues

If you discover a security vulnerability:

1. **DO NOT** open a public GitHub issue
2. Email security concerns to your security team
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

## Security Roadmap

### Planned Security Features

- [x] SQL Injection Protection
- [x] XSS Prevention
- [x] DoS Protection (Rate Limiting)
- [x] Request Size Limits
- [x] Security Headers
- [x] CORS
- [x] Error Sanitization
- [x] UUID Validation
- [ ] JWT Authentication
- [ ] Role-Based Authorization
- [ ] API Key Management
- [ ] Request Signing
- [ ] HTTPS Enforcement
- [ ] Audit Logging
- [ ] Intrusion Detection

## Security Checklist for Deployment

- [ ] HTTPS enabled with valid certificate
- [ ] CORS configured with specific origins (no wildcards)
- [ ] Rate limiting enabled and configured appropriately
- [ ] Database SSL/TLS enabled
- [ ] Strong database passwords
- [ ] Firewall rules configured
- [ ] Logs monitored for suspicious activity
- [ ] Regular security updates applied
- [ ] Backup and disaster recovery plan in place
- [ ] Incident response plan documented

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)
- [CORS Best Practices](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [Rate Limiting Strategies](https://cloud.google.com/architecture/rate-limiting-strategies)
