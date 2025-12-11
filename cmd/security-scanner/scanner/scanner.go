package scanner

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// Scanner performs security scans against a target API
type Scanner struct {
	config     *ScanConfig
	httpClient *http.Client
	limiter    *rate.Limiter
	result     *ScanResult
}

// NewScanner creates a new security scanner
func NewScanner(config *ScanConfig) *Scanner {
	// Create HTTP client with appropriate settings
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.SkipTLS,
		},
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: tr,
	}

	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.MaxRPS), 1)

	return &Scanner{
		config:     config,
		httpClient: client,
		limiter:    limiter,
		result: &ScanResult{
			Target:     config.Target,
			ScanDate:   time.Now(),
			Categories: make(map[string]*Category),
		},
	}
}

// Run executes the security scan
func (s *Scanner) Run() *ScanResult {
	startTime := time.Now()

	if s.config.Verbose {
		fmt.Printf("Starting security scan of %s\n", s.config.Target)
		fmt.Printf("Safe mode: %v, Max RPS: %d\n\n", s.config.SafeMode, s.config.MaxRPS)
	}

	// Run all scan categories
	s.scanTLS()
	s.scanHeaders()
	s.scanAuthentication()
	s.scanAPIEndpoints()

	// Calculate final scores
	s.calculateScores()

	s.result.Duration = time.Since(startTime)

	if s.config.Verbose {
		fmt.Printf("\nScan completed in %v\n", s.result.Duration)
		fmt.Printf("Security Score: %d/100\n", s.result.SecurityScore)
	}

	return s.result
}

// addCategory adds a new category to the result
func (s *Scanner) addCategory(name, description string) *Category {
	category := &Category{
		Name:        name,
		Description: description,
		Tests:       []*TestCase{},
	}
	s.result.Categories[name] = category
	return category
}

// addTest adds a test result to a category
func (s *Scanner) addTest(category *Category, test *TestCase) {
	category.Tests = append(category.Tests, test)
	s.result.TotalTests++

	switch test.Status {
	case TestStatusPass:
		s.result.PassedTests++
	case TestStatusFail:
		s.result.FailedTests++
	case TestStatusWarning:
		s.result.WarningTests++
	}
}

// makeRequest makes an HTTP request with rate limiting
func (s *Scanner) makeRequest(method, path string, headers map[string]string) (*http.Response, error) {
	// Wait for rate limiter
	ctx := context.Background()
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	url := s.config.Target + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return s.httpClient.Do(req)
}

// calculateScores calculates the security scores for each category and overall
func (s *Scanner) calculateScores() {
	totalScore := 0
	categoryCount := 0

	for _, category := range s.result.Categories {
		passed := 0
		failed := 0
		warnings := 0

		for _, test := range category.Tests {
			switch test.Status {
			case TestStatusPass:
				passed++
			case TestStatusFail:
				// Weight failures by severity
				switch test.Severity {
				case SeverityCritical:
					failed += 10
				case SeverityHigh:
					failed += 5
				case SeverityMedium:
					failed += 3
				case SeverityLow:
					failed += 1
				}
			case TestStatusWarning:
				warnings++
			}
		}

		// Calculate category score (0-100)
		total := passed + failed + warnings
		if total > 0 {
			category.Score = (passed * 100) / total
			// Penalize for failures
			category.Score -= (failed * 5)
			if category.Score < 0 {
				category.Score = 0
			}
			totalScore += category.Score
			categoryCount++
		}
	}

	// Calculate overall score
	if categoryCount > 0 {
		s.result.SecurityScore = totalScore / categoryCount
	}
}
