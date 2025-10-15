# docker model reinstall-runner

<!---MARKER_GEN_START-->
Reinstall Docker Model Runner (Docker Engine only)

### Options

| Name             | Type     | Default     | Description                                                                                        |
|:-----------------|:---------|:------------|:---------------------------------------------------------------------------------------------------|
| `--do-not-track` | `bool`   |             | Do not track models usage in Docker Model Runner                                                   |
| `--gpu`          | `string` | `auto`      | Specify GPU support (none\|auto\|cuda)                                                             |
| `--host`         | `string` | `127.0.0.1` | Host address to bind Docker Model Runner                                                           |
| `--port`         | `uint16` | `0`         | Docker container port for Docker Model Runner (default: 12434 for Docker CE, 12435 for Cloud mode) |


<!---MARKER_GEN_END-->

## Description

This command removes the existing Docker Model Runner container and reinstalls it with the specified configuration. Models and images are preserved during reinstallation.
