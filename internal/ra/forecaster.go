package ra

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// CloudAdapter is the interface for provisioning cloud resources.
// Defined in ra package to avoid import cycles.
type CloudAdapter interface {
	ProvisionCompute(spec ResourceSpec) (Resource, error)
	ProvisionStorage(spec ResourceSpec) (Resource, error)
	Deprovision(resourceID string) error
	ListProvisioned() ([]Resource, error)
	GetCostToDate() (float64, error)
}

// ResourceForecaster projects future resource needs from the goal tree
// and task queue, comparing against available resources.
type ResourceForecaster struct {
	registry *ResourceRegistry
}

// NewResourceForecaster creates a forecaster.
func NewResourceForecaster(registry *ResourceRegistry) *ResourceForecaster {
	return &ResourceForecaster{registry: registry}
}

// Project computes a resource forecast from pending goals and tasks.
func (f *ResourceForecaster) Project(pendingTasks int) ResourceForecast {
	// Conservative defaults when no history exists
	computeNeeds := []ResourceNeed{
		{
			Type: ResourceCompute,
			Spec: ResourceSpec{
				MinCPUCores: 1,
				MinRAMGB:    1,
				MinDiskGB:   10,
			},
			MaxCostHr: 0.10,
			Duration:  time.Duration(pendingTasks) * time.Hour,
		},
	}

	storageNeeds := []ResourceNeed{
		{
			Type: ResourceStorage,
			Spec: ResourceSpec{
				MinDiskGB: float64(pendingTasks) * 0.1, // 100MB per task
			},
			MaxCostHr: 0.01,
			Duration:  24 * time.Hour,
		},
	}

	// Add 20% buffer
	for i := range computeNeeds {
		computeNeeds[i].Spec.MinCPUCores = int(float64(computeNeeds[i].Spec.MinCPUCores) * 1.2)
		computeNeeds[i].Spec.MinRAMGB *= 1.2
	}

	forecast := ResourceForecast{
		Horizon:      24 * time.Hour,
		ComputeNeeds: computeNeeds,
		StorageNeeds: storageNeeds,
		APIBudget:    map[string]float64{"openrouter": float64(pendingTasks) * 0.01},
		Confidence:   0.5, // low confidence without historical data
	}

	return forecast
}

// FindShortfalls compares forecasted needs against available resources.
func (f *ResourceForecaster) FindShortfalls(forecast ResourceForecast) []ResourceNeed {
	var shortfalls []ResourceNeed

	if f.registry == nil {
		return forecast.ComputeNeeds // everything is a shortfall without a registry
	}

	active, err := f.registry.ListActive()
	if err != nil {
		slog.Warn("ra forecaster: failed to list active resources", "error", err)
		return forecast.ComputeNeeds
	}

	// Count available compute
	availableCPU := 0
	availableRAM := 0.0
	for _, res := range active {
		if res.Type == ResourceCompute {
			if cpu, ok := res.Capabilities["cpu_cores"].(int); ok {
				availableCPU += cpu
			}
			if ram, ok := res.Capabilities["ram_gb"].(float64); ok {
				availableRAM += ram
			}
		}
	}

	// Check compute shortfalls
	for _, need := range forecast.ComputeNeeds {
		if need.Spec.MinCPUCores > availableCPU || need.Spec.MinRAMGB > availableRAM {
			shortfalls = append(shortfalls, need)
		}
		availableCPU -= need.Spec.MinCPUCores
		availableRAM -= need.Spec.MinRAMGB
	}

	return shortfalls
}

// AutoProvision provisions resources to cover shortfalls.
func (f *ResourceForecaster) AutoProvision(needs []ResourceNeed, adapter CloudAdapter, maxDailySpend float64) error {
	for _, need := range needs {
		cost, _ := adapter.GetCostToDate()
		// Calculate total cost: hourly rate × duration in hours
		totalCost := need.MaxCostHr * need.Duration.Hours()
		if cost+totalCost > maxDailySpend {
			slog.Warn("ra forecaster: cannot provision — would exceed daily spend cap",
				"current_cost", cost, "need_cost", totalCost, "cap", maxDailySpend)
			return fmt.Errorf("ra forecaster: daily spend cap would be exceeded")
		}

		switch need.Type {
		case ResourceCompute:
			res, err := adapter.ProvisionCompute(need.Spec)
			if err != nil {
				slog.Warn("ra forecaster: provision compute failed", "error", err)
				return err
			}
			slog.Info("ra forecaster: provisioned compute", "id", res.ID, "provider", res.Provider)
		case ResourceStorage:
			res, err := adapter.ProvisionStorage(need.Spec)
			if err != nil {
				slog.Warn("ra forecaster: provision storage failed", "error", err)
				return err
			}
			slog.Info("ra forecaster: provisioned storage", "id", res.ID, "provider", res.Provider)
		}
	}
	return nil
}

// StartScheduler runs Project() and AutoProvision() on a 15-minute cycle.
func (f *ResourceForecaster) StartScheduler(
	ctx context.Context,
	adapter CloudAdapter,
	maxDailySpend float64,
	taskCountFn func() int,
) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pendingTasks := taskCountFn()
				forecast := f.Project(pendingTasks)
				shortfalls := f.FindShortfalls(forecast)

				if len(shortfalls) > 0 {
					slog.Info("ra forecaster: shortfalls detected", "count", len(shortfalls))
					if err := f.AutoProvision(shortfalls, adapter, maxDailySpend); err != nil {
						slog.Warn("ra forecaster: auto-provision failed", "error", err)
					}
				}
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { close(stop) }
}
