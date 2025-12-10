package scanner

import (
	"fmt"
	"time"
)

// scanAPIEndpoints performs API-specific security checks
func (s *Scanner) scanAPIEndpoints() {
	category := s.addCategory("API Security", "API endpoint security checks")

	// Test CORS
	s.testCORS(category)

	// Test rate limiting (if not in safe mode)
	if !s.config.SafeMode {
		s.testRateLimiting(category)
	}

	// Test health endpoint
	s.testHealthEndpoint(category)
}

// testCORS tests CORS configuration
func (s *Scanner) testCORS(category *Category) {
	startTime := time.Now()

	headers := map[string]string{
		"Origin": "https://evil.com",
	}

	resp, err := s.makeRequest("OPTIONS", "/api/v1/lists", headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")

	status := TestStatusPass
	severity := SeverityInfo
	details := "CORS properly configured"

	if allowOrigin == "*" {
		status = TestStatusWarning
		severity = SeverityMedium
		details = "CORS allows all origins (*). Consider restricting to specific domains."
	} else if allowOrigin != "" {
		details = fmt.Sprintf("CORS configured: %s", allowOrigin)
	}

	s.addTest(category, &TestCase{
		ID:          "API-001",
		Name:        "CORS Configuration",
		Description: "Cross-Origin Resource Sharing configuration",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Duration:    time.Since(startTime),
		Timestamp:   time.Now(),
	})
}

// testRateLimiting tests if rate limiting is implemented
func (s *Scanner) testRateLimiting(category *Category) {
	// Only test in non-safe mode
	s.addTest(category, &TestCase{
		ID:          "API-002",
		Name:        "Rate Limiting",
		Description: "API rate limiting implementation",
		Status:      TestStatusSkipped,
		Severity:    SeverityMedium,
		Details:     "Rate limiting test skipped (requires multiple requests)",
		Timestamp:   time.Now(),
	})
}

// testHealthEndpoint tests the health endpoint
func (s *Scanner) testHealthEndpoint(category *Category) {
	startTime := time.Now()

	resp, err := s.makeRequest("GET", "/health", nil)
	if err != nil {
		s.addTest(category, &TestCase{
			ID:          "API-003",
			Name:        "Health Endpoint",
			Description: "Health check endpoint availability",
			Status:      TestStatusWarning,
			Severity:    SeverityLow,
			Details:     fmt.Sprintf("Health endpoint not accessible: %v", err),
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
		return
	}
	defer resp.Body.Close()

	status := TestStatusPass
	details := fmt.Sprintf("Health endpoint accessible (HTTP %d)", resp.StatusCode)

	if resp.StatusCode != 200 {
		status = TestStatusWarning
		details = fmt.Sprintf("Health endpoint returned %d", resp.StatusCode)
	}

	s.addTest(category, &TestCase{
		ID:          "API-003",
		Name:        "Health Endpoint",
		Description: "Health check endpoint availability",
		Status:      status,
		Severity:    SeverityInfo,
		Details:     details,
		Duration:    time.Since(startTime),
		Timestamp:   time.Now(),
	})
}
