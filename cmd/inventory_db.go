package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var inventoryDbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage database inventory",
}

var inventoryDbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all DB inventory keys",
	Run: func(cmd *cobra.Command, args []string) {
		items := loadDbInventory()
		if len(items) == 0 {
			fmt.Println("No DB inventory found.")
			return
		}
		fmt.Println("Available DB keys:")
		for k := range items {
			fmt.Println("-", k)
		}
	},
}

var inventoryDbSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a DB inventory item",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		items := loadDbInventory()
		var key, value string
		if len(args) > 0 {
			key = args[0]
		} else {
			prompt := promptui.Prompt{Label: "Enter DB key name"}
			key, _ = prompt.Run()
		}
		if len(args) > 1 {
			value = args[1]
		} else {
			prompt := promptui.Prompt{Label: "Enter DB value (hostname)"}
			value, _ = prompt.Run()
		}
		if key == "" || value == "" {
			fmt.Println("Key and value must not be empty.")
			return
		}
		items[key] = value
		saveDbInventory(items)
		fmt.Println("DB inventory set:", key, "=", value)
	},
}

var inventoryDbGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a DB inventory item interactively",
	Run: func(cmd *cobra.Command, args []string) {
		items := loadDbInventory()
		if len(items) == 0 {
			fmt.Println("No DB inventory found.")
			return
		}
		keys := make([]string, 0, len(items))
		for k := range items {
			keys = append(keys, k)
		}
		prompt := promptui.Select{Label: "Select DB key", Items: keys}
		_, key, err := prompt.Run()
		if err != nil {
			fmt.Println("Prompt failed:", err)
			return
		}
		fmt.Printf("%s = %s\n", key, items[key])
	},
}

func loadDbInventory() map[string]string {
	invFile := getDataDir() + "/db-inventory.json"
	items := map[string]string{}
	b, err := os.ReadFile(invFile)
	if err == nil {
		_ = json.Unmarshal(b, &items)
	}
	return items
}

func saveDbInventory(items map[string]string) {
	invFile := getDataDir() + "/db-inventory.json"
	b, _ := json.MarshalIndent(items, "", "  ")
	_ = os.WriteFile(invFile, b, 0644)
}

func init() {
	inventoryDbCmd.AddCommand(inventoryDbListCmd)
	inventoryDbCmd.AddCommand(inventoryDbSetCmd)
	inventoryDbCmd.AddCommand(inventoryDbGetCmd)
	inventoryCmd.AddCommand(inventoryDbCmd)
}
