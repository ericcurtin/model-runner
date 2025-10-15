package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/spf13/cobra"
)

func newK8sCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "k8s",
		Short: "Kubernetes deployment commands for Docker Model Runner",
		Long: `Kubernetes deployment commands for Docker Model Runner.

This command helps you deploy Docker Model Runner to Kubernetes clusters.`,
	}

	c.AddCommand(
		newK8sApplyCmd(),
		newK8sDeleteCmd(),
		newK8sManifestsCmd(),
	)

	return c
}

func newK8sApplyCmd() *cobra.Command {
	var namespace string

	c := &cobra.Command{
		Use:   "apply",
		Short: "Apply Kubernetes manifests to deploy Docker Model Runner",
		Long: `Apply Kubernetes manifests to deploy Docker Model Runner to a cluster.

This command applies the necessary Kubernetes resources including:
- ServiceAccount and RBAC permissions
- Deployment
- Service

Prerequisites:
- kubectl must be installed and configured
- Access to a Kubernetes cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestsPath, err := getManifestsPath()
			if err != nil {
				return err
			}

			cmd.Println("Applying Docker Model Runner manifests to Kubernetes...")

			kubectlArgs := []string{"apply", "-k", manifestsPath}
			if namespace != "" {
				kubectlArgs = append(kubectlArgs, "-n", namespace)
			}

			cmd.Printf("Running: kubectl %v\n", kubectlArgs)
			cmd.Println("\nTo apply manually, run:")
			cmd.Printf("  kubectl apply -k %s", manifestsPath)
			if namespace != "" {
				cmd.Printf(" -n %s", namespace)
			}
			cmd.Println()

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (default: current context namespace)")

	return c
}

func newK8sDeleteCmd() *cobra.Command {
	var namespace string

	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete Docker Model Runner from Kubernetes",
		Long:  `Delete the Docker Model Runner deployment and all associated resources from Kubernetes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestsPath, err := getManifestsPath()
			if err != nil {
				return err
			}

			cmd.Println("Deleting Docker Model Runner from Kubernetes...")

			kubectlArgs := []string{"delete", "-k", manifestsPath}
			if namespace != "" {
				kubectlArgs = append(kubectlArgs, "-n", namespace)
			}

			cmd.Printf("Running: kubectl %v\n", kubectlArgs)
			cmd.Println("\nTo delete manually, run:")
			cmd.Printf("  kubectl delete -k %s", manifestsPath)
			if namespace != "" {
				cmd.Printf(" -n %s", namespace)
			}
			cmd.Println()

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (default: current context namespace)")

	return c
}

func newK8sManifestsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "manifests",
		Short: "Show the path to Kubernetes manifests",
		Long:  `Display the path to the Kubernetes manifests directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestsPath, err := getManifestsPath()
			if err != nil {
				return err
			}

			cmd.Println("Kubernetes manifests location:")
			cmd.Println(manifestsPath)
			cmd.Println()
			cmd.Println("Usage:")
			cmd.Println("  kubectl apply -k", manifestsPath)
			cmd.Println()
			cmd.Println("Files:")
			files := []string{
				"sa.yaml          - ServiceAccount",
				"rbac.yaml        - Role and RoleBinding",
				"deployment.yaml  - Deployment specification",
				"service.yaml     - Service",
				"kustomization.yaml - Kustomize configuration",
			}
			for _, f := range files {
				cmd.Println(" ", f)
			}

			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}

	return c
}

// getManifestsPath returns the path to the k8s manifests directory
func getManifestsPath() (string, error) {
	// Try to find manifests relative to the executable or in known locations
	// First try: relative to current directory
	if _, err := os.Stat("k8s"); err == nil {
		abs, err := filepath.Abs("k8s")
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		return abs, nil
	}

	// Second try: relative to executable
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exe)
	k8sPath := filepath.Join(exeDir, "k8s")
	if _, err := os.Stat(k8sPath); err == nil {
		return k8sPath, nil
	}

	// Third try: in the repository root (for development)
	// Go up from exe directory to find k8s
	for dir := exeDir; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		k8sPath := filepath.Join(dir, "k8s")
		if _, err := os.Stat(k8sPath); err == nil {
			return k8sPath, nil
		}
	}

	return "", fmt.Errorf("kubernetes manifests directory not found. Please ensure 'k8s' directory exists")
}
