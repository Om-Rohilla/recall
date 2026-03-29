# RECALL — PROJECT MEMORY

> **IMPORTANT: Read this file FIRST before doing ANYTHING.**
> This is the single source of truth for project state. Update it after every session.

---

## IDENTITY

**Project:** Recall — Command Intelligence Engine
**Language:** Go 1.22+
**What it does:** 100% offline CLI tool that eliminates Command Amnesia. Developers search commands by intent, not exact syntax. Learns from personal history. Sub-10ms response.

---

## CURRENT STATUS

| Item | Status |
|------|--------|
| **Current Phase** | Phase 2 — Intelligence (COMPLETE) |
| **Last thing built** | Full Phase 2: intelligence layer, context detection, knowledge base, scoring pipeline, tests |
| **Next thing to build** | Phase 3 — Explain + Compose: command decomposition, flag database, explain/compose commands |
| **Blockers** | None |
| **Known bugs** | None |

---

## WHAT EXISTS (Already Built)

### Documentation (COMPLETE)
- [x] `README.md` — full GitHub-ready README
- [x] `docs/PROBLEM-AND-SOLUTION.md` — problem analysis + solution
- [x] `docs/ARCHITECTURE.md` — full system architecture
- [x] `docs/COMMANDS.md` — complete CLI reference (15 commands)
- [x] `docs/TECH-STACK.md` — stack + 6 build phases
- [x] `docs/MARKETING.md` — launch strategy
- [x] `PROMPT.md` — golden prompt for Claude
- [x] `MEMORY.md` — this file
- [x] `.gitignore` — configured
- [x] `LICENSE` — MIT
- [x] `.cursor/rules/recall-project.mdc` — auto-loaded project rules

### Code — Phase 1 (COMPLETE)
- [x] `go.mod` / `go.sum` — Go module with Cobra, Lipgloss, SQLite, TOML
- [x] `main.go` — entry point
- [x] `Makefile` — build, test, clean, install targets
- [x] `cmd/root.go` — Cobra root command + version
- [x] `cmd/init_cmd.go` — `recall init` command
- [x] `cmd/capture.go` — `recall capture` (hidden, called by hook)
- [x] `cmd/hook.go` — `recall hook zsh|bash`
- [x] `cmd/import_history.go` — `recall import-history`
- [x] `cmd/search.go` — `recall search` + default arg search
- [x] `internal/vault/` — models, schema, store, index (SQLite + FTS5)
- [x] `internal/capture/` — parser, filter, enricher, receiver
- [x] `internal/ui/` — theme (Lipgloss), result card rendering
- [x] `pkg/config/` — TOML config with defaults + env overrides
- [x] `pkg/shell/` — detect, hooks, install
- [x] `scripts/hooks/` — recall.zsh, recall.bash
- [x] `tests/` — 30 table-driven tests (vault + capture)

### Code — Phase 2 (COMPLETE)
- [x] `internal/context/git.go` — Git state detection (repo, branch, dirty)
- [x] `internal/context/project.go` — Project type detection (25+ markers, framework detection)
- [x] `internal/context/session.go` — Session tracking (recent commands, categories)
- [x] `internal/context/detector.go` — Context aggregator (cwd, git, project, environment)
- [x] `internal/intelligence/intent.go` — Intent extraction + synonym dictionary (80+ synonym entries)
- [x] `internal/intelligence/scorer.go` — Multi-signal scoring (text 30%, intent 25%, freq 20%, context 15%, recency 10%)
- [x] `internal/intelligence/engine.go` — 4-stage search orchestrator (intent → candidates → score → rank)
- [x] `internal/intelligence/patterns.go` — Pattern/template extraction (URL, path, hash, IP placeholders)
- [x] `internal/intelligence/knowledge.go` — Knowledge base loader (JSON → SQLite)
- [x] `data/knowledge.json` — 200 curated commands across 12 categories
- [x] `internal/vault/store.go` — Added knowledge CRUD, context queries, max frequency, FTS5 query sanitization
- [x] `cmd/search.go` — Wired intelligence pipeline (replaces raw FTS5 with full scoring)
- [x] `tests/intelligence_test.go` — 20 new tests (49 total)

---

## BUILD PHASES — PROGRESS TRACKER

