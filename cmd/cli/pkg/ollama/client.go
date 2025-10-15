package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client provides methods to interact with the Ollama API.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient creates a new Ollama client with the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// PullRequest represents a request to pull a model.
type PullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

// PullResponse represents a response from the pull endpoint.
type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// Pull pulls a model from the Ollama registry.
func (c *Client) Pull(ctx context.Context, modelName string, progressCallback func(string)) error {
	reqBody := PullRequest{
		Name:   modelName,
		Stream: true,
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	decoder := json.NewDecoder(resp.Body)
	for {
		var pullResp PullResponse
		if err := decoder.Decode(&pullResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode response: %w", err)
		}
		
		if progressCallback != nil {
			if pullResp.Total > 0 {
				percentage := float64(pullResp.Completed) / float64(pullResp.Total) * 100
				progressCallback(fmt.Sprintf("%s (%.1f%%)", pullResp.Status, percentage))
			} else {
				progressCallback(pullResp.Status)
			}
		}
	}
	
	return nil
}

// ListResponse represents a response from the list endpoint.
type ListResponse struct {
	Models []Model `json:"models"`
}

// Model represents an Ollama model.
type Model struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
}

// List lists all models available in Ollama.
func (c *Client) List(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return listResp.Models, nil
}

// RunRequest represents a request to run a model.
type RunRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// RunResponse represents a response from the run endpoint.
type RunResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// Run runs a model with the given prompt.
func (c *Client) Run(ctx context.Context, modelName, prompt string, responseCallback func(string)) error {
	reqBody := RunRequest{
		Model:  modelName,
		Prompt: prompt,
		Stream: true,
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("run failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	decoder := json.NewDecoder(resp.Body)
	for {
		var runResp RunResponse
		if err := decoder.Decode(&runResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode response: %w", err)
		}
		
		if responseCallback != nil && runResp.Response != "" {
			responseCallback(runResp.Response)
		}
		
		if runResp.Done {
			break
		}
	}
	
	return nil
}
