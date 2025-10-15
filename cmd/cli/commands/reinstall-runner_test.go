package commands

import (
	"testing"
)

func TestReinstallRunnerHostFlag(t *testing.T) {
	// Create the reinstall-runner command
	cmd := newReinstallRunner()

	// Verify the --host flag exists
	hostFlag := cmd.Flags().Lookup("host")
	if hostFlag == nil {
		t.Fatal("--host flag not found")
	}

	// Verify the default value
	if hostFlag.DefValue != "127.0.0.1" {
		t.Errorf("Expected default host value to be '127.0.0.1', got '%s'", hostFlag.DefValue)
	}

	// Verify the flag type
	if hostFlag.Value.Type() != "string" {
		t.Errorf("Expected host flag type to be 'string', got '%s'", hostFlag.Value.Type())
	}

	// Test setting the flag value
	testCases := []struct {
		name  string
		value string
	}{
		{"localhost", "127.0.0.1"},
		{"all interfaces", "0.0.0.0"},
		{"specific IP", "192.168.1.100"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the command for each test
			cmd := newReinstallRunner()
			err := cmd.Flags().Set("host", tc.value)
			if err != nil {
				t.Errorf("Failed to set host flag to '%s': %v", tc.value, err)
			}

			// Verify the value was set
			hostValue, err := cmd.Flags().GetString("host")
			if err != nil {
				t.Errorf("Failed to get host flag value: %v", err)
			}
			if hostValue != tc.value {
				t.Errorf("Expected host value to be '%s', got '%s'", tc.value, hostValue)
			}
		})
	}
}

func TestReinstallRunnerCommandFlags(t *testing.T) {
	cmd := newReinstallRunner()

	// Verify all expected flags exist
	expectedFlags := []string{"port", "host", "gpu", "do-not-track"}
	for _, flagName := range expectedFlags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '--%s' not found", flagName)
		}
	}
}

func TestReinstallRunnerCommandType(t *testing.T) {
	cmd := newReinstallRunner()

	// Verify command properties
	if cmd.Use != "reinstall-runner" {
		t.Errorf("Expected command Use to be 'reinstall-runner', got '%s'", cmd.Use)
	}

	if cmd.Short != "Reinstall Docker Model Runner (Docker Engine only)" {
		t.Errorf("Unexpected command Short description: %s", cmd.Short)
	}

	// Verify RunE is set
	if cmd.RunE == nil {
		t.Error("Expected RunE to be set")
	}
}

func TestReinstallRunnerValidArgsFunction(t *testing.T) {
	cmd := newReinstallRunner()

	// The reinstall-runner command should not accept any arguments
	// So ValidArgsFunction should be set to handle no arguments
	if cmd.ValidArgsFunction == nil {
		t.Error("Expected ValidArgsFunction to be set")
	}
}
