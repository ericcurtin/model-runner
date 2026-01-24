package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"golang.org/x/term"
)

// Runner executes the launching of an integration with a model
type Runner interface {
	Run(model string) error
	// String returns the human-readable name of the integration
	String() string
}

// Editor can edit config files (supports multi-model selection)
type Editor interface {
	// Paths returns the paths to the config files for the integration
	Paths() []string
	// Edit updates the config files for the integration with the given models
	Edit(models []string) error
	// Models returns the models currently configured for the integration
	Models() []string
}

// integrations is the registry of available integrations.
var integrations = map[string]Runner{
	"anythingllm": &AnythingLLM{},
	"claude":      &Claude{},
	"codex":       &Codex{},
	"opencode":    &OpenCode{},
	"openwebui":   &OpenWebUI{},
}

// launchIntegration contains the saved configuration for an integration
type launchIntegration struct {
	Models []string `json:"models"`
}

// launchConfig stores integration configurations
type launchConfig struct {
	Integrations map[string]*launchIntegration `json:"integrations"`
}

func launchConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docker", "model-cli", "launch.json"), nil
}

func loadLaunchConfig() (*launchConfig, error) {
	path, err := launchConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &launchConfig{Integrations: make(map[string]*launchIntegration)}, nil
		}
		return nil, err
	}

	var cfg launchConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w, at: %s", err, path)
	}
	if cfg.Integrations == nil {
		cfg.Integrations = make(map[string]*launchIntegration)
	}
	return &cfg, nil
}

func saveLaunchConfig(cfg *launchConfig) error {
	path, err := launchConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func saveLaunchIntegration(appName string, models []string) error {
	if appName == "" {
		return errors.New("app name cannot be empty")
	}

	cfg, err := loadLaunchConfig()
	if err != nil {
		return err
	}

	cfg.Integrations[strings.ToLower(appName)] = &launchIntegration{
		Models: models,
	}

	return saveLaunchConfig(cfg)
}

func loadLaunchIntegration(appName string) (*launchIntegration, error) {
	cfg, err := loadLaunchConfig()
	if err != nil {
		return nil, err
	}

	ic, ok := cfg.Integrations[strings.ToLower(appName)]
	if !ok {
		return nil, os.ErrNotExist
	}

	return ic, nil
}

// ANSI escape sequences for terminal formatting.
const (
	ansiHideCursor = "\033[?25l"
	ansiShowCursor = "\033[?25h"
	ansiBold       = "\033[1m"
	ansiReset      = "\033[0m"
	ansiGray       = "\033[37m"
	ansiClearDown  = "\033[J"
)

const maxDisplayedItems = 10

var errLaunchCancelled = errors.New("cancelled")

type launchSelectItem struct {
	Name        string
	Description string
}

type launchSelectState struct {
	items        []launchSelectItem
	filter       string
	selected     int
	scrollOffset int
}

func newLaunchSelectState(items []launchSelectItem) *launchSelectState {
	return &launchSelectState{items: items}
}

func (s *launchSelectState) filtered() []launchSelectItem {
	return filterLaunchItems(s.items, s.filter)
}

func filterLaunchItems(items []launchSelectItem, filter string) []launchSelectItem {
	if filter == "" {
		return items
	}
	var result []launchSelectItem
	filterLower := strings.ToLower(filter)
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), filterLower) {
			result = append(result, item)
		}
	}
	return result
}

type launchInputEvent int

const (
	launchEventNone launchInputEvent = iota
	launchEventEnter
	launchEventEscape
	launchEventUp
	launchEventDown
	launchEventBackspace
	launchEventChar
)

func (s *launchSelectState) handleInput(event launchInputEvent, char byte) (done bool, result string, err error) {
	filtered := s.filtered()

	switch event {
	case launchEventEnter:
		if len(filtered) > 0 && s.selected < len(filtered) {
			return true, filtered[s.selected].Name, nil
		}
	case launchEventEscape:
		return true, "", errLaunchCancelled
	case launchEventBackspace:
		if len(s.filter) > 0 {
			s.filter = s.filter[:len(s.filter)-1]
			s.selected = 0
			s.scrollOffset = 0
		}
	case launchEventUp:
		if s.selected > 0 {
			s.selected--
			if s.selected < s.scrollOffset {
				s.scrollOffset = s.selected
			}
		}
	case launchEventDown:
		if s.selected < len(filtered)-1 {
			s.selected++
			if s.selected >= s.scrollOffset+maxDisplayedItems {
				s.scrollOffset = s.selected - maxDisplayedItems + 1
			}
		}
	case launchEventChar:
		s.filter += string(char)
		s.selected = 0
		s.scrollOffset = 0
	}

	return false, "", nil
}

