# docker model pull

<!---MARKER_GEN_START-->
Pull a model from Docker Hub or HuggingFace to your local environment

### Options

| Name                            | Type   | Default | Description                                                                       |
|:--------------------------------|:-------|:--------|:----------------------------------------------------------------------------------|
| `--ignore-runtime-memory-check` | `bool` |         | Do not block pull if estimated runtime memory for model exceeds system resources. |


<!---MARKER_GEN_END-->

## Description

Pull a model to your local environment. Downloaded models also appear in the Docker Desktop Dashboard.

## Examples

### Pulling a model from Docker Hub

```console
docker model pull ai/smollm2
```

### Pulling from HuggingFace

You can pull GGUF models directly from [Hugging Face](https://huggingface.co/models?library=gguf).

**Note about quantization:** If no tag is specified, the command tries to pull the `Q4_K_M` version of the model.
If `Q4_K_M` doesn't exist, the command pulls the first GGUF found in the **Files** view of the model on HuggingFace.
To specify the quantization, provide it as a tag, for example:
`docker model pull hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF:Q4_K_S`

```console
docker model pull hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF
```

#### Known Limitations

**Sharded GGUF models:** Some models on HuggingFace use sharded GGUF format (where the model is split across multiple files). 
These models cannot currently be pulled directly from HuggingFace due to OCI registry limitations. 
If you encounter an error about "sharded GGUF", you have two options:

1. Use a non-sharded quantization of the same model if available
2. Upload the model to Docker Hub or another OCI-compliant registry and pull from there

For more information, see: https://github.com/ollama/ollama/issues/5245
