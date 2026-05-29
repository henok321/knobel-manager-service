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

	ip, err := getClientIP(req, false, 1)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip, "untrusted XFF must be ignored")
}

func TestGetClientIP_TrustsRightmostXFFWithOneHop(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 8.8.8.8")

	ip, err := getClientIP(req, true, 1)
	require.NoError(t, err)
	assert.Equal(t, "8.8.8.8", ip, "with 1 trusted hop, right-most XFF entry is used")
}

func TestGetClientIP_TwoHopsReturnsRealClient(t *testing.T) {
	// Topology: client -> proxy1 -> proxy2 -> service.
	// proxy1 appended the client IP; proxy2 appended proxy1's IP.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 172.20.0.2")

	ip, err := getClientIP(req, true, 2)
	require.NoError(t, err)
	assert.Equal(t, "203.0.113.5", ip, "with 2 trusted hops, leftmost-of-trusted entry is the real client")
}

func TestGetClientIP_TwoHopsIgnoresAttackerPrepended(t *testing.T) {
	// Attacker (real IP 203.0.113.5) sent XFF: "victim, fake-proxy" to spoof a chain.
	// proxy1 appended the attacker's real IP; proxy2 appended proxy1's IP.
	// Result: "victim, fake-proxy, 203.0.113.5, 172.20.0.2"
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2, 203.0.113.5, 172.20.0.2")

	ip, err := getClientIP(req, true, 2)
	require.NoError(t, err)
	assert.Equal(t, "203.0.113.5", ip, "attacker-prepended entries must not influence the result")
}

func TestGetClientIP_FallsBackWhenChainShorterThanHops(t *testing.T) {
	// Service configured for 2 hops but XFF only has 1 entry — header is
	// malformed or trust config is wrong. Fall back to RemoteAddr.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	ip, err := getClientIP(req, true, 2)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", ip)
}

func TestGetClientIP_FallsBackWhenEntryAtDepthIsInvalid(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "garbage, 172.20.0.2")

	ip, err := getClientIP(req, true, 2)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", ip, "invalid entry at trusted depth must not silently fall through to less-trusted entries")
}

func TestGetClientIP_FallsBackToRemoteAddrWhenXFFEmpty(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"

	ip, err := getClientIP(req, true, 1)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", ip)
}

func TestGetClientIP_ZeroOrNegativeHopsTreatedAsOne(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 8.8.8.8")

	ip, err := getClientIP(req, true, 0)
	require.NoError(t, err)
	assert.Equal(t, "8.8.8.8", ip, "zero hops defaults to 1 (right-most)")
}
