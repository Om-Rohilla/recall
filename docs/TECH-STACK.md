# Tech Stack, Build Phases & Development Approach

## Development Philosophy: AI-Assisted Engineering (Vibe Coding)

This project is being built using **AI-assisted development** — a modern engineering workflow where human intent drives architecture and design decisions, while AI coding tools handle implementation velocity, debugging, and code generation.

### Tools We Use

| Tool | Role |
|------|------|
| **Claude AI (Anthropic)** | Architecture design, code review, algorithm design, documentation, problem-solving |
| **Cursor IDE** | AI-native code editor with inline completions, multi-file editing, and context-aware suggestions |
| **Claude Code / Claude Chat** | Deep reasoning for complex engineering decisions, refactoring strategies, debugging |
| **GitHub Copilot** | Inline code completion during implementation |

### Why This Approach

Traditional development: Human writes every line, Googles errors, reads docs manually.

AI-assisted development: Human designs the system, defines the architecture, sets quality standards. AI handles boilerplate, catches bugs early, generates tests, writes documentation, and accelerates iteration.

**Result:** 3-5x faster development velocity without sacrificing quality.

### What the AI Does vs. What the Human Does

| Human Decides | AI Assists |
|--------------|-----------|
| What problem to solve | How to implement the solution |
| Architecture & system design | Code generation & boilerplate |
| Feature prioritization | Test generation & edge cases |
| UX decisions & user flow | Documentation writing |
| Code review & quality gates | Debugging & error analysis |
| Release strategy | CI/CD configuration |

The human is the architect. The AI is the builder. Both are essential.

---

## Core Tech Stack

### Language: Go 1.22+

