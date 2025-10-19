package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/model-runner/pkg/distribution/builder"
	"github.com/docker/model-runner/pkg/distribution/packaging"
	"github.com/docker/model-runner/pkg/distribution/registry"
	"github.com/docker/model-runner/pkg/distribution/tarball"
	"github.com/docker/model-runner/pkg/distribution/types"
	"github.com/docker/model-runner/pkg/inference/models"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
)

func newPackagedCmd() *cobra.Command {
	var opts packageOptions

	c := &cobra.Command{
		Use:   "package (--gguf <path> | --safetensors-dir <path>) [--license <path>...] [--context-size <tokens>] [--push] MODEL",
		Short: "Package a GGUF file or Safetensors directory into a Docker model OCI artifact.",
		Long: "Package a GGUF file or Safetensors directory into a Docker model OCI artifact, with optional licenses. The package is sent to the model-runner, unless --push is specified.\n" +
			"When packaging a sharded GGUF model, --gguf should point to the first shard. All shard files should be siblings and should include the index in the file name (e.g. model-00001-of-00015.gguf).\n" +
			"When packaging a Safetensors model, --safetensors-dir should point to a directory containing .safetensors files and config files (*.json, merges.txt). All files will be auto-discovered and config files will be packaged into a tar archive.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(
					"'docker model package' requires 1 argument.\n\n"+
						"Usage:  docker model %s\n\n"+
						"See 'docker model package --help' for more information",
					cmd.Use,
				)
			}

			// Validate that either --gguf or --safetensors-dir is provided (mutually exclusive)
			if opts.ggufPath == "" && opts.safetensorsDir == "" {
				return fmt.Errorf(
					"Either --gguf or --safetensors-dir path is required.\n\n" +
						"See 'docker model package --help' for more information",
				)
			}
			if opts.ggufPath != "" && opts.safetensorsDir != "" {
				return fmt.Errorf(
					"Cannot specify both --gguf and --safetensors-dir. Please use only one format.\n\n" +
						"See 'docker model package --help' for more information",
				)
			}

			// Validate GGUF path if provided
			if opts.ggufPath != "" {
				if !filepath.IsAbs(opts.ggufPath) {
					return fmt.Errorf(
						"GGUF path must be absolute.\n\n" +
							"See 'docker model package --help' for more information",
					)
				}
				opts.ggufPath = filepath.Clean(opts.ggufPath)
			}

			// Validate safetensors directory if provided
			if opts.safetensorsDir != "" {
				if !filepath.IsAbs(opts.safetensorsDir) {
					return fmt.Errorf(
						"Safetensors directory path must be absolute.\n\n" +
							"See 'docker model package --help' for more information",
					)
				}
				opts.safetensorsDir = filepath.Clean(opts.safetensorsDir)

				// Check if it's a directory
				info, err := os.Stat(opts.safetensorsDir)
				if err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf(
							"Safetensors directory does not exist: %s\n\n"+
								"See 'docker model package --help' for more information",
							opts.safetensorsDir,
						)
					}
					return fmt.Errorf("could not access safetensors directory %q: %w", opts.safetensorsDir, err)
				}
				if !info.IsDir() {
					return fmt.Errorf(
						"Safetensors path must be a directory: %s\n\n"+
							"See 'docker model package --help' for more information",
						opts.safetensorsDir,
					)
				}
			}

			for i, l := range opts.licensePaths {
				if !filepath.IsAbs(l) {
					return fmt.Errorf(
						"license path must be absolute.\n\n" +
							"See 'docker model package --help' for more information",
					)
				}
				opts.licensePaths[i] = filepath.Clean(l)
			}

			// Validate dir-tar paths are relative (not absolute)
			for _, dirPath := range opts.dirTarPaths {
				if filepath.IsAbs(dirPath) {
					return fmt.Errorf(
						"dir-tar path must be relative, got absolute path: %s\n\n"+
							"See 'docker model package --help' for more information",
						dirPath,
					)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.tag = args[0]
			if err := packageModel(cmd, opts); err != nil {
				cmd.PrintErrln("Failed to package model")
				return fmt.Errorf("package model: %w", err)
			}
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVar(&opts.ggufPath, "gguf", "", "absolute path to gguf file")
	c.Flags().StringVar(&opts.safetensorsDir, "safetensors-dir", "", "absolute path to directory containing safetensors files and config")
	c.Flags().StringVar(&opts.chatTemplatePath, "chat-template", "", "absolute path to chat template file (must be Jinja format)")
	c.Flags().StringArrayVarP(&opts.licensePaths, "license", "l", nil, "absolute path to a license file")
	c.Flags().StringArrayVar(&opts.dirTarPaths, "dir-tar", nil, "relative path to directory to package as tar (can be specified multiple times)")
	c.Flags().BoolVar(&opts.push, "push", false, "push to registry (if not set, the model is loaded into the Model Runner content store)")
	c.Flags().Uint64Var(&opts.contextSize, "context-size", 0, "context size in tokens")
	return c
}

type packageOptions struct {
	chatTemplatePath string
	contextSize      uint64
	ggufPath         string
	safetensorsDir   string
	licensePaths     []string
	dirTarPaths      []string
	push             bool
	tag              string
}

func packageModel(cmd *cobra.Command, opts packageOptions) error {
	var (
		target builder.Target
		err    error
	)
	if opts.push {
		target, err = registry.NewClient(
			registry.WithUserAgent("docker-model-cli/" + desktop.Version),
		).NewTarget(opts.tag)
	} else {
		target, err = newModelRunnerTarget(desktopClient, opts.tag)
	}
	if err != nil {
		return err
	}

	// Create package builder based on model format
	var pkg *builder.Builder
	if opts.ggufPath != "" {
		cmd.PrintErrf("Adding GGUF file from %q\n", opts.ggufPath)
		pkg, err = builder.FromGGUF(opts.ggufPath)
		if err != nil {
			return fmt.Errorf("add gguf file: %w", err)
		}
	} else {
		// Safetensors model from directory
		cmd.PrintErrf("Scanning directory %q for safetensors model...\n", opts.safetensorsDir)
		safetensorsPaths, tempConfigArchive, err := packaging.PackageFromDirectory(opts.safetensorsDir)
		if err != nil {
			return fmt.Errorf("scan safetensors directory: %w", err)
		}

		// Clean up temp config archive when done
		if tempConfigArchive != "" {
			defer os.Remove(tempConfigArchive)
		}

		cmd.PrintErrf("Found %d safetensors file(s)\n", len(safetensorsPaths))
		pkg, err = builder.FromSafetensors(safetensorsPaths)
		if err != nil {
			return fmt.Errorf("create safetensors model: %w", err)
		}

		// Add config archive if it was created
		if tempConfigArchive != "" {
			cmd.PrintErrf("Adding config archive from directory\n")
			pkg, err = pkg.WithConfigArchive(tempConfigArchive)
			if err != nil {
				return fmt.Errorf("add config archive: %w", err)
			}
		}
	}

	// Set context size
	if opts.contextSize > 0 {
		cmd.PrintErrf("Setting context size %d\n", opts.contextSize)
		pkg = pkg.WithContextSize(opts.contextSize)
	}

	// Add license files
	for _, path := range opts.licensePaths {
		cmd.PrintErrf("Adding license file from %q\n", path)
		pkg, err = pkg.WithLicense(path)
		if err != nil {
			return fmt.Errorf("add license file: %w", err)
		}
	}

	if opts.chatTemplatePath != "" {
		cmd.PrintErrf("Adding chat template file from %q\n", opts.chatTemplatePath)
		if pkg, err = pkg.WithChatTemplateFile(opts.chatTemplatePath); err != nil {
			return fmt.Errorf("add chat template file from path %q: %w", opts.chatTemplatePath, err)
		}
	}

	// Process directory tar archives
	if len(opts.dirTarPaths) > 0 {
		// Determine base directory for resolving relative paths
		var baseDir string
		if opts.safetensorsDir != "" {
			baseDir = opts.safetensorsDir
		} else {
			// For GGUF, use the directory containing the GGUF file
			baseDir = filepath.Dir(opts.ggufPath)
		}

		processor := packaging.NewDirTarProcessor(opts.dirTarPaths, baseDir)
		tarPaths, cleanup, err := processor.Process()
		if err != nil {
			return err
		}
		defer cleanup()

		for _, tarPath := range tarPaths {
			pkg, err = pkg.WithDirTar(tarPath)
			if err != nil {
				return fmt.Errorf("add directory tar: %w", err)
			}
		}
	}

	if opts.push {
		cmd.PrintErrln("Pushing model to registry...")
	} else {
		cmd.PrintErrln("Loading model to Model Runner...")
	}
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		defer pw.Close()
		done <- pkg.Build(cmd.Context(), target, pw)
	}()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		progressLine := scanner.Text()
		if progressLine == "" {
			continue
		}

		// Parse the progress message
		var progressMsg desktop.ProgressMessage
		if err := json.Unmarshal([]byte(html.UnescapeString(progressLine)), &progressMsg); err != nil {
			cmd.PrintErrln("Error displaying progress:", err)
		}

		// Print progress messages
		TUIProgress(progressMsg.Message)
	}
	cmd.PrintErrln("") // newline after progress

	if err := scanner.Err(); err != nil {
		cmd.PrintErrln("Error streaming progress:", err)
	}
	if err := <-done; err != nil {
		if opts.push {
			return fmt.Errorf("failed to save packaged model: %w", err)
		}
		return fmt.Errorf("failed to load packaged model: %w", err)
	}

	if opts.push {
		cmd.PrintErrln("Model pushed successfully")
	} else {
		cmd.PrintErrln("Model loaded successfully")
	}
	return nil
}

