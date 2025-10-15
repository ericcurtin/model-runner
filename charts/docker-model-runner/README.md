# Docker Model Runner Kubernetes Support

Manifests for deploying Docker Model Runner on Kubernetes with ephemeral storage, GPU support, and model pre-pulling capabilities.

## Quickstart

### On Docker Desktop

```
kubectl apply -f static/docker-model-runner-desktop.yaml
kubectl wait --for=condition=Available deployment/docker-model-runner --timeout=5m
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest
```

### On any Kubernetes Cluster

```
kubectl apply -f static/docker-model-runner.yaml
kubectl wait --for=condition=Available deployment/docker-model-runner --timeout=5m
kubectl port-forward deployment/docker-model-runner 31245:12434
```

Then:

```
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest
```

## Helm Configuration

### Basic Configuration

Key configuration options in `values.yaml`:

```yaml
# Storage configuration
storage:
  size: 100Gi
  storageClass: ""  # Set this to the storage class of your cloud provider.

# Model pre-pull configuration
modelInit:
  enabled: false
  models:
    - "ai/smollm2:latest"

# GPU configuration
gpu:
  enabled: false
  vendor: nvidia  # or amd
  count: 1

# NodePort configuration
nodePort:
  enabled: false
  port: 31245
```

### GPU Scheduling

To enable GPU scheduling:

```yaml
gpu:
  enabled: true
  vendor: nvidia  # or amd
  count: 1
```

This will add the appropriate resource requests/limits:
- NVIDIA: `nvidia.com/gpu`
- AMD: `amd.com/gpu`

### Model Pre-pulling

Configure models to pre-pull during pod initialization:

```yaml
modelInit:
  enabled: true
  models:
    - "ai/smollm2:latest"
    - "ai/llama3.2:latest"
    - "ai/mistral:latest"
```

### Cloud Provider Storage Classes

Different cloud providers use different storage class names:

**AWS EKS:**
```yaml
storage:
  storageClass: gp2  # or gp3 for better performance
```

**GCP GKE:**
```yaml
storage:
  storageClass: standard  # or standard-rwo
```

**Azure AKS:**
```yaml
storage:
  storageClass: default  # or managed-premium for SSDs
```

**On-premises/Custom:**
```yaml
storage:
  storageClass: ""  # Use default storage class
```

## Installation

### Using Helm (Recommended)

```bash
# Install with default values
helm install docker-model-runner charts/docker-model-runner

# Install with custom values
helm install docker-model-runner charts/docker-model-runner \
  --set storage.size=200Gi \
  --set storage.storageClass=gp3 \
  --set gpu.enabled=true \
  --set gpu.vendor=nvidia \
  --set gpu.count=1

# Install with model pre-pulling
helm install docker-model-runner charts/docker-model-runner \
  --set modelInit.enabled=true \
  --set modelInit.models[0]=ai/smollm2:latest

# Install with custom values file
helm install docker-model-runner charts/docker-model-runner -f my-values.yaml
```

### Using kubectl with static manifests

Pre-rendered manifests are available in the `static/` directory:

```bash
# For Docker Desktop (with NodePort enabled)
kubectl apply -f charts/docker-model-runner/static/docker-model-runner-desktop.yaml

# For regular Kubernetes
kubectl apply -f charts/docker-model-runner/static/docker-model-runner.yaml

# For AWS EKS
kubectl apply -f charts/docker-model-runner/static/docker-model-runner-eks.yaml

# With pre-pulled smollm2 model
kubectl apply -f charts/docker-model-runner/static/docker-model-runner-smollm2.yaml
```

## Usage

### Testing the Installation

Once installed, set up a port-forward to access the service:

```bash
kubectl port-forward service/docker-model-runner-nodeport 31245:80
```

Then test the model runner:

```bash
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest
```

### Checking Status

```bash
# Check deployment status
kubectl get deployment docker-model-runner

# Check pod status
kubectl get pods -l app.kubernetes.io/name=docker-model-runner

# View logs
kubectl logs -l app.kubernetes.io/name=docker-model-runner --follow

# Check service
kubectl get service docker-model-runner
```

## Upgrading

### Using Helm

