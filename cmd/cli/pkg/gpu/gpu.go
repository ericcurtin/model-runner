package gpu

import (
	"context"
	"os/exec"

	"github.com/docker/docker/client"
)

// GPUSupport encodes the GPU support available on a Docker engine.
type GPUSupport uint8

const (
	// GPUSupportNone indicates no detectable GPU support.
	GPUSupportNone GPUSupport = iota
	// GPUSupportCUDA indicates CUDA GPU support.
	GPUSupportCUDA
	// GPUSupportROCm indicates ROCm GPU support.
	GPUSupportROCm
	// GPUSupportMUSA indicates MUSA GPU support.
	GPUSupportMUSA
	// GPUSupportCANN indicates Ascend NPU support.
	GPUSupportCANN
)

// ProbeGPUSupport determines whether or not the Docker engine has GPU support.
func ProbeGPUSupport(ctx context.Context, dockerClient client.SystemAPIClient) (GPUSupport, error) {
	// Query Docker Engine for its effective configuration.
	// Docker Info is the source of truth for which runtimes are actually usable.
	info, err := dockerClient.Info(ctx)
	if err != nil {
		return GPUSupportNone, err
	}

	// 1. CUDA (NVIDIA)
	// NVIDIA remains the highest priority due to its wide adoption and
	// first-class support in Docker (>= 19.03).
	if _, ok := info.Runtimes["nvidia"]; ok {
		return GPUSupportCUDA, nil
	}

	// 2. ROCm (AMD)
	// Explicit ROCm runtime configured in the Docker Engine.
	if _, ok := info.Runtimes["rocm"]; ok {
		return GPUSupportROCm, nil
	}

	// 3. MUSA (MThreads)
	// Used primarily on specific accelerator platforms.
	if _, ok := info.Runtimes["mthreads"]; ok {
		return GPUSupportMUSA, nil
	}

	// 4. Ascend CANN (Huawei)
	// Ascend NPU runtime registered in Docker.
	if _, ok := info.Runtimes["cann"]; ok {
		return GPUSupportCANN, nil
	}

	// 5. Legacy fallback
	// Older Docker setups may not register the NVIDIA runtime explicitly,
	// but still have the legacy nvidia-container-runtime available on PATH.
	if _, err := exec.LookPath("nvidia-container-runtime"); err == nil {
		return GPUSupportCUDA, nil
	}

	// No known GPU runtime detected.
	return GPUSupportNone, nil
}

// HasNVIDIARuntime determines whether there is an nvidia runtime available
func HasNVIDIARuntime(ctx context.Context, dockerClient client.SystemAPIClient) (bool, error) {
	info, err := dockerClient.Info(ctx)
	if err != nil {
		return false, err
	}
	_, hasNvidia := info.Runtimes["nvidia"]
	return hasNvidia, nil
}

// HasROCmRuntime determines whether there is a ROCm runtime available
func HasROCmRuntime(ctx context.Context, dockerClient client.SystemAPIClient) (bool, error) {
	info, err := dockerClient.Info(ctx)
	if err != nil {
		return false, err
	}
	_, hasROCm := info.Runtimes["rocm"]
	return hasROCm, nil
}

// HasMTHREADSRuntime determines whether there is a mthreads runtime available
func HasMTHREADSRuntime(ctx context.Context, dockerClient client.SystemAPIClient) (bool, error) {
	info, err := dockerClient.Info(ctx)
	if err != nil {
		return false, err
	}
	_, hasMTHREADS := info.Runtimes["mthreads"]
	return hasMTHREADS, nil
}

// HasCANNRuntime determines whether there is a Ascend CANN runtime available
func HasCANNRuntime(ctx context.Context, dockerClient client.SystemAPIClient) (bool, error) {
	info, err := dockerClient.Info(ctx)
	if err != nil {
		return false, err
	}
	_, hasCANN := info.Runtimes["cann"]
	return hasCANN, nil
}
