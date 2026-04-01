//! Error types for dmrlet

use thiserror::Error;

/// Main error type for dmrlet
#[derive(Error, Debug)]
pub enum DmrletError {
    /// Configuration error
    #[error("Configuration error: {0}")]
    Config(String),

    /// Runtime error
    #[error("Runtime error: {0}")]
    Runtime(String),

    /// Scheduler error
    #[error("Scheduler error: {0}")]
    Scheduler(String),

    /// Network error
    #[error("Network error: {0}")]
    Network(String),

    /// Storage error
    #[error("Storage error: {0}")]
    Storage(String),

    /// API error
    #[error("API error: {0}")]
    Api(String),

    /// GPU error
    #[error("GPU error: {0}")]
    Gpu(String),

    /// Deployment not found
    #[error("Deployment not found: {0}")]
    DeploymentNotFound(String),

    /// Worker not found
    #[error("Worker not found: {0}")]
    WorkerNotFound(String),

    /// Model not found
    #[error("Model not found: {0}")]
    ModelNotFound(String),

    /// Resource exhausted
    #[error("Resource exhausted: {0}")]
    ResourceExhausted(String),

    /// IO error
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    /// Serialization error
    #[error("Serialization error: {0}")]
    Serialization(String),

    /// Internal error
    #[error("Internal error: {0}")]
    Internal(String),
}

/// Result type for dmrlet operations
pub type DmrletResult<T> = Result<T, DmrletError>;

impl From<serde_json::Error> for DmrletError {
    fn from(err: serde_json::Error) -> Self {
        DmrletError::Serialization(err.to_string())
    }
}

impl From<toml::de::Error> for DmrletError {
    fn from(err: toml::de::Error) -> Self {
        DmrletError::Config(err.to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_display() {
        let err = DmrletError::Config("invalid config".to_string());
        assert_eq!(err.to_string(), "Configuration error: invalid config");
    }

    #[test]
    fn test_error_from_io() {
        let io_err = std::io::Error::new(std::io::ErrorKind::NotFound, "file not found");
        let err: DmrletError = io_err.into();
        assert!(matches!(err, DmrletError::Io(_)));
    }
}
