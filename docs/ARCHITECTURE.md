# Architecture — How Recall Works

## Overview

Recall is built around one principle: **the fastest answer is one you already have.**

Most "forgotten" commands are ones you've used before. They're sitting in your shell history — unindexed, unsearchable by intent, buried under noise. Recall captures, enriches, indexes, and retrieves them intelligently.

For commands you've never used, a curated knowledge base fills the gap — offline, instant, no API calls.

---

## System Architecture

```
+---------------------------------------------------------------+
|                        USER SHELL                              |
|                    (bash / zsh / fish)                          |
+------+--------+--------------------------+-----------+---------+
       |        |                          |           |
  Shell Hook   Hotkey Binding         Direct CLI    Tab Complete
  (capture)    (Ctrl+Space)           (recall ...)  (inline)
       |        |                          |           |
       v        v                          v           v
+---------------------------------------------------------------+
|                     RECALL BINARY (Go)                         |
+---------------------------------------------------------------+
|                                                                |
|  +--------------+  +-------------------+  +-----------------+  |
|  |   CAPTURE    |  |   INTELLIGENCE    |  |     SURFACE     |  |
|  |   LAYER      |  |      LAYER        |  |      LAYER      |  |
|  |              |  |                   |  |                 |  |
|  | - Hook recv  |  | - Intent parser   |  | - TUI (bubble)  |  |
|  | - Cmd parse  |  | - Pattern match   |  | - Inline prompt |  |
|  | - Enricher   |  | - Scorer/Ranker   |  | - Formatter     |  |
|  | - Filter     |  | - Knowledge base  |  | - Keybindings   |  |
|  +---------+----+  +--------+----------+  +--------+--------+  |
|            |                |                      |            |
|            v                v                      v            |
|  +--------------------------------------------------------+    |
|  |                   VAULT (Storage)                       |    |
|  |                                                         |    |
|  |  SQLite + FTS5 (Full-Text Search)                       |    |
|  |  - commands table (parsed, enriched, indexed)           |    |
|  |  - patterns table (deduplicated command templates)      |    |
|  |  - contexts table (directory, project, git state)       |    |
|  |  - knowledge table (curated command reference)          |    |
|  |  AES-256 encryption at rest                             |    |
|  +--------------------------------------------------------+    |
|                                                                |
|  CONTEXT DETECTOR                                              |
|  - cwd / git branch / project type / recent cmds / env        |
+---------------------------------------------------------------+
```

---

## The Four Layers

### Layer 1: Capture Layer

The capture layer silently records every command you run.

**How it works:**

1. A tiny shell hook is injected into your `.zshrc` / `.bashrc` / `config.fish`
2. The hook fires **after** each command executes (never blocks your workflow)
3. It sends the command data to the Recall binary via a Unix socket
4. The binary parses, enriches, filters, and stores it

**What gets captured per command:**

| Field | Example | Purpose |
|-------|---------|---------|
| `raw_command` | `docker compose -f dev.yml up -d` | The exact command string |
| `parsed_binary` | `docker` | The base command being used |
| `parsed_subcommand` | `compose up` | The subcommand chain |
| `parsed_flags` | `-f dev.yml`, `-d` | Individual flags and arguments |
| `exit_code` | `0` | Success or failure |
| `cwd` | `/home/user/myapp` | Where it was run |
| `git_branch` | `feature/auth` | Active git branch (if in repo) |
| `git_repo` | `myapp` | Repository name |
| `project_type` | `node` | Detected from package.json, go.mod, etc. |
| `timestamp` | `2026-03-25T14:32:01Z` | When it was run |
| `duration_ms` | `3420` | How long it took |
| `session_id` | `sess_abc123` | Groups commands from same terminal session |

**What gets filtered out (never stored):**

- Commands containing `password`, `token`, `secret`, `API_KEY`, `aws_secret`
- Commands matching user-defined exclusion patterns
- Pure noise: bare `ls`, `cd`, `pwd`, `clear` (configurable)

**Performance:**

- Hook execution: **<1ms** overhead per command
- Data is batched and written asynchronously
- Zero impact on your shell responsiveness

---

### Layer 2: Intelligence Layer (The Brain)

This is where Recall transforms from "fancy history search" to "command intelligence."

**The 4-Stage Scoring Pipeline:**

