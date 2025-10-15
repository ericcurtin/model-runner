package model

import "strings"

// IsOllamaModel determines if a model name refers to an Ollama model.
// Ollama models are identified by the "ollama.com/" prefix.
func IsOllamaModel(modelName string) bool {
	return strings.HasPrefix(modelName, "ollama.com/")
}

// StripOllamaPrefix removes the "ollama.com/" prefix from an Ollama model name.
// If the model is not an Ollama model, returns the original name.
func StripOllamaPrefix(modelName string) string {
	return strings.TrimPrefix(modelName, "ollama.com/")
}
