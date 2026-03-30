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

bindkey '^ ' __recall_search_widget   # Ctrl+Space
bindkey '^K' __recall_vault_widget     # Ctrl+K
bindkey '^E' __recall_explain_widget   # Ctrl+E
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

bind -x '"\C- ": __recall_search_binding'   # Ctrl+Space
bind -x '"\C-k": __recall_vault_binding'    # Ctrl+K
bind -x '"\C-x\C-e": __recall_explain_binding'  # Ctrl+X Ctrl+E
`
}

func fishHook() string {
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
bind \c@ __recall_search
# Ctrl+K — vault browser
bind \ck __recall_vault
# Ctrl+E — explain current command
bind \ce __recall_explain
`
}

// SourceLine returns the line to add to the shell config file.
func SourceLine(sh Shell) string {
	return fmt.Sprintf("%s\neval \"$(recall hook %s)\"\n", hookMarker, sh)
}
