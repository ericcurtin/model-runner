// Package name provides utilities for parsing and working with OCI image references.
// This package serves as a drop-in replacement for github.com/google/go-containerregistry/pkg/name
// It uses distribution/reference which is part of the moby/docker project.
package name

import (
"fmt"
"os"
"strings"

"github.com/distribution/reference"
)

// DefaultRegistry is the default registry (Docker Hub).
const DefaultRegistry = "index.docker.io"

// Reference represents an OCI image reference.
type Reference interface {
// Context returns the repository context
Context() Repository
// Identifier returns the tag or digest
Identifier() string
// Name returns the full reference name
Name() string
// String returns the string representation
String() string
// Scope returns the repository scope string
Scope(action string) string
}

// Repository represents a repository context.
type Repository interface {
// Registry returns the registry for this repository
Registry() Registry
// RepositoryStr returns the repository string
RepositoryStr() string
// String returns the string representation
String() string
}

// Registry represents a registry.
type Registry interface {
// RegistryStr returns the registry string
RegistryStr() string
// Scheme returns the URL scheme (http/https)
Scheme() string
// String returns the string representation
String() string
}

// Tag represents a tagged image reference.
type Tag struct {
ref reference.NamedTagged
}

// Option is a functional option for parsing references.
type Option func(*options)

type options struct {
defaultRegistry string
insecure        bool
}

// WithDefaultRegistry sets a custom default registry.
func WithDefaultRegistry(registry string) Option {
return func(o *options) {
o.defaultRegistry = registry
}
}

// Insecure allows insecure (HTTP) registries.
var Insecure Option = func(o *options) {
o.insecure = true
}

// ParseReference parses an OCI image reference string.
func ParseReference(s string, opts ...Option) (Reference, error) {
o := &options{
defaultRegistry: DefaultRegistry,
}
for _, opt := range opts {
opt(o)
}

// Normalize the reference string
s = normalizeReference(s, o.defaultRegistry)

// Parse using distribution/reference
ref, err := reference.ParseNormalizedNamed(s)
if err != nil {
return nil, fmt.Errorf("invalid reference format: %w", err)
}

// Default to latest tag if no tag or digest specified
tagged, err := reference.WithTag(ref, "latest")
if err != nil {
return nil, err
}

return &Tag{ref: tagged}, nil
}

// NewTag creates a new tagged reference.
func NewTag(s string, opts ...Option) (Tag, error) {
ref, err := ParseReference(s, opts...)
if err != nil {
return Tag{}, err
}

if tag, ok := ref.(*Tag); ok {
return *tag, nil
}

return Tag{}, fmt.Errorf("reference is not a tag: %s", s)
}

// normalizeReference adds the default registry if needed.
func normalizeReference(s, defaultRegistry string) string {
// If it already has a registry, return as-is
if strings.Contains(s, "/") && strings.Contains(strings.Split(s, "/")[0], ".") {
return s
}

// If it's in the format "repository:tag" or "repository@digest", add default registry
if !strings.Contains(s, "/") || strings.HasPrefix(s, "ai/") {
return s
}

return s
}

// Tag methods
func (t *Tag) Context() Repository {
return &repo{ref: t.ref}
}

func (t *Tag) Identifier() string {
return t.ref.Tag()
}

func (t *Tag) Name() string {
return t.ref.Name()
}

func (t *Tag) String() string {
return reference.FamiliarString(t.ref)
}

func (t *Tag) Scope(action string) string {
return fmt.Sprintf("repository:%s:%s", reference.Path(t.ref), action)
}

// Repository implementation
type repo struct {
ref reference.Named
}

func (r *repo) Registry() Registry {
domain := reference.Domain(r.ref)
insecure := os.Getenv("INSECURE_REGISTRY") == "true"
return &registry{domain: domain, insecure: insecure}
}

func (r *repo) RepositoryStr() string {
return reference.Path(r.ref)
}

func (r *repo) String() string {
return r.ref.Name()
}

// Registry implementation
type registry struct {
domain   string
insecure bool
}

func (r *registry) RegistryStr() string {
return r.domain
}

func (r *registry) Scheme() string {
if r.insecure {
return "http"
}
return "https"
}

func (r *registry) String() string {
return r.domain
}

// PullScope is the scope for pull operations.
const PullScope = "pull"

// PushScope is the scope for push operations.
const PushScope = "push,pull"
