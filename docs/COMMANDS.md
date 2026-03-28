# Commands Reference

Complete reference for every command Recall offers. All commands follow the pattern:

```bash
recall <command> [arguments] [flags]
```

---

## Quick Reference Table

| Command | Short | Description |
|---------|-------|-------------|
| `recall <query>` | — | Search by intent (default action) |
| `recall search <query>` | `recall s` | Explicit search mode |
| `recall explain <command>` | `recall e` | Break down a command into plain English |
| `recall vault` | `recall v` | Browse your command vault (TUI) |
| `recall stats` | — | Show usage statistics |
| `recall suggest-aliases` | `recall sa` | Get alias suggestions for repetitive commands |
| `recall init` | — | First-time setup (install shell hook) |
| `recall import-history` | `recall ih` | Import existing shell history into vault |
| `recall export` | — | Export your vault (encrypted backup) |
| `recall import` | — | Import a vault backup |
| `recall config` | — | View or edit configuration |
| `recall compose` | `recall c` | Build a command interactively step-by-step |
| `recall hook <shell>` | — | Output shell hook script (used internally) |
| `recall update-kb` | — | Update the curated knowledge base |
| `recall version` | — | Show version info |
| `recall help` | `recall -h` | Show help |

---

## Detailed Command Reference

---

### `recall <query>` — Intent Search (Default)

The primary command. Describe what you want in natural language and get the right command back.

**Usage:**
```bash
recall "find large files over 100mb"
recall "undo last git commit"
recall "kill process on port 3000"
recall "compress folder to tar.gz"
recall "show disk usage by directory"
recall "docker remove all stopped containers"
recall "generate ssh key ed25519"
recall "list all open ports"
```

**Output:**
```
+-- Best Match (confidence: 97%) ----------------------------+
| find . -type f -size +100M -exec ls -lh {} \;             |
|                                                            |
| Context: Used 14 times in ~/projects/                      |
| Last used: 3 days ago                                      |
| Tags: find, disk, cleanup                                  |
|                                                            |
| [Enter] Execute  [e] Edit  [c] Copy  [Tab] More results   |
+------------------------------------------------------------+
```

**Actions on result:**
- **Enter** — Execute the command immediately
- **e** — Edit the command before executing (opens in prompt)
- **c** — Copy to clipboard
- **Tab** — Show next result
- **Shift+Tab** — Show previous result
- **q / Esc** — Cancel

**Flags:**
| Flag | Description |
|------|-------------|
| `--top N` | Show top N results (default: 1) |
| `--no-execute` | Show result but don't offer to execute |
| `--json` | Output as JSON (for scripting) |
| `--vault-only` | Search only personal history, skip knowledge base |
| `--kb-only` | Search only knowledge base, skip personal history |
| `--category <cat>` | Filter by category (git, docker, filesystem, network, etc.) |

**Examples:**
```bash
recall "find big files" --top 5
recall "docker cleanup" --vault-only
recall "ssh tunnel" --kb-only --no-execute
recall "git squash" --json | jq '.command'
```

---

### `recall search <query>` — Explicit Search

Same as `recall <query>` but explicit. Useful when your query might look like a subcommand.

**Usage:**
```bash
recall search "how to export database"
recall s "nginx config test"
```

---

### `recall explain <command>` — Command Explainer

Paste any command and get a human-readable breakdown of every component.

**Usage:**
```bash
recall explain "find . -name '*.log' -mtime +30 -not -path '*/node_modules/*' -delete"
```

**Output:**
```
+-- Command Breakdown ----------------------------------------+
|                                                             |
| find .                          <- search current directory |
|   -name '*.log'                 <- files ending in .log     |
|   -mtime +30                    <- modified more than 30    |
|                                    days ago                 |
|   -not -path '*/node_modules/*' <- skip node_modules        |
|   -delete                       <- [!] DELETE matching files|
|                                                             |
| [!] DESTRUCTIVE: This permanently deletes files.            |
| Tip: Remove -delete and add -print first to preview.        |
+-------------------------------------------------------------+
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--short` | One-line summary only |
| `--json` | Output as JSON |
| `--no-warnings` | Skip safety warnings |

**Examples:**
```bash
recall explain "tar -xzvf archive.tar.gz -C /opt/"
recall explain "awk -F: '{print \$1,\$3}' /etc/passwd"
recall explain "rsync -avz --delete ./src/ user@host:/deploy/"
recall e "chmod -R 755 /var/www"
```

---

### `recall vault` — Vault Browser (TUI)

Opens a full terminal UI to browse, search, and manage your stored commands.

**Usage:**
```bash
recall vault
recall v
```

