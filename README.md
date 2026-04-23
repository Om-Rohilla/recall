<p align="center">
  <h1 align="center">⚡ Recall</h1>
  <p align="center"><strong>Your terminal has perfect memory.</strong></p>
  <p align="center">
    <a href="#installation">Install</a> &bull;
    <a href="#quick-start">Quick Start</a> &bull;
    <a href="#features">Features</a> &bull;
    <a href="#all-commands">Commands</a> &bull;
    <a href="#how-it-works">Architecture</a> &bull;
    <a href="#contributing">Contributing</a>
  </p>
  <p align="center">
    <a href="https://github.com/Om-Rohilla/recall/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/Om-Rohilla/recall/actions/workflows/ci.yml/badge.svg"></a>
    <a href="https://github.com/Om-Rohilla/recall/actions/workflows/release.yml"><img alt="Release" src="https://github.com/Om-Rohilla/recall/actions/workflows/release.yml/badge.svg"></a>
    <img alt="Version" src="https://img.shields.io/github/v/release/Om-Rohilla/recall?color=success">
    <img alt="License" src="https://img.shields.io/badge/license-MIT-blue.svg">
    <img alt="Go Version" src="https://img.shields.io/badge/go-1.24+-00ADD8.svg">
    <img alt="Platform" src="https://img.shields.io/badge/platform-linux%20%7C%20macos-lightgrey.svg">
    <img alt="100% Offline" src="https://img.shields.io/badge/network-100%25%20offline-success.svg">
  </p>
</p>

---

**Recall** is a local-first, fully offline, privacy-first CLI tool that eliminates **Command Amnesia** — the #1 flow-state killer for developers and engineers.

It doesn't just search your history. It **understands intent**.

```bash
$ recall "find large files over 100mb"

╭─ Best Match (confidence: 97%) ──────────────────────────╮
│ find . -type f -size +100M -exec ls -lh {} \;           │
│                                                          │
│ 📂 Used 14 times in ~/projects/                          │
│ 🕐 Last used: 3 days ago                                 │
│ 🏷️  Tags: find, disk, cleanup                            │
│                                                          │
│ [Enter] Execute  [e] Edit  [c] Copy  [Tab] More results  │
╰──────────────────────────────────────────────────────────╯
```

**No internet. No API keys. No subscriptions. Nothing ever leaves your machine.**

---

## The Problem

Every developer does this 12–25 times per day:

```
1. Deep in flow, building something important
2. Need a command — can't remember the exact syntax
3. Ctrl+R... nope
4. Open browser → Google → Stack Overflow → scroll past ads → copy-paste
5. Paste back into terminal
6. Flow state: DESTROYED. Cost: 5–23 minutes per interruption.
```

That's **60–120 minutes of productive time lost every day.** Over a year, an entire month of engineering time — gone to Google.

Existing tools don't fix this:

| Tool | Why It Fails |
|------|-------------|
| `tldr` | You must already know the command name |
| `fzf` + `Ctrl+R` | Dumb string matching — no intent understanding |
| Warp / Copilot CLI | Cloud-dependent, $15/mo, sends your history to servers |
| `man` pages | 4,000 lines of text for a one-line answer |
| `cheat.sh` | Requires internet. Generic. Doesn't know YOUR patterns |

**Recall fills the gap.** The only tool that is:
- ✅ 100% offline — zero network calls, ever
- ✅ Intent-aware — understands "kill node process", not just `kill`
- ✅ Personal — learns YOUR commands and YOUR patterns
- ✅ Contextual — knows your cwd, git branch, project type
- ✅ Instant — <10ms response time on local SQLite+FTS5
- ✅ Zero-config — works immediately after `recall init`

---

## Installation

### Homebrew (macOS / Linux) — Recommended

```bash
brew tap Om-Rohilla/tap
brew install recall
```

### One-Line Install Script

```bash
curl -sSf https://raw.githubusercontent.com/Om-Rohilla/recall/main/scripts/install.sh | sh
```

> **macOS users:** Binary is not yet notarized (planned for v1.1). The installer automatically removes the Gatekeeper quarantine flag. If you install manually, run:
> ```sh
> xattr -d com.apple.quarantine /usr/local/bin/recall
> ```

### Download Binary

