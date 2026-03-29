package a2a

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Test Helpers ---

func newGetRequest(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	return req
}

func newPostRequest(path, body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
}

func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func newTestServer(routes map[string]func(http.ResponseWriter, *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := routes[r.URL.Path]; ok {
			handler(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))
}

// --- Registry Tests ---

func TestRegistryRegister(t *testing.T) {
	reg := NewRegistry()
	reg.Register(AgentCard{Name: "agent1", Capabilities: []string{"research"}})

	card, ok := reg.Get("agent1")
	if !ok {
		t.Fatal("agent not found")
	}
	if card.Name != "agent1" {
		t.Errorf("expected 'agent1', got %q", card.Name)
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(AgentCard{Name: "a1"})
	reg.Register(AgentCard{Name: "a2"})
	reg.Register(AgentCard{Name: "a3"})

	list := reg.List()
	if len(list) != 3 {
		t.Errorf("expected 3 agents, got %d", len(list))
	}
}

func TestRegistryUnregister(t *testing.T) {
	reg := NewRegistry()
	reg.Register(AgentCard{Name: "temp"})
	reg.Unregister("temp")

	_, ok := reg.Get("temp")
	if ok {
		t.Error("agent should be removed")
	}
}

func TestRegistryFindByCapability(t *testing.T) {
	reg := NewRegistry()
	reg.Register(AgentCard{Name: "researcher", Capabilities: []string{"research", "analysis"}})
	reg.Register(AgentCard{Name: "coder", Capabilities: []string{"code", "debug"}})
	reg.Register(AgentCard{Name: "analyst", Capabilities: []string{"analysis", "data"}})

	matches := reg.FindByCapability("analysis")
	if len(matches) != 2 {
		t.Errorf("expected 2 agents with 'analysis', got %d", len(matches))
	}
}

// --- Server Tests ---

func TestServerHandleCard(t *testing.T) {
	card := AgentCard{
		Name: "test-agent", Description: "A test agent",
		Capabilities: []string{"test"}, Endpoint: "localhost:0", Version: "1.0",
	}
	server := NewServer(card, func(task Task) (*TaskResult, error) {
		return &TaskResult{Status: TaskCompleted, Output: "done"}, nil
	})

	req := newGetRequest("/a2a/card")
	w := newRecorder()
	server.handleCard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resultCard AgentCard
	json.NewDecoder(w.Body).Decode(&resultCard)
	if resultCard.Name != "test-agent" {
		t.Errorf("expected 'test-agent', got %q", resultCard.Name)
	}
}

func TestServerHandleTask(t *testing.T) {
	card := AgentCard{Name: "test-agent", Endpoint: "localhost:0"}
	executed := false
	server := NewServer(card, func(task Task) (*TaskResult, error) {
		executed = true
		return &TaskResult{Status: TaskCompleted, Output: "task completed"}, nil
	})

	body, _ := json.Marshal(Task{ID: "task-1", Description: "Test task"})
	req := newPostRequest("/a2a/task", string(body))
	w := newRecorder()
	server.handleTask(w, req)

	if !executed {
		t.Error("handler should have been called")
	}
	var result TaskResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Status != TaskCompleted {
		t.Errorf("expected completed, got %s", result.Status)
	}
	if result.AgentID != "test-agent" {
		t.Errorf("expected agent ID 'test-agent', got %q", result.AgentID)
	}
}

func TestServerHandleTaskFailure(t *testing.T) {
	card := AgentCard{Name: "failing-agent", Endpoint: "localhost:0"}
	server := NewServer(card, func(task Task) (*TaskResult, error) {
		return nil, fmt.Errorf("something went wrong")
	})

	body, _ := json.Marshal(Task{ID: "fail-1", Description: "Will fail"})
	req := newPostRequest("/a2a/task", string(body))
	w := newRecorder()
	server.handleTask(w, req)

	var result TaskResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Status != TaskFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestServerHandleCardWrongMethod(t *testing.T) {
	server := NewServer(AgentCard{}, nil)
	req := newPostRequest("/a2a/card", "")
	w := newRecorder()
	server.handleCard(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- Client Tests ---

func TestClientDiscoverCard(t *testing.T) {
	card := AgentCard{Name: "remote-agent", Capabilities: []string{"research"}, Version: "1.0"}
	ts := newTestServer(map[string]func(http.ResponseWriter, *http.Request){
		"/a2a/card": func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(card) },
	})
	defer ts.Close()

	client := NewClient()
	discovered, err := client.DiscoverCard(ts.URL)
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if discovered.Name != "remote-agent" {
		t.Errorf("expected 'remote-agent', got %q", discovered.Name)
	}
}

func TestClientAssignTask(t *testing.T) {
	ts := newTestServer(map[string]func(http.ResponseWriter, *http.Request){
		"/a2a/task": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(TaskResult{TaskID: "task-1", Status: TaskCompleted, Output: "result data"})
		},
	})
	defer ts.Close()

	client := NewClient()
	result, err := client.AssignTask(ts.URL, Task{ID: "task-1", Description: "Test"})
	if err != nil {
		t.Fatalf("assign failed: %v", err)
	}
	if result.Output != "result data" {
		t.Errorf("expected 'result data', got %q", result.Output)
	}
}

// --- Type Tests ---

func TestAgentCardJSON(t *testing.T) {
	card := AgentCard{Name: "test", Description: "desc", Capabilities: []string{"cap1"}, Endpoint: "http://localhost:8080", Version: "1.0"}
	data, _ := json.Marshal(card)
	var decoded AgentCard
	json.Unmarshal(data, &decoded)
	if decoded.Name != card.Name {
		t.Error("round-trip failed")
	}
}
