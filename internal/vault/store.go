package vault

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db   *sql.DB
	path string
}

func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating vault directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening vault database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	return &Store{db: db, path: dbPath}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) InsertCommand(cmd *Command) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	var existingID int64
	err := s.db.QueryRow("SELECT id FROM commands WHERE raw = ?", cmd.Raw).Scan(&existingID)
	if err == nil {
		if _, err := s.db.Exec(
			"UPDATE commands SET frequency = frequency + 1, last_seen = ?, last_exit = ?, avg_duration = ? WHERE id = ?",
			now, cmd.LastExit, cmd.AvgDuration, existingID,
		); err != nil {
			return 0, fmt.Errorf("updating command frequency: %w", err)
		}
		return existingID, nil
	}

	if cmd.FirstSeen.IsZero() {
		cmd.FirstSeen = time.Now().UTC()
	}
	if cmd.LastSeen.IsZero() {
		cmd.LastSeen = time.Now().UTC()
	}

	result, err := s.db.Exec(
		`INSERT INTO commands (raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cmd.Raw, cmd.Binary, cmd.Subcommand, cmd.Flags, cmd.Category,
		max(cmd.Frequency, 1),
		cmd.FirstSeen.Format(time.RFC3339),
		cmd.LastSeen.Format(time.RFC3339),
		cmd.LastExit, cmd.AvgDuration,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting command: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting inserted command id: %w", err)
	}

	return id, nil
}

func (s *Store) InsertContext(ctx *Context) error {
	if ctx.Timestamp.IsZero() {
		ctx.Timestamp = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO contexts (command_id, cwd, git_repo, git_branch, project_type, timestamp, exit_code, duration_ms, session_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ctx.CommandID, ctx.Cwd, ctx.GitRepo, ctx.GitBranch, ctx.ProjectType,
		ctx.Timestamp.Format(time.RFC3339),
		ctx.ExitCode, ctx.DurationMs, ctx.SessionID,
	)
	if err != nil {
		return fmt.Errorf("inserting context: %w", err)
	}
	return nil
}

func (s *Store) SearchFTS5(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	ftsQuery := buildFTSQuery(query)

	rows, err := s.db.Query(
		`SELECT c.id, c.raw, c.binary_name, c.subcommand, c.flags, c.category,
		        c.frequency, c.first_seen, c.last_seen, c.last_exit, c.avg_duration,
		        rank
		 FROM commands_fts fts
		 JOIN commands c ON c.id = fts.rowid
		 WHERE commands_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`,
		ftsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("FTS5 search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var firstSeen, lastSeen string
		var rank float64
		err := rows.Scan(
			&r.Command.ID, &r.Command.Raw, &r.Command.Binary,
			&r.Command.Subcommand, &r.Command.Flags, &r.Command.Category,
			&r.Command.Frequency, &firstSeen, &lastSeen,
			&r.Command.LastExit, &r.Command.AvgDuration, &rank,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		r.Command.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
		r.Command.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
		r.Score = -rank // FTS5 rank is negative (lower = better)
		r.Confidence = scoreToConfidence(r.Score, r.Command.Frequency)
		r.MatchType = "vault"
		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) GetCommand(id int64) (*Command, error) {
	var cmd Command
	var firstSeen, lastSeen string
	err := s.db.QueryRow(
		`SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration
		 FROM commands WHERE id = ?`, id,
	).Scan(
		&cmd.ID, &cmd.Raw, &cmd.Binary, &cmd.Subcommand, &cmd.Flags,
		&cmd.Category, &cmd.Frequency, &firstSeen, &lastSeen,
		&cmd.LastExit, &cmd.AvgDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("getting command %d: %w", id, err)
	}
	cmd.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
	cmd.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	return &cmd, nil
}

func (s *Store) GetRecentCommands(limit int) ([]Command, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration
		 FROM commands ORDER BY last_seen DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("getting recent commands: %w", err)
	}
	defer rows.Close()

	var cmds []Command
	for rows.Next() {
		var cmd Command
		var firstSeen, lastSeen string
		err := rows.Scan(
			&cmd.ID, &cmd.Raw, &cmd.Binary, &cmd.Subcommand, &cmd.Flags,
			&cmd.Category, &cmd.Frequency, &firstSeen, &lastSeen,
			&cmd.LastExit, &cmd.AvgDuration,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning command: %w", err)
		}
		cmd.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
		cmd.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
		cmds = append(cmds, cmd)
	}
	return cmds, rows.Err()
}

func (s *Store) GetStats() (*VaultStats, error) {
	stats := &VaultStats{}
	err := s.db.QueryRow("SELECT COALESCE(SUM(frequency), 0), COUNT(*) FROM commands").Scan(
		&stats.TotalCommands, &stats.UniqueCommands,
	)
	if err != nil {
		return nil, fmt.Errorf("getting command stats: %w", err)
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM contexts").Scan(&stats.TotalContexts)
	if err != nil {
		return nil, fmt.Errorf("getting context stats: %w", err)
	}
	return stats, nil
}

func (s *Store) BatchInsertCommands(cmds []Command) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO commands (raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(raw) DO UPDATE SET frequency = frequency + 1, last_seen = excluded.last_seen`,
	)
	if err != nil {
		return 0, fmt.Errorf("preparing batch insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, cmd := range cmds {
		if cmd.FirstSeen.IsZero() {
			cmd.FirstSeen = time.Now().UTC()
		}
		if cmd.LastSeen.IsZero() {
			cmd.LastSeen = time.Now().UTC()
		}
		_, err := stmt.Exec(
			cmd.Raw, cmd.Binary, cmd.Subcommand, cmd.Flags, cmd.Category,
			max(cmd.Frequency, 1),
			cmd.FirstSeen.Format(time.RFC3339),
			cmd.LastSeen.Format(time.RFC3339),
			cmd.LastExit, cmd.AvgDuration,
		)
		if err != nil {
			continue
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing batch insert: %w", err)
	}
	return inserted, nil
}

// buildFTSQuery converts a natural language query into an FTS5 query.
// For Phase 1, we split on spaces and OR the terms together.
func buildFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	words := strings.Fields(query)
	var terms []string
	for _, w := range words {
		cleaned := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == '/' {
				return r
			}
			return -1
		}, w)
		if cleaned != "" {
			terms = append(terms, cleaned)
		}
	}

	if len(terms) == 0 {
		return ""
	}

	// Use OR so any matching term surfaces results
	return strings.Join(terms, " OR ")
}

// scoreToConfidence maps FTS5 rank + frequency to a 0-100 confidence percentage.
// Phase 1 uses a simple heuristic; Phase 2 replaces this with multi-signal scoring.
func scoreToConfidence(ftsScore float64, frequency int) float64 {
	textScore := ftsScore / (ftsScore + 1.0) // normalize to 0-1
	freqBoost := float64(frequency) / (float64(frequency) + 10.0)
	confidence := (textScore*0.7 + freqBoost*0.3) * 100
	if confidence > 99 {
		confidence = 99
	}
	return confidence
}
