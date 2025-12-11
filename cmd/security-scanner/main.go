package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"todolist-api/cmd/security-scanner/report"
	"todolist-api/cmd/security-scanner/scanner"
)

func main() {
	// Parse command line flags
	target := flag.String("target", "", "Target URL (e.g., https://api.example.com)")
	safeMode := flag.Bool("safe-mode", true, "Enable safe mode (read-only, limited testing)")
	maxRPS := flag.Int("max-rps", 5, "Maximum requests per second")
	timeout := flag.Int("timeout", 10, "Request timeout in seconds")
	skipTLS := flag.Bool("skip-tls", false, "Skip TLS certificate verification (for testing only)")
	output := flag.String("output", "", "Output file path (default: security-report.html or security-report.json)")
	format := flag.String("format", "html", "Output format (html, json)")
	verbose := flag.Bool("verbose", false, "Verbose output")
	testUser := flag.String("test-user", "", "Test user email (optional)")
	testPassword := flag.String("test-password", "", "Test user password (optional)")

	flag.Parse()

	// Validate required flags
	if *target == "" {
		fmt.Fprintln(os.Stderr, "Error: --target is required")
		fmt.Fprintln(os.Stderr, "\nUsage:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintln(os.Stderr, "  security-scanner --target https://api.example.com")
		fmt.Fprintln(os.Stderr, "  security-scanner --target https://localhost:8443 --skip-tls --output report.html")
		os.Exit(1)
	}

	// Set default output filename based on format if not specified
	if *output == "" {
		if *format == "json" {
			*output = "security-report.json"
		} else {
			*output = "security-report.html"
		}
	}

	// Create scanner config
	config := &scanner.ScanConfig{
		Target:       *target,
		SafeMode:     *safeMode,
		MaxRPS:       *maxRPS,
		Timeout:      time.Duration(*timeout) * time.Second,
		SkipTLS:      *skipTLS,
		TestUser:     *testUser,
		TestPassword: *testPassword,
		Verbose:      *verbose,
	}

	// Print banner
	if *verbose {
		fmt.Println("╔══════════════════════════════════════════════════════════╗")
		fmt.Println("║   TodoList API Security Scanner                         ║")
		fmt.Println("╚══════════════════════════════════════════════════════════╝")
		fmt.Println()
	}

	// Create and run scanner
	s := scanner.NewScanner(config)
	result := s.Run()

	// Generate output
	var err error
	switch *format {
	case "html":
		err = report.GenerateHTML(result, *output)
		if err == nil {
			fmt.Printf("\n✓ HTML report generated: %s\n", *output)
		}
	case "json":
		err = generateJSON(result, *output)
		if err == nil {
			fmt.Printf("\n✓ JSON report generated: %s\n", *output)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format: %s\n", *format)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	if *verbose {
		fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
		fmt.Println("║   Scan Summary                                           ║")
		fmt.Println("╚══════════════════════════════════════════════════════════╝")
		fmt.Printf("Total Tests:    %d\n", result.TotalTests)
		fmt.Printf("Passed:         %d\n", result.PassedTests)
		fmt.Printf("Failed:         %d\n", result.FailedTests)
		fmt.Printf("Warnings:       %d\n", result.WarningTests)
		fmt.Printf("Security Score: %d/100\n", result.SecurityScore)
		fmt.Println()
	}

	// Exit with appropriate code
	if result.FailedTests > 0 {
		os.Exit(1)
	}
}

// generateJSON generates a JSON report
func generateJSON(result *scanner.ScanResult, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
