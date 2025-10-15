# docker model k8s uninstall

Uninstall Docker Model Runner from Kubernetes

## Usage

```
docker model k8s uninstall [flags]
```

## Description

Remove Docker Model Runner deployment from a Kubernetes cluster. This command:

1. Checks for required dependencies (kubectl, helm)
2. Uninstalls the Helm release
3. Optionally deletes the namespace

This command performs a clean removal of the Docker Model Runner installation from your Kubernetes cluster.

## Options

```
  -d, --debug               Enable debug output
  -h, --help                help for uninstall
      --kubeconfig string   Path to kubeconfig file
  -n, --namespace string    Kubernetes namespace (default "docker-model-runner")
  -r, --release string      Helm release name (default "docker-model-runner")
```

## Examples

### Basic Uninstallation

Uninstall Docker Model Runner from the default namespace:

```bash
docker model k8s uninstall
```

### Uninstall from Custom Namespace

Uninstall from a specific namespace:

```bash
docker model k8s uninstall --namespace my-ml-namespace
```

### Uninstall Custom Release

Uninstall a specific Helm release:

```bash
docker model k8s uninstall --release dmr-prod --namespace ml-production
```

### Using Custom Kubeconfig

Uninstall using a specific kubeconfig file:

```bash
docker model k8s uninstall --kubeconfig ~/.kube/production-cluster
```

### With Debug Output

Enable debug output to see detailed uninstallation steps:

```bash
docker model k8s uninstall --debug
```

## Notes

- Requires `kubectl` and `helm` to be installed and available in PATH
- Requires an active Kubernetes cluster context
- The command will delete the namespace, which removes all resources in it
- Persistent volumes may need to be cleaned up manually depending on your storage class configuration
- If the Helm release is not found, a warning is printed but the command continues

## Cleanup Notes

After uninstalling, you may want to manually clean up:

1. **Persistent Volumes**: Check for any remaining PVs
   ```bash
   kubectl get pv | grep docker-model-runner
   ```

2. **Storage Resources**: Verify PVCs are removed
   ```bash
   kubectl get pvc -A | grep docker-model-runner
   ```

3. **Custom Resources**: Check for any CRDs if applicable
   ```bash
   kubectl get crd | grep docker-model-runner
   ```

## See Also

* [docker model k8s](model_k8s.md) - Kubernetes deployment commands
* [docker model k8s install](model_k8s_install.md) - Install Docker Model Runner
* [Docker Model Runner Kubernetes Chart](../../../../charts/docker-model-runner/README.md)
