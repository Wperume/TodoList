package tls

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"

	"todolist-api/internal/logging"
)

// TLSConfig holds TLS/HTTPS configuration
type Config struct {
	Enabled                  bool
	CertFile                 string
	KeyFile                  string
	Port                     string
	HTTPPort                 string // Port for HTTP to HTTPS redirect
	RedirectHTTP             bool   // Redirect HTTP to HTTPS
	MinVersion               uint16 // Minimum TLS version
	MaxVersion               uint16 // Maximum TLS version
	CipherSuites             []uint16
	PreferServerCipherSuites bool
}

// NewTLSConfigFromEnv creates TLS config from environment variables
func NewConfigFromEnv() *Config {
	enabled := getEnvBool("TLS_ENABLED", false)

	config := &Config{
		Enabled:                  enabled,
		CertFile:                 getEnv("TLS_CERT_FILE", "./certs/server.crt"),
		KeyFile:                  getEnv("TLS_KEY_FILE", "./certs/server.key"),
		Port:                     getEnv("TLS_PORT", "8443"),
		HTTPPort:                 getEnv("HTTP_PORT", "8080"),
		RedirectHTTP:             getEnvBool("TLS_REDIRECT_HTTP", true),
		MinVersion:               parseTLSVersion(getEnv("TLS_MIN_VERSION", "1.2")),
		MaxVersion:               parseTLSVersion(getEnv("TLS_MAX_VERSION", "1.3")),
		PreferServerCipherSuites: getEnvBool("TLS_PREFER_SERVER_CIPHERS", true),
	}

	// Set secure cipher suites (TLS 1.2 and 1.3)
	config.CipherSuites = []uint16{
		// TLS 1.3 suites (automatically used if TLS 1.3 is negotiated)
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,

		// TLS 1.2 suites
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	}

	return config
}

// CreateTLSConfig creates a *tls.Config for the server
func (c *Config) CreateTLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, fmt.Errorf("TLS is not enabled")
	}

	// Validate certificate files exist
	if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("certificate file not found: %s", c.CertFile)
	}
	if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("key file not found: %s", c.KeyFile)
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               c.MinVersion,
		MaxVersion:               c.MaxVersion,
		CipherSuites:             c.CipherSuites,
		PreferServerCipherSuites: c.PreferServerCipherSuites,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}

	logging.Logger.Infof("TLS configured: cert=%s, minVersion=%s, maxVersion=%s",
		c.CertFile, tlsVersionString(c.MinVersion), tlsVersionString(c.MaxVersion))

	return tlsConfig, nil
}

// parseTLSVersion parses TLS version string to uint16
func parseTLSVersion(version string) uint16 {
	switch version {
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3":
		return tls.VersionTLS13
	default:
		logging.Logger.Warnf("Unknown TLS version '%s', using TLS 1.2", version)
		return tls.VersionTLS12
	}
}

// tlsVersionString converts TLS version uint16 to string
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
