package llamacpp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/docker/model-runner/pkg/diskusage"
	"github.com/docker/model-runner/pkg/distribution/types"
	"github.com/docker/model-runner/pkg/inference"
	"github.com/docker/model-runner/pkg/inference/backends"
	"github.com/docker/model-runner/pkg/inference/config"
	"github.com/docker/model-runner/pkg/inference/models"
	"github.com/docker/model-runner/pkg/logging"
	"github.com/docker/model-runner/pkg/sandbox"
)

const (
	// Name is the backend name.
	Name = "llama.cpp"
)

// llamaCpp is the llama.cpp-based backend implementation.
type llamaCpp struct {
	// log is the associated logger.
	log logging.Logger
	// modelManager is the shared model manager.
	modelManager *models.Manager
	// serverLog is the logger to use for the llama.cpp server process.
	serverLog       logging.Logger
	updatedLlamaCpp bool
	// vendoredServerStoragePath is the parent path of the vendored version of com.docker.llama-server.
	vendoredServerStoragePath string
	// updatedServerStoragePath is the parent path of the updated version of com.docker.llama-server.
	// It is also where updates will be stored when downloaded.
	updatedServerStoragePath string
	// status is the state in which the llama.cpp backend is in.
	status string
	// config is the configuration for the llama.cpp backend.
	config config.BackendConfig
	// gpuSupported indicates whether the underlying llama-server is built with GPU support.
	gpuSupported bool
}

// New creates a new llama.cpp-based backend.
func New(
	log logging.Logger,
	modelManager *models.Manager,
	serverLog logging.Logger,
	vendoredServerStoragePath string,
	updatedServerStoragePath string,
	conf config.BackendConfig,
) (inference.Backend, error) {
	// If no config is provided, use the default configuration
	if conf == nil {
		conf = NewDefaultLlamaCppConfig()
	}

	return &llamaCpp{
		log:                       log,
		modelManager:              modelManager,
		serverLog:                 serverLog,
		vendoredServerStoragePath: vendoredServerStoragePath,
		updatedServerStoragePath:  updatedServerStoragePath,
		config:                    conf,
	}, nil
}

// Name implements inference.Backend.Name.
func (l *llamaCpp) Name() string {
	return Name
}

// UsesExternalModelManagement implements
// inference.Backend.UsesExternalModelManagement.
func (l *llamaCpp) UsesExternalModelManagement() bool {
	return false
}

// UsesTCP implements inference.Backend.UsesTCP.
func (l *llamaCpp) UsesTCP() bool {
	return false
}

