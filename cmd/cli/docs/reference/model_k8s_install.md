# docker model k8s install

Install Docker Model Runner on Kubernetes

## Usage

```
docker model k8s install [flags]
```

## Description

Deploy Docker Model Runner to a Kubernetes cluster using Helm. This command:

1. Checks for required dependencies (kubectl, helm)
2. Verifies cluster connectivity
3. Creates the namespace if needed
4. Installs the Docker Model Runner Helm chart with specified configuration
5. Provides post-installation instructions

The command uses the Helm chart from `charts/docker-model-runner` in the repository.

## Options

```
  -d, --debug                  Enable debug output
  -h, --help                   help for install
      --kubeconfig string      Path to kubeconfig file
  -n, --namespace string       Kubernetes namespace (default "docker-model-runner")
  -r, --release string         Helm release name (default "docker-model-runner")
  -c, --storage-class string   Storage class for PVC
  -z, --storage-size string    Storage size for models (default "100Gi")
  -f, --values string          Path to Helm values file
```

## Examples

### Basic Installation

Install Docker Model Runner with default settings:

```bash
docker model k8s install
```

After installation, follow the printed instructions to access the service:

```bash
# Wait for deployment to be ready
kubectl wait --for=condition=Available deployment/docker-model-runner -n docker-model-runner --timeout=5m

# Set up port-forward
kubectl port-forward -n docker-model-runner service/docker-model-runner-nodeport 31245:80

# Test the model runner
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest
```

### Custom Storage Configuration

Install with a specific storage class and size:

```bash
docker model k8s install \
  --storage-class fast-ssd \
  --storage-size 200Gi
```

### Custom Namespace

Install to a specific namespace:

```bash
docker model k8s install --namespace my-ml-namespace
```

### Using Custom Values File

Install with a custom Helm values file:

```bash
docker model k8s install -f my-values.yaml
```

Example `my-values.yaml`:

```yaml
storage:
  size: 200Gi
  storageClass: "fast-ssd"

modelInit:
  enabled: true
  models:
    - "ai/smollm2:latest"
    - "ai/llama3.2:latest"

gpu:
  enabled: true
  vendor: nvidia
  count: 1
```

### Multiple Options Combined

Combine multiple configuration options:

```bash
docker model k8s install \
  --namespace ml-production \
  --release dmr-prod \
  --storage-class premium-ssd \
  --storage-size 500Gi \
  --values production-values.yaml \
  --debug
```

### Using Custom Kubeconfig

Install using a specific kubeconfig file:

```bash
docker model k8s install --kubeconfig ~/.kube/production-cluster
```

## Notes

- Requires `kubectl` and `helm` to be installed and available in PATH
- Requires an active Kubernetes cluster context
- The command must be run from the repository root or specify the chart location
- GPU support requires appropriate node configuration and drivers

## See Also

* [docker model k8s](model_k8s.md) - Kubernetes deployment commands
* [docker model k8s uninstall](model_k8s_uninstall.md) - Uninstall Docker Model Runner
* [Docker Model Runner Kubernetes Chart](../../../../charts/docker-model-runner/README.md)
