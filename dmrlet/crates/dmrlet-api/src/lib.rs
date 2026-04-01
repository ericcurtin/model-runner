//! dmrlet-api: REST API server for dmrlet
//!
//! This crate provides the REST API for interacting with dmrlet:
//! - Deployment management
//! - Worker listing
//! - System status

pub mod rest;

pub use rest::create_router;
