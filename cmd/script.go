package cmd

import (
	"github.com/spf13/cobra"
)

// scriptCmd represents the script command
var scriptCmd = &cobra.Command{
	Use:   "script",
	Short: "Manage and execute script inventory",
	Long:  `Conveniently execute, view, and edit predefined scripts (bash for now, later node/deno/python).`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement script inventory logic
	},
}

func init() {
	rootCmd.AddCommand(scriptCmd)
}
