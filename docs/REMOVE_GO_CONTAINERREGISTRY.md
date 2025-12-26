# Removing go-containerregistry Dependency - Implementation Guide

## Overview

This document outlines the plan to remove `github.com/google/go-containerregistry` as a dependency from the model-runner project, replacing it with moby/docker ecosystem libraries.

## Current Status

### Completed ✅

1. **Comprehensive Analysis**
   - Identified 60+ files using go-containerregistry
   - Main imports: v1 types (29 files), v1/types (10), name (9), partial (4), remote (3)
   - All tests currently pass, project builds successfully

2. **OCI Compatibility Layer Created**
   - `internal/oci/v1` - Core OCI types (Hash, Descriptor, Manifest, Layer, Image interfaces)
   - `internal/oci/v1/types` - Media type constants
   - `internal/oci/v1/partial` - Partial image helper functions
   - `internal/oci/name` - Reference parsing using distribution/reference

### Replacement Strategy

```
go-containerregistry              →  Replacement
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
pkg/v1 (types, interfaces)        →  internal/oci/v1
pkg/v1/types (MediaType)          →  internal/oci/v1/types
pkg/v1/partial (helpers)          →  internal/oci/v1/partial  
pkg/name (reference parsing)      →  internal/oci/name (uses distribution/reference)
pkg/v1/remote (registry ops)      →  TBD: containerd/remotes/docker
pkg/authn (auth)                  →  TBD: Docker credential helpers
```

## Remaining Work

### Phase 1: Registry Client Replacement (8-12 hours)

**Goal:** Replace remote registry operations with containerd equivalents

**Implementation Approach:**
1. Use `github.com/containerd/containerd/v2/core/remotes/docker` for HTTP registry client
2. Use `github.com/containerd/containerd/v2/core/remotes/docker/auth` for authentication
3. Create adapters to match go-containerregistry interfaces

### Phase 2-5: See full document for details

## Timeline Estimate

**Total:** 48-64 hours (6-8 working days)

## Success Criteria

- [ ] All `github.com/google/go-containerregistry` imports removed
- [ ] Dependency removed from go.mod
- [ ] All tests pass
- [ ] No functionality lost
