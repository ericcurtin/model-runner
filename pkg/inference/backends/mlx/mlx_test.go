package mlx

import (
	"context"
	"runtime"
	"testing"

	"github.com/docker/model-runner/pkg/inference/models"
	"github.com/sirupsen/logrus"
)

func TestNew(t *testing.T) {
	log := logrus.New()
	log.SetOutput(logrus.StandardLogger().Out)
	
	modelManager := models.NewManager(
		log,
		models.ClientConfig{
			StoreRootPath: t.TempDir(),
			Logger:        log.WithFields(logrus.Fields{"component": "model-manager"}),
		},
		nil,
		nil,
	)

	backend, err := New(log, modelManager, log, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create MLX backend: %v", err)
	}

	if backend.Name() != "mlx" {
		t.Errorf("Expected backend name 'mlx', got '%s'", backend.Name())
	}

	if backend.UsesExternalModelManagement() {
		t.Error("MLX backend should not use external model management")
	}
}

func TestInstallPlatformCheck(t *testing.T) {
	log := logrus.New()
	log.SetOutput(logrus.StandardLogger().Out)
	
	modelManager := models.NewManager(
		log,
		models.ClientConfig{
			StoreRootPath: t.TempDir(),
			Logger:        log.WithFields(logrus.Fields{"component": "model-manager"}),
		},
		nil,
		nil,
	)

	backend, err := New(log, modelManager, log, t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create MLX backend: %v", err)
	}

	ctx := context.Background()
	err = backend.Install(ctx, nil)

	// On non-macOS or non-arm64 platforms, installation should fail
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		if err == nil {
			t.Error("Expected installation to fail on non-macOS arm64 platform")
		}
		if backend.Status() != "platform not supported" {
			t.Errorf("Expected status 'platform not supported', got '%s'", backend.Status())
		}
	}
	// On macOS arm64, we can't fully test installation without Python3,
	// but we can verify it doesn't panic
}

func TestGetDiskUsage(t *testing.T) {
	log := logrus.New()
	log.SetOutput(logrus.StandardLogger().Out)
	
	modelManager := models.NewManager(
		log,
		models.ClientConfig{
			StoreRootPath: t.TempDir(),
			Logger:        log.WithFields(logrus.Fields{"component": "model-manager"}),
		},
		nil,
		nil,
	)

	envPath := t.TempDir()
	backend, err := New(log, modelManager, log, envPath)
	if err != nil {
		t.Fatalf("Failed to create MLX backend: %v", err)
	}

	size, err := backend.GetDiskUsage()
	if err != nil {
		t.Errorf("GetDiskUsage failed: %v", err)
	}
	
	// Size should be >= 0
	if size < 0 {
		t.Errorf("Expected non-negative disk usage, got %d", size)
	}
}
