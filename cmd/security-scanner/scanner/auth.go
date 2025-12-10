package scanner

import (
	"fmt"
	"time"
)

// scanAuthentication performs authentication-related security checks
func (s *Scanner) scanAuthentication() {
	category := s.addCategory("Authentication", "Authentication and authorization checks")

	// Test endpoints without authentication
	s.testUnauthenticatedAccess(category)

	// Test with invalid tokens
	s.testInvalidToken(category)

	// If test credentials provided, test login
	if s.config.TestUser != "" && s.config.TestPassword != "" {
		s.testLoginEndpoint(category)
	}
}

// testUnauthenticatedAccess tests if protected endpoints require auth
func (s *Scanner) testUnauthenticatedAccess(category *Category) {
	startTime := time.Now()

	resp, err := s.makeRequest("GET", "/api/v1/lists", nil)
	if err != nil {
		s.addTest(category, &TestCase{
			ID:          "AUTH-001",
			Name:        "API Reachability",
			Description: "Check if API is reachable",
			Status:      TestStatusError,
			Severity:    SeverityHigh,
			Details:     fmt.Sprintf("Failed to reach API: %v", err),
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
		return
	}
	defer resp.Body.Close()

	status := TestStatusPass
	severity := SeverityInfo
	details := fmt.Sprintf("Protected endpoint requires authentication (HTTP %d)", resp.StatusCode)

	if resp.StatusCode == 200 {
		status = TestStatusFail
		severity = SeverityCritical
		details = "Protected endpoint accessible without authentication"
	} else if resp.StatusCode != 401 {
		status = TestStatusWarning
		severity = SeverityMedium
		details = fmt.Sprintf("Unexpected status code: %d (expected 401)", resp.StatusCode)
	}

	s.addTest(category, &TestCase{
		ID:          "AUTH-001",
		Name:        "Authentication Required",
		Description: "Check if protected endpoints require authentication",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Duration:    time.Since(startTime),
		Timestamp:   time.Now(),
	})
}

// testInvalidToken tests if invalid tokens are rejected
func (s *Scanner) testInvalidToken(category *Category) {
	startTime := time.Now()

	headers := map[string]string{
		"Authorization": "Bearer invalid_token_12345",
	}

	resp, err := s.makeRequest("GET", "/api/v1/lists", headers)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	status := TestStatusPass
	severity := SeverityInfo
	details := "Invalid tokens are rejected"

	if resp.StatusCode == 200 {
		status = TestStatusFail
		severity = SeverityCritical
		details = "Invalid token accepted (critical security issue)"
	} else if resp.StatusCode != 401 {
		status = TestStatusWarning
		severity = SeverityLow
		details = fmt.Sprintf("Unexpected status code: %d", resp.StatusCode)
	}

	s.addTest(category, &TestCase{
		ID:          "AUTH-002",
		Name:        "Invalid Token Rejection",
		Description: "Check if invalid tokens are properly rejected",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Duration:    time.Since(startTime),
		Timestamp:   time.Now(),
	})
}

// testLoginEndpoint tests the login endpoint
func (s *Scanner) testLoginEndpoint(category *Category) {
	// Only in safe mode and if credentials provided
	if !s.config.SafeMode {
		return
	}

	startTime := time.Now()

	resp, err := s.makeRequest("POST", "/api/v1/auth/login", nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	status := TestStatusPass
	details := fmt.Sprintf("Login endpoint reachable (HTTP %d)", resp.StatusCode)

	s.addTest(category, &TestCase{
		ID:          "AUTH-003",
		Name:        "Login Endpoint",
		Description: "Check login endpoint availability",
		Status:      status,
		Severity:    SeverityInfo,
		Details:     details,
		Duration:    time.Since(startTime),
		Timestamp:   time.Now(),
	})
}
