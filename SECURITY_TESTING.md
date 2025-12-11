# Security Testing Guide

This document describes the security testing framework for the TodoList API.

## Overview

The security testing framework consists of two complementary approaches:

1. **Integration Security Tests** - Run automatically as part of the test suite
2. **Standalone Security Scanner** - On-demand tool for scanning live instances

## Integration Security Tests

Integration tests run against a local test server and validate security controls.

### Running Security Tests

```bash
# Run all security tests
make test-security

# Run security tests in verbose mode
make test-security-verbose

# Run all tests (unit + security)
make test-all
```

### Test Categories

#### 1. Injection Tests (`internal/security/injection_test.go`)
- SQL injection in login endpoint
- SQL injection in todo descriptions
- Command injection attempts
- NoSQL injection in UUIDs

#### 2. Authentication Tests (`internal/security/auth_test.go`)
- Missing authentication token
- Invalid authentication tokens
- Expired JWT tokens
- JWT algorithm substitution attacks
- JWT signature verification
- Password complexity requirements

#### 3. Authorization Tests (`internal/security/authorization_test.go`)
- Access control for other users' lists
- Access control for other users' todos
- Insecure Direct Object Reference (IDOR) vulnerabilities
- Mass assignment vulnerabilities
- Privilege escalation attempts

#### 4. Input Validation Tests (`internal/security/validation_test.go`)
- XSS payload handling
- Oversized input handling
- Invalid UUID validation
- Malformed JSON handling
- Special character handling
- Content-Type validation
- Email validation

### Test Output

Tests use the standard Go testing framework and output results to stdout:

```bash
=== RUN   TestSQLInjectionInLogin
=== RUN   TestSQLInjectionInLogin/' OR '1'='1
--- PASS: TestSQLInjectionInLogin (0.02s)
    --- PASS: TestSQLInjectionInLogin/' OR '1'='1 (0.00s)
```

## Standalone Security Scanner

The security scanner is a CLI tool that performs security checks against live API instances.

### Building the Scanner

```bash
# Build the scanner
make build-scanner

# Scanner binary will be at: bin/security-scanner
```

### Running the Scanner

#### Scan Local Instance

```bash
# Scan local development instance
make scan-security

# Or explicitly:
make scan-security TARGET=https://localhost:8443
```

#### Scan Remote Instance

```bash
# Scan staging environment
./bin/security-scanner --target https://staging.example.com --output staging-report.html

# Scan production (safe mode, limited RPS)
./bin/security-scanner \
  --target https://api.example.com \
  --safe-mode \
  --max-rps 1 \
  --output prod-scan.html
```

### Scanner Options

```
--target string          Target URL (required)
--safe-mode             Enable safe mode (read-only testing) (default: true)
                        Set to false for full testing: --safe-mode=false
--max-rps int           Maximum requests per second (default: 5)
--timeout int           Request timeout in seconds (default: 10)
--skip-tls              Skip TLS certificate verification (for testing only)
--output string         Output file path (default: "security-report.html")
--format string         Output format: html or json (default: "html")
--verbose               Verbose output
--test-user string      Test user email (optional)
--test-password string  Test user password (optional)
```

### Safe Mode vs Full Testing

**Safe Mode (default: --safe-mode or --safe-mode=true)**
- ✅ Production-safe, read-only checks
- ✅ TLS/SSL validation
- ✅ Security headers checks
- ✅ CORS configuration
- ✅ Health endpoint checks
- ❌ Skips rate limiting tests (avoids many rapid requests)
- ❌ Skips aggressive authentication tests

**Full Testing Mode (--safe-mode=false)**
- ✅ All safe mode checks
- ✅ Rate limiting tests (sends multiple rapid requests)
- ✅ Comprehensive authentication tests
- ⚠️ More aggressive - use on dev/staging only

**Usage Examples:**
```bash
# Production - always use safe mode
./bin/security-scanner --target https://api.production.com --safe-mode

# Staging - can use full testing
./bin/security-scanner --target https://staging.example.com --safe-mode=false

# Local dev - full testing with verbose output
./bin/security-scanner --target https://localhost:8443 --skip-tls --safe-mode=false --verbose
```

### Scanner Test Categories

#### 1. TLS/SSL Security
- HTTPS enabled
- TLS version (1.2, 1.3)
- Certificate validity and expiration
- Certificate chain validation

#### 2. Security Headers
- Strict-Transport-Security (HSTS)
- Content-Security-Policy (CSP)
- X-Frame-Options
- X-Content-Type-Options
- X-XSS-Protection
- Referrer-Policy
- Server header information disclosure

#### 3. Authentication Security
- Protected endpoint authentication requirements
- Invalid token rejection
- Login endpoint availability

#### 4. API Security
- CORS configuration
- Rate limiting implementation
- Health endpoint availability

