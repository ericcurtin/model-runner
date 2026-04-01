//! REST API handlers

use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::Json,
    routing::{delete, get, post},
    Router,
};
use dmrlet_core::{
    BackendType, DeploymentSpec, DeploymentStatus, DmrletError, Endpoint, GpuInfo,
    ResourceRequirements, detect_gpus,
};
use dmrlet_scheduler::Scheduler;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::info;
use uuid::Uuid;

/// Application state shared across handlers
pub struct AppState {
    pub scheduler: Arc<Scheduler>,
}

/// Create the API router
pub fn create_router(scheduler: Arc<Scheduler>) -> Router {
    let state = Arc::new(AppState { scheduler });

    Router::new()
        .route("/api/v1/deployments", post(create_deployment))
        .route("/api/v1/deployments", get(list_deployments))
        .route("/api/v1/deployments/:id", get(get_deployment))
        .route("/api/v1/deployments/:id", delete(delete_deployment))
        .route("/api/v1/deployments/:id/scale", post(scale_deployment))
        .route("/api/v1/deployments/:id/workers", get(get_workers))
        .route("/api/v1/endpoints", get(get_endpoints))
        .route("/api/v1/gpus", get(get_gpus))
        .route("/api/v1/status", get(get_status))
        .with_state(state)
}

/// Request to create a deployment
#[derive(Debug, Deserialize)]
pub struct CreateDeploymentRequest {
    /// Deployment name
    pub name: String,
    /// Model reference
    pub model: String,
    /// Number of replicas
    #[serde(default = "default_replicas")]
    pub replicas: u32,
    /// Number of GPUs per worker
    #[serde(default)]
    pub gpu_count: u32,
    /// Backend type
    #[serde(default)]
    pub backend: String,
    /// Context size
    #[serde(default = "default_context_size")]
    pub context_size: u32,
}

fn default_replicas() -> u32 {
    1
}

fn default_context_size() -> u32 {
    4096
}

/// Response for a deployment
#[derive(Debug, Serialize)]
pub struct DeploymentResponse {
    pub id: Uuid,
    pub name: String,
    pub model: String,
    pub replicas: u32,
    pub ready_replicas: u32,
    pub phase: String,
}

impl From<DeploymentStatus> for DeploymentResponse {
    fn from(status: DeploymentStatus) -> Self {
        Self {
            id: status.spec.id,
            name: status.spec.name,
            model: status.spec.model,
            replicas: status.spec.replicas,
            ready_replicas: status.ready_replicas,
            phase: status.phase.to_string(),
        }
    }
}

/// Create a new deployment
async fn create_deployment(
    State(state): State<Arc<AppState>>,
    Json(req): Json<CreateDeploymentRequest>,
) -> Result<Json<DeploymentResponse>, (StatusCode, String)> {
    info!(
        name = %req.name,
        model = %req.model,
        replicas = req.replicas,
        "Creating deployment"
    );

    let mut spec = DeploymentSpec::new(req.name, req.model);
    spec.replicas = req.replicas;
    spec.resources = ResourceRequirements {
        memory: None,
        gpu_count: req.gpu_count,
        gpu_ids: Vec::new(),
    };
    spec.backend.context_size = req.context_size;

    if !req.backend.is_empty() {
        spec.backend.backend_type = match req.backend.to_lowercase().as_str() {
            "llama.cpp" | "llamacpp" => BackendType::LlamaCpp,
            "vllm" => BackendType::Vllm,
            "mlx" => BackendType::Mlx,
            _ => BackendType::LlamaCpp,
        };
    }

    let id = state
        .scheduler
        .create_deployment(spec)
        .await
        .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    let status = state
        .scheduler
        .get_deployment_status(id)
        .await
        .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    Ok(Json(DeploymentResponse::from(status)))
}

/// List all deployments
async fn list_deployments(
    State(state): State<Arc<AppState>>,
) -> Result<Json<Vec<DeploymentResponse>>, (StatusCode, String)> {
    let deployments = state.scheduler.list_deployments().await;
    let responses: Vec<DeploymentResponse> = deployments
        .into_iter()
        .map(DeploymentResponse::from)
        .collect();
    Ok(Json(responses))
}

