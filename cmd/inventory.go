package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// inventoryCmd represents the inventory command
var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Manage local hierarchical inventory store",
	Long:  `Manage a local key-value inventory with hierarchical structure, similar to jq queries.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available inventory types:")
		fmt.Println("- db   (Database inventory)")
		fmt.Println("- node (SSH node inventory)")
		fmt.Println("\nTry: tsukuyo inventory db list")
	},
}

func getDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "." // fallback
	}
	dir := home + "/.tsukuyo"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.Mkdir(dir, 0755)
	}
	return dir
}

// Migration command: copy .data inventory files to ~/.tsukuyo
var inventoryMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate inventory data from .data to ~/.tsukuyo",
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Could not determine home directory:", err)
			return
		}
		oldDir := ".data"
		newDir := home + "/.tsukuyo"
		files := []string{"db-inventory.json", "node-inventory.json"}
		for _, f := range files {
			oldPath := oldDir + "/" + f
			newPath := newDir + "/" + f
			if _, err := os.Stat(oldPath); err == nil {
				b, err := os.ReadFile(oldPath)
				if err != nil {
					fmt.Println("Failed to read", oldPath, ":", err)
					continue
				}
				err = os.WriteFile(newPath, b, 0644)
				if err != nil {
					fmt.Println("Failed to write", newPath, ":", err)
					continue
				}
				fmt.Println("Migrated", f, "to", newPath)
			} else {
				fmt.Println("No", f, "found in", oldDir)
			}
		}
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryMigrateCmd)

	rootCmd.AddCommand(inventoryCmd)
}
