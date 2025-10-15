# Docker Model K8s Command

The `docker model k8s` command provides unified tooling for deploying large language models on Kubernetes using distributed inference techniques from both llm-d and AIBrix projects.

## Overview

This command covers the functionality of both llm-d and AIBrix getting started guides, providing a consistent interface for:

- **Distributed model serving** with prefill/decode disaggregation
- **KV cache-aware routing** for efficient inference
- **Smart load balancing** across multiple nodes
- **Support for various inference frameworks** (vLLM, etc.)

## Key Concepts

### Prefill-Decode (PD) Disaggregation

PD disaggregation separates the inference process into two phases:

1. **Prefill phase**: Compute-intensive processing of input tokens in parallel
2. **Decode phase**: Memory bandwidth-bound generation of output tokens one at a time

By running these phases on separate specialized nodes or GPUs, you can:
- Improve GPU utilization
- Increase throughput for long sequences
- Optimize resource allocation

### KV Cache-Aware Routing

KV cache-aware routing intelligently directs requests to nodes that already have cached computation results (KV cache) for similar prompts, dramatically reducing latency and improving efficiency.

## Commands

### `docker model k8s install`

Install prerequisites for Kubernetes distributed inference.

**Supported Providers:**
- `llm-d`: Kubernetes-native distributed inference stack (requires OpenShift 4.17+ or compatible K8s)
- `aibrix`: vLLM-based distributed inference platform (requires Kubernetes 1.24+)

**Examples:**

```bash
# Install using llm-d approach
docker model k8s install --provider llm-d --namespace llm-d-system

# Install using AIBrix approach  
docker model k8s install --provider aibrix --namespace aibrix-system
```

**What gets installed:**
- Gateway API components
- Inference controllers
- Model metadata services
- GPU operators (if applicable)

### `docker model k8s deploy`

Deploy a model on Kubernetes with distributed inference capabilities.

**Deployment Modes:**
- **Base deployment**: Single-pod model serving
- **PD disaggregation**: Separate prefill and decode pods for improved efficiency

**Examples:**

```bash
# Deploy a base model with AIBrix
docker model k8s deploy \
  --model deepseek-ai/DeepSeek-R1-Distill-Llama-8B \
  --provider aibrix

# Deploy with prefill-decode disaggregation
docker model k8s deploy \
  --model deepseek-ai/DeepSeek-R1-Distill-Llama-8B \
  --disaggregation \
  --provider aibrix

# Deploy with llm-d using HuggingFace token
docker model k8s deploy \
  --model Qwen/Qwen3-0.6B \
  --provider llm-d \
  --hf-token <your-token>
```

**Flags:**
- `--model`: Model identifier (required)
- `--provider`: Infrastructure provider (llm-d or aibrix, default: aibrix)
- `--namespace`: Kubernetes namespace (default: default)
- `--disaggregation`: Enable prefill-decode disaggregation
- `--replicas`: Number of replicas for base deployment (default: 1)
- `--hf-token`: HuggingFace token for model access

### `docker model k8s test`

Test the Kubernetes deployment with sample inference requests.

**Examples:**

```bash
# Test AIBrix deployment
docker model k8s test --provider aibrix --namespace default

# Test llm-d deployment with custom endpoint
docker model k8s test \
  --provider llm-d \
  --endpoint http://gateway-service \
  --namespace llm-d-system
```

**What it provides:**
- Commands to get the service endpoint
- Example curl commands for testing
- Instructions for monitoring KV cache routing (llm-d)
- Port-forwarding instructions for local testing

### `docker model k8s uninstall`

Uninstall distributed inference components from Kubernetes.

**Examples:**

```bash
# Uninstall llm-d components
docker model k8s uninstall --provider llm-d --namespace llm-d-system

# Uninstall AIBrix components
docker model k8s uninstall --provider aibrix --namespace aibrix-system
```

## Provider Comparison

### llm-d

**Best for:**
- Large-scale production deployments
- OpenShift environments
- Advanced KV cache optimization
- Long-running, multi-step prompts
- Retrieval-augmented generation (RAG)

**Features:**
- KV cache-aware routing with precise prefix matching
- Disaggregated prefill/decode with NIXL
- vLLM-optimized inference scheduler
- Inference Gateway (IGW) for Kubernetes-native operations

