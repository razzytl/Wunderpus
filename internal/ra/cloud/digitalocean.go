package cloud

import (
	"fmt"
	"sync"
	"time"

	"github.com/wunderpus/wunderpus/internal/ra"
)

// DigitalOceanAdapter implements ra.CloudAdapter for DigitalOcean.
type DigitalOceanAdapter struct {
	mu             sync.Mutex
	apiToken       string
	maxDailySpend  float64
	registry       *ra.ResourceRegistry
	costToDate     float64
	costResetDate  time.Time // date when costToDate was last reset
	provisionedIDs []string
}

// NewDigitalOceanAdapter creates a DO adapter.
func NewDigitalOceanAdapter(apiToken string, maxDailySpend float64, registry *ra.ResourceRegistry) *DigitalOceanAdapter {
	return &DigitalOceanAdapter{
		apiToken:      apiToken,
		maxDailySpend: maxDailySpend,
		registry:      registry,
		costResetDate: time.Now().UTC(),
	}
}

// resetDailyCostIfNewDay resets the cost counter if the calendar day has changed.
func (d *DigitalOceanAdapter) resetDailyCostIfNewDay() {
	now := time.Now().UTC()
	if now.YearDay() != d.costResetDate.YearDay() || now.Year() != d.costResetDate.Year() {
		d.costToDate = 0
		d.costResetDate = now
	}
}

// ProvisionCompute creates a DigitalOcean Droplet.
func (d *DigitalOceanAdapter) ProvisionCompute(spec ra.ResourceSpec) (ra.Resource, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.resetDailyCostIfNewDay()

	costPerHour := 0.008 // $6/mo ≈ $0.008/hr for s-1vcpu-1gb

	if d.costToDate+costPerHour > d.maxDailySpend {
		return ra.Resource{}, fmt.Errorf("ra cloud: daily spend cap would be exceeded ($%.4f + $%.4f > $%.2f)", d.costToDate, costPerHour, d.maxDailySpend)
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
		CostPerHour: costPerHour,
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
	d.mu.Lock()
	defer d.mu.Unlock()
	d.resetDailyCostIfNewDay()

	costPerHour := 0.005

	if d.costToDate+costPerHour > d.maxDailySpend {
		return ra.Resource{}, fmt.Errorf("ra cloud: daily spend cap would be exceeded ($%.4f + $%.4f > $%.2f)", d.costToDate, costPerHour, d.maxDailySpend)
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
		CostPerHour: costPerHour,
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
	d.mu.Lock()
	defer d.mu.Unlock()

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
	d.mu.Lock()
	defer d.mu.Unlock()

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
	d.mu.Lock()
	defer d.mu.Unlock()
	d.resetDailyCostIfNewDay()
	return d.costToDate, nil
}
