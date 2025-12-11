# Security Tests - Implementation Note

## Current Status

The security test files in this directory are **templates** that demonstrate comprehensive security testing patterns. However, they currently require additional setup to run properly.

## Why Tests Are Skipped

The security tests require:
1. A fully configured test router with all handlers
2. Proper JWT token generation and validation
3. Complete API endpoint setup

The `testutil.SetupTestRouter()` function currently returns a minimal router and needs to be implemented with full handler initialization.

## Running Security Tests

Currently, running `make test-security` will skip most tests with the message:
```
Skipping security test in short mode
```

This is intentional until the router setup is completed.

## What the Tests Cover

Even though skipped, the test files serve as:

### 1. **injection_test.go**
- SQL injection attack vectors
- Command injection patterns
- NoSQL injection attempts
- Demonstrates what should be tested

### 2. **auth_test.go**
- JWT token manipulation
- Expired token handling
- Algorithm substitution attacks
- Password complexity requirements

### 3. **authorization_test.go**
- Insecure Direct Object Reference (IDOR)
- Privilege escalation attempts
- Access control verification
- Mass assignment vulnerabilities

### 4. **validation_test.go**
- XSS payload handling
- Oversized input validation
- Special character processing
- Content-type validation

## Implementing Full Tests

To make these tests functional, you need to:

1. **Implement `testutil.SetupTestRouter()`**
   ```go
   func SetupTestRouter(t *testing.T) (*gin.Engine, func()) {
       db := SetupTestDB(t)

       // Initialize all handlers with the test DB
       authHandler := handlers.NewAuthHandler(db, jwtConfig)
       listHandler := handlers.NewListHandler(db)
       todoHandler := handlers.NewTodoHandler(db)

       // Setup router with all middleware and routes
       router := gin.New()
       router.Use(middleware.ErrorHandler())
       // ... add all your routes

       cleanup := func() {
           CleanupTestDB(t, db)
       }

       return router, cleanup
   }
   ```

2. **Implement `testutil.CreateTestUserWithToken()`**
   ```go
   func CreateTestUserWithToken(t *testing.T, router *gin.Engine) (*models.User, string) {
       // Register user via API
       // Generate real JWT token
       // Return user and token
   }
   ```

3. **Remove the `testing.Short()` skip conditions** in each test

## Alternative: Standalone Scanner

Until the integration tests are fully functional, use the **standalone security scanner**:

```bash
# Build the scanner
make build-scanner

# Scan your running instance
make scan-security
```

The scanner provides real security testing against live instances and doesn't require test setup.

## Benefits of These Templates

Even as templates, these test files:
- ✅ Document security test cases that should be covered
- ✅ Provide attack vectors to test against
- ✅ Serve as a checklist for manual security testing
- ✅ Can be enabled incrementally as router setup is completed
- ✅ Follow Go testing best practices

## Quick Win: Enable Tests Incrementally

You can enable tests one at a time by:

1. Implementing router setup for one endpoint
2. Removing `testing.Short()` check for related tests
3. Running `go test ./internal/security/injection_test.go -v`
4. Repeat for each category

## Using Tests as Documentation

Even without running them, you can use these tests to:
- Understand what attack vectors to defend against
- Review security requirements during code review
- Plan security improvements
- Document security expectations

## Next Steps

Choose one of these paths:

**Path 1: Full Integration Tests (Recommended for CI/CD)**
- Implement `testutil.SetupTestRouter()` with full handlers
- Remove `testing.Short()` skip conditions
- Run `make test-security` as part of CI/CD

**Path 2: Use Standalone Scanner (Recommended for Now)**
- Run scanner against dev/staging instances
- Schedule regular production scans
- Review HTML reports

**Path 3: Hybrid Approach (Best)**
- Use scanner for immediate security validation
- Gradually implement integration tests
- End up with both automated and on-demand testing
