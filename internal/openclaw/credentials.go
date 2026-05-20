package openclaw

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hpp-io/hpphub-cli/internal/config"
)

func openClawConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".openclaw", "openclaw.json"), nil
}

// SyncOpenClawCredentials updates HPP provider API keys and base URLs in
// ~/.openclaw/openclaw.json to match the current hpphub login. Also backfills
// vision input metadata on supported models when missing.
func SyncOpenClawCredentials(cfg *config.Config) (bool, error) {
	if cfg == nil || !cfg.IsLoggedIn() {
		return false, nil
	}
	if !isHPPConfigured() {
		return false, nil
	}

	configPath, err := openClawConfigPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}

	var clawConfig map[string]interface{}
	if err := json.Unmarshal(data, &clawConfig); err != nil {
		return false, err
	}

	models, ok := clawConfig["models"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	providers, ok := models["providers"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	changed := false
	anthropicBaseURL := strings.Replace(cfg.BaseURL, "/llm/v1", "/v1", 1)

	if hpp, ok := providers["hpp"].(map[string]interface{}); ok {
		if hpp["apiKey"] != cfg.APIKey {
			hpp["apiKey"] = cfg.APIKey
			changed = true
		}
		if hpp["baseUrl"] != cfg.BaseURL {
			hpp["baseUrl"] = cfg.BaseURL
			changed = true
		}
	}

	if anthropic, ok := providers["hpp-anthropic"].(map[string]interface{}); ok {
		if anthropic["apiKey"] != cfg.APIKey {
			anthropic["apiKey"] = cfg.APIKey
			changed = true
		}
		if anthropic["baseUrl"] != anthropicBaseURL {
			anthropic["baseUrl"] = anthropicBaseURL
			changed = true
		}
	}

	if backfillVisionInputs(providers) {
		changed = true
	}

	if !changed {
		return false, nil
	}

	output, err := json.MarshalIndent(clawConfig, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(configPath, output, 0600); err != nil {
		return false, err
	}
	return true, nil
}

// SyncOpenClawCredentialsWithMessage syncs credentials and prints a status line.
func SyncOpenClawCredentialsWithMessage(cfg *config.Config) error {
	updated, err := SyncOpenClawCredentials(cfg)
	if err != nil {
		return fmt.Errorf("failed to sync OpenClaw credentials: %w", err)
	}
	if updated {
		fmt.Println("  ✓ OpenClaw HPP credentials synced")
	}
	return nil
}
