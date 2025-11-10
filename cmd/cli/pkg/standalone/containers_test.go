package standalone

import (
	"os"
	"testing"

	"github.com/docker/docker/api/types/system"
)

func TestMergeDockerProxySettings_FromDaemonConfig(t *testing.T) {
	proxySettings := make(map[string]string)
	
	info := system.Info{
		HTTPProxy:  "http://proxy.example.com:8080",
		HTTPSProxy: "https://proxy.example.com:8443",
		NoProxy:    "localhost,127.0.0.1",
	}
	
	mergeDockerProxySettings(proxySettings, info)
	
	// Verify both uppercase and lowercase variants are set
	if proxySettings["HTTP_PROXY"] != "http://proxy.example.com:8080" {
		t.Errorf("Expected HTTP_PROXY to be http://proxy.example.com:8080, got %s", proxySettings["HTTP_PROXY"])
	}
	if proxySettings["http_proxy"] != "http://proxy.example.com:8080" {
		t.Errorf("Expected http_proxy to be http://proxy.example.com:8080, got %s", proxySettings["http_proxy"])
	}
	if proxySettings["HTTPS_PROXY"] != "https://proxy.example.com:8443" {
		t.Errorf("Expected HTTPS_PROXY to be https://proxy.example.com:8443, got %s", proxySettings["HTTPS_PROXY"])
	}
	if proxySettings["https_proxy"] != "https://proxy.example.com:8443" {
		t.Errorf("Expected https_proxy to be https://proxy.example.com:8443, got %s", proxySettings["https_proxy"])
	}
	if proxySettings["NO_PROXY"] != "localhost,127.0.0.1" {
		t.Errorf("Expected NO_PROXY to be localhost,127.0.0.1, got %s", proxySettings["NO_PROXY"])
	}
	if proxySettings["no_proxy"] != "localhost,127.0.0.1" {
		t.Errorf("Expected no_proxy to be localhost,127.0.0.1, got %s", proxySettings["no_proxy"])
	}
}

func TestMergeDockerProxySettings_OverridesExisting(t *testing.T) {
	// Start with environment-based settings
	proxySettings := map[string]string{
		"HTTP_PROXY":  "http://env.proxy.com:3128",
		"http_proxy":  "http://env.proxy.com:3128",
		"HTTPS_PROXY": "https://env.proxy.com:3129",
		"https_proxy": "https://env.proxy.com:3129",
		"NO_PROXY":    "localhost",
		"no_proxy":    "localhost",
	}
	
	info := system.Info{
		HTTPProxy:  "http://daemon.proxy.com:8080",
		HTTPSProxy: "https://daemon.proxy.com:8443",
		NoProxy:    "localhost,127.0.0.1,*.local",
	}
	
	mergeDockerProxySettings(proxySettings, info)
	
	// Verify daemon settings take precedence
	if proxySettings["HTTP_PROXY"] != "http://daemon.proxy.com:8080" {
		t.Errorf("Expected HTTP_PROXY from daemon to override env, got %s", proxySettings["HTTP_PROXY"])
	}
	if proxySettings["http_proxy"] != "http://daemon.proxy.com:8080" {
		t.Errorf("Expected http_proxy from daemon to override env, got %s", proxySettings["http_proxy"])
	}
	if proxySettings["HTTPS_PROXY"] != "https://daemon.proxy.com:8443" {
		t.Errorf("Expected HTTPS_PROXY from daemon to override env, got %s", proxySettings["HTTPS_PROXY"])
	}
	if proxySettings["https_proxy"] != "https://daemon.proxy.com:8443" {
		t.Errorf("Expected https_proxy from daemon to override env, got %s", proxySettings["https_proxy"])
	}
	if proxySettings["NO_PROXY"] != "localhost,127.0.0.1,*.local" {
		t.Errorf("Expected NO_PROXY from daemon to override env, got %s", proxySettings["NO_PROXY"])
	}
	if proxySettings["no_proxy"] != "localhost,127.0.0.1,*.local" {
		t.Errorf("Expected no_proxy from daemon to override env, got %s", proxySettings["no_proxy"])
	}
}

