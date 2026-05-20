package openclaw

import (
	"strings"

	"github.com/hpp-io/hpphub-cli/internal/api"
)

var visionInput = []interface{}{"text", "image"}

// modelSupportsVision reports whether a router model ID accepts image input in
// OpenClaw WebChat. Custom providers default to text-only unless input is set.
func modelSupportsVision(modelID string) bool {
	if modelID == "" {
		return false
	}
	lower := strings.ToLower(modelID)
	if strings.Contains(lower, "ollama/") {
		return false
	}
	if strings.Contains(lower, "gpt-image") || strings.Contains(lower, "dall-e") {
		return false
	}
	if strings.Contains(lower, "o3-mini") {
		return false
	}
	if strings.Contains(lower, "claude-") {
		return true
	}
	if strings.Contains(lower, "gpt-4o") || strings.Contains(lower, "gpt-4.1") {
		return true
	}
	if strings.Contains(lower, "gpt-5") {
		return true
	}
	if strings.Contains(lower, "/o3") || strings.HasSuffix(lower, "o3") {
		return true
	}
	if strings.Contains(lower, "o4-mini") {
		return true
	}
	return false
}

func buildModelEntry(m api.Model) map[string]interface{} {
	entry := map[string]interface{}{
		"id":   m.ID,
		"name": m.ID,
	}
	if strings.HasPrefix(m.ID, "anthropic/") {
		entry["id"] = strings.TrimPrefix(m.ID, "anthropic/")
		entry["name"] = m.ID
	}
	if modelSupportsVision(m.ID) {
		entry["input"] = visionInput
	}
	return entry
}

func applyVisionInputToEntry(entry map[string]interface{}) bool {
	id, _ := entry["id"].(string)
	name, _ := entry["name"].(string)
	if !modelSupportsVision(id) && !modelSupportsVision(name) {
		return false
	}
	if existing, ok := entry["input"].([]interface{}); ok && len(existing) >= 2 {
		return false
	}
	entry["input"] = visionInput
	return true
}

func backfillVisionInputs(providers map[string]interface{}) bool {
	changed := false
	for _, providerName := range []string{"hpp", "hpp-anthropic"} {
		provider, ok := providers[providerName].(map[string]interface{})
		if !ok {
			continue
		}
		models, ok := provider["models"].([]interface{})
		if !ok {
			continue
		}
		for _, raw := range models {
			entry, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if applyVisionInputToEntry(entry) {
				changed = true
			}
		}
	}
	return changed
}
