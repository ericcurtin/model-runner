//! Health checking for workers

use dmrlet_core::Endpoint;
use std::time::Duration;
use tracing::{debug, warn};

/// Health checker for workers
pub struct HealthChecker {
    /// HTTP client for health checks
    client: reqwest::Client,
    /// Health check path
    health_path: String,
    /// Timeout duration
    timeout: Duration,
}

impl HealthChecker {
    /// Create a new health checker
    pub fn new(health_path: String, timeout_secs: u64) -> Self {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(timeout_secs))
            .build()
            .expect("Failed to create HTTP client");

        Self {
            client,
            health_path,
            timeout: Duration::from_secs(timeout_secs),
        }
    }

    /// Check the health of an endpoint
    pub async fn check(&self, endpoint: &Endpoint) -> bool {
        let url = format!("{}{}", endpoint.url(), self.health_path);

        match self.client.get(&url).send().await {
            Ok(response) => {
                let healthy = response.status().is_success();
                if healthy {
                    debug!(endpoint = %url, "Health check passed");
                } else {
                    warn!(
                        endpoint = %url,
                        status = %response.status(),
                        "Health check failed"
                    );
                }
                healthy
            }
            Err(e) => {
                warn!(
                    endpoint = %url,
                    error = %e,
                    "Health check error"
                );
                false
            }
        }
    }

    /// Get the timeout duration
    pub fn timeout(&self) -> Duration {
        self.timeout
    }
}

impl Default for HealthChecker {
    fn default() -> Self {
        Self::new("/health".to_string(), 5)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_health_checker_creation() {
        let checker = HealthChecker::new("/health".to_string(), 10);
        assert_eq!(checker.timeout(), Duration::from_secs(10));
    }
}
