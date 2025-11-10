# Testing Guide

This document describes the testing strategy and how to run tests for the Todo List API.

## Test Coverage

Current test coverage:
- **Models**: 100% coverage
- **Middleware**: 90.0% coverage
- **Storage Layer**: 80.2% coverage
- **Overall**: Comprehensive unit tests for core functionality

## Test Structure

```
internal/
├── middleware/
│   ├── ratelimit.go
│   └── ratelimit_test.go        # Rate limiting middleware tests
├── models/
│   ├── models.go
│   └── models_test.go           # Model and validation tests
├── storage/
│   ├── storage.go
│   ├── storage_test.go          # In-memory storage tests
│   ├── postgres.go
│   └── postgres_test.go         # PostgreSQL storage tests
└── testutil/
    └── testutil.go              # Test helpers and utilities
```

## Running Tests

### Quick Test Commands

```bash
# Run all unit tests
make test-unit

# Run tests with coverage report
make test-coverage

# Run tests in verbose mode
make test-verbose

# Run integration tests (requires running server)
make test

# Run specific package tests
go test ./internal/models -v
go test ./internal/storage -v
```

### Test Output

```bash
$ make test-unit
Running unit tests...
ok  	todolist-api/internal/models	0.501s	coverage: 100.0%
ok  	todolist-api/internal/storage	0.745s	coverage: 80.2%
```

## Test Categories

### 1. Model Tests (`internal/models/models_test.go`)

Tests for data models and validation:

- **Priority Constants**: Verifies priority enum values
- **GORM Hooks**: Tests UUID generation on create
- **Model Structure**: Tests model field assignments
- **Request/Response DTOs**: Tests request and response structures

**Example:**
```go
func TestTodoListBeforeCreate(t *testing.T) {
    // Tests that UUID is auto-generated if not provided
}
```

### 2. Middleware Tests (`internal/middleware/ratelimit_test.go`)

Tests for rate limiting middleware:

- **Configuration Loading**: Environment variable parsing and defaults
- **Enable/Disable Behavior**: Middleware correctly honors enabled flag
- **Rate Limit Enforcement**: Requests are properly limited when threshold exceeded
- **Error Response Format**: Correct HTTP 429 response with retry information
- **Helper Functions**: getEnv utility function behavior

**Coverage:**
- Configuration loading from environment variables
- Middleware enable/disable toggle
- Rate limit enforcement (with low limits for testing)
- Error response JSON structure validation
- Edge cases (invalid env values, empty values)

### 3. In-Memory Storage Tests (`internal/storage/storage_test.go`)

Tests for the in-memory storage implementation:

- **CRUD Operations**: Create, Read, Update, Delete for lists and todos
- **Pagination**: List pagination with page/limit
- **Filtering**: Filter todos by priority and completion status
- **Sorting**: Sort todos by date, priority, or creation time
- **Validation**: Duplicate name detection, foreign key constraints
- **Error Handling**: NotFound errors, validation errors

**Coverage:**
- All CRUD operations
- Edge cases (empty results, not found, duplicates)
- Concurrent access (thread-safety via mutexes)

### 4. PostgreSQL Storage Tests (`internal/storage/postgres_test.go`)

Tests for the PostgreSQL/GORM storage implementation:

- Same test coverage as in-memory storage
- Uses SQLite in-memory database for fast testing
- Tests GORM-specific features (soft deletes, relationships)
- Validates database constraints and indexes

**Why SQLite for Testing?**
- Fast: In-memory database, no disk I/O
- No setup required: No PostgreSQL instance needed
- Compatible: GORM abstracts most database differences
- CI/CD friendly: Easy to run in automated environments

## Test Utilities (`internal/testutil/`)

### Helper Functions

**Database Setup:**
```go
db := testutil.SetupTestDB(t)           // Create test database
defer testutil.CleanupTestDB(t, db)     // Clean up after test
```

**Test Data Creation:**
```go
list := testutil.CreateTestList("My List", "Description")
todo := testutil.CreateTestTodo(listID, "Task", models.PriorityHigh)
```

**HTTP Testing:**
```go
req := testutil.MakeJSONRequest(t, "POST", "/api/v1/lists", body)
testutil.ParseJSONResponse(t, w, &response)
```

**Pointer Helpers:**
```go
str := testutil.StringPtr("value")
b := testutil.BoolPtr(true)
p := testutil.PriorityPtr(models.PriorityHigh)
```

