package middleware

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultIPRateLimit   = 100
	DefaultUserRateLimit = 200
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int
	burst    int
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     requestsPerMinute,
		burst:    requestsPerMinute,
	}

	go rl.cleanupVisitors()

	return rl
}

func (rl *rateLimiter) getVisitor(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[key]
	if !exists {
		limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(rl.rate)), rl.burst)
		rl.visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *rateLimiter) cleanupVisitors() {
	for {
		time.Sleep(5 * time.Minute)

		rl.mu.Lock()
		for key, v := range rl.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

var (
	ipLimiter   *rateLimiter
	userLimiter *rateLimiter
	once        sync.Once
	mu          sync.Mutex
)

func initLimiters() {
	ipRate := DefaultIPRateLimit
	userRate := DefaultUserRateLimit

	if ipRateEnv := os.Getenv("RATE_LIMIT_IP"); ipRateEnv != "" {
		if r, err := strconv.Atoi(ipRateEnv); err == nil && r > 0 {
			ipRate = r
		}
	}

	if userRateEnv := os.Getenv("RATE_LIMIT_USER"); userRateEnv != "" {
		if r, err := strconv.Atoi(userRateEnv); err == nil && r > 0 {
			userRate = r
		}
	}

	ipLimiter = newRateLimiter(ipRate)
	userLimiter = newRateLimiter(userRate)
}

// ResetRateLimitForTesting resets the rate limiters - should only be used in tests
func ResetRateLimitForTesting() {
	mu.Lock()
	defer mu.Unlock()
	once = sync.Once{}
	ipLimiter = nil
	userLimiter = nil
}

func RateLimit(next http.Handler) http.Handler {
	once.Do(initLimiters)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		// Strip port from IP address for rate limiting
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}

		ipLimiterInstance := ipLimiter.getVisitor(ip)
		if !ipLimiterInstance.Allow() {
			http.Error(w, `{"error": "rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		if user, ok := UserFromContext(r.Context()); ok && user.Sub != "" {
			userLimiterInstance := userLimiter.getVisitor(user.Sub)
			if !userLimiterInstance.Allow() {
				http.Error(w, `{"error": "rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
