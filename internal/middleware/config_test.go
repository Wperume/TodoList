package middleware

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvBool(t *testing.T) {
	setupTest()
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		setEnv       bool
		expected     bool
	}{
		{
			name:         "returns true when env is 'true'",
			key:          "TEST_BOOL_TRUE",
			defaultValue: false,
			envValue:     "true",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "returns false when env is 'false'",
			key:          "TEST_BOOL_FALSE",
			defaultValue: true,
			envValue:     "false",
			setEnv:       true,
			expected:     false,
		},
		{
			name:         "returns true when env is '1'",
			key:          "TEST_BOOL_ONE",
			defaultValue: false,
			envValue:     "1",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "returns false when env is '0'",
			key:          "TEST_BOOL_ZERO",
			defaultValue: true,
			envValue:     "0",
			setEnv:       true,
			expected:     false,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_BOOL_UNSET",
			defaultValue: true,
			setEnv:       false,
			expected:     true,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_BOOL_INVALID",
			defaultValue: true,
			envValue:     "invalid",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_BOOL_EMPTY",
			defaultValue: false,
			envValue:     "",
			setEnv:       true,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnvBool(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	setupTest()
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		setEnv       bool
		expected     int
	}{
		{
			name:         "returns parsed integer when valid",
			key:          "TEST_INT_VALID",
			defaultValue: 100,
			envValue:     "42",
			setEnv:       true,
			expected:     42,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_INT_UNSET",
			defaultValue: 100,
			setEnv:       false,
			expected:     100,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_INT_INVALID",
			defaultValue: 100,
			envValue:     "not_a_number",
			setEnv:       true,
			expected:     100,
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_INT_EMPTY",
			defaultValue: 100,
			envValue:     "",
			setEnv:       true,
			expected:     100,
		},
		{
			name:         "returns zero when env is '0'",
			key:          "TEST_INT_ZERO",
			defaultValue: 100,
			envValue:     "0",
			setEnv:       true,
			expected:     0,
		},
		{
			name:         "returns negative integer when valid",
			key:          "TEST_INT_NEGATIVE",
			defaultValue: 100,
			envValue:     "-42",
			setEnv:       true,
			expected:     -42,
		},
		{
			name:         "returns large integer when valid",
			key:          "TEST_INT_LARGE",
			defaultValue: 100,
			envValue:     "1048576",
			setEnv:       true,
			expected:     1048576,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnvInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewSecurityConfigFromEnv(t *testing.T) {
	setupTest()
	tests := []struct {
		name                string
		envVars             map[string]string
		expectedMaxSize     int64
		expectedXSS         bool
		expectedProxyCount  int
	}{
		{
			name:                "returns default values when no env vars set",
			envVars:             map[string]string{},
			expectedMaxSize:     1048576, // 1MB
			expectedXSS:         true,
			expectedProxyCount:  0,
		},
		{
			name: "returns custom values from env vars",
			envVars: map[string]string{
				"MAX_REQUEST_BODY_SIZE": "2097152",
				"ENABLE_XSS_PROTECTION": "false",
				"TRUSTED_PROXIES":       "10.0.0.1,192.168.1.1",
			},
			expectedMaxSize:     2097152, // 2MB
			expectedXSS:         false,
			expectedProxyCount:  2,
		},
		{
			name: "handles single trusted proxy",
			envVars: map[string]string{
				"TRUSTED_PROXIES": "10.0.0.1",
			},
			expectedMaxSize:     1048576,
			expectedXSS:         true,
			expectedProxyCount:  1,
		},
		{
			name: "trims whitespace from trusted proxies",
			envVars: map[string]string{
				"TRUSTED_PROXIES": " 10.0.0.1 , 192.168.1.1 , 172.16.0.1 ",
			},
			expectedMaxSize:     1048576,
			expectedXSS:         true,
			expectedProxyCount:  3,
		},
		{
			name: "enables XSS protection by default",
			envVars: map[string]string{
				"MAX_REQUEST_BODY_SIZE": "512000",
			},
			expectedMaxSize:     512000,
			expectedXSS:         true,
			expectedProxyCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv("MAX_REQUEST_BODY_SIZE")
			os.Unsetenv("ENABLE_XSS_PROTECTION")
			os.Unsetenv("TRUSTED_PROXIES")
			defer func() {
				os.Unsetenv("MAX_REQUEST_BODY_SIZE")
				os.Unsetenv("ENABLE_XSS_PROTECTION")
				os.Unsetenv("TRUSTED_PROXIES")
			}()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config := NewSecurityConfigFromEnv()

			assert.NotNil(t, config)
			assert.Equal(t, tt.expectedMaxSize, config.MaxRequestBodySize)
			assert.Equal(t, tt.expectedXSS, config.EnableXSSProtection)
			assert.Len(t, config.TrustedProxies, tt.expectedProxyCount)
		})
	}
}

func TestNewCORSConfigFromEnv(t *testing.T) {
	setupTest()
	tests := []struct {
		name                    string
		envVars                 map[string]string
		expectedEnabled         bool
		expectedOriginsCount    int
		expectedMethodsCount    int
		expectedHeadersCount    int
		expectedExposeCount     int
		expectedAllowCred       bool
		expectedMaxAge          int
	}{
		{
			name:                    "returns default values when no env vars set",
			envVars:                 map[string]string{},
			expectedEnabled:         true,
			expectedOriginsCount:    1, // ["*"]
			expectedMethodsCount:    6, // GET,POST,PUT,DELETE,OPTIONS,PATCH
			expectedHeadersCount:    5, // Origin,Content-Type,Accept,Authorization,X-API-Key
			expectedExposeCount:     2, // Content-Length,Content-Type
			expectedAllowCred:       false,
			expectedMaxAge:          3600,
		},
		{
			name: "returns custom values from env vars",
			envVars: map[string]string{
				"CORS_ENABLED":           "true",
				"CORS_ALLOWED_ORIGINS":   "https://example.com,https://app.example.com",
				"CORS_ALLOWED_METHODS":   "GET,POST",
				"CORS_ALLOWED_HEADERS":   "Content-Type,Authorization",
				"CORS_EXPOSE_HEADERS":    "X-Total-Count",
				"CORS_ALLOW_CREDENTIALS": "true",
				"CORS_MAX_AGE":           "7200",
			},
			expectedEnabled:         true,
			expectedOriginsCount:    2,
			expectedMethodsCount:    2,
			expectedHeadersCount:    2,
			expectedExposeCount:     1,
			expectedAllowCred:       true,
			expectedMaxAge:          7200,
		},
		{
			name: "handles disabled CORS",
			envVars: map[string]string{
				"CORS_ENABLED": "false",
			},
			expectedEnabled:         false,
			expectedOriginsCount:    1,
			expectedMethodsCount:    6,
			expectedHeadersCount:    5,
			expectedExposeCount:     2,
			expectedAllowCred:       false,
			expectedMaxAge:          3600,
		},
		{
			name: "handles wildcard origin",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": "*",
			},
			expectedEnabled:         true,
			expectedOriginsCount:    1,
			expectedMethodsCount:    6,
			expectedHeadersCount:    5,
			expectedExposeCount:     2,
			expectedAllowCred:       false,
			expectedMaxAge:          3600,
		},
		{
			name: "handles multiple origins",
			envVars: map[string]string{
				"CORS_ALLOWED_ORIGINS": "https://example.com,https://app.example.com,https://admin.example.com",
			},
			expectedEnabled:         true,
			expectedOriginsCount:    3,
			expectedMethodsCount:    6,
			expectedHeadersCount:    5,
			expectedExposeCount:     2,
			expectedAllowCred:       false,
			expectedMaxAge:          3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			envKeys := []string{
				"CORS_ENABLED",
				"CORS_ALLOWED_ORIGINS",
				"CORS_ALLOWED_METHODS",
				"CORS_ALLOWED_HEADERS",
				"CORS_EXPOSE_HEADERS",
				"CORS_ALLOW_CREDENTIALS",
				"CORS_MAX_AGE",
			}
			for _, key := range envKeys {
				os.Unsetenv(key)
			}
			defer func() {
				for _, key := range envKeys {
					os.Unsetenv(key)
				}
			}()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config := NewCORSConfigFromEnv()

			assert.NotNil(t, config)
			assert.Equal(t, tt.expectedEnabled, config.Enabled)
			assert.Len(t, config.AllowedOrigins, tt.expectedOriginsCount)
			assert.Len(t, config.AllowedMethods, tt.expectedMethodsCount)
			assert.Len(t, config.AllowedHeaders, tt.expectedHeadersCount)
			assert.Len(t, config.ExposeHeaders, tt.expectedExposeCount)
			assert.Equal(t, tt.expectedAllowCred, config.AllowCredentials)
			assert.Equal(t, tt.expectedMaxAge, config.MaxAge)
		})
	}
}
