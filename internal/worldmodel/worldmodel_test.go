package worldmodel

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// --- Store Tests ---

func TestStoreCreateAndGetEntity(t *testing.T) {
	store := newTestStore(t)

	entity, err := store.UpsertEntity(
		EntityInput{Name: "Acme Corp", Type: EntityOrganization, Properties: map[string]any{"industry": "tech"}},
		0.9, "test-task",
	)
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	if entity.Name != "Acme Corp" {
		t.Errorf("expected name 'Acme Corp', got %q", entity.Name)
	}
	if entity.Type != EntityOrganization {
		t.Errorf("expected type Organization, got %q", entity.Type)
	}
	if entity.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", entity.Confidence)
	}

	// Get by name
	got, err := store.GetEntityByName("Acme Corp")
	if err != nil {
		t.Fatalf("get by name failed: %v", err)
	}
	if got.ID != entity.ID {
		t.Error("IDs don't match")
	}
}

func TestStoreUpsertMerge(t *testing.T) {
	store := newTestStore(t)

	// First insert
	_, _ = store.UpsertEntity(
		EntityInput{Name: "Google", Type: EntityOrganization, Properties: map[string]any{"founded": "1998"}},
		0.9, "task1",
	)

	// Second insert (should merge)
	merged, err := store.UpsertEntity(
		EntityInput{Name: "Google", Type: EntityOrganization, Properties: map[string]any{"hq": "Mountain View"}},
		0.8, "task2",
	)
	if err != nil {
		t.Fatalf("merge upsert failed: %v", err)
	}

	// Should have both properties
	if merged.Properties["founded"] != "1998" {
		t.Error("original property lost during merge")
	}
	if merged.Properties["hq"] != "Mountain View" {
		t.Error("new property not added during merge")
	}
}

func TestStoreRelation(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "John", Type: EntityPerson}, 0.9, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Acme", Type: EntityOrganization}, 0.9, "test")

	rel, err := store.UpsertRelation(
		RelationInput{From: "John", To: "Acme", RelType: "WORKS_AT"},
		0.85,
	)
	if err != nil {
		t.Fatalf("upsert relation failed: %v", err)
	}

	if rel.RelType != "WORKS_AT" {
		t.Errorf("expected rel_type WORKS_AT, got %q", rel.RelType)
	}
}

func TestStoreRelationMissingEntity(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "John", Type: EntityPerson}, 0.9, "test")

	_, err := store.UpsertRelation(
		RelationInput{From: "John", To: "Unknown Corp", RelType: "WORKS_AT"},
		0.8,
	)
	if err == nil {
		t.Error("expected error for missing entity")
	}
}

func TestStoreFindPath(t *testing.T) {
	store := newTestStore(t)

	// Create a chain: Alice -> Bob -> Charlie -> Dave
	_, _ = store.UpsertEntity(EntityInput{Name: "Alice", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Bob", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Charlie", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Dave", Type: EntityPerson}, 1.0, "test")

	_, _ = store.UpsertRelation(RelationInput{From: "Alice", To: "Bob", RelType: "KNOWS"}, 1.0)
	_, _ = store.UpsertRelation(RelationInput{From: "Bob", To: "Charlie", RelType: "KNOWS"}, 1.0)
	_, _ = store.UpsertRelation(RelationInput{From: "Charlie", To: "Dave", RelType: "KNOWS"}, 1.0)

	result, err := store.FindPath("Alice", "Dave", 5)
	if err != nil {
		t.Fatalf("find path failed: %v", err)
	}

	if len(result.Path) != 4 {
		t.Errorf("expected 4 entities in path, got %d", len(result.Path))
	}
	if result.Path[0].Name != "Alice" {
		t.Errorf("path should start with Alice, got %q", result.Path[0].Name)
	}
	if result.Path[3].Name != "Dave" {
		t.Errorf("path should end with Dave, got %q", result.Path[3].Name)
	}
}

func TestStoreFindPathNoPath(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "Island", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Loner", Type: EntityPerson}, 1.0, "test")

	result, err := store.FindPath("Island", "Loner", 5)
	if err != nil {
		t.Fatalf("find path failed: %v", err)
	}

	if result.Confidence != 0 {
		t.Errorf("expected 0 confidence for no-path result, got %f", result.Confidence)
	}
}

func TestStoreQuery(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "Acme", Type: EntityOrganization}, 0.9, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Google", Type: EntityOrganization}, 0.95, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "John", Type: EntityPerson}, 0.8, "test")

	// Query all organizations
	result, err := store.Query("MATCH (a:Organization) RETURN a")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(result.Entities) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(result.Entities))
	}
}

