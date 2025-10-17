package middleware

import (
	"net/http"
	"os"
	"strconv"
)

const (
	DefaultMaxRequestSize = 1048576
)

func SecurityHeaders(next http.Handler) http.Handler {
	maxSize := DefaultMaxRequestSize

	if maxSizeEnv := os.Getenv("MAX_REQUEST_SIZE"); maxSizeEnv != "" {
		if size, err := strconv.ParseInt(maxSizeEnv, 10, 64); err == nil && size > 0 {
			maxSize = int(size)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")

		w.Header().Set("X-Frame-Options", "DENY")

		w.Header().Set("X-XSS-Protection", "1; mode=block")

		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}
