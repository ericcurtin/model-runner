// Package v1 provides OCI image types compatible with the OCI image specification.
// This package serves as a drop-in replacement for github.com/google/go-containerregistry/pkg/v1
package v1

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"

	"github.com/opencontainers/go-digest"
)

// Hash represents a content-addressable hash.
type Hash struct {
	Algorithm string
	Hex       string
}

// NewHash creates a new Hash from a digest string (e.g., "sha256:abc123...").
func NewHash(s string) (Hash, error) {
	d, err := digest.Parse(s)
	if err != nil {
		return Hash{}, err
	}
	return Hash{
		Algorithm: string(d.Algorithm()),
		Hex:       d.Encoded(),
	}, nil
}

// String returns the hash in the format "algorithm:hex".
func (h Hash) String() string {
	return h.Algorithm + ":" + h.Hex
}

// Digest returns the OCI digest representation.
func (h Hash) Digest() digest.Digest {
	return digest.Digest(h.String())
}

// SHA256 computes the sha256 hash of the reader.
func SHA256(r io.Reader) (Hash, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return Hash{}, 0, err
	}
	return Hash{
		Algorithm: "sha256",
		Hex:       hex.EncodeToString(h.Sum(nil)),
	}, n, nil
}

// Hasher returns a new hash.Hash for the given algorithm.
func Hasher(name string) (hash.Hash, error) {
	switch name {
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", name)
	}
}

// Descriptor represents an OCI descriptor.
type Descriptor struct {
	MediaType   MediaType         `json:"mediaType,omitempty"`
	Size        int64             `json:"size"`
	Digest      Hash              `json:"digest"`
	URLs        []string          `json:"urls,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Platform    *Platform         `json:"platform,omitempty"`
}

// MediaType represents a media type string.
type MediaType string

// Platform represents the platform an image is built for.
type Platform struct {
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	OSVersion    string   `json:"os.version,omitempty"`
	OSFeatures   []string `json:"os.features,omitempty"`
	Variant      string   `json:"variant,omitempty"`
}

// Manifest represents an OCI image manifest.
type Manifest struct {
	SchemaVersion int                `json:"schemaVersion"`
	MediaType     MediaType          `json:"mediaType,omitempty"`
	Config        Descriptor         `json:"config"`
	Layers        []Descriptor       `json:"layers"`
	Annotations   map[string]string  `json:"annotations,omitempty"`
}

// ParseManifest parses a manifest from a reader.
func ParseManifest(r io.Reader) (*Manifest, error) {
	var m Manifest
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// RootFS describes the root filesystem.
type RootFS struct {
	Type    string `json:"type"`
	DiffIDs []Hash `json:"diff_ids"`
}

// ConfigFile represents the configuration file for an OCI image.
type ConfigFile struct {
	// Architecture is the CPU architecture
	Architecture string `json:"architecture,omitempty"`
	// OS is the operating system
	OS string `json:"os,omitempty"`
	// RootFS describes the root filesystem
	RootFS RootFS `json:"rootfs,omitempty"`
}

// ParseConfigFile parses a config file from a reader.
func ParseConfigFile(r io.Reader) (*ConfigFile, error) {
	var cf ConfigFile
	if err := json.NewDecoder(r).Decode(&cf); err != nil {
		return nil, err
	}
	return &cf, nil
}

// DeepCopy creates a deep copy of RootFS
func (r *RootFS) DeepCopy() *RootFS {
	if r == nil {
		return nil
	}
	copy := &RootFS{
		Type:    r.Type,
		DiffIDs: make([]Hash, len(r.DiffIDs)),
	}
	for i, h := range r.DiffIDs {
		copy.DiffIDs[i] = h
	}
	return copy
}

// DeepCopyInto copies the receiver into out
func (r *RootFS) DeepCopyInto(out *RootFS) {
	*out = *r
	if r.DiffIDs != nil {
		out.DiffIDs = make([]Hash, len(r.DiffIDs))
		copy(out.DiffIDs, r.DiffIDs)
	}
}

// Layer represents an OCI image layer.
type Layer interface {
	// Digest returns the content hash of the layer
	Digest() (Hash, error)

	// DiffID returns the uncompressed content hash
	DiffID() (Hash, error)

	// Compressed returns a reader for the compressed layer content
	Compressed() (io.ReadCloser, error)

	// Uncompressed returns a reader for the uncompressed layer content
	Uncompressed() (io.ReadCloser, error)

	// Size returns the compressed size of the layer
	Size() (int64, error)

	// MediaType returns the media type of the layer
	MediaType() (MediaType, error)
}

// Image represents an OCI image.
type Image interface {
	// Layers returns the ordered list of layers
	Layers() ([]Layer, error)

	// MediaType returns the media type of the image
	MediaType() (MediaType, error)

	// Size returns the size of the image
	Size() (int64, error)

	// ConfigName returns the hash of the config file
	ConfigName() (Hash, error)

	// ConfigFile returns the parsed config file
	ConfigFile() (*ConfigFile, error)

	// RawConfigFile returns the raw config file bytes
	RawConfigFile() ([]byte, error)

	// Digest returns the hash of the manifest
	Digest() (Hash, error)

	// Manifest returns the image manifest
	Manifest() (*Manifest, error)

	// RawManifest returns the raw manifest bytes
	RawManifest() ([]byte, error)

	// LayerByDigest returns a layer by its digest
	LayerByDigest(hash Hash) (Layer, error)

	// LayerByDiffID returns a layer by its diff ID
	LayerByDiffID(hash Hash) (Layer, error)
}

// Compute the digest of a manifest
func (m *Manifest) Digest() (Hash, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return Hash{}, err
	}
	h, _, err := SHA256(bytes.NewReader(b))
	return h, err
}
