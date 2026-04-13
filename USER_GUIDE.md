# Recall: The Ultimate User Guide

Whether you are a **Student** learning the command line for the first time, or a **Senior Engineer** managing production clusters, Recall is designed to act as your terminal's secondary brain. 

This guide walks you through exactly how you should be using Recall depending on your role, and provides a complete playbook of every command at your disposal.

---

## 👩‍🎓 How Students Use Recall (The "Learning Engine")

When you're learning computer science or web development, the terminal can be terrifying. You have to memorize flags for `git`, `docker`, `npm`, and `linux` tools. 

Instead of writing commands down in a physical notebook or saving them in a messy text file, let Recall remember them for you automatically.

### 1. Stop Memorizing, Start Describing
You learn a command in class or a tutorial: `chmod +x script.sh`. 
A week later, you need to make another script executable. You don't need to Google "how to make file executable linux". 
Just ask Recall what your intent is:
```bash
$ recall "make script executable"
▸ chmod +x start-server.sh  (95% match)
```

### 2. Understand What You Are Running
If you copy a complex command from StackOverflow like `find . -name "*.log" -delete`, **do not run it blindly**. Have Recall explain it to you safely first:
```bash
$ recall explain "find . -name '*.log' -delete"
```
*Recall will break down every single flag and explicitly warn you that `-delete` permanently destroys files.*

### 3. Learn Your Own Habits
At the end of the week, see what you've learned.
```bash
$ recall digest
```
*Shows your command streak and the most common things you did this week.*

---

## 👨‍💻 How Engineers Use Recall (The "Safety Net")

For professionals, Recall is about **Flow State**, **Speed**, and **Safety**. 

### 1. The Complex One-Liner
You just spent 20 minutes crafting a perfect `kubectl` or `aws` CLI command with complex JSON filtering. You run it. It works.
**You don't need to document it.** Recall automatically saved it, attached your current project folder, and tracked that it exited successfully.

### 2. Incident Response (MTTR Reduction)
Production is down. The rollback command is buried in a Slack thread from three months ago. Don't panic.
```bash
$ recall "rollback prod database migration"
▸ flyway migrate -url=jdbc... -target=4   (92% match)
```

### 3. Exporting Knowledge to the Team
You are leaving a project, or a new junior engineer is joining. Export your entire command history securely, password protect it, and give it to them so they have all your muscle memory on Day 1.
```bash
$ recall export --output project-knowledge.enc
```

---

## 🚀 The Recall Journey (Day 1 to 30)

### Day 1: The Setup
1. **Initialize:** `recall init` (sets up your vault and hooks).
2. **Import History:** `recall import-history` (brings in your legacy `~/.bash_history` or `~/.zsh_history`).
3. **Run your first intent search:** `recall "list files sorted by size"`

### Day 7: Exploring Your Habits
After a week, Recall has learned your personal patterns.
- Browse everything visually: `recall vault` (Opens the Terminal UI dashboard).
- See your stats: `recall stats` (Shows how many commands you run, your most used binaries).
- Get alias suggestions: `recall suggest-aliases` (Recall analyzes what you type most and suggests bash/zsh aliases to save you keystrokes).

### Day 30: Power User
You heavily rely on Recall.
- Securely backup your vault: `recall export --output my_vault.enc`
- Sync to the cloud privately: `recall sync --push` 
- Share a beautiful snippet with a coworker: `recall share "docker compose up -d --build"` (Generates a gorgeous SVG graphic you can drop in Slack).

---

## 📖 The Complete Command Playbook

Here is every command you can run, grouped by what you are trying to achieve.

### 🔎 Search & Execute
| Command | What it does | Example |
|---|---|---|
| `recall "intent"` | Searches for a command based on plain English description. | `recall "kill port 8080"` |
| `recall compose` | Interactive step-by-step TUI to build complex commands. | `recall compose` |

### 🧠 Understanding & Learning
| Command | What it does | Example |
|---|---|---|
| `recall explain "cmd"` | Breaks down a command, explains flags, shows danger warnings. | `recall explain "git reset --hard"` |
| `recall roast` | Let the CLI playfully roast you for your terrible command line habits. | `recall roast` |
| `recall share "cmd"` | Generates a beautiful SVG of a command to share on Twitter/Slack. | `recall share "npx create-react-app"` |

### 📊 Analytics & Insights
| Command | What it does | Example |
|---|---|---|
| `recall vault` | Opens an interactive dashboard to browse all your saved commands. | `recall vault` |
| `recall stats` | High-level statistics on your terminal usage. | `recall stats` |
| `recall digest` | Your weekly terminal summary (streak, commands, active days). | `recall digest` |
| `recall wrapped` | Like Spotify Wrapped, but for your terminal. Your yearly review. | `recall wrapped` |
| `recall suggest-aliases`| Analyzes your patterns and proposes shell aliases. | `recall suggest-aliases` |

### 🔐 Vault Management & Backups
| Command | What it does | Example |
|---|---|---|
| `recall export` | Exports your entire vault into an AES-encrypted backup file. | `recall export --output backup.enc` |
| `recall import` | Imports a vault backup (merges or overwrites). | `recall import --input backup.enc` |
| `recall sync` | Pushes/Pulls your encrypted vault to a private GitHub Gist for cloud backup. | `recall sync --push` |
| `recall maintenance` | Cleans the database, vacuums, and rebuilds the FTS search index. | `recall maintenance` |

### ⚙️ Core System
| Command | What it does | Example |
|---|---|---|
| `recall init` | The first utility you run. Sets up hooks. | `recall init` |
| `recall import-history`| Bootstraps memory from your existing shell history. | `recall import-history` |
| `recall config` | Edit capture filters, exclusions, and encryption settings. | `recall config` |
| `recall plugin list` | View installed 3rd-party WASM plugins. | `recall plugin list` |

---

**Remember:** With Recall running, **you don't have to consciously do anything to save commands.** Just use your terminal normally. When you forget something, Recall will be there.
