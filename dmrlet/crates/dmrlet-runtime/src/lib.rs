//! dmrlet-runtime: Runtime abstraction layer
//!
//! This crate provides runtime implementations for running inference workers:
//! - Process-based runtime for macOS/Windows
//! - Container-based runtime for Linux (containerd)

pub mod process;
pub mod traits;

pub use process::ProcessRuntime;
pub use traits::Runtime;
