package middleware

import (
	"errors"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/time/rate"
)

type RateConfig struct {
	Limit             rate.Limit
	Burst             int
	TrustForwardedFor bool
}

func RateLimit(config RateConfig, limiterCache *expirable.LRU[string, *rate.Limiter], next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := getClientIP(r, config.TrustForwardedFor)
		if err != nil {
			http.Error(w, "Invalid IP", http.StatusBadRequest)
			return
		}

		limiter := cachedLimiterByKey(ip, config.Limit, config.Burst, limiterCache)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

var errInvalidIP = errors.New("invalid IP")

func getClientIP(r *http.Request, trustForwardedFor bool) (string, error) {
	if trustForwardedFor {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			for _, v := range slices.Backward(ips) {
				candidate := strings.TrimSpace(v)
				if net.ParseIP(candidate) != nil {
					return candidate, nil
				}
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

func cachedLimiterByKey(key string, limit rate.Limit, burst int, limiterCache *expirable.LRU[string, *rate.Limiter]) *rate.Limiter {
	if l, found := limiterCache.Get(key); found {
		limiterCache.Add(key, l)
		return l
	}
	l := rate.NewLimiter(limit, burst)
	limiterCache.Add(key, l)
	return l
}
