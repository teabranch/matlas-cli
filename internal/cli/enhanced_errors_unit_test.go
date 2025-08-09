package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureCommandErrorHandling(t *testing.T) {
	// Create a root command with subcommands
	rootCmd := &cobra.Command{
		Use:   "root",
		Short: "Root command",
	}

	subCmd1 := &cobra.Command{
		Use:   "sub1",
		Short: "Subcommand 1",
	}

	subCmd2 := &cobra.Command{
		Use:   "sub2",
		Short: "Subcommand 2",
	}

	nestedCmd := &cobra.Command{
		Use:   "nested",
		Short: "Nested subcommand",
	}

	// Build command hierarchy
	subCmd1.AddCommand(nestedCmd)
	rootCmd.AddCommand(subCmd1, subCmd2)

	// Initially, SilenceUsage should be false
	assert.False(t, rootCmd.SilenceUsage)
	assert.False(t, subCmd1.SilenceUsage)
	assert.False(t, subCmd2.SilenceUsage)
	assert.False(t, nestedCmd.SilenceUsage)

	// Configure error handling
	ConfigureCommandErrorHandling(rootCmd)

	// All commands should now have SilenceUsage = true
	assert.True(t, rootCmd.SilenceUsage)
	assert.True(t, subCmd1.SilenceUsage)
	assert.True(t, subCmd2.SilenceUsage)
	assert.True(t, nestedCmd.SilenceUsage)
}

func TestConfigureCommandErrorHandling_EmptyCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "empty",
		Short: "Empty command with no subcommands",
	}

	// Initially SilenceUsage should be false
	assert.False(t, cmd.SilenceUsage)

	// Configure error handling
	ConfigureCommandErrorHandling(cmd)

	// Should be true after configuration
	assert.True(t, cmd.SilenceUsage)
}

func TestWrapCommandWithCleanErrors_NilRunE(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		RunE:  nil, // No RunE function
	}

	// Should not panic with nil RunE
	WrapCommandWithCleanErrors(cmd, nil)

	// RunE should still be nil
	assert.Nil(t, cmd.RunE)
}

func TestWrapCommandWithCleanErrors_SuccessfulExecution(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	executed := false
	originalRunE := func(cmd *cobra.Command, args []string) error {
		executed = true
		return nil
	}

	// Wrap the command
	WrapCommandWithCleanErrors(cmd, originalRunE)

	// RunE should be wrapped
	require.NotNil(t, cmd.RunE)

	// Execute the wrapped function
	err := cmd.RunE(cmd, []string{})

	assert.NoError(t, err)
	assert.True(t, executed)
	// SilenceUsage should remain false on success
	assert.False(t, cmd.SilenceUsage)
}

func TestWrapCommandWithCleanErrors_ErrorExecution(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	executed := false
	expectedError := assert.AnError
	originalRunE := func(cmd *cobra.Command, args []string) error {
		executed = true
		return expectedError
	}

	// Wrap the command
	WrapCommandWithCleanErrors(cmd, originalRunE)

	// RunE should be wrapped
	require.NotNil(t, cmd.RunE)

	// Execute the wrapped function
	err := cmd.RunE(cmd, []string{})

	assert.Equal(t, expectedError, err)
	assert.True(t, executed)
	// SilenceUsage should be true on error
	assert.True(t, cmd.SilenceUsage)
}

func TestWrapCommandWithCleanErrors_PreservesArgs(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	var receivedCmd *cobra.Command
	var receivedArgs []string

	originalRunE := func(cmd *cobra.Command, args []string) error {
		receivedCmd = cmd
		receivedArgs = args
		return nil
	}

	// Wrap the command
	WrapCommandWithCleanErrors(cmd, originalRunE)

	// Execute with specific arguments
	testArgs := []string{"arg1", "arg2", "arg3"}
	err := cmd.RunE(cmd, testArgs)

	assert.NoError(t, err)
	assert.Equal(t, cmd, receivedCmd)
	assert.Equal(t, testArgs, receivedArgs)
}

func TestWrapCommandWithCleanErrors_MultipleCalls(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	callCount := 0
	originalRunE := func(cmd *cobra.Command, args []string) error {
		callCount++
		if callCount == 1 {
			return nil // First call succeeds
		}
		return assert.AnError // Second call fails
	}

	// Wrap the command
	WrapCommandWithCleanErrors(cmd, originalRunE)

	// First execution - success
	err1 := cmd.RunE(cmd, []string{})
	assert.NoError(t, err1)
	assert.False(t, cmd.SilenceUsage) // Should be false after success

	// Reset SilenceUsage to test second call
	cmd.SilenceUsage = false

	// Second execution - error
	err2 := cmd.RunE(cmd, []string{})
	assert.Equal(t, assert.AnError, err2)
	assert.True(t, cmd.SilenceUsage) // Should be true after error

	assert.Equal(t, 2, callCount)
}

func TestWrapCommandWithCleanErrors_ChainedCalls(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	call1 := false
	call2 := false

	runE1 := func(cmd *cobra.Command, args []string) error {
		call1 = true
		return nil
	}

	runE2 := func(cmd *cobra.Command, args []string) error {
		call2 = true
		return assert.AnError
	}

	// First wrap
	WrapCommandWithCleanErrors(cmd, runE1)

	// Second wrap (simulating multiple wrappers)
	WrapCommandWithCleanErrors(cmd, runE2)

	// Should have new wrapper
	assert.NotNil(t, cmd.RunE)

	// Execute - should call runE2, not runE1
	err := cmd.RunE(cmd, []string{})

	assert.Equal(t, assert.AnError, err)
	assert.False(t, call1) // First function should not be called
	assert.True(t, call2)  // Second function should be called
	assert.True(t, cmd.SilenceUsage)
}

func TestConfigureCommandErrorHandling_DeepNesting(t *testing.T) {
	// Create deeply nested command structure
	root := &cobra.Command{Use: "root"}
	level1 := &cobra.Command{Use: "level1"}
	level2 := &cobra.Command{Use: "level2"}
	level3 := &cobra.Command{Use: "level3"}
	level4 := &cobra.Command{Use: "level4"}

	// Build hierarchy
	level3.AddCommand(level4)
	level2.AddCommand(level3)
	level1.AddCommand(level2)
	root.AddCommand(level1)

	// Configure error handling
	ConfigureCommandErrorHandling(root)

	// All levels should have SilenceUsage = true
	assert.True(t, root.SilenceUsage)
	assert.True(t, level1.SilenceUsage)
	assert.True(t, level2.SilenceUsage)
	assert.True(t, level3.SilenceUsage)
	assert.True(t, level4.SilenceUsage)
}

func TestConfigureCommandErrorHandling_MultipleSubcommands(t *testing.T) {
	root := &cobra.Command{Use: "root"}

	// Create multiple subcommands at the same level
	var subcommands []*cobra.Command
	for i := 0; i < 5; i++ {
		subCmd := &cobra.Command{
			Use:   "sub" + string(rune('0'+i)),
			Short: "Subcommand " + string(rune('0'+i)),
		}
		subcommands = append(subcommands, subCmd)
		root.AddCommand(subCmd)
	}

	// Configure error handling
	ConfigureCommandErrorHandling(root)

	// Root and all subcommands should have SilenceUsage = true
	assert.True(t, root.SilenceUsage)
	for i, subCmd := range subcommands {
		assert.True(t, subCmd.SilenceUsage, "Subcommand %d should have SilenceUsage = true", i)
	}
}