/// Get a specific deployment
async fn get_deployment(
    State(state): State<Arc<AppState>>,
    Path(id): Path<Uuid>,
) -> Result<Json<DeploymentResponse>, (StatusCode, String)> {
    let status = state
        .scheduler
        .get_deployment_status(id)
        .await
        .map_err(|e| match e {
            DmrletError::DeploymentNotFound(_) => (StatusCode::NOT_FOUND, e.to_string()),
            _ => (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
        })?;

    Ok(Json(DeploymentResponse::from(status)))
}

/// Delete a deployment
async fn delete_deployment(
    State(state): State<Arc<AppState>>,
    Path(id): Path<Uuid>,
) -> Result<StatusCode, (StatusCode, String)> {
    info!(deployment_id = %id, "Deleting deployment");

    state
        .scheduler
        .delete_deployment(id)
        .await
        .map_err(|e| match e {
            DmrletError::DeploymentNotFound(_) => (StatusCode::NOT_FOUND, e.to_string()),
            _ => (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
        })?;

    Ok(StatusCode::NO_CONTENT)
}

/// Request to scale a deployment
#[derive(Debug, Deserialize)]
pub struct ScaleRequest {
    pub replicas: u32,
}

/// Scale a deployment
async fn scale_deployment(
    State(state): State<Arc<AppState>>,
    Path(id): Path<Uuid>,
    Json(req): Json<ScaleRequest>,
) -> Result<Json<DeploymentResponse>, (StatusCode, String)> {
    info!(
        deployment_id = %id,
        replicas = req.replicas,
        "Scaling deployment"
    );

    state
        .scheduler
        .scale_deployment(id, req.replicas)
        .await
        .map_err(|e| match e {
            DmrletError::DeploymentNotFound(_) => (StatusCode::NOT_FOUND, e.to_string()),
            _ => (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
        })?;

    let status = state
        .scheduler
        .get_deployment_status(id)
        .await
        .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    Ok(Json(DeploymentResponse::from(status)))
}

/// Worker response
#[derive(Debug, Serialize)]
pub struct WorkerResponse {
    pub id: Uuid,
    pub index: u32,
    pub status: String,
    pub endpoint: String,
    pub gpu_ids: Vec<u32>,
}

/// Get workers for a deployment
async fn get_workers(
    State(state): State<Arc<AppState>>,
    Path(id): Path<Uuid>,
) -> Result<Json<Vec<WorkerResponse>>, (StatusCode, String)> {
    let workers = state.scheduler.get_workers(id).await;

    let responses: Vec<WorkerResponse> = workers
        .into_iter()
        .map(|w| WorkerResponse {
            id: w.id,
            index: w.index,
            status: w.status.to_string(),
            endpoint: w.endpoint.url(),
            gpu_ids: w.gpu_ids,
        })
        .collect();

    Ok(Json(responses))
}

/// Get all endpoints for direct access
async fn get_endpoints(
    State(state): State<Arc<AppState>>,
) -> Result<Json<Vec<Endpoint>>, (StatusCode, String)> {
    let endpoints = state.scheduler.get_all_endpoints().await;
    Ok(Json(endpoints))
}

/// Get GPU information
async fn get_gpus() -> Result<Json<GpuInfo>, (StatusCode, String)> {
    let gpu_info = detect_gpus();
    Ok(Json(gpu_info))
}

/// System status response
#[derive(Debug, Serialize)]
pub struct StatusResponse {
    pub version: String,
    pub deployments: usize,
    pub workers: usize,
    pub gpus: GpuInfo,
}

/// Get system status
async fn get_status(
    State(state): State<Arc<AppState>>,
) -> Result<Json<StatusResponse>, (StatusCode, String)> {
    let deployments = state.scheduler.list_deployments().await;
    let worker_count: usize = deployments.iter().map(|d| d.workers.len()).sum();

    Ok(Json(StatusResponse {
        version: env!("CARGO_PKG_VERSION").to_string(),
        deployments: deployments.len(),
        workers: worker_count,
        gpus: detect_gpus(),
    }))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_router() {
        let scheduler = Arc::new(Scheduler::new(30000, 100));
        let _router = create_router(scheduler);
    }
}
