package compose

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Om-Rohilla/recall/internal/capture"
	"github.com/Om-Rohilla/recall/internal/vault"
)

// SaveComposedCommand persists a composed command to the vault.
func SaveComposedCommand(store *vault.Store, raw string) error {
	parsed := capture.Parse(raw)
	flagsJSON, err := json.Marshal(parsed.Flags)
	if err != nil {
		return fmt.Errorf("marshaling flags: %w", err)
	}
	now := time.Now().UTC()

	_, err = store.InsertCommand(&vault.Command{
		Raw:        parsed.Raw,
		Binary:     parsed.Binary,
		Subcommand: parsed.Subcommand,
		Flags:      string(flagsJSON),
		Category:   parsed.Category,
		Frequency:  1,
		FirstSeen:  now,
		LastSeen:   now,
	})
	if err != nil {
		return fmt.Errorf("saving composed command: %w", err)
	}
	return nil
}
