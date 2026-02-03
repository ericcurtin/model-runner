//! Local model cache

use dmrlet_core::DmrletResult;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::path::PathBuf;
use std::time::SystemTime;
use tokio::sync::RwLock;
use tracing::{debug, info, warn};

/// Cached model metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CachedModel {
    /// Model reference (e.g., "ai/llama3:8b")
    pub reference: String,
    /// Path to the model file
    pub path: PathBuf,
    /// Model size in bytes
    pub size: u64,
    /// Last access time
    pub last_accessed: SystemTime,
    /// Download time
    pub downloaded_at: SystemTime,
}

/// Model cache manager
pub struct ModelCache {
    /// Base path for model storage
    base_path: PathBuf,
    /// Maximum cache size in bytes
    max_size: u64,
    /// Current cache size
    current_size: RwLock<u64>,
    /// Cached models indexed by reference
    models: RwLock<HashMap<String, CachedModel>>,
    /// Enable LRU eviction
    lru_enabled: bool,
}

impl ModelCache {
    /// Create a new model cache
    pub fn new(base_path: PathBuf, max_size: u64, lru_enabled: bool) -> Self {
        Self {
            base_path,
            max_size,
            current_size: RwLock::new(0),
            models: RwLock::new(HashMap::new()),
            lru_enabled,
        }
    }

    /// Initialize the cache by scanning existing models
    pub async fn init(&self) -> DmrletResult<()> {
        if !self.base_path.exists() {
            tokio::fs::create_dir_all(&self.base_path).await?;
            info!(path = %self.base_path.display(), "Created model cache directory");
        }

        // Scan for existing models (would read metadata files in production)
        Ok(())
    }

    /// Check if a model is cached
    pub async fn has(&self, reference: &str) -> bool {
        let models = self.models.read().await;
        models.contains_key(reference)
    }

    /// Get the path to a cached model
    pub async fn get(&self, reference: &str) -> Option<PathBuf> {
        let mut models = self.models.write().await;
        if let Some(model) = models.get_mut(reference) {
            model.last_accessed = SystemTime::now();
            Some(model.path.clone())
        } else {
            None
        }
    }

    /// Add a model to the cache
    pub async fn add(
        &self,
        reference: &str,
        path: PathBuf,
        size: u64,
    ) -> DmrletResult<()> {
        // Check if we need to evict
        if self.lru_enabled {
            self.ensure_space(size).await?;
        }

        let model = CachedModel {
            reference: reference.to_string(),
            path,
            size,
            last_accessed: SystemTime::now(),
            downloaded_at: SystemTime::now(),
        };

        let mut models = self.models.write().await;
        let mut current_size = self.current_size.write().await;

        // Remove old entry if exists
        if let Some(old) = models.remove(reference) {
            *current_size = current_size.saturating_sub(old.size);
        }

        *current_size += size;
        models.insert(reference.to_string(), model);

        debug!(
            reference = reference,
            size = size,
            "Added model to cache"
        );

        Ok(())
    }

    /// Remove a model from the cache
    pub async fn remove(&self, reference: &str) -> DmrletResult<()> {
        let mut models = self.models.write().await;
        let mut current_size = self.current_size.write().await;

        if let Some(model) = models.remove(reference) {
            *current_size = current_size.saturating_sub(model.size);

            // Delete the file
            if model.path.exists() {
                tokio::fs::remove_file(&model.path).await?;
            }

            info!(reference = reference, "Removed model from cache");
        }

        Ok(())
    }

    /// List all cached models
    pub async fn list(&self) -> Vec<CachedModel> {
        let models = self.models.read().await;
        models.values().cloned().collect()
    }

    /// Ensure there's enough space for a new model
    async fn ensure_space(&self, needed: u64) -> DmrletResult<()> {
        let current_size = *self.current_size.read().await;

        if current_size + needed <= self.max_size {
            return Ok(());
        }

        // Need to evict models
        let to_free = (current_size + needed).saturating_sub(self.max_size);

        // Clone the models we need to consider for eviction
        let models_to_consider: Vec<CachedModel> = {
            let models = self.models.read().await;
            models.values().cloned().collect()
        };

        let mut models_by_access = models_to_consider;
        models_by_access.sort_by(|a, b| a.last_accessed.cmp(&b.last_accessed));

        let mut freed = 0u64;
        let mut to_remove = Vec::new();

        for model in models_by_access {
            if freed >= to_free {
                break;
            }
            to_remove.push(model.reference.clone());
            freed += model.size;
        }

        for reference in to_remove {
            warn!(
                reference = %reference,
                "Evicting model from cache (LRU)"
            );
            self.remove(&reference).await?;
        }

        Ok(())
    }

    /// Get cache statistics
    pub async fn stats(&self) -> CacheStats {
        let current_size = *self.current_size.read().await;
        let models = self.models.read().await;

        CacheStats {
            total_size: current_size,
            max_size: self.max_size,
            model_count: models.len(),
            utilization: (current_size as f64 / self.max_size as f64) * 100.0,
        }
    }
}

/// Cache statistics
#[derive(Debug, Clone, Serialize)]
pub struct CacheStats {
    /// Current total size in bytes
    pub total_size: u64,
    /// Maximum cache size in bytes
    pub max_size: u64,
    /// Number of cached models
    pub model_count: usize,
    /// Cache utilization percentage
    pub utilization: f64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_model_cache() {
        let cache = ModelCache::new(
            PathBuf::from("/tmp/test-cache"),
            1024 * 1024,
            false,
        );

        assert!(!cache.has("test-model").await);

        cache
            .add("test-model", PathBuf::from("/tmp/model.gguf"), 1024)
            .await
            .unwrap();

        assert!(cache.has("test-model").await);
        assert!(cache.get("test-model").await.is_some());
    }

    #[tokio::test]
    async fn test_cache_stats() {
        let cache = ModelCache::new(PathBuf::from("/tmp/test"), 1024 * 1024, false);

        cache
            .add("model1", PathBuf::from("/tmp/m1.gguf"), 512)
            .await
            .unwrap();
        cache
            .add("model2", PathBuf::from("/tmp/m2.gguf"), 256)
            .await
            .unwrap();

        let stats = cache.stats().await;
        assert_eq!(stats.model_count, 2);
        assert_eq!(stats.total_size, 768);
    }
}
