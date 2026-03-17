package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Model struct {
	ID      string  `json:"id"`
	OwnedBy string  `json:"owned_by"`
	Pricing *Pricing `json:"pricing,omitempty"`
}

type Pricing struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

type ModelsResponse struct {
	Data []Model `json:"data"`
}

// ListModels fetches available models from the router
func ListModels(baseURL string, apiKey string) ([]Model, error) {
	req, err := http.NewRequest("GET", baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var result ModelsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return result.Data, nil
}
