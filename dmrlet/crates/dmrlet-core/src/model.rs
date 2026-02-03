//! Model, Worker, and Endpoint type definitions

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// Represents a deployment specification for a model
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeploymentSpec {
    /// Unique identifier for the deployment
    pub id: Uuid,
    /// Human-readable name
    pub name: String,
    /// Model reference (e.g., "ai/llama3:8b")
    pub model: String,
    /// Number of replicas
    pub replicas: u32,
    /// Resource requirements
    pub resources: ResourceRequirements,
    /// Backend configuration
    pub backend: BackendConfig,
    /// Health check configuration
    pub health: HealthConfig,
    /// Auto-scaling configuration
    pub autoscale: Option<AutoscaleConfig>,
    /// Creation timestamp
    pub created_at: DateTime<Utc>,
    /// Last updated timestamp
    pub updated_at: DateTime<Utc>,
}

impl DeploymentSpec {
    /// Create a new deployment spec with default values
    pub fn new(name: String, model: String) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            name,
            model,
            replicas: 1,
            resources: ResourceRequirements::default(),
            backend: BackendConfig::default(),
            health: HealthConfig::default(),
            autoscale: None,
            created_at: now,
            updated_at: now,
        }
    }
}

/// Resource requirements for a deployment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceRequirements {
    /// Memory limit (e.g., "16Gi")
    pub memory: Option<String>,
    /// Number of GPUs required
    pub gpu_count: u32,
    /// Specific GPU IDs to use
    pub gpu_ids: Vec<u32>,
}

impl Default for ResourceRequirements {
    fn default() -> Self {
        Self {
            memory: None,
            gpu_count: 0,
            gpu_ids: Vec::new(),
        }
    }
}

/// Backend configuration for inference
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackendConfig {
    /// Backend type (llama.cpp, vLLM, MLX, etc.)
    #[serde(rename = "type")]
    pub backend_type: BackendType,
    /// Context size for the model
    pub context_size: u32,
    /// Additional backend-specific arguments
    pub extra_args: Vec<String>,
}

impl Default for BackendConfig {
    fn default() -> Self {
        Self {
            backend_type: BackendType::LlamaCpp,
            context_size: 4096,
            extra_args: Vec::new(),
        }
    }
}

/// Supported backend types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum BackendType {
    #[serde(rename = "llama.cpp")]
    LlamaCpp,
    #[serde(rename = "vllm")]
    Vllm,
    #[serde(rename = "mlx")]
    Mlx,
}

impl std::fmt::Display for BackendType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            BackendType::LlamaCpp => write!(f, "llama.cpp"),
            BackendType::Vllm => write!(f, "vLLM"),
            BackendType::Mlx => write!(f, "MLX"),
        }
    }
}

/// Health check configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthConfig {
    /// Health check endpoint path
    pub path: String,
    /// Check interval in seconds
    pub interval_secs: u32,
    /// Timeout in seconds
    pub timeout_secs: u32,
    /// Number of failures before marking unhealthy
    pub failure_threshold: u32,
}

impl Default for HealthConfig {
    fn default() -> Self {
        Self {
            path: "/health".to_string(),
            interval_secs: 10,
            timeout_secs: 5,
            failure_threshold: 3,
        }
    }
}

/// Auto-scaling configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutoscaleConfig {
    /// Whether auto-scaling is enabled
    pub enabled: bool,
    /// Minimum number of replicas
    pub min_replicas: u32,
    /// Maximum number of replicas
    pub max_replicas: u32,
    /// Target CPU utilization percentage
    pub target_cpu_utilization: Option<u32>,
    /// Target memory utilization percentage
    pub target_memory_utilization: Option<u32>,
}

/// Worker represents a running inference server instance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Worker {
    /// Unique worker identifier
    pub id: Uuid,
    /// Deployment this worker belongs to
    pub deployment_id: Uuid,
    /// Worker index within the deployment
    pub index: u32,
    /// Current status
    pub status: WorkerStatus,
    /// Network endpoint
    pub endpoint: Endpoint,
    /// Process ID (for process-based runtime)
    pub pid: Option<u32>,
    /// Container ID (for container-based runtime)
    pub container_id: Option<String>,
    /// Assigned GPU IDs
    pub gpu_ids: Vec<u32>,
    /// Creation timestamp
    pub created_at: DateTime<Utc>,
    /// Last health check timestamp
    pub last_health_check: Option<DateTime<Utc>>,
}

impl Worker {
    /// Create a new worker
    pub fn new(deployment_id: Uuid, index: u32, port: u16) -> Self {
        Self {
            id: Uuid::new_v4(),
            deployment_id,
            index,
            status: WorkerStatus::Pending,
            endpoint: Endpoint::new("127.0.0.1".to_string(), port),
            pid: None,
            container_id: None,
            gpu_ids: Vec::new(),
            created_at: Utc::now(),
            last_health_check: None,
        }
    }