### Phase 1 — Foundation
| Task | Status | File(s) |
|------|--------|---------|
| Go module init (`go.mod`) | DONE | `go.mod` |
| Entry point | DONE | `main.go` |
| Makefile (build, test, clean) | DONE | `Makefile` |
| Cobra root command | DONE | `cmd/root.go` |
| SQLite vault schema | DONE | `internal/vault/schema.go` |
| Vault store (CRUD + FTS5) | DONE | `internal/vault/store.go` |
| Vault models | DONE | `internal/vault/models.go` |
| FTS5 index management | DONE | `internal/vault/index.go` |
| Command parser | DONE | `internal/capture/parser.go` |
| Capture receiver | DONE | `internal/capture/receiver.go` |
| Context enricher | DONE | `internal/capture/enricher.go` |
| Secret/noise filter | DONE | `internal/capture/filter.go` |
| Shell detection | DONE | `pkg/shell/detect.go` |
| Hook script generation | DONE | `pkg/shell/hooks.go` |
| Hook installer | DONE | `pkg/shell/install.go` |
| Zsh hook script | DONE | `scripts/hooks/recall.zsh` |
| Bash hook script | DONE | `scripts/hooks/recall.bash` |
| Config defaults | DONE | `pkg/config/defaults.go` |
| Config manager | DONE | `pkg/config/config.go` |
| `recall init` command | DONE | `cmd/init_cmd.go` |
| `recall import-history` command | DONE | `cmd/import_history.go` |
| `recall search` command | DONE | `cmd/search.go` |
| Basic result card display | DONE | `internal/ui/result.go` |
| Theme/colors | DONE | `internal/ui/theme.go` |
| Unit tests (vault) | DONE | `tests/vault_test.go` |
| Unit tests (capture) | DONE | `tests/capture_test.go` |
| **PHASE 1 COMPLETE** | **YES** | |

### Phase 2 — Intelligence
| Task | Status | File(s) |
|------|--------|---------|
| Intent extraction engine | DONE | `internal/intelligence/intent.go` |
| Synonym dictionary | DONE | `internal/intelligence/intent.go` |
| Multi-signal scoring algorithm | DONE | `internal/intelligence/scorer.go` |
| Search orchestrator | DONE | `internal/intelligence/engine.go` |
| Pattern extraction | DONE | `internal/intelligence/patterns.go` |
| Knowledge base loader | DONE | `internal/intelligence/knowledge.go` |
| Knowledge base data (200 cmds) | DONE | `data/knowledge.json` |
| Git context detection | DONE | `internal/context/git.go` |
| Project type detection | DONE | `internal/context/project.go` |
| Session tracking | DONE | `internal/context/session.go` |
| Context aggregator | DONE | `internal/context/detector.go` |
| Intelligence tests | DONE | `tests/intelligence_test.go` |
| Wire into search command | DONE | `cmd/search.go` |
| Vault knowledge methods | DONE | `internal/vault/store.go` |
| **PHASE 2 COMPLETE** | **YES** | |

### Phase 3 — Explain + Compose
| Task | Status | File(s) |
|------|--------|---------|
| Command decomposition engine | NOT STARTED | `internal/explain/parser.go` |
| Flag database (200+ tools) | NOT STARTED | `internal/explain/flags.go` |
| `recall explain` command | NOT STARTED | `cmd/explain.go` |
| `recall compose` command | NOT STARTED | `cmd/compose.go` |
| Explain tests | NOT STARTED | `tests/explain_test.go` |
| **PHASE 3 COMPLETE** | **NO** | |

### Phase 4 — UX Polish
| Task | Status | File(s) |
|------|--------|---------|
| Vault browser TUI (Bubbletea) | NOT STARTED | `internal/ui/vault_browser.go` |
| Search TUI (inline) | NOT STARTED | `internal/ui/search.go` |
| `recall vault` command | NOT STARTED | `cmd/vault.go` |
| `recall stats` command | NOT STARTED | `cmd/stats.go` |
| `recall suggest-aliases` command | NOT STARTED | `cmd/suggest.go` |
| Hotkey integration (Ctrl+Space) | NOT STARTED | `pkg/shell/hooks.go` |
| Lipgloss formatting polish | NOT STARTED | `internal/ui/theme.go` |
| **PHASE 4 COMPLETE** | **NO** | |

