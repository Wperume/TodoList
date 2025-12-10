package scanner

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// scanTLS performs TLS/SSL security checks
func (s *Scanner) scanTLS() {
	category := s.addCategory("TLS/SSL", "TLS/SSL configuration and certificate validation")

	parsedURL, err := url.Parse(s.config.Target)
	if err != nil {
		return
	}

	// Skip if not HTTPS
	if parsedURL.Scheme != "https" {
		s.addTest(category, &TestCase{
			ID:          "TLS-001",
			Name:        "HTTPS Enabled",
			Description: "Check if HTTPS is enabled",
			Status:      TestStatusFail,
			Severity:    SeverityCritical,
			Details:     "HTTPS is not enabled. All traffic is unencrypted.",
			Timestamp:   time.Now(),
		})
		return
	}

	// Test TLS connection
	s.testTLSConnection(category, parsedURL.Host)
	s.testTLSVersion(category, parsedURL.Host)
	s.testCertificate(category, parsedURL.Host)
}

// testTLSConnection tests if TLS connection can be established
func (s *Scanner) testTLSConnection(category *Category, host string) {
	startTime := time.Now()

	// Ensure host has port
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	conn, err := tls.Dial("tcp", host, &tls.Config{
		InsecureSkipVerify: false,
	})

	status := TestStatusPass
	details := "TLS connection successful"

	if err != nil {
		status = TestStatusFail
		details = fmt.Sprintf("TLS connection failed: %v", err)
	} else {
		conn.Close()
	}

	s.addTest(category, &TestCase{
		ID:          "TLS-002",
		Name:        "TLS Connection",
		Description: "Check if secure TLS connection can be established",
		Status:      status,
		Severity:    SeverityHigh,
		Details:     details,
		Duration:    time.Since(startTime),
		Timestamp:   time.Now(),
	})
}

// testTLSVersion tests the TLS version
func (s *Scanner) testTLSVersion(category *Category, host string) {
	startTime := time.Now()

	if !strings.Contains(host, ":") {
		host += ":443"
	}

	conn, err := tls.Dial("tcp", host, &tls.Config{
		InsecureSkipVerify: s.config.SkipTLS,
	})

	if err != nil {
		s.addTest(category, &TestCase{
			ID:          "TLS-003",
			Name:        "TLS Version",
			Description: "Check TLS version",
			Status:      TestStatusError,
			Severity:    SeverityMedium,
			Details:     fmt.Sprintf("Could not check TLS version: %v", err),
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
		return
	}
	defer conn.Close()

	state := conn.ConnectionState()
	version := ""
	status := TestStatusPass
	severity := SeverityLow

	switch state.Version {
	case tls.VersionTLS13:
		version = "TLS 1.3"
		details := "Using TLS 1.3 (recommended)"
		s.addTest(category, &TestCase{
			ID:          "TLS-003",
			Name:        "TLS Version",
			Description: "Check TLS version",
			Status:      status,
			Severity:    severity,
			Details:     details,
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
	case tls.VersionTLS12:
		version = "TLS 1.2"
		status = TestStatusPass
		severity = SeverityInfo
		s.addTest(category, &TestCase{
			ID:          "TLS-003",
			Name:        "TLS Version",
			Description: "Check TLS version",
			Status:      status,
			Severity:    severity,
			Details:     fmt.Sprintf("Using %s (acceptable, but TLS 1.3 recommended)", version),
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
	case tls.VersionTLS11:
		version = "TLS 1.1"
		status = TestStatusWarning
		severity = SeverityMedium
		s.addTest(category, &TestCase{
			ID:          "TLS-003",
			Name:        "TLS Version",
			Description: "Check TLS version",
			Status:      status,
			Severity:    severity,
			Details:     fmt.Sprintf("Using %s (deprecated, upgrade to TLS 1.2 or 1.3)", version),
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
	case tls.VersionTLS10:
		version = "TLS 1.0"
		status = TestStatusFail
		severity = SeverityHigh
		s.addTest(category, &TestCase{
			ID:          "TLS-003",
			Name:        "TLS Version",
			Description: "Check TLS version",
			Status:      status,
			Severity:    severity,
			Details:     fmt.Sprintf("Using %s (insecure, must upgrade to TLS 1.2 or 1.3)", version),
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
	default:
		status = TestStatusFail
		severity = SeverityHigh
		s.addTest(category, &TestCase{
			ID:          "TLS-003",
			Name:        "TLS Version",
			Description: "Check TLS version",
			Status:      status,
			Severity:    severity,
			Details:     fmt.Sprintf("Using unknown or insecure TLS version: 0x%x", state.Version),
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
	}
}

// testCertificate tests the SSL certificate
func (s *Scanner) testCertificate(category *Category, host string) {
	startTime := time.Now()

	if !strings.Contains(host, ":") {
		host += ":443"
	}

	conn, err := tls.Dial("tcp", host, &tls.Config{
		InsecureSkipVerify: false,
	})

	if err != nil {
		s.addTest(category, &TestCase{
			ID:          "TLS-004",
			Name:        "Certificate Validity",
			Description: "Check SSL certificate validity",
			Status:      TestStatusFail,
			Severity:    SeverityCritical,
			Details:     fmt.Sprintf("Certificate validation failed: %v", err),
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
		return
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		s.addTest(category, &TestCase{
			ID:          "TLS-004",
			Name:        "Certificate Validity",
			Description: "Check SSL certificate validity",
			Status:      TestStatusFail,
			Severity:    SeverityCritical,
			Details:     "No certificate presented",
			Duration:    time.Since(startTime),
			Timestamp:   time.Now(),
		})
		return
	}

	cert := state.PeerCertificates[0]

	// Check expiration
	now := time.Now()
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)

	status := TestStatusPass
	severity := SeverityInfo
	details := fmt.Sprintf("Certificate is valid. Expires in %d days (%s)", daysUntilExpiry, cert.NotAfter.Format("2006-01-02"))

	if now.Before(cert.NotBefore) {
		status = TestStatusFail
		severity = SeverityCritical
		details = "Certificate is not yet valid"
	} else if now.After(cert.NotAfter) {
		status = TestStatusFail
		severity = SeverityCritical
		details = "Certificate has expired"
	} else if daysUntilExpiry < 7 {
		status = TestStatusWarning
		severity = SeverityHigh
		details = fmt.Sprintf("Certificate expires soon (%d days)", daysUntilExpiry)
	} else if daysUntilExpiry < 30 {
		status = TestStatusWarning
		severity = SeverityMedium
		details = fmt.Sprintf("Certificate expires in %d days", daysUntilExpiry)
	}

	s.addTest(category, &TestCase{
		ID:          "TLS-004",
		Name:        "Certificate Validity",
		Description: "Check SSL certificate validity",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Duration:    time.Since(startTime),
		Timestamp:   time.Now(),
	})

	// Check certificate chain
	s.testCertificateChain(category, state.PeerCertificates)
}

// testCertificateChain validates the certificate chain
func (s *Scanner) testCertificateChain(category *Category, certs []*x509.Certificate) {
	status := TestStatusPass
	severity := SeverityLow
	details := fmt.Sprintf("Certificate chain has %d certificates", len(certs))

	if len(certs) < 2 {
		status = TestStatusWarning
		severity = SeverityMedium
		details = "Certificate chain may be incomplete (only 1 certificate)"
	}

	s.addTest(category, &TestCase{
		ID:          "TLS-005",
		Name:        "Certificate Chain",
		Description: "Validate certificate chain",
		Status:      status,
		Severity:    severity,
		Details:     details,
		Timestamp:   time.Now(),
	})
}
