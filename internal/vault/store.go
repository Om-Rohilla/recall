package vault

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Om-Rohilla/recall/pkg/logging"
	_ "modernc.org/sqlite"
)

type Store struct {
	db       *sql.DB
	path     string
	encKey   []byte
	tempPath string
	sigChan  chan os.Signal
}

func NewStore(dbPath string) (*Store, error) {
	log := logging.Get()
	log.Debug("opening vault", "path", dbPath)

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating vault directory: %w", err)
	}

	encPath := dbPath + ".enc"
	var encKey []byte
	var workingPath string

	if keyHex := os.Getenv("RECALL_VAULT_KEY"); keyHex != "" {
		var err error
		encKey, err = hex.DecodeString(keyHex)
		if err != nil || len(encKey) != KeySize {
			return nil, fmt.Errorf("RECALL_VAULT_KEY must be a %d-byte hex string (%d hex chars)", KeySize, KeySize*2)
		}

		if _, err := os.Stat(encPath); err == nil {
			encData, err := os.ReadFile(encPath)
			if err != nil {
				return nil, fmt.Errorf("reading encrypted vault: %w", err)
			}
			plainData, err := Decrypt(encData, encKey)
			if err != nil {
				return nil, fmt.Errorf("decrypting vault: %w", err)
			}
			workingPath = dbPath + ".tmp." + randomHex(8)
			if err := os.WriteFile(workingPath, plainData, 0o600); err != nil {
				return nil, fmt.Errorf("writing decrypted vault: %w", err)
			}
		} else if _, err := os.Stat(dbPath); err == nil {
			workingPath = dbPath
		} else {
			workingPath = dbPath + ".tmp." + randomHex(8)
		}
	} else {
		workingPath = dbPath
	}

	db, err := sql.Open("sqlite", workingPath)
	if err != nil {
		return nil, fmt.Errorf("opening vault database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	if workingPath == dbPath {
		if err := os.Chmod(dbPath, 0o600); err != nil && !os.IsNotExist(err) {
			db.Close()
			return nil, fmt.Errorf("setting vault file permissions: %w", err)
		}
	}

	s := &Store{db: db, path: dbPath}
	if encKey != nil {
		s.encKey = encKey
		s.tempPath = workingPath
		// Register signal handler to securely clean up temp files on unexpected exit
		s.sigChan = make(chan os.Signal, 1)
		signal.Notify(s.sigChan, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-s.sigChan
			log.Warn("signal received, cleaning up temporary vault files")
			s.db.Close()
			secureDelete(s.tempPath)
			for _, suffix := range []string{"-wal", "-shm"} {
				secureDelete(s.tempPath + suffix)
			}
			os.Exit(1)
		}()
	}
	return s, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func (s *Store) Close() error {
	log := logging.Get()
	log.Debug("closing vault", "path", s.path, "encrypted", s.encKey != nil)

	// Stop signal handler before close
	if s.sigChan != nil {
		signal.Stop(s.sigChan)
	}

	dbCloseErr := s.db.Close()

	if s.encKey != nil && s.tempPath != "" {
		cleanupTemp := func() {
			secureDelete(s.tempPath)
			for _, suffix := range []string{"-wal", "-shm"} {
				secureDelete(s.tempPath + suffix)
			}
		}

		plainData, err := os.ReadFile(s.tempPath)
		if err != nil {
			cleanupTemp()
			return fmt.Errorf("reading vault for encryption: %w", err)
		}
		encData, err := Encrypt(plainData, s.encKey)
		if err != nil {
			cleanupTemp()
			return fmt.Errorf("encrypting vault: %w", err)
		}
		encPath := s.path + ".enc"
		if err := os.WriteFile(encPath, encData, 0o600); err != nil {
			cleanupTemp()
			return fmt.Errorf("writing encrypted vault: %w", err)
		}
		cleanupTemp()
	}

	return dbCloseErr
}

func secureDelete(path string) {
	log := logging.Get()
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	size := info.Size()
	if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
		zeros := make([]byte, 4096)
		for written := int64(0); written < size; {
			n := size - written
			if n > int64(len(zeros)) {
				n = int64(len(zeros))
			}
			w, err := f.Write(zeros[:n])
			if err != nil {
				log.Warn("secure delete write failed", "path", path, "error", err)
				break
			}
			written += int64(w)
		}
		if err := f.Sync(); err != nil {
			log.Warn("secure delete sync failed", "path", path, "error", err)
		}
		f.Close()
	} else {
		log.Warn("secure delete open failed", "path", path, "error", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Warn("secure delete remove failed", "path", path, "error", err)
	}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) InsertCommand(cmd *Command) (int64, error) {
	now := time.Now().UTC()
	if cmd.FirstSeen.IsZero() {
		cmd.FirstSeen = now
	}
	if cmd.LastSeen.IsZero() {
		cmd.LastSeen = now
	}

	result, err := s.db.Exec(
		`INSERT INTO commands (raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(raw) DO UPDATE SET
		   frequency = frequency + 1,
		   last_seen = excluded.last_seen,
		   last_exit = excluded.last_exit,
		   avg_duration = excluded.avg_duration`,
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
		r.Command.FirstSeen = safeParseTime(firstSeen)
		r.Command.LastSeen = safeParseTime(lastSeen)
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
	cmd.FirstSeen = safeParseTime(firstSeen)
	cmd.LastSeen = safeParseTime(lastSeen)
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
	var lastErr error
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
			lastErr = err
			continue
		}
		inserted++
	}
	if inserted == 0 && lastErr != nil {
		return 0, fmt.Errorf("batch insert failed on all rows, last error: %w", lastErr)
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
		ctx.Timestamp = safeParseTime(ts)
		ctxs = append(ctxs, ctx)
	}
	return ctxs, rows.Err()
}

