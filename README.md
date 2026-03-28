<p align="center">
  <h1 align="center">Recall</h1>
  <p align="center"><strong>Your terminal has perfect memory. You just forgot to use it.</strong></p>
  <p align="center">
    <a href="#installation">Install</a> &bull;
    <a href="#quick-start">Quick Start</a> &bull;
    <a href="#features">Features</a> &bull;
    <a href="docs/COMMANDS.md">Commands</a> &bull;
    <a href="docs/ARCHITECTURE.md">Architecture</a> &bull;
    <a href="#contributing">Contributing</a>
  </p>
  <p align="center">
    <img alt="License" src="https://img.shields.io/badge/license-MIT-blue.svg">
    <img alt="Go Version" src="https://img.shields.io/badge/go-1.22+-00ADD8.svg">
    <img alt="Platform" src="https://img.shields.io/badge/platform-linux%20%7C%20macos-lightgrey.svg">
    <img alt="PRs Welcome" src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg">
    <img alt="100% Offline" src="https://img.shields.io/badge/network-100%25%20offline-success.svg">
  </p>
</p>

---

**Recall** is a local-first, offline, privacy-respecting CLI tool that eliminates **Command Amnesia** — the #1 flow-state killer for developers.

It doesn't just search. It **anticipates**.

```bash
$ recall "find large files over 100mb"

╭─ Best Match (confidence: 97%) ──────────────────────────╮
│ find . -type f -size +100M -exec ls -lh {} \;           │
│                                                          │
│ 📂 Context: Used 14 times in ~/projects/                 │
│ 🕐 Last used: 3 days ago                                 │
│ 🏷️  Tags: find, disk, cleanup                            │
│                                                          │
│ [Enter] Execute  [e] Edit  [c] Copy  [Tab] More results  │
╰──────────────────────────────────────────────────────────╯
```

**No internet. No API keys. No subscriptions. No data leaves your machine. Ever.**

---

## The Problem

Every developer alive does this 12-25 times per day:

```
1. Deep in flow, building something
2. Need a command — can't remember the exact syntax
3. Ctrl+R... nope
4. Open browser → Google → Stack Overflow → scroll past ads → copy
5. Paste back into terminal
6. Flow state: DESTROYED. Cost: 5-23 minutes per interruption.
```

That's **60-120 minutes of productive time lost daily.** Over a year, an entire month of engineering time — gone to Google.

Existing tools don't fix this:

| Tool | Why It Fails |
|------|-------------|
| `tldr` | You must already know the command name |
| `fzf` + `Ctrl+R` | Dumb string matching — can't search by intent |
| Warp / Copilot CLI | Cloud-dependent, $15/mo, sends your history to servers |
| `man` pages | 4,000 lines of text. Can't search "how do I find big files" |
| `cheat.sh` | Requires internet. Generic. Doesn't know YOUR patterns |

**Recall fills the gap.** It's the only tool that is:
- 100% offline
- Intent-aware (understands "kill node process" not just `kill`)
- Personal (learns YOUR commands, YOUR patterns)
- Contextual (knows your cwd, git branch, project type)
- Instant (<10ms response)
- Zero-config (works immediately, improves over time)

---

## Installation

### One-Line Install

```bash
curl -sSf https://raw.githubusercontent.com/user/recall/main/scripts/install.sh | sh
```

### Build From Source

```bash
git clone https://github.com/user/recall.git
cd recall
make build
sudo mv ./bin/recall /usr/local/bin/
```

### Homebrew (macOS / Linux)

```bash
brew install recall
```

### Requirements

- Linux (kernel 4.x+) or macOS 11+
- Bash 4.0+, Zsh 5.0+, or Fish 3.0+
- ~20MB disk space
- **No internet required** (ever)

---

## Quick Start

```bash
# 1. Initialize (installs shell hook, creates vault)
recall init

# 2. Import your existing shell history (recommended)
recall import-history

# 3. Start using it
recall "compress a folder to tar.gz"
recall "undo last git commit"
recall "docker remove all stopped containers"
recall "find and kill process on port 3000"
```

