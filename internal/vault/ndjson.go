package vault

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// ExportNDJSON streams all commands and contexts to the provided writer in NDJSON format.
// Each line is a JSON object. Commands are written first, then Contexts.
func (s *Store) ExportNDJSON(w io.Writer, commandsOnly bool) error {
	encoder := json.NewEncoder(w)

	cmdErr := s.StreamCommands(func(cmd Command) error {
		type ndjsonCmd struct {
			Type string `json:"_type"`
			Command
		}
		return encoder.Encode(ndjsonCmd{Type: "command", Command: cmd})
	})
	if cmdErr != nil {
		return fmt.Errorf("streaming commands: %w", cmdErr)
	}

	if !commandsOnly {
		ctxErr := s.StreamContexts(func(ctx Context) error {
			type ndjsonCtx struct {
				Type string `json:"_type"`
				Context
			}
			return encoder.Encode(ndjsonCtx{Type: "context", Context: ctx})
		})
		if ctxErr != nil {
			return fmt.Errorf("streaming contexts: %w", ctxErr)
		}
	}

	return nil
}

// ImportNDJSON reads an NDJSON stream and imports commands and contexts into the store.
// Returns the number of commands and contexts imported.
func (s *Store) ImportNDJSON(r io.Reader) (int, int, error) {
	scanner := bufio.NewScanner(r)
	// Buffer size for large lines (e.g., commands with huge flags)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var cmds []Command
	var ctxs []Context

	cmdsImported := 0
	ctxsImported := 0

	batchSize := 500

	flushCmds := func() error {
		if len(cmds) > 0 {
			inserted, err := s.BatchInsertCommands(cmds)
			if err != nil {
				return err
			}
			cmdsImported += inserted
			cmds = cmds[:0]
		}
		return nil
	}

	flushCtxs := func() error {
		if len(ctxs) > 0 {
			// Actually we don't have BatchInsertContexts yet.
			// Let's just insert them in a transaction line by line if Batch doesn't exist,
			// or we can add BatchInsertContexts later. For now, loop InsertContext.
			tx, err := s.db.Begin()
			if err != nil {
				return err
			}
			
			stmt, err := tx.Prepare(
				`INSERT INTO contexts (command_id, cwd, git_repo, git_branch, project_type, timestamp, exit_code, duration_ms, session_id)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			)
			if err != nil {
				tx.Rollback()
				return err
			}
			defer stmt.Close()

			count := 0
			for _, ctx := range ctxs {
				tsStr := ctx.Timestamp.UTC().Format("2006-01-02T15:04:05Z")
				_, err := stmt.Exec(
					ctx.CommandID, ctx.Cwd, ctx.GitRepo, ctx.GitBranch,
					ctx.ProjectType, tsStr, ctx.ExitCode, ctx.DurationMs, ctx.SessionID,
				)
				if err == nil {
					count++
				}
			}
			if err := tx.Commit(); err != nil {
				return err
			}
			ctxsImported += count
			ctxs = ctxs[:0]
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Fast check for type using simple unmarshal
		var t struct {
			Type string `json:"_type"`
		}
		if err := json.Unmarshal(line, &t); err != nil {
			continue
		}

		if t.Type == "command" {
			var cmdWrapper struct {
				Command
			}
			if err := json.Unmarshal(line, &cmdWrapper); err == nil {
				cmds = append(cmds, cmdWrapper.Command)
				if len(cmds) >= batchSize {
					if err := flushCmds(); err != nil {
						return cmdsImported, ctxsImported, fmt.Errorf("flushing commands: %w", err)
					}
				}
			}
		} else if t.Type == "context" {
			var ctxWrapper struct {
				Context
			}
			if err := json.Unmarshal(line, &ctxWrapper); err == nil {
				ctxs = append(ctxs, ctxWrapper.Context)
				if len(ctxs) >= batchSize {
					if err := flushCtxs(); err != nil {
						return cmdsImported, ctxsImported, fmt.Errorf("flushing contexts: %w", err)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return cmdsImported, ctxsImported, fmt.Errorf("reading ndjson: %w", err)
	}

	if err := flushCmds(); err != nil {
		return cmdsImported, ctxsImported, fmt.Errorf("finalizing commands: %w", err)
	}
	if err := flushCtxs(); err != nil {
		return cmdsImported, ctxsImported, fmt.Errorf("finalizing contexts: %w", err)
	}

	return cmdsImported, ctxsImported, nil
}
