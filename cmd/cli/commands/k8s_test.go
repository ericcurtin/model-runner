package commands

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func TestK8sCommand(t *testing.T) {
	cli, err := command.NewDockerCli()
	if err != nil {
		t.Fatalf("failed to create docker cli: %v", err)
	}

	rootCmd := NewRootCmd(cli)

	// Check that k8s command exists
	k8sCmd, _, err := rootCmd.Find([]string{"k8s"})
	if err != nil {
		t.Fatalf("k8s command not found: %v", err)
	}

	if k8sCmd.Use != "k8s" {
		t.Errorf("expected k8s command, got %s", k8sCmd.Use)
	}

	// Check that install subcommand exists
	installCmd, _, err := k8sCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("install subcommand not found: %v", err)
	}

	if installCmd.Use != "install" {
		t.Errorf("expected install subcommand, got %s", installCmd.Use)
	}

	// Check that uninstall subcommand exists
	uninstallCmd, _, err := k8sCmd.Find([]string{"uninstall"})
	if err != nil {
		t.Fatalf("uninstall subcommand not found: %v", err)
	}

	if uninstallCmd.Use != "uninstall" {
		t.Errorf("expected uninstall subcommand, got %s", uninstallCmd.Use)
	}
}

func TestK8sInstallFlags(t *testing.T) {
	cli, err := command.NewDockerCli()
	if err != nil {
		t.Fatalf("failed to create docker cli: %v", err)
	}

	rootCmd := NewRootCmd(cli)
	k8sCmd, _, err := rootCmd.Find([]string{"k8s"})
	if err != nil {
		t.Fatalf("k8s command not found: %v", err)
	}

	installCmd, _, err := k8sCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("install subcommand not found: %v", err)
	}

	// Check required flags
	requiredFlags := []string{
		"namespace",
		"storage-class",
		"storage-size",
		"values",
		"kubeconfig",
		"release",
		"debug",
	}

	for _, flagName := range requiredFlags {
		flag := installCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("flag %s not found", flagName)
		}
	}

	// Check default values
	namespaceFlag := installCmd.Flags().Lookup("namespace")
	if namespaceFlag.DefValue != "docker-model-runner" {
		t.Errorf("expected default namespace 'docker-model-runner', got %s", namespaceFlag.DefValue)
	}

	storageSizeFlag := installCmd.Flags().Lookup("storage-size")
	if storageSizeFlag.DefValue != "100Gi" {
		t.Errorf("expected default storage size '100Gi', got %s", storageSizeFlag.DefValue)
	}

	releaseFlag := installCmd.Flags().Lookup("release")
	if releaseFlag.DefValue != "docker-model-runner" {
		t.Errorf("expected default release 'docker-model-runner', got %s", releaseFlag.DefValue)
	}
}

func TestK8sUninstallFlags(t *testing.T) {
	cli, err := command.NewDockerCli()
	if err != nil {
		t.Fatalf("failed to create docker cli: %v", err)
	}

	rootCmd := NewRootCmd(cli)
	k8sCmd, _, err := rootCmd.Find([]string{"k8s"})
	if err != nil {
		t.Fatalf("k8s command not found: %v", err)
	}

	uninstallCmd, _, err := k8sCmd.Find([]string{"uninstall"})
	if err != nil {
		t.Fatalf("uninstall subcommand not found: %v", err)
	}

	// Check required flags
	requiredFlags := []string{
		"namespace",
		"kubeconfig",
		"release",
		"debug",
	}

	for _, flagName := range requiredFlags {
		flag := uninstallCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("flag %s not found", flagName)
		}
	}
}

func TestCheckCommand(t *testing.T) {
	// Test with a command that should exist
	err := checkCommand("ls")
	if err != nil {
		t.Errorf("expected ls command to exist: %v", err)
	}

	// Test with a command that should not exist
	err = checkCommand("this-command-definitely-does-not-exist-12345")
	if err == nil {
		t.Error("expected error for non-existent command")
	}
}

func TestFindChartDirectory(t *testing.T) {
	// This test will likely fail in most environments, but it verifies
	// the function doesn't panic
	_, err := findChartDirectory()
	// We expect an error since we're not in the repo root in test environment
	if err == nil {
		t.Log("chart directory found (test environment has access to charts)")
	} else {
		t.Logf("chart directory not found (expected in test environment): %v", err)
	}
}

func TestK8sCommandHelpers(t *testing.T) {
	// Test that helper functions don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("helper function panicked: %v", r)
		}
	}()

	// Test logging helpers
	logInfo("test info message")
	logSuccess("test success message")
	logError("test error message")
}

// Ensure k8s command structure matches expected cobra.Command interface
func TestK8sCommandStructure(t *testing.T) {
	cmd := newK8sCmd()

	if cmd == nil {
		t.Fatal("k8s command is nil")
	}

	if cmd.Use != "k8s" {
		t.Errorf("expected Use to be 'k8s', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("k8s command should have a short description")
	}

	if cmd.Long == "" {
		t.Error("k8s command should have a long description")
	}

	// Verify it has subcommands
	if !cmd.HasSubCommands() {
		t.Error("k8s command should have subcommands")
	}

	// Count subcommands
	subCommands := cmd.Commands()
	if len(subCommands) != 2 {
		t.Errorf("expected 2 subcommands (install, uninstall), got %d", len(subCommands))
	}
}

func TestNewK8sInstallCmd(t *testing.T) {
	cmd := newK8sInstallCmd()

	if cmd == nil {
		t.Fatal("install command is nil")
	}

	if cmd.Use != "install" {
		t.Errorf("expected Use to be 'install', got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("install command should have RunE function")
	}

	// Verify command type
	var _ *cobra.Command = cmd
}

func TestNewK8sUninstallCmd(t *testing.T) {
	cmd := newK8sUninstallCmd()

	if cmd == nil {
		t.Fatal("uninstall command is nil")
	}

	if cmd.Use != "uninstall" {
		t.Errorf("expected Use to be 'uninstall', got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("uninstall command should have RunE function")
	}

	// Verify command type
	var _ *cobra.Command = cmd
}
