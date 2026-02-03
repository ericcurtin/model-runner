//! dmrlet-core: Core types and traits for dmrlet orchestrator
//!
//! This crate provides the fundamental types used throughout the dmrlet system:
//! - Model and deployment specifications
//! - Worker status and endpoint information
//! - Configuration types
//! - Error handling
//! - GPU detection and allocation

pub mod config;
pub mod error;
pub mod gpu;
pub mod model;

pub use config::*;
pub use error::*;
pub use gpu::*;
pub use model::*;
