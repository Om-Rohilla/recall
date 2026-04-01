# Marketing, Positioning & Launch Strategy

## Brand Identity

### Name: **Recall**

- Short, memorable, one word
- Directly describes the action (recalling a command)
- Works as a verb: "Just recall it"
- Easy to type: `recall "query"`
- Available namespace: unlikely to conflict with existing system commands

### Tagline Options

**Primary:**
> "Your terminal has perfect memory. You just forgot to use it."

**Alternatives:**
> "Stop Googling commands. Start recalling them."

> "The command you forgot is already in your vault."

> "Zero-latency, zero-cloud command intelligence."

> "Every command you've ever run, one keystroke away."

### Logo Concept

A minimalist terminal cursor (`▌`) combined with a brain/memory icon. Monochrome — works in any terminal, any README, any slide deck.

---

## Positioning Statement

**For** developers who waste time looking up terminal commands,
**Recall is** a local-first command intelligence tool
**that** surfaces the right command instantly from your personal history and a curated knowledge base,
**unlike** tldr, fzf, or AI-powered shells,
**our product** is 100% offline, learns your patterns, and responds in under 10 milliseconds.

---

## Target Audiences (Ranked by Priority)

### Tier 1 — Core Audience (Early Adopters)

**Linux power users & DevOps engineers**
- Use the terminal 4-8 hours/day
- Manage complex infrastructure (Docker, Kubernetes, cloud CLI)
- Privacy-conscious, prefer open source
- Already use fzf, tldr, or custom aliases
- Pain: constantly forgetting complex flag combinations

**Where they hang out:**
- Hacker News, r/linux, r/commandline, r/devops
- Lobste.rs, dev.to
- Twitter/X (DevOps community)
- Discord (various Linux/DevOps servers)

---

### Tier 2 — Growth Audience

**Backend developers & full-stack engineers**
- Use terminal daily but not exclusively
- Juggle git, docker, npm/pip, SSH, and system commands
- Pain: context switching to Google for syntax

**Where they hang out:**
- r/programming, r/webdev, r/golang
- Dev.to, Hashnode
- YouTube (programming channels)
- Tech Twitter

---

### Tier 3 — Expansion Audience

**Students & junior developers**
- Learning Linux, overwhelmed by syntax
- Can't afford $15/mo for Warp or Copilot
- Need a tool that teaches them while they work
- Pain: every command requires a Google search

**Where they hang out:**
- r/learnprogramming, r/cs50, r/linux4noobs
- University Discord servers, study groups
- YouTube tutorials, TikTok tech content

---

### Tier 4 — Enterprise Audience (Future)

**Security-conscious teams & air-gapped environments**
- Cannot use cloud AI tools (compliance, security)
- Need developer productivity tooling that stays on-prem
- Military, government, finance, healthcare
- Pain: developers are slower because AI tools are banned

---

## Key Differentiators (Messaging Pillars)

Every piece of marketing should emphasize these 5 pillars:

### 1. "100% Offline — Your Commands Never Leave Your Machine"

This is the #1 differentiator. In a world where every "AI tool" phones home, Recall is radically private. No account, no API key, no telemetry, no network calls.

**Use when:** competing with Warp, Copilot CLI, any cloud tool.

### 2. "It Learns You — Not the Other Way Around"

Recall adapts to YOUR patterns. The more you use it, the smarter it gets — YOUR most-used commands rank first, YOUR projects get context boost, YOUR aliases get suggested.

**Use when:** competing with tldr, cheat.sh, man pages (static, impersonal).

### 3. "<10ms — Faster Than Your Thought"

Sub-10ms response time. No loading spinner. No "thinking..." animation. It feels like autocomplete, not a search engine.

**Use when:** competing with AI tools (500-2000ms latency).

### 4. "Free Forever — No Subscription, No Account"

Open source, MIT licensed, zero cost. No freemium trap. No "upgrade to Pro for intent search." Everything is included.

**Use when:** competing with Warp ($15/mo), Copilot ($10/mo).

### 5. "Works on Day 1 — Gets Better Every Day"

Zero-config. Import history, install hook, done. Useful immediately via knowledge base. Gets dramatically better as your vault grows.

**Use when:** competing with Navi (requires manual YAML curation).

---

## Launch Strategy

### Pre-Launch (2 weeks before)

**Goals:** Build anticipation, collect email signups, seed early testers.

1. **Teaser post on Twitter/X:**
   "Building something. 12-25x per day, developers leave their terminal to Google a command. What if you never had to? Stay tuned."

2. **Personal blog post:**
   "The Command Amnesia Problem — Why Developers Waste 250 Hours/Year on Syntax Lookup"
   Pure problem statement. No product mention until the end. Builds credibility.

3. **Early access signup page:**
   Simple landing page: problem statement + email input.
   Target: 200-500 signups before launch.

4. **Private beta with 10-20 developers:**
   Hand-pick power users from DevOps/Linux communities.
   Get feedback, fix bugs, collect testimonials.

---

### Launch Day

