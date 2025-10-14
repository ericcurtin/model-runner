package nim

import (
	"context"
	"errors"
	"net/http"

	"github.com/docker/model-runner/pkg/inference"
	"github.com/docker/model-runner/pkg/inference/models"
	"github.com/docker/model-runner/pkg/logging"
)

const (
	// Name is the backend name.
	Name = "nim"
)

// NIM is the NVIDIA NIM-based backend implementation.
// NIM containers are pre-packaged inference microservices that expose OpenAI-compatible APIs.
type NIM struct {
	// log is the associated logger.
	log logging.Logger
	// modelManager is the shared model manager.
	modelManager *models.Manager
}

// New creates a new NIM-based backend.
func New(log logging.Logger, modelManager *models.Manager) (inference.Backend, error) {
	return &NIM{
		log:          log,
		modelManager: modelManager,
	}, nil
}

// Name implements inference.Backend.Name.
func (n *NIM) Name() string {
	return Name
}

// UsesExternalModelManagement implements
// inference.Backend.UsesExternalModelManagement.
// NIM containers manage models internally, so this returns true.
func (n *NIM) UsesExternalModelManagement() bool {
	return true
}

// Install implements inference.Backend.Install.
func (n *NIM) Install(ctx context.Context, httpClient *http.Client) error {
	// NIM containers are pre-built, no installation needed
	return nil
}

// Run implements inference.Backend.Run.
func (n *NIM) Run(ctx context.Context, socket, model string, mode inference.BackendMode, config *inference.BackendConfiguration) error {
	// TODO: Implement NIM container lifecycle management
	n.log.Warn("NIM backend is not yet fully implemented")
	return errors.New("not implemented")
}

// Status returns a description of the backend's state.
func (n *NIM) Status() string {
	return "not running"
}

// GetDiskUsage returns the disk usage of the backend.
func (n *NIM) GetDiskUsage() (int64, error) {
	// NIM containers are managed by Docker, disk usage is handled separately
	return 0, nil
}

// GetRequiredMemoryForModel returns the required working memory for a given model.
func (n *NIM) GetRequiredMemoryForModel(ctx context.Context, model string, config *inference.BackendConfiguration) (inference.RequiredMemory, error) {
	// NIM containers have their own memory management
	// We can't easily determine this without running the container
	return inference.RequiredMemory{}, errors.New("not implemented")
}
