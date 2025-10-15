package commands

import (
	"fmt"

	"github.com/docker/docker/client"
	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/docker/model-runner/cmd/cli/pkg/standalone"
	"github.com/docker/model-runner/cmd/cli/pkg/types"
	"github.com/spf13/cobra"
)

// cleanupOptions holds common configuration for uninstall/stop commands
type cleanupOptions struct {
	models       bool
	removeImages bool
	ollama       bool
}

// runUninstallOrStop is shared logic for uninstall-runner and stop-runner commands
func runUninstallOrStop(cmd *cobra.Command, opts cleanupOptions) error {
	// Ensure that we're running in a supported model runner context.
	if kind := modelRunner.EngineKind(); kind == types.ModelRunnerEngineKindDesktop {
		// TODO: We may eventually want to auto-forward this to
		// docker desktop disable model-runner, but we should first
		// make install-runner forward in the same way.
		cmd.Println("Standalone uninstallation not supported with Docker Desktop")
		cmd.Println("Use `docker desktop disable model-runner` instead")
		return nil
	} else if kind == types.ModelRunnerEngineKindMobyManual {
		cmd.Println("Standalone uninstallation not supported with MODEL_RUNNER_HOST set")
		return nil
	}

	// Create a Docker client for the active context.
	dockerClient, err := desktop.DockerClientForContext(dockerCLI, dockerCLI.CurrentContext())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Handle Ollama runner cleanup
	if opts.ollama {
		return runUninstallOrStopOllama(cmd, dockerClient, opts)
	}

	// Remove any model runner containers.
	if err := standalone.PruneControllerContainers(cmd.Context(), dockerClient, false, cmd); err != nil {
		return fmt.Errorf("unable to remove model runner container(s): %w", err)
	}

	// Remove model runner images, if requested.
	if opts.removeImages {
		if err := standalone.PruneControllerImages(cmd.Context(), dockerClient, cmd); err != nil {
			return fmt.Errorf("unable to remove model runner image(s): %w", err)
		}
	}

	// Remove model storage, if requested.
	if opts.models {
		if err := standalone.PruneModelStorageVolumes(cmd.Context(), dockerClient, cmd); err != nil {
			return fmt.Errorf("unable to remove model storage volume(s): %w", err)
		}
	}

	return nil
}

// runUninstallOrStopOllama handles cleanup of the Ollama runner
func runUninstallOrStopOllama(cmd *cobra.Command, dockerClient *client.Client, opts cleanupOptions) error {
	// Remove any ollama runner containers.
	if err := standalone.PruneOllamaContainers(cmd.Context(), dockerClient, false, cmd); err != nil {
		return fmt.Errorf("unable to remove ollama runner container(s): %w", err)
	}

	// Remove ollama storage, if requested.
	if opts.models {
		if err := standalone.PruneOllamaStorageVolumes(cmd.Context(), dockerClient, cmd); err != nil {
			return fmt.Errorf("unable to remove ollama storage volume(s): %w", err)
		}
	}

	return nil
}

func newUninstallRunner() *cobra.Command {
	var models, images, ollama bool
	c := &cobra.Command{
		Use:   "uninstall-runner",
		Short: "Uninstall Docker Model Runner (Docker Engine only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstallOrStop(cmd, cleanupOptions{
				models:       models,
				removeImages: images,
				ollama:       ollama,
			})
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().BoolVar(&models, "models", false, "Remove model storage volume")
	c.Flags().BoolVar(&images, "images", false, "Remove "+standalone.ControllerImage+" images")
	c.Flags().BoolVar(&ollama, "ollama", false, "Uninstall Ollama runner instead of Docker Model Runner")
	return c
}