### Phase 5 — Security + Export
| Task | Status | File(s) |
|------|--------|---------|
| AES-256-GCM encryption | NOT STARTED | `internal/vault/crypto.go` |
| Argon2id key derivation | NOT STARTED | `internal/vault/crypto.go` |
| `recall export` command | NOT STARTED | `cmd/export_cmd.go` |
| `recall import` command | NOT STARTED | `cmd/export_cmd.go` |
| `recall config` command | NOT STARTED | `cmd/config_cmd.go` |
| Fish shell hook | NOT STARTED | `scripts/hooks/recall.fish` |
| **PHASE 5 COMPLETE** | **NO** | |

### Phase 6 — Release
| Task | Status | File(s) |
|------|--------|---------|
| GoReleaser config | NOT STARTED | `.goreleaser.yml` |
| GitHub Actions CI/CD | NOT STARTED | `.github/workflows/` |
| Install script | NOT STARTED | `scripts/install.sh` |
| Knowledge base (500 cmds) | NOT STARTED | `data/knowledge.json` |
| **PHASE 6 COMPLETE** | **NO** | |

---

## KEY DECISIONS MADE

| # | Decision | Reasoning | Date |
|---|----------|-----------|------|
| 1 | Go over Rust | Faster compilation, better CLI ecosystem (Cobra + Bubbletea), simpler concurrency, SQLite is the bottleneck not CPU | 2026-03-25 |
| 2 | modernc.org/sqlite over mattn/go-sqlite3 | Pure Go, no CGo, cross-compiles cleanly to all platforms | 2026-03-25 |
| 3 | FTS5 for search over custom indexing | Built into SQLite, handles tokenization + ranking, sub-ms queries | 2026-03-25 |
| 4 | Shell hook fires AFTER execution, not before | Never blocks user workflow, captures exit code + duration | 2026-03-25 |
| 5 | TOML over YAML for config | Simpler, no whitespace footguns, standard in Go ecosystem | 2026-03-25 |
| 6 | AES-256-GCM for vault encryption | Go stdlib, no external deps, industry standard | 2026-03-25 |
| 7 | Scoring weights: text 30%, intent 25%, freq 20%, context 15%, recency 10% | Balanced between relevance and personalization | 2026-03-25 |
| 8 | Phase 1 uses `recall capture` subcommand instead of Unix socket | Simpler, fast enough (<1ms), socket can be added in Phase 2 if needed | 2026-03-28 |
| 9 | Phase 1 FTS5 search uses OR-joined terms | Simple but effective for keyword matching; Phase 2 adds intent expansion | 2026-03-28 |
| 10 | Column named `binary_name` not `binary` in schema | `binary` is a reserved word in some SQL contexts | 2026-03-28 |
| 11 | Synonym dict as Go map, not external file | Faster, no file I/O, compiles into binary, easy to extend | 2026-03-29 |
| 12 | FTS5 query sanitization in store layer | Prevents double-processing when intelligence engine builds queries | 2026-03-29 |
| 13 | Log-normalized frequency scoring | Prevents power-law dominance by very high-frequency commands | 2026-03-29 |
| 14 | Context scoring uses best-match across all contexts for a command | More accurate than averaging — a command used once in matching context should score high | 2026-03-29 |

---

## HARD RULES — NEVER VIOLATE THESE

### Privacy & Security (NON-NEGOTIABLE)
1. **ZERO network calls.** Recall never contacts any server. No telemetry. No analytics. No update checks (except `recall update-kb` which is explicit and optional).
2. **Secret filtering.** Commands matching these patterns are NEVER stored: `password`, `token`, `secret`, `API_KEY`, `aws_secret`, `credentials`, `private_key`. Also filter: `export PASSWORD=...`, `curl -u user:pass ...`
3. **Vault encryption.** AES-256-GCM at rest. Key derived from Argon2id.
4. **No stdout leaking.** Never print stored commands to logs, debug output, or crash reports unless the user explicitly runs a recall command.
5. **Respect NO_COLOR.** If `NO_COLOR` env var is set, disable all colors.

