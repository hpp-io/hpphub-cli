package openclaw

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hpp-io/hpphub-cli/internal/api"
	"github.com/hpp-io/hpphub-cli/internal/auth"
	"github.com/hpp-io/hpphub-cli/internal/config"
)

// Launch runs the full detect → login → configure → start flow
func Launch(modelFlag string, configOnly bool, hubURL string) error {
	// Step 1: Detect OpenClaw
	fmt.Println("Checking OpenClaw installation...")
	version, err := DetectOpenClaw()
	if err != nil {
		fmt.Println("  ✗ OpenClaw not found")
		fmt.Println()
		if !promptYesNo("  Install OpenClaw?") {
			fmt.Println("  Install manually: npm install -g openclaw")
			return nil
		}
		if err := installOpenClaw(); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}
		version, err = DetectOpenClaw()
		if err != nil {
			return fmt.Errorf("OpenClaw still not found after install")
		}
	}
	fmt.Printf("  ✓ OpenClaw detected (%s)\n", version)

	// Step 2: Login check
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if hubURL != "" {
		cfg.HubURL = hubURL
	}

	if !cfg.IsLoggedIn() {
		fmt.Println()
		fmt.Println("Not logged in. Starting login flow...")
		if err := runLogin(cfg); err != nil {
			return err
		}
		// Reload config after login
		cfg, err = config.Load()
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("  ✓ Logged in as %s\n", cfg.Email)
	}

	if cfg.APIKey != "" {
		suffix := cfg.APIKey[len(cfg.APIKey)-4:]
		fmt.Printf("  ✓ API key: ...%s\n", suffix)
	}

	// Step 3: Model selection
	selectedModel := modelFlag
	if selectedModel == "" {
		models, err := api.ListModels(cfg.BaseURL, cfg.APIKey)
		if err != nil {
			return fmt.Errorf("failed to list models: %w", err)
		}
		if len(models) == 0 {
			return fmt.Errorf("no models available")
		}

		// Filter text models (exclude image models)
		var textModels []api.Model
		for _, m := range models {
			if !strings.Contains(m.ID, "dall-e") && !strings.Contains(m.ID, "image") {
				textModels = append(textModels, m)
			}
		}

		selectedModel = selectModel(textModels)
		if selectedModel == "" {
			return fmt.Errorf("no model selected")
		}
	}
	fmt.Printf("  ✓ Model: %s\n", selectedModel)

	// Step 4: Configure OpenClaw
	fmt.Println("  Configuring OpenClaw...")
	if err := configureOpenClaw(cfg, selectedModel); err != nil {
		return fmt.Errorf("failed to configure: %w", err)
	}
	fmt.Println("  ✓ HPP provider configured in OpenClaw")

	// Validate config
	if err := validateOpenClawConfig(); err != nil {
		fmt.Printf("  ⚠ Config validation: %s\n", err)
	}

	if configOnly {
		fmt.Println()
		fmt.Println("Configuration complete. Run 'openclaw gateway start' to start.")
		return nil
	}

	// Step 5: Start OpenClaw
	fmt.Println("  Starting OpenClaw gateway...")
	if err := startOpenClaw(); err != nil {
		fmt.Printf("  ⚠ Failed to start gateway: %s\n", err)
		fmt.Println("  You can start it manually: openclaw gateway start")
	} else {
		fmt.Println("  ✓ OpenClaw gateway running")
	}

	fmt.Println()
	fmt.Println("You're all set! Send a message via Telegram, WhatsApp, or other connected channels.")

	return nil
}

