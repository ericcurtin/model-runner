package mlx

// Package mlx provides an inference backend using Apple's MLX framework.
// MLX is optimized for Apple Silicon and uses the unified memory architecture
// for efficient model inference.
//
// This backend:
//   - Only works on macOS with Apple Silicon (arm64)
//   - Requires Python 3.8 or later
//   - Uses models in safetensors format
//   - Automatically installs mlx-lm Python package in a virtual environment
//   - Runs mlx_lm.server to provide OpenAI-compatible API
//
// The mlx-lm server is invoked as:
//   python -m mlx_lm.server --model <path> --host <socket>

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/model-runner/pkg/diskusage"
	"github.com/docker/model-runner/pkg/inference"
	"github.com/docker/model-runner/pkg/inference/models"
	"github.com/docker/model-runner/pkg/logging"
	"github.com/docker/model-runner/pkg/sandbox"
	"github.com/docker/model-runner/pkg/tailbuffer"
)

const (
	// Name is the backend name.
	Name = "mlx"
)

// mlx is the MLX-based backend implementation.
type mlx struct {
	// log is the associated logger.
	log logging.Logger
	// modelManager is the shared model manager.
	modelManager *models.Manager
	// serverLog is the logger to use for the mlx-lm server process.
	serverLog logging.Logger
	// mlxEnvPath is the path to the Python virtual environment for MLX.
	mlxEnvPath string
	// status is the state in which the mlx backend is in.
	status string
}

// New creates a new MLX-based backend.
func New(log logging.Logger, modelManager *models.Manager, serverLog logging.Logger, mlxEnvPath string) (inference.Backend, error) {
	return &mlx{
		log:          log,
		modelManager: modelManager,
		serverLog:    serverLog,
		mlxEnvPath:   mlxEnvPath,
		status:       "not installed",
	}, nil
}

// Name implements inference.Backend.Name.
func (m *mlx) Name() string {
	return Name
}

// UsesExternalModelManagement implements
// inference.Backend.UsesExternalModelManagement.
func (l *mlx) UsesExternalModelManagement() bool {
	return false
}

// Install implements inference.Backend.Install.
func (m *mlx) Install(ctx context.Context, httpClient *http.Client) error {
	// MLX is only supported on macOS with Apple Silicon
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		m.status = "platform not supported"
		return errors.New("MLX backend is only supported on macOS with Apple Silicon (arm64)")
	}

	m.status = "installing"
	m.log.Info("Installing MLX backend...")

	// Check if Python3 is available
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		m.status = "python3 not found"
		return fmt.Errorf("python3 is required for MLX backend: %w", err)
	}
	m.log.Infof("Found python3 at: %s", pythonPath)

	// Create virtual environment directory if it doesn't exist
	if err := os.MkdirAll(m.mlxEnvPath, 0o755); err != nil {
		m.status = "failed to create venv directory"
		return fmt.Errorf("failed to create MLX environment directory: %w", err)
	}

	venvPath := filepath.Join(m.mlxEnvPath, "mlx-venv")

	// Check if virtual environment already exists and is valid
	pipPath := filepath.Join(venvPath, "bin", "pip")
	venvPythonPath := filepath.Join(venvPath, "bin", "python3")

	// Check if mlx-lm is already installed by trying to import it
	if _, err := os.Stat(venvPythonPath); err == nil {
		// Try to verify mlx_lm installation
		cmd := exec.CommandContext(ctx, venvPythonPath, "-c", "import mlx_lm")
		if err := cmd.Run(); err == nil {
			// MLX-LM is already installed
			m.log.Info("MLX-LM is already installed")
			m.status = "installed"
			return nil
		}
	}

	// Create virtual environment
	m.log.Info("Creating Python virtual environment for MLX...")
	venvCmd := exec.CommandContext(ctx, pythonPath, "-m", "venv", venvPath)
	if output, err := venvCmd.CombinedOutput(); err != nil {
		m.status = "failed to create venv"
		return fmt.Errorf("failed to create virtual environment: %w\nOutput: %s", err, output)
	}

	// Install mlx-lm package
	m.log.Info("Installing mlx-lm package (this may take a few minutes)...")
	pipCmd := exec.CommandContext(ctx, pipPath, "install", "-q", "mlx-lm")
	if output, err := pipCmd.CombinedOutput(); err != nil {
		m.status = "failed to install mlx-lm"
		return fmt.Errorf("failed to install mlx-lm: %w\nOutput: %s", err, output)
	}

	// Verify installation
	verifyCmd := exec.CommandContext(ctx, venvPythonPath, "-c", "import mlx_lm")
	if err := verifyCmd.Run(); err != nil {
		m.status = "installation verification failed"
		return fmt.Errorf("mlx-lm installation verification failed: %w", err)
	}

	m.log.Info("MLX backend installed successfully")
	m.status = "installed"
	return nil
}

