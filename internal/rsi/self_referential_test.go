package rsi

import (
	"strings"
	"testing"
)

func TestSelfReferentialFirewall_BlockedWhenDisabled(t *testing.T) {
	config := DefaultSelfReferentialConfig() // enabled=false
	fw := NewSelfReferentialFirewall(config)

	// Diff targeting internal/rsi/ should be blocked
	diff := `--- a/internal/rsi/fitness.go
+++ b/internal/rsi/fitness.go
@@ -5,3 +5,3 @@
-func Score() float64 {
-	return 0.05
+func Score() float64 {
+	return 0.10
}
`
	err := fw.CheckDiff(diff)
	if err == nil {
		t.Fatal("should block RSI self-modification when disabled")
	}
	fwErr, ok := err.(*FirewallError)
	if !ok {
		t.Fatalf("expected FirewallError, got %T", err)
	}
	if !fwErr.Blocked {
		t.Fatal("should be blocked")
	}
}

func TestSelfReferentialFirewall_AllowedWhenEnabled(t *testing.T) {
	config := SelfReferentialConfig{
		Enabled:             true,
		MinSuccessfulCycles: 10,
		MaxRecentRollbacks:  0,
	}
	fw := NewSelfReferentialFirewall(config)

	// Diff targeting internal/rsi/ should pass when enabled
	diff := `--- a/internal/rsi/fitness.go
+++ b/internal/rsi/fitness.go
@@ -5,3 +5,3 @@
-func Score() float64 {
-	return 0.05
+func Score() float64 {
+	return 0.10
}
`
	err := fw.CheckDiff(diff)
	if err != nil {
		t.Fatalf("should allow RSI self-modification when enabled: %v", err)
	}
}

func TestSelfReferentialFirewall_CmdAlwaysBlocked(t *testing.T) {
	config := SelfReferentialConfig{Enabled: true} // even when enabled
	fw := NewSelfReferentialFirewall(config)

	// cmd/ is ALWAYS blocked, even with self-referential RSI enabled
	diff := `--- a/cmd/wunderpus/main.go
+++ b/cmd/wunderpus/main.go
@@ -1,3 +1,3 @@
-func main() {
+func main() {
 }
`
	err := fw.CheckDiff(diff)
	if err == nil {
		t.Fatal("cmd/ should ALWAYS be blocked")
	}
}

func TestSelfReferentialFirewall_FirewallItselfBlocked(t *testing.T) {
	config := SelfReferentialConfig{Enabled: true}
	fw := NewSelfReferentialFirewall(config)

	diff := `--- a/internal/rsi/self_referential.go
+++ b/internal/rsi/self_referential.go
@@ -1,1 +1,1 @@
-old
+new
`
	err := fw.CheckDiff(diff)
	if err == nil {
		t.Fatal("firewall itself should ALWAYS be blocked")
	}
	if !strings.Contains(err.Error(), "firewall itself") {
		t.Fatalf("expected firewall-itself error, got: %v", err)
	}
}

func TestSelfReferentialFirewall_OutsideInternalBlocked(t *testing.T) {
	config := SelfReferentialConfig{Enabled: true}
	fw := NewSelfReferentialFirewall(config)

	diff := `--- a/pkg/other/file.go
+++ b/pkg/other/file.go
@@ -1,1 +1,1 @@
-old
+new
`
	err := fw.CheckDiff(diff)
	if err == nil {
		t.Fatal("paths outside internal/ should be blocked")
	}
}

func TestIsEligibleForSelfReferential(t *testing.T) {
	config := DefaultSelfReferentialConfig()

	// Not enough cycles
	if IsEligibleForSelfReferential(config, 5, 0) {
		t.Fatal("should not be eligible with only 5 cycles (need 10)")
	}

	// Enough cycles, no rollbacks
	if !IsEligibleForSelfReferential(config, 10, 0) {
		t.Fatal("should be eligible with 10 cycles and 0 rollbacks")
	}

	// Enough cycles but has rollbacks
	if IsEligibleForSelfReferential(config, 10, 1) {
		t.Fatal("should not be eligible with rollbacks")
	}

	// Already enabled
	config.Enabled = true
	if !IsEligibleForSelfReferential(config, 0, 0) {
		t.Fatal("should always be eligible when already enabled")
	}
}