func launchParseInput(r io.Reader) (launchInputEvent, byte, error) {
	buf := make([]byte, 3)
	n, err := r.Read(buf)
	if err != nil {
		return 0, 0, err
	}

	switch {
	case n == 1 && buf[0] == 13:
		return launchEventEnter, 0, nil
	case n == 1 && (buf[0] == 3 || buf[0] == 27):
		return launchEventEscape, 0, nil
	case n == 1 && buf[0] == 127:
		return launchEventBackspace, 0, nil
	case n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 65:
		return launchEventUp, 0, nil
	case n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 66:
		return launchEventDown, 0, nil
	case n == 1 && buf[0] >= 32 && buf[0] < 127:
		return launchEventChar, buf[0], nil
	}

	return launchEventNone, 0, nil
}

func launchClearLines(n int) {
	if n > 0 {
		fmt.Fprintf(os.Stderr, "\033[%dA", n)
		fmt.Fprint(os.Stderr, ansiClearDown)
	}
}

func renderLaunchSelect(w io.Writer, prompt string, s *launchSelectState) int {
	filtered := s.filtered()

	fmt.Fprintf(w, "%s %s\r\n", prompt, s.filter)
	lineCount := 1

	if len(filtered) == 0 {
		fmt.Fprintf(w, "  %s(no matches)%s\r\n", ansiGray, ansiReset)
		lineCount++
	} else {
		displayCount := min(len(filtered), maxDisplayedItems)

		for i := range displayCount {
			idx := s.scrollOffset + i
			if idx >= len(filtered) {
				break
			}
			item := filtered[idx]
			prefix := "    "
			if idx == s.selected {
				prefix = "  " + ansiBold + "> "
			}
			if item.Description != "" {
				fmt.Fprintf(w, "%s%s%s %s- %s%s\r\n", prefix, item.Name, ansiReset, ansiGray, item.Description, ansiReset)
			} else {
				fmt.Fprintf(w, "%s%s%s\r\n", prefix, item.Name, ansiReset)
			}
			lineCount++
		}

		if remaining := len(filtered) - s.scrollOffset - displayCount; remaining > 0 {
			fmt.Fprintf(w, "  %s... and %d more%s\r\n", ansiGray, remaining, ansiReset)
			lineCount++
		}
	}

	return lineCount
}

// launchSelectPrompt prompts the user to select a single item from a list.
func launchSelectPrompt(prompt string, items []launchSelectItem) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select from")
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() {
		fmt.Fprint(os.Stderr, ansiShowCursor)
		term.Restore(fd, oldState)
	}()

	fmt.Fprint(os.Stderr, ansiHideCursor)
	state := newLaunchSelectState(items)
	var lastLineCount int

	render := func() {
		launchClearLines(lastLineCount)
		lastLineCount = renderLaunchSelect(os.Stderr, prompt, state)
	}

	render()

	for {
		event, char, err := launchParseInput(os.Stdin)
		if err != nil {
			return "", err
		}

		done, result, err := state.handleInput(event, char)
		if done {
			launchClearLines(lastLineCount)
			if err != nil {
				return "", err
			}
			return result, nil
		}

		render()
	}
}

func selectIntegration() (string, error) {
	if len(integrations) == 0 {
		return "", fmt.Errorf("no integrations available")
	}

	names := make([]string, 0, len(integrations))
	for name := range integrations {
		names = append(names, name)
	}
	slices.Sort(names)

	var items []launchSelectItem
	for _, name := range names {
		r := integrations[name]
		description := r.String()
		if conn, err := loadLaunchIntegration(name); err == nil && len(conn.Models) > 0 {
			description = fmt.Sprintf("%s (%s)", r.String(), conn.Models[0])
		}
		items = append(items, launchSelectItem{Name: name, Description: description})
	}

	return launchSelectPrompt("Select integration:", items)
}

