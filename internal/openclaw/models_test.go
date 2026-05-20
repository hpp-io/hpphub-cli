package openclaw

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hpp-io/hpphub-cli/internal/api"
	"github.com/hpp-io/hpphub-cli/internal/config"
)

func TestModelSupportsVision(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"openai/gpt-4o", true},
		{"openai/gpt-4.1-mini", true},
		{"openai/gpt-5-nano", true},
		{"openai/o3", true},
		{"openai/o3-mini", false},
		{"openai/o4-mini", true},
		{"openai/gpt-image-1", false},
		{"ollama/gpt-oss:120b", false},
		{"anthropic/claude-sonnet-4-6", true},
		{"claude-sonnet-4-6", true},
	}
	for _, tt := range tests {
		if got := modelSupportsVision(tt.id); got != tt.want {
			t.Errorf("modelSupportsVision(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestBuildModelEntry(t *testing.T) {
	vision := buildModelEntry(api.Model{ID: "openai/gpt-4o"})
	if vision["input"] == nil {
		t.Fatal("expected vision input on gpt-4o")
	}

	textOnly := buildModelEntry(api.Model{ID: "ollama/gpt-oss:120b"})
	if textOnly["input"] != nil {
		t.Fatal("did not expect input on ollama model")
	}

	anthropic := buildModelEntry(api.Model{ID: "anthropic/claude-sonnet-4-6"})
	if anthropic["id"] != "claude-sonnet-4-6" {
		t.Fatalf("id = %v", anthropic["id"])
	}
	if anthropic["input"] == nil {
		t.Fatal("expected vision input on claude model")
	}
}

func TestSyncOpenClawCredentials(t *testing.T) {
	home := t.TempDir()
	openclawDir := filepath.Join(home, ".openclaw")
	if err := os.MkdirAll(openclawDir, 0700); err != nil {
		t.Fatal(err)
	}

	initial := map[string]interface{}{
		"models": map[string]interface{}{
			"providers": map[string]interface{}{
				"hpp": map[string]interface{}{
					"apiKey":  "hpph_oldkey",
					"baseUrl": "https://router.hpp.io/llm/v1",
					"models": []interface{}{
						map[string]interface{}{
							"id":   "openai/gpt-4o",
							"name": "openai/gpt-4o",
						},
					},
				},
				"hpp-anthropic": map[string]interface{}{
					"apiKey":  "hpph_oldkey",
					"baseUrl": "https://router.hpp.io/v1",
					"models": []interface{}{
						map[string]interface{}{
							"id":   "claude-sonnet-4-6",
							"name": "anthropic/claude-sonnet-4-6",
						},
					},
				},
			},
		},
	}
	data, err := json.MarshalIndent(initial, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(openclawDir, "openclaw.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)

	cfg := &config.Config{
		APIKey:  "hpph_newkey",
		BaseURL: "https://router.hpp.io/llm/v1",
		Email:   "new@example.com",
	}

	updated, err := SyncOpenClawCredentials(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("expected credentials sync to update file")
	}

	out, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]interface{}
	if err := json.Unmarshal(out, &saved); err != nil {
		t.Fatal(err)
	}
	providers := saved["models"].(map[string]interface{})["providers"].(map[string]interface{})
	hpp := providers["hpp"].(map[string]interface{})
	if hpp["apiKey"] != "hpph_newkey" {
		t.Fatalf("hpp apiKey = %v", hpp["apiKey"])
	}
	anthropic := providers["hpp-anthropic"].(map[string]interface{})
	if anthropic["apiKey"] != "hpph_newkey" {
		t.Fatalf("anthropic apiKey = %v", anthropic["apiKey"])
	}

	models := hpp["models"].([]interface{})
	entry := models[0].(map[string]interface{})
	if entry["input"] == nil {
		t.Fatal("expected vision input backfill on gpt-4o")
	}
}

func TestSyncOpenClawCredentialsNoOpWhenNotConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &config.Config{APIKey: "hpph_test", BaseURL: "https://router.hpp.io/llm/v1"}
	updated, err := SyncOpenClawCredentials(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("expected no update when openclaw is not configured")
	}
}
