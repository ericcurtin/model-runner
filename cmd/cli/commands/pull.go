package commands

import (
	"fmt"
	"os"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/docker/model-runner/cmd/cli/pkg/model"
	"github.com/docker/model-runner/cmd/cli/pkg/ollama"
	"github.com/docker/model-runner/cmd/cli/pkg/standalone"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	var ignoreRuntimeMemoryCheck bool

	c := &cobra.Command{
		Use:   "pull MODEL",
		Short: "Pull a model from Docker Hub or HuggingFace to your local environment",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(
					"'docker model pull' requires 1 argument.\n\n" +
						"Usage:  docker model pull MODEL\n\n" +
						"See 'docker model pull --help' for more information",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := ensureStandaloneRunnerAvailable(cmd.Context(), cmd); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner: %w", err)
			}
			return pullModel(cmd, desktopClient, args[0], ignoreRuntimeMemoryCheck)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().BoolVar(&ignoreRuntimeMemoryCheck, "ignore-runtime-memory-check", false, "Do not block pull if estimated runtime memory for model exceeds system resources.")

	return c
}

func pullModel(cmd *cobra.Command, desktopClient *desktop.Client, modelName string, ignoreRuntimeMemoryCheck bool) error {
	// Check if this is an Ollama model
	if model.IsOllamaModel(modelName) {
		return pullOllamaModel(cmd, modelName)
	}

	var progress func(string)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		progress = TUIProgress
	} else {
		progress = RawProgress
	}
	response, progressShown, err := desktopClient.Pull(modelName, ignoreRuntimeMemoryCheck, progress)

	// Add a newline before any output (success or error) if progress was shown.
	if progressShown {
		cmd.Println()
	}

	if err != nil {
		return handleNotRunningError(handleClientError(err, "Failed to pull model"))
	}

	cmd.Println(response)
	return nil
}

func pullOllamaModel(cmd *cobra.Command, modelName string) error {
	// Ensure the Ollama runner is available
	dockerClient, err := desktop.DockerClientForContext(dockerCLI, dockerCLI.CurrentContext())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Check if ollama container is running
	ctrID, _, _, err := standalone.FindOllamaContainer(cmd.Context(), dockerClient)
	if err != nil {
		return fmt.Errorf("failed to find Ollama container: %w", err)
	}
	if ctrID == "" {
		return fmt.Errorf("Ollama runner is not running. Please start it with 'docker model install-runner --ollama' or 'docker model start-runner --ollama'")
	}

	// Strip the ollama.com prefix and pull the model
	strippedModelName := model.StripOllamaPrefix(modelName)
	ollamaClient := ollama.NewClient("http://localhost:" + fmt.Sprintf("%d", standalone.DefaultOllamaPort))
	
	cmd.Printf("Pulling Ollama model %s...\n", strippedModelName)
	
	var progress func(string)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		progress = TUIProgress
	} else {
		progress = RawProgress
	}
	
	err = ollamaClient.Pull(cmd.Context(), strippedModelName, progress)
	if err != nil {
		return fmt.Errorf("failed to pull Ollama model: %w", err)
	}
	
	cmd.Printf("\nSuccessfully pulled %s\n", modelName)
	return nil
}

func TUIProgress(message string) {
	fmt.Print("\r\033[K", message)
}

func RawProgress(message string) {
	fmt.Println(message)
}
