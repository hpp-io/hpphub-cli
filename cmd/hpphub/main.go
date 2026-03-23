package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hpp-io/hpphub-cli/internal/api"
	"github.com/hpp-io/hpphub-cli/internal/auth"
	"github.com/hpp-io/hpphub-cli/internal/config"
	"github.com/hpp-io/hpphub-cli/internal/openclaw"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	cobra.CheckErr(newCLI().ExecuteContext(context.Background()))
}

func newCLI() *cobra.Command {
	root := &cobra.Command{
		Use:     "hpphub",
		Short:   "HPP Hub CLI — connect OpenClaw to HPP",
		Version: version,
	}

	root.AddCommand(
		loginCmd(),
		logoutCmd(),
		whoamiCmd(),
		modelsCmd(),
		launchCmd(),
		setupCmd(),
		uninstallCmd(),
	)

	return root
}

// ─── login ───────────────────────────────────────────────────

func loginCmd() *cobra.Command {
	var hubURL string
	var force bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to HPP Hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if cfg.IsLoggedIn() && !force {
				fmt.Printf("Already logged in as %s\n", cfg.Email)
				fmt.Println("Use --force to re-login.")
				return nil
			}

			if hubURL != "" {
				cfg.HubURL = hubURL
			}
			hub := cfg.GetHubURL()

			// Step 1: Request device code
			fmt.Println("Requesting device code...")
			dc, err := auth.RequestDeviceCode(hub)
			if err != nil {
				return fmt.Errorf("failed to request code: %w", err)
			}

			// Step 2: Show code and open browser
			fmt.Println()
			fmt.Printf("  Your code: %s\n", dc.UserCode)
			fmt.Println()

			if err := auth.OpenBrowser(dc.VerificationURL); err != nil {
				fmt.Printf("  Open this URL in your browser:\n  %s\n\n", dc.VerificationURL)
			} else {
				fmt.Println("  Browser opened. Enter the code and authorize.")
			}

			fmt.Println("  Waiting for approval...")

			// Step 3: Poll for token
			token, err := auth.PollForToken(hub, dc.DeviceCode, dc.Interval, dc.ExpiresIn)
			if err != nil {
				return fmt.Errorf("authorization failed: %w", err)
			}

			// Step 4: Save config
			cfg.APIKey = token.APIKey
			cfg.BaseURL = token.BaseURL
			cfg.Email = token.Email
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Println()
			fmt.Printf("  ✓ Logged in as %s\n", token.Email)
			if token.APIKey != "" {
				suffix := token.APIKey[len(token.APIKey)-4:]
				fmt.Printf("  ✓ API key saved: ...%s\n", suffix)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&hubURL, "hub-url", "", "Hub URL (default: https://hub.hpp.io)")
	cmd.Flags().BoolVar(&force, "force", false, "Force re-login")

	return cmd
}

// ─── logout ──────────────────────────────────────────────────

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out from HPP Hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if !cfg.IsLoggedIn() {
				fmt.Println("Not logged in.")
				return nil
			}
			if err := cfg.Clear(); err != nil {
				return err
			}
			fmt.Println("Logged out.")
			return nil
		},
	}
}

// ─── whoami ──────────────────────────────────────────────────

func whoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current login status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if !cfg.IsLoggedIn() {
				fmt.Println("Not logged in. Run 'hpphub login' first.")
				return nil
			}
			fmt.Printf("Logged in as %s\n", cfg.Email)
			return nil
		},
	}
}

// ─── models ──────────────────────────────────────────────────

func modelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available models",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if !cfg.IsLoggedIn() {
				return fmt.Errorf("not logged in. Run 'hpphub login' first")
			}

			models, err := api.ListModels(cfg.BaseURL, cfg.APIKey)
			if err != nil {
				return err
			}

			if len(models) == 0 {
				fmt.Println("No models available.")
				return nil
			}

			fmt.Printf("%-12s %-42s %10s %10s\n", "PROVIDER", "MODEL", "INPUT", "OUTPUT")
			fmt.Println(strings.Repeat("─", 78))

			for _, m := range models {
				input := ""
				output := ""
				if m.Pricing != nil {
					input = fmt.Sprintf("$%.2f/M", m.Pricing.Input*1e6)
					output = fmt.Sprintf("$%.2f/M", m.Pricing.Output*1e6)
				}
				fmt.Printf("%-12s %-42s %10s %10s\n", m.OwnedBy, m.ID, input, output)
			}

			fmt.Printf("\n%d models available\n", len(models))
			return nil
		},
	}
}

