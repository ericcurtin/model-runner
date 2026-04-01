//! OCI-based model store (placeholder)
//!
//! This module will provide OCI registry integration for pulling models.
//! For now, it provides a basic interface that can be expanded later.

use dmrlet_core::{DmrletError, DmrletResult};
use std::path::PathBuf;
use tracing::info;

/// OCI model store for pulling models from registries
pub struct OciStore {
    /// Base path for model storage
    base_path: PathBuf,
}

impl OciStore {
    /// Create a new OCI store
    pub fn new(base_path: PathBuf) -> Self {
        Self { base_path }
    }

    /// Pull a model from a registry
    ///
    /// This is a placeholder implementation. A full implementation would:
    /// 1. Parse the model reference
    /// 2. Authenticate with the registry
    /// 3. Pull the model layers
    /// 4. Extract and store the model file
    pub async fn pull(&self, reference: &str) -> DmrletResult<PathBuf> {
        info!(reference = reference, "Pulling model (placeholder)");

        // For now, just return an error indicating this is not implemented
        Err(DmrletError::ModelNotFound(format!(
            "OCI pulling not yet implemented for: {}",
            reference
        )))
    }

    /// Check if a model exists in the store
    pub fn exists(&self, reference: &str) -> bool {
        let path = self.model_path(reference);
        path.exists()
    }

    /// Get the local path for a model
    pub fn model_path(&self, reference: &str) -> PathBuf {
        // Use URL-safe base64 encoding to handle any characters in the reference
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};

        // Create a hash-based filename to avoid path issues
        let mut hasher = DefaultHasher::new();
        reference.hash(&mut hasher);
        let hash = hasher.finish();

        // Also include a sanitized version for readability
        let safe_name: String = reference
            .chars()
            .map(|c| if c.is_alphanumeric() || c == '-' { c } else { '_' })
            .collect();

        // Truncate to reasonable length and append hash for uniqueness
        let truncated = if safe_name.len() > 50 {
            &safe_name[..50]
        } else {
            &safe_name
        };

        self.base_path.join(format!("{}_{:016x}.gguf", truncated, hash))
    }

    /// List available models
    pub async fn list(&self) -> DmrletResult<Vec<String>> {
        // Note: This returns hash-based filenames since we use hash-based storage.
        // A metadata file approach would be needed for proper reference tracking.
        let mut models = Vec::new();

        if !self.base_path.exists() {
            return Ok(models);
        }

        let mut entries = tokio::fs::read_dir(&self.base_path).await?;
        while let Some(entry) = entries.next_entry().await? {
            let path = entry.path();
            if path.extension().map_or(false, |e| e == "gguf") {
                if let Some(name) = path.file_stem() {
                    models.push(name.to_string_lossy().to_string());
                }
            }
        }

        Ok(models)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_model_path() {
        let store = OciStore::new(PathBuf::from("/var/lib/dmrlet/models"));
        let path = store.model_path("ai/llama3:8b");
        // Path should start with the base path and be a .gguf file
        assert!(path.starts_with("/var/lib/dmrlet/models"));
        assert!(path.extension().map_or(false, |e| e == "gguf"));
        // Same reference should always produce the same path
        let path2 = store.model_path("ai/llama3:8b");
        assert_eq!(path, path2);
        // Different references should produce different paths
        let path3 = store.model_path("ai/llama3:70b");
        assert_ne!(path, path3);
    }
}
