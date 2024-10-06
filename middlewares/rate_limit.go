package middlewares

import (
	"golang.org/x/time/rate"
	"net/http"
)

var limiter = rate.NewLimiter(2, 5) // Allow 2 requests per second, burst up to 5

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