```
Query: "find big files"
         |
         v
+-------------------------------+
|  STAGE 1: Intent Extraction   |    "find big files"
|  - Tokenize + normalize       |  -> tokens: [find, big, files]
|  - Expand synonyms            |  -> expanded: [find, search, locate,
|  - Detect category            |               big, large, huge, >100M,
|                               |               files, file, directory]
|                               |  -> category: filesystem
+---------------+---------------+
                |
                v
+-------------------------------+
|  STAGE 2: Candidate Fetch     |
|  - FTS5 query on vault        |  -> 47 candidates from history
|  - FTS5 query on knowledge    |  -> 12 candidates from knowledge base
|  - Merge + deduplicate        |  -> 53 unique candidates
+---------------+---------------+
                |
                v
+-------------------------------+
|  STAGE 3: Multi-Signal Score  |   For each candidate:
|                               |
|  textRelevance  (0-1)  x0.30 |   <- TF-IDF + trigram similarity
|  intentMatch    (0-1)  x0.25 |   <- Does it match the parsed intent?
|  personalFreq   (0-1)  x0.20 |   <- How often YOU use this command
|  contextBoost   (0-1)  x0.15 |   <- Does context match? (cwd, project)
|  recency        (0-1)  x0.10 |   <- How recently used
|                        ------+
|  FINAL SCORE    (0-1)        |
+---------------+---------------+
                |
                v
+-------------------------------+
|  STAGE 4: Rank + Present      |
|  - Sort by final score        |  -> Top result: confidence 97%
|  - Format with context        |  -> Show metadata, warnings, tips
|  - Present in TUI             |
+-------------------------------+
```

**Each scoring signal explained:**

**Text Relevance (30%)** — How closely the query matches the command text using TF-IDF weighting and trigram character similarity. This handles typos and partial matches.

**Intent Match (25%)** — Does the command accomplish what the user described? Uses a synonym dictionary and intent templates. "kill process on port 3000" maps to the intent `{action: terminate, target: process, filter: port}`, which matches `lsof -ti:3000 | xargs kill`.

**Personal Frequency (20%)** — How often YOU specifically have used this command. If you've run `docker compose up -d` 200 times, it ranks higher than a generic `docker run` suggestion. This is the personalization signal.

**Context Boost (15%)** — Does the current context match the context where this command was previously used? If you're in a Go project directory and the command was previously used in Go projects, it gets a boost. Considers: cwd, project type, git repo, recent commands.

**Recency (10%)** — More recently used commands get a slight boost. A command used yesterday is more relevant than one used 6 months ago (all else being equal).

---

### Layer 3: Surface Layer (UX)

The surface layer handles how results are presented to the user.

**Three modes of interaction:**

**Mode 1: Direct CLI**
```bash
$ recall "compress a folder"
```
Runs the full pipeline, displays formatted result card in terminal.

**Mode 2: Inline Hotkey (Primary Mode)**
```
Ctrl+Space -> Opens search overlay directly in your shell prompt
```
Like fzf — replaces the prompt temporarily. Type your query, see results live, press Enter to execute. You never leave context.

**Mode 3: TUI Browser**
```bash
$ recall vault
```
Full terminal UI for browsing, searching, filtering your command vault. Built with Bubbletea + Lipgloss for a polished, modern look.

**Result Card Format:**
```
+-- Best Match (confidence: 97%) ---------------------------+
| find . -type f -size +100M -exec ls -lh {} \;            |
|                                                           |
| Context: Used 14 times in ~/projects/                     |
| Last used: 3 days ago                                     |
| Tags: find, disk, cleanup                                 |
|                                                           |
| [Enter] Execute  [e] Edit  [c] Copy  [Tab] More results  |
+-----------------------------------------------------------+
```

---

### Layer 4: Vault (Storage)

All data lives in a single encrypted SQLite database on your machine.

**Database Schema:**

