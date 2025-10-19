package middleware

import (
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type keyFunc func(r *http.Request) string

var (
	limiterCache *cache.Cache
)

func getLimiter(key string, limit rate.Limit, burst int) *rate.Limiter {
	if limiterCache == nil {
		// Default expiration: 15 minutes, cleanup interval: 30 minutes
		limiterCache = cache.New(15*time.Minute, 30*time.Minute)
	}

	if l, found := limiterCache.Get(key); found {
		return l.(*rate.Limiter)
	}
	l := rate.NewLimiter(limit, burst)
	limiterCache.Set(key, l, cache.DefaultExpiration)
	return l
}

func RateLimit(keyFunc keyFunc, limit rate.Limit, burst int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := getLimiter(keyFunc(r), limit, burst)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
