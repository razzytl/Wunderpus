package ra

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// RateLimits tracks per-key usage quotas.
type RateLimits struct {
	RequestsPerMinute int       `json:"requests_per_minute"`
	RequestsRemaining int       `json:"requests_remaining"`
	ResetAt           time.Time `json:"reset_at"`
}

// APIKeyEntry holds a single API key with metadata.
type APIKeyEntry struct {
	Provider  string     `json:"provider"`
	Encrypted []byte     `json:"encrypted"` // AES-256-GCM encrypted
	IsFree    bool       `json:"is_free"`
	Limits    RateLimits `json:"limits"`
	ExpiresAt *time.Time `json:"expires_at"`
	Priority  int        `json:"priority"` // lower = higher priority
}

// APIKeyManager manages API key lifecycle: registration, prioritized retrieval,
// rate limit tracking, and rotation.
type APIKeyManager struct {
	keys   map[string][]*APIKeyEntry // provider → keys
	mu     sync.RWMutex
	encKey []byte
}

// NewAPIKeyManager creates a key manager with the given encryption key.
func NewAPIKeyManager(encKey []byte) *APIKeyManager {
	return &APIKeyManager{
		keys:   make(map[string][]*APIKeyEntry),
		encKey: encKey,
	}
}

// Register adds an API key for a provider.
func (k *APIKeyManager) Register(provider string, key string, isFree bool, limits RateLimits) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	encrypted, err := EncryptCreds([]byte(key), k.encKey)
	if err != nil {
		return fmt.Errorf("ra keymanager: encrypt failed: %w", err)
	}

	entry := &APIKeyEntry{
		Provider:  provider,
		Encrypted: encrypted,
		IsFree:    isFree,
		Limits:    limits,
		Priority:  len(k.keys[provider]), // append priority
	}

	k.keys[provider] = append(k.keys[provider], entry)

	// Sort: free keys first, then by priority
	sort.Slice(k.keys[provider], func(i, j int) bool {
		if k.keys[provider][i].IsFree != k.keys[provider][j].IsFree {
			return k.keys[provider][i].IsFree
		}
		return k.keys[provider][i].Priority < k.keys[provider][j].Priority
	})

	return nil
}

// Get returns the best available API key for a provider.
// Free-tier keys are prioritized. Keys with exhausted quotas are skipped.
func (k *APIKeyManager) Get(provider string) (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	keys, ok := k.keys[provider]
	if !ok || len(keys) == 0 {
		return "", fmt.Errorf("ra keymanager: no keys registered for provider %s", provider)
	}

	for _, entry := range keys {
		// Skip expired
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			continue
		}
		// Skip exhausted
		if entry.Limits.RequestsRemaining <= 0 && entry.Limits.RequestsPerMinute > 0 {
			continue
		}

		// Decrypt
		decrypted, err := DecryptCreds(entry.Encrypted, k.encKey)
		if err != nil {
			slog.Warn("ra keymanager: decrypt failed", "provider", provider, "error", err)
			continue
		}

		// Decrement quota
		if entry.Limits.RequestsPerMinute > 0 {
			entry.Limits.RequestsRemaining--
		}

		return string(decrypted), nil
	}

	return "", fmt.Errorf("ra keymanager: all keys exhausted for provider %s", provider)
}

// Rotate marks the current primary key as expired and selects the next one.
func (k *APIKeyManager) Rotate(provider string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	keys, ok := k.keys[provider]
	if !ok || len(keys) == 0 {
		return fmt.Errorf("ra keymanager: no keys to rotate for %s", provider)
	}

	// Mark first non-expired key as expired
	now := time.Now()
	for _, entry := range keys {
		if entry.ExpiresAt == nil || now.Before(*entry.ExpiresAt) {
			entry.ExpiresAt = &now
			slog.Info("ra keymanager: rotated key", "provider", provider)
			return nil
		}
	}

	return fmt.Errorf("ra keymanager: no active keys to rotate for %s", provider)
}

// KeyCount returns the number of keys registered for a provider.
func (k *APIKeyManager) KeyCount(provider string) int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return len(k.keys[provider])
}

// ResetQuotas resets all rate limit counters for a provider.
func (k *APIKeyManager) ResetQuotas(provider string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	for _, entry := range k.keys[provider] {
		entry.Limits.RequestsRemaining = entry.Limits.RequestsPerMinute
		entry.Limits.ResetAt = time.Now().Add(time.Minute)
	}
}
