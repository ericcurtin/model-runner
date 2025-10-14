package commands

import (
	"fmt"
	"github.com/docker/model-runner/cmd/cli/pkg/types"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/docker/model-runner/cmd/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

func newStopRunner() *cobra.Command {
	var models bool
	c := &cobra.Command{
		Use:   "stop-runner",
		Short: "Stop Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure that we're running in a supported model runner context.
			if kind := modelRunner.EngineKind(); kind == types.ModelRunnerEngineKindDesktop {
				cmd.Println("Standalone stop not supported with Docker Desktop")
				cmd.Println("Use `docker desktop disable model-runner` instead")
				return nil
			} else if kind == types.ModelRunnerEngineKindMobyManual {
				cmd.Println("Standalone stop not supported with MODEL_RUNNER_HOST set")
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

			// Skip image removal for stop-runner (unlike uninstall-runner)

			// Remove model storage, if requested.
			if models {
				if err := standalone.PruneModelStorageVolumes(cmd.Context(), dockerClient, cmd); err != nil {
					return fmt.Errorf("unable to remove model storage volume(s): %w", err)
				}
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().BoolVar(&models, "models", false, "Remove model storage volume")
	return c
}
