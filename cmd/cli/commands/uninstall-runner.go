package commands

import (
	"fmt"
	"github.com/docker/model-runner/cmd/cli/pkg/types"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/docker/model-runner/cmd/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

// cleanupOptions holds common configuration for uninstall/stop commands
type cleanupOptions struct {
	models       bool
	removeImages bool
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

func newUninstallRunner() *cobra.Command {
	var models, images bool
	c := &cobra.Command{
		Use:   "uninstall-runner",
		Short: "Uninstall Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstallOrStop(cmd, cleanupOptions{
				models:       models,
				removeImages: images,
			})
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().BoolVar(&models, "models", false, "Remove model storage volume")
	c.Flags().BoolVar(&images, "images", false, "Remove "+standalone.ControllerImage+" images")
	return c
}
