package middleware

import (
	"net/http"
	"os"
	"strconv"
)

const (
	// DefaultMaxRequestSize is 1MB
	DefaultMaxRequestSize = 1048576
)

// SecurityHeaders adds security-related HTTP headers and request size limits to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	maxSize := DefaultMaxRequestSize

	// Allow override via environment variable
	if maxSizeEnv := os.Getenv("MAX_REQUEST_SIZE"); maxSizeEnv != "" {
		if size, err := strconv.ParseInt(maxSizeEnv, 10, 64); err == nil && size > 0 {
			maxSize = int(size)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Limit request body size to prevent memory exhaustion attacks
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))
		}

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS protection in browsers
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS (only if not in local development)
		// Note: HSTS should only be set when serving over HTTPS
		// The production environment (Fly.io) should handle this at the edge
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy - restrict resource loading
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Referrer policy - control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy - disable unnecessary browser features
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}
