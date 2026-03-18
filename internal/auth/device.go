package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	APIKey    string `json:"api_key,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	Email     string `json:"email,omitempty"`
	Error     string `json:"error,omitempty"`
	ErrorDesc string `json:"error_description,omitempty"`
}

// RequestDeviceCode calls POST /auth/device/code
func RequestDeviceCode(hubURL string) (*DeviceCodeResponse, error) {
	resp, err := http.Post(hubURL+"/auth/device/code", "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Hub: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Hub returned %d: %s", resp.StatusCode, string(body))
	}

	var result DeviceCodeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &result, nil
}

// PollForToken polls POST /auth/device/token until approved or expired
func PollForToken(hubURL string, deviceCode string, interval int, timeout int) (*TokenResponse, error) {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	pollInterval := time.Duration(interval) * time.Second

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		payload, _ := json.Marshal(map[string]string{"device_code": deviceCode})
		resp, err := http.Post(hubURL+"/auth/device/token", "application/json", bytes.NewReader(payload))
		if err != nil {
			continue // Retry on network error
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result TokenResponse
		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		// Still pending
		if result.Error == "authorization_pending" {
			continue
		}

		// Error
		if result.Error != "" {
			return nil, fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
		}

		// Success
		return &result, nil
	}

	return nil, fmt.Errorf("authorization timed out after %d seconds", timeout)
}

// OpenBrowser opens a URL in the default browser
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
