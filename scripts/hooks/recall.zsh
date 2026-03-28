#!/usr/bin/env zsh
# Recall shell hook for Zsh
# Captures commands after execution with minimal overhead
# This file is sourced via: eval "$(recall hook zsh)"

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
