package provider

import (
	"testing"
	"time"
)

// TestResponseCache_Get_Empty tests retrieving from empty cache
func TestResponseCache_Get_Empty(t *testing.T) {
	cache := NewResponseCache(time.Hour)
	req := &CompletionRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
		},
	}

	resp, found := cache.Get(req)
	if found {
		t.Error("expected not found for empty cache")
	}
	if resp != nil {
		t.Error("expected nil response for empty cache")
	}
}

// TestResponseCache_SetAndGet tests basic set and get operations
func TestResponseCache_SetAndGet(t *testing.T) {
	cache := NewResponseCache(time.Hour)
	req := &CompletionRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
		},
	}
	resp := &CompletionResponse{
		Content: "Hello!",
	}

	cache.Set(req, resp)

	got, found := cache.Get(req)
	if !found {
		t.Error("expected to find cached response")
	}
	if got == nil {
		t.Fatal("expected non-nil response")
	}
	if got.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got %q", got.Content)
	}
}

// TestResponseCache_Expiration tests that cached entries expire
func TestResponseCache_Expiration(t *testing.T) {
	cache := NewResponseCache(time.Millisecond) // Very short TTL
	req := &CompletionRequest{
		Model: "test-model",
	}
	resp := &CompletionResponse{Content: "test"}

	cache.Set(req, resp)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	got, found := cache.Get(req)
	if found {
		t.Error("expected entry to be expired")
	}
	if got != nil {
		t.Error("expected nil response for expired entry")
	}
}

// TestResponseCache_DifferentRequests tests that different requests get different cache entries
func TestResponseCache_DifferentRequests(t *testing.T) {
	cache := NewResponseCache(time.Hour)

	req1 := &CompletionRequest{Model: "model1", Messages: []Message{{Role: RoleUser, Content: "hello"}}}
	req2 := &CompletionRequest{Model: "model2", Messages: []Message{{Role: RoleUser, Content: "hello"}}}

	resp1 := &CompletionResponse{Content: "response1"}
	resp2 := &CompletionResponse{Content: "response2"}

	cache.Set(req1, resp1)
	cache.Set(req2, resp2)

	got1, _ := cache.Get(req1)
	got2, _ := cache.Get(req2)

	if got1.Content != "response1" {
		t.Errorf("expected response1, got %q", got1.Content)
	}
	if got2.Content != "response2" {
		t.Errorf("expected response2, got %q", got2.Content)
	}
}

// TestResponseCache_ConcurrentAccess tests concurrent read/write
func TestResponseCache_ConcurrentAccess(t *testing.T) {
	cache := NewResponseCache(time.Hour)
	done := make(chan bool)

	// Writer
	go func() {
		for i := 0; i < 100; i++ {
			req := &CompletionRequest{Model: "model", Messages: []Message{{Role: RoleUser, Content: "test"}}}
			resp := &CompletionResponse{Content: "test"}
			cache.Set(req, resp)
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 100; i++ {
			req := &CompletionRequest{Model: "model", Messages: []Message{{Role: RoleUser, Content: "test"}}}
			cache.Get(req)
		}
		done <- true
	}()

	<-done
	<-done
}

// TestResponseCache_SameRequestDifferentMessages tests different messages in same request
func TestResponseCache_SameRequestDifferentMessages(t *testing.T) {
	cache := NewResponseCache(time.Hour)

	req1 := &CompletionRequest{
		Model: "model",
		Messages: []Message{
			{Role: RoleUser, Content: "hello"},
		},
	}
	req2 := &CompletionRequest{
		Model: "model",
		Messages: []Message{
			{Role: RoleUser, Content: "goodbye"},
		},
	}

	resp1 := &CompletionResponse{Content: "response1"}
	resp2 := &CompletionResponse{Content: "response2"}

	cache.Set(req1, resp1)
	cache.Set(req2, resp2)

	got1, _ := cache.Get(req1)
	got2, _ := cache.Get(req2)

	if got1.Content == got2.Content {
		t.Error("expected different cache entries for different messages")
	}
}

// TestResponseCache_HashConsistency tests that the same request produces the same hash
func TestResponseCache_HashConsistency(t *testing.T) {
	cache := NewResponseCache(time.Hour)

	req1 := &CompletionRequest{Model: "model", Messages: []Message{{Role: RoleUser, Content: "test"}}}
	req2 := &CompletionRequest{Model: "model", Messages: []Message{{Role: RoleUser, Content: "test"}}}

	resp := &CompletionResponse{Content: "response"}
	cache.Set(req1, resp)

	got, found := cache.Get(req2)
	if !found {
		t.Error("expected to find cached response with identical request")
	}
	if got.Content != "response" {
		t.Errorf("expected 'response', got %q", got.Content)
	}
}