func TestStoreQueryWithRelation(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "John", Type: EntityPerson}, 0.9, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Acme", Type: EntityOrganization}, 0.9, "test")
	_, _ = store.UpsertRelation(RelationInput{From: "John", To: "Acme", RelType: "WORKS_AT"}, 0.85)

	result, err := store.Query(`MATCH (a:Person)-[:WORKS_AT]->(b:Organization) WHERE a.name = "John" RETURN a, b`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(result.Entities) < 2 {
		t.Errorf("expected at least 2 entities, got %d", len(result.Entities))
	}
	if len(result.Relations) != 1 {
		t.Errorf("expected 1 relation, got %d", len(result.Relations))
	}
}

func TestStoreSearchEntities(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "Google Inc", Type: EntityOrganization}, 0.9, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Alphabet Inc", Type: EntityOrganization}, 0.9, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "John Doe", Type: EntityPerson}, 0.8, "test")

	results, err := store.SearchEntities("Google", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result for 'Google', got %d", len(results))
	}
}

func TestStoreCount(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "A", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "B", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertRelation(RelationInput{From: "A", To: "B", RelType: "KNOWS"}, 1.0)

	entities, relations, err := store.Count()
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if entities != 2 {
		t.Errorf("expected 2 entities, got %d", entities)
	}
	if relations != 1 {
		t.Errorf("expected 1 relation, got %d", relations)
	}
}

func TestStoreConfidenceDecay(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "Old Entity", Type: EntityConcept}, 1.0, "test")

	// Confidence decay won't affect entities < 30 days old
	affected, err := store.ApplyConfidenceDecay()
	if err != nil {
		t.Fatalf("decay failed: %v", err)
	}
	if affected != 0 {
		t.Errorf("expected 0 affected (entity too new), got %d", affected)
	}
}

func TestStoreListEntities(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "A", Type: EntityPerson}, 0.9, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "B", Type: EntityPerson}, 0.8, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "C", Type: EntityOrganization}, 0.95, "test")

	people, err := store.ListEntities(EntityPerson, 10)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(people) != 2 {
		t.Errorf("expected 2 people, got %d", len(people))
	}
}

// --- Query Interface Tests ---

func TestQueryInterfaceContext(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "Python", Type: EntityTechnology}, 0.9, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Django", Type: EntityTechnology}, 0.85, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "Acme Corp", Type: EntityOrganization}, 0.9, "test")

	qi := NewQueryInterface(store, nil)
	entities, err := qi.Context("build a web app with Python and Django")
	if err != nil {
		t.Fatalf("context failed: %v", err)
	}

	if len(entities) == 0 {
		t.Error("expected at least 1 entity from context query")
	}
}

func TestQueryInterfaceKeywords(t *testing.T) {
	store := newTestStore(t)

	_, _ = store.UpsertEntity(EntityInput{Name: "React", Type: EntityTechnology}, 0.9, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "React Native", Type: EntityTechnology}, 0.85, "test")

	qi := NewQueryInterface(store, nil)
	result, err := qi.Ask("What do I know about React?")
	if err != nil {
		t.Fatalf("ask failed: %v", err)
	}

	if len(result.Entities) == 0 {
		t.Error("expected entities matching 'React'")
	}
}

// --- Extractor Tests ---

func TestParseExtractedFact(t *testing.T) {
	response := `{
		"entities": [
			{"name": "Acme Corp", "type": "Organization", "properties": {"industry": "tech"}},
			{"name": "John Smith", "type": "Person", "properties": {"role": "CEO"}}
		],
		"relations": [
			{"from": "John Smith", "to": "Acme Corp", "rel_type": "WORKS_AT", "properties": {}}
		]
	}`

	fact, err := parseExtractedFact(response)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(fact.Entities) != 2 {
		t.Errorf("expected 2 entities, got %d", len(fact.Entities))
	}
	if len(fact.Relations) != 1 {
		t.Errorf("expected 1 relation, got %d", len(fact.Relations))
	}
	if fact.Entities[0].Name != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %q", fact.Entities[0].Name)
	}
}

