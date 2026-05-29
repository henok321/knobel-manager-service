package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestCachedLimiterByKey_RefreshesTTLOnHit(t *testing.T) {
	const ttl = 100 * time.Millisecond
	cache := expirable.NewLRU[string, *rate.Limiter](10, nil, ttl)

	first := cachedLimiterByKey("ip", 1, 1, cache)

	// Repeatedly fetch within TTL — each hit should refresh expiry.
	for range 5 {
		time.Sleep(ttl / 2)
		again := cachedLimiterByKey("ip", 1, 1, cache)
		assert.Same(t, first, again, "active client must keep the same limiter across TTL boundaries")
	}

	// Stop accessing the key; let TTL elapse fully.
	time.Sleep(ttl * 2)
	fresh := cachedLimiterByKey("ip", 1, 1, cache)
	assert.NotSame(t, first, fresh, "idle entry should be evicted after TTL")
}

func TestGetClientIP_IgnoresXFFWhenUntrusted(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	ip, err := getClientIP(req, false)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip, "untrusted XFF must be ignored")
}

func TestGetClientIP_TrustsRightmostXFFWhenTrusted(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 8.8.8.8")

	ip, err := getClientIP(req, true)
	require.NoError(t, err)
	assert.Equal(t, "8.8.8.8", ip, "right-most XFF entry must be used when trusted")
}

func TestGetClientIP_TrustedXFFSkipsInvalidEntries(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "1.1.1.1, garbage")

	ip, err := getClientIP(req, true)
	require.NoError(t, err)
	assert.Equal(t, "1.1.1.1", ip, "must skip non-IP entries from right when searching XFF")
}

func TestGetClientIP_FallsBackToRemoteAddrWhenXFFEmpty(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"

	ip, err := getClientIP(req, true)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", ip)
}
