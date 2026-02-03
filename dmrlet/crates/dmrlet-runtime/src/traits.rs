//! Runtime trait definitions

use async_trait::async_trait;
use dmrlet_core::{DmrletResult, Worker};

/// Runtime trait for managing inference workers
#[async_trait]
pub trait Runtime: Send + Sync {
    /// Start a new worker
    async fn start_worker(&self, worker: &mut Worker, model_path: &str) -> DmrletResult<()>;

    /// Stop a running worker
    async fn stop_worker(&self, worker: &Worker) -> DmrletResult<()>;

    /// Check if a worker is running
    async fn is_running(&self, worker: &Worker) -> DmrletResult<bool>;

    /// Get the runtime name
    fn name(&self) -> &'static str;
}
