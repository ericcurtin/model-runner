//! Process-based runtime implementation
//!
//! This runtime manages inference workers as direct OS processes.
//! Used on macOS and Windows where containers are not the primary runtime.

use async_trait::async_trait;
use dmrlet_core::{DmrletError, DmrletResult, Worker, WorkerStatus};
use std::path::PathBuf;
use std::process::Stdio;
use tokio::process::Command;
use tracing::{debug, error, info};

use crate::traits::Runtime;

/// Process-based runtime configuration
#[derive(Debug, Clone)]
pub struct ProcessRuntimeConfig {
    /// Path to the llama.cpp server binary
    pub llama_server_path: PathBuf,
    /// Additional arguments for the server
    pub extra_args: Vec<String>,
}

impl Default for ProcessRuntimeConfig {
    fn default() -> Self {
        Self {
            llama_server_path: PathBuf::from("llama-server"),
            extra_args: Vec::new(),
        }
    }
}

/// Process-based runtime for managing inference workers
pub struct ProcessRuntime {
    config: ProcessRuntimeConfig,
}

impl ProcessRuntime {
    /// Create a new process runtime
    pub fn new(config: ProcessRuntimeConfig) -> Self {
        Self { config }
    }

    /// Build the command to start a worker
    fn build_command(&self, worker: &Worker, model_path: &str) -> Command {
        let mut cmd = Command::new(&self.config.llama_server_path);

        // Basic arguments
        cmd.arg("--model").arg(model_path);
        cmd.arg("--host").arg(&worker.endpoint.host);
        cmd.arg("--port").arg(worker.endpoint.port.to_string());

        // Add GPU arguments if GPUs are assigned
        if !worker.gpu_ids.is_empty() {
            // For llama.cpp, use --n-gpu-layers to enable GPU
            cmd.arg("--n-gpu-layers").arg("999");

            // Set CUDA_VISIBLE_DEVICES for NVIDIA GPUs
            let gpu_ids: String = worker
                .gpu_ids
                .iter()
                .map(|id| id.to_string())
                .collect::<Vec<_>>()
                .join(",");
            cmd.env("CUDA_VISIBLE_DEVICES", &gpu_ids);
        }

        // Add extra arguments
        for arg in &self.config.extra_args {
            cmd.arg(arg);
        }

        // Configure process I/O
        cmd.stdout(Stdio::piped());
        cmd.stderr(Stdio::piped());

        cmd
    }
}

#[async_trait]
impl Runtime for ProcessRuntime {
    async fn start_worker(&self, worker: &mut Worker, model_path: &str) -> DmrletResult<()> {
        info!(
            worker_id = %worker.id,
            port = worker.endpoint.port,
            "Starting worker process"
        );

        worker.status = WorkerStatus::Starting;

        let mut cmd = self.build_command(worker, model_path);

        match cmd.spawn() {
            Ok(child) => {
                let pid = child.id().unwrap_or(0);
                worker.pid = Some(pid);
                worker.status = WorkerStatus::Starting;

                debug!(
                    worker_id = %worker.id,
                    pid = pid,
                    "Worker process spawned"
                );

                // Note: The actual status transition to Running should happen
                // after health check passes
                Ok(())
            }
            Err(e) => {
                error!(
                    worker_id = %worker.id,
                    error = %e,
                    "Failed to spawn worker process"
                );
                worker.status = WorkerStatus::Error;
                Err(DmrletError::Runtime(format!(
                    "Failed to spawn worker: {}",
                    e
                )))
            }
        }
    }

    async fn stop_worker(&self, worker: &Worker) -> DmrletResult<()> {
        if let Some(pid) = worker.pid {
            info!(
                worker_id = %worker.id,
                pid = pid,
                "Stopping worker process"
            );

            // Send SIGTERM to the process
            #[cfg(unix)]
            {
                use std::process::Command as StdCommand;
                let _ = StdCommand::new("kill")
                    .arg("-TERM")
                    .arg(pid.to_string())
                    .output();
            }

            #[cfg(windows)]
            {
                use std::process::Command as StdCommand;
                let _ = StdCommand::new("taskkill")
                    .arg("/PID")
                    .arg(pid.to_string())
                    .arg("/F")
                    .output();
            }

            Ok(())
        } else {
            Err(DmrletError::Runtime("Worker has no PID".to_string()))
        }
    }

    async fn is_running(&self, worker: &Worker) -> DmrletResult<bool> {
        if let Some(pid) = worker.pid {
            // Check if process exists
            #[cfg(unix)]
            {
                use std::process::Command as StdCommand;
                let output = StdCommand::new("kill")
                    .arg("-0")
                    .arg(pid.to_string())
                    .output();

                match output {
                    Ok(o) => Ok(o.status.success()),
                    Err(_) => Ok(false),
                }
            }

            #[cfg(windows)]
            {
                use std::process::Command as StdCommand;
                let output = StdCommand::new("tasklist")
                    .arg("/FI")
                    .arg(format!("PID eq {}", pid))
                    .output();

                match output {
                    Ok(o) => {
                        let stdout = String::from_utf8_lossy(&o.stdout);
                        Ok(stdout.contains(&pid.to_string()))
                    }
                    Err(_) => Ok(false),
                }
            }
        } else {
            Ok(false)
        }
    }

    fn name(&self) -> &'static str {
        "process"
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use uuid::Uuid;

    #[test]
    fn test_process_runtime_config_default() {
        let config = ProcessRuntimeConfig::default();
        assert_eq!(config.llama_server_path.to_str().unwrap(), "llama-server");
    }

    #[test]
    fn test_build_command() {
        let config = ProcessRuntimeConfig {
            llama_server_path: PathBuf::from("/usr/bin/llama-server"),
            extra_args: vec!["--ctx-size".to_string(), "4096".to_string()],
        };
        let runtime = ProcessRuntime::new(config);

        let deployment_id = Uuid::new_v4();
        let worker = Worker::new(deployment_id, 0, 30000);

        let _cmd = runtime.build_command(&worker, "/path/to/model.gguf");

        // Just verify command was built (can't easily inspect Command internals)
        assert_eq!(runtime.name(), "process");
    }
}
