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
        // Convert reference to a safe filename
        let safe_name = reference
            .replace('/', "_")
            .replace(':', "_");
        self.base_path.join(format!("{}.gguf", safe_name))
    }

    /// List available models
    pub async fn list(&self) -> DmrletResult<Vec<String>> {
        let mut models = Vec::new();

        if !self.base_path.exists() {
            return Ok(models);
        }

        let mut entries = tokio::fs::read_dir(&self.base_path).await?;
        while let Some(entry) = entries.next_entry().await? {
            let path = entry.path();
            if path.extension().map_or(false, |e| e == "gguf") {
                if let Some(name) = path.file_stem() {
                    // Convert back from safe filename
                    let reference = name.to_string_lossy().replace('_', "/");
                    models.push(reference);
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
        assert_eq!(
            path.to_str().unwrap(),
            "/var/lib/dmrlet/models/ai_llama3_8b.gguf"
        );
    }
}
