package edge

import (
	"runtime"
	"testing"
)

func TestEdgeMode_AutoDetect(t *testing.T) {
	cfg := MinimalModeConfig{
		Enabled:    true,
		AutoDetect: true,
	}

	edge := NewMinimalMode(cfg)

	if edge == nil {
		t.Error("Expected edge mode to be created")
	}

	if edge.MaxMemoryMB <= 0 {
		t.Errorf("Expected positive memory, got %d", edge.MaxMemoryMB)
	}
}

func TestEdgeMode_CheckFeatureAvailable(t *testing.T) {
	edge := &EdgeMode{
		Enabled:       true,
		WASSandbox:    false,
		VisionEnabled: false,
	}

	// When in edge mode, features should be limited
	if edge.CheckFeatureAvailable("sandbox") {
		t.Error("Expected sandbox to be disabled in edge mode")
	}
}

func TestDetectEnvironment(t *testing.T) {
	env := DetectEnvironment()

	if env.Platform != runtime.GOOS {
		t.Errorf("Expected platform %s, got %s", runtime.GOOS, env.Platform)
	}

	// Should detect CPU cores
	if env.CPUCores <= 0 {
		t.Error("Expected positive CPU count")
	}
}

func TestOllamaClient_SelectModel(t *testing.T) {
	client := &OllamaClient{}

	model := client.SelectModel(nil, "simple")
	if model != "qwen2.5-3b" {
		t.Errorf("Expected simple model selection, got %s", model)
	}

	model = client.SelectModel(nil, "reasoning")
	if model != "llama3.1-8b" {
		t.Errorf("Expected reasoning model selection, got %s", model)
	}
}

func TestLocalLLMEngine_Fallback(t *testing.T) {
	engine := &LocalLLMEngine{
		fallback: FallbackChain{
			models: []string{"model1", "model2"},
			index:  0,
		},
	}

	if engine.fallback.current() != "model1" {
		t.Error("Expected first model in fallback chain")
	}

	engine.fallback.Fallback()
	if engine.fallback.current() != "model2" {
		t.Error("Expected second model after fallback")
	}
}
