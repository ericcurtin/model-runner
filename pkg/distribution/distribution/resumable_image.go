package distribution

import (
	"context"
	"net/http"

	"github.com/docker/model-runner/pkg/distribution/internal/store"
	"github.com/docker/model-runner/pkg/distribution/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// resumableImage wraps a v1.Image and returns ResumableLayers
type resumableImage struct {
	v1.Image
	store      *store.LocalStore
	httpClient *http.Client
	reference  string
	registry   *registry.Client
	ctx        context.Context
}

// Layers returns wrapped layers with resumable download support
func (ri *resumableImage) Layers() ([]v1.Layer, error) {
	layers, err := ri.Image.Layers()
	if err != nil {
		return nil, err
	}

	// Wrap each layer with resumable support
	wrapped := make([]v1.Layer, len(layers))
	for i, layer := range layers {
		// Get digest for this layer
		digest, err := layer.Digest()
		if err != nil {
			// If we can't get digest, just use original layer
			wrapped[i] = layer
			continue
		}

		// Get blob URL for this layer
		blobURL, err := ri.registry.BlobURL(ri.reference, digest)
		if err != nil {
			// If we can't get URL, just use original layer
			wrapped[i] = layer
			continue
		}

		// Get auth token
		authToken, err := ri.registry.BearerToken(ri.ctx, ri.reference)
		if err != nil {
			// If we can't get auth token, try without it
			authToken = ""
		}

		// Create resumable layer
		wrapped[i] = store.NewResumableLayer(layer, ri.store, ri.httpClient, blobURL, authToken)
	}

	return wrapped, nil
}

// wrapWithResumableImage wraps an image to support resumable downloads
func (c *Client) wrapWithResumableImage(ctx context.Context, img v1.Image, reference string, transport http.RoundTripper) v1.Image {
	return &resumableImage{
		Image:      img,
		store:      c.store,
		httpClient: &http.Client{Transport: transport},
		reference:  reference,
		registry:   c.registry,
		ctx:        ctx,
	}
}
