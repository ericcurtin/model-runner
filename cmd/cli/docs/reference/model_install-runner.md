# docker model install-runner

<!---MARKER_GEN_START-->
Install Docker Model Runner (Docker Engine only)

### Options

| Name             | Type     | Default     | Description                                                                                            |
|:-----------------|:---------|:------------|:-------------------------------------------------------------------------------------------------------|
| `--do-not-track` | `bool`   |             | Do not track models usage in Docker Model Runner                                                       |
| `--gpu`          | `string` | `auto`      | Specify GPU support (none\|auto\|cuda)                                                                 |
| `--host`         | `string` | `127.0.0.1` | Host address to bind Docker Model Runner                                                               |
| `--port`         | `uint16` | `0`         | Docker container port for Docker Model Runner (default: 12434 for Docker Engine, 12435 for Cloud mode) |


<!---MARKER_GEN_END-->

## Description

This command runs implicitly when a docker model command is executed. You can run this command explicitly to add a new configuration.

## Proxy Configuration

When running behind a corporate firewall or proxy, Docker Model Runner automatically passes proxy environment variables from the host to the container. Ensure that `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables are set in your environment before running this command.

Example:
```bash
export HTTP_PROXY=http://proxy.example.com:3128
export HTTPS_PROXY=http://proxy.example.com:3128
export NO_PROXY=localhost,127.0.0.1
docker model install-runner
```

The proxy settings will be inherited by the model-runner container, allowing it to pull models from registries through the proxy.
