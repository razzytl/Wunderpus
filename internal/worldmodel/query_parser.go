package worldmodel

import (
	"fmt"
	"regexp"
	"strings"
)

// parsedQuery holds the parsed components of a cypher-like query.
type parsedQuery struct {
	StartType   string // entity type filter for start node
	StartName   string // name filter for start node
	RelType     string // relation type filter
	EndType     string // entity type filter for end node
	ReturnStart bool   // return start node
	ReturnEnd   bool   // return end node
	MaxDepth    int    // max traversal depth
}

// parseCypherQuery parses a simplified cypher-like query string.
// Supported: MATCH (a:Type)-[:REL]->(b:Type) WHERE a.name = "X" RETURN a, b
func parseCypherQuery(query string) (*parsedQuery, error) {
	q := &parsedQuery{
		MaxDepth:    5,
		ReturnStart: true,
		ReturnEnd:   true,
	}

	query = strings.TrimSpace(query)

	// Parse MATCH clause
	matchRe := regexp.MustCompile(`MATCH\s+\((\w+)(?::(\w+))?\)(?:\s*-\s*\[\s*:(\w+)\s*\]\s*->\s*\((\w+)(?::(\w+))?\))?`)
	matches := matchRe.FindStringSubmatch(query)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid MATCH clause: %s", query)
	}

	// matches[1] = start alias, [2] = start type, [3] = rel type, [4] = end alias, [5] = end type
	if len(matches) > 2 && matches[2] != "" {
		q.StartType = matches[2]
	}
	if len(matches) > 3 && matches[3] != "" {
		q.RelType = matches[3]
	}
	if len(matches) > 5 && matches[5] != "" {
		q.EndType = matches[5]
	}

	// Parse WHERE clause for name filter
	whereRe := regexp.MustCompile(`WHERE\s+(\w+)\.name\s*=\s*"([^"]+)"`)
	whereMatches := whereRe.FindStringSubmatch(query)
	if len(whereMatches) >= 3 {
		q.StartName = whereMatches[2]
	}

	return q, nil
}

// executeParsedQuery runs a parsed query against the store.
func (s *Store) executeParsedQuery(q *parsedQuery) (*QueryResult, error) {
	result := &QueryResult{}

	// Build entity query
	var entityQuery strings.Builder
	var args []any

	entityQuery.WriteString(`SELECT id, type, name, properties, created_at, updated_at, confidence, source, is_dynamic FROM entities WHERE 1=1`)

	if q.StartType != "" {
		entityQuery.WriteString(` AND type = ?`)
		args = append(args, q.StartType)
	}
	if q.StartName != "" {
		entityQuery.WriteString(` AND name = ?`)
		args = append(args, q.StartName)
	}

	rows, err := s.db.Query(entityQuery.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result.Entities, _ = s.scanEntities(rows)

	// If there's a relation traversal, find connected entities
	if q.RelType != "" && len(result.Entities) > 0 {
		var allConnected []Entity
		var allRelations []Relation

		for _, entity := range result.Entities {
			relQuery := `SELECT id, from_entity, to_entity, rel_type, properties, confidence, created_at
				FROM relations WHERE from_entity = ?`
			relArgs := []any{entity.ID}

			if q.RelType != "" {
				relQuery += ` AND rel_type = ?`
				relArgs = append(relArgs, q.RelType)
			}

			relRows, err := s.db.Query(relQuery, relArgs...)
			if err != nil {
				continue
			}
			defer relRows.Close()

			relations, _ := s.scanRelations(relRows)
			allRelations = append(allRelations, relations...)

			// Load connected entities
			for _, rel := range relations {
				connected, err := s.GetEntity(rel.ToEntity)
				if err == nil {
					// Filter by end type if specified
					if q.EndType == "" || string(connected.Type) == q.EndType {
						allConnected = append(allConnected, *connected)
					}
				}
			}
		}

		result.Entities = append(result.Entities, allConnected...)
		result.Relations = allRelations
	}

	// Calculate aggregate confidence
	var totalConf float64
	for _, e := range result.Entities {
		totalConf += e.Confidence
	}
	if len(result.Entities) > 0 {
		result.Confidence = totalConf / float64(len(result.Entities))
	}

	return result, nil
}