### Understanding Scan Results

The scanner generates an HTML report with:

- **Security Score**: 0-100 based on test results
- **Category Scores**: Individual scores for each category
- **Test Results**: Pass/Fail/Warning for each test
- **Severity Levels**: Critical, High, Medium, Low, Info
- **Details**: Specific findings and recommendations

#### Score Interpretation

- **80-100**: Excellent security posture
- **60-79**: Good, but some improvements needed
- **40-59**: Fair, multiple issues to address
- **0-39**: Poor, critical issues present

### Example Workflow

1. **Development**: Run integration tests during development
   ```bash
   make test-security
   ```

2. **Pre-Deployment**: Scan staging environment
   ```bash
   ./bin/security-scanner --target https://staging.example.com
   ```

3. **Post-Deployment**: Scan production (carefully)
   ```bash
   ./bin/security-scanner \
     --target https://api.example.com \
     --safe-mode \
     --max-rps 1
   ```

4. **Regular Audits**: Schedule weekly/monthly scans
   ```bash
   # Add to cron or CI/CD pipeline
   ./bin/security-scanner --target $API_URL --format json --output scan-$(date +%Y%m%d).json
   ```

## Safety Considerations

### Integration Tests
- ✅ Safe to run anytime - uses isolated test database
- ✅ Runs in CI/CD pipeline
- ✅ No impact on production

### Scanner - Local/Staging
- ✅ Safe to run against local instances
- ✅ Safe to run against staging
- ⚠️  May generate test data

### Scanner - Production
- ⚠️  Use `--safe-mode` (enabled by default)
- ⚠️  Limit RPS with `--max-rps 1`
- ⚠️  Run during low-traffic periods
- ⚠️  May trigger security monitoring alerts
- ⚠️  Coordinate with ops team

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Security Tests

on: [push, pull_request]

jobs:
  security-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run security tests
        run: make test-security

      - name: Build scanner
        run: make build-scanner

      - name: Scan staging
        if: github.ref == 'refs/heads/main'
        run: |
          ./bin/security-scanner \
            --target ${{ secrets.STAGING_URL }} \
            --output security-report.html

      - name: Upload report
        uses: actions/upload-artifact@v3
        with:
          name: security-report
          path: security-report.html
```

## Interpreting Results

### Critical Findings
- **Authentication bypass**: Immediate fix required
- **SQL injection vulnerabilities**: Critical security risk
- **Missing HTTPS**: All traffic is unencrypted
- **Expired certificates**: Service may be inaccessible

### High Priority
- **Weak TLS configuration**: Upgrade to TLS 1.2+
- **Missing security headers**: Add HSTS, CSP, etc.
- **Authorization failures**: Users accessing others' data

### Medium Priority
- **Information disclosure**: Server header exposes version
- **Weak CORS policy**: Overly permissive origins
- **Missing rate limiting**: Vulnerable to DoS

### Low Priority
- **Legacy security headers**: X-XSS-Protection recommendations
- **Certificate expiring soon**: Plan renewal

## Best Practices

1. **Run integration tests on every commit**
   ```bash
   make test-security
   ```

2. **Scan staging before deploying to production**
   ```bash
   make scan-security TARGET=https://staging.example.com
   ```

3. **Schedule regular production scans**
   - Weekly or monthly
   - Use safe mode and low RPS
   - Review reports promptly

4. **Track security score over time**
   - Save JSON reports
   - Monitor for regressions
   - Set minimum score thresholds

5. **Address critical findings immediately**
   - Authentication/authorization issues
   - Injection vulnerabilities
   - TLS/certificate problems

## Troubleshooting

### Tests Failing Locally

```bash
# Ensure test database is available
docker-compose up -d postgres

# Run with verbose output
make test-security-verbose
```

### Scanner Cannot Connect

```bash
# Check if target is reachable
curl -k https://localhost:8443/health

# Use --skip-tls for self-signed certificates
./bin/security-scanner --target https://localhost:8443 --skip-tls

# Increase timeout for slow connections
./bin/security-scanner --target https://api.example.com --timeout 30
```

### Rate Limiting Issues

```bash
# Reduce request rate
./bin/security-scanner --target https://api.example.com --max-rps 1

# Use safe mode (fewer tests)
./bin/security-scanner --target https://api.example.com --safe-mode
```

## Contributing

To add new security tests:

1. **Integration Tests**: Add to `internal/security/*_test.go`
2. **Scanner Tests**: Add to `cmd/security-scanner/scanner/*.go`
3. **Update Documentation**: Document new tests here
4. **Run Tests**: Verify with `make test-security`

## Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [JWT Security Best Practices](https://tools.ietf.org/html/rfc8725)
- [TLS Best Practices](https://wiki.mozilla.org/Security/Server_Side_TLS)
