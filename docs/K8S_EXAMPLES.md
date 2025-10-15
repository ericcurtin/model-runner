# Docker Model Runner Kubernetes Examples

This document provides comprehensive examples for deploying Docker Model Runner to Kubernetes using the `docker model k8s` command.

## Overview

The `docker model k8s` command provides functionality similar to llm-d and aibrix for deploying models to Kubernetes, but it uses Docker Model Runner's native infrastructure without requiring external dependencies.

## Prerequisites

- A Kubernetes cluster (Docker Desktop, minikube, kind, EKS, GKE, AKS, etc.)
- `kubectl` configured to access your cluster
- `helm` (optional, for Helm-based deployments)
- Docker Model CLI installed

## Basic Examples

### 1. Simple Installation

Deploy Docker Model Runner with default settings:

```bash
# Install to the default namespace
docker model k8s install

# Check the deployment status
docker model k8s status

# Access the service via port-forward
kubectl port-forward deployment/docker-model-runner 31245:12434

# Test the deployment
MODEL_RUNNER_HOST=http://localhost:31245 docker model list
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest "Hello, Kubernetes!"
```

### 2. Installation with Custom Namespace

Deploy to a specific namespace:

```bash
# Install to a custom namespace
docker model k8s install --namespace my-models

# Check status
docker model k8s status --namespace my-models

# Clean up
docker model k8s uninstall --namespace my-models
```

### 3. Docker Desktop Installation with NodePort

For Docker Desktop users, NodePort enables direct access without port-forwarding:

```bash
# Install with NodePort enabled
docker model k8s install --node-port

# Access directly
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest "Tell me about Kubernetes"

# Clean up
docker model k8s uninstall
```

## GPU Examples

### 4. NVIDIA GPU Deployment

Deploy with NVIDIA GPU support:

```bash
# Install with NVIDIA GPU
docker model k8s install \
  --gpu \
  --gpu-vendor nvidia \
  --gpu-count 1 \
  --namespace gpu-models

# Verify GPU allocation
kubectl get pods -n gpu-models -o yaml | grep -A 5 "resources:"

# Test with a GPU-accelerated model
kubectl port-forward -n gpu-models deployment/docker-model-runner 31245:12434
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/llama3.2:latest "What is AI?"

# Clean up
docker model k8s uninstall --namespace gpu-models
```

### 5. AMD GPU Deployment

Deploy with AMD GPU support:

```bash
# Install with AMD GPU
docker model k8s install \
  --gpu \
  --gpu-vendor amd \
  --gpu-count 1

# Check status
docker model k8s status
```

### 6. Multi-GPU Deployment

Deploy with multiple GPUs:

```bash
# Install with multiple GPUs
docker model k8s install \
  --gpu \
  --gpu-vendor nvidia \
  --gpu-count 2 \
  --namespace multi-gpu

# Verify
kubectl describe pod -n multi-gpu -l app.kubernetes.io/name=docker-model-runner
```

## Storage Examples

### 7. Custom Storage Size

Deploy with a larger storage volume:

```bash
# Install with 200Gi storage
docker model k8s install \
  --storage-size 200Gi \
  --namespace large-models

# Verify storage
kubectl get pvc -n large-models
```

### 8. Custom Storage Class

Deploy with a specific storage class (e.g., for cloud providers):

```bash
# AWS EKS with gp3 storage
docker model k8s install \
  --storage-class gp3 \
  --storage-size 150Gi \
  --namespace production

# GKE with pd-ssd storage
docker model k8s install \
  --storage-class pd-ssd \
  --storage-size 150Gi \
  --namespace production

# Azure AKS with managed-premium storage
docker model k8s install \
  --storage-class managed-premium \
  --storage-size 150Gi \
  --namespace production
```

## Model Pre-pulling Examples

### 9. Single Model Pre-pull

Pre-pull a model during pod initialization:

```bash
# Install with model pre-pulling
docker model k8s install \
  --models ai/smollm2:latest \
  --namespace prepull-demo

# Check init container logs
kubectl logs -n prepull-demo -l app.kubernetes.io/name=docker-model-runner -c model-init

# Test the pre-pulled model
kubectl port-forward -n prepull-demo deployment/docker-model-runner 31245:12434
MODEL_RUNNER_HOST=http://localhost:31245 docker model list
```

### 10. Multiple Models Pre-pull

Pre-pull multiple models:

