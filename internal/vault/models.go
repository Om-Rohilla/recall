package vault

import "time"

type Command struct {
	ID          int64     `json:"id"`
	Raw         string    `json:"raw"`
	Binary      string    `json:"binary"`
	Subcommand  string    `json:"subcommand,omitempty"`
	Flags       string    `json:"flags,omitempty"` // JSON array
	Category    string    `json:"category,omitempty"`
	Frequency   int       `json:"frequency"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	LastExit    *int      `json:"last_exit,omitempty"`
	AvgDuration *float64  `json:"avg_duration,omitempty"`
}

type Context struct {
	ID          int64     `json:"id"`
	CommandID   int64     `json:"command_id"`
	Cwd         string    `json:"cwd,omitempty"`
	GitRepo     string    `json:"git_repo,omitempty"`
	GitBranch   string    `json:"git_branch,omitempty"`
	ProjectType string    `json:"project_type,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	ExitCode    *int      `json:"exit_code,omitempty"`
	DurationMs  *int      `json:"duration_ms,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
}

type Knowledge struct {
	ID          int64  `json:"id"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Intents     string `json:"intents"`      // JSON array
	Category    string `json:"category"`
	FlagsDoc    string `json:"flags_doc,omitempty"`    // JSON object
	Examples    string `json:"examples,omitempty"`     // JSON array
	DangerLevel string `json:"danger_level,omitempty"` // safe, caution, destructive
}

type Pattern struct {
	ID             int64  `json:"id"`
	Template       string `json:"template"`
	Frequency      int    `json:"frequency"`
	SuggestedAlias string `json:"suggested_alias,omitempty"`
}

// AliasSuggestion tracks a suggested alias and its adoption status.
type AliasSuggestion struct {
	ID            int64  `json:"id"`
	Command       string `json:"command"`
	Alias         string `json:"alias"`
	SuggestedAt   string `json:"suggested_at"`
	AdoptedAt     string `json:"adopted_at,omitempty"`
	AdoptionCount int    `json:"adoption_count"`
}

type SearchResult struct {
	Command    Command `json:"command"`
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"` // 0-100 percentage
	MatchType  string  `json:"match_type"` // "vault" or "knowledge"
}

type VaultStats struct {
	TotalCommands  int `json:"total_commands"`
	UniqueCommands int `json:"unique_commands"`
	TotalContexts  int `json:"total_contexts"`
}

type CategoryCount struct {
	Category       string `json:"category"`
	Count          int    `json:"count"`
	TotalFrequency int    `json:"total_frequency"`
}

// CaptureData is the raw data received from a shell hook.
type CaptureData struct {
	RawCommand  string
	ExitCode    int
	Cwd         string
	Timestamp   time.Time
	DurationMs  int
	GitRepo     string
	GitBranch   string
	ProjectType string
	SessionID   string
}

// ExportData is the JSON-serializable vault export payload.
type ExportData struct {
	Version      int       `json:"version"`
	ExportedAt   time.Time `json:"exported_at"`
	Commands     []Command `json:"commands"`
	Contexts     []Context `json:"contexts,omitempty"`
	Patterns     []Pattern `json:"patterns,omitempty"`
	CommandCount int       `json:"command_count"`
	ContextCount int       `json:"context_count"`
}
