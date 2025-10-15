package commands

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/spf13/cobra"
)

//go:embed k8s/*
var k8sFS embed.FS

func newK8sCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "Kubernetes deployment utilities for Docker Model Runner",
		Long: `Deploy and manage Docker Model Runner on Kubernetes clusters.

This command provides utilities for deploying Docker Model Runner to Kubernetes
using either Helm charts or static Kubernetes manifests.`,
		ValidArgsFunction: completion.NoComplete,
	}

	cmd.AddCommand(
		newK8sManifestCmd(),
		newK8sHelmInfoCmd(),
	)

	return cmd
}

func newK8sManifestCmd() *cobra.Command {
	var outputDir string
	var manifestType string

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Generate Kubernetes manifests",
		Long: `Generate Kubernetes manifests for Docker Model Runner deployment.

Available manifest types:
  - default: Basic deployment (ClusterIP service)
  - desktop: Deployment with NodePort for Docker Desktop
  - gpu: Deployment with GPU support
  - smollm2: Deployment with pre-pulled smollm2 model
  - eks: Deployment optimized for AWS EKS
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine which manifest to use
			var manifestFile string
			switch manifestType {
			case "default":
				manifestFile = "k8s/static/docker-model-runner.yaml"
			case "desktop":
				manifestFile = "k8s/static/docker-model-runner-desktop.yaml"
			case "smollm2":
				manifestFile = "k8s/static/docker-model-runner-smollm2.yaml"
			case "eks":
				manifestFile = "k8s/static/docker-model-runner-eks.yaml"
			default:
				return fmt.Errorf("unknown manifest type: %s", manifestType)
			}

			// Read the embedded manifest
			manifestContent, err := k8sFS.ReadFile(manifestFile)
			if err != nil {
				return fmt.Errorf("failed to read manifest: %w", err)
			}

			// Determine output location
			if outputDir == "" {
				// Print to stdout
				fmt.Println(string(manifestContent))
				return nil
			}

			// Create output directory if it doesn't exist
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Write to file
			outputFile := filepath.Join(outputDir, fmt.Sprintf("docker-model-runner-%s.yaml", manifestType))
			if err := os.WriteFile(outputFile, manifestContent, 0644); err != nil {
				return fmt.Errorf("failed to write manifest: %w", err)
			}

			cmd.Printf("Manifest written to: %s\n", outputFile)
			cmd.Println("\nTo deploy, run:")
			cmd.Printf("  kubectl apply -f %s\n", outputFile)
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory for manifest (default: stdout)")
	cmd.Flags().StringVarP(&manifestType, "type", "t", "default", "Manifest type (default|desktop|smollm2|eks)")

	return cmd
}

func newK8sHelmInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm-info",
		Short: "Display Helm chart information and usage",
		Long:  `Display information about the Docker Model Runner Helm chart and how to use it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := `Docker Model Runner Helm Chart
================================

The Helm chart for Docker Model Runner is located at:
  charts/docker-model-runner

Quick Start
-----------

1. Install with default values:
   helm install docker-model-runner charts/docker-model-runner

2. Install with custom storage:
   helm install docker-model-runner charts/docker-model-runner \
     --set storage.size=200Gi \
     --set storage.storageClass=gp3

3. Install with GPU support:
   helm install docker-model-runner charts/docker-model-runner \
     --set gpu.enabled=true \
     --set gpu.vendor=nvidia \
     --set gpu.count=1

4. Install with model pre-pulling:
   helm install docker-model-runner charts/docker-model-runner \
     --set modelInit.enabled=true \
     --set modelInit.models[0]=ai/smollm2:latest

Key Configuration Options
-------------------------

storage.size         - Size of persistent volume (default: 100Gi)
storage.storageClass - Storage class for cloud providers (default: "")
gpu.enabled          - Enable GPU scheduling (default: false)
gpu.vendor           - GPU vendor: nvidia or amd (default: nvidia)
gpu.count            - Number of GPUs to request (default: 1)
modelInit.enabled    - Enable model pre-pulling (default: false)
modelInit.models     - List of models to pre-pull (default: [])
nodePort.enabled     - Enable NodePort service (default: false)
nodePort.port        - NodePort port number (default: 31245)

For More Information
--------------------

View the full README:
  cat charts/docker-model-runner/README.md

View available values:
  helm show values charts/docker-model-runner

Render manifests locally:
  helm template docker-model-runner charts/docker-model-runner

For complete documentation, visit:
  https://docs.docker.com/ai/model-runner/kubernetes/
`
			cmd.Println(info)
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}

	return cmd
}
