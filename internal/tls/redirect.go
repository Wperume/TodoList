package tls

import (
	"fmt"
	"net/http"
	"strings"

	"todolist-api/internal/logging"
)

// HTTPSRedirectHandler creates an HTTP handler that redirects to HTTPS
func HTTPSRedirectHandler(httpsPort string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the host without port
		host := r.Host
		if colonPos := strings.LastIndex(host, ":"); colonPos != -1 {
			host = host[:colonPos]
		}

		// Build HTTPS URL
		var httpsURL string
		if httpsPort == "443" {
			// Default HTTPS port, don't include in URL
			httpsURL = fmt.Sprintf("https://%s%s", host, r.RequestURI)
		} else {
			// Non-standard port, include it
			httpsURL = fmt.Sprintf("https://%s:%s%s", host, httpsPort, r.RequestURI)
		}

		logging.Logger.WithFields(map[string]interface{}{
			"client_ip": r.RemoteAddr,
			"http_url":  r.URL.String(),
			"https_url": httpsURL,
			"method":    r.Method,
		}).Debug("HTTP to HTTPS redirect")

		// Redirect with 301 (Permanent) or 308 (Permanent, preserves method)
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})
}
