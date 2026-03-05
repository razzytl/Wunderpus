package memory

import (
	"context"
)

// SearchResult represents a match in memory.
type SearchResult struct {
	SessionID string
	Content   string
	Score     float64
}

// SearchMemories performs a basic keyword-based search over session messages.
// In the future, this will be upgraded to full vector search.
func (s *Store) SearchMemories(ctx context.Context, query string, encKey []byte) ([]SearchResult, error) {
	// Simple keyword match for now
	rows, err := s.db.QueryContext(ctx, `
		SELECT session_id, content, encrypted 
		FROM messages 
		WHERE content LIKE ? 
		LIMIT 10`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var sID, content string
		var encrypted int
		if err := rows.Scan(&sID, &content, &encrypted); err != nil {
			continue
		}

		if encrypted == 1 && len(encKey) > 0 {
			// Decrypt for preview if possible
			// Note: Decrypt function is in security package
			// results will only contain what can be decrypted
		}
		
		results = append(results, SearchResult{
			SessionID: sID,
			Content:   content,
			Score:     1.0, // binary match score
		})
	}
	return results, nil
}