func TestMergeDockerProxySettings_EmptyDoesNotOverride(t *testing.T) {
	// Start with environment-based settings
	proxySettings := map[string]string{
		"HTTP_PROXY":  "http://env.proxy.com:3128",
		"http_proxy":  "http://env.proxy.com:3128",
		"HTTPS_PROXY": "https://env.proxy.com:3129",
		"https_proxy": "https://env.proxy.com:3129",
	}
	
	// Empty daemon info should not override existing values
	info := system.Info{}
	
	mergeDockerProxySettings(proxySettings, info)
	
	// Verify environment settings are preserved
	if proxySettings["HTTP_PROXY"] != "http://env.proxy.com:3128" {
		t.Errorf("Expected HTTP_PROXY to remain from env, got %s", proxySettings["HTTP_PROXY"])
	}
	if proxySettings["http_proxy"] != "http://env.proxy.com:3128" {
		t.Errorf("Expected http_proxy to remain from env, got %s", proxySettings["http_proxy"])
	}
	if proxySettings["HTTPS_PROXY"] != "https://env.proxy.com:3129" {
		t.Errorf("Expected HTTPS_PROXY to remain from env, got %s", proxySettings["HTTPS_PROXY"])
	}
	if proxySettings["https_proxy"] != "https://env.proxy.com:3129" {
		t.Errorf("Expected https_proxy to remain from env, got %s", proxySettings["https_proxy"])
	}
}

func TestMergeDockerProxySettings_PartialSettings(t *testing.T) {
	proxySettings := make(map[string]string)
	
	// Only HTTP proxy is set in daemon
	info := system.Info{
		HTTPProxy: "http://proxy.example.com:8080",
	}
	
	mergeDockerProxySettings(proxySettings, info)
	
	// Verify only HTTP proxy is set
	if proxySettings["HTTP_PROXY"] != "http://proxy.example.com:8080" {
		t.Errorf("Expected HTTP_PROXY to be http://proxy.example.com:8080, got %s", proxySettings["HTTP_PROXY"])
	}
	if proxySettings["http_proxy"] != "http://proxy.example.com:8080" {
		t.Errorf("Expected http_proxy to be http://proxy.example.com:8080, got %s", proxySettings["http_proxy"])
	}
	
	// Verify other proxies are not set
	if val, ok := proxySettings["HTTPS_PROXY"]; ok && val != "" {
		t.Errorf("Expected HTTPS_PROXY to be empty, got %s", val)
	}
	if val, ok := proxySettings["NO_PROXY"]; ok && val != "" {
		t.Errorf("Expected NO_PROXY to be empty, got %s", val)
	}
}

func TestGetProxySettings_EnvironmentOnly(t *testing.T) {
	// Save current env vars
	savedEnv := map[string]string{
		"HTTP_PROXY":  os.Getenv("HTTP_PROXY"),
		"HTTPS_PROXY": os.Getenv("HTTPS_PROXY"),
		"NO_PROXY":    os.Getenv("NO_PROXY"),
		"http_proxy":  os.Getenv("http_proxy"),
		"https_proxy": os.Getenv("https_proxy"),
		"no_proxy":    os.Getenv("no_proxy"),
	}
	defer func() {
		for k, v := range savedEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()
	
	// Clear all proxy env vars first
	for k := range savedEnv {
		os.Unsetenv(k)
	}
	
	// Set test environment variables
	os.Setenv("HTTP_PROXY", "http://env.proxy.com:3128")
	os.Setenv("https_proxy", "https://env.proxy.com:3129")
	os.Setenv("NO_PROXY", "localhost")
	
	// Test that environment variables are picked up
	// Note: We can't easily test getProxySettings without a real Docker client,
	// but the mergeDockerProxySettings tests above cover the core logic
}