Pre-built binaries for every platform are on the [Releases page](https://github.com/Om-Rohilla/recall/releases/latest):

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `recall_*_macOS_arm64.tar.gz` |
| macOS (Intel) | `recall_*_macOS_x86_64.tar.gz` |
| Linux (x86_64) | `recall_*_linux_x86_64.tar.gz` |

### Build From Source

```bash
git clone https://github.com/Om-Rohilla/recall.git
cd recall
CGO_ENABLED=1 go build -tags sqlite_fts5 -o recall .
sudo mv recall /usr/local/bin/
```

### Requirements

- macOS 11+ or Linux (kernel 4.x+)
- Bash 4.0+, Zsh 5.0+, or Fish 3.0+
- ~30MB disk space
- **No internet connection required — ever**

---

## Quick Start

```bash
# 1. Initialize (installs shell hook, creates encrypted vault)
recall init

# 2. Restart your shell (or source your rc file)
source ~/.zshrc   # or ~/.bashrc / ~/.config/fish/config.fish

# 3. Start using it immediately
recall "compress a folder to tar.gz"
recall "undo last git commit"
recall "docker remove all stopped containers"
recall "find and kill process on port 3000"
```

> **Tip:** After `recall init`, every command you run is automatically captured and indexed in the background — you don't need to do anything.

---

## Features

### 🔍 Intent Search — Describe What You Want

Don't remember the exact command? Just describe it:

```bash
$ recall "find files modified in the last 24 hours"

╭─ Best Match (confidence: 94%) ──────────────────────────╮
│ find . -type f -mtime -1                                 │
╰──────────────────────────────────────────────────────────╯
```

Interactive TUI lets you navigate, preview, copy, or execute results directly.

### 🌍 Context-Aware — Knows Where You Are

```bash
~/myapp (git:feature/auth) $ recall "undo last commit"

╭─ Context: Git repo · branch: feature/auth ──────────────╮
│ git reset --soft HEAD~1                                   │
│                                                           │
│ ⚠️  Soft reset — keeps changes staged                     │
│ 💡 You're on a feature branch (safe to reset)             │
╰───────────────────────────────────────────────────────────╯
```

Uses your current directory, git state, project type, and recent command history to surface the most relevant result.

### 🔐 Command Vault — AES-256 Encrypted

Every command you run is silently captured, parsed for secrets, and stored in an encrypted SQLite vault:

```bash
$ recall vault   # Opens interactive TUI browser

📊 12,847 commands  |  342 unique patterns  |  🔒 Encrypted
🔥 Most used this week: git commit, docker compose up, kubectl get pods
💎 Rare but valuable: that openssl command from 3 months ago
```

Browse, filter, and manage your history by category, project, frequency, or recency.

> **Backup tip:** The vault is a single SQLite file (default `~/.local/share/recall/vault.db`). In WAL mode, a complete backup requires copying both `vault.db` **and** any `vault.db-wal` file while Recall is not running. Use `recall export` for a safe, encrypted backup.

### 🧠 Explain Mode — Understand Any Command

```bash
$ recall explain "find . -name '*.log' -mtime +30 -delete"

find .               ← search current directory
  -name '*.log'      ← files ending in .log
  -mtime +30         ← modified more than 30 days ago
  -delete            ← ⚠️ DELETE matching files permanently

⚠️  DESTRUCTIVE: This permanently deletes files.
💡 Remove -delete and add -print first to preview.
```

### 💡 Smart Aliases — Auto-Generated From Your Patterns

```bash
$ recall suggest-aliases

You type this 23×/week:
  docker compose -f docker-compose.dev.yml up -d
→ Suggested alias: dcdev

[a] Accept all  [s] Select  [i] Ignore
```

### 📦 Compose Mode — Build Commands Interactively

Guided wizard for constructing complex commands from parts.

### 📊 Stats & Wrapped — Know Your Terminal Habits

```bash
$ recall stats          # Usage stats, streaks, top commands
$ recall wrapped        # Weekly Terminal Wrapped — your CLI story
$ recall roast          # AI-style playful roast of your CLI habits
```

### 🔗 Share — Generate Beautiful Command Cards

```bash
$ recall share "git log --oneline --graph --all"
✅ SVG card generated → ./recall_share_git_log.svg
```

Generates a gorgeous, shareable SVG image of any command for tweets, blog posts, or documentation.

### ☁️ Sync — E2E Encrypted Cross-Device Sync

```bash
$ recall sync           # Sync vault across all your machines
```

Uses GitHub Gists as a zero-infrastructure backend. Your vault is encrypted **before** it leaves your device — even GitHub cannot read your data.

### ⌨️ Hotkey Integration

| Shortcut | Action |
|----------|--------|
| `Ctrl+Space` | Inline search — never leave your prompt |
| `Ctrl+E` | Explain command at current prompt |
| `Ctrl+K` | Open vault browser |

### 🔌 Plugin System — Pure WASM Sandbox

Extend Recall without touching your file system or network:

```bash
recall plugin install my-plugin.wasm
recall plugin list
recall plugin run my-plugin
```

Plugins run in a **pure WebAssembly (WASI)** sandbox via `wazero`. Zero file system access. Zero network access. Zero environment variable leakage.

---

## Privacy & Security

Recall was designed from the ground up to be privacy-first. Here's exactly how:

| Guarantee | Implementation |
|-----------|---------------|
| **Nothing leaves your machine** | Zero network calls in core binary. `recall sync` uses E2E encryption before upload. |
| **Vault encrypted at rest** | AES-256-GCM via SQLCipher. Key stored in OS keyring with file-based fallback. |
| **Secrets never stored** | 50+ pattern filter blocks `password=`, `token=`, JWTs, API keys, PATs, Stripe keys, hex strings, env exports. |
| **FTS5 injection hardened** | All search queries sanitized — operator injection is not possible. |
| **Crash-safe** | SIGTERM/SIGINT handlers wipe any in-memory sensitive data before exit. |
| **Rate limited** | 50ms minimum capture interval prevents flood/DoS from scripts. |
| **Plugin isolation** | WASM sandbox — plugins cannot access FS, network, or env vars. |
| **Open source** | Every line is auditable at github.com/Om-Rohilla/recall. |

---

## All Commands

| Command | Description |
|---------|-------------|
| `recall "query"` | Search by intent (main command) |
| `recall explain "cmd"` | Break down any command with safety warnings |
| `recall vault` | Browse your personal command vault (interactive TUI) |
| `recall stats` | Usage statistics, streaks, and top patterns |
| `recall wrapped` | Weekly Terminal Wrapped summary |
| `recall roast` | Playful roast of your CLI habits |
| `recall share "cmd"` | Generate a shareable SVG card of a command |
| `recall suggest-aliases` | Get smart alias suggestions from your patterns |
| `recall compose` | Interactively build complex commands |
| `recall sync` | E2E encrypted cross-device sync via GitHub Gists |
| `recall init` | First-time setup — vault creation + shell hook install |
| `recall import-history` | Import existing shell history into vault |
| `recall export` | Export vault backup (encrypted) |
| `recall import` | Import vault backup |
| `recall config` | View and edit configuration |
| `recall plugin list\|install\|run` | Manage WASM plugins |
| `recall version` | Show version and build info |
| `recall completion` | Generate shell completion scripts |

Run `recall --help` or `recall <command> --help` for full flag reference.

---

## How It Works

Recall uses a **4-stage local scoring pipeline** — no LLM, no API, no cloud:

```
Query: "find big files"
    ↓
1. Intent Extraction — tokenize, expand synonyms, detect category
    ↓
2. Candidate Fetch — FTS5 full-text search across vault + 500+ knowledge base entries
    ↓
3. Multi-Signal Scoring:
   • Text relevance   (30%) — TF-IDF + trigram matching
   • Intent match     (25%) — does it accomplish the stated goal?
   • Personal freq    (20%) — how often YOU use this command
   • Context match    (15%) — same project/directory/git branch?
   • Recency          (10%) — recently used commands rank higher
    ↓
4. Rank → Present top result with confidence score
```

**Response time: <10ms.** Everything runs locally against an embedded SQLite database with FTS5 full-text search. No model inference. No API round-trips.

---

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.24+ |
| CLI framework | Cobra |
| TUI | Bubbletea + Lipgloss |
| Storage | SQLite + FTS5 (via go-sqlcipher, AES-256) |
| Encryption | AES-256-GCM (SQLCipher) + Argon2id (exports) |
| Key storage | OS keyring (zalando/go-keyring) |
| Sync backend | GitHub Gists (E2E encrypted) |
| Plugin runtime | wazero (pure WASM/WASI) |
| Config format | TOML |
| Shell hooks | Bash / Zsh / Fish |
| Release | GoReleaser + Homebrew tap |

Single static binary. No runtime dependencies. No Docker. No daemons.

---

## Project Structure

```
recall/
├── main.go                # Entry point
├── cmd/                   # All CLI commands (Cobra)
│   ├── search.go          # Main intent search
│   ├── vault.go           # TUI vault browser
│   ├── explain.go         # Command explainer
│   ├── compose.go         # Interactive composer
│   ├── share.go           # SVG card generator
│   ├── sync_cmd.go        # E2E encrypted sync
│   ├── stats.go           # Usage statistics
│   ├── wrapped.go         # Weekly wrapped
│   ├── roast.go           # CLI habit roast
│   ├── suggest.go         # Alias suggestions
│   ├── plugin_cmd.go      # WASM plugin manager
│   └── ...
├── internal/
│   ├── capture/           # Shell hook → parse → secret-filter → store
│   ├── intelligence/      # Scoring engine + knowledge base
│   ├── vault/             # SQLite + FTS5 + AES-256 encryption
│   ├── context/           # Git, project, session detection
│   ├── explain/           # Command decomposition
│   ├── compose/           # Interactive wizard
│   └── ui/                # Bubbletea TUI components
├── pkg/
│   ├── shell/             # Hook generation & installation
│   ├── config/            # TOML configuration
│   ├── plugin/            # WASM sandbox runner
│   ├── logging/           # Structured logging
│   └── errors/            # Typed error handling
├── data/
│   └── knowledge.json     # 500+ curated command knowledge base entries
├── scripts/
│   ├── install.sh         # One-line installer
│   └── hooks/             # recall.bash / recall.zsh / recall.fish
├── tests/                 # Integration + security test suite
└── .goreleaser.yaml       # Multi-platform release configuration
```

---

## Comparison

```
                    HIGH PERSONALIZATION
                           ▲
                           │
                    Recall ★  ← you are here
                           │
          Navi ●           │           ● Warp AI
                           │
     ──────────────────────┼──────────────────────►
     OFFLINE / PRIVATE     │     CLOUD / CONNECTED
                           │
          tldr ●           │           ● Copilot CLI
                           │
          fzf ●            │           ● cheat.sh
                           │
                    LOW PERSONALIZATION
```

| Feature | Recall | tldr | fzf | Warp | Copilot CLI |
|---------|--------|------|-----|------|-------------|
| Search by intent | ✅ | ❌ | ❌ | ✅ | ✅ |
| 100% offline | ✅ | Partial | ✅ | ❌ | ❌ |
| Learns your patterns | ✅ | ❌ | ❌ | ✅ | ❌ |
| Context-aware | ✅ | ❌ | ❌ | Partial | ❌ |
| < 10ms response | ✅ | ✅ | ✅ | ❌ | ❌ |
| Free forever | ✅ | ✅ | ✅ | ❌ | ❌ |
| Privacy-first | ✅ | ✅ | ✅ | ❌ | ❌ |
| Encrypted vault | ✅ | ❌ | ❌ | ❌ | ❌ |
| Explain commands | ✅ | Partial | ❌ | ✅ | ✅ |
| Plugin system | ✅ | ❌ | ❌ | ✅ | ❌ |

---

## Contributing

Contributions are welcome! Here's how:

```bash
git clone https://github.com/Om-Rohilla/recall.git
cd recall
go mod tidy
go test -tags sqlite_fts5 ./...
CGO_ENABLED=1 go build -tags sqlite_fts5 -o recall .
./recall "test query"
```

### Highest Impact Contributions

- **Knowledge Base** — Add entries to `data/knowledge.json` (one entry = instant value for all users)
- **Secret Filter Patterns** — Expand `internal/capture/filter.go` with new sensitive data patterns
- **Shell Hook Edge Cases** — Improve `scripts/hooks/` for unusual bash/zsh/fish configurations
- **Bug Reports** — Open issues with exact reproduction steps and OS/shell version

### Development Requirements

- Go 1.24+
- GCC (for CGO / SQLCipher)
- `libsqlite3-dev` (Linux only)

---

## Roadmap

- [x] Phase 1: CLI skeleton, vault, shell hooks, basic search
- [x] Phase 2: Multi-signal scoring, context detection, 500+ knowledge base
- [x] Phase 3: Explain mode, compose mode, interactive TUI
- [x] Phase 4: Bubbletea TUI, hotkey integration, alias suggestions
- [x] Phase 5: AES-256 encryption, export/import, Fish shell support
- [x] Phase 6: GoReleaser CI/CD, Homebrew tap, multi-platform binaries
- [x] Phase 7: Security hardening — FTS5 injection prevention, 50+ secret filter patterns, rate limiting, crash-safe signal handling
- [x] Phase 8: Ghost Cloud Sync (E2E), `recall share` SVG cards, `recall roast`, zero-I/O DB locking, WASM plugin sandbox — **v1.0 Production Release**
- [ ] Phase 9: Windows support, `ripgrep`-based search fallback, plugin marketplace

---

## License

MIT License — see [LICENSE](LICENSE) for details.

<p align="center">
  <strong>Stop Googling commands. Start recalling them.</strong>
  <br><br>
  <a href="#installation">Install Recall</a> — it takes 30 seconds.
  <br><br>
  <a href="https://github.com/Om-Rohilla/recall/releases/latest">⬇️ Download Latest Release</a>
  &nbsp;|&nbsp;
  <a href="https://github.com/Om-Rohilla/recall/issues/new">🐛 Report a Bug</a>
  &nbsp;|&nbsp;
  <a href="https://github.com/Om-Rohilla/recall/discussions">💬 Discussions</a>
</p>
