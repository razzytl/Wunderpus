package ra

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

// ResourceType classifies what kind of resource this is.
type ResourceType string

const (
	ResourceCompute   ResourceType = "compute"
	ResourceStorage   ResourceType = "storage"
	ResourceAPIKey    ResourceType = "api_key"
	ResourceFinancial ResourceType = "financial"
	ResourceData      ResourceType = "data"
)

// ResourceStatus is the lifecycle state of a resource.
type ResourceStatus string

const (
	ResourceStatusActive       ResourceStatus = "active"
	ResourceStatusProvisioning ResourceStatus = "provisioning"
	ResourceStatusExhausted    ResourceStatus = "exhausted"
	ResourceStatusReleased     ResourceStatus = "released"
	ResourceStatusFailed       ResourceStatus = "failed"
)

// Resource represents a provisioned resource managed by the agent.
type Resource struct {
	ID           string                 `json:"id"`
	Type         ResourceType           `json:"type"`
	Provider     string                 `json:"provider"` // aws, gcp, digitalocean, vast_ai, local
	Capabilities map[string]interface{} `json:"capabilities"`
	CostPerHour  float64                `json:"cost_per_hour"`
	AcquiredAt   time.Time              `json:"acquired_at"`
	ExpiresAt    *time.Time             `json:"expires_at"`
	Status       ResourceStatus         `json:"status"`
	Credentials  []byte                 `json:"credentials"` // AES-256-GCM encrypted
}

// ResourceSpec describes the minimum requirements for a resource request.
type ResourceSpec struct {
	Type        ResourceType `json:"type"`
	MinCPUCores int          `json:"min_cpu_cores"`
	MinRAMGB    float64      `json:"min_ram_gb"`
	MinDiskGB   float64      `json:"min_disk_gb"`
	Region      string       `json:"region"`
}

// ResourceRequest is a request to provision a resource.
type ResourceRequest struct {
	Type      ResourceType  `json:"type"`
	MinSpec   ResourceSpec  `json:"min_spec"`
	MaxCostHr float64       `json:"max_cost_hr"`
	Duration  time.Duration `json:"duration"`
	Priority  float64       `json:"priority"`
}

// Credentials holds unencrypted resource credentials.
type Credentials struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	APIKey   string `json:"api_key"`
}

// EncryptCreds encrypts credentials using AES-256-GCM.
func EncryptCreds(creds []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ra: creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ra: creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("ra: generating nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, creds, nil), nil
}

// DecryptCreds decrypts AES-256-GCM encrypted credentials.
func DecryptCreds(encrypted []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ra: creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ra: creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("ra: ciphertext too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// ResourceNeed describes a shortfall between forecasted needs and available resources.
type ResourceNeed struct {
	Type      ResourceType
	Spec      ResourceSpec
	MaxCostHr float64
	Duration  time.Duration
}

// ResourceForecast summarizes projected resource needs.
type ResourceForecast struct {
	Horizon      time.Duration
	ComputeNeeds []ResourceNeed
	StorageNeeds []ResourceNeed
	APIBudget    map[string]float64
	Confidence   float64
}
