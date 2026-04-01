package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"github.com/Om-Rohilla/recall/pkg/logging"
)

// HookPayload is the data sent to a plugin for a specific hook.
type HookPayload struct {
	HookName string      `json:"hook_name"`
	Data     interface{} `json:"data"` // specific to the hook
}

// HookResult is the data returned by a plugin.
type HookResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	// Additional data can be added here if plugins are allowed to mutate state
}

// ExecuteHook sends the payload to all plugins registered for the hook.
// It executes them sequentially. Timeout per plugin is 5s.
func (r *Registry) ExecuteHook(hookName string, data interface{}) {
	plugins, err := r.List()
	if err != nil {
		logging.Get().Warn("failed to list plugins for hook execution", "error", err)
		return
	}

	payload := HookPayload{
		HookName: hookName,
		Data:     data,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logging.Get().Error("failed to marshal hook payload", "error", err)
		return
	}

	for _, p := range plugins {
		// Check if plugin supports this hook
		supports := false
		for _, h := range p.Hooks {
			if h == hookName {
				supports = true
				break
			}
		}
		if !supports {
			continue
		}

		// Execute
		r.executeSingle(p, hookName, payloadBytes)
	}
}

func (r *Registry) executeSingle(p Plugin, hookName string, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.Path)
	cmd.Stdin = bytes.NewReader(payload)
	
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	log := logging.Get().With("plugin", p.Name, "hook", hookName, "duration_ms", duration.Milliseconds())

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Warn("plugin execution timed out")
		} else {
			log.Warn("plugin execution failed", "error", err, "stderr", stderr.String())
		}
		return
	}

	// Try to parse the result (optional — plugins may just perform side effects)
	var result HookResult
	if stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			log.Debug("failed to parse plugin output", "error", err, "stdout", stdout.String())
			return
		}
		if !result.Success && result.Error != "" {
			log.Warn("plugin reported error", "err_msg", result.Error)
		} else {
			log.Debug("plugin executed successfully")
		}
	} else {
		log.Debug("plugin executed with no output")
	}
}
