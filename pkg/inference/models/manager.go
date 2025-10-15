package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/model-runner/pkg/diskusage"
	"github.com/docker/model-runner/pkg/distribution/distribution"
	"github.com/docker/model-runner/pkg/distribution/registry"
	"github.com/docker/model-runner/pkg/distribution/types"
	"github.com/docker/model-runner/pkg/inference"
	"github.com/docker/model-runner/pkg/inference/memory"
	"github.com/docker/model-runner/pkg/logging"
	"github.com/docker/model-runner/pkg/middleware"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"
)

const (
	// maximumConcurrentModelPulls is the maximum number of concurrent model
	// pulls that a model manager will allow.
	maximumConcurrentModelPulls = 2
	defaultOrg                  = "ai"
	defaultTag                  = "latest"
)

// Manager manages inference model pulls and storage.
type Manager struct {
	// log is the associated logger.
	log logging.Logger
	// pullTokens is a semaphore used to restrict the maximum number of
	// concurrent pull requests.
	pullTokens chan struct{}
	// router is the HTTP request router.
	router *http.ServeMux
	// httpHandler is the HTTP request handler, which wraps router with
	// the server-level middleware.
	httpHandler http.Handler
	// distributionClient is the client for model distribution.
	distributionClient *distribution.Client
	// registryClient is the client for model registry.
	registryClient *registry.Client
	// lock is used to synchronize access to the models manager's router.
	lock sync.RWMutex
	// memoryEstimator is used to calculate runtime memory requirements for models.
	memoryEstimator memory.MemoryEstimator
}

type ClientConfig struct {
	// StoreRootPath is the root path for the model store.
	StoreRootPath string
	// Logger is the logger to use.
	Logger *logrus.Entry
	// Transport is the HTTP transport to use.
	Transport http.RoundTripper
	// UserAgent is the user agent to use.
	UserAgent string
}

// NewManager creates a new model's manager.
func NewManager(log logging.Logger, c ClientConfig, allowedOrigins []string, memoryEstimator memory.MemoryEstimator) *Manager {
	// Create the model distribution client.
	distributionClient, err := distribution.NewClient(
		distribution.WithStoreRootPath(c.StoreRootPath),
		distribution.WithLogger(c.Logger),
		distribution.WithTransport(c.Transport),
		distribution.WithUserAgent(c.UserAgent),
	)
	if err != nil {
		log.Errorf("Failed to create distribution client: %v", err)
		// Continue without distribution client. The model manager will still
		// respond to requests, but may return errors if the client is required.
	}

	// Create the model registry client.
	registryClient := registry.NewClient(
		registry.WithTransport(c.Transport),
		registry.WithUserAgent(c.UserAgent),
	)

	// Create the manager.
	m := &Manager{
		log:                log,
		pullTokens:         make(chan struct{}, maximumConcurrentModelPulls),
		router:             http.NewServeMux(),
		distributionClient: distributionClient,
		registryClient:     registryClient,
		memoryEstimator:    memoryEstimator,
	}

	// Register routes.
	m.router.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	for route, handler := range m.routeHandlers() {
		m.router.HandleFunc(route, handler)
	}

	m.RebuildRoutes(allowedOrigins)

	// Populate the pull concurrency semaphore.
	for i := 0; i < maximumConcurrentModelPulls; i++ {
		m.pullTokens <- struct{}{}
	}

	// Manager successfully initialized.
	return m
}

func (m *Manager) RebuildRoutes(allowedOrigins []string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	// Update handlers that depend on the allowed origins.
	m.httpHandler = middleware.CorsMiddleware(allowedOrigins, m.router)
}

// normalizeModelName adds the default organization prefix (ai/) and tag (:latest) if missing.
func normalizeModelName(model string) string {
	// If the model is empty, return as-is
	if model == "" {
		return model
	}

	// Normalize HuggingFace model names (lowercase)
	if strings.HasPrefix(model, "hf.co/") {
		model = strings.ToLower(model)
	}

	// If model starts with "hf.co/" or contains a registry (has a dot before the first slash),
	// don't add default org
	if strings.HasPrefix(model, "hf.co/") {
		// For HuggingFace models, just ensure :latest tag if no tag specified
		if !strings.Contains(model, ":") {
			return model + ":" + defaultTag
		}
		return model
	}

	// Check if model contains a registry (domain with dot before first slash)
	firstSlash := strings.Index(model, "/")
	if firstSlash > 0 && strings.Contains(model[:firstSlash], ".") {
		// Has a registry, just ensure tag
		if !strings.Contains(model, ":") {
			return model + ":" + defaultTag
		}
		return model
	}

	// Split by colon to check for tag
	parts := strings.SplitN(model, ":", 2)
	nameWithOrg := parts[0]
	tag := defaultTag
	if len(parts) == 2 {
		tag = parts[1]
	}

	// If name doesn't contain a slash, add the default org
	if !strings.Contains(nameWithOrg, "/") {
		nameWithOrg = defaultOrg + "/" + nameWithOrg
	}

	return nameWithOrg + ":" + tag
}

