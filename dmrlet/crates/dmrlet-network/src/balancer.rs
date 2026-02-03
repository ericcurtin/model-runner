//! Load balancing strategies

use dmrlet_core::{Endpoint, LoadBalanceStrategy};
use std::sync::atomic::{AtomicUsize, Ordering};
use tracing::debug;

/// Load balancer for distributing requests across workers
pub struct LoadBalancer {
    /// Load balancing strategy
    strategy: LoadBalanceStrategy,
    /// Counter for round-robin
    counter: AtomicUsize,
}

impl LoadBalancer {
    /// Create a new load balancer
    pub fn new(strategy: LoadBalanceStrategy) -> Self {
        Self {
            strategy,
            counter: AtomicUsize::new(0),
        }
    }

    /// Select an endpoint from the list
    pub fn select<'a>(&self, endpoints: &'a [Endpoint]) -> Option<&'a Endpoint> {
        if endpoints.is_empty() {
            return None;
        }

        let index = match self.strategy {
            LoadBalanceStrategy::RoundRobin => {
                let idx = self.counter.fetch_add(1, Ordering::Relaxed) % endpoints.len();
                idx
            }
            LoadBalanceStrategy::Random => {
                use std::time::{SystemTime, UNIX_EPOCH};
                let seed = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap()
                    .subsec_nanos() as usize;
                seed % endpoints.len()
            }
            LoadBalanceStrategy::LeastConnections => {
                // For now, fallback to round-robin
                // A real implementation would track connection counts
                let idx = self.counter.fetch_add(1, Ordering::Relaxed) % endpoints.len();
                idx
            }
        };

        debug!(
            strategy = ?self.strategy,
            selected_index = index,
            total_endpoints = endpoints.len(),
            "Selected endpoint"
        );

        endpoints.get(index)
    }

    /// Get the current strategy
    pub fn strategy(&self) -> LoadBalanceStrategy {
        self.strategy
    }
}

impl Default for LoadBalancer {
    fn default() -> Self {
        Self::new(LoadBalanceStrategy::RoundRobin)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_endpoints() -> Vec<Endpoint> {
        vec![
            Endpoint::new("127.0.0.1".to_string(), 30000),
            Endpoint::new("127.0.0.1".to_string(), 30001),
            Endpoint::new("127.0.0.1".to_string(), 30002),
        ]
    }

    #[test]
    fn test_round_robin() {
        let lb = LoadBalancer::new(LoadBalanceStrategy::RoundRobin);
        let endpoints = create_test_endpoints();

        let e1 = lb.select(&endpoints).unwrap();
        let e2 = lb.select(&endpoints).unwrap();
        let e3 = lb.select(&endpoints).unwrap();
        let e4 = lb.select(&endpoints).unwrap();

        assert_eq!(e1.port, 30000);
        assert_eq!(e2.port, 30001);
        assert_eq!(e3.port, 30002);
        assert_eq!(e4.port, 30000); // Wraps around
    }

    #[test]
    fn test_empty_endpoints() {
        let lb = LoadBalancer::default();
        let endpoints: Vec<Endpoint> = vec![];

        assert!(lb.select(&endpoints).is_none());
    }
}