// GetContextsForCommandBatch fetches contexts for multiple command IDs in a single query.
func (s *Store) GetContextsForCommandBatch(commandIDs []int64) (map[int64][]Context, error) {
	result := make(map[int64][]Context, len(commandIDs))
	if len(commandIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(commandIDs))
	args := make([]interface{}, len(commandIDs))
	for i, id := range commandIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT id, command_id, cwd, git_repo, git_branch, project_type, timestamp, exit_code, duration_ms, session_id
		 FROM contexts WHERE command_id IN (%s) ORDER BY timestamp DESC`,
		strings.Join(placeholders, ","),
	)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch getting contexts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ctx Context
		var ts string
		err := rows.Scan(&ctx.ID, &ctx.CommandID, &ctx.Cwd, &ctx.GitRepo,
			&ctx.GitBranch, &ctx.ProjectType, &ts,
			&ctx.ExitCode, &ctx.DurationMs, &ctx.SessionID)
		if err != nil {
			return nil, fmt.Errorf("scanning batch context: %w", err)
		}
		ctx.Timestamp = safeParseTime(ts)
		result[ctx.CommandID] = append(result[ctx.CommandID], ctx)
	}

	return result, rows.Err()
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
// All user terms are quoted with double quotes to prevent FTS5 operator
// injection (OR, AND, NOT in user input are treated as literals).
// Pre-built queries from BuildFTSQuery use OR operators between quoted terms.
func sanitizeFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	words := strings.Fields(query)
	var cleaned []string
	for _, w := range words {
		// Preserve OR operators from internally-built FTS queries (BuildFTSQuery)
		if strings.EqualFold(w, "OR") {
			if len(cleaned) > 0 { // only add OR between terms, never leading
				cleaned = append(cleaned, "OR")
			}
			continue
		}
		// Strip special characters that could be FTS5 syntax
		term := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == '/' {
				return r
			}
			return -1
		}, w)
		if term != "" {
			// Quote every term to prevent injection of AND/NOT/NEAR operators
			cleaned = append(cleaned, `"`+term+`"`)
		}
	}

	if len(cleaned) == 0 {
		return ""
	}

	// Remove trailing OR operator
	for len(cleaned) > 0 && cleaned[len(cleaned)-1] == "OR" {
		cleaned = cleaned[:len(cleaned)-1]
	}
	// Remove consecutive OR operators
	var result []string
	prevOR := false
	for _, c := range cleaned {
		if c == "OR" {
			if prevOR || len(result) == 0 {
				continue
			}
			prevOR = true
		} else {
			prevOR = false
		}
		result = append(result, c)
	}

	if len(result) == 0 {
		return ""
	}

	return strings.Join(result, " ")
}

func (s *Store) GetAllCommands(sortBy string, limit int) ([]Command, error) {
	const baseQuery = `SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration FROM commands`
	const safeMax = 100000

	if limit <= 0 {
		limit = safeMax
	}

	var rows *sql.Rows
	var err error

	switch sortBy {
	case "frequency":
		rows, err = s.db.Query(baseQuery+` ORDER BY frequency DESC LIMIT ?`, limit)
	case "alpha":
		rows, err = s.db.Query(baseQuery+` ORDER BY raw ASC LIMIT ?`, limit)
	default:
		rows, err = s.db.Query(baseQuery+` ORDER BY last_seen DESC LIMIT ?`, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("getting all commands: %w", err)
	}
	defer rows.Close()
	return scanCommands(rows)
}

// GetCommandsPaginated returns a page of commands with the given sort order.
func (s *Store) GetCommandsPaginated(sortBy string, limit, offset int) ([]Command, error) {
	const baseQuery = `SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration FROM commands`

	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	switch sortBy {
	case "frequency":
		rows, err = s.db.Query(baseQuery+` ORDER BY frequency DESC LIMIT ? OFFSET ?`, limit, offset)
	case "alpha":
		rows, err = s.db.Query(baseQuery+` ORDER BY raw ASC LIMIT ? OFFSET ?`, limit, offset)
	default:
		rows, err = s.db.Query(baseQuery+` ORDER BY last_seen DESC LIMIT ? OFFSET ?`, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("getting paginated commands: %w", err)
	}
	defer rows.Close()
	return scanCommands(rows)
}

// GetCommandsByCategoryPaginated returns a page of commands filtered by category.
func (s *Store) GetCommandsByCategoryPaginated(category string, limit, offset int) ([]Command, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration
		 FROM commands WHERE category = ? ORDER BY frequency DESC LIMIT ? OFFSET ?`, category, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("getting paginated commands by category: %w", err)
	}
	defer rows.Close()
	return scanCommands(rows)
}

