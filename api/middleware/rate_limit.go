package middleware

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

var errInvalidIP = errors.New("invalid IP")

func getClientIP(r *http.Request) (string, error) {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if ip := net.ParseIP(clientIP); ip != nil {
				return clientIP, nil
			}
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if net.ParseIP(r.RemoteAddr) != nil {
			return r.RemoteAddr, nil
		}
		return "", errInvalidIP
	}
	return ip, nil
}

var (
	limiterCache *cache.Cache
	initOnce     sync.Once
)

type RateConfig struct {
	Limit                rate.Limit
	Burst                int
	CacheDefaultDuration time.Duration
	CacheCleanupPeriod   time.Duration
}

func initCache(cacheDefaultDuration, cacheCleanupPeriod time.Duration) {
	initOnce.Do(func() {
		limiterCache = cache.New(cacheDefaultDuration, cacheCleanupPeriod)
	})
}

func cachedLimiterByKey(key string, limit rate.Limit, burst int, cacheDefaultDuration, cacheCleanupPeriod time.Duration) *rate.Limiter {
	initCache(cacheDefaultDuration, cacheCleanupPeriod)

	if l, found := limiterCache.Get(key); found {
		return l.(*rate.Limiter)
	}
	l := rate.NewLimiter(limit, burst)
	limiterCache.Set(key, l, cache.DefaultExpiration)
	return l
}

func RateLimit(config RateConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := getClientIP(r)
		if err != nil {
			http.Error(w, "Invalid IP", http.StatusBadRequest)
			return
		}

		limiter := cachedLimiterByKey(ip, config.Limit, config.Burst, config.CacheDefaultDuration, config.CacheCleanupPeriod)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
