package distribution

import (
	"errors"
	"fmt"

	"github.com/docker/model-runner/pkg/distribution/internal/store"
	"github.com/docker/model-runner/pkg/distribution/registry"
	"github.com/docker/model-runner/pkg/distribution/types"
)

var (
	ErrInvalidReference     = registry.ErrInvalidReference
	ErrModelNotFound        = store.ErrModelNotFound // model not found in store
	ErrUnsupportedMediaType = errors.New(fmt.Sprintf(
		"client supports only models of type %q and older - try upgrading",
		types.MediaTypeModelConfigV01,
	))
	ErrUnsupportedFormat = errors.New("vLLM backend currently only implemented for x86_64 Nvidia platforms")
	ErrConflict          = errors.New("resource conflict")
)