// detectOpenClaw checks if openclaw is installed and returns version
// DetectOpenClaw checks if openclaw is installed and returns version
func DetectOpenClaw() (string, error) {
	out, err := exec.Command("openclaw", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// installOpenClaw installs openclaw using the official install script
// The script handles Node.js detection and installation automatically
func installOpenClaw() error {
	fmt.Println("  Installing OpenClaw...")

	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("bash", "-c", "curl -fsSL https://openclaw.ai/install.sh | bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "windows":
		if isWSL() {
			cmd := exec.Command("bash", "-c", "curl -fsSL https://openclaw.ai/install.sh | bash")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
		cmd := exec.Command("powershell", "-Command", "iwr -useb https://openclaw.ai/install.ps1 | iex")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("automatic install not supported on %s.\n"+
			"Install from https://docs.openclaw.ai/install and rerun hpphub launch openclaw", runtime.GOOS)
	}
}

// runLogin performs the Device Code Flow login
func runLogin(cfg *config.Config) error {
	hub := cfg.GetHubURL()

	dc, err := auth.RequestDeviceCode(hub)
	if err != nil {
		return fmt.Errorf("failed to request code: %w", err)
	}

	fmt.Println()
	fmt.Printf("  Your code: %s\n", dc.UserCode)
	fmt.Println()

	if err := auth.OpenBrowser(dc.VerificationURL); err != nil {
		fmt.Printf("  Open this URL in your browser:\n  %s\n\n", dc.VerificationURL)
	} else {
		fmt.Println("  Browser opened. Enter the code and authorize.")
	}

	fmt.Println("  Waiting for approval...")

	token, err := auth.PollForToken(hub, dc.DeviceCode, dc.Interval, dc.ExpiresIn)
	if err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	cfg.APIKey = token.APIKey
	cfg.BaseURL = token.BaseURL
	cfg.Email = token.Email
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("  ✓ Logged in as %s\n", token.Email)
	return nil
}

// selectModel shows a simple numbered list for model selection
func selectModel(models []api.Model) string {
	fmt.Println()
	fmt.Println("  Available models:")
	for i, m := range models {
		pricing := ""
		if m.Pricing != nil {
			pricing = fmt.Sprintf(" ($%.2f/$%.2f per M tokens)", m.Pricing.Input*1e6, m.Pricing.Output*1e6)
		}
		fmt.Printf("  %2d. %s%s\n", i+1, m.ID, pricing)
	}

	fmt.Println()
	fmt.Print("  Select model (number): ")

	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil || choice < 1 || choice > len(models) {
		return ""
	}

	return models[choice-1].ID
}

// configureOpenClaw writes HPP provider config to ~/.openclaw/openclaw.json
func configureOpenClaw(cfg *config.Config, model string) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".openclaw", "openclaw.json")

	// Read existing config or start fresh
	var clawConfig map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &clawConfig)
	}
	if clawConfig == nil {
		clawConfig = make(map[string]interface{})
	}

	// Build models.providers.hpp
	models, ok := clawConfig["models"].(map[string]interface{})
	if !ok {
		models = make(map[string]interface{})
	}
	models["mode"] = "merge"

	providers, ok := models["providers"].(map[string]interface{})
	if !ok {
		providers = make(map[string]interface{})
	}

	// Fetch model list and split by provider type
	apiModels, _ := api.ListModels(cfg.BaseURL, cfg.APIKey)
	var openaiModels []map[string]interface{}
	var anthropicModels []map[string]interface{}
	for _, m := range apiModels {
		entry := map[string]interface{}{
			"id":   m.ID,
			"name": m.ID,
		}
		if strings.HasPrefix(m.ID, "anthropic/") {
			// Strip prefix for Anthropic native API (model ID without "anthropic/")
			entry["id"] = strings.TrimPrefix(m.ID, "anthropic/")
			entry["name"] = m.ID
			anthropicModels = append(anthropicModels, entry)
		} else {
			openaiModels = append(openaiModels, entry)
		}
	}

	// Derive Anthropic base URL from OpenAI base URL
	// e.g., "https://router.hpp.io/llm/v1" → "https://router.hpp.io/v1"
	anthropicBaseURL := strings.Replace(cfg.BaseURL, "/llm/v1", "/v1", 1)

	// hpp provider — OpenAI-compatible models
	providers["hpp"] = map[string]interface{}{
		"baseUrl": cfg.BaseURL,
		"apiKey":  cfg.APIKey,
		"api":     "openai-completions",
		"models":  openaiModels,
	}

	// hpp-anthropic provider — Anthropic native API models
	if len(anthropicModels) > 0 {
		providers["hpp-anthropic"] = map[string]interface{}{
			"baseUrl": anthropicBaseURL,
			"apiKey":  cfg.APIKey,
			"api":     "anthropic-messages",
			"models":  anthropicModels,
		}
	}

	models["providers"] = providers
	clawConfig["models"] = models

	// Set default model with correct provider prefix
	agents, ok := clawConfig["agents"].(map[string]interface{})
	if !ok {
		agents = make(map[string]interface{})
	}
	defaults, ok := agents["defaults"].(map[string]interface{})
	if !ok {
		defaults = make(map[string]interface{})
	}
	defaultPrimary := "hpp/" + model
	if strings.HasPrefix(model, "anthropic/") {
		defaultPrimary = "hpp-anthropic/" + strings.TrimPrefix(model, "anthropic/")
	}
	defaults["model"] = map[string]interface{}{
		"primary": defaultPrimary,
	}
	agents["defaults"] = defaults
	clawConfig["agents"] = agents

	// Ensure gateway.mode is set (required for gateway to start)
	gateway, ok := clawConfig["gateway"].(map[string]interface{})
	if !ok {
		gateway = make(map[string]interface{})
	}
	if gateway["mode"] == nil {
		gateway["mode"] = "local"
		clawConfig["gateway"] = gateway
	}

	// Write config
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	output, err := json.MarshalIndent(clawConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, output, 0644)
}

// validateOpenClawConfig runs openclaw config validate
func validateOpenClawConfig() error {
	cmd := exec.Command("openclaw", "config", "validate")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

// startOpenClaw starts the openclaw gateway
func startOpenClaw() error {
	if runtime.GOOS == "windows" {
		// Windows: run gateway in foreground (daemon not supported)
		fmt.Println("  Starting gateway in foreground (press Ctrl+C to stop)...")
		cmd := exec.Command("openclaw", "gateway")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	// macOS/Linux: start as daemon
	cmd := exec.Command("openclaw", "gateway", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isWSL detects if running inside Windows Subsystem for Linux
func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

// RunCommand runs an openclaw CLI command
func RunCommand(args ...string) error {
	cmd := exec.Command("openclaw", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// promptYesNo asks a yes/no question
func promptYesNo(question string) bool {
	fmt.Printf("%s (Y/n): ", question)
	var answer string
	fmt.Scanln(&answer)
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes"
}