## Coverage Reports

### Generate Coverage Report

```bash
make test-coverage
```

This generates:
- `coverage.out` - Machine-readable coverage data
- `coverage.html` - HTML coverage report

### View Coverage in Browser

```bash
make test-coverage
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Coverage by Package

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Output:
```
todolist-api/internal/models/models.go:32:	BeforeCreate		100.0%
todolist-api/internal/models/models.go:66:	BeforeCreate		100.0%
todolist-api/internal/storage/storage.go:31:	CreateList		100.0%
...
total:						(statements)		82.5%
```

## Writing New Tests

### Test File Naming

- Test files must end with `_test.go`
- Place in the same package as the code being tested
- Use `package <name>_test` for black-box testing (optional)

### Test Function Naming

```go
func TestFunctionName(t *testing.T) {           // Tests FunctionName
func TestStructName_MethodName(t *testing.T) {  // Tests StructName.MethodName
```

### Table-Driven Tests

```go
func TestPriorityConstants(t *testing.T) {
    tests := []struct {
        name     string
        priority Priority
        expected string
    }{
        {"Low priority", PriorityLow, "low"},
        {"Medium priority", PriorityMedium, "medium"},
        {"High priority", PriorityHigh, "high"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.expected, string(tt.priority))
        })
    }
}
```

### Using Subtests

```go
func TestCreateList(t *testing.T) {
    t.Run("successfully creates a list", func(t *testing.T) {
        // Test successful creation
    })

    t.Run("fails when list name already exists", func(t *testing.T) {
        // Test duplicate name error
    })
}
```

### Assertions

We use `testify/assert` and `testify/require`:

```go
// assert - test continues on failure
assert.Equal(t, expected, actual)
assert.NotNil(t, value)
assert.True(t, condition)

// require - test stops on failure
require.NoError(t, err)
require.NotNil(t, value)
```

## Best Practices

### 1. Test Independence
- Each test should be independent
- Use `t.Run()` for subtests
- Clean up resources with `defer`

### 2. Use Test Helpers
- Create helper functions for common setup
- Use `testutil` package for shared utilities
- Keep tests readable and maintainable

### 3. Test Edge Cases
- Empty inputs
- Nil pointers
- Not found scenarios
- Duplicate data
- Boundary conditions

### 4. Fast Tests
- Use in-memory databases
- Mock external dependencies
- Run unit tests before integration tests
- Use `-short` flag for quick feedback

### 5. Clear Error Messages
```go
assert.Equal(t, expected, actual, "Failed to create list")
require.NoError(t, err, "Database connection failed")
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - run: go test ./... -v -coverprofile=coverage.out
      - run: go tool cover -func=coverage.out
```

## Test Data

### Sample Test List
```go
list := &models.TodoList{
    Name:        "Work Tasks",
    Description: "Tasks for work projects",
}
```

### Sample Test Todo
```go
todo := &models.Todo{
    Description: "Complete documentation",
    Priority:    models.PriorityHigh,
    DueDate:     &dueDate,
    Completed:   false,
}
```

## Debugging Tests

### Run Single Test
```bash
go test ./internal/models -run TestPriorityConstants -v
```

### Enable Verbose SQL Logging
```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),  // Show SQL
})
```

### Print Debug Info
```go
t.Logf("List ID: %s", list.ID)
t.Logf("Todo count: %d", len(todos))
```

## Future Testing

### Planned Additions
- [ ] Handler/HTTP endpoint tests
- [ ] Integration tests with real PostgreSQL
- [ ] Load/performance tests
- [ ] Authentication/authorization tests
- [ ] API contract tests (OpenAPI validation)

### Handler Testing Example
```go
func TestListHandler_GetAllLists(t *testing.T) {
    store := testutil.SetupTestDB(t)
    handler := handlers.NewListHandler(store)

    req := testutil.MakeJSONRequest(t, "GET", "/api/v1/lists", nil)
    w := httptest.NewRecorder()

    handler.GetAllLists(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [GORM Testing](https://gorm.io/docs/testing.html)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)

## Summary

- ✅ **100% model coverage** - All data structures tested
- ✅ **80%+ storage coverage** - Core business logic tested
- ✅ **Fast tests** - Sub-second execution with SQLite
- ✅ **Easy to run** - Simple make commands
- ✅ **Maintainable** - Clear structure and helpers
- ✅ **CI-ready** - No external dependencies required

Run `make test-unit` before committing code!