// ─── launch ──────────────────────────────────────────────────

func launchCmd() *cobra.Command {
	var model string
	var configOnly bool
	var hubURL string

	cmd := &cobra.Command{
		Use:   "launch <integration>",
		Short: "Launch an integration with HPP (e.g., openclaw)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			integration := strings.ToLower(args[0])

			switch integration {
			case "openclaw":
				return openclaw.Launch(model, configOnly, hubURL)
			case "claude":
				persist, _ := cmd.Flags().GetBool("persist")
				unpersist, _ := cmd.Flags().GetBool("unpersist")
				if unpersist {
					return unpersistClaudeConfig()
				}
				return launchClaude(model, hubURL, persist)
			default:
				return fmt.Errorf("unknown integration: %s\nAvailable: openclaw, claude", integration)
			}
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "Model to use (skip interactive selection)")
	cmd.Flags().BoolVar(&configOnly, "config", false, "Configure only, don't start")
	cmd.Flags().StringVar(&hubURL, "hub-url", "", "Hub URL (default: https://hub.hpp.io)")
	cmd.Flags().Bool("persist", false, "Save HPP settings to shell profile (claude works without hpphub)")
	cmd.Flags().Bool("unpersist", false, "Remove HPP settings from shell profile")

	return cmd
}

// ─── launch claude ──────────────────────────────────────────

func launchClaude(model string, hubURL string, persist bool) error {
	// Step 1: Check Claude Code
	fmt.Println("Checking Claude Code installation...")
	claudePath, err := findClaude()
	if err != nil {
		fmt.Println("  ✗ Claude Code not found")
		fmt.Println()
		if !promptYesNo("  Install Claude Code?") {
			fmt.Println("  Install from https://code.claude.com/docs/en/quickstart")
			return nil
		}
		if err := installClaude(); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}
		claudePath, err = findClaude()
		if err != nil {
			return fmt.Errorf("Claude Code still not found after install")
		}
	}
	fmt.Printf("  ✓ Claude Code detected (%s)\n", claudePath)

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
		if err := openclaw.RunLogin(cfg); err != nil {
			return err
		}
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

	// Step 3: Set model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	// Strip anthropic/ prefix if present
	model = strings.TrimPrefix(model, "anthropic/")
	fmt.Printf("  ✓ Model: %s\n", model)

	// Step 4: Derive Anthropic base URL
	// Claude Code appends /v1/messages itself, so base URL should not include /v1
	// e.g., "https://router.hpp.io/llm/v1" → "https://router.hpp.io"
	anthropicBaseURL := strings.Replace(cfg.BaseURL, "/llm/v1", "", 1)

	// Step 5: Persist or launch
	if persist {
		return persistClaudeConfig(anthropicBaseURL, cfg.APIKey, model)
	}

	fmt.Println()
	fmt.Println("  Starting Claude Code with HPP...")
	fmt.Println()

	cmd := exec.Command(claudePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"ANTHROPIC_BASE_URL="+anthropicBaseURL,
		"ANTHROPIC_API_KEY="+cfg.APIKey,
		"ANTHROPIC_DEFAULT_SONNET_MODEL="+model,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL="+model,
		"ANTHROPIC_DEFAULT_OPUS_MODEL="+model,
		"CLAUDE_CODE_SUBAGENT_MODEL="+model,
	)
	return cmd.Run()
}

