package cmd

import (
	"bytes"

	"github.com/spf13/cobra"
)

// executeCommand executes a cobra command and returns its output and any error.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	b := new(bytes.Buffer)
	root.SetOut(b)
	root.SetErr(b)
	root.SetArgs(args)
	err := root.Execute()
	return b.String(), err
}
