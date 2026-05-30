package middleware

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/time/rate"
)

type RateConfig struct {
	Limit             rate.Limit
	Burst             int
	TrustForwardedFor bool
	TrustedProxyHops  int
}

func RateLimit(config RateConfig, limiterCache *expirable.LRU[string, *rate.Limiter]) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, err := getClientIP(r, config.TrustForwardedFor, config.TrustedProxyHops)
			if err != nil {
				http.Error(w, "Invalid IP", http.StatusBadRequest)
				return
			}

			slog.InfoContext(r.Context(), "Client IP", "ip", ip)

			limiter := cachedLimiterByKey(ip, config.Limit, config.Burst, limiterCache)

			if !limiter.Allow() {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

var errInvalidIP = errors.New("invalid IP")

func getClientIP(r *http.Request, trustForwardedFor bool, trustedProxyHops int) (string, error) {
	if trustForwardedFor {
		hops := max(trustedProxyHops, 1)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			slog.InfoContext(r.Context(), "X-Forwarded-For", "xff", xff)
			ips := strings.Split(xff, ",")
			if len(ips) >= hops {
				candidate := strings.TrimSpace(ips[len(ips)-hops])
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