// Run implements inference.Backend.Run.
func (m *mlx) Run(ctx context.Context, socket, model string, mode inference.BackendMode, config *inference.BackendConfiguration) error {
	// Only support completion mode for now
	if mode != inference.BackendModeCompletion {
		return fmt.Errorf("MLX backend only supports completion mode, got: %s", mode.String())
	}

	bundle, err := m.modelManager.GetBundle(model)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	// Get the safetensors directory path
	safetensorsPath := bundle.SafetensorsPath()
	if safetensorsPath == "" {
		return errors.New("model does not contain safetensors files; MLX backend requires safetensors format")
	}

	// Remove existing socket if it exists
	if err := os.RemoveAll(socket); err != nil && !os.IsNotExist(err) {
		m.log.Warnf("failed to remove socket file %s: %v\n", socket, err)
		m.log.Warnln("MLX may not be able to start")
	}

	venvPath := filepath.Join(m.mlxEnvPath, "mlx-venv")
	pythonPath := filepath.Join(venvPath, "bin", "python3")

	// Verify Python is available in venv
	if _, err := os.Stat(pythonPath); err != nil {
		return fmt.Errorf("python3 not found in MLX virtual environment: %w", err)
	}

	// Build arguments for mlx_lm.server
	// The server is run as: python -m mlx_lm.server --model <path> --host <socket>
	args := []string{
		"-m", "mlx_lm.server",
		"--model", safetensorsPath,
		"--host", socket,
	}

	// Add context size if specified
	if config != nil && config.ContextSize > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", config.ContextSize))
	}

	m.log.Infof("MLX args: %v", args)
	tailBuf := tailbuffer.NewTailBuffer(1024)
	serverLogStream := m.serverLog.Writer()
	out := io.MultiWriter(serverLogStream, tailBuf)

	mlxSandbox, err := sandbox.Create(
		ctx,
		sandbox.ConfigurationLlamaCpp,
		func(command *exec.Cmd) {
			command.Cancel = func() error {
				return command.Process.Signal(os.Interrupt)
			}
			command.Stdout = serverLogStream
			command.Stderr = out
		},
		venvPath,
		pythonPath,
		args...,
	)
	if err != nil {
		return fmt.Errorf("unable to start MLX server: %w", err)
	}
	defer mlxSandbox.Close()

	mlxErrors := make(chan error, 1)
	go func() {
		mlxErr := mlxSandbox.Command().Wait()
		serverLogStream.Close()

		errOutput := new(strings.Builder)
		if _, err := io.Copy(errOutput, tailBuf); err != nil {
			m.log.Warnf("failed to read server output tail: %w", err)
		}

		if len(errOutput.String()) != 0 {
			mlxErr = fmt.Errorf("MLX server exit status: %w\nwith output: %s", mlxErr, errOutput.String())
		} else {
			mlxErr = fmt.Errorf("MLX server exit status: %w", mlxErr)
		}

		mlxErrors <- mlxErr
		close(mlxErrors)
		if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
			m.log.Warnf("failed to remove socket file %s on exit: %w\n", socket, err)
		}
	}()
	defer func() {
		<-mlxErrors
	}()

	select {
	case <-ctx.Done():
		return nil
	case mlxErr := <-mlxErrors:
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		return fmt.Errorf("MLX server terminated unexpectedly: %w", mlxErr)
	}
}

func (m *mlx) Status() string {
	return m.status
}

func (m *mlx) GetDiskUsage() (int64, error) {
	size, err := diskusage.Size(m.mlxEnvPath)
	if err != nil {
		return 0, fmt.Errorf("error while getting MLX environment size: %v", err)
	}
	return size, nil
}

func (m *mlx) GetRequiredMemoryForModel(ctx context.Context, model string, config *inference.BackendConfiguration) (inference.RequiredMemory, error) {
	// For MLX on Apple Silicon, models run on unified memory
	// We need to estimate based on model size
	bundle, err := m.modelManager.GetBundle(model)
	if err != nil {
		return inference.RequiredMemory{}, fmt.Errorf("getting model bundle: %w", err)
	}

	// Get the safetensors path to estimate size
	safetensorsPath := bundle.SafetensorsPath()
	if safetensorsPath == "" {
		return inference.RequiredMemory{}, errors.New("model does not contain safetensors files")
	}

	// Estimate memory based on model files
	// For safetensors models, we estimate roughly 1.2x the model size for overhead
	var totalSize int64
	err = filepath.Walk(safetensorsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return inference.RequiredMemory{}, fmt.Errorf("calculating model size: %w", err)
	}

	// Add overhead for context and computation (approximately 20% overhead)
	estimatedMemory := uint64(float64(totalSize) * 1.2)

	// On Apple Silicon, we use unified memory (RAM = VRAM)
	return inference.RequiredMemory{
		RAM:  estimatedMemory,
		VRAM: 0, // Unified memory architecture
	}, nil
}
