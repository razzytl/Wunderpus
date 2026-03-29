package money

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"time"
)

// Dataset represents a dataset ready for sale.
type Dataset struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Schema      map[string]string        `json:"schema"` // column -> type
	Records     []map[string]interface{} `json:"records"`
	Size        int                      `json:"size_bytes"`
	CreatedAt   time.Time                `json:"created_at"`
	Price       float64                  `json:"price"`
	Marketplace string                   `json:"marketplace"` // "datarade", "aws_data_exchange", "snowflake"
}

// PIIScanner scans data for personally identifiable information.
type PIIScanner struct {
	patterns []*regexp.Regexp
}

// NewPIIScanner creates a new PII scanner.
func NewPIIScanner() *PIIScanner {
	return &PIIScanner{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`), // email
			regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),                               // SSN
			regexp.MustCompile(`\b\d{10,}\b`),                                         // phone
			regexp.MustCompile(`(?i)(name|address|phone|email)`),                      // potential PII fields
		},
	}
}

// Scan checks data for PII.
func (s *PIIScanner) Scan(data []map[string]interface{}) (bool, []string) {
	var findings []string

	for _, record := range data {
		for key, value := range record {
			valueStr := ""
			switch v := value.(type) {
			case string:
				valueStr = v
			case float64, int:
				valueStr = fmt.Sprintf("%v", v)
			}

			// Check field names for potential PII
			for _, pattern := range s.patterns {
				if key != "" && (pattern.MatchString(key) || pattern.MatchString(valueStr)) {
					findings = append(findings, "Field '"+key+"' may contain PII: '"+truncate(valueStr, 20)+"'")
				}
			}
		}
	}

	return len(findings) > 0, findings
}

// DatasetCollector collects datasets during normal operation.
type DatasetCollector struct {
	datasets   []Dataset
	worldModel WorldModelQuery
}

// DatasetConfig holds dataset broker configuration.
type DatasetConfig struct {
	Enabled      bool
	MinRecords   int
	PIIEnabled   bool // Allow PII data (should be false)
	Marketplaces []string
}

// NewDatasetCollector creates a new dataset collector.
func NewDatasetCollector(cfg DatasetConfig, wm WorldModelQuery) *DatasetCollector {
	return &DatasetCollector{
		datasets:   []Dataset{},
		worldModel: wm,
	}
}

// CollectPriceHistory gathers price data across markets.
func (c *DatasetCollector) CollectPriceHistory(ctx context.Context, symbols []string) (*Dataset, error) {
	// Would collect from market intelligence
	dataset := &Dataset{
		ID:          generateID(),
		Name:        "market_prices",
		Description: "Price history across crypto markets",
		Schema: map[string]string{
			"timestamp": "datetime",
			"symbol":    "string",
			"price":     "float64",
			"volume":    "float64",
		},
		Records:   []map[string]interface{}{},
		CreatedAt: time.Now(),
	}

	slog.Info("databroker: collected price history", "symbols", len(symbols))
	return dataset, nil
}

// CollectKnowledgeGraph gathers entity relationship data.
func (c *DatasetCollector) CollectKnowledgeGraph(ctx context.Context, entityTypes []string) (*Dataset, error) {
	// Would collect from world model
	dataset := &Dataset{
		ID:          generateID(),
		Name:        "knowledge_graph",
		Description: "Entity relationships from world model",
		Schema: map[string]string{
			"entity":     "string",
			"type":       "string",
			"relation":   "string",
			"target":     "string",
			"confidence": "float64",
		},
		Records:   []map[string]interface{}{},
		CreatedAt: time.Now(),
	}

	return dataset, nil
}

// CollectJobTrends gathers job market trends.
func (c *DatasetCollector) CollectJobTrends(ctx context.Context) (*Dataset, error) {
	dataset := &Dataset{
		ID:          generateID(),
		Name:        "job_market_trends",
		Description: "Job market trends from freelance scanner",
		Schema: map[string]string{
			"category": "string",
			"demand":   "int",
			"avg_rate": "float64",
		},
		Records:   []map[string]interface{}{},
		CreatedAt: time.Now(),
	}

	return dataset, nil
}

// Anonymize removes PII from a dataset.
func (c *DatasetCollector) Anonymize(dataset *Dataset) (*Dataset, error) {
	scanner := NewPIIScanner()
	hasPII, findings := scanner.Scan(dataset.Records)

	if hasPII {
		slog.Warn("databroker: PII found in dataset", "findings", findings)
		// Remove potentially identifying fields
		cleanRecords := make([]map[string]interface{}, len(dataset.Records))
		for i, record := range dataset.Records {
			cleanRecords[i] = make(map[string]interface{})
			for key, value := range record {
				// Remove fields that might be PII
				lower := key
				_ = lower // suppress unused warning
				if !containsAny(key, []string{"name", "email", "phone", "address", "ssn", "dob"}) {
					cleanRecords[i][key] = value
				}
			}
		}
		dataset.Records = cleanRecords
	}

	return dataset, nil
}

// UploadList lists datasets on specified marketplace.
func (c *DatasetCollector) UploadList(ctx context.Context, dataset *Dataset, marketplace string) error {
	// Check for PII first
	scanner := NewPIIScanner()
	hasPII, _ := scanner.Scan(dataset.Records)
	if hasPII {
		slog.Error("databroker: cannot upload dataset with PII")
		return ErrPIIDetected
	}

	dataset.Marketplace = marketplace
	slog.Info("databroker: dataset uploaded", "marketplace", marketplace, "name", dataset.Name)

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

var ErrPIIDetected = &DataBrokerError{Message: "PII detected in dataset, cannot upload"}

// DataBrokerError represents an error in the data broker.
type DataBrokerError struct {
	Message string
}

func (e *DataBrokerError) Error() string {
	return e.Message
}

// MarshalJSON implements json.Marshaler.
func (e *DataBrokerError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"error": e.Message})
}