**TUI Features:**
- Full-text search with live filtering
- Browse by category (git, docker, filesystem, network, system, etc.)
- Browse by project / directory
- Sort by: frequency, recency, category
- View command details (contexts, frequency, last used, exit codes)
- Delete entries
- Tag commands manually
- Export selected commands

**Keybindings in TUI:**
| Key | Action |
|-----|--------|
| `/` | Start searching |
| `Enter` | Select command → execute / copy / edit |
| `Tab` | Switch between views (list / categories / projects) |
| `d` | Delete selected entry |
| `t` | Add/edit tags |
| `i` | Show full details |
| `q` / `Esc` | Exit vault |
| `?` | Show help |

**Flags:**
| Flag | Description |
|------|-------------|
| `--category <cat>` | Open filtered to specific category |
| `--project <name>` | Open filtered to specific project |
| `--sort <field>` | Sort by: `frequency`, `recency`, `alpha` |

---

### `recall stats` — Usage Statistics

Shows how you use your terminal — most used commands, patterns, categories.

**Usage:**
```bash
recall stats
```

**Output:**
```
+-- Recall Stats --------------------------------------------+
|                                                            |
| Vault: 12,847 commands captured | 342 unique patterns      |
| Capture period: 47 days                                    |
|                                                            |
| Top Commands (this week):                                  |
|   1. git commit -m "..."              (47 times)           |
|   2. docker compose up -d             (23 times)           |
|   3. kubectl get pods -n staging      (19 times)           |
|   4. go test ./...                    (17 times)           |
|   5. recall "..."                     (14 times)           |
|                                                            |
| Top Categories:                                            |
|   git: 34%  docker: 22%  filesystem: 18%  k8s: 12%        |
|                                                            |
| Rare but Valuable (used < 3 times):                        |
|   1. openssl req -x509 -newkey...     (used 2x, ever)     |
|   2. iptables -A INPUT -p tcp...      (used 1x)           |
|   3. ffmpeg -i input.mp4 -vf...       (used 1x)           |
|                                                            |
| Search Stats:                                              |
|   Queries today: 8                                         |
|   First-result accuracy: 89%                               |
|   Avg query-to-execute: 3.2s                               |
+------------------------------------------------------------+
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--period <days>` | Stats for last N days (default: 7) |
| `--json` | Output as JSON |
| `--all` | Show all-time stats |

---

### `recall suggest-aliases` — Alias Suggestions

Analyzes your command patterns and suggests shell aliases for repetitive commands.

**Usage:**
```bash
recall suggest-aliases
recall sa
```

**Output:**
```
+-- Suggested Aliases ----------------------------------------+
|                                                             |
| You type this 23x/week:                                     |
|   docker compose -f docker-compose.dev.yml up -d            |
| -> Suggested alias: dcdev                                   |
|                                                             |
| You type this 15x/week:                                     |
|   kubectl get pods -n staging -o wide                       |
| -> Suggested alias: kpods                                   |
|                                                             |
| You type this 11x/week:                                     |
|   git log --oneline --graph --all                           |
| -> Suggested alias: glog                                    |
|                                                             |
| [a] Accept all  [s] Select individually  [i] Ignore         |
+-------------------------------------------------------------+
```

**What happens when you accept:**
- Writes alias to your `.zshrc` / `.bashrc` / `config.fish`
- Sources the file so it takes effect immediately
- Stores in vault so aliases persist across `recall export/import`

**Flags:**
| Flag | Description |
|------|-------------|
| `--min-freq <n>` | Only suggest for commands used N+ times/week (default: 5) |
| `--dry-run` | Show suggestions without writing to shell config |
| `--shell <sh>` | Target shell config (auto-detected by default) |

---

### `recall init` — First-Time Setup

Initializes Recall on your system. Run once after installing.

**Usage:**
```bash
recall init
```

**What it does:**
1. Detects your shell (zsh, bash, fish)
2. Installs the capture hook into your shell config
3. Creates the vault database at `~/.local/share/recall/vault.db`
4. Loads the curated knowledge base
5. Offers to import existing shell history

**Flags:**
| Flag | Description |
|------|-------------|
| `--shell <sh>` | Force a specific shell (skip auto-detection) |
| `--no-import` | Skip the history import prompt |
| `--vault-path <path>` | Custom vault location |

---

### `recall import-history` — Import Shell History

Imports your existing shell history into the vault. Gives Recall a head start.

**Usage:**
```bash
recall import-history
recall ih
```

**What it imports:**
- Zsh: `~/.zsh_history` (with timestamps if `EXTENDED_HISTORY` is set)
- Bash: `~/.bash_history`
- Fish: `~/.local/share/fish/fish_history`

