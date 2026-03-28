package vault

import "fmt"

// RebuildFTSIndex drops and rebuilds the FTS5 index from the commands table.
// Use after bulk imports or if the index becomes inconsistent.
func (s *Store) RebuildFTSIndex() error {
	if _, err := s.db.Exec("INSERT INTO commands_fts(commands_fts) VALUES('rebuild')"); err != nil {
		return fmt.Errorf("rebuilding commands FTS5 index: %w", err)
	}
	return nil
}

// RebuildKnowledgeFTSIndex rebuilds the knowledge FTS5 index.
func (s *Store) RebuildKnowledgeFTSIndex() error {
	if _, err := s.db.Exec("INSERT INTO knowledge_fts(knowledge_fts) VALUES('rebuild')"); err != nil {
		return fmt.Errorf("rebuilding knowledge FTS5 index: %w", err)
	}
	return nil
}

// OptimizeIndex runs FTS5 merge optimization to improve query performance.
func (s *Store) OptimizeIndex() error {
	if _, err := s.db.Exec("INSERT INTO commands_fts(commands_fts, rank) VALUES('merge', 500)"); err != nil {
		return fmt.Errorf("optimizing FTS5 index: %w", err)
	}
	return nil
}

// IntegrityCheck verifies the FTS5 index is consistent with the content table.
func (s *Store) IntegrityCheck() error {
	if _, err := s.db.Exec("INSERT INTO commands_fts(commands_fts) VALUES('integrity-check')"); err != nil {
		return fmt.Errorf("FTS5 integrity check failed: %w", err)
	}
	return nil
}
