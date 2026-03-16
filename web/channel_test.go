package web

import (
	"testing"
)

func TestChannel(t *testing.T) {
	// We use a nil server just to test the wrapper structure
	var srv *Server
	ch := NewChannel(srv)

	if ch.Name() != "web-ui" {
		t.Errorf("Expected Name() to be 'web-ui', got '%s'", ch.Name())
	}

	if ch.GetServer() != nil {
		t.Errorf("Expected GetServer() to be nil, got %v", ch.GetServer())
	}
}
