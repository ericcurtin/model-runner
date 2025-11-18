package middleware

import (
	"net/http"

	"github.com/docker/model-runner/pkg/inference"
)

// AliasHandler provides path aliasing by prepending the inference prefix to incoming request paths.
type AliasHandler struct {
	Handler http.Handler
}

func (h *AliasHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clone the request with modified path, prepending the inference prefix.
	r2 := r.Clone(r.Context())
	r2.URL.Path = inference.InferencePrefix + r.URL.Path

	h.Handler.ServeHTTP(w, r2)
}
