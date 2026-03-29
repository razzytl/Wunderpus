package worldmodel

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Extractor extracts entities and relations from text using LLM calls.
// After every task completion, it runs extraction on the task's outputs
// and episodic memory entries.
type Extractor struct {
	store    *Store
	llm      LLMCaller
	embedder Embedder
}

// Embedder generates embedding vectors for text (for deduplication).
type Embedder interface {
	EmbedSingle(text string) ([]float64, error)
}

// NewExtractor creates a new knowledge extractor.
func NewExtractor(store *Store, llm LLMCaller) *Extractor {
	return &Extractor{store: store, llm: llm}
}

// SetEmbedder configures the embedder for semantic deduplication.
func (e *Extractor) SetEmbedder(embedder Embedder) {
	e.embedder = embedder
}

// Extract runs knowledge extraction on the given text.
// It identifies entities and relations, assigns confidence based on source quality,
// and stores them in the world model.
func (e *Extractor) Extract(text string, sourceType string, taskID string) (*ExtractedFact, error) {
	if e.llm == nil {
		return nil, fmt.Errorf("extractor: LLM not configured")
	}

	slog.Info("worldmodel: extracting knowledge", "sourceType", sourceType, "textLen", len(text))

	// LLM extraction call
	response, err := e.llm.Complete(LLMRequest{
		SystemPrompt: extractorSystemPrompt,
		UserPrompt:   fmt.Sprintf("Extract entities and relations from this text:\n\n%s", text),
		Temperature:  0.2,
		MaxTokens:    2000,
	})
	if err != nil {
		return nil, fmt.Errorf("extractor: LLM call failed: %w", err)
	}

	// Parse the extracted fact
	fact, err := parseExtractedFact(response)
	if err != nil {
		return nil, fmt.Errorf("extractor: parse response: %w", err)
	}

	// Assign confidence based on source quality
	confidence := ConfidenceForSource(sourceType)

	// Store entities
	storedCount := 0
	for _, entityInput := range fact.Entities {
		if entityInput.Name == "" {
			continue
		}

		// Semantic deduplication: check if a similar entity already exists
		if e.embedder != nil {
			if deduped := e.findSemanticallySimilar(entityInput.Name, entityInput.Type); deduped != nil {
				// Merge properties into existing entity instead of creating duplicate
				slog.Debug("worldmodel: dedup entity via embedding",
					"new", entityInput.Name, "existing", deduped.Name)
				entityInput.Properties = mergeProperties(deduped.Properties, entityInput.Properties)
			}
		}

		// Ensure properties exist
		if entityInput.Properties == nil {
			entityInput.Properties = make(map[string]interface{})
		}
		entityInput.Properties["extracted_from"] = taskID
		entityInput.Properties["extracted_at"] = time.Now().Format(time.RFC3339)

		_, err := e.store.UpsertEntity(entityInput, confidence, taskID)
		if err != nil {
			slog.Warn("worldmodel: failed to store entity", "name", entityInput.Name, "error", err)
			continue
		}
		storedCount++
	}

	// Store relations
	for _, relInput := range fact.Relations {
		if relInput.From == "" || relInput.To == "" {
			continue
		}

		if relInput.Properties == nil {
			relInput.Properties = make(map[string]interface{})
		}

		_, err := e.store.UpsertRelation(relInput, confidence)
		if err != nil {
			slog.Debug("worldmodel: relation not stored (entity missing?)",
				"from", relInput.From, "to", relInput.To, "error", err)
			continue
		}
	}

	slog.Info("worldmodel: extraction complete",
		"entities", storedCount,
		"relations", len(fact.Relations),
		"confidence", confidence)

	return fact, nil
}

// ExtractFromTask is a convenience method for extracting from a completed task.
func (e *Extractor) ExtractFromTask(taskDescription, taskOutput, taskID string) (*ExtractedFact, error) {
	combined := fmt.Sprintf("Task: %s\n\nOutput:\n%s", taskDescription, taskOutput)
	return e.Extract(combined, "inference", taskID)
}

const extractorSystemPrompt = `You are a knowledge extraction system. Extract entities and relations from text.

Respond with ONLY valid JSON matching this schema:
{
  "entities": [
    {
      "name": "Entity Name",
      "type": "Person|Organization|Product|API|Technology|Market|Event|Concept|File|URL|Price|Tool",
      "properties": {"key": "value"}
    }
  ],
  "relations": [
    {
      "from": "Entity Name 1",
      "to": "Entity Name 2",
      "rel_type": "WORKS_AT|CREATES|USES|OWNS|COMPETES_WITH|PART_OF|LOCATED_IN|RELATED_TO",
      "properties": {}
    }
  ]
}

Be thorough but precise. Only extract facts explicitly stated or strongly implied.
Output ONLY the JSON. No markdown, no explanation.`

// parseExtractedFact parses the LLM extraction response into an ExtractedFact.
func parseExtractedFact(response string) (*ExtractedFact, error) {
	// Strip markdown code fences
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) >= 2 {
			endIdx := len(lines) - 1
			for i := len(lines) - 1; i > 0; i-- {
				if strings.TrimSpace(lines[i]) == "```" {
					endIdx = i
					break
				}
			}
			response = strings.Join(lines[1:endIdx], "\n")
		}
	}
	response = strings.TrimSpace(response)

	var fact ExtractedFact
	if err := json.Unmarshal([]byte(response), &fact); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate: entities must have name and type
	for i, e := range fact.Entities {
		if e.Name == "" {
			return nil, fmt.Errorf("entity %d has empty name", i)
		}
		if e.Type == "" {
			fact.Entities[i].Type = EntityConcept // default type
		}
	}

	// Validate: relations must have from, to, rel_type
	for i, r := range fact.Relations {
		if r.From == "" || r.To == "" {
			return nil, fmt.Errorf("relation %d has empty from/to", i)
		}
		if r.RelType == "" {
			fact.Relations[i].RelType = "RELATED_TO"
		}
	}

	return &fact, nil
}

// findSemanticallySimilar checks if an existing entity is semantically similar
// (embedding cosine > 0.9) to the given name. Returns the matching entity or nil.
func (e *Extractor) findSemanticallySimilar(name string, entityType EntityType) *Entity {
	if e.embedder == nil {
		return nil
	}

	// Get embedding for the new entity name
	newEmbedding, err := e.embedder.EmbedSingle(name)
	if err != nil {
		return nil
	}

	// Get existing entities of the same type
	existing, err := e.store.ListEntities(entityType, 100)
	if err != nil {
		return nil
	}

	// Check cosine similarity
	for _, entity := range existing {
		existingEmbedding, err := e.embedder.EmbedSingle(entity.Name)
		if err != nil {
			continue
		}
		sim := cosineSimilarity(newEmbedding, existingEmbedding)
		if sim > 0.9 {
			return &entity
		}
	}

	return nil
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	guess := x / 2
	for i := 0; i < 20; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}
