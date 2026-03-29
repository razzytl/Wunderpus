package worldmodel

import (
	"fmt"
	"log/slog"
	"strings"
)

// LLMCaller abstracts LLM calls for the world model.
type LLMCaller interface {
	Complete(req LLMRequest) (string, error)
}

// LLMRequest is a simplified LLM request for world model operations.
type LLMRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}

// QueryInterface provides natural language querying of the world model.
// It converts NL questions to graph queries via LLM, executes them,
// and returns structured results with confidence scores.
type QueryInterface struct {
	store *Store
	llm   LLMCaller
}

// NewQueryInterface creates a new query interface.
func NewQueryInterface(store *Store, llm LLMCaller) *QueryInterface {
	return &QueryInterface{store: store, llm: llm}
}

// Ask converts a natural language question to a graph query and executes it.
func (q *QueryInterface) Ask(question string) (*QueryResult, error) {
	slog.Info("worldmodel: NL query", "question", question)

	if q.llm == nil {
		// Fallback: search entities by keywords from the question
		return q.searchByKeywords(question)
	}

	// Use LLM to convert NL to cypher-like query
	cypherQuery, err := q.nlToCypher(question)
	if err != nil {
		slog.Warn("worldmodel: NL to cypher failed, falling back to keyword search", "error", err)
		return q.searchByKeywords(question)
	}

	slog.Debug("worldmodel: generated query", "cypher", cypherQuery)

	// Execute the query
	result, err := q.store.Query(cypherQuery)
	if err != nil {
		return nil, fmt.Errorf("worldmodel: query execution: %w", err)
	}

	// Generate natural language answer
	if q.llm != nil && len(result.Entities) > 0 {
		answer, err := q.generateAnswer(question, result)
		if err == nil {
			result.Answer = answer
		}
	}

	return result, nil
}

// Context returns the most relevant entities for a given task description.
// This is used to inject world model context into LLM prompts.
func (q *QueryInterface) Context(taskDescription string) ([]Entity, error) {
	// Search for entities matching task keywords
	entities, err := q.store.SearchEntities(taskDescription, 10)
	if err != nil {
		return nil, err
	}

	// Also search by type if we can infer it
	keywords := extractKeywords(taskDescription)
	for _, kw := range keywords {
		more, err := q.store.SearchEntities(kw, 5)
		if err == nil {
			entities = append(entities, more...)
		}
	}

	// Deduplicate and sort by confidence
	seen := make(map[string]bool)
	var result []Entity
	for _, e := range entities {
		if !seen[e.ID] {
			seen[e.ID] = true
			result = append(result, e)
		}
	}

	return result, nil
}

// nlToCypher converts a natural language question to a cypher-like query.
func (q *QueryInterface) nlToCypher(question string) (string, error) {
	resp, err := q.llm.Complete(LLMRequest{
		SystemPrompt: nlToCypherSystemPrompt,
		UserPrompt:   fmt.Sprintf("Convert this question to a graph query: %s", question),
		Temperature:  0.1,
		MaxTokens:    200,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp), nil
}

// generateAnswer creates a natural language answer from query results.
func (q *QueryInterface) generateAnswer(question string, result *QueryResult) (string, error) {
	var context strings.Builder
	context.WriteString("Entities found:\n")
	for _, e := range result.Entities {
		context.WriteString(fmt.Sprintf("- %s (%s): confidence %.2f\n", e.Name, e.Type, e.Confidence))
	}
	if len(result.Relations) > 0 {
		context.WriteString("\nRelations found:\n")
		for _, r := range result.Relations {
			context.WriteString(fmt.Sprintf("- %s -> %s (%s)\n", r.FromEntity, r.ToEntity, r.RelType))
		}
	}

	resp, err := q.llm.Complete(LLMRequest{
		SystemPrompt: "Answer the question based on the provided knowledge graph data. Be concise and cite confidence scores.",
		UserPrompt:   fmt.Sprintf("Question: %s\n\nData:\n%s", question, context.String()),
		Temperature:  0.3,
		MaxTokens:    500,
	})
	if err != nil {
		return "", err
	}

	return resp, nil
}

// searchByKeywords does a simple keyword-based search when LLM is unavailable.
func (q *QueryInterface) searchByKeywords(question string) (*QueryResult, error) {
	keywords := extractKeywords(question)

	var allEntities []Entity
	seen := make(map[string]bool)

	for _, kw := range keywords {
		entities, err := q.store.SearchEntities(kw, 10)
		if err != nil {
			continue
		}
		for _, e := range entities {
			if !seen[e.ID] {
				seen[e.ID] = true
				allEntities = append(allEntities, e)
			}
		}
	}

	var totalConf float64
	for _, e := range allEntities {
		totalConf += e.Confidence
	}
	conf := 0.0
	if len(allEntities) > 0 {
		conf = totalConf / float64(len(allEntities))
	}

	return &QueryResult{
		Entities:   allEntities,
		Confidence: conf,
	}, nil
}

const nlToCypherSystemPrompt = `You are a graph query generator. Convert natural language questions to simplified cypher-like queries.

Syntax:
- MATCH (a:Type) WHERE a.name = "X" RETURN a
- MATCH (a:Type)-[:RELATION]->(b:Type) WHERE a.name = "X" RETURN a, b

Entity types: Person, Organization, Product, API, Technology, Market, Event, Concept, File, URL, Price, Tool

Return ONLY the query string. No explanation.`

// extractKeywords pulls meaningful words from a string for search.
func extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"what": true, "who": true, "where": true, "when": true, "how": true,
		"do": true, "does": true, "did": true, "can": true, "will": true,
		"about": true, "for": true, "of": true, "in": true, "to": true,
		"and": true, "or": true, "not": true, "with": true, "from": true,
	}

	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:")
		if len(w) > 2 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}
	return keywords
}
