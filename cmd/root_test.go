package cmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// newTestRootCmd creates a new instance of the root command for testing purposes.
// This helps in isolating tests from each other.
func newTestRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tsukuyo",
		Short: "A CLI tool to streamline SSH connections and inventory management",
		Long:  `Tsukuyo is a command-line tool designed to automate and streamline various operational tasks.`,
	}
	cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// Add subcommands to test that they are listed in help output
	cmd.AddCommand(&cobra.Command{Use: "ssh", Short: "Connect to a node"})
	cmd.AddCommand(&cobra.Command{Use: "tsh", Short: "Connect with Teleport"})
	return cmd
}

// executeCommandC is a helper function to execute a cobra command and capture its output.
func executeCommandC(cmd *cobra.Command, args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()

	return stdout.String(), stderr.String(), err
}

func TestRootCmd(t *testing.T) {
	t.Run("help flag", func(t *testing.T) {
		cmd := newTestRootCmd()
		stdout, _, err := executeCommandC(cmd, "--help")
		assert.NoError(t, err)
		assert.Contains(t, stdout, cmd.Long)
		assert.Contains(t, stdout, "Usage:")
		assert.Contains(t, stdout, "ssh", "Help output should contain subcommands")
	})

	t.Run("no arguments", func(t *testing.T) {
		cmd := newTestRootCmd()
		// For the root command without subcommands, no args should show help.
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return nil
		}
		stdout, _, err := executeCommandC(cmd)
		assert.NoError(t, err)
		assert.Contains(t, stdout, cmd.Long)
		assert.Contains(t, stdout, "Usage:")
	})

	t.Run("unknown command", func(t *testing.T) {
		cmd := newTestRootCmd()
		_, stderr, err := executeCommandC(cmd, "nonexistentcommand")
		assert.Error(t, err)
		assert.Contains(t, stderr, "Error: unknown command \"nonexistentcommand\" for \"tsukuyo\"")
	})

	t.Run("toggle flag", func(t *testing.T) {
		cmd := newTestRootCmd()
		_, _, err := executeCommandC(cmd, "--toggle")
		assert.NoError(t, err)

		toggle, err := cmd.Flags().GetBool("toggle")
		assert.NoError(t, err)
		assert.True(t, toggle)
	})
}

// TestExecute is a basic smoke test for the Execute function.
func TestExecute(t *testing.T) {
	// This test ensures the function can be called without panicking.
	// It's difficult to test the os.Exit behavior directly.

	// Redirect stdout and stderr to prevent test output pollution
	oldOut := rootCmd.OutOrStdout()
	oldErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	defer func() {
		rootCmd.SetOut(oldOut)
		rootCmd.SetErr(oldErr)
		// Reset args for other tests
		rootCmd.SetArgs([]string{})
	}()

	// We test the success path.
	Execute()
}