| Aspect | Detail |
|--------|--------|
| **Why Go** | Single binary output, fast compilation, excellent CLI ecosystem, simple concurrency model, readable by any developer |
| **Why not Rust** | Overkill for this use case. Go's performance is more than sufficient (bottleneck is SQLite I/O, not CPU). Rust's learning curve would slow contributors. |
| **Why not Python** | Too slow for startup time (300ms+ vs Go's 5ms). Requires runtime installation. Not ideal for distributing CLI tools. |
| **Why not Node** | Same startup/distribution problems as Python. Not a serious choice for system-level CLI tools. |

### CLI Framework: Cobra

| Aspect | Detail |
|--------|--------|
| **What** | Industry-standard Go CLI framework |
| **Used by** | kubectl, docker, gh, hugo, helm — the biggest CLI tools in the ecosystem |
| **Why** | Subcommand routing, flag parsing, help generation, shell completion — all built in |
| **Package** | `github.com/spf13/cobra` |

### TUI Framework: Bubbletea + Lipgloss

| Aspect | Detail |
|--------|--------|
| **What** | Elm-architecture terminal UI framework + styling library |
| **Used by** | Charm.sh ecosystem, gum, soft-serve |
| **Why** | Beautiful, composable, testable terminal UIs. Makes `recall vault` look premium. |
| **Packages** | `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss` |

### Storage: SQLite + FTS5

| Aspect | Detail |
|--------|--------|
| **What** | Embedded relational database with full-text search |
| **Why SQLite** | Zero-config, serverless, single-file database. Perfect for local-only tools. |
| **Why FTS5** | Full-text search extension built into SQLite. Tokenizes text, builds inverted index, supports ranked queries. Makes intent search sub-millisecond. |
| **Go Driver** | `modernc.org/sqlite` (pure Go, no CGo, cross-compiles cleanly) |
| **Alternative** | `github.com/mattn/go-sqlite3` (CGo, slightly faster, harder to cross-compile) |

### Encryption: Go Standard Library

| Aspect | Detail |
|--------|--------|
| **Algorithm** | AES-256-GCM |
| **Key Derivation** | Argon2id (from machine identity + user-defined password) |
| **Package** | `crypto/aes`, `crypto/cipher`, `golang.org/x/crypto/argon2` |
| **Why stdlib** | No external dependency. Audited, battle-tested. |

### Configuration: TOML

| Aspect | Detail |
|--------|--------|
| **What** | Human-readable configuration format |
| **Why TOML** | Simpler than YAML (no whitespace footguns), richer than INI, standard in Go/Rust ecosystem |
| **Package** | `github.com/BurntSushi/toml` |

### Shell Integration: Custom Hook Scripts

| Shell | Hook File | Mechanism |
|-------|-----------|-----------|
| Zsh | `recall.zsh` | `precmd` / `preexec` hooks |
| Bash | `recall.bash` | `PROMPT_COMMAND` / `trap DEBUG` |
| Fish | `recall.fish` | `fish_postexec` event |

---

## Full Dependency List

### Direct Dependencies (Go Modules)

```
github.com/spf13/cobra          # CLI framework
github.com/charmbracelet/bubbletea  # TUI framework
github.com/charmbracelet/lipgloss   # TUI styling
github.com/charmbracelet/bubbles    # TUI components (text input, list, spinner)
modernc.org/sqlite               # Pure Go SQLite driver
github.com/BurntSushi/toml       # TOML config parser
golang.org/x/crypto              # Argon2 key derivation
```

### No Runtime Dependencies

Recall compiles to a **single static binary**. No runtime, no interpreter, no package manager, no Docker. Download, run, done.

---

## Project File Structure

```
recall/
|
|-- README.md                          # Premium GitHub README (File 5)
|-- LICENSE                            # MIT License
|-- go.mod                             # Go module definition
|-- go.sum                             # Dependency checksums
|-- Makefile                           # Build, test, install, release targets
|-- .gitignore                         # Git ignore rules
|-- .goreleaser.yml                    # Cross-platform release automation
|
|-- main.go                            # Entry point — CLI router
|
|-- cmd/                               # CLI command definitions (Cobra)
|   |-- root.go                        # Root command + global flags
|   |-- search.go                      # recall "query" / recall search
|   |-- vault.go                       # recall vault — TUI browser
|   |-- explain.go                     # recall explain "cmd"
|   |-- hook.go                        # recall hook zsh|bash|fish
|   |-- capture.go                     # recall capture (called by hook)
|   |-- suggest.go                     # recall suggest-aliases
|   |-- stats.go                       # recall stats
|   |-- config_cmd.go                  # recall config
|   |-- compose.go                     # recall compose (interactive builder)
|   |-- export_cmd.go                  # recall export / import
|   '-- init_cmd.go                    # recall init
|
|-- internal/                          # Private application logic
|   |-- capture/
|   |   |-- receiver.go                # Receives commands from shell hook
|   |   |-- parser.go                  # Parses raw command into structured form
|   |   |-- enricher.go                # Adds context (git, cwd, project type)
|   |   '-- filter.go                  # Secret detection + noise filtering
|   |
|   |-- intelligence/
|   |   |-- engine.go                  # Core search + matching orchestrator
|   |   |-- intent.go                  # Natural language -> command intent
|   |   |-- scorer.go                  # Multi-signal relevance scoring
|   |   |-- patterns.go                # Command template extraction
|   |   '-- knowledge.go               # Built-in knowledge base loader
|   |
|   |-- vault/
|   |   |-- store.go                   # SQLite operations (CRUD + FTS5)
|   |   |-- models.go                  # Data structures (Command, Context, etc.)
|   |   |-- schema.go                  # Database schema + migrations
|   |   |-- index.go                   # FTS5 index management
|   |   '-- crypto.go                  # AES-256 encryption / decryption
|   |
|   |-- context/
|   |   |-- detector.go                # Aggregates all context signals
|   |   |-- git.go                     # Git repo state detection
|   |   |-- project.go                 # Project type detection
|   |   '-- session.go                 # Recent command session tracking
|   |
|   |-- explain/
|   |   |-- parser.go                  # Command decomposition engine
|   |   '-- flags.go                   # Flag database for 200+ tools
|   |
|   '-- ui/
|       |-- search.go                  # Interactive search TUI
|       |-- vault_browser.go           # Vault browsing TUI
|       |-- result.go                  # Result card rendering
|       '-- theme.go                   # Colors, borders, styling
|
|-- pkg/                               # Public, reusable packages
|   |-- shell/
|   |   |-- detect.go                  # Detect current shell
|   |   |-- hooks.go                   # Generate shell hook scripts
|   |   '-- install.go                 # Hook installer (writes to rc files)
|   |
|   '-- config/
|       |-- config.go                  # User configuration (TOML read/write)
|       '-- defaults.go                # Default configuration values
|
|-- data/
|   '-- knowledge.json                 # Curated knowledge base (~500 commands)
|
|-- scripts/
|   |-- install.sh                     # curl-pipe installer for end users
|   '-- hooks/
|       |-- recall.zsh                 # Zsh hook script
|       |-- recall.bash                # Bash hook script
|       '-- recall.fish                # Fish hook script
|
|-- docs/                              # Documentation
|   |-- PROBLEM-AND-SOLUTION.md        # File 1: Problem & Solution
|   |-- ARCHITECTURE.md                # File 2: How it works
|   |-- COMMANDS.md                    # File 3: Full CLI reference
|   |-- TECH-STACK.md                  # File 4: This file
|   '-- MARKETING.md                   # File 6: Marketing & launch
|
'-- tests/
    |-- intelligence_test.go           # Intent matching + scoring tests
    |-- capture_test.go                # Capture pipeline tests
    |-- vault_test.go                  # Storage + FTS5 tests
    |-- explain_test.go                # Explain engine tests
    '-- testdata/
        |-- sample_history.txt         # Test fixture: sample shell history
        '-- sample_commands.json       # Test fixture: command samples
```

---

## Build Phases

### Phase 1 — Foundation (Weeks 1-2)

**Goal:** Working CLI that can capture commands and search history.

**Deliverables:**
- Go project scaffold with Cobra CLI
- SQLite vault with FTS5 full-text search
- Shell hook for Zsh and Bash (capture after execution)
- `recall init` — first-time setup
- `recall import-history` — import existing shell history
- Basic search: `recall "query"` using FTS5 against vault
- Formatted result card display

**Technical focus:**
- `main.go` + `cmd/` package setup
- `internal/vault/` — SQLite schema, CRUD, FTS5 queries
- `internal/capture/` — hook receiver, basic parser
- `pkg/shell/` — hook generation and installation
- `scripts/hooks/` — shell hook scripts

**Exit criteria:** User can install, import history, search by keywords, and get results.

---

### Phase 2 — Intelligence (Weeks 3-4)

**Goal:** Search that understands intent, not just keywords.

**Deliverables:**
- Intent extraction engine (tokenize, normalize, expand synonyms)
- Multi-signal scoring algorithm (text + intent + frequency + context + recency)
- Context detection (cwd, git state, project type, recent commands)
- Curated knowledge base (`data/knowledge.json`) — first 200 commands
- Results ranked by confidence score

**Technical focus:**
- `internal/intelligence/` — engine, intent parser, scorer
- `internal/context/` — git, project, session detection
- `data/knowledge.json` — manual curation of command intents
- Synonym dictionary for intent expansion

**Exit criteria:** `recall "kill process on port 3000"` returns `lsof -ti:3000 | xargs kill` with high confidence.

---

### Phase 3 — Explain + Compose (Weeks 5-6)

**Goal:** Users can understand any command and build new ones interactively.

**Deliverables:**
- `recall explain "command"` — flag-by-flag breakdown
- `recall compose` — interactive command wizard
- Flag database for 200+ common tools (find, tar, grep, awk, sed, docker, git, kubectl, ssh, curl, etc.)
- Danger detection (destructive flags highlighted)

**Technical focus:**
- `internal/explain/` — command decomposition, flag database
- `cmd/explain.go`, `cmd/compose.go`
- `data/knowledge.json` — expand with flag documentation

**Exit criteria:** Any command can be pasted and explained. User can build `find` and `tar` commands interactively.

---

### Phase 4 — UX Polish (Weeks 7-8)

**Goal:** Premium terminal experience.

**Deliverables:**
- Bubbletea TUI for `recall vault` (full browser)
- Inline search hotkey (`Ctrl+Space`)
- `recall suggest-aliases` — alias generation
- `recall stats` — usage statistics
- Beautiful formatting with Lipgloss (colors, borders, icons)
- Result cards with confidence scores, context, and action hints

**Technical focus:**
- `internal/ui/` — all TUI components
- `cmd/suggest.go`, `cmd/stats.go`
- Theme system, color palette

**Exit criteria:** The tool looks and feels like a premium, polished product.

---

### Phase 5 — Security & Export (Weeks 9-10)

**Goal:** Vault encryption, export/import, Fish shell support.

**Deliverables:**
- AES-256-GCM encryption at rest for vault
- `recall export` / `recall import` — encrypted vault transfer
- Fish shell hook support
- Secret filtering (configurable patterns)
- `recall config` — full configuration management

**Technical focus:**
- `internal/vault/crypto.go`
- `cmd/export_cmd.go`, `cmd/config_cmd.go`
- `scripts/hooks/recall.fish`

**Exit criteria:** Vault is encrypted, transferable between machines, and Fish users are supported.

---

### Phase 6 — Release & Distribution (Weeks 11-12)

**Goal:** Ready for public release.

**Deliverables:**
- GoReleaser config for cross-platform builds (Linux amd64/arm64, macOS amd64/arm64)
- `install.sh` — curl-pipe installer
- GitHub Actions CI/CD (test, build, release)
- Homebrew formula
- AUR package (Arch Linux)
- Comprehensive README with GIF demos
- Knowledge base expanded to 500+ commands

**Technical focus:**
- `.goreleaser.yml`
- `.github/workflows/` — CI/CD pipeline
- `scripts/install.sh`
- Final testing across shells and OS versions

**Exit criteria:** Anyone can install with one command and start using immediately.

---

## Testing Strategy

| Layer | Tool | What We Test |
|-------|------|-------------|
| Unit tests | `go test` | Intent parsing, scoring algorithm, FTS5 queries, command parser, explain engine |
| Integration tests | `go test` + temp SQLite | Full pipeline: capture -> store -> search -> rank |
| Shell tests | `bats` (Bash testing) | Hook installation, capture flow, shell compatibility |
| Fuzzing | `go test -fuzz` | Intent parser handles arbitrary input without panic |
| Benchmarks | `go test -bench` | Search latency < 10ms, capture overhead < 1ms |

---

## CI/CD Pipeline

```
Push to main / PR
       |
       v
GitHub Actions
       |
       +-- go vet / golint / staticcheck (linting)
       +-- go test ./... (unit + integration tests)
       +-- go test -race ./... (race condition detection)
       +-- go test -bench ./... (performance regression)
       |
       v (on tag push)
GoReleaser
       |
       +-- Build: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
       +-- Package: tar.gz, deb, rpm
       +-- Publish: GitHub Releases
       +-- Update: Homebrew tap, AUR PKGBUILD
```

---

## Supported Platforms

| Platform | Architecture | Status |
|----------|-------------|--------|
| Linux (Ubuntu, Debian, Fedora, Arch) | amd64 | Primary |
| Linux | arm64 | Supported |
| macOS | amd64 (Intel) | Supported |
| macOS | arm64 (Apple Silicon) | Supported |
| Windows (WSL2) | amd64 | Best-effort |

---

## Minimum System Requirements

| Resource | Requirement |
|----------|-------------|
| OS | Linux kernel 4.x+ / macOS 11+ |
| Shell | Bash 4.0+, Zsh 5.0+, or Fish 3.0+ |
| Disk | ~20MB (binary + vault) |
| RAM | ~10MB during use |
| Network | None (100% offline) |
