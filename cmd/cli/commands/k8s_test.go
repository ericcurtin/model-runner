package commands

import (
	"testing"
)

func TestK8sCommandStructure(t *testing.T) {
	cmd := newK8sCmd()

	// Verify command properties
	if cmd.Use != "k8s" {
		t.Errorf("Expected command Use to be 'k8s', got '%s'", cmd.Use)
	}

	if cmd.Short != "Kubernetes deployment commands for distributed AI inference" {
		t.Errorf("Unexpected command Short description: %s", cmd.Short)
	}

	// Verify subcommands exist
	expectedSubcommands := []string{"install", "deploy", "test", "uninstall"}
	for _, subcommandName := range expectedSubcommands {
		found := false
		for _, subcommand := range cmd.Commands() {
			if subcommand.Name() == subcommandName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found", subcommandName)
		}
	}
}

func TestK8sInstallCommandFlags(t *testing.T) {
	cmd := newK8sInstallCmd()

	// Verify command properties
	if cmd.Use != "install" {
		t.Errorf("Expected command Use to be 'install', got '%s'", cmd.Use)
	}

	// Verify all expected flags exist
	expectedFlags := []string{"namespace", "provider"}
	for _, flagName := range expectedFlags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '--%s' not found", flagName)
		}
	}

	// Verify default values
	namespaceFlag := cmd.Flags().Lookup("namespace")
	if namespaceFlag.DefValue != "default" {
		t.Errorf("Expected default namespace to be 'default', got '%s'", namespaceFlag.DefValue)
	}

	providerFlag := cmd.Flags().Lookup("provider")
	if providerFlag.DefValue != "aibrix" {
		t.Errorf("Expected default provider to be 'aibrix', got '%s'", providerFlag.DefValue)
	}

	// Verify RunE is set
	if cmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}
}

func TestK8sDeployCommandFlags(t *testing.T) {
	cmd := newK8sDeployCmd()

	// Verify command properties
	if cmd.Use != "deploy" {
		t.Errorf("Expected command Use to be 'deploy', got '%s'", cmd.Use)
	}

	// Verify all expected flags exist
	expectedFlags := []string{"model", "namespace", "disaggregation", "replicas", "hf-token", "provider"}
	for _, flagName := range expectedFlags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '--%s' not found", flagName)
		}
	}

	// Verify default values
	namespaceFlag := cmd.Flags().Lookup("namespace")
	if namespaceFlag.DefValue != "default" {
		t.Errorf("Expected default namespace to be 'default', got '%s'", namespaceFlag.DefValue)
	}

	providerFlag := cmd.Flags().Lookup("provider")
	if providerFlag.DefValue != "aibrix" {
		t.Errorf("Expected default provider to be 'aibrix', got '%s'", providerFlag.DefValue)
	}

	disaggregationFlag := cmd.Flags().Lookup("disaggregation")
	if disaggregationFlag.DefValue != "false" {
		t.Errorf("Expected default disaggregation to be 'false', got '%s'", disaggregationFlag.DefValue)
	}

	replicasFlag := cmd.Flags().Lookup("replicas")
	if replicasFlag.DefValue != "1" {
		t.Errorf("Expected default replicas to be '1', got '%s'", replicasFlag.DefValue)
	}

	// Verify RunE is set
	if cmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}
}

func TestK8sTestCommandFlags(t *testing.T) {
	cmd := newK8sTestCmd()

	// Verify command properties
	if cmd.Use != "test" {
		t.Errorf("Expected command Use to be 'test', got '%s'", cmd.Use)
	}

	// Verify all expected flags exist
	expectedFlags := []string{"namespace", "endpoint", "provider"}
	for _, flagName := range expectedFlags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '--%s' not found", flagName)
		}
	}

	// Verify default values
	namespaceFlag := cmd.Flags().Lookup("namespace")
	if namespaceFlag.DefValue != "default" {
		t.Errorf("Expected default namespace to be 'default', got '%s'", namespaceFlag.DefValue)
	}

	providerFlag := cmd.Flags().Lookup("provider")
	if providerFlag.DefValue != "aibrix" {
		t.Errorf("Expected default provider to be 'aibrix', got '%s'", providerFlag.DefValue)
	}

	// Verify RunE is set
	if cmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}
}

func TestK8sUninstallCommandFlags(t *testing.T) {
	cmd := newK8sUninstallCmd()

	// Verify command properties
	if cmd.Use != "uninstall" {
		t.Errorf("Expected command Use to be 'uninstall', got '%s'", cmd.Use)
	}

	// Verify all expected flags exist
	expectedFlags := []string{"namespace", "provider"}
	for _, flagName := range expectedFlags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '--%s' not found", flagName)
		}
	}

	// Verify default values
	namespaceFlag := cmd.Flags().Lookup("namespace")
	if namespaceFlag.DefValue != "default" {
		t.Errorf("Expected default namespace to be 'default', got '%s'", namespaceFlag.DefValue)
	}

	providerFlag := cmd.Flags().Lookup("provider")
	if providerFlag.DefValue != "aibrix" {
		t.Errorf("Expected default provider to be 'aibrix', got '%s'", providerFlag.DefValue)
	}

	// Verify RunE is set
	if cmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}
}

func TestK8sProviderValidation(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		model    string
	}{
		{"aibrix provider", "aibrix", "deepseek-ai/DeepSeek-R1-Distill-Llama-8B"},
		{"llm-d provider", "llm-d", "Qwen/Qwen3-0.6B"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newK8sDeployCmd()
			err := cmd.Flags().Set("provider", tc.provider)
			if err != nil {
				t.Errorf("Failed to set provider flag to '%s': %v", tc.provider, err)
			}

			err = cmd.Flags().Set("model", tc.model)
			if err != nil {
				t.Errorf("Failed to set model flag to '%s': %v", tc.model, err)
			}

			// Verify the value was set
			providerValue, err := cmd.Flags().GetString("provider")
			if err != nil {
				t.Errorf("Failed to get provider flag value: %v", err)
			}
			if providerValue != tc.provider {
				t.Errorf("Expected provider value to be '%s', got '%s'", tc.provider, providerValue)
			}

			modelValue, err := cmd.Flags().GetString("model")
			if err != nil {
				t.Errorf("Failed to get model flag value: %v", err)
			}
			if modelValue != tc.model {
				t.Errorf("Expected model value to be '%s', got '%s'", tc.model, modelValue)
			}
		})
	}
}
