//! Main scheduler logic

use dmrlet_core::{
    detect_gpus, DeploymentSpec, DeploymentStatus, DmrletError, DmrletResult, Worker, WorkerStatus,
};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{debug, info, warn};
use uuid::Uuid;

use crate::gpu_allocator::GpuAllocator;
use crate::placement::{DefaultPlacementStrategy, PlacementStrategy};

type DeploymentsMap = HashMap<Uuid, DeploymentSpec>;
type WorkersMap = HashMap<Uuid, Worker>;
type PortsSet = std::collections::HashSet<u16>;

/// Scheduler manages deployments and worker placement
pub struct Scheduler {
    /// Deployments indexed by ID
    deployments: RwLock<DeploymentsMap>,
    /// Workers indexed by ID
    workers: RwLock<WorkersMap>,
    /// GPU allocator
    gpu_allocator: RwLock<GpuAllocator>,
    /// Placement strategy
    placement_strategy: Arc<dyn PlacementStrategy>,
    /// Base port for workers
    base_port: u16,
    /// Maximum port for workers
    max_port: u16,
    /// Allocated ports
    allocated_ports: RwLock<PortsSet>,
}

impl Scheduler {
    /// Create a new scheduler
    pub fn new(base_port: u16, max_workers: u32) -> Self {
        let gpu_info = detect_gpus();
        let max_port = base_port + max_workers as u16;

        info!(
            gpus = gpu_info.total_count,
            base_port = base_port,
            max_port = max_port,
            "Scheduler initialized"
        );

        Self {
            deployments: RwLock::new(HashMap::new()),
            workers: RwLock::new(HashMap::new()),
            gpu_allocator: RwLock::new(GpuAllocator::new(gpu_info)),
            placement_strategy: Arc::new(DefaultPlacementStrategy),
            base_port,
            max_port,
            allocated_ports: RwLock::new(std::collections::HashSet::new()),
        }
    }

    /// Create a new deployment
    pub async fn create_deployment(&self, spec: DeploymentSpec) -> DmrletResult<Uuid> {
        let id = spec.id;

        info!(
            deployment_id = %id,
            name = %spec.name,
            model = %spec.model,
            replicas = spec.replicas,
            "Creating deployment"
        );

        // Store the deployment
        self.deployments.write().await.insert(id, spec.clone());

        // Create workers for the deployment
        for i in 0..spec.replicas {
            match self.schedule_worker(&spec, i).await {
                Ok(worker) => {
                    debug!(
                        worker_id = %worker.id,
                        port = worker.endpoint.port,
                        "Worker scheduled"
                    );
                    self.workers.write().await.insert(worker.id, worker);
                }
                Err(e) => {
                    warn!(
                        deployment_id = %id,
                        worker_index = i,
                        error = %e,
                        "Failed to schedule worker"
                    );
                }
            }
        }

        Ok(id)
    }

    /// Schedule a worker for a deployment
    async fn schedule_worker(
        &self,
        spec: &DeploymentSpec,
        worker_index: u32,
    ) -> DmrletResult<Worker> {
        // Get available resources
        let available_gpus: Vec<u32> = {
            let allocator = self.gpu_allocator.read().await;
            allocator
                .get_gpu_info()
                .iter()
                .filter(|s| !s.allocated)
                .map(|s| s.device.index)
                .collect()
        };

        let available_ports = self.get_available_ports().await;

        // Make placement decision
        let existing_workers = {
            let workers = self.workers.read().await;
            workers
                .values()
                .filter(|w| w.deployment_id == spec.id)
                .count() as u32
        };

        let decision = self
            .placement_strategy
            .place(spec, existing_workers, &available_gpus, &available_ports)
            .ok_or_else(|| {
                DmrletError::ResourceExhausted("No resources available for worker".to_string())
            })?;

        // Allocate resources
        if spec.resources.gpu_count > 0 {
            let mut allocator = self.gpu_allocator.write().await;
            allocator.allocate(spec.resources.gpu_count)?;
        }

        // Allocate port
        self.allocated_ports.write().await.insert(decision.port);

        // Create worker
        let mut worker = Worker::new(spec.id, worker_index, decision.port);
        worker.gpu_ids = decision.gpu_ids;

        Ok(worker)
    }

    /// Get available ports
    async fn get_available_ports(&self) -> Vec<u16> {
        let allocated = self.allocated_ports.read().await;
        (self.base_port..self.max_port)
            .filter(|p| !allocated.contains(p))
            .collect()
    }

    /// Delete a deployment
    pub async fn delete_deployment(&self, id: Uuid) -> DmrletResult<()> {
        info!(deployment_id = %id, "Deleting deployment");

        // Remove deployment
        let spec = self.deployments.write().await.remove(&id);

        if spec.is_none() {
            return Err(DmrletError::DeploymentNotFound(id.to_string()));
        }

        // Mark workers for termination
        let worker_ids: Vec<Uuid> = {
            let workers = self.workers.read().await;
            workers
                .values()
                .filter(|w| w.deployment_id == id)
                .map(|w| w.id)
                .collect()
        };

        for worker_id in worker_ids {
            self.remove_worker(worker_id).await?;
        }

        Ok(())
    }

    /// Remove a worker and release its resources
    async fn remove_worker(&self, worker_id: Uuid) -> DmrletResult<()> {
        let worker = self.workers.write().await.remove(&worker_id);

        if let Some(w) = worker {
            // Release GPUs
            if !w.gpu_ids.is_empty() {
                let mut allocator = self.gpu_allocator.write().await;
                allocator.release(&w.gpu_ids);
            }

            // Release port
            self.allocated_ports.write().await.remove(&w.endpoint.port);

            debug!(worker_id = %worker_id, "Worker removed");
        }

        Ok(())
    }