// modelRunnerTarget loads model to Docker Model Runner via models/load endpoint
type modelRunnerTarget struct {
	client *desktop.Client
	tag    name.Tag
}

func newModelRunnerTarget(client *desktop.Client, tag string) (*modelRunnerTarget, error) {
	target := &modelRunnerTarget{
		client: client,
	}
	if tag != "" {
		var err error
		// Normalize the tag to add default namespace (ai/) and tag (:latest) if missing
		normalizedTag := models.NormalizeModelName(tag)
		target.tag, err = name.NewTag(normalizedTag)
		if err != nil {
			return nil, fmt.Errorf("invalid tag: %w", err)
		}
	}
	return target, nil
}

func (t *modelRunnerTarget) Write(ctx context.Context, mdl types.ModelArtifact, progressWriter io.Writer) error {
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		target, err := tarball.NewTarget(pw)
		if err != nil {
			errCh <- err
			return
		}
		errCh <- target.Write(ctx, mdl, progressWriter)
	}()

	loadErr := t.client.LoadModel(ctx, pr)
	writeErr := <-errCh

	if loadErr != nil {
		return fmt.Errorf("loading model archive: %w", loadErr)
	}
	if writeErr != nil {
		return fmt.Errorf("writing model archive: %w", writeErr)
	}
	id, err := mdl.ID()
	if err != nil {
		return fmt.Errorf("get model ID: %w", err)
	}
	if t.tag.String() != "" {
		if err := desktopClient.Tag(id, parseRepo(t.tag), t.tag.TagStr()); err != nil {
			return fmt.Errorf("tag model: %w", err)
		}
	}
	return nil
}
