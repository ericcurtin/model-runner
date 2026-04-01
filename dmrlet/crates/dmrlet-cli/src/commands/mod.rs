//! CLI commands implementation

use anyhow::Result;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// API client for communicating with the daemon
pub struct ApiClient {
    base_url: String,
    client: reqwest::Client,
}

impl ApiClient {
    pub fn new(base_url: &str) -> Self {
        Self {
            base_url: base_url.trim_end_matches('/').to_string(),
            client: reqwest::Client::new(),
        }
    }

    pub fn url(&self, path: &str) -> String {
        format!("{}{}", self.base_url, path)
    }
}

/// Deployment response from API
#[derive(Debug, Deserialize)]
pub struct DeploymentResponse {
    pub id: Uuid,
    pub name: String,
    pub model: String,
    pub replicas: u32,
    pub ready_replicas: u32,
    pub phase: String,
}

/// Worker response from API
#[derive(Debug, Deserialize)]
pub struct WorkerResponse {
    #[allow(dead_code)]
    pub id: Uuid,
    pub index: u32,
    pub status: String,
    pub endpoint: String,
    pub gpu_ids: Vec<u32>,
}

/// Endpoint response
#[derive(Debug, Deserialize)]
pub struct EndpointResponse {
    pub host: String,
    pub port: u16,
    pub tls: bool,
}

/// GPU device response
#[derive(Debug, Deserialize)]
pub struct GpuDevice {
    pub index: u32,
    pub name: String,
    pub memory_total: u64,
    pub memory_free: u64,
    pub vendor: String,
    pub available: bool,
}

/// GPU info response
#[derive(Debug, Deserialize)]
pub struct GpuInfo {
    pub devices: Vec<GpuDevice>,
    pub total_count: u32,
    pub available_count: u32,
}

/// Status response
#[derive(Debug, Deserialize)]
pub struct StatusResponse {
    pub version: String,
    pub deployments: usize,
    pub workers: usize,
    pub gpus: GpuInfo,
}

/// Deploy a model
pub async fn deploy(
    client: &ApiClient,
    model: String,
    replicas: u32,
    gpu: u32,
    backend: String,
    name: Option<String>,
) -> Result<()> {
    let name = name.unwrap_or_else(|| {
        // Generate name from model
        model
            .split('/')
            .last()
            .unwrap_or(&model)
            .split(':')
            .next()
            .unwrap_or(&model)
            .to_string()
    });

    #[derive(Serialize)]
    struct CreateRequest {
        name: String,
        model: String,
        replicas: u32,
        gpu_count: u32,
        backend: String,
    }

    let req = CreateRequest {
        name: name.clone(),
        model: model.clone(),
        replicas,
        gpu_count: gpu,
        backend,
    };

    let response = client
        .client
        .post(client.url("/api/v1/deployments"))
        .json(&req)
        .send()
        .await?;

    if response.status().is_success() {
        let deployment: DeploymentResponse = response.json().await?;
        println!("Deployment '{}' created successfully", deployment.name);
        println!("  ID: {}", deployment.id);
        println!("  Model: {}", deployment.model);
        println!("  Replicas: {}/{}", deployment.ready_replicas, deployment.replicas);
        println!("  Phase: {}", deployment.phase);
    } else {
        let error = response.text().await?;
        eprintln!("Failed to create deployment: {}", error);
    }

    Ok(())
}

/// Scale a deployment
pub async fn scale(client: &ApiClient, deployment: String, replicas: u32) -> Result<()> {
    // Try to parse as UUID first, otherwise search by name
    let id = parse_deployment_id(client, &deployment).await?;

    #[derive(Serialize)]
    struct ScaleRequest {
        replicas: u32,
    }

    let response = client
        .client
        .post(client.url(&format!("/api/v1/deployments/{}/scale", id)))
        .json(&ScaleRequest { replicas })
        .send()
        .await?;

    if response.status().is_success() {
        let deployment: DeploymentResponse = response.json().await?;
        println!("Deployment '{}' scaled to {} replicas", deployment.name, replicas);
    } else {
        let error = response.text().await?;
        eprintln!("Failed to scale deployment: {}", error);
    }

    Ok(())
}

/// Delete a deployment
pub async fn delete(client: &ApiClient, deployment: String) -> Result<()> {
    let id = parse_deployment_id(client, &deployment).await?;

    let response = client
        .client
        .delete(client.url(&format!("/api/v1/deployments/{}", id)))
        .send()
        .await?;

    if response.status().is_success() {
        println!("Deployment '{}' deleted", deployment);
    } else {
        let error = response.text().await?;
        eprintln!("Failed to delete deployment: {}", error);
    }

    Ok(())
}

