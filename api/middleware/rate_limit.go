package middleware

import (
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type keyFunc func(r *http.Request) string

var limiterCache *cache.Cache

type RateConfig struct {
	Limit                rate.Limit
	Burst                int
	KeyFunc              keyFunc
	CacheDefaultDuration time.Duration
	CacheCleanupPeriod   time.Duration
}

func getLimiter(key string, limit rate.Limit, burst int, cacheDefaultDuration, cacheCleanupPeriod time.Duration) *rate.Limiter {
	if limiterCache == nil {
		limiterCache = cache.New(cacheDefaultDuration, cacheCleanupPeriod)
	}

	if l, found := limiterCache.Get(key); found {
		return l.(*rate.Limiter)
	}
	l := rate.NewLimiter(limit, burst)
	limiterCache.Set(key, l, cache.DefaultExpiration)
	return l
}

func RateLimit(config RateConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := getLimiter(config.KeyFunc(r), config.Limit, config.Burst, config.CacheDefaultDuration, config.CacheCleanupPeriod)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
