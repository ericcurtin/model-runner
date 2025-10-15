package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// k8sOptions holds options for k8s commands
type k8sOptions struct {
	namespace    string
	storageClass string
	storageSize  string
	valuesFile   string
	kubeconfig   string
	helmRelease  string
	debug        bool
}

func newK8sCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "Kubernetes deployment commands for Docker Model Runner",
		Long:  "Install, uninstall, and manage Docker Model Runner on Kubernetes clusters",
	}

	cmd.AddCommand(
		newK8sInstallCmd(),
		newK8sUninstallCmd(),
	)

	return cmd
}

func newK8sInstallCmd() *cobra.Command {
	opts := &k8sOptions{
		namespace:   "docker-model-runner",
		storageSize: "100Gi",
		helmRelease: "docker-model-runner",
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Docker Model Runner on Kubernetes",
		Long:  "Deploy Docker Model Runner to a Kubernetes cluster using Helm or static manifests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sInstall(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", opts.namespace, "Kubernetes namespace")
	cmd.Flags().StringVarP(&opts.storageClass, "storage-class", "c", "", "Storage class for PVC")
	cmd.Flags().StringVarP(&opts.storageSize, "storage-size", "z", opts.storageSize, "Storage size for models")
	cmd.Flags().StringVarP(&opts.valuesFile, "values", "f", "", "Path to Helm values file")
	cmd.Flags().StringVar(&opts.kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	cmd.Flags().StringVarP(&opts.helmRelease, "release", "r", opts.helmRelease, "Helm release name")
	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "Enable debug output")

	return cmd
}

func newK8sUninstallCmd() *cobra.Command {
	opts := &k8sOptions{
		namespace:   "docker-model-runner",
		helmRelease: "docker-model-runner",
	}

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Docker Model Runner from Kubernetes",
		Long:  "Remove Docker Model Runner deployment from a Kubernetes cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sUninstall(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", opts.namespace, "Kubernetes namespace")
	cmd.Flags().StringVar(&opts.kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	cmd.Flags().StringVarP(&opts.helmRelease, "release", "r", opts.helmRelease, "Helm release name")
	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "Enable debug output")

	return cmd
}

func runK8sInstall(ctx context.Context, opts *k8sOptions) error {
	// Check dependencies
	if err := checkCommand("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found: %w", err)
	}
	if err := checkCommand("helm"); err != nil {
		return fmt.Errorf("helm not found: %w", err)
	}

	// Check cluster connectivity
	logInfo("🔍 Checking cluster connectivity...")
	if err := runKubectl(opts, "cluster-info"); err != nil {
		return fmt.Errorf("cannot connect to cluster: %w", err)
	}
	logSuccess("✅ Connected to cluster")

	// Create namespace
	logInfo(fmt.Sprintf("📦 Creating namespace %s...", opts.namespace))
	if err := runKubectl(opts, "create", "namespace", opts.namespace, "--dry-run=client", "-o", "yaml"); err == nil {
		if err := runKubectl(opts, "apply", "-f", "-"); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}
	logSuccess("Namespace ready")

	// Get the chart directory
	chartDir, err := findChartDirectory()
	if err != nil {
		return fmt.Errorf("failed to find chart directory: %w", err)
	}

	logInfo("🛠️ Installing Docker Model Runner with Helm...")

	// Build helm command
	helmArgs := []string{"upgrade", "--install", opts.helmRelease, chartDir}
	helmArgs = append(helmArgs, "--namespace", opts.namespace)
	helmArgs = append(helmArgs, "--create-namespace")

	if opts.storageClass != "" {
		helmArgs = append(helmArgs, "--set", fmt.Sprintf("storage.storageClass=%s", opts.storageClass))
	}
	if opts.storageSize != "" {
		helmArgs = append(helmArgs, "--set", fmt.Sprintf("storage.size=%s", opts.storageSize))
	}
	if opts.valuesFile != "" {
		helmArgs = append(helmArgs, "-f", opts.valuesFile)
	}
	if opts.debug {
		helmArgs = append(helmArgs, "--debug")
	}

	if opts.kubeconfig != "" {
		helmArgs = append(helmArgs, "--kubeconfig", opts.kubeconfig)
	}

	if err := runHelm(helmArgs...); err != nil {
		return fmt.Errorf("helm install failed: %w", err)
	}

	logSuccess("✅ Docker Model Runner installed successfully")

	// Print post-install instructions
	logInfo("\n📝 Installation complete! Next steps:")
	logInfo(fmt.Sprintf("   1. Wait for deployment: kubectl wait --for=condition=Available deployment/%s -n %s --timeout=5m", opts.helmRelease, opts.namespace))
	logInfo(fmt.Sprintf("   2. Set up port-forward: kubectl port-forward -n %s service/%s-nodeport 31245:80", opts.namespace, opts.helmRelease))
	logInfo("   3. Test the model runner: MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest")

	return nil
}

func runK8sUninstall(ctx context.Context, opts *k8sOptions) error {
	// Check dependencies
	if err := checkCommand("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found: %w", err)
	}
	if err := checkCommand("helm"); err != nil {
		return fmt.Errorf("helm not found: %w", err)
	}

	logInfo("🧹 Uninstalling Docker Model Runner...")

	// Uninstall helm release
	helmArgs := []string{"uninstall", opts.helmRelease, "--namespace", opts.namespace}
	if opts.kubeconfig != "" {
		helmArgs = append(helmArgs, "--kubeconfig", opts.kubeconfig)
	}

	if err := runHelm(helmArgs...); err != nil {
		logError(fmt.Sprintf("Warning: helm uninstall failed: %v", err))
	}

	// Optionally delete namespace
	logInfo(fmt.Sprintf("🗑️  Deleting namespace %s...", opts.namespace))
	if err := runKubectl(opts, "delete", "namespace", opts.namespace, "--ignore-not-found"); err != nil {
		logError(fmt.Sprintf("Warning: namespace deletion failed: %v", err))
	}

	logSuccess("✅ Docker Model Runner uninstalled successfully")

	return nil
}

// Helper functions

func checkCommand(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("command %s not found in PATH", name)
	}
	return nil
}

func runKubectl(opts *k8sOptions, args ...string) error {
	cmdArgs := []string{}
	if opts.kubeconfig != "" {
		cmdArgs = append(cmdArgs, "--kubeconfig", opts.kubeconfig)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("kubectl", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runHelm(args ...string) error {
	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findChartDirectory() (string, error) {
	// Try to find the chart directory relative to the current working directory
	// or from the embedded charts
	possiblePaths := []string{
		"charts/docker-model-runner",
		"../charts/docker-model-runner",
		"../../charts/docker-model-runner",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, "Chart.yaml")); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("chart directory not found. Please run from repository root or specify chart path")
}

// Logging helpers with colors
func logInfo(msg string) {
	fmt.Printf("\033[34mℹ️  %s\033[0m\n", msg)
}

func logSuccess(msg string) {
	fmt.Printf("\033[32m✅ %s\033[0m\n", msg)
}

func logError(msg string) {
	fmt.Fprintf(os.Stderr, "\033[31m❌ %s\033[0m\n", msg)
}
