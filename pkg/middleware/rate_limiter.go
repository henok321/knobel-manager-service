package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	rate      int
	burst     int
	tokens    int
	lastCheck time.Time
	mu        sync.Mutex
}

func NewRateLimiter(rate int, burst int) *RateLimiter {
	return &RateLimiter{
		rate:      rate,
		burst:     burst,
		tokens:    burst,
		lastCheck: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastCheck).Seconds()
	rl.lastCheck = now

	rl.tokens += int(elapsed * float64(rl.rate))
	if rl.tokens > rl.burst {
		rl.tokens = rl.burst
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

func RateLimiterMiddleware(rate int, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, burst)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
