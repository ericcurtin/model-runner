package commands

import (
	"os"
	"testing"
)

func TestResolveServerURL(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		customURL   string
		port        int
		dmr         bool
		llamacpp    bool
		ollama      bool
		openrouter  bool
		expectURL   string
		expectOAI   bool
		wantErr     bool
		setupEnv    func()
		cleanupEnv  func()
	}{
		{
			name:      "no flags specified",
			expectURL: "",
			expectOAI: false,
			wantErr:   false,
		},
		{
			name:      "host and port specified",
			host:      "192.168.1.1",
			port:      8080,
			expectURL: "http://192.168.1.1:8080",
			expectOAI: false,
			wantErr:   false,
		},
		{
			name:      "only host specified",
			host:      "192.168.1.1",
			expectURL: "http://192.168.1.1:12434",
			expectOAI: false,
			wantErr:   false,
		},
		{
			name:      "only port specified",
			port:      8080,
			expectURL: "http://127.0.0.1:8080",
			expectOAI: false,
			wantErr:   false,
		},
		{
			name:      "dmr flag specified",
			dmr:       true,
			expectURL: "http://127.0.0.1:12434/engines/llama.cpp/v1",
			expectOAI: true,
			wantErr:   false,
		},
		{
			name:      "llamacpp flag specified",
			llamacpp:  true,
			expectURL: "http://127.0.0.1:8080/v1",
			expectOAI: true,
			wantErr:   false,
		},
		{
			name:      "ollama flag specified",
			ollama:    true,
			expectURL: "http://127.0.0.1:11434/v1",
			expectOAI: true,
			wantErr:   false,
		},
		{
			name:       "openrouter flag without API key",
			openrouter: true,
			wantErr:    true,
		},
		{
			name:       "openrouter flag with API key",
			openrouter: true,
			expectURL:  "https://openrouter.ai/api/v1",
			expectOAI:  true,
			wantErr:    false,
			setupEnv: func() {
				os.Setenv("OPENAI_API_KEY", "test-key")
			},
			cleanupEnv: func() {
				os.Unsetenv("OPENAI_API_KEY")
			},
		},
		{
			name:      "custom URL specified",
			customURL: "http://custom.server.com:9000/v1",
			expectURL: "http://custom.server.com:9000/v1",
			expectOAI: true,
			wantErr:   false,
		},
		{
			name:      "multiple preset flags (dmr + llamacpp)",
			dmr:       true,
			llamacpp:  true,
			wantErr:   true,
		},
		{
			name:      "multiple preset flags (url + ollama)",
			customURL: "http://test.com/v1",
			ollama:    true,
			wantErr:   true,
		},
		{
			name:      "host/port with preset flag (host + dmr)",
			host:      "192.168.1.1",
			dmr:       true,
			wantErr:   true,
		},
		{
			name:      "host/port with url",
			host:      "192.168.1.1",
			customURL: "http://test.com/v1",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			url, useOAI, apiKey, err := resolveServerURL(tt.host, tt.customURL, tt.port, tt.dmr, tt.llamacpp, tt.ollama, tt.openrouter)

			if (err != nil) != tt.wantErr {
				t.Errorf("resolveServerURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if url != tt.expectURL {
				t.Errorf("resolveServerURL() url = %v, want %v", url, tt.expectURL)
			}

			if useOAI != tt.expectOAI {
				t.Errorf("resolveServerURL() useOAI = %v, want %v", useOAI, tt.expectOAI)
			}

			// For openrouter, check that API key is returned
			if tt.openrouter && !tt.wantErr {
				if apiKey == "" {
					t.Errorf("resolveServerURL() expected API key for openrouter, got empty string")
				}
			}
		})
	}
}

func TestValidateBackend(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		wantErr bool
	}{
		{
			name:    "valid backend llama.cpp",
			backend: "llama.cpp",
			wantErr: false,
		},
		{
			name:    "valid backend openai",
			backend: "openai",
			wantErr: false,
		},
		{
			name:    "invalid backend",
			backend: "invalid",
			wantErr: true,
		},
		{
			name:    "empty backend",
			backend: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBackend(tt.backend)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBackend() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnsureAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		backend    string
		setupEnv   func()
		cleanupEnv func()
		wantErr    bool
		wantKey    string
	}{
		{
			name:    "non-openai backend",
			backend: "llama.cpp",
			wantErr: false,
			wantKey: "",
		},
		{
			name:    "openai backend without key",
			backend: "openai",
			wantErr: true,
		},
		{
			name:    "openai backend with key",
			backend: "openai",
			setupEnv: func() {
				os.Setenv("OPENAI_API_KEY", "test-key")
			},
			cleanupEnv: func() {
				os.Unsetenv("OPENAI_API_KEY")
			},
			wantErr: false,
			wantKey: "test-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			key, err := ensureAPIKey(tt.backend)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if key != tt.wantKey {
				t.Errorf("ensureAPIKey() key = %v, want %v", key, tt.wantKey)
			}
		})
	}
}
