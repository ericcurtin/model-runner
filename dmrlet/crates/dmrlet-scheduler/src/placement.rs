//! Worker placement decisions

use dmrlet_core::DeploymentSpec;

/// Placement decision for a worker
#[derive(Debug, Clone)]
pub struct PlacementDecision {
    /// Assigned GPU indices
    pub gpu_ids: Vec<u32>,
    /// Assigned port number
    pub port: u16,
    /// Worker index
    pub worker_index: u32,
}

/// Strategy for making placement decisions
pub trait PlacementStrategy: Send + Sync {
    /// Make a placement decision for a new worker
    fn place(
        &self,
        spec: &DeploymentSpec,
        existing_worker_count: u32,
        available_gpus: &[u32],
        available_ports: &[u16],
    ) -> Option<PlacementDecision>;
}

/// Default placement strategy
pub struct DefaultPlacementStrategy;

impl PlacementStrategy for DefaultPlacementStrategy {
    fn place(
        &self,
        spec: &DeploymentSpec,
        existing_worker_count: u32,
        available_gpus: &[u32],
        available_ports: &[u16],
    ) -> Option<PlacementDecision> {
        // Check if we have resources
        if available_ports.is_empty() {
            return None;
        }

        let gpu_count = spec.resources.gpu_count as usize;
        if gpu_count > 0 && available_gpus.len() < gpu_count {
            return None;
        }

        let gpu_ids = if gpu_count > 0 {
            available_gpus[..gpu_count].to_vec()
        } else {
            Vec::new()
        };

        Some(PlacementDecision {
            gpu_ids,
            port: available_ports[0],
            worker_index: existing_worker_count,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_placement_no_gpus() {
        let spec = DeploymentSpec::new("test".to_string(), "model".to_string());
        let strategy = DefaultPlacementStrategy;

        let decision = strategy.place(&spec, 0, &[], &[30000, 30001]);
        assert!(decision.is_some());

        let d = decision.unwrap();
        assert!(d.gpu_ids.is_empty());
        assert_eq!(d.port, 30000);
    }

    #[test]
    fn test_default_placement_with_gpus() {
        let mut spec = DeploymentSpec::new("test".to_string(), "model".to_string());
        spec.resources.gpu_count = 2;

        let strategy = DefaultPlacementStrategy;
        let decision = strategy.place(&spec, 0, &[0, 1, 2, 3], &[30000]);

        assert!(decision.is_some());
        let d = decision.unwrap();
        assert_eq!(d.gpu_ids, vec![0, 1]);
    }

    #[test]
    fn test_placement_insufficient_gpus() {
        let mut spec = DeploymentSpec::new("test".to_string(), "model".to_string());
        spec.resources.gpu_count = 4;

        let strategy = DefaultPlacementStrategy;
        let decision = strategy.place(&spec, 0, &[0, 1], &[30000]);

        assert!(decision.is_none());
    }
}
