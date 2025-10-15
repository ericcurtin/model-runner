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
// Examples:
//   - "gemma3" -> "ai/gemma3:latest"
//   - "gemma3:v1" -> "ai/gemma3:v1"
//   - "myorg/gemma3" -> "myorg/gemma3:latest"
//   - "ai/gemma3:latest" -> "ai/gemma3:latest" (unchanged)
//   - "hf.co/model" -> "hf.co/model" (unchanged - has registry)
func normalizeModelName(model string) string {
	// If the model is empty, return as-is
	if model == "" {
		return model
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
//   - "ai/gemma3:v1" -> "gemma3:v1"
//   - "myorg/gemma3:latest" -> "myorg/gemma3:latest"
//   - "gemma3:latest" -> "gemma3"
//   - "hf.co/bartowski/model:latest" -> "hf.co/bartowski/model:latest"
func stripDefaultsFromModelName(model string) string {
	// Check if model has ai/ prefix without tag (implicitly :latest)
	if strings.HasPrefix(model, defaultOrg+"/") {
		// Strip ai/ prefix
		model = strings.TrimPrefix(model, defaultOrg+"/")
		return model
	}

	// Check if model has :latest but no slash (no org specified)
	if strings.HasSuffix(model, ":"+defaultTag) {
		// Strip :latest
		model = strings.TrimSuffix(model, ":"+defaultTag)
		return model
	}

	// For other cases (custom org with :latest, hf.co with :latest, ai/ with custom tag, etc.), keep as-is
	return model
}
