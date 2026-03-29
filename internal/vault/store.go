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

	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

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

func (s *Store) SearchKnowledgeFTS5(query string, limit int) ([]Knowledge, error) {
	if limit <= 0 {
		limit = 20
	}

	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := s.db.Query(
		`SELECT k.id, k.command, k.description, k.intents, k.category,
		        k.flags_doc, k.examples, k.danger_level, rank
		 FROM knowledge_fts kf
		 JOIN knowledge k ON k.id = kf.rowid
		 WHERE knowledge_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`,
		ftsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("knowledge FTS5 search: %w", err)
	}
	defer rows.Close()

	var results []Knowledge
	for rows.Next() {
		var k Knowledge
		var rank float64
		err := rows.Scan(
			&k.ID, &k.Command, &k.Description, &k.Intents, &k.Category,
			&k.FlagsDoc, &k.Examples, &k.DangerLevel, &rank,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning knowledge result: %w", err)
		}
		results = append(results, k)
	}
	return results, rows.Err()
}

func (s *Store) InsertKnowledge(k *Knowledge) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO knowledge (command, description, intents, category, flags_doc, examples, danger_level)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		k.Command, k.Description, k.Intents, k.Category,
		k.FlagsDoc, k.Examples, k.DangerLevel,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting knowledge: %w", err)
	}
	return result.LastInsertId()
}

func (s *Store) BatchInsertKnowledge(entries []Knowledge) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT OR IGNORE INTO knowledge (command, description, intents, category, flags_doc, examples, danger_level)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return 0, fmt.Errorf("preparing knowledge insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, k := range entries {
		_, err := stmt.Exec(k.Command, k.Description, k.Intents, k.Category,
			k.FlagsDoc, k.Examples, k.DangerLevel)
		if err != nil {
			continue
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing knowledge insert: %w", err)
	}
	return inserted, nil
}

func (s *Store) KnowledgeCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM knowledge").Scan(&count)
	return count, err
}

func (s *Store) GetContextsForCommand(commandID int64) ([]Context, error) {
	rows, err := s.db.Query(
		`SELECT id, command_id, cwd, git_repo, git_branch, project_type, timestamp, exit_code, duration_ms, session_id
		 FROM contexts WHERE command_id = ? ORDER BY timestamp DESC LIMIT 10`,
		commandID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting contexts for command %d: %w", commandID, err)
	}
	defer rows.Close()

	var ctxs []Context
	for rows.Next() {
		var ctx Context
		var ts string
		err := rows.Scan(&ctx.ID, &ctx.CommandID, &ctx.Cwd, &ctx.GitRepo,
			&ctx.GitBranch, &ctx.ProjectType, &ts,
			&ctx.ExitCode, &ctx.DurationMs, &ctx.SessionID)
		if err != nil {
			return nil, fmt.Errorf("scanning context: %w", err)
		}
		ctx.Timestamp, _ = time.Parse(time.RFC3339, ts)
		ctxs = append(ctxs, ctx)
	}
	return ctxs, rows.Err()
}

func (s *Store) GetMaxFrequency() (int, error) {
	var maxFreq int
	err := s.db.QueryRow("SELECT COALESCE(MAX(frequency), 1) FROM commands").Scan(&maxFreq)
	if err != nil {
		return 1, fmt.Errorf("getting max frequency: %w", err)
	}
	if maxFreq == 0 {
		maxFreq = 1
	}
	return maxFreq, nil
}

// sanitizeFTSQuery ensures a query string is safe for FTS5.
// Accepts pre-built FTS5 queries (with OR operators) or simple terms.
func sanitizeFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	words := strings.Fields(query)
	var cleaned []string
	for _, w := range words {
		if strings.EqualFold(w, "OR") || strings.EqualFold(w, "AND") || strings.EqualFold(w, "NOT") {
			cleaned = append(cleaned, strings.ToUpper(w))
			continue
		}
		term := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == '/' || r == '*' {
				return r
			}
			return -1
		}, w)
		if term != "" {
			cleaned = append(cleaned, term)
		}
	}

	if len(cleaned) == 0 {
		return ""
	}

	// Remove leading/trailing operators and consecutive operators
	var result []string
	prevOp := true
	for _, c := range cleaned {
		isOp := c == "OR" || c == "AND" || c == "NOT"
		if isOp && prevOp {
			continue
		}
		result = append(result, c)
		prevOp = isOp
	}

	// Trim trailing operator
	for len(result) > 0 {
		last := result[len(result)-1]
		if last == "OR" || last == "AND" || last == "NOT" {
			result = result[:len(result)-1]
		} else {
			break
		}
	}

	if len(result) == 0 {
		return ""
	}

	return strings.Join(result, " ")
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
