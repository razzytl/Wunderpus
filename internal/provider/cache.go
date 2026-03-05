package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// CacheEntry stores a cached response.
type CacheEntry struct {
	Response  *CompletionResponse
	ExpiresAt time.Time
}

// ResponseCache is an in-memory TTL cache for provider responses.
type ResponseCache struct {
	mu     sync.RWMutex
	data   map[string]CacheEntry
	ttl    time.Duration
}

// NewResponseCache creates a new response cache.
func NewResponseCache(ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		data: make(map[string]CacheEntry),
		ttl:  ttl,
	}
}

// Get retrieves a response from the cache if it exists and hasn't expired.
func (c *ResponseCache) Get(req *CompletionRequest) (*CompletionResponse, bool) {
	key := c.hashRequest(req)
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Response, true
}

// Set stores a response in the cache.
func (c *ResponseCache) Set(req *CompletionRequest, resp *CompletionResponse) {
	key := c.hashRequest(req)
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = CacheEntry{
		Response:  resp,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func (c *ResponseCache) hashRequest(req *CompletionRequest) string {
	b, _ := json.Marshal(req)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
