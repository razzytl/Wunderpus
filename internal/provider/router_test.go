package provider

import (
	"context"
	"testing"
	"time"
)

type testProvider struct {
	name string
}

func (p *testProvider) Name() string { return p.name }

func (p *testProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{Content: "success"}, nil
}

func (p *testProvider) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	ch <- StreamChunk{Content: "test", Done: true}
	close(ch)
	return ch, nil
}

func TestRouterCreation(t *testing.T) {
	t.Run("set_active_provider", func(t *testing.T) {
		router := &Router{
			providers: map[string]Provider{
				"test": &testProvider{name: "test"},
			},
		}

		err := router.SetActive("test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if router.activeName != "test" {
			t.Errorf("expected test, got %s", router.activeName)
		}
	})

	t.Run("set_nonexistent_provider", func(t *testing.T) {
		router := &Router{
			providers: map[string]Provider{
				"test": &testProvider{name: "test"},
			},
		}

		err := router.SetActive("nonexistent")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestRouterList(t *testing.T) {
	router := &Router{
		providers: map[string]Provider{
			"test1": &testProvider{name: "test1"},
			"test2": &testProvider{name: "test2"},
		},
	}

	names := router.List()
	if len(names) != 2 {
		t.Errorf("expected 2, got %d", len(names))
	}
}

func TestRouterStats(t *testing.T) {
	t.Run("record_success", func(t *testing.T) {
		router := &Router{
			stats: map[string]*ProviderStats{
				"test": {FailCount: 5},
			},
		}

		router.recordSuccess("test", 100*time.Millisecond)

		if router.stats["test"].FailCount != 0 {
			t.Errorf("expected 0, got %d", router.stats["test"].FailCount)
		}
		if len(router.stats["test"].Latencies) != 1 {
			t.Errorf("expected 1 latency, got %d", len(router.stats["test"].Latencies))
		}
	})

	t.Run("record_failure", func(t *testing.T) {
		router := &Router{
			stats: map[string]*ProviderStats{
				"test": {},
			},
		}

		router.recordFailure("test")

		if router.stats["test"].FailCount != 1 {
			t.Errorf("expected 1, got %d", router.stats["test"].FailCount)
		}
		if router.stats["test"].LastFailure.IsZero() {
			t.Error("expected LastFailure to be set")
		}
	})
}

func TestRouterIsHealthy(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		router := &Router{
			stats: map[string]*ProviderStats{
				"test": {FailCount: 0},
			},
		}

		if !router.isHealthy("test") {
			t.Error("expected router to be healthy")
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		router := &Router{
			stats: map[string]*ProviderStats{
				"test": {FailCount: 10, LastFailure: time.Now()},
			},
		}

		if router.isHealthy("test") {
			t.Error("expected router to be unhealthy")
		}
	})

	t.Run("no_stats", func(t *testing.T) {
		router := &Router{
			stats: map[string]*ProviderStats{},
		}

		if !router.isHealthy("nonexistent") {
			t.Error("expected router without stats to be healthy")
		}
	})
}

func TestRouterProviderAccess(t *testing.T) {
	router := &Router{
		providers: map[string]Provider{
			"test": &testProvider{name: "test"},
		},
	}

	t.Run("get_existing", func(t *testing.T) {
		p := router.Get("test")
		if p == nil {
			t.Error("expected non-nil provider")
		}
	})

	t.Run("get_nonexisting", func(t *testing.T) {
		p := router.Get("nonexistent")
		if p != nil {
			t.Error("expected nil provider")
		}
	})
}

func TestRouterCompleteWithFallback(t *testing.T) {
	t.Run("successful_completion", func(t *testing.T) {
		provider := &testProvider{name: "test"}

		router := &Router{
			providers: map[string]Provider{
				"test": provider,
			},
			activeName:     "test",
			activeProvider: provider,
			stats:          map[string]*ProviderStats{},
		}

		resp, err := router.CompleteWithFallback(context.Background(), &CompletionRequest{
			Messages: []Message{{Role: RoleUser, Content: "test"}},
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp.Content != "success" {
			t.Errorf("expected success, got %s", resp.Content)
		}
	})
}

func TestGetProviderHealth(t *testing.T) {
	router := &Router{
		stats: map[string]*ProviderStats{
			"test1": {FailCount: 5, LastFailure: time.Now()},
			"test2": {FailCount: 0},
		},
		cooldown: NewCooldownTracker(),
	}

	health := router.GetProviderHealth()

	if len(health) != 2 {
		t.Errorf("expected 2 providers, got %d", len(health))
	}
	if health["test1"]["fail_count"].(int) != 5 {
		t.Errorf("expected fail_count 5, got %v", health["test1"]["fail_count"])
	}
	if health["test2"]["fail_count"].(int) != 0 {
		t.Errorf("expected fail_count 0, got %v", health["test2"]["fail_count"])
	}
}