### Code Quality (ALWAYS FOLLOW)
6. **No panics.** Always return errors. Let the caller handle display.
7. **Wrap all errors.** `fmt.Errorf("context: %w", err)` — every error must have context.
8. **No business logic in cmd/.** Commands are thin wrappers. Logic lives in `internal/`.
9. **Test everything critical.** Scoring algorithm, FTS5 queries, secret filtering, command parsing — all must have tests.
10. **<1ms capture overhead.** The shell hook must be invisible to the user.
11. **<10ms search latency.** Queries must feel instant. Use prepared statements. Use FTS5.

### Architecture (DON'T CHANGE WITHOUT DISCUSSION)
12. **4 layers.** Capture → Intelligence → Surface → Vault. Don't merge them.
13. **Single SQLite file.** Everything in one database at `~/.local/share/recall/vault.db`.
14. **Single binary.** No sidecar processes, no daemons, no background services.
15. **Shell hooks are shell scripts.** The hook is a tiny shell script that calls `recall capture`. Not a Go binary running in the background.

---

## LESSONS LEARNED

| # | Lesson | Context |
|---|--------|---------|
| 1 | modernc.org/sqlite v1.48 requires Go 1.25+, go mod auto-upgraded | Phase 1: go.mod switched from go 1.24.1 to 1.25.0 |
| 2 | FTS5 triggers must be created after the virtual table | Phase 1: schema.go creation order matters |
| 3 | Use `ON CONFLICT(raw) DO UPDATE` for batch imports — much cleaner than check-then-insert | Phase 1: store.go BatchInsertCommands |
| 4 | Parser classifies `npm install express` as subcommand="install express" since npm is a multi-cmd tool — acceptable for Phase 1, refine in Phase 2 | Phase 1: parser tests |
| 5 | FTS5 OR queries fail if "OR" tokens are double-wrapped — store layer must sanitize, not re-wrap | Phase 2: FTS5 query handling between intelligence engine and vault |
| 6 | Knowledge base FTS5 content-sync table needs explicit rebuild after batch insert via transaction | Phase 2: LoadKnowledgeBase calls RebuildKnowledgeFTSIndex after batch |
| 7 | Stop word removal before synonym expansion prevents noise in FTS5 queries | Phase 2: intent.go tokenization pipeline |

---

## SESSION LOG

_(Update this after each work session so the next session knows where we left off.)_

| Session | Date | What Was Done | What's Next |
|---------|------|--------------|-------------|
| 1 | 2026-03-25 | Created all documentation: README, 5 docs files, PROMPT.md, MEMORY.md, .gitignore, LICENSE, cursor rules. No code yet. | Start Phase 1: `go.mod`, `main.go`, Cobra skeleton |
| 2 | 2026-03-28 | Built ENTIRE Phase 1 Foundation: Go scaffold, Cobra CLI (7 commands), SQLite vault with FTS5, capture pipeline (parser, filter, enricher, receiver), shell hooks (zsh+bash), config system (TOML), Lipgloss UI (theme + result cards), 30 tests. All passing. Pushed 8 commits to GitHub. | Start Phase 2: Intent extraction, multi-signal scoring, context detection, knowledge base |
| 3 | 2026-03-29 | Built ENTIRE Phase 2 Intelligence: context detection (git, project, session, env), intent extraction with 80+ synonym entries, 5-signal scoring algorithm (text/intent/freq/context/recency), search orchestrator, pattern extraction, knowledge base loader + 200 curated commands, wired intelligence into search cmd. 49 tests all passing. Pushed to GitHub. | Start Phase 3: Explain + Compose — command decomposition, flag database, explain/compose commands |

---

## HOW TO USE THIS FILE

**At the START of every session:**
1. Read this file FIRST
2. Check CURRENT STATUS to know where we are
3. Check SESSION LOG for what was done last
4. Check the relevant Phase tracker for what's next
5. Start building from where we left off

**At the END of every session:**
1. Update CURRENT STATUS (current phase, last built, next to build)
2. Update the Phase tracker (mark tasks DONE / IN PROGRESS)
3. Add a SESSION LOG entry
4. Add any LESSONS LEARNED
5. Add any KEY DECISIONS that were made
6. Commit the updated MEMORY.md to git

**If you're a new Claude session and have no prior context:**
Read this file + `PROMPT.md` + `docs/ARCHITECTURE.md`. That gives you everything.
