package shell

import (
	"fmt"
	"strings"
)

const hookMarker = "# Added by Recall — do not edit"

// HookKeys holds the resolved shell keybinding sequences for a given shell.
type HookKeys struct {
	Search  string // key sequence for inline search
	Explain string // key sequence for explain
	Vault   string // key sequence for vault browser
}

// DefaultHookKeys returns the safe default bindings that avoid readline conflicts.
// Alt+R for explain (was Ctrl+E — moves cursor to end-of-line in readline/bash/zsh).
// Alt+K for vault   (was Ctrl+K — kills to end-of-line in readline).
func DefaultHookKeys(sh Shell) HookKeys {
	switch sh {
	case Zsh:
		return HookKeys{
			Search:  "^ ",   // Ctrl+Space
			Explain: "\\er", // Alt+R
			Vault:   "\\ek", // Alt+K
		}
	case Bash:
		return HookKeys{
			Search:  "\\C- ",  // Ctrl+Space
			Explain: "\\M-r",  // Alt+R
			Vault:   "\\M-k",  // Alt+K
		}
	case Fish:
		return HookKeys{
			Search:  "\\c@",   // Ctrl+Space (NUL)
			Explain: "\\er",   // Alt+R
			Vault:   "\\ek",   // Alt+K
		}
	}
	return HookKeys{Search: "ctrl+space", Explain: "alt+r", Vault: "alt+k"}
}

// hotkeyConfigToShellSeq converts a human-readable hotkey string (e.g. "alt+r")
// into the appropriate shell escape sequence for the given shell.
func hotkeyConfigToShellSeq(sh Shell, hk string) string {
	hk = strings.ToLower(strings.TrimSpace(hk))
	switch sh {
	case Zsh:
		mapping := map[string]string{
			"ctrl+space": "^ ",
			"alt+r":      "\\er",
			"alt+k":      "\\ek",
			"ctrl+e":     "^E",
			"ctrl+k":     "^K",
		}
		if v, ok := mapping[hk]; ok {
			return v
		}
	case Bash:
		mapping := map[string]string{
			"ctrl+space": "\\C- ",
			"alt+r":      "\\M-r",
			"alt+k":      "\\M-k",
			"ctrl+e":     "\\C-e",
			"ctrl+k":     "\\C-k",
		}
		if v, ok := mapping[hk]; ok {
			return v
		}
	case Fish:
		mapping := map[string]string{
			"ctrl+space": "\\c@",
			"alt+r":      "\\er",
			"alt+k":      "\\ek",
			"ctrl+e":     "\\ce",
			"ctrl+k":     "\\ck",
		}
		if v, ok := mapping[hk]; ok {
			return v
		}
	}
	return hk // fallback: pass through as-is
}

// HookScript returns the shell hook script content for the given shell using default bindings.
func HookScript(sh Shell) (string, error) {
	keys := DefaultHookKeys(sh)
	return HookScriptWithKeys(sh, keys)
}

// HookScriptWithKeys returns the shell hook script content with the given hotkey bindings.
func HookScriptWithKeys(sh Shell, keys HookKeys) (string, error) {
	switch sh {
	case Zsh:
		return zshHook(keys), nil
	case Bash:
		return bashHook(keys), nil
	case Fish:
		return fishHook(keys), nil
	default:
		return "", fmt.Errorf("unsupported shell for hook: %s", sh)
	}
}

// HotkeySeqsFromConfig converts config hotkey strings to shell sequences.
func HotkeySeqsFromConfig(sh Shell, search, explain, vault string) HookKeys {
	defaults := DefaultHookKeys(sh)
	keys := HookKeys{
		Search:  defaults.Search,
		Explain: defaults.Explain,
		Vault:   defaults.Vault,
	}
	if search != "" {
		keys.Search = hotkeyConfigToShellSeq(sh, search)
	}
	if explain != "" {
		keys.Explain = hotkeyConfigToShellSeq(sh, explain)
	}
	if vault != "" {
		keys.Vault = hotkeyConfigToShellSeq(sh, vault)
	}
	return keys
}

func zshHook(keys HookKeys) string {
	return `# Recall shell hook for Zsh
# Captures commands after execution with minimal overhead

__recall_preexec() {
    __recall_cmd="$1"
    __recall_start=$EPOCHREALTIME
}

__recall_precmd() {
    local exit_code=$?
    [ -z "$__recall_cmd" ] && return

    local end=$EPOCHREALTIME
    local duration_ms=0
    if [ -n "$__recall_start" ] && [ -n "$end" ]; then
        duration_ms=$(printf '%.0f' $(( ($end - $__recall_start) * 1000 )))
    fi

    recall capture \
        --command "$__recall_cmd" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration "$duration_ms" \
        --session "$RECALL_SESSION_ID" \
        2>/dev/null &!

    __recall_cmd=""
    __recall_start=""
}

if [ -z "$RECALL_SESSION_ID" ]; then
    export RECALL_SESSION_ID="sess_$(date +%s)_$$"
fi

autoload -Uz add-zsh-hook
add-zsh-hook preexec __recall_preexec
add-zsh-hook precmd __recall_precmd

# Recall hotkey bindings
__recall_search_widget() {
    local selected
    selected=$(recall search-tui 2>/dev/null)
    if [ -n "$selected" ]; then
        LBUFFER="$selected"
    fi
    zle reset-prompt
}
zle -N __recall_search_widget

__recall_vault_widget() {
    recall vault </dev/tty
    zle reset-prompt
}
zle -N __recall_vault_widget

__recall_explain_widget() {
    if [ -n "$BUFFER" ]; then
        recall explain "$BUFFER" </dev/tty
        zle reset-prompt
    fi
}
zle -N __recall_explain_widget

bindkey '` + keys.Search + `' __recall_search_widget   # search
bindkey '` + keys.Vault + `' __recall_vault_widget     # vault (Alt+K — avoids Ctrl+K kill-line conflict)
bindkey '` + keys.Explain + `' __recall_explain_widget # explain (Alt+R — avoids Ctrl+E end-of-line conflict)
`
}

