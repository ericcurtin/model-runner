package commands

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
)

// ValidBackends is a map of valid backends
var ValidBackends = map[string]bool{
	"llama.cpp": true,
	"openai":    true,
}

// ServerPreset represents a preconfigured server endpoint
type ServerPreset struct {
	Name string
	URL  string
}

// ServerPresets defines the available server presets
var ServerPresets = []ServerPreset{
	{"dmr", "http://127.0.0.1:12434/engines/llama.cpp/v1"},
	{"llamacpp", "http://127.0.0.1:8080/v1"},
	{"ollama", "http://127.0.0.1:11434/v1"},
	{"openrouter", "https://openrouter.ai/api/v1"},
}

// validateBackend checks if the provided backend is valid
func validateBackend(backend string) error {
	if !ValidBackends[backend] {
		return fmt.Errorf("invalid backend '%s'. Valid backends are: %s",
			backend, ValidBackendsKeys())
	}
	return nil
}

// ensureAPIKey retrieves the API key if needed
func ensureAPIKey(backend string) (string, error) {
	if backend == "openai" {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return "", errors.New("OPENAI_API_KEY environment variable is required when using --backend=openai")
		}
		return apiKey, nil
	}
	return "", nil
}

// resolveServerURL determines the server URL from flags
// Returns: (url, useOpenAI, apiKey, error)
func resolveServerURL(host, customURL string, port int, dmr, llamacpp, ollama, openrouter bool) (string, bool, string, error) {
	// Count how many server options are specified
	presetCount := 0
	if dmr {
		presetCount++
	}
	if llamacpp {
		presetCount++
	}
	if ollama {
		presetCount++
	}
	if openrouter {
		presetCount++
	}
	if customURL != "" {
		presetCount++
	}

	// Check for conflicting options
	if presetCount > 1 {
		return "", false, "", errors.New("only one of --url, --dmr, --llamacpp, --ollama, or --openrouter can be specified")
	}

	// Check for conflicting host/port with URL/preset options
	hostPortSpecified := host != "" || port != 0
	urlPresetSpecified := customURL != "" || dmr || llamacpp || ollama || openrouter

	if hostPortSpecified && urlPresetSpecified {
		return "", false, "", errors.New("cannot specify both --host/--port and --url/preset options (--dmr, --llamacpp, --ollama, --openrouter)")
	}

	// Resolve the URL
	var serverURL string
	useOpenAI := false
	apiKey := ""

	if customURL != "" {
		serverURL = customURL
		useOpenAI = true
	} else if dmr {
		serverURL = "http://127.0.0.1:12434/engines/llama.cpp/v1"
		useOpenAI = true
	} else if llamacpp {
		serverURL = "http://127.0.0.1:8080/v1"
		useOpenAI = true
	} else if ollama {
		serverURL = "http://127.0.0.1:11434/v1"
		useOpenAI = true
	} else if openrouter {
		serverURL = "https://openrouter.ai/api/v1"
		useOpenAI = true
		// For openrouter, require API key
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return "", false, "", errors.New("OPENAI_API_KEY environment variable is required when using --openrouter")
		}
	} else if hostPortSpecified {
		// Use custom host/port for model-runner endpoint
		if host == "" {
			host = "127.0.0.1"
		}
		if port == 0 {
			port = 12434
		}
		serverURL = fmt.Sprintf("http://%s:%d", host, port)
		useOpenAI = false
	}

	// For OpenAI-compatible endpoints, check for API key (optional for most, required for openrouter)
	if useOpenAI && apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	return serverURL, useOpenAI, apiKey, nil
}

func ValidBackendsKeys() string {
	keys := slices.Collect(maps.Keys(ValidBackends))
	slices.Sort(keys)
	return strings.Join(keys, ", ")
}
