package nim

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

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
// For NIM containers, this method:
// 1. Validates the model reference is a valid NIM container
// 2. Sets up a reverse proxy to forward requests to an external NIM endpoint
// 3. Blocks until the context is cancelled
func (n *NIM) Run(ctx context.Context, socket, model string, mode inference.BackendMode, config *inference.BackendConfiguration) error {
	if !IsNIMReference(model) {
		return fmt.Errorf("model %s does not appear to be a NIM reference", model)
	}

	n.log.Infof("Starting NIM backend proxy for model: %s", model)

	// For now, we assume the NIM container is already running externally
	// and accessible at a configured endpoint. In a full implementation,
	// this would use the Docker API to start/stop the NIM container.
	
	// Get the NIM endpoint from environment or use default
	nimEndpoint := getNIMEndpoint(model)
	
	// Parse the NIM endpoint URL
	targetURL, err := url.Parse(nimEndpoint)
	if err != nil {
		return fmt.Errorf("invalid NIM endpoint URL: %w", err)
	}

	// Create a reverse proxy to forward requests to the NIM container
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	
	// Customize the proxy director to handle path rewriting if needed
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// NIM containers expose OpenAI-compatible API at /v1/*
		// The model-runner will send requests to the Unix socket,
		// which we forward to the NIM container
	}

	// Create a Unix socket listener
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket listener: %w", err)
	}
	defer listener.Close()

	// Create HTTP server with the proxy handler
	server := &http.Server{
		Handler: proxy,
	}

	// Start the server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	n.log.Infof("NIM backend proxy started, forwarding requests to %s", nimEndpoint)

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		n.log.Info("Shutting down NIM backend proxy")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			n.log.Warnf("Error during NIM backend shutdown: %v", err)
		}
		return nil
	case err := <-serverErr:
		return fmt.Errorf("NIM backend server error: %w", err)
	}
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

// getNIMEndpoint returns the endpoint URL for a NIM container.
// This is a simplified implementation that assumes the NIM container
// is already running at a known endpoint.
// TODO: Implement actual Docker container management to start/stop NIM containers
func getNIMEndpoint(model string) string {
	// For now, return a placeholder endpoint
	// In a full implementation, this would:
	// 1. Check if a NIM container for this model is already running
	// 2. If not, pull and start the NIM container
	// 3. Return the container's endpoint (typically http://container-name:8000)
	
	// Default NIM container port is 8000
	return "http://localhost:8000"
}