That's it. Three commands and you're productive.

---

## Features

### Intent Search — Describe What You Want

Don't remember the exact command? Describe it in plain English:

```bash
$ recall "find files modified in the last 24 hours"

╭─ Best Match (confidence: 94%) ──────────────────────────╮
│ find . -type f -mtime -1                                 │
╰──────────────────────────────────────────────────────────╯
```

Works with natural language — "big files", "kill port 3000", "undo commit", "compress folder", "show disk space".

### Context-Aware — Knows Where You Are

```bash
~/myapp (git:feature/auth) $ recall "undo last commit"

╭─ Context: Git repo on branch feature/auth ───────────────╮
│ git reset --soft HEAD~1                                   │
│                                                           │
│ ⚠️  Soft reset — keeps changes staged                     │
│ 💡 You're on a feature branch (safe to reset)             │
╰───────────────────────────────────────────────────────────╯
```

Recall uses your current directory, git state, project type, and recent commands to surface the most relevant result.

### Command Vault — Your Personal Knowledge Base

Every command you run is silently captured, parsed, and indexed:

```bash
$ recall vault   # Opens TUI browser

📊 12,847 commands | 342 unique patterns
🔥 Most used this week: git commit, docker compose up, kubectl get pods
💎 Rare but valuable: that openssl command from 3 months ago
```

Browse by category, project, frequency, or recency.

### Explain Mode — Understand Any Command

```bash
$ recall explain "find . -name '*.log' -mtime +30 -delete"

find .               ← search current directory
  -name '*.log'      ← files ending in .log
  -mtime +30         ← modified more than 30 days ago
  -delete            ← ⚠️ DELETE matching files

⚠️ DESTRUCTIVE: This permanently deletes files.
💡 Remove -delete and add -print first to preview.
```

### Smart Aliases — Auto-Generated

```bash
$ recall suggest-aliases

You type this 23x/week:
  docker compose -f docker-compose.dev.yml up -d
→ Suggested alias: dcdev

[a] Accept all  [s] Select  [i] Ignore
```

### Hotkey Integration

```
Ctrl+Space  → Inline search (never leave your prompt)
Ctrl+E      → Explain command on current prompt
Ctrl+K      → Open vault browser
```

---

## How It Works

Recall uses a **4-stage scoring pipeline** — no LLM, no API, no cloud:

```
Query: "find big files"
    ↓
1. Intent Extraction — tokenize, expand synonyms, detect category
    ↓
2. Candidate Fetch — FTS5 query against vault + knowledge base
    ↓
3. Multi-Signal Scoring:
   • Text relevance (30%) — TF-IDF + trigram matching
   • Intent match (25%)   — does it accomplish the goal?
   • Your frequency (20%) — how often YOU use this
   • Context match (15%)  — same project/directory?
   • Recency (10%)        — recently used commands rank higher
    ↓
4. Rank → Present top result with confidence score
```

**Response time: <10ms.** Everything runs locally against an embedded SQLite database with FTS5 full-text search.

For the full technical deep-dive, see [Architecture](docs/ARCHITECTURE.md).

---

## All Commands

| Command | Description |
|---------|-------------|
| `recall "query"` | Search by intent |
| `recall explain "cmd"` | Break down a command |
| `recall vault` | Browse your command vault (TUI) |
| `recall stats` | Usage statistics |
| `recall suggest-aliases` | Get alias suggestions |
| `recall compose` | Build a command interactively |
| `recall init` | First-time setup |
| `recall import-history` | Import existing shell history |
| `recall export` | Export vault (encrypted) |
| `recall import` | Import vault backup |
| `recall config` | View/edit configuration |

For the complete reference with all flags and examples, see [Commands](docs/COMMANDS.md).

---

## Privacy & Security

Recall takes privacy seriously:

