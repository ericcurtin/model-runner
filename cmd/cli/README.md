# Docker Model CLI

A powerful command-line interface for managing, running, packaging, and deploying AI/ML models using Docker. This CLI lets you install and control the Docker Model Runner, interact with models, manage model artifacts, and integrate with OpenAI and other backends—all from your terminal.

## Features
- **Install Model Runner**: Easily set up the Docker Model Runner for local or cloud environments with GPU support.
- **Run Models**: Execute models with prompts or in interactive chat mode, supporting multiline input and OpenAI-style backends.
- **List Models**: View all models available locally or via OpenAI, with options for JSON and quiet output.
- **Package Models**: Convert GGUF files into Docker model OCI artifacts and push them to registries, including license and context size options.
- **Configure Models**: Set runtime flags and context sizes for models.
- **Logs & Status**: Stream logs and check the status of the Model Runner and individual models.
- **Tag, Pull, Push, Remove, Unload**: Full lifecycle management for model artifacts.
- **Compose & Desktop Integration**: Advanced orchestration and desktop support for model backends.
- **Kubernetes Deployment**: Deploy and manage Docker Model Runner on Kubernetes with GPU support, model pre-pulling, and custom storage configurations.

## Building
1. **Clone the repo:**
   ```bash
   git clone https://github.com/docker/model-cli.git
   cd model-cli
   ```
2. **Build the CLI:**
   ```bash
   make build
   ```
3. **Install Model Runner:**
   ```bash
   ./model install-runner
   ```
   Use `--gpu cuda` for GPU support, or `--gpu auto` for automatic detection.

## Usage
Run `./model --help` to see all commands and options.

### Common Commands
- `model install-runner` — Install the Docker Model Runner
- `model run MODEL [PROMPT]` — Run a model with a prompt or enter chat mode
- `model list` — List available models
- `model package --gguf <path> --push <target>` — Package and push a model
- `model logs` — View logs
- `model status` — Check runner status
- `model configure MODEL [flags]` — Configure model runtime
- `model unload MODEL` — Unload a model
- `model tag SOURCE TARGET` — Tag a model
- `model pull MODEL` — Pull a model
- `model push MODEL` — Push a model
- `model rm MODEL` — Remove a model
- `model k8s install` — Deploy Docker Model Runner to Kubernetes
- `model k8s status` — Check Kubernetes deployment status
- `model k8s uninstall` — Remove Docker Model Runner from Kubernetes

## Example: Interactive Chat
```bash
./model run llama.cpp "What is the capital of France?"
```
Or enter chat mode:
```bash
./model run llama.cpp
Interactive chat mode started. Type '/bye' to exit.
> """
Tell me a joke.
"""
```

## Kubernetes Deployment

Deploy Docker Model Runner to Kubernetes for distributed model serving, similar to llm-d and aibrix, but using Docker Model Runner's native infrastructure.

### Quick Start

#### Basic Installation
```bash
# Install with default settings
docker model k8s install

# Check the status
docker model k8s status

# Access the service (requires port-forward)
kubectl port-forward deployment/docker-model-runner 31245:12434
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest
```

#### Installation with GPU Support
```bash
# Install with NVIDIA GPU support
docker model k8s install --gpu --gpu-vendor nvidia --gpu-count 1

# Install with AMD GPU support
docker model k8s install --gpu --gpu-vendor amd --gpu-count 1
```

#### Installation with Model Pre-pulling
```bash
# Pre-pull models during pod initialization
docker model k8s install --models ai/smollm2:latest,ai/llama3.2:latest
```

#### Installation with Custom Storage
```bash
# Configure storage size and class
docker model k8s install --storage-size 200Gi --storage-class fast-ssd
```

#### Installation for Docker Desktop
```bash
# Enable NodePort for easy access
docker model k8s install --node-port

# Access directly without port-forward
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest
```

### Using Helm
```bash
# Install using Helm (requires Helm to be installed)
docker model k8s install --helm

# Uninstall using Helm
docker model k8s uninstall --helm
```

### Complete Example
```bash
# Deploy with GPU, model pre-pulling, and custom storage
docker model k8s install \
  --gpu \
  --gpu-vendor nvidia \
  --gpu-count 1 \
  --models ai/smollm2:latest \
  --storage-size 150Gi \
  --storage-class gp2 \
  --namespace my-models

# Check status
docker model k8s status --namespace my-models

# Test the deployment
kubectl port-forward -n my-models deployment/docker-model-runner 31245:12434
MODEL_RUNNER_HOST=http://localhost:31245 docker model list
MODEL_RUNNER_HOST=http://localhost:31245 docker model run ai/smollm2:latest "Tell me about Kubernetes"

# Clean up
docker model k8s uninstall --namespace my-models
```

### Key Features

- **No External Dependencies**: Uses Docker Model Runner's native Helm charts - no need for llm-d or aibrix
- **GPU Support**: Built-in support for NVIDIA and AMD GPUs with automatic resource allocation
- **Model Pre-pulling**: Automatically download models during pod initialization
- **Flexible Storage**: Configure storage size and class for your cloud provider
- **Easy Access**: NodePort support for Docker Desktop, or standard port-forwarding for production
- **Helm or kubectl**: Choose between Helm-based deployment or plain Kubernetes manifests

## Advanced
- **Packaging:**
  Add licenses and set context size when packaging models for distribution.

## Development
- **Run unit tests:**
  ```bash
  make unit-tests
  ```
- **Generate docs:**
  ```bash
  make docs
  ```

## License
[Apache 2.0](LICENSE)

