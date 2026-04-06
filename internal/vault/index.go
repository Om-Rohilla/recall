package vault

import (
	"database/sql"
	"fmt"
)

// hasFTSTable returns true if the given FTS5 virtual table exists in the database.
// Used to gracefully degrade on SQLite builds that lack FTS5 support.
func hasFTSTable(db *sql.DB, table string) bool {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
	).Scan(&count)
	return err == nil && count > 0
}

// RebuildFTSIndex drops and rebuilds the FTS5 index from the commands table.
// Use after bulk imports or if the index becomes inconsistent.
// No-ops gracefully if FTS5 is unavailable (not compiled into this SQLite build).
func (s *Store) RebuildFTSIndex() error {
	if !hasFTSTable(s.db, "commands_fts") {
		return nil // FTS5 not available — degrade gracefully
	}
	if _, err := s.db.Exec("INSERT INTO commands_fts(commands_fts) VALUES('rebuild')"); err != nil {
		return fmt.Errorf("rebuilding commands FTS5 index: %w", err)
	}
	return nil
}

// RebuildKnowledgeFTSIndex rebuilds the knowledge FTS5 index.
// No-ops gracefully if FTS5 is unavailable.
func (s *Store) RebuildKnowledgeFTSIndex() error {
	if !hasFTSTable(s.db, "knowledge_fts") {
		return nil
	}
	if _, err := s.db.Exec("INSERT INTO knowledge_fts(knowledge_fts) VALUES('rebuild')"); err != nil {
		return fmt.Errorf("rebuilding knowledge FTS5 index: %w", err)
	}
	return nil
}

// OptimizeIndex runs FTS5 merge optimization to improve query performance.
// No-ops gracefully if FTS5 is unavailable.
func (s *Store) OptimizeIndex() error {
	if !hasFTSTable(s.db, "commands_fts") {
		return nil
	}
	if _, err := s.db.Exec("INSERT INTO commands_fts(commands_fts, rank) VALUES('merge', 500)"); err != nil {
		return fmt.Errorf("optimizing FTS5 index: %w", err)
	}
	return nil
}

// IntegrityCheck verifies the FTS5 index is consistent with the content table.
// No-ops gracefully if FTS5 is unavailable.
func (s *Store) IntegrityCheck() error {
	if !hasFTSTable(s.db, "commands_fts") {
		return nil
	}
	if _, err := s.db.Exec("INSERT INTO commands_fts(commands_fts) VALUES('integrity-check')"); err != nil {
		return fmt.Errorf("FTS5 integrity check failed: %w", err)
	}
	return nil
}