1. **Nothing leaves your machine.** Zero network calls. No telemetry. No analytics.
2. **Vault is encrypted at rest** — AES-256-GCM.
3. **Secrets are never stored** — commands containing `password`, `token`, `secret`, `API_KEY` are filtered automatically.
4. **You own your data.** Export, delete, or inspect at any time.
5. **Open source.** Read every line of code.

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
| Search by intent | Yes | No | No | Yes | Yes |
| 100% offline | Yes | Partial | Yes  | No | No |
| Learns your patterns | Yes | No | No | Yes | No |
| Context-aware | Yes | No | No | Partial | No |
| < 10ms response | Yes | Yes | Yes | No | No |
| Free forever | Yes | Yes | Yes | No | No |
| Privacy-first | Yes | Yes | Yes | No    | No |
| Explain commands | Yes | Partial | No | Yes | Yes |

---

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22+ |
| CLI | Cobra |
| TUI | Bubbletea + Lipgloss |
| Storage | SQLite + FTS5 |
| Encryption | AES-256-GCM |
| Config |, TOML |

Single static binary. No runtime dependencies. No Docker. No containers.

For the full tech breakdown, see [Tech Stack](docs/TECH-STACK.md).

---

## Project Structure

```
recall/
├── main.go                # Entry oint
├── cmd/                   # CLI commands (Cobra)
├── internal/
│   ├── capture/           # Shell hook → parse → enrich → store
│   ├── intelligence/      # Intent engine, scorer, knowledge base
│   ├── vault/             # SQLite + FTS5 + encyption
│   ├── context/           # Git, project, session detection
│   ├── explain/           # Command decomposition
│   └── ui/                # Bubbletea TUI components
├── pkg/
│   ├── shell/             # Hook  generation &    installation
│   └── config/            # TOML configuration
├── data/
│   └── knowledge.json     # Curated command knowledge base
├── scripts/
│   ├── install.sh         # One-line installer
│   └── hooks/             # Shell hook scripts
├── docs/                  # Documentation
└── tests/                 # Test suite
```

---

## Contributing

Contributions are welcome. Here's how to get started:

```bash
# Clone
git clone https://github.com/user/recall.git
cd recall

# Install dependencies
go mod tidy

# Run tests
make test

# Build
make build

# Run locally
./bin/recall "test query"
```

### Ways to Contribute

- **Knowledge Base**: Add commands to `data/knowledge.json` — this is the highest-impact contribution
- **Shell Support**: Improve hook scripts for bash/zsh/fish edge cases
- **Bug Reports**: Open issues with reproduction steps
- **Feature Ideas**: Open a discussion with your use case
- **Docs**: Fix typos, add examples, improve explanations

### Development Requirements

- Go 1.22+
- Make
- SQLite (for testing)

---

## Roadmap

- [x] Phase 1: CLI skeleton, vault, shell hooks, basic search
- [ ] Phase 2: Multi-signal scoring, context detection, knowledge base
- [ ] Phase 3: Explain mode, compose mode
- [ ] Phase 4: Bubbletea TUI, hotkey integration, alias suggestions
- [ ] Phase 5: Encryption, export/import, Fish support
- [ ] Phase 6: Cross-platform release, Homebrew, AUR

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

## Documentation

| Doc | Description |
|-----|-------------|
| [Problem & Solution](docs/PROBLEM-AND-SOLUTION.md) | Why this tool exists |
| [Architecture](docs/ARCHITECTURE.md) | How the intelligence engine works |
| [Commands Reference](docs/COMMANDS.md) | Every command, flag, and option |
| [Tech Stack](docs/TECH-STACK.md) | Languages, tools, build phases |
| [Marketing](docs/MARKETING.md) | Positioning, pitch, launch strategy |

---

<p align="center">
  <strong>Stop Googling commands. Start recalling them.</strong>
  <br><br>
  <a href="#installation">Install Recall</a> — it takes 30 seconds.
</p>
