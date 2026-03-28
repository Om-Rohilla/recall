package shell

import "fmt"

const hookMarker = "# Added by Recall — do not edit"

// HookScript returns the shell hook script content for the given shell.
func HookScript(sh Shell) (string, error) {
	switch sh {
	case Zsh:
		return zshHook(), nil
	case Bash:
		return bashHook(), nil
	case Fish:
		return fishHook(), nil
	default:
		return "", fmt.Errorf("unsupported shell for hook: %s", sh)
	}
}

func zshHook() string {
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
`
}

func bashHook() string {
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
`
}

func fishHook() string {
	return `# Recall shell hook for Fish
# Phase 5 — not yet fully implemented

function __recall_postexec --on-event fish_postexec
    set -l cmd $argv[1]
    set -l exit_code $status

    if not set -q RECALL_SESSION_ID
        set -gx RECALL_SESSION_ID "sess_"(date +%s)"_"(echo %self)
    end

    recall capture \
        --command "$cmd" \
        --exit-code "$exit_code" \
        --cwd "$PWD" \
        --duration "0" \
        --session "$RECALL_SESSION_ID" \
        2>/dev/null &
end
`
}

// SourceLine returns the line to add to the shell config file.
func SourceLine(sh Shell) string {
	return fmt.Sprintf("%s\neval \"$(recall hook %s)\"\n", hookMarker, sh)
}