// Install implements inference.Backend.Install.
func (l *llamaCpp) Install(ctx context.Context, httpClient *http.Client) error {
	l.updatedLlamaCpp = false

	// We don't currently support this backend on Windows. We'll likely
	// never support it on Intel Macs.
	if (runtime.GOOS == "darwin" && runtime.GOARCH == "amd64") ||
		(runtime.GOOS == "windows" && runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return errors.New("platform not supported")
	}

	llamaServerBin := "com.docker.llama-server"
	if runtime.GOOS == "windows" {
		llamaServerBin = "com.docker.llama-server.exe"
	}

	l.status = "installing"

	// Temporary workaround for dynamically downloading llama.cpp from Docker Hub.
	// Internet access and an available docker/docker-model-backend-llamacpp:latest on Docker Hub are required.
	// Even if docker/docker-model-backend-llamacpp:latest has been downloaded before, we still require its
	// digest to be equal to the one on Docker Hub.
	llamaCppPath := filepath.Join(l.updatedServerStoragePath, llamaServerBin)
	if err := l.ensureLatestLlamaCpp(ctx, l.log, httpClient, llamaCppPath, l.vendoredServerStoragePath); err != nil {
		l.log.Infof("failed to ensure latest llama.cpp: %v\n", err)
		if !errors.Is(err, errLlamaCppUpToDate) && !errors.Is(err, errLlamaCppUpdateDisabled) {
			l.status = fmt.Sprintf("failed to install llama.cpp: %v", err)
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
	} else {
		l.updatedLlamaCpp = true
	}

	l.gpuSupported = l.checkGPUSupport(ctx)
	l.log.Infof("installed llama-server with gpuSupport=%t", l.gpuSupported)

	return nil
}

// Run implements inference.Backend.Run.
func (l *llamaCpp) Run(ctx context.Context, socket, model string, _ string, mode inference.BackendMode, config *inference.BackendConfiguration) error {
	bundle, err := l.modelManager.GetBundle(model)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	var draftBundle types.ModelBundle
	if config != nil && config.Speculative != nil && config.Speculative.DraftModel != "" {
		draftBundle, err = l.modelManager.GetBundle(config.Speculative.DraftModel)
		if err != nil {
			return fmt.Errorf("failed to get draft model: %w", err)
		}
	}

	binPath := l.vendoredServerStoragePath
	if l.updatedLlamaCpp {
		binPath = l.updatedServerStoragePath
	}

	args, err := l.config.GetArgs(bundle, socket, mode, config)
	if err != nil {
		return fmt.Errorf("failed to get args for llama.cpp: %w", err)
	}

	if draftBundle != nil && config != nil && config.Speculative != nil {
		draftPath := draftBundle.GGUFPath()
		if draftPath != "" {
			args = append(args, "--model-draft", draftPath)
			if config.Speculative.NumTokens > 0 {
				args = append(args, "--draft-max", strconv.Itoa(config.Speculative.NumTokens))
			}
			if config.Speculative.MinAcceptanceRate > 0 {
				args = append(args, "--draft-p-min", strconv.FormatFloat(config.Speculative.MinAcceptanceRate, 'f', 2, 64))
			}
		}
	}

	return backends.RunBackend(ctx, backends.RunnerConfig{
		BackendName:     "llama.cpp",
		Socket:          socket,
		BinaryPath:      filepath.Join(binPath, "com.docker.llama-server"),
		SandboxPath:     binPath,
		SandboxConfig:   sandbox.ConfigurationLlamaCpp,
		Args:            args,
		Logger:          l.log,
		ServerLogWriter: l.serverLog.Writer(),
	})
}

func (l *llamaCpp) Status() string {
	return l.status
}

func (l *llamaCpp) GetDiskUsage() (int64, error) {
	size, err := diskusage.Size(l.updatedServerStoragePath)
	if err != nil {
		return 0, fmt.Errorf("error while getting store size: %w", err)
	}
	return size, nil
}

func (l *llamaCpp) checkGPUSupport(ctx context.Context) bool {
	binPath := l.vendoredServerStoragePath
	if l.updatedLlamaCpp {
		binPath = l.updatedServerStoragePath
	}
	var output bytes.Buffer
	llamaCppSandbox, err := sandbox.Create(
		ctx,
		sandbox.ConfigurationLlamaCpp,
		func(command *exec.Cmd) {
			command.Stdout = &output
			command.Stderr = &output
		},
		binPath,
		filepath.Join(binPath, "com.docker.llama-server"),
		"--list-devices",
	)
	if err != nil {
		l.log.Warnf("Failed to start sandboxed llama.cpp process to probe GPU support: %v", err)
		return false
	}
	defer llamaCppSandbox.Close()
	if err := llamaCppSandbox.Command().Wait(); err != nil {
		l.log.Warnf("Failed to determine if llama-server is built with GPU support: %v", err)
		return false
	}
	sc := bufio.NewScanner(strings.NewReader(output.String()))
	expectDev := false
	devRe := regexp.MustCompile(`\s{2}.*:\s`)
	ndevs := 0
	for sc.Scan() {
		if expectDev {
			if devRe.MatchString(sc.Text()) {
				ndevs++
			}
		} else {
			expectDev = strings.HasPrefix(sc.Text(), "Available devices:")
		}
	}
	return ndevs > 0
}
