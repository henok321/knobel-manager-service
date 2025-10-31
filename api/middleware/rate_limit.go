package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if ip := net.ParseIP(clientIP); ip != nil {
				return clientIP
			}
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if net.ParseIP(r.RemoteAddr) != nil {
			return r.RemoteAddr
		}
		return "unknown"
	}
	return ip
}

var limiterCache *cache.Cache

type RateConfig struct {
	Limit                rate.Limit
	Burst                int
	CacheDefaultDuration time.Duration
	CacheCleanupPeriod   time.Duration
}

func cachedLimiterByKey(key string, limit rate.Limit, burst int, cacheDefaultDuration, cacheCleanupPeriod time.Duration) *rate.Limiter {
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
		limiter := cachedLimiterByKey(getClientIP(r), config.Limit, config.Burst, config.CacheDefaultDuration, config.CacheCleanupPeriod)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