**Flags:**
| Flag | Description |
|------|-------------|
| `--file <path>` | Import from a specific history file |
| `--limit <n>` | Import only the last N commands |
| `--dry-run` | Show what would be imported without storing |

---

### `recall export` — Export Vault

Creates an encrypted backup of your vault for backup or transfer to another machine.

**Usage:**
```bash
recall export --output ~/recall-vault.enc
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--output <path>` | Output file path (required) |
| `--password` | Set custom encryption password (prompted securely) |
| `--plain` | Export as unencrypted JSON (not recommended) |
| `--commands-only` | Export only commands, skip config/stats |

---

### `recall import` — Import Vault

Restores a vault from a previously exported backup.

**Usage:**
```bash
recall import --input ~/recall-vault.enc
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--input <path>` | Input file path (required) |
| `--password` | Decryption password (prompted if not provided) |
| `--merge` | Merge with existing vault (default: replace) |

---

### `recall config` — Configuration

View or modify Recall settings.

**Usage:**
```bash
recall config                  # Show current config
recall config set key value    # Set a value
recall config get key          # Get a value
recall config reset            # Reset to defaults
recall config path             # Show config file path
```

**Configurable settings:**

| Key | Default | Description |
|-----|---------|-------------|
| `vault.path` | `~/.local/share/recall/vault.db` | Database location |
| `vault.encrypt` | `true` | Enable/disable encryption |
| `capture.enabled` | `true` | Enable/disable command capture |
| `capture.noise_filter` | `true` | Filter out ls, cd, pwd, clear |
| `capture.secret_patterns` | `["password","token","secret","API_KEY"]` | Patterns to never store |
| `capture.exclude_commands` | `["ls","cd","pwd","clear"]` | Commands to never store |
| `search.top_results` | `1` | Default number of results |
| `search.min_confidence` | `0.3` | Minimum confidence to show |
| `ui.theme` | `default` | TUI color theme |
| `ui.hotkey` | `ctrl+space` | Inline search hotkey |
| `alias.min_frequency` | `5` | Min weekly uses to suggest alias |

**Config file location:** `~/.config/recall/config.toml`

---

### `recall compose` — Interactive Command Builder

Build a command step-by-step through an interactive wizard. For when you know the tool but not the flags.

**Usage:**
```bash
recall compose
recall c
```

**Flow:**
```
$ recall compose

What tool do you want to use?
> find

What do you want to find?
> files

Filter by name pattern? (leave empty to skip)
> *.log

Filter by size? (e.g., +100M, -1K)
> +50M

Filter by modification time? (e.g., +30 days, -7 days)
> +30

What to do with results? [list / delete / move / exec]
> list

Generated command:
  find . -type f -name "*.log" -size +50M -mtime +30 -exec ls -lh {} \;

[Enter] Execute  [e] Edit  [c] Copy  [s] Save to vault
```

---

### `recall hook <shell>` — Output Hook Script

Outputs the shell hook script. Used internally during `recall init`, but can be called manually.

**Usage:**
```bash
recall hook zsh     # Output zsh hook
recall hook bash    # Output bash hook
recall hook fish    # Output fish hook
```

**Typical use (in .zshrc):**
```bash
eval "$(recall hook zsh)"
```

---

### `recall update-kb` — Update Knowledge Base

Updates the curated command knowledge base to the latest version.

**Usage:**
```bash
recall update-kb
```

Downloads the latest `knowledge.json` from the official repository and replaces the local copy. This is the **only** command that makes a network call, and it's completely optional.

---

### `recall version` — Version Info

```bash
$ recall version

recall v0.1.0
  built:   2026-03-25
  go:      1.22.1
  os:      linux/amd64
  vault:   ~/.local/share/recall/vault.db (12,847 entries)
```

---

## Hotkey Bindings

These work inside your running shell session after `recall init`:

| Hotkey | Action |
|--------|--------|
| `Ctrl+Space` | Open inline search (type intent, see results, press Enter to execute) |
| `Ctrl+E` | Explain the command currently typed on the prompt |
| `Ctrl+K` | Open vault browser |

All hotkeys are configurable via `recall config`.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | No results found |
| `3` | Invalid arguments |
| `4` | Vault error (corruption, permission) |
| `5` | Config error |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `RECALL_VAULT` | Override vault database path |
| `RECALL_CONFIG` | Override config file path |
| `RECALL_NO_CAPTURE` | Set to `1` to temporarily disable capture |
| `RECALL_DEBUG` | Set to `1` for verbose debug output |
| `NO_COLOR` | Set to `1` to disable colored output (standard) |
