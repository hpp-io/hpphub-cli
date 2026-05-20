package openclaw

import (
	"fmt"

	"github.com/hpp-io/hpphub-cli/internal/config"
)

// PrintAccountSummary prints the active HPP account details after login.
func PrintAccountSummary(cfg *config.Config) {
	fmt.Printf("  ✓ Logged in as %s\n", cfg.Email)
	if cfg.GetHubURL() != "" {
		fmt.Printf("  ✓ Hub:    %s\n", cfg.GetHubURL())
	}
	if cfg.BaseURL != "" {
		fmt.Printf("  ✓ Router: %s\n", cfg.BaseURL)
	}
	if cfg.APIKey != "" {
		suffix := cfg.APIKey[len(cfg.APIKey)-4:]
		fmt.Printf("  ✓ API key: ...%s\n", suffix)
	}
}
