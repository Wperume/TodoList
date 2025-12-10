package scanner

import "time"

// ScanConfig holds configuration for the security scanner
type ScanConfig struct {
	Target        string        // Target URL (e.g., https://api.example.com)
	SafeMode      bool          // Enable safe mode (read-only, limited testing)
	MaxRPS        int           // Maximum requests per second
	Timeout       time.Duration // Request timeout
	SkipTLS       bool          // Skip TLS verification (for testing only)
	TestUser      string        // Test user email (optional)
	TestPassword  string        // Test user password (optional)
	Verbose       bool          // Verbose output
}

// ScanResult holds the results of a security scan
type ScanResult struct {
	Target        string                `json:"target"`
	ScanDate      time.Time             `json:"scan_date"`
	Duration      time.Duration         `json:"duration"`
	TotalTests    int                   `json:"total_tests"`
	PassedTests   int                   `json:"passed_tests"`
	FailedTests   int                   `json:"failed_tests"`
	WarningTests  int                   `json:"warning_tests"`
	SecurityScore int                   `json:"security_score"` // 0-100
	Categories    map[string]*Category  `json:"categories"`
}

// Category represents a category of security tests
type Category struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Tests       []*TestCase  `json:"tests"`
	Score       int          `json:"score"` // 0-100
}

// TestCase represents a single security test
type TestCase struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      TestStatus    `json:"status"`
	Severity    Severity      `json:"severity"`
	Details     string        `json:"details"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
}

// TestStatus represents the status of a test
type TestStatus string

const (
	TestStatusPass    TestStatus = "pass"
	TestStatusFail    TestStatus = "fail"
	TestStatusWarning TestStatus = "warning"
	TestStatusSkipped TestStatus = "skipped"
	TestStatusError   TestStatus = "error"
)

// Severity represents the severity of a finding
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)