```bash
# Upgrade with new values
helm upgrade docker-model-runner charts/docker-model-runner \
  --set storage.size=200Gi

# Upgrade with custom values file
helm upgrade docker-model-runner charts/docker-model-runner -f my-values.yaml
```

### Using kubectl

```bash
# Reapply the manifest
kubectl apply -f charts/docker-model-runner/static/docker-model-runner.yaml
```

## Uninstalling

### Using Helm

```bash
helm uninstall docker-model-runner
```

### Using kubectl

```bash
kubectl delete -f charts/docker-model-runner/static/docker-model-runner.yaml
```

## Values Reference

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Docker image repository | `docker/model-runner` |
| `image.tag` | Docker image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `80` |
| `service.targetPort` | Container port | `12434` |
| `nodePort.enabled` | Enable NodePort service | `false` |
| `nodePort.port` | NodePort port number | `31245` |
| `storage.size` | PVC size | `100Gi` |
| `storage.storageClass` | Storage class name | `""` (default) |
| `storage.accessMode` | Volume access mode | `ReadWriteOnce` |
| `modelInit.enabled` | Enable model pre-pulling | `false` |
| `modelInit.models` | List of models to pre-pull | `[]` |
| `gpu.enabled` | Enable GPU support | `false` |
| `gpu.vendor` | GPU vendor (nvidia/amd) | `nvidia` |
| `gpu.count` | Number of GPUs | `1` |
| `resources.limits` | Resource limits | `{}` |
| `resources.requests` | Resource requests | `{}` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Pod tolerations | `[]` |
| `affinity` | Pod affinity | `{}` |

## Troubleshooting

### Pod is Pending

Check if the storage class is available:
```bash
kubectl get storageclass
```

If no default storage class exists, specify one:
```yaml
storage:
  storageClass: standard  # or your cluster's storage class
```

### Model Pulling Fails

Check the init container logs:
```bash
kubectl logs <pod-name> -c model-puller
```

Ensure the models exist in the Docker Hub registry or your configured registry.

### GPU Not Available

Check if GPU drivers are installed on nodes:
```bash
kubectl get nodes -o json | jq '.items[].status.allocatable'
```

Verify the GPU device plugin is running:
```bash
# For NVIDIA
kubectl get daemonset -n kube-system nvidia-device-plugin-daemonset

# For AMD
kubectl get daemonset -n kube-system amd-gpu-device-plugin
```

### Service Not Accessible

For NodePort issues on Docker Desktop:
```bash
# Check if the service is created
kubectl get service docker-model-runner-nodeport

# Get the node port
kubectl get service docker-model-runner-nodeport -o jsonpath='{.spec.ports[0].nodePort}'
```

For port-forward issues:
```bash
# Ensure pod is running
kubectl get pods -l app.kubernetes.io/name=docker-model-runner

# Try direct pod port-forward
kubectl port-forward <pod-name> 31245:12434
```

## Advanced Configuration

### Custom Resource Limits

```yaml
resources:
  limits:
    cpu: 4000m
    memory: 8Gi
  requests:
    cpu: 2000m
    memory: 4Gi
```

### Node Affinity for GPU Nodes

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: accelerator
          operator: In
          values:
          - nvidia-tesla-v100
          - nvidia-tesla-t4
```

### Environment Variables

```yaml
env:
  - name: MODEL_RUNNER_LOG_LEVEL
    value: "debug"
  - name: DISABLE_SERVER_UPDATE
    value: "true"
  - name: LLAMA_SERVER_VERSION
    value: "v1.0.0"
```

### Ingress Configuration

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: model-runner.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: model-runner-tls
      hosts:
        - model-runner.example.com
```

## Security Considerations

The deployment runs as a non-root user (UID 2000) by default with the following security context:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 2000
  capabilities:
    drop:
    - ALL
  allowPrivilegeEscalation: false
```

For production deployments, consider:
- Using private image registries
- Enabling pod security policies/standards
- Implementing network policies
- Using secrets for sensitive configuration
- Enabling RBAC appropriately

## Support

For issues and questions:
- GitHub Issues: https://github.com/docker/model-runner/issues
- Documentation: https://docs.docker.com/ai/model-runner/

## License

Apache 2.0