// GetCommandCount returns the total number of commands in the vault.
func (s *Store) GetCommandCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM commands").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("getting command count: %w", err)
	}
	return count, nil
}

// GetCommandCountByCategory returns the total number of commands in a category.
func (s *Store) GetCommandCountByCategory(category string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM commands WHERE category = ?", category).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("getting command count by category: %w", err)
	}
	return count, nil
}

func (s *Store) GetCommandsByCategory(category string, limit int) ([]Command, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(
		`SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration
		 FROM commands WHERE category = ? ORDER BY frequency DESC LIMIT ?`, category, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("getting commands by category: %w", err)
	}
	defer rows.Close()
	return scanCommands(rows)
}

func (s *Store) DeleteCommand(id int64) error {
	if _, err := s.db.Exec("DELETE FROM commands WHERE id = ?", id); err != nil {
		return fmt.Errorf("deleting command %d: %w", id, err)
	}
	return nil
}

func (s *Store) GetCategories() ([]CategoryCount, error) {
	rows, err := s.db.Query(
		`SELECT category, COUNT(*) as cnt, SUM(frequency) as total_freq
		 FROM commands WHERE category != '' GROUP BY category ORDER BY total_freq DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("getting categories: %w", err)
	}
	defer rows.Close()
	var cats []CategoryCount
	for rows.Next() {
		var c CategoryCount
		if err := rows.Scan(&c.Category, &c.Count, &c.TotalFrequency); err != nil {
			return nil, fmt.Errorf("scanning category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *Store) GetTopCommands(period int, limit int) ([]Command, error) {
	if limit <= 0 {
		limit = 10
	}
	since := time.Now().UTC().AddDate(0, 0, -period).Format(time.RFC3339)
	rows, err := s.db.Query(
		`SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration
		 FROM commands WHERE last_seen >= ? ORDER BY frequency DESC LIMIT ?`, since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("getting top commands: %w", err)
	}
	defer rows.Close()
	return scanCommands(rows)
}

func (s *Store) GetRareCommands(maxFreq int, limit int) ([]Command, error) {
	if limit <= 0 {
		limit = 10
	}
	if maxFreq <= 0 {
		maxFreq = 3
	}
	rows, err := s.db.Query(
		`SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration
		 FROM commands WHERE frequency <= ? AND length(raw) > 10 ORDER BY length(raw) DESC LIMIT ?`, maxFreq, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("getting rare commands: %w", err)
	}
	defer rows.Close()
	return scanCommands(rows)
}

func (s *Store) GetVaultPeriod() (time.Time, time.Time, error) {
	var first, last string
	err := s.db.QueryRow("SELECT COALESCE(MIN(first_seen), ''), COALESCE(MAX(last_seen), '') FROM commands").Scan(&first, &last)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("getting vault period: %w", err)
	}
	firstTime := safeParseTime(first)
	lastTime := safeParseTime(last)
	return firstTime, lastTime, nil
}

func (s *Store) GetHighFrequencyCommands(minFreq int) ([]Command, error) {
	rows, err := s.db.Query(
		`SELECT id, raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration
		 FROM commands WHERE frequency >= ? ORDER BY frequency DESC LIMIT 10000`, minFreq,
	)
	if err != nil {
		return nil, fmt.Errorf("getting high frequency commands: %w", err)
	}
	defer rows.Close()
	return scanCommands(rows)
}

func scanCommands(rows *sql.Rows) ([]Command, error) {
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
		cmd.FirstSeen = safeParseTime(firstSeen)
		cmd.LastSeen = safeParseTime(lastSeen)
		cmds = append(cmds, cmd)
	}
	return cmds, rows.Err()
}

func (s *Store) GetAllContexts() ([]Context, error) {
	rows, err := s.db.Query(
		`SELECT id, command_id, cwd, git_repo, git_branch, project_type, timestamp, exit_code, duration_ms, session_id
		 FROM contexts ORDER BY timestamp DESC LIMIT 100000`,
	)
	if err != nil {
		return nil, fmt.Errorf("getting all contexts: %w", err)
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
		ctx.Timestamp = safeParseTime(ts)
		ctxs = append(ctxs, ctx)
	}
	return ctxs, rows.Err()
}

func (s *Store) GetAllPatterns() ([]Pattern, error) {
	rows, err := s.db.Query(
		`SELECT id, template, frequency, suggested_alias FROM patterns ORDER BY frequency DESC LIMIT 50000`,
	)
	if err != nil {
		return nil, fmt.Errorf("getting all patterns: %w", err)
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var p Pattern
		if err := rows.Scan(&p.ID, &p.Template, &p.Frequency, &p.SuggestedAlias); err != nil {
			return nil, fmt.Errorf("scanning pattern: %w", err)
		}
		patterns = append(patterns, p)
	}
	return patterns, rows.Err()
}

// ExportVaultData exports all vault data for serialization.
func (s *Store) ExportVaultData(commandsOnly bool) (*ExportData, error) {
	cmds, err := s.GetAllCommands("recency", 100000)
	if err != nil {
		return nil, fmt.Errorf("exporting commands: %w", err)
	}

	data := &ExportData{
		Version:      1,
		ExportedAt:   time.Now().UTC(),
		Commands:     cmds,
		CommandCount: len(cmds),
	}

	if !commandsOnly {
		ctxs, err := s.GetAllContexts()
		if err != nil {
			return nil, fmt.Errorf("exporting contexts: %w", err)
		}
		data.Contexts = ctxs
		data.ContextCount = len(ctxs)

		patterns, err := s.GetAllPatterns()
		if err != nil {
			return nil, fmt.Errorf("exporting patterns: %w", err)
		}
		data.Patterns = patterns
	}

	return data, nil
}

// ImportVaultData imports vault data, replacing or merging based on the merge flag.
// The entire operation is wrapped in a transaction to prevent data loss on crash.
func (s *Store) ImportVaultData(data *ExportData, merge bool) (importedCmds, importedCtxs int, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("beginning import transaction: %w", err)
	}
	defer tx.Rollback()

	if !merge {
		if _, err := tx.Exec("DELETE FROM contexts"); err != nil {
			return 0, 0, fmt.Errorf("clearing contexts: %w", err)
		}
		if _, err := tx.Exec("DELETE FROM commands"); err != nil {
			return 0, 0, fmt.Errorf("clearing commands: %w", err)
		}
		if _, err := tx.Exec("DELETE FROM patterns"); err != nil {
			return 0, 0, fmt.Errorf("clearing patterns: %w", err)
		}
	}

	idMap := make(map[int64]int64)

	now := time.Now().UTC().Format(time.RFC3339)
	for _, cmd := range data.Commands {
		oldID := cmd.ID
		cmd.ID = 0

		if merge {
			var existingID int64
			qErr := tx.QueryRow("SELECT id FROM commands WHERE raw = ?", cmd.Raw).Scan(&existingID)
			if qErr == nil {
				if _, uErr := tx.Exec(
					"UPDATE commands SET frequency = frequency + ? WHERE id = ?",
					cmd.Frequency, existingID,
				); uErr != nil {
					continue
				}
				idMap[oldID] = existingID
				importedCmds++
				continue
			}
		}

		if cmd.FirstSeen.IsZero() {
			cmd.FirstSeen = time.Now().UTC()
		}
		if cmd.LastSeen.IsZero() {
			cmd.LastSeen = time.Now().UTC()
		}
		result, iErr := tx.Exec(
			`INSERT INTO commands (raw, binary_name, subcommand, flags, category, frequency, first_seen, last_seen, last_exit, avg_duration)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(raw) DO UPDATE SET frequency = frequency + excluded.frequency, last_seen = ?`,
			cmd.Raw, cmd.Binary, cmd.Subcommand, cmd.Flags, cmd.Category,
			max(cmd.Frequency, 1),
			cmd.FirstSeen.Format(time.RFC3339),
			cmd.LastSeen.Format(time.RFC3339),
			cmd.LastExit, cmd.AvgDuration, now,
		)
		if iErr != nil {
			continue
		}
		newID, _ := result.LastInsertId()
		idMap[oldID] = newID
		importedCmds++
	}

	for _, ctx := range data.Contexts {
		newCmdID, ok := idMap[ctx.CommandID]
		if !ok {
			continue
		}
		ctx.CommandID = newCmdID
		ctx.ID = 0
		if ctx.Timestamp.IsZero() {
			ctx.Timestamp = time.Now().UTC()
		}
		if _, cErr := tx.Exec(
			`INSERT INTO contexts (command_id, cwd, git_repo, git_branch, project_type, timestamp, exit_code, duration_ms, session_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ctx.CommandID, ctx.Cwd, ctx.GitRepo, ctx.GitBranch, ctx.ProjectType,
			ctx.Timestamp.Format(time.RFC3339),
			ctx.ExitCode, ctx.DurationMs, ctx.SessionID,
		); cErr != nil {
			continue
		}
		importedCtxs++
	}

	for _, p := range data.Patterns {
		_, _ = tx.Exec(
			`INSERT INTO patterns (template, frequency, suggested_alias) VALUES (?, ?, ?)
			 ON CONFLICT(template) DO UPDATE SET frequency = frequency + excluded.frequency`,
			p.Template, p.Frequency, p.SuggestedAlias,
		)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("committing import transaction: %w", err)
	}

	if err := s.RebuildFTSIndex(); err != nil {
		return importedCmds, importedCtxs, fmt.Errorf("rebuilding FTS index after import: %w", err)
	}

	return importedCmds, importedCtxs, nil
}

