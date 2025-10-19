package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type keyFunc func(r *http.Request) string

var (
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
)

func getLimiter(key string, limit rate.Limit, burst int) *rate.Limiter {
	if limiters == nil {
		limiters = make(map[string]*rate.Limiter)
	}

	l := limiters[key]
	if l == nil {
		l = rate.NewLimiter(limit, burst)
		limiters[key] = l
	}
	return l
}

func RateLimit(keyFunc keyFunc, limit rate.Limit, burst int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		limiter := getLimiter(keyFunc(r), limit, burst)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
