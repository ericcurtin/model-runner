package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/spf13/cobra"
)

func newK8sCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "k8s",
		Short: "Deploy and manage Docker Model Runner on Kubernetes",
		Long: `Deploy and manage Docker Model Runner on Kubernetes.

This command provides functionality for deploying models to Kubernetes,
similar to llm-d and aibrix, but using Docker Model Runner's native infrastructure.`,
	}

	c.AddCommand(
		newK8sInstallCmd(),
		newK8sUninstallCmd(),
		newK8sStatusCmd(),
	)

	return c
}

func newK8sInstallCmd() *cobra.Command {
	var namespace string
	var storageSize string
	var storageClass string
	var gpuEnabled bool
	var gpuVendor string
	var gpuCount int
	var models []string
	var nodePort bool
	var port int
	var useHelm bool

	c := &cobra.Command{
		Use:   "install",
		Short: "Install Docker Model Runner on Kubernetes",
		Long: `Install Docker Model Runner on Kubernetes cluster.

This command deploys Docker Model Runner to your Kubernetes cluster using either
Helm or plain Kubernetes manifests. It provides functionality for distributed
model serving without requiring external dependencies like llm-d or aibrix.

Examples:
  # Install with default settings
  docker model k8s install

  # Install with GPU support
  docker model k8s install --gpu --gpu-vendor nvidia --gpu-count 1

  # Install with model pre-pulling
  docker model k8s install --models ai/smollm2:latest,ai/llama3.2:latest

  # Install with custom storage
  docker model k8s install --storage-size 200Gi --storage-class fast-ssd

  # Install with NodePort for Docker Desktop
  docker model k8s install --node-port`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if useHelm {
				return installWithHelm(cmd.Context(), namespace, storageSize, storageClass, gpuEnabled, gpuVendor, gpuCount, models, nodePort, port)
			}
			return installWithKubectl(cmd.Context(), namespace, storageSize, storageClass, gpuEnabled, gpuVendor, gpuCount, models, nodePort, port)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	c.Flags().StringVar(&storageSize, "storage-size", "100Gi", "Storage size for model cache")
	c.Flags().StringVar(&storageClass, "storage-class", "", "Storage class for persistent volume")
	c.Flags().BoolVar(&gpuEnabled, "gpu", false, "Enable GPU support")
	c.Flags().StringVar(&gpuVendor, "gpu-vendor", "nvidia", "GPU vendor (nvidia or amd)")
	c.Flags().IntVar(&gpuCount, "gpu-count", 1, "Number of GPUs to request")
	c.Flags().StringSliceVar(&models, "models", []string{}, "Models to pre-pull during initialization")
	c.Flags().BoolVar(&nodePort, "node-port", false, "Enable NodePort service (for Docker Desktop)")
	c.Flags().IntVar(&port, "port", 31245, "NodePort port number")
	c.Flags().BoolVar(&useHelm, "helm", false, "Use Helm for installation (requires Helm to be installed)")

	return c
}

func newK8sUninstallCmd() *cobra.Command {
	var namespace string
	var useHelm bool

	c := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Docker Model Runner from Kubernetes",
		Long: `Uninstall Docker Model Runner from Kubernetes cluster.

This command removes the Docker Model Runner deployment from your Kubernetes cluster.

Examples:
  # Uninstall from default namespace
  docker model k8s uninstall

  # Uninstall from specific namespace
  docker model k8s uninstall --namespace my-namespace`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if useHelm {
				return uninstallWithHelm(cmd.Context(), namespace)
			}
			return uninstallWithKubectl(cmd.Context(), namespace)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	c.Flags().BoolVar(&useHelm, "helm", false, "Use Helm for uninstallation")

	return c
}

func newK8sStatusCmd() *cobra.Command {
	var namespace string

	c := &cobra.Command{
		Use:   "status",
		Short: "Check status of Docker Model Runner on Kubernetes",
		Long: `Check the status of Docker Model Runner deployment on Kubernetes.

This command shows the current state of the Docker Model Runner pods and services.

Examples:
  # Check status in default namespace
  docker model k8s status

  # Check status in specific namespace
  docker model k8s status --namespace my-namespace`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showK8sStatus(cmd.Context(), namespace)
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")

	return c
}

// installWithHelm installs Docker Model Runner using Helm
func installWithHelm(ctx context.Context, namespace, storageSize, storageClass string, gpuEnabled bool, gpuVendor string, gpuCount int, models []string, nodePort bool, port int) error {
	// Find the Helm chart directory
	chartDir, err := findChartDirectory()
	if err != nil {
		return fmt.Errorf("failed to find Helm chart: %w", err)
	}

	fmt.Printf("Installing Docker Model Runner to namespace '%s' using Helm...\n", namespace)

	// Build Helm command
	args := []string{
		"install", "docker-model-runner", chartDir,
		"--namespace", namespace,
		"--create-namespace",
	}

	// Add custom values
	if storageSize != "" && storageSize != "100Gi" {
		args = append(args, "--set", fmt.Sprintf("storage.size=%s", storageSize))
	}
	if storageClass != "" {
		args = append(args, "--set", fmt.Sprintf("storage.storageClass=%s", storageClass))
	}
	if gpuEnabled {
		args = append(args, "--set", "gpu.enabled=true")
		args = append(args, "--set", fmt.Sprintf("gpu.vendor=%s", gpuVendor))
		args = append(args, "--set", fmt.Sprintf("gpu.count=%d", gpuCount))
	}
	if len(models) > 0 {
		args = append(args, "--set", "modelInit.enabled=true")
		for i, model := range models {
			args = append(args, "--set", fmt.Sprintf("modelInit.models[%d]=%s", i, model))
		}
	}
	if nodePort {
		args = append(args, "--set", "nodePort.enabled=true")
		args = append(args, "--set", fmt.Sprintf("nodePort.port=%d", port))
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm install failed: %w", err)
	}

	fmt.Println("\n✓ Installation complete!")
	fmt.Println("\nTo check the status, run:")
	fmt.Printf("  docker model k8s status --namespace %s\n", namespace)

	if nodePort {
		fmt.Printf("\nTo access the service, run:\n")
		fmt.Printf("  MODEL_RUNNER_HOST=http://localhost:%d docker model run ai/smollm2:latest\n", port)
	} else {
		fmt.Printf("\nTo access the service, set up port forwarding:\n")
		fmt.Printf("  kubectl port-forward -n %s deployment/docker-model-runner 31245:12434\n", namespace)
		fmt.Printf("  MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest\n")
	}

	return nil
}

// installWithKubectl installs Docker Model Runner using kubectl and pre-rendered manifests
func installWithKubectl(ctx context.Context, namespace, storageSize, storageClass string, gpuEnabled bool, gpuVendor string, gpuCount int, models []string, nodePort bool, port int) error {
	chartDir, err := findChartDirectory()
	if err != nil {
		return fmt.Errorf("failed to find Helm chart: %w", err)
	}

	fmt.Printf("Installing Docker Model Runner to namespace '%s' using kubectl...\n", namespace)

	// Use helm template to render the manifests
	args := []string{
		"template", "docker-model-runner", chartDir,
		"--namespace", namespace,
	}

	// Add custom values
	if storageSize != "" && storageSize != "100Gi" {
		args = append(args, "--set", fmt.Sprintf("storage.size=%s", storageSize))
	}
	if storageClass != "" {
		args = append(args, "--set", fmt.Sprintf("storage.storageClass=%s", storageClass))
	}
	if gpuEnabled {
		args = append(args, "--set", "gpu.enabled=true")
		args = append(args, "--set", fmt.Sprintf("gpu.vendor=%s", gpuVendor))
		args = append(args, "--set", fmt.Sprintf("gpu.count=%d", gpuCount))
	}
	if len(models) > 0 {
		args = append(args, "--set", "modelInit.enabled=true")
		for i, model := range models {
			args = append(args, "--set", fmt.Sprintf("modelInit.models[%d]=%s", i, model))
		}
	}
	if nodePort {
		args = append(args, "--set", "nodePort.enabled=true")
		args = append(args, "--set", fmt.Sprintf("nodePort.port=%d", port))
	}

	helmCmd := exec.CommandContext(ctx, "helm", args...)
	manifestBytes, err := helmCmd.Output()
	if err != nil {
		return fmt.Errorf("helm template failed: %w", err)
	}

	// Create namespace if it doesn't exist
	createNsCmd := exec.CommandContext(ctx, "kubectl", "create", "namespace", namespace)
	_ = createNsCmd.Run() // Ignore error if namespace already exists

	// Apply the manifests
	kubectlCmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-", "--namespace", namespace)
	kubectlCmd.Stdin = bytes.NewReader(manifestBytes)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr

	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	fmt.Println("\n✓ Installation complete!")
	
	// Wait for deployment to be available
	fmt.Println("\nWaiting for deployment to be ready...")
	waitCmd := exec.CommandContext(ctx, "kubectl", "wait",
		"--for=condition=Available",
		"deployment/docker-model-runner",
		"--timeout=5m",
		"--namespace", namespace)
	waitCmd.Stdout = os.Stdout
	waitCmd.Stderr = os.Stderr
	if err := waitCmd.Run(); err != nil {
		fmt.Println("\nWarning: Deployment may not be ready yet. Check status with:")
		fmt.Printf("  docker model k8s status --namespace %s\n", namespace)
	} else {
		fmt.Println("\n✓ Deployment is ready!")
	}

	fmt.Println("\nTo check the status, run:")
	fmt.Printf("  docker model k8s status --namespace %s\n", namespace)

	if nodePort {
		fmt.Printf("\nTo access the service, run:\n")
		fmt.Printf("  MODEL_RUNNER_HOST=http://localhost:%d docker model run ai/smollm2:latest\n", port)
	} else {
		fmt.Printf("\nTo access the service, set up port forwarding:\n")
		fmt.Printf("  kubectl port-forward -n %s deployment/docker-model-runner 31245:12434\n", namespace)
		fmt.Printf("  MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest\n")
	}

	return nil
}

// uninstallWithHelm uninstalls Docker Model Runner using Helm
func uninstallWithHelm(ctx context.Context, namespace string) error {
	fmt.Printf("Uninstalling Docker Model Runner from namespace '%s' using Helm...\n", namespace)

	cmd := exec.CommandContext(ctx, "helm", "uninstall", "docker-model-runner", "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm uninstall failed: %w", err)
	}

	fmt.Println("✓ Uninstallation complete!")
	return nil
}

// uninstallWithKubectl uninstalls Docker Model Runner using kubectl
func uninstallWithKubectl(ctx context.Context, namespace string) error {
	fmt.Printf("Uninstalling Docker Model Runner from namespace '%s' using kubectl...\n", namespace)

	// Delete the deployment
	cmd := exec.CommandContext(ctx, "kubectl", "delete", "deployment", "docker-model-runner", "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // Ignore error if not found

	// Delete the service
	cmd = exec.CommandContext(ctx, "kubectl", "delete", "service", "docker-model-runner", "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // Ignore error if not found

	// Delete the NodePort service if it exists
	cmd = exec.CommandContext(ctx, "kubectl", "delete", "service", "docker-model-runner-nodeport", "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // Ignore error if not found

	// Delete the configmap if it exists
	cmd = exec.CommandContext(ctx, "kubectl", "delete", "configmap", "docker-model-runner-init", "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // Ignore error if not found

	fmt.Println("✓ Uninstallation complete!")
	return nil
}

// showK8sStatus shows the status of Docker Model Runner on Kubernetes
func showK8sStatus(ctx context.Context, namespace string) error {
	fmt.Printf("Checking status of Docker Model Runner in namespace '%s'...\n\n", namespace)

	// Check deployment status
	fmt.Println("=== Deployment Status ===")
	cmd := exec.CommandContext(ctx, "kubectl", "get", "deployment", "docker-model-runner", "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	fmt.Println("\n=== Pod Status ===")
	cmd = exec.CommandContext(ctx, "kubectl", "get", "pods", "-l", "app.kubernetes.io/name=docker-model-runner", "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	fmt.Println("\n=== Service Status ===")
	cmd = exec.CommandContext(ctx, "kubectl", "get", "service", "-l", "app.kubernetes.io/name=docker-model-runner", "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	// Try to get the endpoint
	fmt.Println("\n=== Endpoint Information ===")
	cmd = exec.CommandContext(ctx, "kubectl", "get", "endpoints", "docker-model-runner", "--namespace", namespace, "-o", "wide")
	output, err := cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		fmt.Print(string(output))
		
		// Check if there's a NodePort service
		cmd = exec.CommandContext(ctx, "kubectl", "get", "service", "docker-model-runner-nodeport", "--namespace", namespace, "-o", "jsonpath={.spec.ports[0].nodePort}")
		if portBytes, err := cmd.Output(); err == nil && len(portBytes) > 0 {
			fmt.Printf("\nNodePort available at: http://localhost:%s\n", string(portBytes))
			fmt.Printf("Test with: MODEL_RUNNER_HOST=http://localhost:%s docker model run ai/smollm2:latest\n", string(portBytes))
		} else {
			fmt.Println("\nTo access the service, set up port forwarding:")
			fmt.Printf("  kubectl port-forward -n %s deployment/docker-model-runner 31245:12434\n", namespace)
			fmt.Printf("  MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest\n")
		}
	}

	return nil
}

// findChartDirectory locates the Helm chart directory
func findChartDirectory() (string, error) {
	// Try to find the chart relative to the binary location
	possiblePaths := []string{
		"charts/docker-model-runner",
		"../../../charts/docker-model-runner",
		"../../charts/docker-model-runner",
		"/charts/docker-model-runner",
	}

	// Get the executable path
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		possiblePaths = append([]string{
			filepath.Join(exeDir, "charts/docker-model-runner"),
			filepath.Join(exeDir, "../charts/docker-model-runner"),
			filepath.Join(exeDir, "../../charts/docker-model-runner"),
			filepath.Join(exeDir, "../../../charts/docker-model-runner"),
		}, possiblePaths...)
	}

	// Check if we're in the repository
	if wd, err := os.Getwd(); err == nil {
		possiblePaths = append([]string{
			filepath.Join(wd, "charts/docker-model-runner"),
		}, possiblePaths...)
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			// Check if Chart.yaml exists
			chartFile := filepath.Join(path, "Chart.yaml")
			if _, err := os.Stat(chartFile); err == nil {
				absPath, _ := filepath.Abs(path)
				return absPath, nil
			}
		}
	}

	return "", fmt.Errorf("Helm chart not found. Please run this command from the model-runner repository root, or ensure the charts/ directory is available")
}

// waitForPods waits for pods to be ready
func waitForPods(ctx context.Context, namespace string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.CommandContext(ctx, "kubectl", "get", "pods",
			"-l", "app.kubernetes.io/name=docker-model-runner",
			"--namespace", namespace,
			"-o", "jsonpath={.items[*].status.phase}")
		
		output, err := cmd.Output()
		if err == nil {
			phases := strings.Fields(string(output))
			allRunning := len(phases) > 0
			for _, phase := range phases {
				if phase != "Running" {
					allRunning = false
					break
				}
			}
			if allRunning {
				return nil
			}
		}
		
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("timeout waiting for pods to be ready")
}
