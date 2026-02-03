//! dmrlet-scheduler: GPU-aware scheduler for dmrlet
//!
//! This crate provides scheduling logic for placing workers on available resources:
//! - GPU allocation and tracking
//! - Worker placement decisions
//! - Resource management

pub mod gpu_allocator;
pub mod placement;
pub mod scheduler;

pub use gpu_allocator::GpuAllocator;
pub use placement::PlacementDecision;
pub use scheduler::Scheduler;
