// Package partial provides helpers for working with partial OCI images.
// This package serves as a drop-in replacement for github.com/google/go-containerregistry/pkg/v1/partial
package partial

import (
"bytes"
"io"

v1 "github.com/docker/model-runner/internal/oci/v1"
)

// WithRawManifest defines the subset of v1.Image used by Digest.
type WithRawManifest interface {
RawManifest() ([]byte, error)
}

// Digest computes the digest of the manifest.
func Digest(i WithRawManifest) (v1.Hash, error) {
rawManifest, err := i.RawManifest()
if err != nil {
return v1.Hash{}, err
}
hash, _, err := v1.SHA256(bytes.NewReader(rawManifest))
return hash, err
}

// WithRawConfigFile defines the subset of v1.Image used by ConfigName.
type WithRawConfigFile interface {
RawConfigFile() ([]byte, error)
}

// ConfigName computes the digest of the config file.
func ConfigName(i WithRawConfigFile) (v1.Hash, error) {
rawConfig, err := i.RawConfigFile()
if err != nil {
return v1.Hash{}, err
}
hash, _, err := v1.SHA256(bytes.NewReader(rawConfig))
return hash, err
}

// WithConfigFile defines the subset of v1.Image used by ConfigFile.
type WithConfigFile interface {
WithRawConfigFile
}

// ConfigFile returns the parsed config file.
func ConfigFile(i WithConfigFile) (*v1.ConfigFile, error) {
rawConfig, err := i.RawConfigFile()
if err != nil {
return nil, err
}
return v1.ParseConfigFile(bytes.NewReader(rawConfig))
}

// WithLayers defines the subset of v1.Image used by Size.
type WithLayers interface {
Layers() ([]v1.Layer, error)
}

// Size computes the total size of the image.
func Size(i WithLayers) (int64, error) {
layers, err := i.Layers()
if err != nil {
return 0, err
}
size := int64(0)
for _, layer := range layers {
layerSize, err := layer.Size()
if err != nil {
return 0, err
}
size += layerSize
}
return size, nil
}

// WithManifest defines the subset of v1.Image used by Manifest.
type WithManifest interface {
RawManifest() ([]byte, error)
}

// Manifest returns the parsed manifest.
func Manifest(i WithManifest) (*v1.Manifest, error) {
rawManifest, err := i.RawManifest()
if err != nil {
return nil, err
}
return v1.ParseManifest(bytes.NewReader(rawManifest))
}

// Descriptor returns a descriptor for a layer.
func Descriptor(l v1.Layer) (*v1.Descriptor, error) {
digest, err := l.Digest()
if err != nil {
return nil, err
}
size, err := l.Size()
if err != nil {
return nil, err
}
mediaType, err := l.MediaType()
if err != nil {
return nil, err
}
return &v1.Descriptor{
MediaType: mediaType,
Size:      size,
Digest:    digest,
}, nil
}

// ConfigLayer returns a layer containing the raw config file.
func ConfigLayer(i WithRawConfigFile) (v1.Layer, error) {
rawConfig, err := i.RawConfigFile()
if err != nil {
return nil, err
}
return &configLayer{rawConfig: rawConfig}, nil
}

type configLayer struct {
rawConfig []byte
digest    *v1.Hash
}

func (c *configLayer) Digest() (v1.Hash, error) {
if c.digest != nil {
return *c.digest, nil
}
h, _, err := v1.SHA256(bytes.NewReader(c.rawConfig))
if err != nil {
return v1.Hash{}, err
}
c.digest = &h
return h, nil
}

func (c *configLayer) DiffID() (v1.Hash, error) {
return c.Digest()
}

func (c *configLayer) Compressed() (io.ReadCloser, error) {
return io.NopCloser(bytes.NewReader(c.rawConfig)), nil
}

func (c *configLayer) Uncompressed() (io.ReadCloser, error) {
return io.NopCloser(bytes.NewReader(c.rawConfig)), nil
}

func (c *configLayer) Size() (int64, error) {
return int64(len(c.rawConfig)), nil
}

func (c *configLayer) MediaType() (v1.MediaType, error) {
return "application/vnd.oci.image.config.v1+json", nil
}
