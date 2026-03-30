package worldmodel

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Store is a SQLite-backed knowledge graph using adjacency tables.
// It stores entities as nodes and relations as edges, with recursive CTE
// queries for path finding.
type Store struct {
	db       *sql.DB
	eventPub EventPublisher
}

// EventPublisher publishes events on the event bus.
type EventPublisher interface {
	PublishEntityCreated(name string, entityType EntityType)
	PublishRelationCreated(from, to, relType string)
}

// NewStore creates a new knowledge graph store at the given SQLite path.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("worldmodel: open db: %w", err)
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

	schema := `
	CREATE TABLE IF NOT EXISTS entities (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		properties TEXT DEFAULT '{}',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		confidence REAL DEFAULT 1.0,
		source TEXT DEFAULT '',
		is_dynamic INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS relations (
		id TEXT PRIMARY KEY,
		from_entity TEXT NOT NULL,
		to_entity TEXT NOT NULL,
		rel_type TEXT NOT NULL,
		properties TEXT DEFAULT '{}',
		confidence REAL DEFAULT 1.0,
		created_at TEXT NOT NULL,
		FOREIGN KEY (from_entity) REFERENCES entities(id),
		FOREIGN KEY (to_entity) REFERENCES entities(id)
	);

	CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(type);
	CREATE INDEX IF NOT EXISTS idx_entities_name ON entities(name);
	CREATE INDEX IF NOT EXISTS idx_entities_confidence ON entities(confidence);
	CREATE INDEX IF NOT EXISTS idx_relations_from ON relations(from_entity);
	CREATE INDEX IF NOT EXISTS idx_relations_to ON relations(to_entity);
	CREATE INDEX IF NOT EXISTS idx_relations_type ON relations(rel_type);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("worldmodel: create schema: %w", err)
	}

	return &Store{db: db}, nil
}

// SetEventPublisher configures the event publisher for world model events.
func (s *Store) SetEventPublisher(pub EventPublisher) {
	s.eventPub = pub
}

// UpsertEntity creates or updates an entity. If an entity with the same name
// and type exists, it merges properties and updates confidence.
func (s *Store) UpsertEntity(input EntityInput, confidence float64, source string) (*Entity, error) {
	// Check for existing entity
	existing, err := s.findEntityByNameType(input.Name, input.Type)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	now := time.Now()

	if existing != nil {
		// Merge properties and update confidence (weighted average)
		merged := mergeProperties(existing.Properties, input.Properties)
		newConfidence := (existing.Confidence + confidence) / 2

		propsJSON, _ := json.Marshal(merged)
		_, err = s.db.Exec(`
			UPDATE entities SET properties = ?, updated_at = ?, confidence = ?, source = ?
			WHERE id = ?`,
			string(propsJSON), now.Format(time.RFC3339), newConfidence, source, existing.ID)
		if err != nil {
			return nil, fmt.Errorf("worldmodel: update entity: %w", err)
		}

		existing.Properties = merged
		existing.UpdatedAt = now
		existing.Confidence = newConfidence
		existing.Source = source
		return existing, nil
	}

	// Create new entity
	id := uuid.New().String()
	propsJSON, _ := json.Marshal(input.Properties)

	_, err = s.db.Exec(`
		INSERT INTO entities (id, type, name, properties, created_at, updated_at, confidence, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, string(input.Type), input.Name, string(propsJSON),
		now.Format(time.RFC3339), now.Format(time.RFC3339), confidence, source)
	if err != nil {
		return nil, fmt.Errorf("worldmodel: insert entity: %w", err)
	}

	// Publish event
	if s.eventPub != nil {
		s.eventPub.PublishEntityCreated(input.Name, input.Type)
	}

	return &Entity{
		ID:         id,
		Type:       input.Type,
		Name:       input.Name,
		Properties: input.Properties,
		CreatedAt:  now,
		UpdatedAt:  now,
		Confidence: confidence,
		Source:     source,
	}, nil
}

