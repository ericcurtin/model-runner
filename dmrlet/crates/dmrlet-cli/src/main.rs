//! dmrlet CLI
//!
//! Command-line interface for interacting with the dmrlet daemon.

mod commands;

use clap::{Parser, Subcommand};
use tracing::Level;
use tracing_subscriber::FmtSubscriber;

/// dmrlet - Kubernetes-like orchestrator for Docker Model Runner
#[derive(Parser, Debug)]
#[command(name = "dmrlet")]
#[command(version, about, long_about = None)]
struct Cli {
    /// Daemon API address
    #[arg(long, default_value = "http://localhost:9090", global = true)]
    api: String,

    /// Enable verbose output
    #[arg(short, long, global = true)]
    verbose: bool,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// Deploy a model
    Deploy {
        /// Model to deploy (e.g., ai/llama3:8b)
        model: String,

        /// Number of replicas
        #[arg(long, default_value_t = 1)]
        replicas: u32,

        /// Number of GPUs per worker
        #[arg(long, default_value_t = 0)]
        gpu: u32,

        /// Backend type (llama.cpp, vllm, mlx)
        #[arg(long, default_value = "llama.cpp")]
        backend: String,

        /// Deployment name (defaults to model name)
        #[arg(long)]
        name: Option<String>,
    },

    /// Scale a deployment
    Scale {
        /// Deployment name or ID
        deployment: String,

        /// Number of replicas
        replicas: u32,
    },

    /// Delete a deployment
    Delete {
        /// Deployment name or ID
        deployment: String,
    },

    /// Get deployment status
    Status {
        /// Deployment name or ID (optional, shows all if not provided)
        deployment: Option<String>,
    },

    /// List all deployments
    Ps,

    /// Show worker endpoints
    Endpoints,

    /// Show GPU information
    Gpus,

    /// Show system status
    Top,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    // Initialize logging
    let log_level = if cli.verbose {
        Level::DEBUG
    } else {
        Level::WARN
    };

    let subscriber = FmtSubscriber::builder()
        .with_max_level(log_level)
        .with_target(false)
        .finish();
    let _ = tracing::subscriber::set_global_default(subscriber);

    let client = commands::ApiClient::new(&cli.api);

    match cli.command {
        Commands::Deploy {
            model,
            replicas,
            gpu,
            backend,
            name,
        } => {
            commands::deploy(&client, model, replicas, gpu, backend, name).await?;
        }
        Commands::Scale {
            deployment,
            replicas,
        } => {
            commands::scale(&client, deployment, replicas).await?;
        }
        Commands::Delete { deployment } => {
            commands::delete(&client, deployment).await?;
        }
        Commands::Status { deployment } => {
            commands::status(&client, deployment).await?;
        }
        Commands::Ps => {
            commands::ps(&client).await?;
        }
        Commands::Endpoints => {
            commands::endpoints(&client).await?;
        }
        Commands::Gpus => {
            commands::gpus(&client).await?;
        }
        Commands::Top => {
            commands::top(&client).await?;
        }
    }

    Ok(())
}