    /// Check if the worker is healthy
    pub fn is_healthy(&self) -> bool {
        matches!(self.status, WorkerStatus::Running)
    }
}

/// Worker status
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum WorkerStatus {
    /// Worker is being created
    Pending,
    /// Worker is starting up
    Starting,
    /// Worker is running and healthy
    Running,
    /// Worker is unhealthy
    Unhealthy,
    /// Worker is being terminated
    Terminating,
    /// Worker has terminated
    Terminated,
    /// Worker encountered an error
    Error,
}

impl std::fmt::Display for WorkerStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            WorkerStatus::Pending => write!(f, "Pending"),
            WorkerStatus::Starting => write!(f, "Starting"),
            WorkerStatus::Running => write!(f, "Running"),
            WorkerStatus::Unhealthy => write!(f, "Unhealthy"),
            WorkerStatus::Terminating => write!(f, "Terminating"),
            WorkerStatus::Terminated => write!(f, "Terminated"),
            WorkerStatus::Error => write!(f, "Error"),
        }
    }
}

/// Network endpoint for a worker
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Endpoint {
    /// Host address
    pub host: String,
    /// Port number
    pub port: u16,
    /// Whether TLS is enabled
    pub tls: bool,
}

impl Endpoint {
    /// Create a new endpoint
    pub fn new(host: String, port: u16) -> Self {
        Self {
            host,
            port,
            tls: false,
        }
    }

    /// Get the URL for this endpoint
    pub fn url(&self) -> String {
        let scheme = if self.tls { "https" } else { "http" };
        format!("{}://{}:{}", scheme, self.host, self.port)
    }
}

/// Deployment status summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeploymentStatus {
    /// Deployment specification
    pub spec: DeploymentSpec,
    /// Current workers
    pub workers: Vec<Worker>,
    /// Number of ready replicas
    pub ready_replicas: u32,
    /// Number of available replicas
    pub available_replicas: u32,
    /// Overall deployment phase
    pub phase: DeploymentPhase,
}

impl DeploymentStatus {
    /// Create a new deployment status
    pub fn new(spec: DeploymentSpec, workers: Vec<Worker>) -> Self {
        let ready_replicas = workers.iter().filter(|w| w.is_healthy()).count() as u32;
        let available_replicas = workers
            .iter()
            .filter(|w| !matches!(w.status, WorkerStatus::Terminated | WorkerStatus::Error))
            .count() as u32;

        let phase = if ready_replicas == spec.replicas {
            DeploymentPhase::Ready
        } else if ready_replicas > 0 {
            DeploymentPhase::Progressing
        } else if workers.is_empty() {
            DeploymentPhase::Pending
        } else {
            DeploymentPhase::Progressing
        };

        Self {
            spec,
            workers,
            ready_replicas,
            available_replicas,
            phase,
        }
    }
}

/// Deployment phase
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum DeploymentPhase {
    /// Deployment is pending
    Pending,
    /// Deployment is progressing
    Progressing,
    /// All replicas are ready
    Ready,
    /// Deployment failed
    Failed,
}

impl std::fmt::Display for DeploymentPhase {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            DeploymentPhase::Pending => write!(f, "Pending"),
            DeploymentPhase::Progressing => write!(f, "Progressing"),
            DeploymentPhase::Ready => write!(f, "Ready"),
            DeploymentPhase::Failed => write!(f, "Failed"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_deployment_spec_new() {
        let spec = DeploymentSpec::new("test-deployment".to_string(), "ai/llama3:8b".to_string());
        assert_eq!(spec.name, "test-deployment");
        assert_eq!(spec.model, "ai/llama3:8b");
        assert_eq!(spec.replicas, 1);
    }

    #[test]
    fn test_worker_new() {
        let deployment_id = Uuid::new_v4();
        let worker = Worker::new(deployment_id, 0, 30000);
        assert_eq!(worker.deployment_id, deployment_id);
        assert_eq!(worker.index, 0);
        assert_eq!(worker.endpoint.port, 30000);
        assert_eq!(worker.status, WorkerStatus::Pending);
    }

    #[test]
    fn test_endpoint_url() {
        let endpoint = Endpoint::new("127.0.0.1".to_string(), 30000);
        assert_eq!(endpoint.url(), "http://127.0.0.1:30000");

        let tls_endpoint = Endpoint {
            host: "localhost".to_string(),
            port: 443,
            tls: true,
        };
        assert_eq!(tls_endpoint.url(), "https://localhost:443");
    }

    #[test]
    fn test_deployment_status() {
        let spec = DeploymentSpec::new("test".to_string(), "model".to_string());
        let workers = vec![];
        let status = DeploymentStatus::new(spec, workers);
        assert_eq!(status.phase, DeploymentPhase::Pending);
    }
}
