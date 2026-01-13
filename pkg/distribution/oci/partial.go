package oci

import (
	"bytes"
	"encoding/json"
)

// Helpers for computing image metadata from partial information.

// WithRawManifest represents types that can provide raw manifest bytes.
type WithRawManifest interface {
	RawManifest() ([]byte, error)
}

// WithManifest represents types that can provide a manifest.
type WithManifest interface {
	Manifest() (*Manifest, error)
}

// WithRawConfigFile represents types that can provide raw config file bytes.
type WithRawConfigFile interface {
	RawConfigFile() ([]byte, error)
}

// WithConfigFile represents types that can provide a config file.
type WithConfigFile interface {
	ConfigFile() (*ConfigFile, error)
}

// WithLayers represents types that can provide layers.
type WithLayers interface {
	Layers() ([]Layer, error)
}

// Digest computes the digest of an image from its raw manifest.
func Digest(i WithRawManifest) (Hash, error) {
	raw, err := i.RawManifest()
	if err != nil {
		return Hash{}, err
	}
	h, _, err := SHA256(bytes.NewReader(raw))
	return h, err
}

// Size computes the total size of an image (manifest + config + layers).
func Size(i interface {
	WithRawManifest
	WithRawConfigFile
	WithLayers
}) (int64, error) {
	rawManifest, err := i.RawManifest()
	if err != nil {
		return 0, err
	}

	rawConfig, err := i.RawConfigFile()
	if err != nil {
		return 0, err
	}

	layers, err := i.Layers()
	if err != nil {
		return 0, err
	}

	size := int64(len(rawManifest)) + int64(len(rawConfig))
	for _, l := range layers {
		s, err := l.Size()
		if err != nil {
			return 0, err
		}
		size += s
	}
	return size, nil
}

// ConfigName computes the config name (digest of config file) from raw config bytes.
func ConfigName(i WithRawConfigFile) (Hash, error) {
	raw, err := i.RawConfigFile()
	if err != nil {
		return Hash{}, err
	}
	h, _, err := SHA256(bytes.NewReader(raw))
	return h, err
}

// RawManifest computes the raw manifest bytes from a manifest object.
func RawManifest(i WithManifest) ([]byte, error) {
	m, err := i.Manifest()
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}
