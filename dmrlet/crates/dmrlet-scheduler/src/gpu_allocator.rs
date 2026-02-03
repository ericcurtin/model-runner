//! GPU allocator for tracking and assigning GPU devices

use dmrlet_core::{DmrletError, DmrletResult, GpuDevice, GpuInfo};
use std::collections::HashSet;
use tracing::{debug, info};

/// GPU allocator that tracks GPU device assignments
pub struct GpuAllocator {
    /// Available GPU devices
    devices: Vec<GpuDevice>,
    /// Set of allocated GPU indices
    allocated: HashSet<u32>,
}

impl GpuAllocator {
    /// Create a new GPU allocator from GPU info
    pub fn new(gpu_info: GpuInfo) -> Self {
        Self {
            devices: gpu_info.devices,
            allocated: HashSet::new(),
        }
    }

    /// Create an empty allocator (no GPUs)
    pub fn empty() -> Self {
        Self {
            devices: Vec::new(),
            allocated: HashSet::new(),
        }
    }

    /// Get the total number of GPUs
    pub fn total_count(&self) -> u32 {
        self.devices.len() as u32
    }

    /// Get the number of available GPUs
    pub fn available_count(&self) -> u32 {
        self.devices
            .iter()
            .filter(|d| d.available && !self.allocated.contains(&d.index))
            .count() as u32
    }

    /// Allocate the requested number of GPUs
    ///
    /// Returns the indices of the allocated GPUs
    pub fn allocate(&mut self, count: u32) -> DmrletResult<Vec<u32>> {
        if count == 0 {
            return Ok(Vec::new());
        }

        let available: Vec<u32> = self
            .devices
            .iter()
            .filter(|d| d.available && !self.allocated.contains(&d.index))
            .map(|d| d.index)
            .collect();

        if available.len() < count as usize {
            return Err(DmrletError::ResourceExhausted(format!(
                "Not enough GPUs available: requested {}, available {}",
                count,
                available.len()
            )));
        }

        let allocated_indices: Vec<u32> = available.into_iter().take(count as usize).collect();

        for idx in &allocated_indices {
            self.allocated.insert(*idx);
        }

        info!(
            gpus = ?allocated_indices,
            "Allocated GPUs"
        );

        Ok(allocated_indices)
    }

    /// Release previously allocated GPUs
    pub fn release(&mut self, indices: &[u32]) {
        for idx in indices {
            if self.allocated.remove(idx) {
                debug!(gpu = idx, "Released GPU");
            }
        }
    }

    /// Get information about all GPUs
    pub fn get_gpu_info(&self) -> Vec<GpuDeviceStatus> {
        self.devices
            .iter()
            .map(|d| GpuDeviceStatus {
                device: d.clone(),
                allocated: self.allocated.contains(&d.index),
            })
            .collect()
    }
}

/// GPU device with allocation status
#[derive(Debug, Clone)]
pub struct GpuDeviceStatus {
    /// Device information
    pub device: GpuDevice,
    /// Whether this device is currently allocated
    pub allocated: bool,
}

#[cfg(test)]
mod tests {
    use super::*;
    use dmrlet_core::GpuVendor;

    fn create_test_gpu_info(count: u32) -> GpuInfo {
        let devices: Vec<GpuDevice> = (0..count)
            .map(|i| GpuDevice {
                index: i,
                name: format!("Test GPU {}", i),
                memory_total: 16 * 1024 * 1024 * 1024,
                memory_free: 16 * 1024 * 1024 * 1024,
                vendor: GpuVendor::Nvidia,
                available: true,
                utilization: Some(0),
            })
            .collect();

        GpuInfo {
            total_count: count,
            available_count: count,
            devices,
        }
    }

    #[test]
    fn test_gpu_allocator_empty() {
        let allocator = GpuAllocator::empty();
        assert_eq!(allocator.total_count(), 0);
        assert_eq!(allocator.available_count(), 0);
    }

    #[test]
    fn test_allocate_gpus() {
        let gpu_info = create_test_gpu_info(4);
        let mut allocator = GpuAllocator::new(gpu_info);

        assert_eq!(allocator.available_count(), 4);

        let allocated = allocator.allocate(2).unwrap();
        assert_eq!(allocated.len(), 2);
        assert_eq!(allocator.available_count(), 2);
    }

    #[test]
    fn test_release_gpus() {
        let gpu_info = create_test_gpu_info(2);
        let mut allocator = GpuAllocator::new(gpu_info);

        let allocated = allocator.allocate(2).unwrap();
        assert_eq!(allocator.available_count(), 0);

        allocator.release(&allocated);
        assert_eq!(allocator.available_count(), 2);
    }

    #[test]
    fn test_allocate_insufficient_gpus() {
        let gpu_info = create_test_gpu_info(2);
        let mut allocator = GpuAllocator::new(gpu_info);

        let result = allocator.allocate(4);
        assert!(result.is_err());
    }
}
