//! Configuration types for dmrlet

use serde::{Deserialize, Serialize};
use std::path::PathBuf;

/// Main daemon configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DaemonConfig {
    /// API server configuration
    pub api: ApiConfig,
    /// Runtime configuration
    pub runtime: RuntimeConfig,
    /// Network configuration
    pub network: NetworkConfig,
    /// Storage configuration
    pub storage: StorageConfig,
    /// Logging configuration
    pub logging: LoggingConfig,
}

impl Default for DaemonConfig {
    fn default() -> Self {
        Self {
            api: ApiConfig::default(),
            runtime: RuntimeConfig::default(),
            network: NetworkConfig::default(),
            storage: StorageConfig::default(),
            logging: LoggingConfig::default(),
        }
    }
}

impl DaemonConfig {
    /// Load configuration from a TOML file
    pub fn from_file(path: &std::path::Path) -> Result<Self, crate::DmrletError> {
        let content = std::fs::read_to_string(path).map_err(|e| {
            crate::DmrletError::Config(format!("Failed to read config file: {}", e))
        })?;
        toml::from_str(&content)
            .map_err(|e| crate::DmrletError::Config(format!("Failed to parse config: {}", e)))
    }
}

/// API server configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiConfig {
    /// Address to bind the REST API server
    pub rest_address: String,
    /// Port for the REST API server
    pub rest_port: u16,
    /// Address to bind the gRPC server
    pub grpc_address: String,
    /// Port for the gRPC server
    pub grpc_port: u16,
    /// Enable CORS
    pub cors_enabled: bool,
    /// Allowed CORS origins
    pub cors_origins: Vec<String>,
}

impl Default for ApiConfig {
    fn default() -> Self {
        Self {
            rest_address: "0.0.0.0".to_string(),
            rest_port: 9090,
            grpc_address: "0.0.0.0".to_string(),
            grpc_port: 9091,
            cors_enabled: true,
            cors_origins: vec!["*".to_string()],
        }
    }
}

/// Runtime configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuntimeConfig {
    /// Runtime type (process or containerd)
    pub runtime_type: RuntimeType,
    /// Path to llama.cpp server binary
    pub llama_server_path: Option<PathBuf>,
    /// Base port for worker allocation
    pub worker_base_port: u16,
    /// Maximum number of workers
    pub max_workers: u32,
    /// Worker idle timeout in seconds (for eviction)
    pub worker_idle_timeout_secs: u64,
}

impl Default for RuntimeConfig {
    fn default() -> Self {
        Self {
            runtime_type: RuntimeType::Process,
            llama_server_path: None,
            worker_base_port: 30000,
            max_workers: 100,
            worker_idle_timeout_secs: 3600,
        }
    }
}

/// Runtime type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum RuntimeType {
    /// Process-based runtime (macOS, Windows)
    Process,
    /// Container-based runtime (Linux with containerd)
    Containerd,
}

/// Network configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkConfig {
    /// Address for the load balancer
    pub lb_address: String,
    /// Port for the load balancer
    pub lb_port: u16,
    /// Load balancing strategy
    pub lb_strategy: LoadBalanceStrategy,
    /// Health check interval in seconds
    pub health_check_interval_secs: u64,
    /// Health check timeout in seconds
    pub health_check_timeout_secs: u64,
}

impl Default for NetworkConfig {
    fn default() -> Self {
        Self {
            lb_address: "0.0.0.0".to_string(),
            lb_port: 8080,
            lb_strategy: LoadBalanceStrategy::RoundRobin,
            health_check_interval_secs: 10,
            health_check_timeout_secs: 5,
        }
    }
}

/// Load balancing strategy
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum LoadBalanceStrategy {
    /// Round-robin load balancing
    RoundRobin,
    /// Least connections load balancing
    LeastConnections,
    /// Random load balancing
    Random,
}

/// Storage configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StorageConfig {
    /// Path to model storage directory
    pub models_path: PathBuf,
    /// Maximum cache size in bytes
    pub max_cache_size: u64,
    /// Enable LRU eviction
    pub lru_eviction: bool,
}

impl Default for StorageConfig {
    fn default() -> Self {
        Self {
            models_path: PathBuf::from("/var/lib/dmrlet/models"),
            max_cache_size: 100 * 1024 * 1024 * 1024, // 100 GB
            lru_eviction: true,
        }
    }
}

/// Logging configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoggingConfig {
    /// Log level
    pub level: String,
    /// Log format (json or text)
    pub format: String,
    /// Log file path (if any)
    pub file: Option<PathBuf>,
}

impl Default for LoggingConfig {
    fn default() -> Self {
        Self {
            level: "info".to_string(),
            format: "text".to_string(),
            file: None,
        }
    }
}

/// Deployment configuration file format (TOML)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeploymentConfig {
    /// Deployment settings
    pub deployment: DeploymentSettings,
    /// Resource settings
    pub resources: Option<ResourceSettings>,
    /// Backend settings
    pub backend: Option<BackendSettings>,
    /// Health check settings
    pub health: Option<HealthSettings>,
    /// Auto-scale settings
    pub autoscale: Option<AutoscaleSettings>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeploymentSettings {
    pub name: String,
    pub model: String,
    pub replicas: Option<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceSettings {
    pub memory: Option<String>,
    pub gpu_count: Option<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BackendSettings {
    #[serde(rename = "type")]
    pub backend_type: Option<String>,
    pub context_size: Option<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthSettings {
    pub path: Option<String>,
    pub interval: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AutoscaleSettings {
    pub enabled: Option<bool>,
    pub min_replicas: Option<u32>,
    pub max_replicas: Option<u32>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_daemon_config() {
        let config = DaemonConfig::default();
        assert_eq!(config.api.rest_port, 9090);
        assert_eq!(config.network.lb_port, 8080);
    }

    #[test]
    fn test_deployment_config_parse() {
        let toml_str = r#"
[deployment]
name = "test-service"
model = "ai/llama3:8b"
replicas = 2

[resources]
memory = "16Gi"
gpu_count = 1

[backend]
type = "llama.cpp"
context_size = 4096
"#;
        let config: DeploymentConfig = toml::from_str(toml_str).unwrap();
        assert_eq!(config.deployment.name, "test-service");
        assert_eq!(config.deployment.replicas, Some(2));
    }
}
