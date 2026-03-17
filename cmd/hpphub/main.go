package main

import (
	"context"
	"fmt"
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
			cfg.Token = token.AccessToken
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

	cmd := &cobra.Command{
		Use:   "launch <integration>",
		Short: "Launch an integration with HPP (e.g., openclaw)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			integration := strings.ToLower(args[0])

			switch integration {
			case "openclaw":
				return openclaw.Launch(model, configOnly)
			default:
				return fmt.Errorf("unknown integration: %s\nAvailable: openclaw", integration)
			}
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "Model to use (skip interactive selection)")
	cmd.Flags().BoolVar(&configOnly, "config", false, "Configure only, don't start")

	return cmd
}