**Goals:** Maximum visibility on Hacker News + Reddit. Drive GitHub stars.

#### Hacker News Post

**Title:** "Show HN: Recall — 100% offline command intelligence for your terminal"

**Post structure:**
- One-paragraph problem statement (command amnesia, 60-120 min/day lost)
- What Recall does (intent search, vault, explain, aliases)
- Key differentiators (offline, <10ms, learns your patterns, free)
- Demo GIF (3-5 second loop showing `recall "find big files"`)
- GitHub link

**Timing:** Tuesday or Wednesday, 9-10 AM ET (peak HN traffic).

#### Reddit Posts

**Subreddits (staggered over 48 hours):**

| Subreddit | Post Angle |
|-----------|-----------|
| r/commandline | "I built a tool that gives your terminal perfect memory" |
| r/linux | "100% offline alternative to AI-powered shells — no cloud, no subscription" |
| r/devops | "Stop Googling kubectl flags — Recall learns your infrastructure commands" |
| r/golang | "Built a CLI tool in Go with Bubbletea + SQLite FTS5 — here's what I learned" |
| r/programming | "The Command Amnesia Problem: Why developers waste 250 hours/year on syntax" |

**Rules:**
- Lead with the problem, not the product
- Include the demo GIF in every post
- Be in the comments responding to questions for the first 6 hours
- Never be defensive. Acknowledge limitations honestly.

#### Dev.to / Hashnode Article

**Title:** "I Built a Tool That Eliminates Command Amnesia — Here's How"

Long-form article (2000-3000 words):
1. The problem (with data)
2. Why existing tools fail (analysis)
3. What I built (with screenshots)
4. How it works technically (architecture overview)
5. What I learned building it
6. Try it yourself (link)

---

### Post-Launch (weeks 1-4)

**Goals:** Sustain momentum, grow community, iterate on feedback.

1. **Respond to every GitHub issue within 24 hours**
2. **Weekly changelog posts** — show progress, new commands in KB
3. **"Recall tip of the day"** — tweet one useful `recall` query daily
4. **Community knowledge base contributions** — make it easy for users to add commands
5. **YouTube demo video** (3-5 minutes) — show real workflow, not marketing
6. **Collect and publish testimonials**

---

## Content Strategy (Ongoing)

### Blog Post Ideas

| Title | Angle |
|-------|-------|
| "The 50 Most-Googled Terminal Commands (And How to Stop Googling Them)" | SEO play, link to Recall |
| "Why I Left Warp for an Offline Tool" | Comparison piece, privacy angle |
| "How TF-IDF Powers a Sub-10ms Command Search" | Technical deep-dive, attracts HN |
| "My Terminal Setup in 2026: Shell + Recall + tmux" | Workflow article, developer blog |
| "Building a CLI in Go with Bubbletea: Lessons Learned" | Go community engagement |
| "Privacy-First Developer Tools: Why Offline Matters" | Thought leadership |
| "From 50,000 History Lines to Instant Recall: How I Tamed My Shell" | Personal story |

### Social Media Templates

**Twitter/X (problem tweet):**
```
Developers lose 250+ hours/year to this cycle:

1. Forget a command
2. Open browser
3. Google it
4. Stack Overflow
5. Copy/paste

12-25x per day. Every day.

What if your terminal just... remembered?
```

**Twitter/X (feature tweet):**
```
$ recall "kill process on port 3000"

→ lsof -ti:3000 | xargs kill

< 10ms. No internet. No API key. No subscription.

Just your history + a curated knowledge base.
```

**Twitter/X (comparison tweet):**
```
Warp AI: 800ms, cloud, $15/mo, sends commands to servers
Copilot CLI: 1200ms, cloud, $10/mo, hallucinates flags
tldr: offline, but you need to already know the command name

Recall: 8ms, offline, free, learns YOUR patterns

Not the same category.
```

---

## Demo Strategy

### GIF Demos (For README, tweets, posts)

**Demo 1: Intent Search**
```
$ recall "find large files over 100mb"
→ Shows result card with confidence score
→ User presses Enter, command executes
```
Duration: 4 seconds. Loop.

**Demo 2: Explain Mode**
```
$ recall explain "find . -name '*.log' -mtime +30 -delete"
→ Flag-by-flag breakdown appears
→ Destructive warning highlighted in red
```
Duration: 5 seconds.

**Demo 3: Vault Browser**
```
$ recall vault
→ TUI opens, user browses by category
→ Filters to "docker", selects a command
```
Duration: 6 seconds.

**Demo 4: Alias Suggestion**
```
$ recall suggest-aliases
→ Shows repetitive commands
→ User presses 'a' to accept all
→ Aliases written to .zshrc
```
Duration: 5 seconds.

### Video Demo (YouTube)

**Length:** 3-5 minutes.
**Structure:**
1. (0:00) The problem — show the Google workflow in real time
2. (0:30) Install Recall — one command
3. (0:45) Import history — instant vault
4. (1:00) Intent search — 3-4 examples
5. (2:00) Explain mode — break down a complex command
6. (2:30) Vault browser — show the TUI
7. (3:00) Stats + alias suggestions
8. (3:30) "No internet was used in the making of this video"
9. (4:00) Call to action — GitHub link, star the repo