func persistClaudeConfig(baseURL, apiKey, model string) error {
	home, _ := os.UserHomeDir()

	// Detect shell profile
	shell := os.Getenv("SHELL")
	var profilePath string
	switch {
	case strings.Contains(shell, "zsh"):
		profilePath = filepath.Join(home, ".zshrc")
	case strings.Contains(shell, "bash"):
		profilePath = filepath.Join(home, ".bashrc")
	default:
		if runtime.GOOS == "windows" {
			// Windows PowerShell profile
			profilePath = filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		} else {
			profilePath = filepath.Join(home, ".bashrc")
		}
	}

	marker := "# hpphub claude config"
	var lines []string

	if runtime.GOOS == "windows" {
		lines = []string{
			marker,
			fmt.Sprintf(`$env:ANTHROPIC_BASE_URL = "%s"`, baseURL),
			fmt.Sprintf(`$env:ANTHROPIC_API_KEY = "%s"`, apiKey),
			fmt.Sprintf(`$env:ANTHROPIC_DEFAULT_SONNET_MODEL = "%s"`, model),
			fmt.Sprintf(`$env:ANTHROPIC_DEFAULT_HAIKU_MODEL = "%s"`, model),
			fmt.Sprintf(`$env:ANTHROPIC_DEFAULT_OPUS_MODEL = "%s"`, model),
			fmt.Sprintf(`$env:CLAUDE_CODE_SUBAGENT_MODEL = "%s"`, model),
			"# end hpphub claude config",
		}
	} else {
		lines = []string{
			marker,
			fmt.Sprintf(`export ANTHROPIC_BASE_URL="%s"`, baseURL),
			fmt.Sprintf(`export ANTHROPIC_API_KEY="%s"`, apiKey),
			fmt.Sprintf(`export ANTHROPIC_DEFAULT_SONNET_MODEL="%s"`, model),
			fmt.Sprintf(`export ANTHROPIC_DEFAULT_HAIKU_MODEL="%s"`, model),
			fmt.Sprintf(`export ANTHROPIC_DEFAULT_OPUS_MODEL="%s"`, model),
			fmt.Sprintf(`export CLAUDE_CODE_SUBAGENT_MODEL="%s"`, model),
			"# end hpphub claude config",
		}
	}

	block := strings.Join(lines, "\n") + "\n"

	// Read existing profile
	existing, _ := os.ReadFile(profilePath)
	content := string(existing)

	// Remove old config if exists
	if idx := strings.Index(content, marker); idx >= 0 {
		endMarker := "# end hpphub claude config"
		if endIdx := strings.Index(content[idx:], endMarker); endIdx >= 0 {
			content = content[:idx] + content[idx+endIdx+len(endMarker)+1:]
		}
	}

	// Append new config
	content = strings.TrimRight(content, "\n") + "\n\n" + block

	if err := os.MkdirAll(filepath.Dir(profilePath), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(profilePath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  ✓ HPP settings saved to %s\n", profilePath)
	fmt.Println()
	fmt.Println("  Restart your terminal, then just run:")
	fmt.Println("    claude")
	fmt.Println()
	fmt.Println("  To remove, run:")
	fmt.Println("    hpphub launch claude --unpersist")

	return nil
}

func unpersistClaudeConfig() error {
	home, _ := os.UserHomeDir()
	shell := os.Getenv("SHELL")
	var profilePath string
	switch {
	case strings.Contains(shell, "zsh"):
		profilePath = filepath.Join(home, ".zshrc")
	case strings.Contains(shell, "bash"):
		profilePath = filepath.Join(home, ".bashrc")
	default:
		if runtime.GOOS == "windows" {
			profilePath = filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		} else {
			profilePath = filepath.Join(home, ".bashrc")
		}
	}

	existing, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", profilePath, err)
	}

	content := string(existing)
	marker := "# hpphub claude config"
	endMarker := "# end hpphub claude config"

	idx := strings.Index(content, marker)
	if idx < 0 {
		fmt.Println("  No HPP settings found in shell profile.")
		return nil
	}

	endIdx := strings.Index(content[idx:], endMarker)
	if endIdx >= 0 {
		content = content[:idx] + content[idx+endIdx+len(endMarker)+1:]
	}

	content = strings.TrimRight(content, "\n") + "\n"
	if err := os.WriteFile(profilePath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Printf("  ✓ HPP settings removed from %s\n", profilePath)
	fmt.Println("  Restart your terminal to apply.")
	return nil
}

func findClaude() (string, error) {
	if p, err := exec.LookPath("claude"); err == nil {
		return p, nil
	}
	// Check common install locations
	home, _ := os.UserHomeDir()
	name := "claude"
	if runtime.GOOS == "windows" {
		name = "claude.exe"
	}
	candidates := []string{
		filepath.Join(home, ".claude", "local", name),
		filepath.Join(home, ".local", "bin", name),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("claude not found")
}

func installClaude() error {
	fmt.Println("  Installing Claude Code...")
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("bash", "-c", "curl -fsSL https://claude.ai/install.sh | bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "windows":
		cmd := exec.Command("powershell", "-Command", "irm https://claude.ai/install.ps1 | iex")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("automatic install not supported on %s.\nInstall from https://code.claude.com/docs/en/quickstart", runtime.GOOS)
	}
}

func promptYesNo(question string) bool {
	fmt.Printf("%s (Y/n): ", question)
	var answer string
	fmt.Scanln(&answer)
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes"
}

// ─── setup ──────────────────────────────────────────────────

func setupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup <channel>",
		Short: "Set up a messaging channel (e.g., telegram)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			channel := strings.ToLower(args[0])

			switch channel {
			case "telegram":
				return setupTelegram()
			default:
				return fmt.Errorf("unknown channel: %s\nAvailable: telegram", channel)
			}
		},
	}

	return cmd
}

