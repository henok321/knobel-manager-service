package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/henok321/knobel-manager-service/integrationtests/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	tests := []struct {
		name                string
		ipRateLimit         string
		userRateLimit       string
		requests            int
		authHeader          string
		remoteAddr          string
		xForwardedFor       string
		expectedOK          int
		expectedRateLimited int
	}{
		{
			name:                "IP rate limit exceeded",
			ipRateLimit:         "2",
			userRateLimit:       "100",
			requests:            3,
			remoteAddr:          "192.168.1.1",
			expectedOK:          2,
			expectedRateLimited: 1,
		},
		{
			name:                "IP rate limit with X-Forwarded-For",
			ipRateLimit:         "2",
			userRateLimit:       "100",
			requests:            3,
			remoteAddr:          "127.0.0.1",
			xForwardedFor:       "10.0.0.1",
			expectedOK:          2,
			expectedRateLimited: 1,
		},
		{
			name:                "User rate limit exceeded",
			ipRateLimit:         "100",
			userRateLimit:       "2",
			requests:            3,
			authHeader:          "Bearer test-user-123",
			remoteAddr:          "192.168.1.1",
			expectedOK:          2,
			expectedRateLimited: 1,
		},
		{
			name:                "No rate limit exceeded",
			ipRateLimit:         "10",
			userRateLimit:       "10",
			requests:            5,
			remoteAddr:          "192.168.1.2",
			expectedOK:          5,
			expectedRateLimited: 0,
		},
		{
			name:                "User rate limit with auth",
			ipRateLimit:         "100",
			userRateLimit:       "2",
			requests:            2,
			authHeader:          "Bearer user-1",
			remoteAddr:          "192.168.1.3",
			expectedOK:          2,
			expectedRateLimited: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			os.Setenv("RATE_LIMIT_IP", tt.ipRateLimit)
			os.Setenv("RATE_LIMIT_USER", tt.userRateLimit)
			defer os.Unsetenv("RATE_LIMIT_IP")
			defer os.Unsetenv("RATE_LIMIT_USER")

			// Reset the once variable to allow re-initialization with new env vars
			once = sync.Once{}

			// Create a test handler with middleware stack
			finalHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("success"))
			})

			var handler http.Handler
			if tt.authHeader != "" {
				// Apply auth first, then rate limit (auth sets user in context for rate limit)
				handler = Authentication(mock.FirebaseAuthMock{}, RateLimit(finalHandler))
			} else {
				// Apply only rate limit middleware for public endpoints
				handler = RateLimit(finalHandler)
			}

			okCount := 0
			rateLimitedCount := 0

			// Make requests
			for range tt.requests {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr

				if tt.xForwardedFor != "" {
					req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
				}

				if tt.authHeader != "" {
					req.Header.Set("Authorization", tt.authHeader)
				}

				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				switch rr.Code {
				case http.StatusOK:
					okCount++
					assert.Equal(t, "success", rr.Body.String())
				case http.StatusTooManyRequests:
					rateLimitedCount++
					assert.JSONEq(t, `{"error": "rate limit exceeded"}`, rr.Body.String())
				default:
					t.Errorf("Unexpected status code: %d", rr.Code)
				}
			}

			assert.Equal(t, tt.expectedOK, okCount, "Number of successful requests")
			assert.Equal(t, tt.expectedRateLimited, rateLimitedCount, "Number of rate limited requests")
		})
	}
}

func TestRateLimit_DifferentIPs(t *testing.T) {
	// Set low IP rate limit for testing
	os.Setenv("RATE_LIMIT_IP", "2")
	defer os.Unsetenv("RATE_LIMIT_IP")

	// Reset the once variable to allow re-initialization with new env vars
	once = sync.Once{}

	// Create a test handler
	handler := RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))

	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	// Each IP should be able to make 2 requests successfully
	for _, ip := range ips {
		for i := range 2 {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = ip

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code, "IP %s, request %d should succeed", ip, i+1)
		}
	}
}

func TestRateLimit_DifferentUsers(t *testing.T) {
	// Set low user rate limit for testing
	os.Setenv("RATE_LIMIT_USER", "2")
	os.Setenv("RATE_LIMIT_IP", "100")
	defer os.Unsetenv("RATE_LIMIT_USER")
	defer os.Unsetenv("RATE_LIMIT_IP")

	// Reset the once variable to allow re-initialization with new env vars
	once = sync.Once{}

	// Create a test handler with auth middleware
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})
	// Apply auth first, then rate limit (auth sets user in context for rate limit)
	handler := Authentication(mock.FirebaseAuthMock{}, RateLimit(finalHandler))

	userTokens := []string{"Bearer user-1", "Bearer user-2", "Bearer user-3"}

	// Each user should be able to make 2 requests successfully
	for _, token := range userTokens {
		for i := range 2 {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1"
			req.Header.Set("Authorization", token)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code, "User %s, request %d should succeed", token, i+1)
		}
	}
}