```bash
# Install with multiple models
docker model k8s install \
  --models ai/smollm2:latest,ai/llama3.2:latest,ai/mistral:latest \
  --storage-size 200Gi \
  --namespace multi-model

# Verify all models are loaded
kubectl port-forward -n multi-model deployment/docker-model-runner 31245:12434
MODEL_RUNNER_HOST=http://localhost:31245 docker model list
```

## Advanced Examples

### 11. Production Deployment with All Features

Complete production deployment with GPU, storage, and model pre-pulling:

```bash
# Production deployment
docker model k8s install \
  --namespace production \
  --gpu \
  --gpu-vendor nvidia \
  --gpu-count 1 \
  --storage-size 300Gi \
  --storage-class fast-ssd \
  --models ai/llama3.2:latest,ai/mistral:latest

# Monitor deployment
kubectl get all -n production
docker model k8s status --namespace production

# Test deployment
kubectl port-forward -n production deployment/docker-model-runner 31245:12434
MODEL_RUNNER_HOST=http://localhost:31245 docker model list
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/llama3.2:latest "Explain Kubernetes deployments"
```

### 12. Using Helm Directly

For advanced users who prefer Helm:

```bash
# Install using Helm
docker model k8s install \
  --helm \
  --namespace helm-demo \
  --gpu \
  --gpu-vendor nvidia

# Check Helm releases
helm list -n helm-demo

# Uninstall using Helm
docker model k8s uninstall --helm --namespace helm-demo
```

## Comparison with llm-d and aibrix

### Key Differences

1. **No External Dependencies**: Unlike llm-d and aibrix, `docker model k8s` uses Docker Model Runner's native infrastructure
2. **Simple CLI**: Single command installation instead of complex setup scripts
3. **Integrated**: Works seamlessly with other Docker Model commands
4. **Flexible**: Supports both Helm and kubectl-based deployments

### Migration from llm-d

If you're using llm-d, you can achieve similar functionality:

**llm-d approach:**
```bash
# llm-d requires multiple steps
git clone https://github.com/llm-d-incubation/llm-d-infra.git
cd llm-d-infra/quickstart
./dependencies/install-deps.sh
cd gateway-control-plane-providers
./install-gateway-provider-dependencies.sh
helmfile apply -f istio.helmfile.yaml
# ... more steps
```

**docker model k8s approach:**
```bash
# Single command
docker model k8s install --gpu --models ai/llama3.2:latest
```

### Migration from aibrix

If you're using aibrix, you can achieve similar functionality:

**aibrix approach:**
```bash
# aibrix requires custom resources
kubectl create -f https://github.com/vllm-project/aibrix/releases/download/v0.4.1/aibrix-dependency-v0.4.1.yaml
kubectl create -f https://github.com/vllm-project/aibrix/releases/download/v0.4.1/aibrix-core-v0.4.1.yaml
# Wait for pods, then deploy models
kubectl apply -f model.yaml
```

**docker model k8s approach:**
```bash
# Single command
docker model k8s install --gpu --models ai/deepseek-r1-distill-llama-8b:latest
```

## Troubleshooting

### Check Pod Status

```bash
# Get pod status
kubectl get pods -n <namespace> -l app.kubernetes.io/name=docker-model-runner

# View pod logs
kubectl logs -n <namespace> -l app.kubernetes.io/name=docker-model-runner

# Describe pod for detailed info
kubectl describe pod -n <namespace> -l app.kubernetes.io/name=docker-model-runner
```

### Check Model Pre-pull Progress

```bash
# View init container logs
kubectl logs -n <namespace> -l app.kubernetes.io/name=docker-model-runner -c model-init
```

### Access Service Without Port-Forward

```bash
# Option 1: Use NodePort (Docker Desktop)
docker model k8s install --node-port

# Option 2: Use kubectl proxy
kubectl proxy &
# Access at http://localhost:8001/api/v1/namespaces/<namespace>/services/docker-model-runner:80/proxy/

# Option 3: Create a LoadBalancer service (cloud providers)
# Modify the service type in values.yaml or use kubectl patch
```

## Best Practices

1. **Use Namespaces**: Isolate deployments using namespaces
2. **Set Resource Limits**: Configure appropriate storage sizes for your models
3. **Use Storage Classes**: Specify storage classes appropriate for your cloud provider
4. **Pre-pull Models**: Use `--models` flag to reduce startup time
5. **Monitor Resources**: Use `docker model k8s status` to monitor deployments
6. **Clean Up**: Always run `docker model k8s uninstall` when done

## Next Steps

- Explore the [Helm chart documentation](../charts/docker-model-runner/README.md)
- Read the [CLI documentation](../cmd/cli/README.md)
- Check out the [main README](../README.md) for more information