func setupTelegram() error {
	// Check OpenClaw
	if _, err := openclaw.DetectOpenClaw(); err != nil {
		return fmt.Errorf("OpenClaw is not installed. Run 'hpphub launch openclaw' first")
	}

	// Run Telegram setup
	openclaw.SetupTelegram()

	// Restart gateway or guide
	if runtime.GOOS == "windows" {
		fmt.Println()
		fmt.Println("  To apply changes, start the gateway in a new terminal:")
		fmt.Println("    openclaw gateway")
	} else {
		fmt.Println("  Restarting gateway...")
		if err := openclaw.RunCommand("gateway", "restart"); err != nil {
			fmt.Printf("  ⚠ Gateway restart failed: %s\n", err)
			fmt.Println("  Try manually: openclaw gateway restart")
			return nil
		}
		fmt.Println("  ✓ Gateway restarted")
	}

	// Health check
	fmt.Println("  Checking connection...")
	if err := openclaw.RunCommand("health"); err != nil {
		fmt.Printf("  ⚠ Health check: %s\n", err)
	} else {
		fmt.Println("  ✓ Telegram bot connected!")
	}

	fmt.Println()
	fmt.Println("  Send a message to your bot in Telegram to test it.")

	return nil
}

// ─── uninstall ──────────────────────────────────────────────

func uninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove hpphub CLI and its configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()

			// Step 1: Remove HPP provider from OpenClaw config
			if _, err := openclaw.DetectOpenClaw(); err == nil {
				fmt.Println("  Removing HPP provider from OpenClaw config...")
				_ = openclaw.RunCommand("config", "unset", "models.providers.hpp")
				fmt.Println("  ✓ HPP provider removed")
			}

			// Step 2: Remove hpphub config
			configDir := config.Dir()
			if _, err := os.Stat(configDir); err == nil {
				if err := os.RemoveAll(configDir); err != nil {
					fmt.Printf("  ⚠ Failed to remove %s: %s\n", configDir, err)
				} else {
					fmt.Printf("  ✓ Config removed (%s)\n", configDir)
				}
			}

			// Step 3: Remove binary
			exe, err := os.Executable()
			if err == nil {
				fmt.Printf("  To complete uninstall, remove the binary:\n")
				if runtime.GOOS == "windows" {
					fmt.Printf("    del \"%s\"\n", exe)
				} else {
					fmt.Printf("    sudo rm %s\n", exe)
				}
			}

			fmt.Println()
			fmt.Println("  ✓ hpphub uninstalled")
			return nil
		},
	}
}
