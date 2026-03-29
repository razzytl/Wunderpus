package worldmodel

import "time"

// EntityType classifies entities in the knowledge graph.
type EntityType string

const (
	EntityPerson       EntityType = "Person"
	EntityOrganization EntityType = "Organization"
	EntityProduct      EntityType = "Product"
	EntityAPI          EntityType = "API"
	EntityTechnology   EntityType = "Technology"
	EntityMarket       EntityType = "Market"
	EntityEvent        EntityType = "Event"
	EntityConcept      EntityType = "Concept"
	EntityFile         EntityType = "File"
	EntityURL          EntityType = "URL"
	EntityPrice        EntityType = "Price"
	EntityTool         EntityType = "Tool"
)

// Entity is a node in the knowledge graph.
type Entity struct {
	ID         string                 `json:"id"`
	Type       EntityType             `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Confidence float64                `json:"confidence"`
	Source     string                 `json:"source"` // which task produced this fact
	IsDynamic  bool                   `json:"is_dynamic"`
}

// Relation is an edge between two entities.
type Relation struct {
	ID         string                 `json:"id"`
	FromEntity string                 `json:"from_entity"`
	ToEntity   string                 `json:"to_entity"`
	RelType    string                 `json:"rel_type"`
	Properties map[string]interface{} `json:"properties"`
	Confidence float64                `json:"confidence"`
	CreatedAt  time.Time              `json:"created_at"`
}

// QueryResult holds the output of a graph query.
type QueryResult struct {
	Entities   []Entity   `json:"entities"`
	Relations  []Relation `json:"relations"`
	Path       []Entity   `json:"path,omitempty"` // for path queries
	Confidence float64    `json:"confidence"`
	Answer     string     `json:"answer,omitempty"` // for NL queries
}

// ExtractedFact is what the knowledge extractor produces.
type ExtractedFact struct {
	Entities  []EntityInput   `json:"entities"`
	Relations []RelationInput `json:"relations"`
}

// EntityInput is used for creating/updating entities.
type EntityInput struct {
	Name       string                 `json:"name"`
	Type       EntityType             `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// RelationInput is used for creating/updating relations.
type RelationInput struct {
	From       string                 `json:"from"`
	To         string                 `json:"to"`
	RelType    string                 `json:"rel_type"`
	Properties map[string]interface{} `json:"properties"`
}

// SourceQuality defines confidence levels based on data source.
const (
	ConfidenceDirectAPI     = 0.95 // Direct API data
	ConfidenceLLMAuthority  = 0.80 // LLM extraction from authoritative source
	ConfidenceUserStatement = 0.70 // User statement
	ConfidenceLLMInference  = 0.60 // LLM inference
)

// ConfidenceForSource returns the confidence score for a given source quality.
func ConfidenceForSource(sourceType string) float64 {
	switch sourceType {
	case "api":
		return ConfidenceDirectAPI
	case "authoritative":
		return ConfidenceLLMAuthority
	case "user":
		return ConfidenceUserStatement
	case "inference":
		return ConfidenceLLMInference
	default:
		return ConfidenceLLMInference
	}
}
