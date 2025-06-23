package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var inventoryNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage SSH node inventory",
}

var inventoryNodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all SSH node inventory keys",
	Run: func(cmd *cobra.Command, args []string) {
		items := loadNodeInventory()
		if len(items) == 0 {
			fmt.Println("No SSH node inventory found.")
			return
		}
		fmt.Println("Available SSH node keys:")
		for k := range items {
			fmt.Println("-", k)
		}
	},
}

var inventoryNodeSetCmd = &cobra.Command{
	Use:   "set [name] [host] [user]",
	Short: "Set a SSH node inventory item",
	Args:  cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		items := loadNodeInventory()
		var name, host, user string
		if len(args) > 0 {
			name = args[0]
		} else {
			prompt := promptui.Prompt{Label: "Enter node name (alias)"}
			name, _ = prompt.Run()
		}
		if name == "" {
			fmt.Println("Name must not be empty.")
			return
		}
		if len(args) > 1 {
			host = args[1]
		} else {
			prompt := promptui.Prompt{Label: "Enter node host (hostname or IP)"}
			host, _ = prompt.Run()
		}
		if host == "" {
			fmt.Println("Host must not be empty.")
			return
		}
		if len(args) > 2 {
			user = args[2]
		} else {
			if u := os.Getenv("USER"); u != "" {
				user = u
			}
			prompt := promptui.Prompt{Label: "SSH user", Default: user}
			user, _ = prompt.Run()
		}
		if user == "" {
			fmt.Println("User must not be empty.")
			return
		}
		entry := NodeInventoryEntry{Name: name, Host: host, Type: "ssh", User: user}
		items[name] = entry
		saveNodeInventory(items)
		fmt.Printf("Node '%s' set to host '%s' with user '%s'\n", name, host, user)
	},
}

var inventoryNodeGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get a SSH node inventory item (interactive if no name provided)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		items := loadNodeInventory()
		if len(items) == 0 {
			fmt.Println("No SSH node inventory found.")
			return
		}
		if len(args) > 0 {
			key := args[0]
			entry, ok := items[key]
			if !ok {
				fmt.Println("Node not found.")
				return
			}
			fmt.Printf("%s: host=%s, type=%s, port=%d, user=%s\n", entry.Name, entry.Host, entry.Type, entry.Port, entry.User)
			return
		}
		keys := make([]string, 0, len(items))
		for k := range items {
			keys = append(keys, k)
		}
		prompt := promptui.Select{Label: "Select node key", Items: keys}
		_, key, err := prompt.Run()
		if err != nil {
			fmt.Println("Prompt failed:", err)
			return
		}
		entry := items[key]
		fmt.Printf("%s: host=%s, type=%s, port=%d, user=%s\n", entry.Name, entry.Host, entry.Type, entry.Port, entry.User)
	},
}

func loadNodeInventory() map[string]NodeInventoryEntry {
	invFile := getDataDir() + "/node-inventory.json"
	items := map[string]NodeInventoryEntry{}
	b, err := os.ReadFile(invFile)
	if err == nil {
		_ = json.Unmarshal(b, &items)
	}
	return items
}

func saveNodeInventory(items map[string]NodeInventoryEntry) {
	invFile := getDataDir() + "/node-inventory.json"
	b, _ := json.MarshalIndent(items, "", "  ")
	_ = os.WriteFile(invFile, b, 0644)
}

func init() {
	inventoryNodeCmd.AddCommand(inventoryNodeListCmd)
	inventoryNodeCmd.AddCommand(inventoryNodeSetCmd)
	inventoryNodeCmd.AddCommand(inventoryNodeGetCmd)
	inventoryCmd.AddCommand(inventoryNodeCmd)
}
