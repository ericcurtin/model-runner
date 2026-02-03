//! Service discovery for dmrlet

use dmrlet_core::Endpoint;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::debug;
use uuid::Uuid;

type EndpointsMap = HashMap<Uuid, Vec<Endpoint>>;

/// Service discovery registry
pub struct ServiceDiscovery {
    /// Endpoints indexed by deployment ID
    endpoints: Arc<RwLock<EndpointsMap>>,
}

impl ServiceDiscovery {
    /// Create a new service discovery registry
    pub fn new() -> Self {
        Self {
            endpoints: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// Register an endpoint for a deployment
    pub async fn register(&self, deployment_id: Uuid, endpoint: Endpoint) {
        let mut endpoints = self.endpoints.write().await;
        endpoints
            .entry(deployment_id)
            .or_insert_with(Vec::new)
            .push(endpoint.clone());

        debug!(
            deployment_id = %deployment_id,
            endpoint = %endpoint.url(),
            "Registered endpoint"
        );
    }

    /// Unregister an endpoint for a deployment
    pub async fn unregister(&self, deployment_id: Uuid, port: u16) {
        let mut endpoints = self.endpoints.write().await;
        if let Some(eps) = endpoints.get_mut(&deployment_id) {
            eps.retain(|e| e.port != port);
            if eps.is_empty() {
                endpoints.remove(&deployment_id);
            }
        }

        debug!(
            deployment_id = %deployment_id,
            port = port,
            "Unregistered endpoint"
        );
    }

    /// Get all endpoints for a deployment
    pub async fn get_endpoints(&self, deployment_id: Uuid) -> Vec<Endpoint> {
        let endpoints = self.endpoints.read().await;
        endpoints.get(&deployment_id).cloned().unwrap_or_default()
    }

    /// Get all endpoints across all deployments
    pub async fn get_all_endpoints(&self) -> Vec<Endpoint> {
        let endpoints = self.endpoints.read().await;
        endpoints.values().flatten().cloned().collect()
    }

    /// Clear all endpoints for a deployment
    pub async fn clear(&self, deployment_id: Uuid) {
        let mut endpoints = self.endpoints.write().await;
        endpoints.remove(&deployment_id);

        debug!(deployment_id = %deployment_id, "Cleared all endpoints");
    }
}

impl Default for ServiceDiscovery {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_register_and_get() {
        let sd = ServiceDiscovery::new();
        let deployment_id = Uuid::new_v4();
        let endpoint = Endpoint::new("127.0.0.1".to_string(), 30000);

        sd.register(deployment_id, endpoint).await;

        let endpoints = sd.get_endpoints(deployment_id).await;
        assert_eq!(endpoints.len(), 1);
        assert_eq!(endpoints[0].port, 30000);
    }

    #[tokio::test]
    async fn test_unregister() {
        let sd = ServiceDiscovery::new();
        let deployment_id = Uuid::new_v4();

        sd.register(deployment_id, Endpoint::new("127.0.0.1".to_string(), 30000))
            .await;
        sd.register(deployment_id, Endpoint::new("127.0.0.1".to_string(), 30001))
            .await;

        sd.unregister(deployment_id, 30000).await;

        let endpoints = sd.get_endpoints(deployment_id).await;
        assert_eq!(endpoints.len(), 1);
        assert_eq!(endpoints[0].port, 30001);
    }

    #[tokio::test]
    async fn test_get_all_endpoints() {
        let sd = ServiceDiscovery::new();
        let id1 = Uuid::new_v4();
        let id2 = Uuid::new_v4();

        sd.register(id1, Endpoint::new("127.0.0.1".to_string(), 30000))
            .await;
        sd.register(id2, Endpoint::new("127.0.0.1".to_string(), 30001))
            .await;

        let all = sd.get_all_endpoints().await;
        assert_eq!(all.len(), 2);
    }
}
