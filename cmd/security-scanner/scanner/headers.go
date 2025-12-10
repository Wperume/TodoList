package scanner

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// scanHeaders performs security header checks
func (s *Scanner) scanHeaders() {
	category := s.addCategory("Security Headers", "HTTP security headers configuration")

	resp, err := s.makeRequest("GET", "/health", nil)
	if err != nil {
		s.addTest(category, &TestCase{
			ID:          "HDR-001",
			Name:        "Health Endpoint Reachable",
			Description: "Check if health endpoint is reachable",
			Status:      TestStatusError,
			Severity:    SeverityCritical,
			Details:     fmt.Sprintf("Failed to reach health endpoint: %v", err),
			Timestamp:   time.Now(),
		})
		return
	}
	defer resp.Body.Close()

	// Check security headers
	s.checkHSTSHeader(category, resp)
	s.checkContentSecurityPolicy(category, resp)
	s.checkXFrameOptions(category, resp)
	s.checkXContentTypeOptions(category, resp)
	s.checkXXSSProtection(category, resp)
	s.checkReferrerPolicy(category, resp)
	s.checkServerHeader(category, resp)
}

// checkHSTSHeader checks for HSTS header
func (s *Scanner) checkHSTSHeader(category *Category, resp *http.Response) {
	hsts := resp.Header.Get("Strict-Transport-Security")

	status := TestStatusPass
	severity := SeverityInfo
	details := "HSTS header present"

	if hsts == "" {
		status = TestStatusFail
		severity = SeverityHigh
		details = "HSTS header missing. Add 'Strict-Transport-Security' header"
	} else {
		// Check max-age
		if strings.Contains(hsts, "max-age=") {
			details = fmt.Sprintf("HSTS configured: %s", hsts)
			// Check if includeSubDomains is set
			if !strings.Contains(hsts, "includeSubDomains") {
				status = TestStatusWarning
				severity = SeverityLow
				details += " (consider adding 'includeSubDomains')"
			}
		} else {
			status = TestStatusWarning
			severity = SeverityMedium
			details = "HSTS header present but max-age not set"
		}
	}

	s.addTest(category, &TestCase{
		ID:          "HDR-002",
		Name:        "HSTS Header",
		Description: "HTTP Strict Transport Security header",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Timestamp:   time.Now(),
	})
}

// checkContentSecurityPolicy checks CSP header
func (s *Scanner) checkContentSecurityPolicy(category *Category, resp *http.Response) {
	csp := resp.Header.Get("Content-Security-Policy")

	status := TestStatusPass
	severity := SeverityInfo
	details := "CSP header present"

	if csp == "" {
		status = TestStatusWarning
		severity = SeverityMedium
		details = "Content-Security-Policy header missing"
	} else {
		details = "CSP configured"
		// Check for unsafe directives
		if strings.Contains(csp, "'unsafe-inline'") {
			status = TestStatusWarning
			severity = SeverityMedium
			details += " (uses 'unsafe-inline')"
		}
		if strings.Contains(csp, "'unsafe-eval'") {
			status = TestStatusWarning
			severity = SeverityMedium
			details += " (uses 'unsafe-eval')"
		}
	}

	s.addTest(category, &TestCase{
		ID:          "HDR-003",
		Name:        "Content Security Policy",
		Description: "Content-Security-Policy header",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Timestamp:   time.Now(),
	})
}

// checkXFrameOptions checks X-Frame-Options header
func (s *Scanner) checkXFrameOptions(category *Category, resp *http.Response) {
	xfo := resp.Header.Get("X-Frame-Options")

	status := TestStatusPass
	severity := SeverityInfo
	details := fmt.Sprintf("X-Frame-Options: %s", xfo)

	if xfo == "" {
		status = TestStatusFail
		severity = SeverityMedium
		details = "X-Frame-Options header missing (clickjacking protection)"
	} else if strings.ToUpper(xfo) != "DENY" && strings.ToUpper(xfo) != "SAMEORIGIN" {
		status = TestStatusWarning
		severity = SeverityLow
		details = fmt.Sprintf("X-Frame-Options has unusual value: %s", xfo)
	}

	s.addTest(category, &TestCase{
		ID:          "HDR-004",
		Name:        "X-Frame-Options",
		Description: "Clickjacking protection header",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Timestamp:   time.Now(),
	})
}

// checkXContentTypeOptions checks X-Content-Type-Options header
func (s *Scanner) checkXContentTypeOptions(category *Category, resp *http.Response) {
	xcto := resp.Header.Get("X-Content-Type-Options")

	status := TestStatusPass
	severity := SeverityInfo
	details := "X-Content-Type-Options: nosniff"

	if xcto == "" {
		status = TestStatusWarning
		severity = SeverityLow
		details = "X-Content-Type-Options header missing"
	} else if strings.ToLower(xcto) != "nosniff" {
		status = TestStatusWarning
		severity = SeverityLow
		details = fmt.Sprintf("X-Content-Type-Options has unexpected value: %s", xcto)
	}

	s.addTest(category, &TestCase{
		ID:          "HDR-005",
		Name:        "X-Content-Type-Options",
		Description: "MIME-sniffing protection",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Timestamp:   time.Now(),
	})
}

// checkXXSSProtection checks X-XSS-Protection header
func (s *Scanner) checkXXSSProtection(category *Category, resp *http.Response) {
	xxss := resp.Header.Get("X-XSS-Protection")

	status := TestStatusPass
	severity := SeverityInfo
	details := fmt.Sprintf("X-XSS-Protection: %s", xxss)

	if xxss == "" {
		status = TestStatusWarning
		severity = SeverityLow
		details = "X-XSS-Protection header missing (legacy browsers)"
	} else if xxss != "1; mode=block" && xxss != "1" {
		status = TestStatusWarning
		severity = SeverityLow
		details = fmt.Sprintf("X-XSS-Protection has unusual value: %s", xxss)
	}

	s.addTest(category, &TestCase{
		ID:          "HDR-006",
		Name:        "X-XSS-Protection",
		Description: "XSS filter header (legacy)",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Timestamp:   time.Now(),
	})
}

// checkReferrerPolicy checks Referrer-Policy header
func (s *Scanner) checkReferrerPolicy(category *Category, resp *http.Response) {
	rp := resp.Header.Get("Referrer-Policy")

	status := TestStatusPass
	severity := SeverityInfo
	details := fmt.Sprintf("Referrer-Policy: %s", rp)

	if rp == "" {
		status = TestStatusWarning
		severity = SeverityLow
		details = "Referrer-Policy header missing"
	}

	s.addTest(category, &TestCase{
		ID:          "HDR-007",
		Name:        "Referrer-Policy",
		Description: "Referrer information control",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Timestamp:   time.Now(),
	})
}

// checkServerHeader checks if Server header exposes version info
func (s *Scanner) checkServerHeader(category *Category, resp *http.Response) {
	server := resp.Header.Get("Server")

	status := TestStatusPass
	severity := SeverityInfo
	details := "Server header not exposing version"

	if server == "" {
		details = "Server header not present (good)"
	} else if strings.Contains(strings.ToLower(server), "version") ||
		strings.Contains(server, "/") {
		status = TestStatusWarning
		severity = SeverityLow
		details = fmt.Sprintf("Server header may expose version: %s", server)
	} else {
		details = fmt.Sprintf("Server: %s", server)
	}

	s.addTest(category, &TestCase{
		ID:          "HDR-008",
		Name:        "Server Header",
		Description: "Check for information disclosure in Server header",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Timestamp:   time.Now(),
	})
}
