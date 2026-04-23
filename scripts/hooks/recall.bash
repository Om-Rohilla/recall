#!/usr/bin/env bash
# Recall shell hook for Bash
# Captures commands after execution with minimal overhead
# This file is sourced via: eval "$(recall hook bash)"
#
# NOTE: Commands excluded by HISTCONTROL may not be captured since Recall
# reads from 'history 1'. Specifically:
#   - HISTCONTROL=ignorespace: commands with a leading space are skipped
#   - HISTCONTROL=ignoredups:  exact duplicate commands are skipped
# To capture all commands, add to your .bashrc:
#   export HISTCONTROL=
# Recall's own secret filter (see 'recall config show') prevents sensitive
# commands from being stored regardless of HISTCONTROL.

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
bind -x '"\C-e": __recall_explain_binding'  # Ctrl+E
