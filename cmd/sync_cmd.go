package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "E2E Cloud Sync (Ghost Mode) via GitHub Gists",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	token := os.Getenv("RECALL_GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("RECALL_GITHUB_TOKEN environment variable is required for Ghost Sync")
	}

	cfg := config.Get()
	
	// We'll sync the pending commands asynchronously
	queueFile := filepath.Join(filepath.Dir(cfg.Vault.Path), "pending.ndjson")
	if _, err := os.Stat(queueFile); os.IsNotExist(err) {
		fmt.Println(ui.DimStyle.Render("Nothing to sync!"))
		return nil
	}

	data, err := os.ReadFile(queueFile)
	if err != nil {
		return fmt.Errorf("reading local queue: %w", err)
	}

	key, err := vault.GetOrGenerateVaultKey()
	if err != nil {
		return fmt.Errorf("getting vault key for sync crypto: %w", err)
	}
	encData, err := vault.Encrypt(data, key)
	if err != nil {
		return fmt.Errorf("encrypting sync data: %w", err)
	}
	payloadStr := base64.StdEncoding.EncodeToString(encData)

	gistID := os.Getenv("RECALL_GIST_ID")
	if gistID == "" {
		// Create new gist
		fmt.Println(ui.DimStyle.Render("Creating new secure E2E Encrypted Ghost Sync Gist..."))
		newID, err := createGist(token, payloadStr)
		if err != nil {
			return err
		}
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Ghost Sync initialized! Store this and set it as RECALL_GIST_ID: %s", newID)))
		return nil
	}

	// Update existing gist to push our local pending edits
	fmt.Println(ui.DimStyle.Render(fmt.Sprintf("Syncing to Ghost Gist (ID: %s)...", gistID)))
	if err := updateGist(token, gistID, payloadStr); err != nil {
		return err
	}

	fmt.Println(ui.SuccessStyle.Render("✓ E2E Sync Complete! Cross-device state updated."))
	return nil
}

type gistFile struct {
	Content string `json:"content"`
}

type gistPayload struct {
	Description string              `json:"description"`
	Public      bool                `json:"public"`
	Files       map[string]gistFile `json:"files"`
}

func createGist(token, content string) (string, error) {
	payload := gistPayload{
		Description: "Recall CLI - E2E Encrypted Sync Vault",
		Public:      false,
		Files: map[string]gistFile{
			"pending.ndjson.enc": {Content: content},
		},
	}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.github.com/gists", bytes.NewReader(b))
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create gist: %s", string(body))
	}

	var res struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	return res.ID, nil
}

func updateGist(token, id, content string) error {
	payload := gistPayload{
		Description: "Recall CLI - E2E Encrypted Sync Vault",
		Files: map[string]gistFile{
			"pending.ndjson.enc": {Content: content},
		},
	}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PATCH", "https://api.github.com/gists/"+id, bytes.NewReader(b))
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update gist: %s", string(body))
	}

	return nil
}