// selectModels lets the user select a model for an integration
func selectModels(cmd *cobra.Command) ([]string, error) {
	models, err := desktopClient.List()
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no models available, run 'docker model pull <model>' first")
	}

	var items []launchSelectItem
	seen := make(map[string]bool)
	for _, m := range models {
		for _, tag := range m.Tags {
			// Strip default prefix and tag for cleaner display
			displayName := stripDefaultsFromModelName(tag)
			if !seen[displayName] {
				seen[displayName] = true
				items = append(items, launchSelectItem{Name: displayName})
			}
		}
	}

	slices.SortFunc(items, func(a, b launchSelectItem) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	model, err := launchSelectPrompt("Select model:", items)
	if err != nil {
		return nil, err
	}
	return []string{model}, nil
}

func runIntegration(name, modelName string) error {
	r, ok := integrations[name]
	if !ok {
		return fmt.Errorf("unknown integration: %s", name)
	}
	fmt.Fprintf(os.Stderr, "\nLaunching %s with %s...\n", r, modelName)
	return r.Run(modelName)
}

func newLaunchCmd() *cobra.Command {
	var modelFlag string
	var configFlag bool

	cmd := &cobra.Command{
		Use:   "launch [INTEGRATION]",
		Short: "Launch an integration with Docker Model Runner",
		Long: `Launch an integration configured with Docker Model Runner models.

Supported integrations:
  anythingllm   AnythingLLM
  claude        Claude Code
  codex         Codex
  opencode      OpenCode
  openwebui     Open WebUI

Examples:
  docker model launch
  docker model launch claude
  docker model launch claude --model <model>
  docker model launch opencode --config (does not auto-launch)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) > 0 {
				name = args[0]
			} else {
				var err error
				name, err = selectIntegration()
				if errors.Is(err, errLaunchCancelled) {
					return nil
				}
				if err != nil {
					return err
				}
			}

			r, ok := integrations[strings.ToLower(name)]
			if !ok {
				return fmt.Errorf("unknown integration: %s", name)
			}

			// If launching without --model, use saved config if available
			if !configFlag && modelFlag == "" {
				if config, err := loadLaunchIntegration(name); err == nil && len(config.Models) > 0 {
					return runIntegration(name, config.Models[0])
				}
			}

			var models []string
			if modelFlag != "" {
				// When --model is specified, merge with existing models (new model becomes default)
				models = []string{modelFlag}
				if existing, err := loadLaunchIntegration(name); err == nil && len(existing.Models) > 0 {
					for _, m := range existing.Models {
						if m != modelFlag {
							models = append(models, m)
						}
					}
				}
			} else {
				var err error
				models, err = selectModels(cmd)
				if errors.Is(err, errLaunchCancelled) {
					return nil
				}
				if err != nil {
					return err
				}
			}

			if editor, isEditor := r.(Editor); isEditor {
				paths := editor.Paths()
				if len(paths) > 0 {
					fmt.Fprintf(os.Stderr, "This will modify your %s configuration:\n", r)
					for _, p := range paths {
						fmt.Fprintf(os.Stderr, "  %s\n", p)
					}
					fmt.Fprintln(os.Stderr)
				}
			}

			if err := saveLaunchIntegration(name, models); err != nil {
				return fmt.Errorf("failed to save: %w", err)
			}

			if editor, isEditor := r.(Editor); isEditor {
				if err := editor.Edit(models); err != nil {
					return fmt.Errorf("setup failed: %w", err)
				}
			}

			if _, isEditor := r.(Editor); isEditor {
				if len(models) == 1 {
					fmt.Fprintf(os.Stderr, "Added %s to %s\n", models[0], r)
				} else {
					fmt.Fprintf(os.Stderr, "Added %d models to %s (default: %s)\n", len(models), r, models[0])
				}
			}

			if configFlag {
				fmt.Fprintf(os.Stderr, "Run 'docker model launch %s' to start with %s\n", strings.ToLower(name), models[0])
				return nil
			}

			return runIntegration(name, models[0])
		},
	}

	cmd.Flags().StringVar(&modelFlag, "model", "", "Model to use")
	cmd.Flags().BoolVar(&configFlag, "config", false, "Configure without launching")
	return cmd
}

// getModelRunnerURL returns the Model Runner OpenAI-compatible API endpoint URL
func getModelRunnerURL() string {
	if url := os.Getenv("MODEL_RUNNER_HOST"); url != "" {
		return strings.TrimSuffix(url, "/") + "/engines/llama.cpp/v1"
	}
	return "http://localhost:12434/engines/llama.cpp/v1"
}

// =============================================================================
// AnythingLLM Integration
// =============================================================================

// AnythingLLM implements Runner for AnythingLLM integration
type AnythingLLM struct{}

func (a *AnythingLLM) String() string { return "AnythingLLM" }

func (a *AnythingLLM) Run(model string) error {
	// AnythingLLM is typically run as a Docker container
	// Check if docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is not installed. AnythingLLM runs as a Docker container")
	}

	fmt.Fprintf(os.Stderr, `
To use AnythingLLM with Docker Model Runner:

1. Start AnythingLLM container:
   docker run -d -p 3001:3001 \
     --add-host=host.docker.internal:host-gateway \
     -v anythingllm-storage:/app/server/storage \
     --name anythingllm \
     mintplexlabs/anythingllm

2. Open http://localhost:3001 in your browser

3. Configure the LLM Provider:
   - Go to Settings > LLM
   - Select "Generic OpenAI" as the LLM Provider
   - Set Base URL: http://host.docker.internal:12434/engines/llama.cpp/v1
   - Set Model: %s

Note: Use 'host.docker.internal' to connect to Docker Model Runner from within the container.
`, model)
	return nil
}

// =============================================================================
// Claude Code Integration
// =============================================================================

// Claude implements Runner for Claude Code integration
type Claude struct{}

func (c *Claude) String() string { return "Claude Code" }

func (c *Claude) args(model string) []string {
	if model != "" {
		return []string{"--model", model}
	}
	return nil
}

func (c *Claude) Run(model string) error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude is not installed, install from https://docs.anthropic.com/en/docs/claude-code")
	}

	cmd := exec.Command("claude", c.args(model)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"ANTHROPIC_BASE_URL="+getModelRunnerURL(),
		"ANTHROPIC_API_KEY=",
		"ANTHROPIC_AUTH_TOKEN=docker-model-runner",
	)
	return cmd.Run()
}

// =============================================================================
// Codex Integration
// =============================================================================

// Codex implements Runner for Codex integration
type Codex struct{}

func (c *Codex) String() string { return "Codex" }

func (c *Codex) args(model string) []string {
	args := []string{"--oss"}
	if model != "" {
		args = append(args, "-m", model)
	}
	return args
}

func (c *Codex) Run(model string) error {
	if err := checkCodexVersion(); err != nil {
		return err
	}

	cmd := exec.Command("codex", c.args(model)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func checkCodexVersion() error {
	if _, err := exec.LookPath("codex"); err != nil {
		return fmt.Errorf("codex is not installed, install with: npm install -g @openai/codex")
	}

	out, err := exec.Command("codex", "--version").Output()
	if err != nil {
		return fmt.Errorf("failed to get codex version: %w", err)
	}

	// Parse output like "codex-cli 0.87.0"
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return fmt.Errorf("unexpected codex version output: %s", string(out))
	}

	version := "v" + fields[len(fields)-1]
	minVersion := "v0.81.0"

	if semver.Compare(version, minVersion) < 0 {
		return fmt.Errorf("codex version %s is too old, minimum required is %s, update with: npm update -g @openai/codex", fields[len(fields)-1], "0.81.0")
	}

	return nil
}

// =============================================================================
// OpenCode Integration
// =============================================================================

// OpenCode implements Runner and Editor for OpenCode integration
type OpenCode struct{}

func (o *OpenCode) String() string { return "OpenCode" }

func (o *OpenCode) Run(model string) error {
	if _, err := exec.LookPath("opencode"); err != nil {
		return fmt.Errorf("opencode is not installed, install from https://opencode.ai")
	}

	// Call Edit() to ensure config is up-to-date before launch
	models := []string{model}
	if config, err := loadLaunchIntegration("opencode"); err == nil && len(config.Models) > 0 {
		models = config.Models
	}
	if err := o.Edit(models); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	cmd := exec.Command("opencode")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (o *OpenCode) Paths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var paths []string
	p := filepath.Join(home, ".config", "opencode", "opencode.json")
	if _, err := os.Stat(p); err == nil {
		paths = append(paths, p)
	}
	sp := filepath.Join(home, ".local", "state", "opencode", "model.json")
	if _, err := os.Stat(sp); err == nil {
		paths = append(paths, sp)
	}
	return paths
}

func (o *OpenCode) Edit(modelList []string) error {
	if len(modelList) == 0 {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	config := make(map[string]any)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &config) // Ignore parse errors; treat missing/corrupt files as empty
	}

	config["$schema"] = "https://opencode.ai/config.json"

	provider, ok := config["provider"].(map[string]any)
	if !ok {
		provider = make(map[string]any)
	}

	dockerModelRunner, ok := provider["docker-model-runner"].(map[string]any)
	if !ok {
		dockerModelRunner = map[string]any{
			"npm":  "@ai-sdk/openai-compatible",
			"name": "Docker Model Runner (local)",
			"options": map[string]any{
				"baseURL": getModelRunnerURL(),
			},
		}
	}

	models, ok := dockerModelRunner["models"].(map[string]any)
	if !ok {
		models = make(map[string]any)
	}

	selectedSet := make(map[string]bool)
	for _, m := range modelList {
		selectedSet[m] = true
	}

	for name, cfg := range models {
		if cfgMap, ok := cfg.(map[string]any); ok {
			if displayName, ok := cfgMap["name"].(string); ok {
				if strings.HasSuffix(displayName, "[Docker Model Runner]") && !selectedSet[name] {
					delete(models, name)
				}
			}
		}
	}

	for _, model := range modelList {
		models[model] = map[string]any{
			"name": fmt.Sprintf("%s [Docker Model Runner]", model),
		}
	}

	dockerModelRunner["models"] = models
	provider["docker-model-runner"] = dockerModelRunner
	config["provider"] = provider

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		return err
	}

	statePath := filepath.Join(home, ".local", "state", "opencode", "model.json")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return err
	}

	state := map[string]any{
		"recent":   []any{},
		"favorite": []any{},
		"variant":  map[string]any{},
	}
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &state) // Ignore parse errors; use defaults
	}

	recent, _ := state["recent"].([]any)

	modelSet := make(map[string]bool)
	for _, m := range modelList {
		modelSet[m] = true
	}

	// Filter out existing Docker Model Runner models we're about to re-add
	var newRecent []any
	for _, entry := range recent {
		e, ok := entry.(map[string]any)
		if !ok || e["providerID"] != "docker-model-runner" {
			newRecent = append(newRecent, entry)
			continue
		}
		modelID, _ := e["modelID"].(string)
		if !modelSet[modelID] {
			newRecent = append(newRecent, entry)
		}
	}

	// Prepend models in reverse order so first model ends up first
	for i := len(modelList) - 1; i >= 0; i-- {
		model := modelList[i]
		newRecent = append([]any{map[string]any{
			"providerID": "docker-model-runner",
			"modelID":    model,
		}}, newRecent...)
	}

	const maxRecentModels = 10
	if len(newRecent) > maxRecentModels {
		newRecent = newRecent[:maxRecentModels]
	}

	state["recent"] = newRecent

	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, stateData, 0o644)
}

func (o *OpenCode) Models() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.json"))
	if err != nil {
		return nil
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}
	provider, _ := config["provider"].(map[string]any)
	dockerModelRunner, _ := provider["docker-model-runner"].(map[string]any)
	models, _ := dockerModelRunner["models"].(map[string]any)
	if len(models) == 0 {
		return nil
	}
	keys := make([]string, 0, len(models))
	for k := range models {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// =============================================================================
// Open WebUI Integration
// =============================================================================

// OpenWebUI implements Runner for Open WebUI integration
type OpenWebUI struct{}

func (o *OpenWebUI) String() string { return "Open WebUI" }

func (o *OpenWebUI) Run(model string) error {
	// Open WebUI is typically run as a Docker container
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is not installed. Open WebUI runs as a Docker container")
	}

	fmt.Fprintf(os.Stderr, `
To use Open WebUI with Docker Model Runner:

1. Start Open WebUI container:
   docker run -d -p 3000:8080 \
     --add-host=host.docker.internal:host-gateway \
     -e OPENAI_API_BASE_URL=http://host.docker.internal:12434/engines/llama.cpp/v1 \
     -e OPENAI_API_KEY=docker-model-runner \
     -v open-webui:/app/backend/data \
     --name open-webui \
     ghcr.io/open-webui/open-webui:main

2. Open http://localhost:3000 in your browser

3. Create an account and select model: %s

Note: Use 'host.docker.internal' to connect to Docker Model Runner from within the container.
`, model)

	// Try to open the browser
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", "http://localhost:3000").Start()
	case "linux":
		_ = exec.Command("xdg-open", "http://localhost:3000").Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://localhost:3000").Start()
	}

	return nil
}
