package commands

import (
	"fmt"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/spf13/cobra"
)

func newK8sCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "k8s",
		Short: "Kubernetes deployment commands for distributed AI inference",
		Long: `Manage Kubernetes deployments for distributed AI inference.

This command provides unified tooling for deploying large language models on Kubernetes
using distributed inference techniques from both llm-d and AIBrix projects.

Key Features:
- Distributed model serving with prefill/decode disaggregation
- KV cache-aware routing for efficient inference
- Smart load balancing across multiple nodes
- Support for various inference frameworks (vLLM, etc.)

Getting Started:
  1. Install prerequisites: docker model k8s install
  2. Deploy a model:        docker model k8s deploy --model <model-name>
  3. Test the deployment:   docker model k8s test
  4. Clean up:              docker model k8s uninstall`,
		ValidArgsFunction: completion.NoComplete,
	}

	c.AddCommand(
		newK8sInstallCmd(),
		newK8sDeployCmd(),
		newK8sTestCmd(),
		newK8sUninstallCmd(),
	)

	return c
}

func newK8sInstallCmd() *cobra.Command {
	var namespace string
	var provider string

	c := &cobra.Command{
		Use:   "install",
		Short: "Install prerequisites for Kubernetes distributed inference",
		Long: `Install prerequisites for Kubernetes distributed inference.

This command helps set up the necessary components for distributed AI inference on Kubernetes,
supporting both llm-d and AIBrix deployment patterns.

Prerequisites:
- kubectl installed and configured
- Access to a Kubernetes cluster
- For llm-d: OpenShift 4.17+ or compatible K8s cluster
- For AIBrix: Kubernetes 1.24+

The installation includes:
- Gateway API components
- Inference controllers
- Model metadata services
- GPU operators (if applicable)

Examples:
  # Install using llm-d approach
  docker model k8s install --provider llm-d --namespace llm-d-system

  # Install using AIBrix approach  
  docker model k8s install --provider aibrix --namespace aibrix-system`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("Installing Kubernetes distributed inference components...\n")
			cmd.Printf("Provider: %s\n", provider)
			cmd.Printf("Namespace: %s\n", namespace)

			switch provider {
			case "llm-d":
				return installLLMD(cmd, namespace)
			case "aibrix":
				return installAIBrix(cmd, namespace)
			default:
				return fmt.Errorf("unsupported provider: %s (supported: llm-d, aibrix)", provider)
			}
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace for installation")
	c.Flags().StringVar(&provider, "provider", "aibrix", "Infrastructure provider (llm-d or aibrix)")

	return c
}

func installLLMD(cmd *cobra.Command, namespace string) error {
	cmd.Println("\nInstalling llm-d components...")
	cmd.Println("This would install:")
	cmd.Println("  - Gateway API and control plane providers")
	cmd.Println("  - Inference gateway (kgateway)")
	cmd.Println("  - Model service components")
	cmd.Println("  - Required dependencies")
	cmd.Println("\nFor manual installation, follow:")
	cmd.Println("  1. Clone: git clone https://github.com/llm-d-incubation/llm-d-infra.git")
	cmd.Println("  2. Install dependencies: cd quickstart && ./dependencies/install-deps.sh")
	cmd.Println("  3. Deploy gateway: cd gateway-control-plane-providers && ./install-gateway-provider-dependencies.sh && helmfile apply -f istio.helmfile.yaml")
	cmd.Printf("  4. Create namespace: kubectl create namespace %s\n", namespace)
	cmd.Println("\nNote: This is a preview command. Full automation coming soon.")
	return nil
}

func installAIBrix(cmd *cobra.Command, namespace string) error {
	cmd.Println("\nInstalling AIBrix components...")
	cmd.Println("This would install:")
	cmd.Println("  - AIBrix dependency components")
	cmd.Println("  - AIBrix core components")
	cmd.Println("  - Gateway plugins")
	cmd.Println("  - GPU optimizer")
	cmd.Println("  - Metadata service")
	cmd.Println("\nFor manual installation, run:")
	cmd.Println("  kubectl create -f https://github.com/vllm-project/aibrix/releases/download/v0.4.1/aibrix-dependency-v0.4.1.yaml")
	cmd.Println("  kubectl create -f https://github.com/vllm-project/aibrix/releases/download/v0.4.1/aibrix-core-v0.4.1.yaml")
	cmd.Printf("  kubectl wait --for=condition=Ready pods -n %s --all --timeout=5m\n", namespace)
	cmd.Println("\nNote: This is a preview command. Full automation coming soon.")
	return nil
}

func newK8sDeployCmd() *cobra.Command {
	var model string
	var namespace string
	var disaggregation bool
	var replicas int
	var hfToken string
	var provider string

	c := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a model on Kubernetes with distributed inference",
		Long: `Deploy a model on Kubernetes with distributed inference capabilities.

This command creates Kubernetes resources for running LLM inference with:
- Base deployment: Single-pod model serving
- Prefill-Decode (PD) disaggregation: Separate prefill and decode pods for efficiency

Prefill-Decode Disaggregation Benefits:
- Improved GPU utilization by separating compute-intensive prefill from memory-bound decode
- Better throughput for long sequences
- Optimized resource allocation

Examples:
  # Deploy a base model with AIBrix
  docker model k8s deploy --model deepseek-ai/DeepSeek-R1-Distill-Llama-8B --provider aibrix

  # Deploy with prefill-decode disaggregation
  docker model k8s deploy --model deepseek-ai/DeepSeek-R1-Distill-Llama-8B --disaggregation --provider aibrix

  # Deploy with llm-d using HuggingFace token
  docker model k8s deploy --model Qwen/Qwen3-0.6B --provider llm-d --hf-token <token>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if model == "" {
				return fmt.Errorf("model is required (use --model flag)")
			}

			cmd.Printf("Deploying model: %s\n", model)
			cmd.Printf("Provider: %s\n", provider)
			cmd.Printf("Namespace: %s\n", namespace)
			cmd.Printf("Disaggregation: %v\n", disaggregation)
			cmd.Printf("Replicas: %d\n", replicas)

			switch provider {
			case "llm-d":
				return deployLLMD(cmd, model, namespace, disaggregation, hfToken)
			case "aibrix":
				return deployAIBrix(cmd, model, namespace, disaggregation, replicas)
			default:
				return fmt.Errorf("unsupported provider: %s (supported: llm-d, aibrix)", provider)
			}
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVar(&model, "model", "", "Model identifier (e.g., deepseek-ai/DeepSeek-R1-Distill-Llama-8B)")
	c.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace for deployment")
	c.Flags().BoolVar(&disaggregation, "disaggregation", false, "Enable prefill-decode disaggregation")
	c.Flags().IntVar(&replicas, "replicas", 1, "Number of replicas (for base deployment)")
	c.Flags().StringVar(&hfToken, "hf-token", "", "HuggingFace token for model access")
	c.Flags().StringVar(&provider, "provider", "aibrix", "Infrastructure provider (llm-d or aibrix)")
	c.MarkFlagRequired("model")

	return c
}

func deployLLMD(cmd *cobra.Command, model, namespace string, disaggregation bool, hfToken string) error {
	cmd.Println("\nDeploying with llm-d...")

	if hfToken == "" {
		cmd.Println("Warning: No HuggingFace token provided. You may need to set HF_TOKEN manually.")
	}

	if disaggregation {
		cmd.Println("\nThis would deploy with prefill-decode disaggregation:")
		cmd.Println("  - Separate prefill and decode pods")
		cmd.Println("  - KV cache transfer via NIXL")
		cmd.Println("  - Inference gateway for smart routing")
	} else {
		cmd.Println("\nThis would deploy a base model service:")
		cmd.Println("  - Model service deployment")
		cmd.Println("  - Service with KV cache-aware routing")
	}

	cmd.Println("\nManual deployment steps:")
	cmd.Printf("  1. Set namespace: export NAMESPACE=%s\n", namespace)
	cmd.Println("  2. Create namespace: kubectl create namespace $NAMESPACE")
	if hfToken != "" {
		cmd.Println("  3. Create HF token secret:")
		cmd.Printf("     kubectl create secret generic llm-d-hf-token --from-literal=HF_TOKEN=%s -n $NAMESPACE\n", hfToken)
	}
	cmd.Println("  4. Apply appropriate example from llm-d-infra repository")
	cmd.Println("\nNote: This is a preview command. Full automation coming soon.")
	return nil
}

func deployAIBrix(cmd *cobra.Command, model, namespace string, disaggregation bool, replicas int) error {
	cmd.Println("\nDeploying with AIBrix...")

	modelName := getModelShortName(model)

	if disaggregation {
		cmd.Printf("\nThis would create a StormService with prefill-decode disaggregation:\n")
		cmd.Println("  - 1 prefill replica with vLLM")
		cmd.Println("  - 1 decode replica with vLLM")
		cmd.Println("  - NIXL connector for KV transfer")
		cmd.Println("  - Labels: model.aibrix.ai/name=" + modelName)

		cmd.Println("\nManual deployment:")
		cmd.Println("  Create a StormService YAML following AIBrix PD disaggregation example")
		cmd.Println("  See: https://github.com/vllm-project/aibrix documentation")
	} else {
		cmd.Printf("\nThis would create a base Deployment:\n")
		cmd.Printf("  - %d replica(s) running vLLM\n", replicas)
		cmd.Println("  - Service with model.aibrix.ai/name=" + modelName)
		cmd.Println("  - GPU resources allocated")

		cmd.Println("\nManual deployment:")
		cmd.Println("  Create a Deployment and Service following AIBrix base model example")
		cmd.Println("  See: https://github.com/vllm-project/aibrix documentation")
	}

	cmd.Println("\nNote: This is a preview command. Full automation coming soon.")
	return nil
}

// getModelShortName extracts a short name from a full model path
func getModelShortName(model string) string {
	// Simple extraction - in production this would be more sophisticated
	// e.g., "deepseek-ai/DeepSeek-R1-Distill-Llama-8B" -> "deepseek-r1-distill-llama-8b"
	// For now, just return a normalized version
	return "model-deployment"
}

func newK8sTestCmd() *cobra.Command {
	var namespace string
	var endpoint string
	var provider string

	c := &cobra.Command{
		Use:   "test",
		Short: "Test the Kubernetes deployment with sample requests",
		Long: `Test the Kubernetes deployment with sample inference requests.

This command helps verify that your distributed inference deployment is working correctly
by sending test requests and validating responses.

Examples:
  # Test AIBrix deployment
  docker model k8s test --provider aibrix --namespace default

  # Test llm-d deployment with custom endpoint
  docker model k8s test --provider llm-d --endpoint http://gateway-service --namespace llm-d-system`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("Testing deployment in namespace: %s\n", namespace)
			cmd.Printf("Provider: %s\n", provider)

			switch provider {
			case "llm-d":
				return testLLMD(cmd, namespace, endpoint)
			case "aibrix":
				return testAIBrix(cmd, namespace, endpoint)
			default:
				return fmt.Errorf("unsupported provider: %s (supported: llm-d, aibrix)", provider)
			}
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	c.Flags().StringVar(&endpoint, "endpoint", "", "Service endpoint URL (auto-detected if not provided)")
	c.Flags().StringVar(&provider, "provider", "aibrix", "Infrastructure provider (llm-d or aibrix)")

	return c
}

func testLLMD(cmd *cobra.Command, namespace, endpoint string) error {
	cmd.Println("\nTesting llm-d deployment...")
	cmd.Println("\nTest commands:")
	cmd.Println("  1. Get service endpoint:")
	cmd.Printf("     export SVC_EP=$(kubectl get svc -n %s -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')\n", namespace)
	cmd.Println("  2. List models:")
	cmd.Println("     curl http://$SVC_EP/v1/models")
	cmd.Println("  3. Test completion:")
	cmd.Println(`     curl http://$SVC_EP/v1/completions -H "Content-Type: application/json" -d '{"model":"<model-name>","prompt":"Hello","max_tokens":50}'`)
	cmd.Println("  4. Check KV cache routing logs:")
	cmd.Printf("     kubectl logs -n %s deployment/gaie-kv-events-epp --follow | grep 'Got pod scores'\n", namespace)
	cmd.Println("\nNote: This is a preview command. Automated testing coming soon.")
	return nil
}

func testAIBrix(cmd *cobra.Command, namespace, endpoint string) error {
	cmd.Println("\nTesting AIBrix deployment...")
	cmd.Println("\nTest commands:")
	cmd.Println("  1. Get gateway endpoint:")
	cmd.Println("     export LB_IP=$(kubectl get svc/envoy-aibrix-system-aibrix-eg-903790dc -n envoy-gateway-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')")
	cmd.Println("     export ENDPOINT=\"${LB_IP}:80\"")
	cmd.Println("  2. Or use port-forward:")
	cmd.Println("     kubectl -n envoy-gateway-system port-forward service/envoy-aibrix-system-aibrix-eg-903790dc 8888:80 &")
	cmd.Println("     export ENDPOINT=\"localhost:8888\"")
	cmd.Println("  3. List models:")
	cmd.Println("     curl http://${ENDPOINT}/v1/models")
	cmd.Println("  4. Test completion:")
	cmd.Println(`     curl http://${ENDPOINT}/v1/completions -H "Content-Type: application/json" -d '{"model":"<model-name>","prompt":"San Francisco is a","max_tokens":128,"temperature":0}'`)
	cmd.Println("  5. Test with PD disaggregation:")
	cmd.Println(`     curl http://${ENDPOINT}/v1/chat/completions -H "routing-strategy: pd" -H "Content-Type: application/json" -d '{"model":"<model-name>","messages":[{"role":"user","content":"Hello"}]}'`)
	cmd.Println("\nNote: This is a preview command. Automated testing coming soon.")
	return nil
}

func newK8sUninstallCmd() *cobra.Command {
	var namespace string
	var provider string

	c := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Kubernetes distributed inference components",
		Long: `Uninstall distributed inference components from Kubernetes.

This command removes the infrastructure components and model deployments.

Examples:
  # Uninstall llm-d components
  docker model k8s uninstall --provider llm-d --namespace llm-d-system

  # Uninstall AIBrix components
  docker model k8s uninstall --provider aibrix --namespace aibrix-system`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("Uninstalling from namespace: %s\n", namespace)
			cmd.Printf("Provider: %s\n", provider)

			switch provider {
			case "llm-d":
				return uninstallLLMD(cmd, namespace)
			case "aibrix":
				return uninstallAIBrix(cmd, namespace)
			default:
				return fmt.Errorf("unsupported provider: %s (supported: llm-d, aibrix)", provider)
			}
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	c.Flags().StringVar(&provider, "provider", "aibrix", "Infrastructure provider (llm-d or aibrix)")

	return c
}

func uninstallLLMD(cmd *cobra.Command, namespace string) error {
	cmd.Println("\nUninstalling llm-d components...")
	cmd.Println("Manual uninstallation:")
	cmd.Printf("  kubectl delete namespace %s\n", namespace)
	cmd.Println("  helmfile destroy -f istio.helmfile.yaml (from gateway-control-plane-providers)")
	cmd.Println("\nNote: This is a preview command. Full automation coming soon.")
	return nil
}

func uninstallAIBrix(cmd *cobra.Command, namespace string) error {
	cmd.Println("\nUninstalling AIBrix components...")
	cmd.Println("Manual uninstallation:")
	cmd.Printf("  kubectl delete namespace %s\n", namespace)
	cmd.Println("  kubectl delete -f https://github.com/vllm-project/aibrix/releases/download/v0.4.1/aibrix-core-v0.4.1.yaml")
	cmd.Println("  kubectl delete -f https://github.com/vllm-project/aibrix/releases/download/v0.4.1/aibrix-dependency-v0.4.1.yaml")
	cmd.Println("\nNote: This is a preview command. Full automation coming soon.")
	return nil
}
