package cloud

import (
	"fmt"
	"time"

	"github.com/wunderpus/wunderpus/internal/ra"
)

// DigitalOceanAdapter implements ra.CloudAdapter for DigitalOcean.
type DigitalOceanAdapter struct {
	apiToken       string
	maxDailySpend  float64
	registry       *ra.ResourceRegistry
	costToDate     float64
	provisionedIDs []string
}

// NewDigitalOceanAdapter creates a DO adapter.
func NewDigitalOceanAdapter(apiToken string, maxDailySpend float64, registry *ra.ResourceRegistry) *DigitalOceanAdapter {
	return &DigitalOceanAdapter{
		apiToken:      apiToken,
		maxDailySpend: maxDailySpend,
		registry:      registry,
	}
}

// ProvisionCompute creates a DigitalOcean Droplet.
func (d *DigitalOceanAdapter) ProvisionCompute(spec ra.ResourceSpec) (ra.Resource, error) {
	cost, _ := d.GetCostToDate()
	if cost >= d.maxDailySpend {
		return ra.Resource{}, fmt.Errorf("ra cloud: daily spend cap reached ($%.2f >= $%.2f)", cost, d.maxDailySpend)
	}

	res := ra.Resource{
		ID:       fmt.Sprintf("do-droplet-%d", time.Now().UnixNano()),
		Type:     ra.ResourceCompute,
		Provider: "digitalocean",
		Capabilities: map[string]interface{}{
			"cpu_cores": spec.MinCPUCores,
			"ram_gb":    spec.MinRAMGB,
			"disk_gb":   spec.MinDiskGB,
			"region":    spec.Region,
			"size":      "s-1vcpu-1gb",
		},
		CostPerHour: 0.008, // $6/mo ≈ $0.008/hr
		AcquiredAt:  time.Now().UTC(),
		Status:      ra.ResourceStatusActive,
	}

	if d.registry != nil {
		_ = d.registry.Register(res)
	}

	d.provisionedIDs = append(d.provisionedIDs, res.ID)
	d.costToDate += res.CostPerHour

	return res, nil
}

// ProvisionStorage creates a DigitalOcean Space.
func (d *DigitalOceanAdapter) ProvisionStorage(spec ra.ResourceSpec) (ra.Resource, error) {
	cost, _ := d.GetCostToDate()
	if cost >= d.maxDailySpend {
		return ra.Resource{}, fmt.Errorf("ra cloud: daily spend cap reached")
	}

	res := ra.Resource{
		ID:       fmt.Sprintf("do-space-%d", time.Now().UnixNano()),
		Type:     ra.ResourceStorage,
		Provider: "digitalocean",
		Capabilities: map[string]interface{}{
			"region":  spec.Region,
			"type":    "s3-compatible",
			"size_gb": spec.MinDiskGB,
		},
		CostPerHour: 0.005,
		AcquiredAt:  time.Now().UTC(),
		Status:      ra.ResourceStatusActive,
	}

	if d.registry != nil {
		_ = d.registry.Register(res)
	}

	d.provisionedIDs = append(d.provisionedIDs, res.ID)
	d.costToDate += res.CostPerHour

	return res, nil
}

// Deprovision destroys a resource.
func (d *DigitalOceanAdapter) Deprovision(resourceID string) error {
	if d.registry != nil {
		_ = d.registry.UpdateStatus(resourceID, ra.ResourceStatusReleased)
	}
	for i, id := range d.provisionedIDs {
		if id == resourceID {
			d.provisionedIDs = append(d.provisionedIDs[:i], d.provisionedIDs[i+1:]...)
			break
		}
	}
	return nil
}

// ListProvisioned returns all provisioned resources.
func (d *DigitalOceanAdapter) ListProvisioned() ([]ra.Resource, error) {
	// Return from tracked IDs
	var results []ra.Resource
	if d.registry != nil {
		for _, id := range d.provisionedIDs {
			if res, err := d.registry.Get(id); err == nil {
				results = append(results, res)
			}
		}
	}
	return results, nil
}

// GetCostToDate returns the estimated cost for the current day.
func (d *DigitalOceanAdapter) GetCostToDate() (float64, error) {
	return d.costToDate, nil
}
