package commands

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/docker/cli/cli-plugins/hooks"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/pkg/errors"
)

const (
	defaultOrg = "ai"
	defaultTag = "latest"
)

const (
	enableViaCLI = "Enable Docker Model Runner via the CLI → docker desktop enable model-runner"
	enableViaGUI = "Enable Docker Model Runner via the GUI → Go to Settings->AI->Enable Docker Model Runner"
)

var notRunningErr = fmt.Errorf("Docker Model Runner is not running. Please start it and try again.\n")

func handleClientError(err error, message string) error {
	if errors.Is(err, desktop.ErrServiceUnavailable) {
		return notRunningErr
	}
	return errors.Wrap(err, message)
}

func handleNotRunningError(err error) error {
	if errors.Is(err, notRunningErr) {
		var buf bytes.Buffer
		hooks.PrintNextSteps(&buf, []string{enableViaCLI, enableViaGUI})
		return fmt.Errorf("%w\n%s", err, strings.TrimRight(buf.String(), "\n"))
	}
	return err
}

// normalizeModelName adds the default organization prefix (ai/) and tag (:latest) if missing.
// It also converts Hugging Face model names to lowercase.
// Examples:
//   - "gemma3" -> "ai/gemma3:latest"
//   - "gemma3:v1" -> "ai/gemma3:v1"
//   - "myorg/gemma3" -> "myorg/gemma3:latest"
//   - "ai/gemma3:latest" -> "ai/gemma3:latest" (unchanged)
//   - "hf.co/model" -> "hf.co/model:latest" (unchanged - has registry)
//   - "hf.co/Model" -> "hf.co/model:latest" (converted to lowercase)
func normalizeModelName(model string) string {
	// If the model is empty, return as-is
	if model == "" {
		return model
	}

	// Normalize HuggingFace model names to lowercase first
	if strings.HasPrefix(model, "hf.co/") {
		model = strings.ToLower(model)
	}

	// If model starts with "hf.co/" or contains a registry (has a dot before the first slash),
	// don't add default org
	if strings.HasPrefix(model, "hf.co/") {
		// For HuggingFace models, just ensure :latest tag if no tag specified
		if !strings.Contains(model, ":") {
			return model + ":" + defaultTag
		}
		return model
	}

	// Check if model contains a registry (domain with dot before first slash)
	firstSlash := strings.Index(model, "/")
	if firstSlash > 0 && strings.Contains(model[:firstSlash], ".") {
		// Has a registry, just ensure tag
		if !strings.Contains(model, ":") {
			return model + ":" + defaultTag
		}
		return model
	}

	// Split by colon to check for tag
	parts := strings.SplitN(model, ":", 2)
	nameWithOrg := parts[0]
	tag := defaultTag
	if len(parts) == 2 {
		tag = parts[1]
	}

	// If name doesn't contain a slash, add the default org
	if !strings.Contains(nameWithOrg, "/") {
		nameWithOrg = defaultOrg + "/" + nameWithOrg
	}

	return nameWithOrg + ":" + tag
}

// stripDefaultsFromModelName removes the default "ai/" prefix and ":latest" tag for display.
// Examples:
//   - "ai/gemma3:latest" -> "gemma3"
//   - "ai/gemma3:v1" -> "ai/gemma3:v1"
//   - "myorg/gemma3:latest" -> "myorg/gemma3:latest"
//   - "gemma3:latest" -> "gemma3"
//   - "hf.co/bartowski/model:latest" -> "hf.co/bartowski/model:latest"
func stripDefaultsFromModelName(model string) string {
	// For models with a registry (hf.co, docker.io, etc.), keep as-is
	if strings.Contains(model, ".") {
		return model
	}

	// Check if model has ai/ prefix without tag (implicitly :latest) - strip just ai/
	if strings.HasPrefix(model, defaultOrg+"/") {
		model = strings.TrimPrefix(model, defaultOrg+"/")
	}

	// Check if model has :latest but no slash (no org specified) - strip :latest
	if strings.HasSuffix(model, ":"+defaultTag) {
		model = strings.TrimSuffix(model, ":"+defaultTag)
	}

	// For other cases (ai/ with custom tag, custom org with :latest, etc.), keep as-is
	return model
}
