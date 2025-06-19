package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
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
		// Interactive fallback
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

func getDataDir() string {
	dir := ".data"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.Mkdir(dir, 0755)
	}
	return dir
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
	inventoryDbCmd.AddCommand(inventoryDbListCmd)
	inventoryDbCmd.AddCommand(inventoryDbSetCmd)
	inventoryDbCmd.AddCommand(inventoryDbGetCmd)
	inventoryCmd.AddCommand(inventoryDbCmd)

	inventoryNodeCmd.AddCommand(inventoryNodeListCmd)
	inventoryNodeCmd.AddCommand(inventoryNodeSetCmd)
	inventoryNodeCmd.AddCommand(inventoryNodeGetCmd)
	inventoryCmd.AddCommand(inventoryNodeCmd)

	rootCmd.AddCommand(inventoryCmd)
}
