# docker model k8s

Kubernetes deployment commands for Docker Model Runner

## Usage

```
docker model k8s [command]
```

## Description

Install, uninstall, and manage Docker Model Runner on Kubernetes clusters.

The `k8s` command provides simple deployment and management of Docker Model Runner on Kubernetes using Helm. It leverages the existing Helm chart in the `charts/docker-model-runner` directory.

## Subcommands

| Command | Description |
|---------|-------------|
| [docker model k8s install](model_k8s_install.md) | Install Docker Model Runner on Kubernetes |
| [docker model k8s uninstall](model_k8s_uninstall.md) | Uninstall Docker Model Runner from Kubernetes |

## Options

```
  -h, --help   help for k8s
```

## Examples

### Install Docker Model Runner on Kubernetes

```bash
# Install with default settings
docker model k8s install

# Install with custom storage class
docker model k8s install --storage-class fast-ssd --storage-size 200Gi

# Install with custom values file
docker model k8s install -f custom-values.yaml

# Install to specific namespace
docker model k8s install --namespace my-namespace
```

### Uninstall Docker Model Runner from Kubernetes

```bash
# Uninstall from default namespace
docker model k8s uninstall

# Uninstall from specific namespace
docker model k8s uninstall --namespace my-namespace
```

## See Also

* [docker model](model.md) - Docker Model Runner
* [Docker Model Runner Kubernetes Chart](../../../../charts/docker-model-runner/README.md)
