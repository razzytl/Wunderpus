package ra

import (
	"testing"
	"time"
)

func TestAPIKeyManager_FreeKeyFirst(t *testing.T) {
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
	}

	mgr := NewAPIKeyManager(encKey)

	// Register 3 keys: paid, free, paid
	mgr.Register("openrouter", "paid-key-1", false, RateLimits{RequestsPerMinute: 100, RequestsRemaining: 100})
	mgr.Register("openrouter", "free-key", true, RateLimits{RequestsPerMinute: 10, RequestsRemaining: 10})
	mgr.Register("openrouter", "paid-key-2", false, RateLimits{RequestsPerMinute: 100, RequestsRemaining: 100})

	// Get should return free key first
	key, err := mgr.Get("openrouter")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if key != "free-key" {
		t.Fatalf("expected free-key first, got %s", key)
	}
}

func TestAPIKeyManager_QuotaExhausted(t *testing.T) {
	encKey := make([]byte, 32)
	mgr := NewAPIKeyManager(encKey)

	mgr.Register("test", "key1", true, RateLimits{RequestsPerMinute: 2, RequestsRemaining: 2})

	// Use up the quota
	mgr.Get("test") // remaining = 1
	mgr.Get("test") // remaining = 0

	// Next get should fail
	_, err := mgr.Get("test")
	if err == nil {
		t.Fatal("should fail when all keys exhausted")
	}
}

func TestAPIKeyManager_Rotate(t *testing.T) {
	encKey := make([]byte, 32)
	mgr := NewAPIKeyManager(encKey)

	// Register both as non-free to avoid sort interference
	mgr.Register("test", "key1", false, RateLimits{RequestsPerMinute: 100, RequestsRemaining: 100})
	mgr.Register("test", "key2", false, RateLimits{RequestsPerMinute: 100, RequestsRemaining: 100})

	// Rotate first key
	mgr.Rotate("test")

	// Force expiration to be clearly in the past
	mgr.mu.Lock()
	for _, entry := range mgr.keys["test"] {
		if entry.ExpiresAt != nil {
			past := time.Now().Add(-1 * time.Hour)
			entry.ExpiresAt = &past
		}
		break
	}
	mgr.mu.Unlock()

	// Get should now return key2 (key1 is expired)
	key, _ := mgr.Get("test")
	if key != "key2" {
		t.Fatalf("expected key2 after rotation, got %s", key)
	}
}

func TestAPIKeyManager_NoKeysRegistered(t *testing.T) {
	encKey := make([]byte, 32)
	mgr := NewAPIKeyManager(encKey)

	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Fatal("should fail for unregistered provider")
	}
}

func TestAPIKeyManager_ResetQuotas(t *testing.T) {
	encKey := make([]byte, 32)
	mgr := NewAPIKeyManager(encKey)

	mgr.Register("test", "key1", true, RateLimits{RequestsPerMinute: 2, RequestsRemaining: 2})

	mgr.Get("test")
	mgr.Get("test")

	// Exhausted
	_, err := mgr.Get("test")
	if err == nil {
		t.Fatal("should be exhausted")
	}

	// Reset
	mgr.ResetQuotas("test")

	// Should work again
	key, err := mgr.Get("test")
	if err != nil {
		t.Fatalf("should work after reset: %v", err)
	}
	if key != "key1" {
		t.Fatalf("expected key1, got %s", key)
	}
}