// UpsertRelation creates or updates a relation between two entities.
func (s *Store) UpsertRelation(input RelationInput, confidence float64) (*Relation, error) {
	// Verify entities exist
	fromEntity, err := s.GetEntityByName(input.From)
	if err != nil {
		return nil, fmt.Errorf("worldmodel: from entity %q not found: %w", input.From, err)
	}
	toEntity, err := s.GetEntityByName(input.To)
	if err != nil {
		return nil, fmt.Errorf("worldmodel: to entity %q not found: %w", input.To, err)
	}

	// Check for existing relation
	existing, err := s.findRelation(fromEntity.ID, toEntity.ID, input.RelType)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	now := time.Now()

	if existing != nil {
		// Update confidence
		newConf := (existing.Confidence + confidence) / 2
		_, err = s.db.Exec(`UPDATE relations SET confidence = ? WHERE id = ?`, newConf, existing.ID)
		if err != nil {
			return nil, err
		}
		existing.Confidence = newConf
		return existing, nil
	}

	// Create new relation
	id := uuid.New().String()
	propsJSON, _ := json.Marshal(input.Properties)

	_, err = s.db.Exec(`
		INSERT INTO relations (id, from_entity, to_entity, rel_type, properties, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, fromEntity.ID, toEntity.ID, input.RelType, string(propsJSON),
		confidence, now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("worldmodel: insert relation: %w", err)
	}

	// Publish event
	if s.eventPub != nil {
		s.eventPub.PublishRelationCreated(input.From, input.To, input.RelType)
	}

	return &Relation{
		ID:         id,
		FromEntity: fromEntity.ID,
		ToEntity:   toEntity.ID,
		RelType:    input.RelType,
		Properties: input.Properties,
		Confidence: confidence,
		CreatedAt:  now,
	}, nil
}

// GetEntity retrieves an entity by ID.
func (s *Store) GetEntity(id string) (*Entity, error) {
	var e Entity
	var propsJSON string
	var createdAt, updatedAt string
	var isDynamic int

	err := s.db.QueryRow(`
		SELECT id, type, name, properties, created_at, updated_at, confidence, source, is_dynamic
		FROM entities WHERE id = ?`, id).
		Scan(&e.ID, &e.Type, &e.Name, &propsJSON, &createdAt, &updatedAt, &e.Confidence, &e.Source, &isDynamic)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(propsJSON), &e.Properties)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	e.IsDynamic = isDynamic == 1
	return &e, nil
}

// GetEntityByName retrieves an entity by name (first match).
func (s *Store) GetEntityByName(name string) (*Entity, error) {
	var e Entity
	var propsJSON string
	var createdAt, updatedAt string
	var isDynamic int

	err := s.db.QueryRow(`
		SELECT id, type, name, properties, created_at, updated_at, confidence, source, is_dynamic
		FROM entities WHERE name = ? ORDER BY confidence DESC LIMIT 1`, name).
		Scan(&e.ID, &e.Type, &e.Name, &propsJSON, &createdAt, &updatedAt, &e.Confidence, &e.Source, &isDynamic)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(propsJSON), &e.Properties)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	e.IsDynamic = isDynamic == 1
	return &e, nil
}

// Query executes a cypher-like query against the graph.
// Supported syntax:
//
//	MATCH (a:Person)-[:WORKS_AT]->(b:Organization) WHERE a.name = "John" RETURN a, b
func (s *Store) Query(cypher string) (*QueryResult, error) {
	q, err := parseCypherQuery(cypher)
	if err != nil {
		return nil, fmt.Errorf("worldmodel: parse query: %w", err)
	}

	return s.executeParsedQuery(q)
}

// FindPath finds a path between two entities using recursive CTE.
// Returns up to maxHops hops.
func (s *Store) FindPath(fromName, toName string, maxHops int) (*QueryResult, error) {
	if maxHops <= 0 {
		maxHops = 5
	}

	fromEntity, err := s.GetEntityByName(fromName)
	if err != nil {
		return nil, fmt.Errorf("worldmodel: from entity %q not found: %w", fromName, err)
	}
	toEntity, err := s.GetEntityByName(toName)
	if err != nil {
		return nil, fmt.Errorf("worldmodel: to entity %q not found: %w", toName, err)
	}

	// Recursive CTE for path finding
	query := `
	WITH RECURSIVE path_search(id, path, depth) AS (
		SELECT ?, CAST(? AS TEXT), 0
		UNION ALL
		SELECT r.to_entity,
		       path_search.path || ',' || r.to_entity,
		       path_search.depth + 1
		FROM relations r
		JOIN path_search ON r.from_entity = path_search.id
		WHERE path_search.depth < ?
		  AND path_search.path NOT LIKE '%' || r.to_entity || '%'
	)
	SELECT path, depth FROM path_search WHERE id = ? LIMIT 1;
	`

	var pathStr string
	var depth int
	err = s.db.QueryRow(query, fromEntity.ID, fromEntity.ID, maxHops, toEntity.ID).
		Scan(&pathStr, &depth)
	if err == sql.ErrNoRows {
		return &QueryResult{
			Entities:   []Entity{*fromEntity},
			Confidence: 0,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("worldmodel: find path: %w", err)
	}

	// Load all entities in the path
	ids := strings.Split(pathStr, ",")
	var pathEntities []Entity
	var pathRelations []Relation
	var totalConfidence float64

	for _, id := range ids {
		entity, err := s.GetEntity(id)
		if err == nil {
			pathEntities = append(pathEntities, *entity)
			totalConfidence += entity.Confidence
		}
	}

	// Load relations along the path
	for i := 0; i < len(ids)-1; i++ {
		rel, err := s.findRelation(ids[i], ids[i+1], "")
		if err == nil && rel != nil {
			pathRelations = append(pathRelations, *rel)
		}
	}

	avgConf := totalConfidence / float64(len(pathEntities))
	if len(pathEntities) == 0 {
		avgConf = 0
	}

	return &QueryResult{
		Entities:   pathEntities,
		Relations:  pathRelations,
		Path:       pathEntities,
		Confidence: avgConf,
	}, nil
}

// ApplyConfidenceDecay reduces confidence for entities not seen in 30 days.
// After 30 days, confidence drops by 10% per day.
func (s *Store) ApplyConfidenceDecay() (int, error) {
	cutoff := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)

	res, err := s.db.Exec(`
		UPDATE entities
		SET confidence = MAX(0.0, confidence - 0.1 * (julianday('now') - julianday(updated_at) - 30)),
		    updated_at = ?
		WHERE updated_at < ? AND confidence > 0`,
		time.Now().Format(time.RFC3339), cutoff)
	if err != nil {
		return 0, fmt.Errorf("worldmodel: confidence decay: %w", err)
	}

	count, _ := res.RowsAffected()
	if count > 0 {
		slog.Info("worldmodel: confidence decay applied", "entities_affected", count)
	}
	return int(count), nil
}

// ListEntities returns entities filtered by type, with optional limit.
func (s *Store) ListEntities(entityType EntityType, limit int) ([]Entity, error) {
	query := `SELECT id, type, name, properties, created_at, updated_at, confidence, source, is_dynamic
		FROM entities WHERE type = ? ORDER BY confidence DESC`
	args := []any{string(entityType)}

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanEntities(rows)
}

// SearchEntities searches entities by name substring.
func (s *Store) SearchEntities(nameQuery string, limit int) ([]Entity, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, type, name, properties, created_at, updated_at, confidence, source, is_dynamic
		FROM entities WHERE name LIKE ? ORDER BY confidence DESC LIMIT ?`,
		"%"+nameQuery+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanEntities(rows)
}

// GetRelations returns all relations for an entity.
func (s *Store) GetRelations(entityID string) ([]Relation, error) {
	rows, err := s.db.Query(`
		SELECT id, from_entity, to_entity, rel_type, properties, confidence, created_at
		FROM relations WHERE from_entity = ? OR to_entity = ?`,
		entityID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRelations(rows)
}

// Count returns the total number of entities and relations.
func (s *Store) Count() (entities, relations int, err error) {
	err = s.db.QueryRow("SELECT COUNT(*) FROM entities").Scan(&entities)
	if err != nil {
		return entities, relations, err
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM relations").Scan(&relations)
	return entities, relations, err
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// --- Internal helpers ---

func (s *Store) findEntityByNameType(name string, entityType EntityType) (*Entity, error) {
	var e Entity
	var propsJSON string
	var createdAt, updatedAt string
	var isDynamic int

	err := s.db.QueryRow(`
		SELECT id, type, name, properties, created_at, updated_at, confidence, source, is_dynamic
		FROM entities WHERE name = ? AND type = ?`, name, string(entityType)).
		Scan(&e.ID, &e.Type, &e.Name, &propsJSON, &createdAt, &updatedAt, &e.Confidence, &e.Source, &isDynamic)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(propsJSON), &e.Properties)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	e.IsDynamic = isDynamic == 1
	return &e, nil
}

func (s *Store) findRelation(fromID, toID, relType string) (*Relation, error) {
	var r Relation
	var propsJSON string
	var createdAt string

	var err error
	if relType != "" {
		err = s.db.QueryRow(`
			SELECT id, from_entity, to_entity, rel_type, properties, confidence, created_at
			FROM relations WHERE from_entity = ? AND to_entity = ? AND rel_type = ?`,
			fromID, toID, relType).
			Scan(&r.ID, &r.FromEntity, &r.ToEntity, &r.RelType, &propsJSON, &r.Confidence, &createdAt)
	} else {
		err = s.db.QueryRow(`
			SELECT id, from_entity, to_entity, rel_type, properties, confidence, created_at
			FROM relations WHERE from_entity = ? AND to_entity = ?`,
			fromID, toID).
			Scan(&r.ID, &r.FromEntity, &r.ToEntity, &r.RelType, &propsJSON, &r.Confidence, &createdAt)
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(propsJSON), &r.Properties)
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &r, nil
}

func (s *Store) scanEntities(rows *sql.Rows) ([]Entity, error) {
	var entities []Entity
	for rows.Next() {
		var e Entity
		var propsJSON string
		var createdAt, updatedAt string
		var isDynamic int
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &propsJSON, &createdAt, &updatedAt, &e.Confidence, &e.Source, &isDynamic); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(propsJSON), &e.Properties)
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		e.IsDynamic = isDynamic == 1
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (s *Store) scanRelations(rows *sql.Rows) ([]Relation, error) {
	var relations []Relation
	for rows.Next() {
		var r Relation
		var propsJSON string
		var createdAt string
		if err := rows.Scan(&r.ID, &r.FromEntity, &r.ToEntity, &r.RelType, &propsJSON, &r.Confidence, &createdAt); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(propsJSON), &r.Properties)
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		relations = append(relations, r)
	}
	return relations, rows.Err()
}

// mergeProperties merges two property maps (second overrides first).
func mergeProperties(existing, incoming map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range incoming {
		merged[k] = v
	}
	return merged
}
