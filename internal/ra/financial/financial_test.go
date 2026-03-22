package financial

import (
	"context"
	"testing"
)

func TestFinancialAcquisition_DisabledByDefault(t *testing.T) {
	config := FinancialConfig{Enabled: false}
	fa := NewFinancialAcquisition(config)

	if fa.IsEnabled() {
		t.Fatal("should be disabled by default")
	}

	_, err := fa.CreatePaymentLink(context.Background(), StripeConfig{})
	if err != ErrFinancialAcquisitionDisabled {
		t.Fatalf("expected ErrFinancialAcquisitionDisabled, got %v", err)
	}

	_, err = fa.ScanBounties(context.Background(), []string{"go"})
	if err != ErrFinancialAcquisitionDisabled {
		t.Fatalf("expected ErrFinancialAcquisitionDisabled, got %v", err)
	}

	_, err = fa.SubmitBounty(context.Background(), Bounty{}, "solution")
	if err != ErrFinancialAcquisitionDisabled {
		t.Fatalf("expected ErrFinancialAcquisitionDisabled, got %v", err)
	}
}

func TestFinancialAcquisition_EnabledOperations(t *testing.T) {
	config := FinancialConfig{Enabled: true}
	fa := NewFinancialAcquisition(config)

	if !fa.IsEnabled() {
		t.Fatal("should be enabled")
	}

	link, err := fa.CreatePaymentLink(context.Background(), StripeConfig{
		PaymentLinkURL: "https://pay.example.com",
	})
	if err != nil {
		t.Fatalf("CreatePaymentLink: %v", err)
	}
	if link != "https://pay.example.com" {
		t.Fatalf("expected URL, got %s", link)
	}

	bounties, err := fa.ScanBounties(context.Background(), []string{"go", "rust"})
	if err != nil {
		t.Fatalf("ScanBounties: %v", err)
	}
	if bounties == nil {
		t.Fatal("should return empty slice, not nil")
	}

	result, err := fa.SubmitBounty(context.Background(), Bounty{ID: "b1", Title: "Fix bug"}, "code fix")
	if err != nil {
		t.Fatalf("SubmitBounty: %v", err)
	}
	if !result.Success {
		t.Fatal("submission should succeed")
	}
}
