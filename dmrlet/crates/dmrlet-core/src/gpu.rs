//! GPU detection and allocation

use serde::{Deserialize, Serialize};

/// Represents a GPU device
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GpuDevice {
    /// Device index
    pub index: u32,
    /// Device name
    pub name: String,
    /// Total memory in bytes
    pub memory_total: u64,
    /// Free memory in bytes
    pub memory_free: u64,
    /// GPU vendor
    pub vendor: GpuVendor,
    /// Whether the device is available for allocation
    pub available: bool,
    /// Current utilization percentage (0-100)
    pub utilization: Option<u32>,
}

/// GPU vendor types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum GpuVendor {
    Nvidia,
    Amd,
    Intel,
    Apple,
    Unknown,
}

impl std::fmt::Display for GpuVendor {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            GpuVendor::Nvidia => write!(f, "NVIDIA"),
            GpuVendor::Amd => write!(f, "AMD"),
            GpuVendor::Intel => write!(f, "Intel"),
            GpuVendor::Apple => write!(f, "Apple"),
            GpuVendor::Unknown => write!(f, "Unknown"),
        }
    }
}

/// GPU information for the system
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct GpuInfo {
    /// List of detected GPU devices
    pub devices: Vec<GpuDevice>,
    /// Total number of GPUs
    pub total_count: u32,
    /// Number of available GPUs
    pub available_count: u32,
}

impl GpuInfo {
    /// Create empty GPU info (no GPUs detected)
    pub fn empty() -> Self {
        Self {
            devices: Vec::new(),
            total_count: 0,
            available_count: 0,
        }
    }

    /// Get available device indices
    pub fn available_indices(&self) -> Vec<u32> {
        self.devices
            .iter()
            .filter(|d| d.available)
            .map(|d| d.index)
            .collect()
    }
}

/// Detect GPUs on the system
///
/// This is a platform-specific function that detects available GPUs.
/// On Linux/Windows, it uses NVML for NVIDIA GPUs.
/// On macOS, it detects Apple Silicon GPUs.
pub fn detect_gpus() -> GpuInfo {
    // Try to detect GPUs based on platform
    #[cfg(target_os = "macos")]
    {
        detect_apple_gpus()
    }

    #[cfg(not(target_os = "macos"))]
    {
        detect_nvidia_gpus().unwrap_or_else(|_| GpuInfo::empty())
    }
}

/// Detect Apple Silicon GPUs (macOS only)
#[cfg(target_os = "macos")]
fn detect_apple_gpus() -> GpuInfo {
    // On macOS, we assume Apple Silicon with unified memory
    // The actual GPU capabilities would be detected via Metal APIs
    // For now, we return a single Apple GPU
    let device = GpuDevice {
        index: 0,
        name: "Apple Silicon GPU".to_string(),
        memory_total: 0, // Unified memory, would need sysctl to get
        memory_free: 0,
        vendor: GpuVendor::Apple,
        available: true,
        utilization: None,
    };

    GpuInfo {
        devices: vec![device],
        total_count: 1,
        available_count: 1,
    }
}

/// Detect NVIDIA GPUs using NVML
#[cfg(not(target_os = "macos"))]
fn detect_nvidia_gpus() -> Result<GpuInfo, crate::DmrletError> {
    // NVML detection would go here
    // For now, return empty as NVML might not be available
    Ok(GpuInfo::empty())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_gpu_info_empty() {
        let info = GpuInfo::empty();
        assert_eq!(info.total_count, 0);
        assert!(info.devices.is_empty());
    }

    #[test]
    fn test_available_indices() {
        let info = GpuInfo {
            devices: vec![
                GpuDevice {
                    index: 0,
                    name: "GPU 0".to_string(),
                    memory_total: 1024,
                    memory_free: 512,
                    vendor: GpuVendor::Nvidia,
                    available: true,
                    utilization: Some(50),
                },
                GpuDevice {
                    index: 1,
                    name: "GPU 1".to_string(),
                    memory_total: 1024,
                    memory_free: 0,
                    vendor: GpuVendor::Nvidia,
                    available: false,
                    utilization: Some(100),
                },
            ],
            total_count: 2,
            available_count: 1,
        };

        let indices = info.available_indices();
        assert_eq!(indices, vec![0]);
    }

    #[test]
    fn test_gpu_vendor_display() {
        assert_eq!(GpuVendor::Nvidia.to_string(), "NVIDIA");
        assert_eq!(GpuVendor::Apple.to_string(), "Apple");
    }
}
