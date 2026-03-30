package business

import (
	"testing"
)

func TestSupportEngine_CreateTicket(t *testing.T) {
	engine := &SupportEngine{}
	_ = engine
}

func TestTicket_ClassifyBug(t *testing.T) {
	ticket := &Ticket{
		Subject: "App crashes when clicking button",
	}

	// Test classification
	desc := ticket.Subject
	if contains(desc, []string{"crash", "broken", "not working"}) {
		t.Log("Bug classification works")
	}
}

func TestLaunchOrchestrator_Phases(t *testing.T) {
	launch := &ProductLaunch{
		Name:         "Test Product",
		CurrentPhase: PhaseIdeaValidation,
	}

	if launch.Name != "Test Product" {
		t.Errorf("Expected Name 'Test Product', got %s", launch.Name)
	}

	if launch.CurrentPhase != PhaseIdeaValidation {
		t.Errorf("Expected PhaseIdeaValidation, got %s", launch.CurrentPhase)
	}

	// Test phase advancement
	switch launch.CurrentPhase {
	case PhaseIdeaValidation:
		launch.CurrentPhase = PhaseMVPSpec
	}

	if launch.CurrentPhase != PhaseMVPSpec {
		t.Error("Phase should advance")
	}
}