// scoreToConfidence maps FTS5 rank + frequency to a 0-100 confidence percentage.
// Used by SearchFTS5 for direct FTS queries that bypass the intelligence scoring pipeline.
func scoreToConfidence(ftsScore float64, frequency int) float64 {
	textScore := ftsScore / (ftsScore + 1.0)
	freqBoost := float64(frequency) / (float64(frequency) + 10.0)
	confidence := (textScore*0.7 + freqBoost*0.3) * 100
	if confidence > 99 {
		confidence = 99
	}
	return confidence
}

// safeParseTime parses an RFC3339 timestamp, logging a warning on failure
// instead of silently returning zero time.
func safeParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		logging.Get().Warn("failed to parse timestamp", "value", s, "error", err)
		return time.Time{}
	}
	return t
}

// Vacuum runs SQLite VACUUM to reclaim unused space and defragment the database.
func (s *Store) Vacuum() error {
	if _, err := s.db.Exec("PRAGMA incremental_vacuum"); err != nil {
		return fmt.Errorf("incremental vacuum: %w", err)
	}
	if _, err := s.db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}
	return nil
}

// PruneOldCommands deletes commands whose last_seen is older than daysOld days.
// Returns the number of deleted commands.
func (s *Store) PruneOldCommands(daysOld int) (int, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -daysOld).Format(time.RFC3339)
	result, err := s.db.Exec("DELETE FROM commands WHERE last_seen < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("pruning old commands: %w", err)
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// VaultFileSize returns the size of the vault database file in bytes.
func (s *Store) VaultFileSize() (int64, error) {
	info, err := os.Stat(s.path)
	if err != nil {
		return 0, fmt.Errorf("stat vault file: %w", err)
	}
	return info.Size(), nil
}

