//! dmrlet daemon
//!
//! Main daemon process that orchestrates model deployments.

use clap::Parser;
use dmrlet_api::create_router;
use dmrlet_scheduler::Scheduler;
use std::net::SocketAddr;
use std::sync::Arc;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

/// dmrlet daemon - Kubernetes-like orchestrator for Docker Model Runner
#[derive(Parser, Debug)]
#[command(name = "dmrletd")]
#[command(version, about, long_about = None)]
struct Args {
    /// Address to bind the API server
    #[arg(long, default_value = "0.0.0.0")]
    address: String,

    /// Port for the REST API server
    #[arg(long, default_value_t = 9090)]
    port: u16,

    /// Base port for worker allocation
    #[arg(long, default_value_t = 30000)]
    worker_base_port: u16,

    /// Maximum number of workers
    #[arg(long, default_value_t = 100)]
    max_workers: u32,

    /// Log level
    #[arg(long, default_value = "info")]
    log_level: String,
}

#[tokio::main]
async fn main() {
    let args = Args::parse();

    // Initialize logging
    let log_level = match args.log_level.to_lowercase().as_str() {
        "trace" => Level::TRACE,
        "debug" => Level::DEBUG,
        "info" => Level::INFO,
        "warn" => Level::WARN,
        "error" => Level::ERROR,
        _ => Level::INFO,
    };

    let subscriber = FmtSubscriber::builder()
        .with_max_level(log_level)
        .with_target(false)
        .finish();
    tracing::subscriber::set_global_default(subscriber).expect("Failed to set subscriber");

    info!("Starting dmrlet daemon v{}", env!("CARGO_PKG_VERSION"));

    // Create scheduler
    let scheduler = Arc::new(Scheduler::new(args.worker_base_port, args.max_workers));

    // Create API router
    let router = create_router(scheduler);

    // Bind and serve
    let addr: SocketAddr = format!("{}:{}", args.address, args.port)
        .parse()
        .expect("Invalid address");

    info!("API server listening on {}", addr);
    info!("Worker ports starting at {}", args.worker_base_port);

    let listener = tokio::net::TcpListener::bind(addr).await.expect("Failed to bind");
    axum::serve(listener, router).await.expect("Server error");
}
