# NVIDIA NIM Demo

This demo shows how to use NVIDIA Inference Microservices (NIM) with Docker Model Runner.

## Prerequisites

1. NVIDIA GPU with appropriate drivers
2. Docker with GPU support
3. Access to NVIDIA NGC registry (nvcr.io)

## Setup

### 1. Authenticate with NVIDIA Registry

```bash
docker login nvcr.io
# Use your NGC API key as the password
```

### 2. Start a NIM Container

Choose a NIM model from the [NVIDIA NGC catalog](https://catalog.ngc.nvidia.com/containers?filters=&orderBy=weightPopularDESC&query=nim). For example, to run Google's Gemma 3 1B model:

```bash
# Pull the NIM container
docker pull nvcr.io/nim/google/gemma-3-1b-it:latest

# Run the NIM container with GPU support
docker run -d \
  --name nim-gemma \
  --gpus all \
  -p 8000:8000 \
  -e NGC_API_KEY=$NGC_API_KEY \
  nvcr.io/nim/google/gemma-3-1b-it:latest

# Wait for the container to be ready (check logs)
docker logs -f nim-gemma
# Wait until you see "Application startup complete"
```

### 3. Build Model Runner

```bash
# From the repository root
cd model-runner
make build

# Build the CLI
cd cmd/cli
make build
```

### 4. Start Model Runner

```bash
# From the repository root
./model-runner
# The server will start on port 12434 by default
```

### 5. Use the NIM Backend

In another terminal:

```bash
# Set environment variable to point to your model runner
export MODEL_RUNNER_HOST=http://localhost:12434

# Run a simple chat completion
cd cmd/cli
./model-cli run --backend nim nvcr.io/nim/google/gemma-3-1b-it:latest \
  "What is the capital of France?"

# Interactive chat mode
./model-cli run --backend nim nvcr.io/nim/google/gemma-3-1b-it:latest
# Type your prompts and press Enter
# Type /bye to exit
```

## Testing the NIM Endpoint Directly

You can also test the NIM container directly using curl:

```bash
# Check health
curl http://localhost:8000/health

# List models
curl http://localhost:8000/v1/models

# Chat completion
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "google/gemma-3-1b-it",
    "messages": [
      {"role": "user", "content": "What is AI?"}
    ],
    "max_tokens": 100
  }'
```

## Available NIM Models

Visit [NVIDIA NGC Catalog](https://catalog.ngc.nvidia.com/orgs/nim/teams/meta/containers/llama3-8b-instruct) for available NIM containers. Some popular options:

- `nvcr.io/nim/meta/llama3-8b-instruct`
- `nvcr.io/nim/google/gemma-3-1b-it`
- `nvcr.io/nim/mistralai/mistral-7b-instruct-v0.3`

## Troubleshooting

### NIM Container Not Starting

Check logs:
```bash
docker logs nim-gemma
```

Common issues:
- Insufficient GPU memory
- Missing NGC API key
- Network issues downloading model weights

### Model Runner Can't Connect to NIM

1. Verify NIM is running:
   ```bash
   curl http://localhost:8000/health
   ```

2. Check if port 8000 is accessible:
   ```bash
   docker ps | grep nim-gemma
   ```

3. Ensure model-runner is using the correct endpoint (currently hardcoded to localhost:8000)

### Backend Not Found

Make sure you're using the `--backend nim` flag:
```bash
./model-cli run --backend nim <model-name> "<prompt>"
```

## Cleanup

```bash
# Stop and remove the NIM container
docker stop nim-gemma
docker rm nim-gemma

# Remove the image (optional)
docker rmi nvcr.io/nim/google/gemma-3-1b-it:latest
```

## Limitations

Current implementation:
- NIM endpoint is hardcoded to `localhost:8000`
- Single NIM container supported
- NIM container must be started manually

Future enhancements will add:
- Automatic NIM container management
- Support for multiple concurrent NIM containers
- Configurable NIM endpoints
