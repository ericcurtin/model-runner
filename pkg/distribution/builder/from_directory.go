package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/model-runner/pkg/distribution/files"
	"github.com/docker/model-runner/pkg/distribution/internal/mutate"
	"github.com/docker/model-runner/pkg/distribution/internal/partial"
	"github.com/docker/model-runner/pkg/distribution/oci"
	"github.com/docker/model-runner/pkg/distribution/types"
)

// FromDirectory creates a Builder from a directory containing model files.
// It recursively scans the directory and adds each non-hidden file as a separate layer.
// Each layer's filepath annotation preserves the relative path from the directory root.
//
// The directory structure is fully preserved, enabling support for nested HuggingFace models
// like Qwen3-TTS that have subdirectories (text_encoder/, vae/, etc.).
//
// Example directory structure:
//
//	model_dir/
//	  config.json               -> layer with annotation "config.json"
//	  model.safetensors         -> layer with annotation "model.safetensors"
//	  text_encoder/
//	    config.json             -> layer with annotation "text_encoder/config.json"
//	    model.safetensors       -> layer with annotation "text_encoder/model.safetensors"
func FromDirectory(dirPath string) (*Builder, error) {
	// Verify directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dirPath)
	}

	var layers []oci.Layer
	var diffIDs []oci.Hash
	var detectedFormat types.Format
	var weightFiles []string

	// Walk the directory tree recursively
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == dirPath {
			return nil
		}

		// Skip hidden files and directories (starting with .)
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories (but continue walking into them)
		if info.IsDir() {
			return nil
		}

		// Skip symlinks for security
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Calculate relative path from the directory root
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}

		// Classify the file to determine media type
		fileType := files.Classify(path)
		mediaType := fileTypeToMediaType(fileType)

		// Track format from weight files
		switch fileType {
		case files.FileTypeSafetensors:
			if detectedFormat == "" {
				detectedFormat = types.FormatSafetensors
			}
			weightFiles = append(weightFiles, path)
		case files.FileTypeGGUF:
			if detectedFormat == "" {
				detectedFormat = types.FormatGGUF
			}
			weightFiles = append(weightFiles, path)
		case files.FileTypeDDUF:
			if detectedFormat == "" {
				detectedFormat = types.FormatDiffusers
			}
			weightFiles = append(weightFiles, path)
		}

		// Create layer with relative path annotation
		layer, err := partial.NewLayerWithRelativePath(path, relPath, mediaType)
		if err != nil {
			return fmt.Errorf("create layer for %q: %w", relPath, err)
		}

		diffID, err := layer.DiffID()
		if err != nil {
			return fmt.Errorf("get diffID for %q: %w", relPath, err)
		}

		layers = append(layers, layer)
		diffIDs = append(diffIDs, diffID)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	if len(layers) == 0 {
		return nil, fmt.Errorf("no files found in directory: %s", dirPath)
	}

	if len(weightFiles) == 0 {
		return nil, fmt.Errorf("no weight files (safetensors, GGUF, or DDUF) found in directory: %s", dirPath)
	}

	// Build config - use the first weight file for metadata extraction
	config := types.Config{
		Format: detectedFormat,
	}

	// TODO: Extract additional metadata from weight files if needed
	// For safetensors, we might want to read config.json from the directory

	// Build the model
	created := time.Now()
	mdl := &partial.BaseModel{
		ModelConfigFile: types.ConfigFile{
			Config: config,
			Descriptor: types.Descriptor{
				Created: &created,
			},
			RootFS: oci.RootFS{
				Type:    "rootfs",
				DiffIDs: diffIDs,
			},
		},
		LayerList: layers,
	}

	return &Builder{
		model: mdl,
	}, nil
}

// fileTypeToMediaType converts a FileType to the corresponding OCI MediaType
func fileTypeToMediaType(ft files.FileType) oci.MediaType {
	switch ft {
	case files.FileTypeGGUF:
		return types.MediaTypeGGUF
	case files.FileTypeSafetensors:
		return types.MediaTypeSafetensors
	case files.FileTypeDDUF:
		return types.MediaTypeDDUF
	case files.FileTypeLicense:
		return types.MediaTypeLicense
	case files.FileTypeChatTemplate:
		return types.MediaTypeChatTemplate
	case files.FileTypeConfig:
		return types.MediaTypeModelFile
	default:
		// For unknown files, use the generic model file type
		return types.MediaTypeModelFile
	}
}

// WithFileLayer adds an individual file layer with a relative path annotation.
// This is useful for adding files that should be extracted to a specific path.
func (b *Builder) WithFileLayer(absPath, relPath string) (*Builder, error) {
	// Classify the file to determine media type
	fileType := files.Classify(absPath)
	mediaType := fileTypeToMediaType(fileType)

	layer, err := partial.NewLayerWithRelativePath(absPath, relPath, mediaType)
	if err != nil {
		return nil, fmt.Errorf("file layer from %q: %w", absPath, err)
	}

	return &Builder{
		model:          mutate.AppendLayers(b.model, layer),
		originalLayers: b.originalLayers,
	}, nil
}