/// Get deployment status
pub async fn status(client: &ApiClient, deployment: Option<String>) -> Result<()> {
    match deployment {
        Some(name) => {
            let id = parse_deployment_id(client, &name).await?;
            let response = client
                .client
                .get(client.url(&format!("/api/v1/deployments/{}", id)))
                .send()
                .await?;

            if response.status().is_success() {
                let dep: DeploymentResponse = response.json().await?;
                print_deployment_details(&dep);

                // Get workers
                let workers_response = client
                    .client
                    .get(client.url(&format!("/api/v1/deployments/{}/workers", id)))
                    .send()
                    .await?;

                if workers_response.status().is_success() {
                    let workers: Vec<WorkerResponse> = workers_response.json().await?;
                    if !workers.is_empty() {
                        println!("\nWorkers:");
                        for w in workers {
                            println!(
                                "  [{}] {} - {} (GPUs: {:?})",
                                w.index, w.status, w.endpoint, w.gpu_ids
                            );
                        }
                    }
                }
            } else {
                let error = response.text().await?;
                eprintln!("Deployment not found: {}", error);
            }
        }
        None => {
            // Show all deployments
            ps(client).await?;
        }
    }

    Ok(())
}

/// List all deployments
pub async fn ps(client: &ApiClient) -> Result<()> {
    let response = client
        .client
        .get(client.url("/api/v1/deployments"))
        .send()
        .await?;

    if response.status().is_success() {
        let deployments: Vec<DeploymentResponse> = response.json().await?;

        if deployments.is_empty() {
            println!("No deployments found");
        } else {
            println!(
                "{:<36} {:<20} {:<25} {:<10} {:<10}",
                "ID", "NAME", "MODEL", "REPLICAS", "PHASE"
            );
            println!("{}", "-".repeat(100));
            for dep in deployments {
                println!(
                    "{:<36} {:<20} {:<25} {}/{:<7} {:<10}",
                    dep.id, dep.name, dep.model, dep.ready_replicas, dep.replicas, dep.phase
                );
            }
        }
    } else {
        let error = response.text().await?;
        eprintln!("Failed to list deployments: {}", error);
    }

    Ok(())
}

/// Show worker endpoints
pub async fn endpoints(client: &ApiClient) -> Result<()> {
    let response = client
        .client
        .get(client.url("/api/v1/endpoints"))
        .send()
        .await?;

    if response.status().is_success() {
        let endpoints: Vec<EndpointResponse> = response.json().await?;

        if endpoints.is_empty() {
            println!("No endpoints available");
        } else {
            println!("Available endpoints for direct access:");
            for ep in endpoints {
                let scheme = if ep.tls { "https" } else { "http" };
                println!("  {}://{}:{}", scheme, ep.host, ep.port);
            }
        }
    } else {
        let error = response.text().await?;
        eprintln!("Failed to get endpoints: {}", error);
    }

    Ok(())
}

/// Show GPU information
pub async fn gpus(client: &ApiClient) -> Result<()> {
    let response = client.client.get(client.url("/api/v1/gpus")).send().await?;

    if response.status().is_success() {
        let gpu_info: GpuInfo = response.json().await?;

        println!(
            "GPUs: {} total, {} available",
            gpu_info.total_count, gpu_info.available_count
        );

        if !gpu_info.devices.is_empty() {
            println!();
            for device in gpu_info.devices {
                let mem_total = device.memory_total / (1024 * 1024 * 1024);
                let mem_free = device.memory_free / (1024 * 1024 * 1024);
                println!(
                    "[{}] {} ({}) - {}/{}GB - {}",
                    device.index,
                    device.name,
                    device.vendor,
                    mem_free,
                    mem_total,
                    if device.available {
                        "Available"
                    } else {
                        "In Use"
                    }
                );
            }
        }
    } else {
        let error = response.text().await?;
        eprintln!("Failed to get GPU info: {}", error);
    }

    Ok(())
}

/// Show system status
pub async fn top(client: &ApiClient) -> Result<()> {
    let response = client
        .client
        .get(client.url("/api/v1/status"))
        .send()
        .await?;

    if response.status().is_success() {
        let status: StatusResponse = response.json().await?;

        println!("dmrlet v{}", status.version);
        println!();
        println!("Deployments: {}", status.deployments);
        println!("Workers: {}", status.workers);
        println!(
            "GPUs: {} total, {} available",
            status.gpus.total_count, status.gpus.available_count
        );
    } else {
        let error = response.text().await?;
        eprintln!("Failed to get status: {}", error);
    }

    Ok(())
}

/// Helper to parse deployment ID (UUID or name)
async fn parse_deployment_id(client: &ApiClient, deployment: &str) -> Result<Uuid> {
    // Try parsing as UUID first
    if let Ok(id) = Uuid::parse_str(deployment) {
        return Ok(id);
    }

    // Otherwise, search by name
    let response = client
        .client
        .get(client.url("/api/v1/deployments"))
        .send()
        .await?;

    if response.status().is_success() {
        let deployments: Vec<DeploymentResponse> = response.json().await?;
        for dep in deployments {
            if dep.name == deployment {
                return Ok(dep.id);
            }
        }
    }

    anyhow::bail!("Deployment '{}' not found", deployment)
}

/// Helper to print deployment details
fn print_deployment_details(dep: &DeploymentResponse) {
    println!("Deployment: {}", dep.name);
    println!("  ID: {}", dep.id);
    println!("  Model: {}", dep.model);
    println!("  Replicas: {}/{}", dep.ready_replicas, dep.replicas);
    println!("  Phase: {}", dep.phase);
}
