package main

import (
	"context"
	"fmt"
	"os"
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
			default:
				return fmt.Errorf("unknown integration: %s\nAvailable: openclaw", integration)
			}
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "Model to use (skip interactive selection)")
	cmd.Flags().BoolVar(&configOnly, "config", false, "Configure only, don't start")
	cmd.Flags().StringVar(&hubURL, "hub-url", "", "Hub URL (default: https://hub.hpp.io)")

	return cmd
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
	// Step 1: Check OpenClaw
	if _, err := openclaw.DetectOpenClaw(); err != nil {
		return fmt.Errorf("OpenClaw is not installed. Run 'hpphub launch openclaw' first")
	}

	// Step 2: Guide
	fmt.Println()
	fmt.Println("  To create a Telegram bot:")
	fmt.Println()
	fmt.Println("  1. Open Telegram and talk to @BotFather")
	fmt.Println("  2. Send /newbot and follow the steps")
	fmt.Println("  3. Copy the bot token")
	fmt.Println()

	// Step 3: Token input
	fmt.Print("  Paste your Telegram bot token: ")
	var token string
	if _, err := fmt.Scanln(&token); err != nil || token == "" {
		return fmt.Errorf("no token provided")
	}
	token = strings.TrimSpace(token)

	// Step 4: Set token in OpenClaw
	fmt.Println("  Configuring Telegram...")
	if err := openclaw.RunCommand("config", "set", "channels.telegram.botToken", token); err != nil {
		return fmt.Errorf("failed to set bot token: %w", err)
	}
	fmt.Println("  ✓ Bot token saved")

	// Step 5: User ID (optional but recommended)
	fmt.Println()
	fmt.Println("  To restrict who can use the bot, enter your Telegram user ID.")
	fmt.Println("  (Get it from @userinfobot in Telegram)")
	fmt.Println()
	fmt.Print("  Your Telegram user ID (or press Enter to skip): ")
	var userID string
	fmt.Scanln(&userID)
	userID = strings.TrimSpace(userID)

	if userID != "" {
		// Validate: must be numeric
		isNumeric := true
		for _, c := range userID {
			if c < '0' || c > '9' {
				isNumeric = false
				break
			}
		}
		if !isNumeric {
			fmt.Println("  ⚠ Telegram user ID must be a number (e.g., 8228669492)")
			fmt.Println("    Get it from @userinfobot in Telegram")
			fmt.Println("  Skipped — bot will use pairing mode")
		} else {
			allowFrom := fmt.Sprintf(`["%s"]`, userID)
			if err := openclaw.RunCommand("config", "set", "channels.telegram.allowFrom", allowFrom); err != nil {
				fmt.Printf("  ⚠ Failed to set allowFrom: %s\n", err)
			} else {
				fmt.Println("  ✓ Access restricted to your account")
			}
		}
	} else {
		fmt.Println("  ⚠ Skipped — bot will use pairing mode (new users need approval)")
	}

	// Step 6: Restart gateway
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

	// Step 7: Health check
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