func TestParseExtractedFactWithFences(t *testing.T) {
	response := "```json\n{\"entities\": [{\"name\": \"Test\", \"type\": \"Concept\", \"properties\": {}}], \"relations\": []}\n```"

	fact, err := parseExtractedFact(response)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(fact.Entities) != 1 {
		t.Errorf("expected 1 entity, got %d", len(fact.Entities))
	}
}

func TestParseExtractedFactDefaults(t *testing.T) {
	// Missing type should default to Concept
	response := `{"entities": [{"name": "Something"}], "relations": []}`

	fact, err := parseExtractedFact(response)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if fact.Entities[0].Type != EntityConcept {
		t.Errorf("expected default type Concept, got %q", fact.Entities[0].Type)
	}
}

// --- Updater Tests ---

func TestUpdaterHandleToolSynthesized(t *testing.T) {
	store := newTestStore(t)
	extractor := NewExtractor(store, nil)
	updater := NewUpdater(store, extractor, nil)

	updater.HandleToolSynthesized("pdf_parser", "Parses PDF files")

	entity, err := store.GetEntityByName("pdf_parser")
	if err != nil {
		t.Fatalf("tool entity not created: %v", err)
	}

	if entity.Type != EntityTool {
		t.Errorf("expected type Tool, got %q", entity.Type)
	}
	if entity.Properties["origin"] != "synthesized" {
		t.Error("expected origin=synthesized")
	}
}

// --- Utility Tests ---

func TestConfidenceForSource(t *testing.T) {
	tests := []struct {
		source   string
		expected float64
	}{
		{"api", 0.95},
		{"authoritative", 0.80},
		{"user", 0.70},
		{"inference", 0.60},
		{"unknown", 0.60},
	}

	for _, tt := range tests {
		got := ConfidenceForSource(tt.source)
		if got != tt.expected {
			t.Errorf("ConfidenceForSource(%q) = %f, want %f", tt.source, got, tt.expected)
		}
	}
}

func TestMergeProperties(t *testing.T) {
	existing := map[string]any{"a": 1, "b": 2}
	incoming := map[string]any{"b": 3, "c": 4}

	merged := mergeProperties(existing, incoming)

	if merged["a"] != 1 {
		t.Error("original property 'a' lost")
	}
	if merged["b"] != 3 {
		t.Error("property 'b' should be overridden by incoming")
	}
	if merged["c"] != 4 {
		t.Error("new property 'c' not added")
	}
}

func TestExtractKeywords(t *testing.T) {
	kw := extractKeywords("What do I know about the Google company?")
	found := false
	for _, k := range kw {
		if k == "google" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'google' in keywords, got %v", kw)
	}
}

// --- Large-scale Test ---

func TestStore1000Entities3HopPath(t *testing.T) {
	store := newTestStore(t)

	// Create 1000 entities
	for i := 0; i < 1000; i++ {
		name := entityNames[i%len(entityNames)]
		etype := entityTypes[i%len(entityTypes)]
		_, _ = store.UpsertEntity(
			EntityInput{
				Name:       name + "_" + itoa(i),
				Type:       etype,
				Properties: map[string]any{"index": i},
			},
			1.0, "bulk-test",
		)
	}

	// Create a 3-hop path: A_0 -> B_1 -> C_2 -> D_3
	_, _ = store.UpsertEntity(EntityInput{Name: "NodeA", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "NodeB", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "NodeC", Type: EntityPerson}, 1.0, "test")
	_, _ = store.UpsertEntity(EntityInput{Name: "NodeD", Type: EntityPerson}, 1.0, "test")

	_, _ = store.UpsertRelation(RelationInput{From: "NodeA", To: "NodeB", RelType: "LINKS"}, 1.0)
	_, _ = store.UpsertRelation(RelationInput{From: "NodeB", To: "NodeC", RelType: "LINKS"}, 1.0)
	_, _ = store.UpsertRelation(RelationInput{From: "NodeC", To: "NodeD", RelType: "LINKS"}, 1.0)

	result, err := store.FindPath("NodeA", "NodeD", 5)
	if err != nil {
		t.Fatalf("3-hop path failed: %v", err)
	}

	if len(result.Path) != 4 {
		t.Errorf("expected 4-node path, got %d", len(result.Path))
	}

	entities, relations, _ := store.Count()
	t.Logf("Store has %d entities, %d relations", entities, relations)
}

// --- Test Helpers ---

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return store
}

var (
	entityNames = []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta"}
	entityTypes = []EntityType{EntityPerson, EntityOrganization, EntityTechnology, EntityConcept, EntityProduct}
)

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
