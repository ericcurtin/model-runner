// Package swagger provides a Swagger UI handler for API documentation.
package swagger

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed openapi.yaml
var openapiSpec []byte

// Handler serves the Swagger UI and OpenAPI specification.
type Handler struct{}

// NewHandler creates a new Swagger UI handler.
func NewHandler() *Handler {
	return &Handler{}
}

// ServeHTTP serves the Swagger UI or OpenAPI specification.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	switch path {
	case "", "index.html":
		h.serveSwaggerUI(w, r)
	case "openapi.yaml":
		h.serveOpenAPISpec(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveOpenAPISpec serves the OpenAPI specification file.
func (h *Handler) serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(openapiSpec)
}

// serveSwaggerUI serves the Swagger UI HTML page.
func (h *Handler) serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerUIHTML))
}

// swaggerUIHTML is the HTML template for Swagger UI.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Docker Model Runner API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *,
        *:before,
        *:after {
            box-sizing: inherit;
        }
        body {
            margin: 0;
            background: #fafafa;
        }
        .swagger-ui .topbar {
            background-color: #1d63ed;
        }
        .swagger-ui .topbar .download-url-wrapper .select-label {
            display: flex;
            align-items: center;
            width: 100%;
            max-width: 600px;
        }
        .swagger-ui .info .title {
            color: #1d63ed;
        }
        .swagger-ui .opblock.opblock-post {
            border-color: #49cc90;
            background: rgba(73, 204, 144, .1);
        }
        .swagger-ui .opblock.opblock-get {
            border-color: #61affe;
            background: rgba(97, 175, 254, .1);
        }
        .swagger-ui .opblock.opblock-delete {
            border-color: #f93e3e;
            background: rgba(249, 62, 62, .1);
        }
        .custom-header {
            background: linear-gradient(135deg, #1d63ed 0%, #0d47a1 100%);
            color: white;
            padding: 20px 40px;
            text-align: center;
        }
        .custom-header h1 {
            margin: 0 0 10px 0;
            font-size: 28px;
            font-weight: 600;
        }
        .custom-header p {
            margin: 0;
            opacity: 0.9;
            font-size: 14px;
        }
        .custom-header a {
            color: white;
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="custom-header">
        <h1>🐳 Docker Model Runner API</h1>
        <p>Run AI models locally with OpenAI, Anthropic, and Ollama compatible APIs</p>
    </div>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "/openapi.yaml",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                docExpansion: "list",
                filter: true,
                showExtensions: true,
                showCommonExtensions: true,
                tryItOutEnabled: true
            });
        };
    </script>
</body>
</html>
`
