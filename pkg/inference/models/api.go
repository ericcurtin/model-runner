package models

import (
	"bytes"
	"encoding/json"
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
	// BearerToken is an optional bearer token for authentication.
	BearerToken string `json:"bearer-token,omitempty"`
}

// SimpleModel is a wrapper that allows creating a model with modified configuration
type SimpleModel struct {
	types.Model
	ConfigValue     types.ModelConfig
	DescriptorValue types.Descriptor
}

func (s *SimpleModel) Config() (types.ModelConfig, error) {
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

	return &OpenAIModel{
		ID:      id,
		Object:  "model",
		Created: created,
		OwnedBy: "docker",
	}, nil
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
}

// OpenAIModelList represents a list of models using OpenAI conventions.
type OpenAIModelList struct {
	// Object is the object type. For OpenAIModelList, it is always "list".
	Object string `json:"object"`
	// Data is the list of models.
	Data []*OpenAIModel `json:"data"`
}

// ModelConfigWrapper wraps the types.ModelConfig interface to handle JSON marshaling/unmarshaling
type ModelConfigWrapper struct {
	ModelConfig types.ModelConfig
}

// UnmarshalJSON implements json.Unmarshaler for ModelConfigWrapper
func (m *ModelConfigWrapper) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as the concrete Config type first
	var config types.Config
	if err := json.Unmarshal(data, &config); err == nil {
		m.ModelConfig = &config
		return nil
	}

	// If that fails, we could try other possible types that implement ModelConfig
	// For now, we'll return an error since we expect the types.Config format
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Create a Config from the raw data by marshaling back and unmarshaling as Config
	jsonData, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	var config2 types.Config // Changed variable name to avoid conflict
	if err := json.Unmarshal(jsonData, &config2); err != nil {
		return err
	}

	m.ModelConfig = &config2
	return nil
}

// MarshalJSON implements json.Marshaler for ModelConfigWrapper
func (m ModelConfigWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.ModelConfig)
}

// GetFormat returns the model format.
func (m ModelConfigWrapper) GetFormat() types.Format {
	if m.ModelConfig == nil {
		return ""
	}
	return m.ModelConfig.GetFormat()
}

// GetContextSize returns the context size configuration.
func (m ModelConfigWrapper) GetContextSize() *int32 {
	if m.ModelConfig == nil {
		return nil
	}
	return m.ModelConfig.GetContextSize()
}

// GetSize returns the parameter size (e.g., "8B").
func (m ModelConfigWrapper) GetSize() string {
	if m.ModelConfig == nil {
		return ""
	}
	return m.ModelConfig.GetSize()
}

// GetArchitecture returns the model architecture.
func (m ModelConfigWrapper) GetArchitecture() string {
	if m.ModelConfig == nil {
		return ""
	}
	return m.ModelConfig.GetArchitecture()
}

// GetParameters returns the parameters description.
func (m ModelConfigWrapper) GetParameters() string {
	if m.ModelConfig == nil {
		return ""
	}
	return m.ModelConfig.GetParameters()
}

// GetQuantization returns the quantization method.
func (m ModelConfigWrapper) GetQuantization() string {
	if m.ModelConfig == nil {
		return ""
	}
	return m.ModelConfig.GetQuantization()
}

type Model struct {
	// ID is the globally unique model identifier.
	ID string `json:"id"`
	// Tags are the list of tags associated with the model.
	Tags []string `json:"tags,omitempty"`
	// Created is the Unix epoch timestamp corresponding to the model creation.
	Created int64 `json:"created"`
	// Config describes the model. Can be either Docker format (*types.Config)
	// or ModelPack format (*modelpack.Model).
	Config *ModelConfigWrapper `json:"config"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Model.
// This is necessary because Config is an interface type (types.ModelConfig),
// and Go's standard JSON decoder cannot unmarshal directly into an interface.
// We use json.RawMessage to defer parsing of the config field, allowing for
// future extension to support multiple ModelConfig implementations.
func (m *Model) UnmarshalJSON(data []byte) error {
	type Alias Model
	aux := struct {
		*Alias
		Config json.RawMessage `json:"config"`
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(aux.Config) == 0 || bytes.Equal(aux.Config, []byte("null")) {
		m.Config = nil
		return nil
	}

	var cfg types.Config
	if err := json.Unmarshal(aux.Config, &cfg); err != nil {
		return err
	}

	wrapper := &ModelConfigWrapper{ModelConfig: &cfg}
	m.Config = wrapper

	return nil
}