**Prerequisites:**
- OpenShift 4.17+ or compatible Kubernetes
- NVIDIA GPU Operator 25.3+
- Node Feature Discovery Operator 4.18+
- No service mesh or Istio (conflicts with gateway)

### AIBrix

**Best for:**
- General Kubernetes clusters
- vLLM-based workloads
- Rapid prototyping
- Flexible deployment patterns

**Features:**
- StormService CRD for orchestration
- Multiple routing strategies (least-request, prefix-cache, pd)
- Gateway plugins and GPU optimization
- Redis-based metadata service

**Prerequisites:**
- Kubernetes 1.24+
- GPU support (NVIDIA/AMD device plugins)

## Getting Started Workflows

### Quick Start with AIBrix

1. **Install AIBrix components:**
   ```bash
   docker model k8s install --provider aibrix
   ```

2. **Deploy a model:**
   ```bash
   docker model k8s deploy \
     --model deepseek-ai/DeepSeek-R1-Distill-Llama-8B \
     --provider aibrix
   ```

3. **Test the deployment:**
   ```bash
   docker model k8s test --provider aibrix
   ```

### Advanced Setup with llm-d

1. **Install llm-d infrastructure:**
   ```bash
   docker model k8s install --provider llm-d --namespace llm-d-precise
   ```

2. **Deploy with PD disaggregation:**
   ```bash
   docker model k8s deploy \
     --model Qwen/Qwen3-0.6B \
     --provider llm-d \
     --namespace llm-d-precise \
     --disaggregation \
     --hf-token $HF_TOKEN
   ```

3. **Test KV cache-aware routing:**
   ```bash
   docker model k8s test --provider llm-d --namespace llm-d-precise
   ```

## Common Use Cases

### 1. Cost-Efficient Inference

Use disaggregation to optimize GPU utilization:

```bash
docker model k8s deploy \
  --model meta-llama/Llama-3.2-70B \
  --disaggregation \
  --provider aibrix
```

### 2. High-Throughput Serving

Deploy multiple replicas for load distribution:

```bash
docker model k8s deploy \
  --model deepseek-ai/DeepSeek-R1-Distill-Llama-8B \
  --replicas 3 \
  --provider aibrix
```

### 3. Long Context Processing

Use llm-d's KV cache optimization for long sequences:

```bash
docker model k8s deploy \
  --model Qwen/Qwen3-0.6B \
  --provider llm-d \
  --disaggregation
```

## Troubleshooting

### Pod Stuck in Pending State

If pods are stuck in pending, check for GPU taints:

```bash
# For llm-d deployments
kubectl get nodes -o json | jq '.items[].spec.taints'

# Add tolerations to your deployment
kubectl patch deployment <deployment-name> \
  -p '{"spec":{"template":{"spec":{"tolerations":[{"key":"nvidia.com/gpu","operator":"Equal","value":"NVIDIA-L40S-PRIVATE","effect":"NoSchedule"}]}}}}'
```

### Gateway Not Accessible

For AIBrix, check if the gateway service has an external IP:

```bash
kubectl get svc -n envoy-gateway-system

# If no external IP, use port-forward:
kubectl -n envoy-gateway-system port-forward \
  service/envoy-aibrix-system-aibrix-eg-903790dc 8888:80
```

### Model Download Issues

For llm-d, ensure HuggingFace token is correctly set:

```bash
# Create token secret manually
kubectl create secret generic llm-d-hf-token \
  --from-literal=HF_TOKEN=$HF_TOKEN \
  -n <namespace>
```

## Next Steps

- Review the [llm-d documentation](https://github.com/llm-d-incubation/llm-d-infra) for advanced features
- Explore [AIBrix examples](https://github.com/vllm-project/aibrix) for custom deployments
- Join the communities:
  - [llm-d Slack](https://llm-d.slack.com)
  - AIBrix GitHub Discussions

## Current Limitations

This is a preview command. Current version provides:
- ✅ Comprehensive guidance and examples
- ✅ Command structure and help text
- ✅ Manual deployment instructions
- 🔄 Automated installation (coming soon)
- 🔄 Automated deployment (coming soon)
- 🔄 Automated testing (coming soon)

Full automation of the installation, deployment, and testing workflows is planned for future releases.
