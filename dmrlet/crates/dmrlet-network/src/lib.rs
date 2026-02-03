//! dmrlet-network: Networking and load balancing
//!
//! This crate provides networking functionality:
//! - Health checking for workers
//! - L7 load balancing
//! - Service discovery

pub mod balancer;
pub mod discovery;
pub mod health;

pub use balancer::LoadBalancer;
pub use discovery::ServiceDiscovery;
pub use health::HealthChecker;
