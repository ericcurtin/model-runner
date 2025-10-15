package commands

import (
	"fmt"
	"github.com/docker/model-runner/cmd/cli/pkg/types"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
	gpupkg "github.com/docker/model-runner/cmd/cli/pkg/gpu"
	"github.com/docker/model-runner/cmd/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

func newReinstallRunner() *cobra.Command {
	var port uint16
	var host string
	var gpuMode string
	var doNotTrack bool
	c := &cobra.Command{
		Use:   "reinstall-runner",
		Short: "Reinstall Docker Model Runner (Docker Engine only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure that we're running in a supported model runner context.
			engineKind := modelRunner.EngineKind()
			if engineKind == types.ModelRunnerEngineKindDesktop {
				cmd.Println("Standalone reinstallation not supported with Docker Desktop")
				cmd.Println("Use `docker desktop disable model-runner` and `docker desktop enable model-runner` instead")
				return nil
			} else if engineKind == types.ModelRunnerEngineKindMobyManual {
				cmd.Println("Standalone reinstallation not supported with MODEL_RUNNER_HOST set")
				return nil
			}

			if port == 0 {
				// Use "0" as a sentinel default flag value so it's not displayed automatically.
				// The default values are written in the usage string.
				// Hence, the user currently won't be able to set the port to 0 in order to get a random available port.
				port = standalone.DefaultControllerPortMoby
			}
			// HACK: If we're in a Cloud context, then we need to use a
			// different default port because it conflicts with Docker Desktop's
			// default model runner host-side port. Unfortunately we can't make
			// the port flag default dynamic (at least not easily) because of
			// when context detection happens. So assume that a default value
			// indicates that we want the Cloud default port. This is less
			// problematic in Cloud since the UX there is mostly invisible.
			if engineKind == types.ModelRunnerEngineKindCloud &&
				port == standalone.DefaultControllerPortMoby {
				port = standalone.DefaultControllerPortCloud
			}

			// Set the appropriate environment.
			environment := "moby"
			if engineKind == types.ModelRunnerEngineKindCloud {
				environment = "cloud"
			}

			// Create a Docker client for the active context.
			dockerClient, err := desktop.DockerClientForContext(dockerCLI, dockerCLI.CurrentContext())
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}

			// Remove any model runner containers (but keep models and images).
			if err := standalone.PruneControllerContainers(cmd.Context(), dockerClient, false, cmd); err != nil {
				return fmt.Errorf("unable to remove model runner container(s): %w", err)
			}

			// Determine GPU support.
			var gpu gpupkg.GPUSupport
			if gpuMode == "auto" {
				gpu, err = gpupkg.ProbeGPUSupport(cmd.Context(), dockerClient)
				if err != nil {
					return fmt.Errorf("unable to probe GPU support: %w", err)
				}
			} else if gpuMode == "cuda" {
				gpu = gpupkg.GPUSupportCUDA
			} else if gpuMode != "none" {
				return fmt.Errorf("unknown GPU specification: %q", gpuMode)
			}

			// Ensure that we have an up-to-date copy of the image.
			if err := standalone.EnsureControllerImage(cmd.Context(), dockerClient, gpu, cmd); err != nil {
				return fmt.Errorf("unable to pull latest standalone model runner image: %w", err)
			}

			// Ensure that we have a model storage volume.
			modelStorageVolume, err := standalone.EnsureModelStorageVolume(cmd.Context(), dockerClient, cmd)
			if err != nil {
				return fmt.Errorf("unable to initialize standalone model storage: %w", err)
			}

			// Create the model runner container.
			if err := standalone.CreateControllerContainer(cmd.Context(), dockerClient, port, host, environment, doNotTrack, gpu, modelStorageVolume, cmd, engineKind); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner container: %w", err)
			}

			// Poll until we get a response from the model runner.
			return waitForStandaloneRunnerAfterInstall(cmd.Context())
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().Uint16Var(&port, "port", 0,
		"Docker container port for Docker Model Runner (default: 12434 for Docker CE, 12435 for Cloud mode)")
	c.Flags().StringVar(&host, "host", "127.0.0.1", "Host address to bind Docker Model Runner")
	c.Flags().StringVar(&gpuMode, "gpu", "auto", "Specify GPU support (none|auto|cuda)")
	c.Flags().BoolVar(&doNotTrack, "do-not-track", false, "Do not track models usage in Docker Model Runner")
	return c
}
