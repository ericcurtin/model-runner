//! dmrlet-store: Model storage
//!
//! This crate provides model storage functionality:
//! - Local model caching
//! - OCI-based model handling (placeholder)
//! - LRU eviction

pub mod cache;
pub mod oci;

pub use cache::ModelCache;
pub use oci::OciStore;
