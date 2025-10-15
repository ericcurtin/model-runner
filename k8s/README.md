# Docker Model Runner Kubernetes Deployment

This directory contains Kubernetes manifests for deploying Docker Model Runner on Kubernetes clusters.

## Overview

These manifests provide a basic deployment configuration for running Docker Model Runner in Kubernetes with:
- Service account and RBAC permissions
- Deployment with configurable resources
- Service for cluster-internal access
- Persistent storage for models

## Quick Start

### Deploy using kubectl

```bash
kubectl apply -k .
```

This will deploy the Docker Model Runner to your current namespace.

### Deploy to a specific namespace

```bash
kubectl apply -k . -n <namespace>
```

### Verify the deployment

```bash
kubectl get pods -l app=docker-model-runner
kubectl get service docker-model-runner
```

## Configuration

### Resource Limits

The default deployment requests:
- CPU: 4 cores (limit: 8 cores)
- Memory: 8Gi (limit: 16Gi)

To modify these, edit the `deployment.yaml` file.

### Storage

By default, the deployment uses `emptyDir` for model storage. For persistent storage:

1. Create a PersistentVolumeClaim
2. Update the volume configuration in `deployment.yaml`

### Image

The default image is `ghcr.io/docker/model-runner:latest`. To use a different version:

```bash
kubectl set image deployment/docker-model-runner model-runner=ghcr.io/docker/model-runner:v1.0.0
```

## Access

The service exposes the model runner on port 80 within the cluster. To access from outside:

### Port forward (for testing)

```bash
kubectl port-forward service/docker-model-runner 12434:80
```

Then access at `http://localhost:12434`

### Create an Ingress or NodePort

For production access, configure an Ingress or change the service type to NodePort/LoadBalancer.

## Cleanup

```bash
kubectl delete -k .
```

## GPU Support

To enable GPU support, add resource requests to the deployment:

```yaml
resources:
  limits:
    nvidia.com/gpu: 1  # For NVIDIA GPUs
    # or
    amd.com/gpu: 1     # For AMD GPUs
```

Also ensure your cluster has the appropriate GPU drivers and device plugins installed.