    /// Scale a deployment
    pub async fn scale_deployment(&self, id: Uuid, replicas: u32) -> DmrletResult<()> {
        let spec = {
            let mut deployments = self.deployments.write().await;
            let spec = deployments
                .get_mut(&id)
                .ok_or_else(|| DmrletError::DeploymentNotFound(id.to_string()))?;
            spec.replicas = replicas;
            spec.clone()
        };

        let current_workers: Vec<Worker> = {
            let workers = self.workers.read().await;
            workers
                .values()
                .filter(|w| w.deployment_id == id)
                .cloned()
                .collect()
        };

        let current_count = current_workers.len() as u32;

        if replicas > current_count {
            // Scale up
            for i in current_count..replicas {
                match self.schedule_worker(&spec, i).await {
                    Ok(worker) => {
                        self.workers.write().await.insert(worker.id, worker);
                    }
                    Err(e) => {
                        warn!(error = %e, "Failed to schedule worker during scale up");
                    }
                }
            }
        } else if replicas < current_count {
            // Scale down
            let workers_to_remove: Vec<Uuid> = current_workers
                .iter()
                .rev()
                .take((current_count - replicas) as usize)
                .map(|w| w.id)
                .collect();

            for worker_id in workers_to_remove {
                self.remove_worker(worker_id).await?;
            }
        }

        info!(
            deployment_id = %id,
            replicas = replicas,
            "Deployment scaled"
        );

        Ok(())
    }

    /// Get deployment status
    pub async fn get_deployment_status(&self, id: Uuid) -> DmrletResult<DeploymentStatus> {
        let spec = {
            let deployments = self.deployments.read().await;
            deployments
                .get(&id)
                .cloned()
                .ok_or_else(|| DmrletError::DeploymentNotFound(id.to_string()))?
        };

        let workers: Vec<Worker> = {
            let workers = self.workers.read().await;
            workers
                .values()
                .filter(|w| w.deployment_id == id)
                .cloned()
                .collect()
        };

        Ok(DeploymentStatus::new(spec, workers))
    }

    /// List all deployments
    pub async fn list_deployments(&self) -> Vec<DeploymentStatus> {
        let deployments = self.deployments.read().await;
        let workers = self.workers.read().await;

        let mut statuses = Vec::new();

        for spec in deployments.values() {
            let deployment_workers: Vec<Worker> = workers
                .values()
                .filter(|w| w.deployment_id == spec.id)
                .cloned()
                .collect();
            statuses.push(DeploymentStatus::new(spec.clone(), deployment_workers));
        }

        statuses
    }

    /// Get all workers for a deployment
    pub async fn get_workers(&self, deployment_id: Uuid) -> Vec<Worker> {
        let workers = self.workers.read().await;
        workers
            .values()
            .filter(|w| w.deployment_id == deployment_id)
            .cloned()
            .collect()
    }

    /// Get all endpoints (for direct access)
    pub async fn get_all_endpoints(&self) -> Vec<dmrlet_core::Endpoint> {
        let workers = self.workers.read().await;
        workers
            .values()
            .filter(|w| w.status == WorkerStatus::Running)
            .map(|w| w.endpoint.clone())
            .collect()
    }

    /// Update worker status
    pub async fn update_worker_status(
        &self,
        worker_id: Uuid,
        status: WorkerStatus,
    ) -> DmrletResult<()> {
        let mut workers = self.workers.write().await;
        if let Some(worker) = workers.get_mut(&worker_id) {
            worker.status = status;
            Ok(())
        } else {
            Err(DmrletError::WorkerNotFound(worker_id.to_string()))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_deployment() {
        let scheduler = Scheduler::new(30000, 100);
        let spec = DeploymentSpec::new("test".to_string(), "model".to_string());

        let id = scheduler.create_deployment(spec).await.unwrap();
        let status = scheduler.get_deployment_status(id).await.unwrap();

        assert_eq!(status.spec.name, "test");
    }

    #[tokio::test]
    async fn test_delete_deployment() {
        let scheduler = Scheduler::new(30000, 100);
        let spec = DeploymentSpec::new("test".to_string(), "model".to_string());

        let id = scheduler.create_deployment(spec).await.unwrap();
        scheduler.delete_deployment(id).await.unwrap();

        let result = scheduler.get_deployment_status(id).await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_scale_deployment() {
        let scheduler = Scheduler::new(30000, 100);
        let mut spec = DeploymentSpec::new("test".to_string(), "model".to_string());
        spec.replicas = 2;

        let id = scheduler.create_deployment(spec).await.unwrap();

        // Scale up
        scheduler.scale_deployment(id, 4).await.unwrap();
        let status = scheduler.get_deployment_status(id).await.unwrap();
        assert_eq!(status.spec.replicas, 4);

        // Scale down
        scheduler.scale_deployment(id, 1).await.unwrap();
        let status = scheduler.get_deployment_status(id).await.unwrap();
        assert_eq!(status.spec.replicas, 1);
    }

    #[tokio::test]
    async fn test_list_deployments() {
        let scheduler = Scheduler::new(30000, 100);

        let spec1 = DeploymentSpec::new("test1".to_string(), "model1".to_string());
        let spec2 = DeploymentSpec::new("test2".to_string(), "model2".to_string());

        scheduler.create_deployment(spec1).await.unwrap();
        scheduler.create_deployment(spec2).await.unwrap();

        let list = scheduler.list_deployments().await;
        assert_eq!(list.len(), 2);
    }
}
