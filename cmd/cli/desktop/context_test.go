package desktop

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	clientpkg "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerClientForContext_CurrentContext(t *testing.T) {
	// Create a mock Docker CLI with default options
	opts := flags.NewClientOptions()
	cli, err := command.NewDockerCli()
	require.NoError(t, err)

	// Initialize the CLI with default options
	err = cli.Initialize(opts)
	require.NoError(t, err)

	// Get the current context name
	currentContext := cli.CurrentContext()

	// Call DockerClientForContext with the current context
	client, err := DockerClientForContext(cli, currentContext)

	// We expect the function to succeed
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify that we got a concrete client
	assert.IsType(t, &clientpkg.Client{}, client)

	// The client should be the same as the one from the CLI
	cliClient := cli.Client()
	require.NotNil(t, cliClient)

	// Since we're returning the same client, they should be the same instance
	concreteCliClient, ok := cliClient.(*clientpkg.Client)
	require.True(t, ok, "CLI client should be a concrete *clientpkg.Client")

	// Verify that both clients point to the same instance
	assert.Same(t, concreteCliClient, client, "Should reuse the CLI's client for the current context")
}

func TestDockerClientForContext_CurrentContextReusesClient(t *testing.T) {
	// This test verifies that when we request a client for the current context,
	// we get the same client instance back, which ensures proper handling of
	// special transports like SSH.

	opts := flags.NewClientOptions()
	cli, err := command.NewDockerCli()
	require.NoError(t, err)

	err = cli.Initialize(opts)
	require.NoError(t, err)

	currentContext := cli.CurrentContext()

	// Call DockerClientForContext twice with the current context
	client1, err1 := DockerClientForContext(cli, currentContext)
	require.NoError(t, err1)
	require.NotNil(t, client1)

	client2, err2 := DockerClientForContext(cli, currentContext)
	require.NoError(t, err2)
	require.NotNil(t, client2)

	// Both calls should return the same client instance
	assert.Same(t, client1, client2, "Should return the same client instance for the current context")
}