```sql
-- Every command ever captured (deduplicated)
CREATE TABLE commands (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    raw           TEXT NOT NULL,
    binary        TEXT NOT NULL,
    subcommand    TEXT,
    flags         TEXT,           -- JSON array of parsed flags
    category      TEXT,           -- git, docker, filesystem, network, etc.
    frequency     INTEGER DEFAULT 1,
    first_seen    TEXT NOT NULL,
    last_seen     TEXT NOT NULL,
    last_exit     INTEGER,
    avg_duration  REAL
);

-- Full-text search index (FTS5)
CREATE VIRTUAL TABLE commands_fts USING fts5(
    raw, binary, subcommand, category,
    content='commands',
    content_rowid='id'
);

-- Context attached to each command usage
CREATE TABLE contexts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    command_id    INTEGER REFERENCES commands(id),
    cwd           TEXT,
    git_repo      TEXT,
    git_branch    TEXT,
    project_type  TEXT,
    timestamp     TEXT NOT NULL,
    exit_code     INTEGER,
    duration_ms   INTEGER,
    session_id    TEXT
);

-- Curated knowledge base (ships with the binary)
CREATE TABLE knowledge (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    command       TEXT NOT NULL,
    description   TEXT NOT NULL,
    intents       TEXT NOT NULL,   -- JSON array of natural language intents
    category      TEXT NOT NULL,
    flags_doc     TEXT,            -- JSON object of flag explanations
    examples      TEXT,            -- JSON array of usage examples
    danger_level  TEXT DEFAULT 'safe'  -- safe, caution, destructive
);

-- Deduplicated command templates (patterns)
CREATE TABLE patterns (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    template      TEXT NOT NULL,   -- e.g., "docker compose -f {file} up -d"
    frequency     INTEGER DEFAULT 1,
    suggested_alias TEXT
);
```

**Storage location:** `~/.local/share/recall/vault.db`

**Encryption:** AES-256-GCM at rest. Key derived from machine identity (hostname + user + salt). Decrypted only in memory during active use.

**Size estimate:** 50,000 commands ~ 15MB. Negligible.

---

## Context Detection System

The context detector runs on every query and every capture event.

**Signals collected:**

| Signal | How Detected | Used For |
|--------|-------------|----------|
| Current directory | `$PWD` | Match commands to projects |
| Git repository | `.git` directory detection | Group commands by repo |
| Git branch | `git branch --show-current` | Understand workflow stage |
| Git dirty state | `git status --porcelain` | Warn about destructive commands |
| Project type | Presence of `package.json`, `go.mod`, `Cargo.toml`, `requirements.txt`, `Makefile` | Filter suggestions by ecosystem |
| Recent commands | Last 10 commands in session | Predict next likely command |
| Environment | Key env vars (`VIRTUAL_ENV`, `NODE_ENV`, `KUBECONFIG`) | Understand active tooling |
| Time of day | System clock | Minor signal for work patterns |

---

## Data Flow (End to End)

```
User types command
        |
        v
Shell executes command normally (zero interference)
        |
        v
Shell hook fires AFTER execution
        |
        v
Sends to Recall binary: {command, exit_code, cwd, timestamp, duration}
        |
        v
Recall enriches: {+ git_state, + project_type, + session_id}
        |
        v
Recall filters: secret? noise? excluded pattern?
        |
        v
Recall stores: SQLite INSERT or UPDATE frequency
        |
        v
FTS5 index auto-updates

---

User queries: recall "compress folder"
        |
        v
Intent extraction: [compress, folder] -> [compress, zip, tar, archive, folder, directory]
        |
        v
FTS5 search: vault + knowledge base -> candidates
        |
        v
Multi-signal scoring: text + intent + frequency + context + recency
        |
        v
Rank and present top results
        |
        v
User presses Enter -> command executes
```

---

## Explain Engine

The explain feature works by parsing commands against a local flag database.

**Process:**

1. Tokenize command into binary + subcommands + flags + arguments
2. Look up each flag in the flag database (200+ commands covered)
3. Generate human-readable annotation for each component
4. Detect danger flags (`-delete`, `--force`, `rm -rf`, `DROP TABLE`)
5. Add safety warnings and alternative suggestions

**Example:**

Input: `find . -name '*.log' -mtime +30 -delete`

Output:
```
find .                -> search current directory
  -name '*.log'       -> files ending in .log
  -mtime +30          -> modified more than 30 days ago
  -delete             -> [DESTRUCTIVE] permanently delete matches

Warning: This permanently deletes files.
Tip: Remove -delete and add -print first to preview.
```

No API call. No LLM. Pure local parsing. Sub-millisecond.

---

## Performance Targets

| Operation | Target | How Achieved |
|-----------|--------|-------------|
| Shell hook overhead | <1ms | Async write via Unix socket, batched |
| Search query (vault) | <10ms | SQLite FTS5 is microsecond-scale |
| Search query (knowledge) | <5ms | Pre-loaded in memory at startup |
| Full pipeline (query to result) | <50ms | All local, zero network |
| Vault import (50K commands) | <5s | Batch INSERT with transaction |
| Binary startup | <20ms | Go compiles to native binary, no runtime |
| Memory usage (idle) | <10MB | SQLite is on-disk, not in-memory |
| Database size (50K commands) | ~15MB | Compact schema, deduplication |
