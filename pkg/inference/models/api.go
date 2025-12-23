package models

import (
	"fmt"

	"github.com/docker/model-runner/pkg/distribution/types"
)

// ModelCreateRequest represents a model create request. It is designed to
// follow Docker Engine API conventions, most closely following the request
// associated with POST /images/create. At the moment is only designed to
// facilitate pulls, though in the future it may facilitate model building and
// refinement (such as fine tuning, quantization, or distillation).
type ModelCreateRequest struct {
	// From is the name of the model to pull.
	From string `json:"from"`
	// IgnoreRuntimeMemoryCheck indicates whether the server should check if it has sufficient
	// memory to run the given model (assuming default configuration).
	IgnoreRuntimeMemoryCheck bool `json:"ignore-runtime-memory-check,omitempty"`
	// BearerToken is an optional bearer token for authentication.
	BearerToken string `json:"bearer-token,omitempty"`
}

// ModelPackageRequest represents a model package request, which creates a new model
// from an existing one with modified properties (e.g., context size).
type ModelPackageRequest struct {
	// From is the name of the source model to package from.
	From string `json:"from"`
	// Tag is the name to give the new packaged model.
	Tag string `json:"tag"`
	// ContextSize specifies the context size to set for the new model.
	ContextSize uint64 `json:"context-size,omitempty"`
}

// SimpleModel is a wrapper that allows creating a model with modified configuration
type SimpleModel struct {
	types.Model
	ConfigValue     types.Config
	DescriptorValue types.Descriptor
}

func (s *SimpleModel) Config() (types.Config, error) {
	return s.ConfigValue, nil
}

func (s *SimpleModel) Descriptor() (types.Descriptor, error) {
	return s.DescriptorValue, nil
}

// ToOpenAIList converts the model list to its OpenAI API representation. This function never
// returns a nil slice (though it may return an empty slice).
func ToOpenAIList(l []types.Model) (*OpenAIModelList, error) {
	// Convert the constituent models.
	models := make([]*OpenAIModel, len(l))
	for i, model := range l {
		openAI, err := ToOpenAI(model)
		if err != nil {
			return nil, fmt.Errorf("convert model: %w", err)
		}
		models[i] = openAI
	}

	// Create the OpenAI model list.
	return &OpenAIModelList{
		Object: "list",
		Data:   models,
	}, nil
}

// ToOpenAI converts a types.Model to its OpenAI API representation.
func ToOpenAI(m types.Model) (*OpenAIModel, error) {
	desc, err := m.Descriptor()
	if err != nil {
		return nil, fmt.Errorf("get descriptor: %w", err)
	}

	created := int64(0)
	if desc.Created != nil {
		created = desc.Created.Unix()
	}

	id, err := m.ID()
	if err != nil {
		return nil, fmt.Errorf("get model ID: %w", err)
	}
	if tags := m.Tags(); len(tags) > 0 {
		id = tags[0]
	}

	cfg, err := m.Config()
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	// Build the OpenAI model with metadata
	result := &OpenAIModel{
		ID:           id,
		Object:       "model",
		Created:      created,
		OwnedBy:      "docker",
		Architecture: cfg.Architecture,
		Quantization: cfg.Quantization,
		Parameters:   cfg.Parameters,
	}

	// Add format if available
	if cfg.Format != "" {
		result.Format = string(cfg.Format)
	}

	// Add context length if available
	if cfg.ContextSize != nil {
		result.ContextLength = cfg.ContextSize
	}

	return result, nil
}

// OpenAIModel represents a locally stored model using OpenAI conventions.
type OpenAIModel struct {
	// ID is the model tag.
	ID string `json:"id"`
	// Object is the object type. For OpenAIModel, it is always "model".
	Object string `json:"object"`
	// Created is the Unix epoch timestamp corresponding to the model creation.
	Created int64 `json:"created"`
	// OwnedBy is the model owner. At the moment, it is always "docker".
	OwnedBy string `json:"owned_by"`
	// Architecture is the model architecture (e.g., "llama", "mistral").
	Architecture string `json:"architecture,omitempty"`
	// ContextLength is the maximum context window size in tokens.
	ContextLength *uint64 `json:"context_length,omitempty"`
	// Format is the model format (e.g., "gguf", "safetensors").
	Format string `json:"format,omitempty"`
	// Quantization is the quantization method used (e.g., "Q4_K_M", "Q8_0").
	Quantization string `json:"quantization,omitempty"`
	// Parameters is the approximate number of parameters (e.g., "7B", "13B").
	Parameters string `json:"parameters,omitempty"`
}

// OpenAIModelList represents a list of models using OpenAI conventions.
type OpenAIModelList struct {
	// Object is the object type. For OpenAIModelList, it is always "list".
	Object string `json:"object"`
	// Data is the list of models.
	Data []*OpenAIModel `json:"data"`
}

type Model struct {
	// ID is the globally unique model identifier.
	ID string `json:"id"`
	// Tags are the list of tags associated with the model.
	Tags []string `json:"tags,omitempty"`
	// Created is the Unix epoch timestamp corresponding to the model creation.
	Created int64 `json:"created"`
	// Config describes the model.
	Config types.Config `json:"config"`
}
