// Package types provides media type constants for OCI images.
// This package serves as a drop-in replacement for github.com/google/go-containerregistry/pkg/v1/types
package types

// MediaType represents a media type string.
type MediaType string

// OCI media types
const (
OCIManifestSchema1     MediaType = "application/vnd.oci.image.manifest.v1+json"
OCIImageIndex          MediaType = "application/vnd.oci.image.index.v1+json"
OCIConfigJSON          MediaType = "application/vnd.oci.image.config.v1+json"
OCILayer               MediaType = "application/vnd.oci.image.layer.v1.tar"
OCILayerGzip           MediaType = "application/vnd.oci.image.layer.v1.tar+gzip"
OCILayerZstd           MediaType = "application/vnd.oci.image.layer.v1.tar+zstd"
OCIRestrictedLayer     MediaType = "application/vnd.oci.image.layer.nondistributable.v1.tar"
OCIRestrictedLayerGzip MediaType = "application/vnd.oci.image.layer.nondistributable.v1.tar+gzip"
OCIRestrictedLayerZstd MediaType = "application/vnd.oci.image.layer.nondistributable.v1.tar+zstd"
)

// Docker media types
const (
DockerManifestSchema1       MediaType = "application/vnd.docker.distribution.manifest.v1+json"
DockerManifestSchema1Signed MediaType = "application/vnd.docker.distribution.manifest.v1+prettyjws"
DockerManifestSchema2       MediaType = "application/vnd.docker.distribution.manifest.v2+json"
DockerManifestList          MediaType = "application/vnd.docker.distribution.manifest.list.v2+json"
DockerLayer                 MediaType = "application/vnd.docker.image.rootfs.diff.tar"
DockerLayerGzip             MediaType = "application/vnd.docker.image.rootfs.diff.tar.gzip"
DockerConfigJSON            MediaType = "application/vnd.docker.container.image.v1+json"
DockerPluginConfig          MediaType = "application/vnd.docker.plugin.v1+json"
DockerForeignLayer          MediaType = "application/vnd.docker.image.rootfs.foreign.diff.tar"
DockerForeignLayerGzip      MediaType = "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip"
)
