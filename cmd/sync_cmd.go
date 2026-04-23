package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

// syncHTTPClient is a shared HTTP client with strict timeouts.
// Using http.DefaultClient is prohibited — it has no timeout and can hang forever.
var syncHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "E2E Cloud Sync (Ghost Mode) via GitHub Gists",
	RunE:  runSync,
}

var syncTokenFile string

func init() {
	syncCmd.Flags().StringVar(&syncTokenFile, "token-file", "",
		"read GitHub token from file (more secure than env var — avoids ps aux exposure)")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	// Prefer --token-file over env var: env vars are visible in ps aux output.
	token := os.Getenv("RECALL_GITHUB_TOKEN")
	if syncTokenFile != "" {
		raw, err := os.ReadFile(syncTokenFile)
		if err != nil {
			return fmt.Errorf("reading token file: %w", err)
		}
		token = strings.TrimSpace(string(raw))
	}
	if token == "" {
		return fmt.Errorf("GitHub token required: set RECALL_GITHUB_TOKEN env var or use --token-file")
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

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/gists", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("building create gist request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := syncHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("creating gist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create gist: %s", string(body))
	}

	var gistResponse struct {
		ID     string `json:"id"`
		Public bool   `json:"public"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gistResponse); err != nil {
		return "", fmt.Errorf("decoding gist response: %w", err)
	}

	// Security: verify the gist was created private
	if gistResponse.Public {
		return "", fmt.Errorf("SECURITY: created Gist is public — aborting to prevent data exposure")
	}

	return gistResponse.ID, nil
}

func updateGist(token, id, content string) error {
	payload := gistPayload{
		Description: "Recall CLI - E2E Encrypted Sync Vault",
		Files: map[string]gistFile{
			"pending.ndjson.enc": {Content: content},
		},
	}
	b, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "PATCH", "https://api.github.com/gists/"+id, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("building update gist request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := syncHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("updating gist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update gist: %s", string(body))
	}

	return nil
}
