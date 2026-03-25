package cloud

import (
	"testing"

	"github.com/wunderpus/wunderpus/internal/ra"
)

func tempCloudDB(t *testing.T) string {
	return t.TempDir() + "/test_cloud.db"
}

func TestDigitalOceanAdapter_ProvisionAndDeprovision(t *testing.T) {
	reg, _ := ra.NewResourceRegistry(tempCloudDB(t))
	defer reg.Close()

	adapter := NewDigitalOceanAdapter("test-token", 10.0, reg)

	// Provision compute
	res, err := adapter.ProvisionCompute(ra.ResourceSpec{
		MinCPUCores: 1,
		MinRAMGB:    1,
		MinDiskGB:   25,
		Region:      "nyc1",
	})
	if err != nil {
		t.Fatalf("ProvisionCompute: %v", err)
	}
	if res.Type != ra.ResourceCompute {
		t.Fatalf("expected compute, got %s", res.Type)
	}
	if res.Provider != "digitalocean" {
		t.Fatalf("expected digitalocean, got %s", res.Provider)
	}

	// Registry should show the resource
	got, err := reg.Get(res.ID)
	if err != nil {
		t.Fatalf("registry.Get: %v", err)
	}
	if got.Status != ra.ResourceStatusActive {
		t.Fatalf("expected active, got %s", got.Status)
	}

	// ListProvisioned should return it
	list, _ := adapter.ListProvisioned()
	if len(list) != 1 {
		t.Fatalf("expected 1 provisioned, got %d", len(list))
	}

	// Deprovision
	err = adapter.Deprovision(res.ID)
	if err != nil {
		t.Fatalf("Deprovision: %v", err)
	}

	// Registry should show released
	got, _ = reg.Get(res.ID)
	if got.Status != ra.ResourceStatusReleased {
		t.Fatalf("expected released after deprovision, got %s", got.Status)
	}

	// ListProvisioned should be empty
	list, _ = adapter.ListProvisioned()
	if len(list) != 0 {
		t.Fatalf("expected 0 provisioned after deprovision, got %d", len(list))
	}
}

func TestDigitalOceanAdapter_ProvisionStorage(t *testing.T) {
	reg, _ := ra.NewResourceRegistry(tempCloudDB(t))
	defer reg.Close()

	adapter := NewDigitalOceanAdapter("test-token", 10.0, reg)

	res, err := adapter.ProvisionStorage(ra.ResourceSpec{
		MinDiskGB: 100,
		Region:    "nyc3",
	})
	if err != nil {
		t.Fatalf("ProvisionStorage: %v", err)
	}
	if res.Type != ra.ResourceStorage {
		t.Fatalf("expected storage, got %s", res.Type)
	}
}

func TestDigitalOceanAdapter_SpendCap(t *testing.T) {
	reg, _ := ra.NewResourceRegistry(tempCloudDB(t))
	defer reg.Close()

	// Set cap at $0.01 — one provision costs $0.008, second should fail
	adapter := NewDigitalOceanAdapter("test-token", 0.01, reg)

	// First provision should succeed
	_, err := adapter.ProvisionCompute(ra.ResourceSpec{MinCPUCores: 1})
	if err != nil {
		t.Fatalf("first provision should succeed: %v", err)
	}

	// Second provision should exceed cap ($0.008 + $0.008 = $0.016 > $0.01)
	_, err = adapter.ProvisionCompute(ra.ResourceSpec{MinCPUCores: 1})
	if err == nil {
		t.Fatal("second provision should fail — spend cap exceeded")
	}
}

func TestDigitalOceanAdapter_GetCostToDate(t *testing.T) {
	adapter := NewDigitalOceanAdapter("test-token", 100.0, nil)

	cost, _ := adapter.GetCostToDate()
	if cost != 0 {
		t.Fatalf("initial cost should be 0, got %f", cost)
	}

	adapter.ProvisionCompute(ra.ResourceSpec{MinCPUCores: 1})

	cost, _ = adapter.GetCostToDate()
	if cost <= 0 {
		t.Fatal("cost should be > 0 after provisioning")
	}
}
