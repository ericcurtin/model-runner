package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/commands/formatter"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/docker/model-runner/cmd/cli/pkg/ollama"
	"github.com/docker/model-runner/cmd/cli/pkg/standalone"
	disttypes "github.com/docker/model-runner/pkg/distribution/types"
	dmrm "github.com/docker/model-runner/pkg/inference/models"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// listOllamaModels attempts to list models from the Ollama runner.
// Returns an empty list (no error) if the Ollama runner is not available.
func listOllamaModels() ([]ollama.Model, error) {
	// Check if ollama container is running
	dockerClient, err := desktop.DockerClientForContext(dockerCLI, dockerCLI.CurrentContext())
	if err != nil {
		return nil, nil // Silently skip if we can't get a docker client
	}

	ctx := context.Background()
	ctrID, _, _, err := standalone.FindOllamaContainer(ctx, dockerClient)
	if err != nil || ctrID == "" {
		return nil, nil // Ollama runner not available, skip silently
	}

	// List models from Ollama
	ollamaClient := ollama.NewClient("http://localhost:" + fmt.Sprintf("%d", standalone.DefaultOllamaPort))
	models, err := ollamaClient.List(ctx)
	if err != nil {
		return nil, nil // Silently skip on error
	}

	return models, nil
}

func newListCmd() *cobra.Command {
	var jsonFormat, openai, quiet bool
	var backend string
	c := &cobra.Command{
		Use:     "list [OPTIONS]",
		Aliases: []string{"ls"},
		Short:   "List the models pulled to your local environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate backend if specified
			if backend != "" {
				if err := validateBackend(backend); err != nil {
					return err
				}
			}

			if (backend == "openai" || openai) && quiet {
				return fmt.Errorf("--quiet flag cannot be used with --openai flag or OpenAI backend")
			}

			// Validate API key for OpenAI backend
			apiKey, err := ensureAPIKey(backend)
			if err != nil {
				return err
			}

			// If we're doing an automatic install, only show the installation
			// status if it won't corrupt machine-readable output.
			var standaloneInstallPrinter standalone.StatusPrinter
			if !jsonFormat && !openai && !quiet && backend == "" {
				standaloneInstallPrinter = cmd
			}
			if _, err := ensureStandaloneRunnerAvailable(cmd.Context(), standaloneInstallPrinter); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner: %w", err)
			}
			var modelFilter string
			if len(args) > 0 {
				modelFilter = args[0]
			}
			models, err := listModels(openai, backend, desktopClient, quiet, jsonFormat, apiKey, modelFilter)
			if err != nil {
				return err
			}
			cmd.Print(models)
			return nil
		},
		ValidArgsFunction: completion.ModelNamesAndTags(getDesktopClient, 1),
	}
	c.Flags().BoolVar(&jsonFormat, "json", false, "List models in a JSON format")
	c.Flags().BoolVar(&openai, "openai", false, "List models in an OpenAI format")
	c.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only show model IDs")
	c.Flags().StringVar(&backend, "backend", "", fmt.Sprintf("Specify the backend to use (%s)", ValidBackendsKeys()))
	c.Flags().MarkHidden("backend")
	return c
}

func listModels(openai bool, backend string, desktopClient *desktop.Client, quiet bool, jsonFormat bool, apiKey string, modelFilter string) (string, error) {
	if openai || backend == "openai" {
		models, err := desktopClient.ListOpenAI(backend, apiKey)
		if err != nil {
			err = handleClientError(err, "Failed to list models")
			return "", handleNotRunningError(err)
		}
		return formatter.ToStandardJSON(models)
	}
	models, err := desktopClient.List()
	if err != nil {
		err = handleClientError(err, "Failed to list models")
		return "", handleNotRunningError(err)
	}

	// Try to list Ollama models if the Ollama runner is available
	ollamaModels, err := listOllamaModels()
	if err == nil && len(ollamaModels) > 0 {
		// Convert Ollama models to dmrm.Model format and add them to the list
		for _, om := range ollamaModels {
			// Create a synthetic model entry for the Ollama model
			dmrModel := dmrm.Model{
				ID:      "ollama:" + om.Digest,
				Tags:    []string{"ollama.com/" + om.Name},
				Created: 0, // We don't have creation time from Ollama
				Config: disttypes.Config{
					Size: units.HumanSize(float64(om.Size)),
				},
			}
			models = append(models, dmrModel)
		}
	}

	if modelFilter != "" {
		var filteredModels []dmrm.Model
		for _, m := range models {
			hasMatchingTag := false
			for _, tag := range m.Tags {
				modelName, _, _ := strings.Cut(tag, ":")
				if slices.Contains([]string{modelName, tag + ":latest", tag}, modelFilter) {
					hasMatchingTag = true
					break
				}
			}
			if hasMatchingTag {
				filteredModels = append(filteredModels, m)
			}
		}
		models = filteredModels
	}

	if jsonFormat {
		return formatter.ToStandardJSON(models)
	}
	if quiet {
		var modelIDs string
		for _, m := range models {
			if len(m.ID) < 19 {
				fmt.Fprintf(os.Stderr, "invalid image ID for model: %v\n", m)
				continue
			}
			modelIDs += fmt.Sprintf("%s\n", m.ID[7:19])
		}
		return modelIDs, nil
	}
	return prettyPrintModels(models), nil
}

func prettyPrintModels(models []dmrm.Model) string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"MODEL NAME", "PARAMETERS", "QUANTIZATION", "ARCHITECTURE", "MODEL ID", "CREATED", "SIZE"})

	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)

	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT, // MODEL
		tablewriter.ALIGN_LEFT, // PARAMETERS
		tablewriter.ALIGN_LEFT, // QUANTIZATION
		tablewriter.ALIGN_LEFT, // ARCHITECTURE
		tablewriter.ALIGN_LEFT, // MODEL ID
		tablewriter.ALIGN_LEFT, // CREATED
		tablewriter.ALIGN_LEFT, // SIZE
	})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	for _, m := range models {
		if len(m.Tags) == 0 {
			appendRow(table, "<none>", m)
			continue
		}
		for _, tag := range m.Tags {
			appendRow(table, tag, m)
		}
	}

	table.Render()
	return buf.String()
}

func appendRow(table *tablewriter.Table, tag string, model dmrm.Model) {
	if len(model.ID) < 19 {
		fmt.Fprintf(os.Stderr, "invalid model ID for model: %v\n", model)
		return
	}
	table.Append([]string{
		tag,
		model.Config.Parameters,
		model.Config.Quantization,
		model.Config.Architecture,
		model.ID[7:19],
		units.HumanDuration(time.Since(time.Unix(model.Created, 0))) + " ago",
		model.Config.Size,
	})
}
