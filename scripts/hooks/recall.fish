#!/usr/bin/env fish
# Recall shell hook for Fish
# Captures commands after execution with minimal overhead
# This file is sourced via: recall hook fish | source

if not set -q RECALL_SESSION_ID
    set -gx RECALL_SESSION_ID "sess_"(date +%s)"_"(fish_pid)
end

function __recall_postexec --on-event fish_postexec
    set -l cmd $argv[1]
    set -l exit_code $status
    set -l cwd $PWD

    # Measure duration using fish's built-in CMD_DURATION (milliseconds)
    set -l duration_ms 0
    if set -q CMD_DURATION
        set duration_ms $CMD_DURATION
    end

    # Capture in background to avoid blocking the shell
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
