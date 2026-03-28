#!/usr/bin/env bash
# Recall shell hook for Bash
# Captures commands after execution with minimal overhead
# This file is sourced via: eval "$(recall hook bash)"

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