func bashHook(keys HookKeys) string {
	return `# Recall shell hook for Bash
# Captures commands after execution with minimal overhead

__recall_preexec() {
    [ -n "$COMP_LINE" ] && return
    __recall_cmd="$(HISTTIMEFORMAT= history 1 | sed 's/^ *[0-9]* *//')"
    __recall_start=$EPOCHREALTIME
    if [ -z "$__recall_start" ]; then
        __recall_start=$(date +%s%N | cut -b1-13)
    fi
}

__recall_precmd() {
    local exit_code=$?
    [ -z "$__recall_cmd" ] && return

    local end=$EPOCHREALTIME
    local duration_ms=0
    if [ -n "$end" ] && [ -n "$__recall_start" ]; then
        duration_ms=$(printf '%.0f' $(echo "($end - $__recall_start) * 1000" | bc 2>/dev/null || echo 0))
    fi

    recall capture \
        --command "$__recall_cmd" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration "$duration_ms" \
        --session "$RECALL_SESSION_ID" \
        2>/dev/null &

    __recall_cmd=""
    __recall_start=""
}

if [ -z "$RECALL_SESSION_ID" ]; then
    export RECALL_SESSION_ID="sess_$(date +%s)_$$"
fi

trap '__recall_preexec' DEBUG
PROMPT_COMMAND="__recall_precmd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"

# Recall hotkey bindings
__recall_search_binding() {
    local selected
    selected=$(recall search-tui 2>/dev/null)
    if [ -n "$selected" ]; then
        READLINE_LINE="$selected"
        READLINE_POINT=${#READLINE_LINE}
    fi
}

__recall_vault_binding() {
    recall vault </dev/tty
}

__recall_explain_binding() {
    if [ -n "$READLINE_LINE" ]; then
        recall explain "$READLINE_LINE" </dev/tty
    fi
}

bind -x '"` + keys.Search + `": __recall_search_binding'  # search
bind -x '"` + keys.Vault + `": __recall_vault_binding'    # vault (Alt+K — avoids Ctrl+K kill-line conflict)
bind -x '"` + keys.Explain + `": __recall_explain_binding' # explain (Alt+R — avoids Ctrl+E end-of-line conflict)
`
}

func fishHook(keys HookKeys) string {
	return `# Recall shell hook for Fish
# Captures commands after execution with minimal overhead

if not set -q RECALL_SESSION_ID
    set -gx RECALL_SESSION_ID "sess_"(date +%s)"_"(fish_pid)
end

function __recall_postexec --on-event fish_postexec
    set -l exit_code $status
    set -l cmd $argv[1]
    set -l cwd $PWD

    # Fish provides CMD_DURATION in milliseconds
    set -l duration_ms 0
    if set -q CMD_DURATION
        set duration_ms $CMD_DURATION
    end

    command recall capture \
        --command "$cmd" \
        --exit-code "$exit_code" \
        --cwd "$cwd" \
        --duration "$duration_ms" \
        --session "$RECALL_SESSION_ID" \
        2>/dev/null &
    disown 2>/dev/null
end

# Recall hotkey bindings
function __recall_search
    set -l selected (command recall search-tui 2>/dev/null)
    if test -n "$selected"
        commandline -r -- $selected
    end
    commandline -f repaint
end

function __recall_vault
    command recall vault </dev/tty
    commandline -f repaint
end

function __recall_explain
    set -l cmd (commandline -b)
    if test -n "$cmd"
        command recall explain "$cmd" </dev/tty
        commandline -f repaint
    end
end

# Ctrl+Space — inline search
bind ` + keys.Search + ` __recall_search
# Alt+K — vault browser (avoids Ctrl+K which is a word-kill in fish)
bind ` + keys.Vault + ` __recall_vault
# Alt+R — explain current command (avoids Ctrl+E which moves cursor to EOL)
bind ` + keys.Explain + ` __recall_explain
`
}

// SourceLine returns the line to add to the shell config file.
func SourceLine(sh Shell) string {
	return fmt.Sprintf("%s\neval \"$(recall hook %s)\"\n", hookMarker, sh)
}