func (m *Manager) routeHandlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"POST " + inference.ModelsPrefix + "/create":                          m.handleCreateModel,
		"POST " + inference.ModelsPrefix + "/load":                            m.handleLoadModel,
		"GET " + inference.ModelsPrefix:                                       m.handleGetModels,
		"GET " + inference.ModelsPrefix + "/{name...}":                        m.handleGetModel,
		"DELETE " + inference.ModelsPrefix + "/{name...}":                     m.handleDeleteModel,
		"POST " + inference.ModelsPrefix + "/{nameAndAction...}":              m.handleModelAction,
		"GET " + inference.InferencePrefix + "/{backend}/v1/models":           m.handleOpenAIGetModels,
		"GET " + inference.InferencePrefix + "/{backend}/v1/models/{name...}": m.handleOpenAIGetModel,
		"GET " + inference.InferencePrefix + "/v1/models":                     m.handleOpenAIGetModels,
		"GET " + inference.InferencePrefix + "/v1/models/{name...}":           m.handleOpenAIGetModel,
	}
}

// handleCreateModel handles POST <inference-prefix>/models/create requests.
func (m *Manager) handleCreateModel(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Decode the request.
	var request ModelCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Normalize the model name to add defaults
	request.From = normalizeModelName(request.From)

	// Pull the model. In the future, we may support additional operations here
	// besides pulling (such as model building).
	if memory.RuntimeMemoryCheckEnabled() && !request.IgnoreRuntimeMemoryCheck {
		m.log.Infof("Will estimate memory required for %q", request.From)
		proceed, req, totalMem, err := m.memoryEstimator.HaveSufficientMemoryForModel(r.Context(), request.From, nil)
		if err != nil {
			m.log.Warnf("Failed to validate sufficient system memory for model %q: %s", request.From, err)
			// Prefer staying functional in case of unexpected estimation errors.
			proceed = true
		}
		if !proceed {
			errstr := fmt.Sprintf("Runtime memory requirement for model %q exceeds total system memory: required %d RAM %d VRAM, system %d RAM %d VRAM", request.From, req.RAM, req.VRAM, totalMem.RAM, totalMem.VRAM)
			m.log.Warnf(errstr)
			http.Error(w, errstr, http.StatusInsufficientStorage)
			return
		}
	}
	if err := m.PullModel(request.From, r, w); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			m.log.Infof("Request canceled/timed out while pulling model %q", request.From)
			return
		}
		if errors.Is(err, registry.ErrInvalidReference) {
			m.log.Warnf("Invalid model reference %q: %v", request.From, err)
			http.Error(w, "Invalid model reference", http.StatusBadRequest)
			return
		}
		if errors.Is(err, registry.ErrUnauthorized) {
			m.log.Warnf("Unauthorized to pull model %q: %v", request.From, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, registry.ErrModelNotFound) {
			m.log.Warnf("Failed to pull model %q: %v", request.From, err)
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, distribution.ErrUnsupportedFormat) {
			m.log.Warnf("Unsupported model format for %q: %v", request.From, err)
			http.Error(w, distribution.ErrUnsupportedFormat.Error(), http.StatusUnsupportedMediaType)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleLoadModel handles POST <inference-prefix>/models/load requests.
func (m *Manager) handleLoadModel(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	if _, err := m.distributionClient.LoadModel(r.Body, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// handleGetModels handles GET <inference-prefix>/models requests.
func (m *Manager) handleGetModels(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Query models.
	models, err := m.distributionClient.ListModels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	apiModels := make([]*Model, len(models))
	for i, model := range models {
		apiModels[i], err = ToModel(model)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Write the response.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apiModels); err != nil {
		m.log.Warnln("Error while encoding model listing response:", err)
	}
}

// handleGetModel handles GET <inference-prefix>/models/{name} requests.
func (m *Manager) handleGetModel(w http.ResponseWriter, r *http.Request) {
	// Normalize model name
	modelName := normalizeModelName(r.PathValue("name"))
	
	// Parse remote query parameter
	remote := false
	if r.URL.Query().Has("remote") {
		if val, err := strconv.ParseBool(r.URL.Query().Get("remote")); err != nil {
			m.log.Warnln("Error while parsing remote query parameter:", err)
		} else {
			remote = val
		}
	}

	if remote && m.registryClient == nil {
		http.Error(w, "registry client unavailable", http.StatusServiceUnavailable)
		return
	}

	var apiModel *Model
	var err error

	if remote {
		apiModel, err = getRemoteModel(r.Context(), m, modelName)
	} else {
		apiModel, err = getLocalModel(m, modelName)
	}

	if err != nil {
		if errors.Is(err, distribution.ErrModelNotFound) || errors.Is(err, registry.ErrModelNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the response.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apiModel); err != nil {
		m.log.Warnln("Error while encoding model response:", err)
	}
}

// ResolveModelID resolves a model reference to a model ID. If resolution fails, it returns the original ref.
func (m *Manager) ResolveModelID(modelRef string) string {
	// Sanitize modelRef to prevent log forgery
	sanitizedModelRef := strings.ReplaceAll(modelRef, "\n", "")
	sanitizedModelRef = strings.ReplaceAll(sanitizedModelRef, "\r", "")

	model, err := m.GetModel(sanitizedModelRef)
	if err != nil {
		m.log.Warnf("Failed to resolve model ref %s to ID: %v", sanitizedModelRef, err)
		return sanitizedModelRef
	}

	modelID, err := model.ID()
	if err != nil {
		m.log.Warnf("Failed to get model ID for ref %s: %v", sanitizedModelRef, err)
		return sanitizedModelRef
	}

	return modelID
}

func getLocalModel(m *Manager, name string) (*Model, error) {
	if m.distributionClient == nil {
		return nil, errors.New("model distribution service unavailable")
	}

	// Query the model.
	model, err := m.GetModel(name)
	if err != nil {
		return nil, err
	}

	return ToModel(model)
}

func getRemoteModel(ctx context.Context, m *Manager, name string) (*Model, error) {
	if m.registryClient == nil {
		return nil, errors.New("registry client unavailable")
	}

	m.log.Infoln("Getting remote model:", name)
	model, err := m.registryClient.Model(ctx, name)
	if err != nil {
		return nil, err
	}

	id, err := model.ID()
	if err != nil {
		return nil, err
	}

	descriptor, err := model.Descriptor()
	if err != nil {
		return nil, err
	}

	config, err := model.Config()
	if err != nil {
		return nil, err
	}

	apiModel := &Model{
		ID:      id,
		Tags:    nil,
		Created: descriptor.Created.Unix(),
		Config:  config,
	}

	return apiModel, nil
}

// handleDeleteModel handles DELETE <inference-prefix>/models/{name} requests.
// query params:
// - force: if true, delete the model even if it has multiple tags
func (m *Manager) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// TODO: We probably want the manager to have a lock / unlock mechanism for
	// models so that active runners can retain / release a model, analogous to
	// a container blocking the release of an image. However, unlike containers,
	// runners are only evicted when idle or when memory is needed, so users
	// won't be able to release the images manually. Perhaps we can unlink the
	// corresponding GGUF files from disk and allow the OS to clean them up once
	// the runner process exits (though this won't work for Windows, where we
	// might need some separate cleanup process).

	// Normalize model name
	modelName := normalizeModelName(r.PathValue("name"))
	
	var force bool
	if r.URL.Query().Has("force") {
		if val, err := strconv.ParseBool(r.URL.Query().Get("force")); err != nil {
			m.log.Warnln("Error while parsing force query parameter:", err)
		} else {
			force = val
		}
	}

	resp, err := m.distributionClient.DeleteModel(modelName, force)
	if err != nil {
		if errors.Is(err, distribution.ErrModelNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, distribution.ErrConflict) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		m.log.Warnln("Error while deleting model:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("error writing response: %v", err), http.StatusInternalServerError)
	}
}

// handleOpenAIGetModels handles GET <inference-prefix>/<backend>/v1/models and
// GET /<inference-prefix>/v1/models requests.
func (m *Manager) handleOpenAIGetModels(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Query models.
	available, err := m.distributionClient.ListModels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	models, err := ToOpenAIList(available)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the response.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		m.log.Warnln("Error while encoding OpenAI model listing response:", err)
	}
}

// handleOpenAIGetModel handles GET <inference-prefix>/<backend>/v1/models/{name}
// and GET <inference-prefix>/v1/models/{name} requests.
func (m *Manager) handleOpenAIGetModel(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Normalize model name
	modelName := normalizeModelName(r.PathValue("name"))

	// Query the model.
	model, err := m.GetModel(modelName)
	if err != nil {
		if errors.Is(err, distribution.ErrModelNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Write the response.
	w.Header().Set("Content-Type", "application/json")
	openaiModel, err := ToOpenAI(model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(openaiModel); err != nil {
		m.log.Warnln("Error while encoding OpenAI model response:", err)
	}
}

// handleTagModel handles POST <inference-prefix>/models/{nameAndAction} requests.
// Action is one of:
// - tag: tag the model with a repository and tag (e.g. POST <inference-prefix>/models/my-org/my-repo:latest/tag})
// - push: pushes a tagged model to the registry
func (m *Manager) handleModelAction(w http.ResponseWriter, r *http.Request) {
	model, action := path.Split(r.PathValue("nameAndAction"))
	model = strings.TrimRight(model, "/")
	// Normalize model name
	model = normalizeModelName(model)
	switch action {
	case "tag":
		m.handleTagModel(w, r, model)
	case "push":
		m.handlePushModel(w, r, model)
	default:
		http.Error(w, fmt.Sprintf("unknown action %q", action), http.StatusNotFound)
	}
}

// handleTagModel handles POST <inference-prefix>/models/{name}/tag requests.
// The query parameters are:
// - repo: the repository to tag the model with (required)
// - tag: the tag to apply to the model (required)
func (m *Manager) handleTagModel(w http.ResponseWriter, r *http.Request, model string) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Extract query parameters.
	repo := r.URL.Query().Get("repo")
	tag := r.URL.Query().Get("tag")

	// Validate query parameters.
	if repo == "" || tag == "" {
		http.Error(w, "missing repo or tag query parameter", http.StatusBadRequest)
		return
	}

	// Construct the target string.
	target := fmt.Sprintf("%s:%s", repo, tag)

	// Call the Tag method on the distribution client with source and modelName.
	if err := m.distributionClient.Tag(model, target); err != nil {
		m.log.Warnf("Failed to apply tag %q to model %q: %v", target, model, err)

		if errors.Is(err, distribution.ErrModelNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with success.
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("Model %q tagged successfully with %q", model, target)))
}

// handlePushModel handles POST <inference-prefix>/models/{name}/push requests.
func (m *Manager) handlePushModel(w http.ResponseWriter, r *http.Request, model string) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Call the PushModel method on the distribution client.
	if err := m.PushModel(model, r, w); err != nil {
		if errors.Is(err, distribution.ErrInvalidReference) {
			m.log.Warnf("Invalid model reference %q: %v", model, err)
			http.Error(w, "Invalid model reference", http.StatusBadRequest)
			return
		}
		if errors.Is(err, distribution.ErrModelNotFound) {
			m.log.Warnf("Failed to push model %q: %v", model, err)
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, registry.ErrUnauthorized) {
			m.log.Warnf("Unauthorized to push model %q: %v", model, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetDiskUsage returns the disk usage of the model store.
func (m *Manager) GetDiskUsage() (int64, error, int) {
	if m.distributionClient == nil {
		return 0, errors.New("model distribution service unavailable"), http.StatusServiceUnavailable
	}

	storePath := m.distributionClient.GetStorePath()
	size, err := diskusage.Size(storePath)
	if err != nil {
		return 0, fmt.Errorf("error while getting store size: %v", err), http.StatusInternalServerError
	}

	return size, nil, http.StatusOK
}

// ServeHTTP implement net/http.Handler.ServeHTTP.
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	m.httpHandler.ServeHTTP(w, r)
}

// IsModelInStore checks if a given model is in the local store.
func (m *Manager) IsModelInStore(ref string) (bool, error) {
	return m.distributionClient.IsModelInStore(ref)
}

// GetModel returns a single model.
func (m *Manager) GetModel(ref string) (types.Model, error) {
	model, err := m.distributionClient.GetModel(ref)
	if err != nil {
		return nil, fmt.Errorf("error while getting model: %w", err)
	}
	return model, err
}

// GetRemoteModel returns a single remote model.
func (m *Manager) GetRemoteModel(ctx context.Context, ref string) (types.ModelArtifact, error) {
	model, err := m.registryClient.Model(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("error while getting remote model: %w", err)
	}
	return model, nil
}

// GetRemoteModelBlobURL returns the URL of a given model blob.
func (m *Manager) GetRemoteModelBlobURL(ref string, digest v1.Hash) (string, error) {
	blobURL, err := m.registryClient.BlobURL(ref, digest)
	if err != nil {
		return "", fmt.Errorf("error while getting remote model blob URL: %w", err)
	}
	return blobURL, nil
}

// BearerTokenForModel returns the bearer token needed to pull a given model.
func (m *Manager) BearerTokenForModel(ctx context.Context, ref string) (string, error) {
	tok, err := m.registryClient.BearerToken(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("error while getting bearer token for model: %w", err)
	}
	return tok, nil
}

// GetBundle returns model bundle.
func (m *Manager) GetBundle(ref string) (types.ModelBundle, error) {
	bundle, err := m.distributionClient.GetBundle(ref)
	if err != nil {
		return nil, fmt.Errorf("error while getting model bundle: %w", err)
	}
	return bundle, err
}

// PullModel pulls a model to local storage. Any error it returns is suitable
// for writing back to the client.
func (m *Manager) PullModel(model string, r *http.Request, w http.ResponseWriter) error {
	// Restrict model pull concurrency.
	select {
	case <-m.pullTokens:
	case <-r.Context().Done():
		return context.Canceled
	}
	defer func() {
		m.pullTokens <- struct{}{}
	}()

	// Set up response headers for streaming
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Check Accept header to determine content type
	acceptHeader := r.Header.Get("Accept")
	isJSON := acceptHeader == "application/json"

	if isJSON {
		w.Header().Set("Content-Type", "application/json")
	} else {
		// Defaults to text/plain
		w.Header().Set("Content-Type", "text/plain")
	}

	// Create a flusher to ensure chunks are sent immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// Create a progress writer that writes to the response
	progressWriter := &progressResponseWriter{
		writer:  w,
		flusher: flusher,
		isJSON:  isJSON,
	}

	// Pull the model using the Docker model distribution client
	m.log.Infoln("Pulling model:", model)
	err := m.distributionClient.PullModel(r.Context(), model, progressWriter)
	if err != nil {
		return fmt.Errorf("error while pulling model: %w", err)
	}

	return nil
}

// PushModel pushes a model from the store to the registry.
func (m *Manager) PushModel(model string, r *http.Request, w http.ResponseWriter) error {
	// Set up response headers for streaming
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Check Accept header to determine content type
	acceptHeader := r.Header.Get("Accept")
	isJSON := acceptHeader == "application/json"

	if isJSON {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "text/plain")
	}

	// Create a flusher to ensure chunks are sent immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming not supported")
	}

	// Create a progress writer that writes to the response
	progressWriter := &progressResponseWriter{
		writer:  w,
		flusher: flusher,
		isJSON:  isJSON,
	}

	// Pull the model using the Docker model distribution client
	m.log.Infoln("Pushing model:", model)
	err := m.distributionClient.PushModel(r.Context(), model, progressWriter)
	if err != nil {
		return fmt.Errorf("error while pushing model: %w", err)
	}

	return nil
}

// progressResponseWriter implements io.Writer to write progress updates to the HTTP response
type progressResponseWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	isJSON  bool
}

func (w *progressResponseWriter) Write(p []byte) (n int, err error) {
	var data []byte
	if w.isJSON {
		// For JSON, write the raw bytes without escaping
		data = p
	} else {
		// For plain text, escape HTML
		escapedData := html.EscapeString(string(p))
		data = []byte(escapedData)
	}

	n, err = w.writer.Write(data)
	if err != nil {
		return 0, err
	}
	// Flush the response to ensure the chunk is sent immediately
	if w.flusher != nil {
		w.flusher.Flush()
	}
	return n, nil
}