---

## Metrics to Track

### GitHub Metrics
| Metric | Launch Target (Week 1) | Growth Target (Month 3) |
|--------|----------------------|------------------------|
| Stars | 500+ | 3,000+ |
| Forks | 30+ | 200+ |
| Contributors | 5+ | 20+ |
| Issues (active) | 20+ | 50+ |
| KB contributions | 10+ commands | 200+ commands |

### Usage Metrics (anonymous, opt-in only)
| Metric | Target |
|--------|--------|
| Daily active queries per user | 10+ |
| First-result accuracy | >85% |
| Google replacement rate | >90% after 1 month |
| Vault size after 30 days | 5,000+ commands |
| Alias adoption rate | >50% of suggestions accepted |

### Content Metrics
| Metric | Target |
|--------|--------|
| Hacker News front page | Yes |
| Blog post views | 10,000+ total in month 1 |
| Twitter impressions | 50,000+ in launch week |
| YouTube demo views | 5,000+ in month 1 |

---

## Pitch Versions

### One-Liner
> "100% offline command intelligence — search your terminal by intent, not exact syntax."

### Elevator Pitch (30 seconds)
> "Developers waste 60-120 minutes every day Googling terminal commands. Recall fixes this — it's an offline CLI tool that searches your command history by intent, not exact text. Type 'find big files' and get the exact command you used last week. It learns your patterns, works in under 10 milliseconds, and nothing ever leaves your machine. Think of it as a personal search engine for your terminal."

### Portfolio Pitch (For interviews)
> "I built Recall, a command intelligence engine that solves 'Command Amnesia' — the daily problem of forgetting terminal syntax. Unlike AI-powered tools that require internet and send your data to servers, Recall is 100% offline. It uses SQLite FTS5 for full-text search, a multi-signal scoring algorithm (TF-IDF, intent matching, frequency weighting, context detection), and shell hooks for invisible capture. The architecture has four layers: capture, intelligence, surface, and vault. I built it in Go using Cobra for the CLI, Bubbletea for the TUI, and SQLite for storage. It responds in under 10ms and learns the user's personal command patterns over time."

### Investor Pitch (Future, if relevant)
> "250 million developers lose 250+ hours per year to command syntax lookup. Existing solutions are either dumb (tldr, fzf) or cloud-dependent (Warp, Copilot). Recall is the first offline, privacy-first command intelligence tool. It sits in an empty competitive quadrant — high personalization, fully offline. The flywheel effect (gets smarter with use) creates natural lock-in without vendor lock-in. We're open source with a path to enterprise (air-gapped environments, compliance-heavy industries where cloud AI tools are banned). Target market: 250M developers, starting with 50M Linux power users."

---

## Competitive Response Guide

**"How is this different from tldr?"**
> tldr requires you to already know the command name. If you know you need `find`, tldr helps. But if you're thinking "how do I find big files" — tldr can't help. Recall understands intent. Also, tldr gives the same answer to everyone. Recall learns YOUR patterns.

**"Why not just use ChatGPT / Copilot?"**
> Three reasons: (1) Privacy — your commands never leave your machine. (2) Speed — 10ms vs 1-2 seconds. (3) Reliability — no hallucinated flags, no API downtime. For environments where cloud AI is banned (government, finance, military), Recall is the only option.

**"Can't fzf + Ctrl+R do this?"**
> fzf does string matching. It can't find "that docker cleanup command" if you search for "container cleanup." It doesn't understand intent, doesn't rank by relevance, doesn't know your project context, and doesn't explain commands. Recall builds on what fzf does well (fast fuzzy matching) and adds intelligence on top.

**"Why not use Warp?"**
> Warp is a full terminal replacement that costs $15/mo and sends your data to their servers. Recall is a lightweight tool that works inside your existing terminal (any terminal), is free forever, and is 100% offline. Different philosophy: Recall augments your workflow; Warp replaces it.

**"This seems like a niche tool."**
> Every developer who uses a terminal is a potential user. That's 250M+ people worldwide. "Googling terminal commands" is the most universal developer behavior that hasn't been properly solved with an offline tool. It's not niche — it's invisible. That's why no one has built it.

---

## Timeline

| Week | Activity |
|------|----------|
| Week -2 | Teaser tweets, personal blog post about the problem |
| Week -1 | Private beta (10-20 users), collect feedback + testimonials |
| **Week 0** | **LAUNCH: HN, Reddit, Dev.to, YouTube demo** |
| Week 1 | Respond to feedback, fix bugs, engage comments |
| Week 2 | Follow-up blog post: "Week 1 Results + What We Learned" |
| Week 3 | Technical deep-dive post (HN bait), community KB drive |
| Week 4 | v0.2 release with top-requested features |
| Month 2 | Conference lightning talks, podcast appearances |
| Month 3 | v0.3 release, Homebrew/AUR packages, enterprise outreach |
