package ra

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// ResourceRegistry provides SQLite-backed persistence for resources.
type ResourceRegistry struct {
	db *sql.DB
}

// NewResourceRegistry opens or creates the resource database.
func NewResourceRegistry(dbPath string) (*ResourceRegistry, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ra registry: opening db: %w", err)
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS resources (
			id           TEXT PRIMARY KEY,
			type         TEXT NOT NULL,
			provider     TEXT NOT NULL,
			capabilities TEXT NOT NULL DEFAULT '{}',
			cost_per_hour REAL NOT NULL DEFAULT 0.0,
			acquired_at  TEXT NOT NULL,
			expires_at   TEXT,
			status       TEXT NOT NULL DEFAULT 'active',
			credentials  BLOB
		);
		CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type);
		CREATE INDEX IF NOT EXISTS idx_resources_status ON resources(status);
		CREATE INDEX IF NOT EXISTS idx_resources_provider ON resources(provider);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ra registry: creating schema: %w", err)
	}

	return &ResourceRegistry{db: db}, nil
}

// Register adds a resource to the registry.
func (r *ResourceRegistry) Register(res Resource) error {
	var expiresAt *string
	if res.ExpiresAt != nil {
		t := res.ExpiresAt.Format(time.RFC3339Nano)
		expiresAt = &t
	}

	capBytes, _ := json.Marshal(res.Capabilities)

	_, err := r.db.Exec(`
		INSERT INTO resources (id, type, provider, capabilities, cost_per_hour, acquired_at, expires_at, status, credentials)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		res.ID, string(res.Type), res.Provider, string(capBytes),
		res.CostPerHour, res.AcquiredAt.Format(time.RFC3339Nano),
		expiresAt, string(res.Status), res.Credentials,
	)
	return err
}

// Get retrieves a resource by ID.
func (r *ResourceRegistry) Get(id string) (Resource, error) {
	var res Resource
	var typeStr, statusStr, acquiredStr string
	var capsStr string
	var expiresStr sql.NullString

	err := r.db.QueryRow(`SELECT id, type, provider, capabilities, cost_per_hour, acquired_at, expires_at, status, credentials FROM resources WHERE id = ?`, id).
		Scan(&res.ID, &typeStr, &res.Provider, &capsStr, &res.CostPerHour, &acquiredStr, &expiresStr, &statusStr, &res.Credentials)
	if err != nil {
		return res, fmt.Errorf("ra registry: get: %w", err)
	}

	res.Type = ResourceType(typeStr)
	res.Status = ResourceStatus(statusStr)
	res.AcquiredAt, _ = time.Parse(time.RFC3339Nano, acquiredStr)
	if expiresStr.Valid {
		t, _ := time.Parse(time.RFC3339Nano, expiresStr.String)
		res.ExpiresAt = &t
	}
	json.Unmarshal([]byte(capsStr), &res.Capabilities)

	return res, nil
}

// ListByType returns all resources of the given type.
func (r *ResourceRegistry) ListByType(t ResourceType) ([]Resource, error) {
	rows, err := r.db.Query(`SELECT id, type, provider, capabilities, cost_per_hour, acquired_at, expires_at, status FROM resources WHERE type = ?`, string(t))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanResources(rows)
}

// ListActive returns all resources with status "active".
func (r *ResourceRegistry) ListActive() ([]Resource, error) {
	return r.ListByStatus(ResourceStatusActive)
}

// UpdateStatus changes a resource's status.
func (r *ResourceRegistry) UpdateStatus(id string, status ResourceStatus) error {
	_, err := r.db.Exec(`UPDATE resources SET status = ? WHERE id = ?`, string(status), id)
	return err
}

// Deregister removes a resource from the registry.
func (r *ResourceRegistry) Deregister(id string) error {
	_, err := r.db.Exec(`DELETE FROM resources WHERE id = ?`, id)
	return err
}

// ListByStatus returns all resources with the given status.
func (r *ResourceRegistry) ListByStatus(status ResourceStatus) ([]Resource, error) {
	rows, err := r.db.Query(`SELECT id, type, provider, capabilities, cost_per_hour, acquired_at, expires_at, status FROM resources WHERE status = ?`, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanResources(rows)
}

// scanResources iterates sql.Rows and returns a slice of Resources.
func (r *ResourceRegistry) scanResources(rows *sql.Rows) ([]Resource, error) {
	var resources []Resource
	for rows.Next() {
		var res Resource
		var typeStr, statusStr, acquiredStr string
		var capsStr string
		var expiresStr sql.NullString

		if err := rows.Scan(&res.ID, &typeStr, &res.Provider, &capsStr, &res.CostPerHour, &acquiredStr, &expiresStr, &statusStr); err != nil {
			return nil, fmt.Errorf("ra registry: scan: %w", err)
		}
		res.Type = ResourceType(typeStr)
		res.Status = ResourceStatus(statusStr)
		res.AcquiredAt, _ = time.Parse(time.RFC3339Nano, acquiredStr)
		if expiresStr.Valid {
			t, _ := time.Parse(time.RFC3339Nano, expiresStr.String)
			res.ExpiresAt = &t
		}
		json.Unmarshal([]byte(capsStr), &res.Capabilities)
		resources = append(resources, res)
	}
	return resources, rows.Err()
}

// TotalCostPerHour returns the sum of cost_per_hour for all active resources.
func (r *ResourceRegistry) TotalCostPerHour() (float64, error) {
	var total sql.NullFloat64
	err := r.db.QueryRow(`SELECT SUM(cost_per_hour) FROM resources WHERE status = 'active'`).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Float64, nil
	}
	return 0, nil
}

// Close closes the database.
func (r *ResourceRegistry) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// InitLocalResources registers the current machine as a local compute resource.
func (r *ResourceRegistry) InitLocalResources() error {
	res := Resource{
		ID:       "local-compute",
		Type:     ResourceCompute,
		Provider: "local",
		Capabilities: map[string]interface{}{
			"type": "local-machine",
		},
		CostPerHour: 0,
		AcquiredAt:  time.Now().UTC(),
		Status:      ResourceStatusActive,
	}
	return r.Register(res)
}
