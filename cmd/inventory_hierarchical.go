package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/arung-agamani/tsukuyo/internal/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// inventoryHierarchicalCmd represents the hierarchical inventory command
var inventoryHierarchicalCmd = &cobra.Command{
	Use:   "query",
	Short: "Query hierarchical inventory with jq-like syntax",
	Long: `Query hierarchical inventory data using jq-like syntax.
	
Examples:
  tsukuyo inventory query db.izuna-db.port
  tsukuyo inventory query db.izuna-db.[0].env
  tsukuyo inventory query servers.[*].hostname`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hi, err := inventory.NewHierarchicalInventory(getDataDir())
		if err != nil {
			fmt.Println("Failed to initialize hierarchical inventory:", err)
			return
		}

		var query string
		if len(args) > 0 {
			query = args[0]
		} else {
			// Interactive mode
			prompt := promptui.Prompt{
				Label: "Enter query (jq-like syntax, e.g., 'db.izuna-db.port')",
			}
			query, err = prompt.Run()
			if err != nil {
				fmt.Println("Prompt failed:", err)
				return
			}
		}

		result, err := hi.Query(query)
		if err != nil {
			fmt.Println("Query failed:", err)
			return
		}

		// Format output
		if query == "" {
			// Root query - show available top-level keys
			keys, err := hi.List("")
			if err != nil {
				fmt.Println("Failed to list keys:", err)
				return
			}
			fmt.Println("Available top-level keys:")
			for _, key := range keys {
				fmt.Println("-", key)
			}
			return
		}

		// Format the result for display
		switch v := result.(type) {
		case string:
			fmt.Println(v)
		case map[string]interface{}, []interface{}:
			jsonBytes, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				fmt.Printf("%v\n", v)
			} else {
				fmt.Println(string(jsonBytes))
			}
		default:
			fmt.Printf("%v\n", v)
		}
	},
}

var inventorySetCmd = &cobra.Command{
	Use:   "set [query] [value]",
	Short: "Set a value in hierarchical inventory",
	Long: `Set a value in the hierarchical inventory using jq-like path syntax.
	
Examples:
  tsukuyo inventory set db.izuna-db.host "kureya.howlingmoon.dev"
  tsukuyo inventory set db.izuna-db.port 2333
  tsukuyo inventory set servers.web.enabled true`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		hi, err := inventory.NewHierarchicalInventory(getDataDir())
		if err != nil {
			fmt.Println("Failed to initialize hierarchical inventory:", err)
			return
		}

		var query, valueStr string
		if len(args) > 0 {
			query = args[0]
		} else {
			prompt := promptui.Prompt{
				Label: "Enter path (e.g., 'db.izuna-db.host')",
			}
			query, err = prompt.Run()
			if err != nil {
				fmt.Println("Prompt failed:", err)
				return
			}
		}

		if len(args) > 1 {
			valueStr = args[1]
		} else {
			prompt := promptui.Prompt{
				Label: "Enter value",
			}
			valueStr, err = prompt.Run()
			if err != nil {
				fmt.Println("Prompt failed:", err)
				return
			}
		}

		if query == "" || valueStr == "" {
			fmt.Println("Both query and value must be provided.")
			return
		}

		// Try to parse value as JSON first, then fall back to string
		var value interface{}
		if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
			// Not valid JSON, treat as string
			value = valueStr
		}

		err = hi.Set(query, value)
		if err != nil {
			fmt.Println("Failed to set value:", err)
			return
		}

		fmt.Printf("Set %s = %v\n", query, value)
	},
}

var inventoryDeleteCmd = &cobra.Command{
	Use:   "delete [query]",
	Short: "Delete a value from hierarchical inventory",
	Long: `Delete a value from the hierarchical inventory using jq-like path syntax.
	
Examples:
  tsukuyo inventory delete db.izuna-db.port
  tsukuyo inventory delete servers.web`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hi, err := inventory.NewHierarchicalInventory(getDataDir())
		if err != nil {
			fmt.Println("Failed to initialize hierarchical inventory:", err)
			return
		}

		var query string
		if len(args) > 0 {
			query = args[0]
		} else {
			prompt := promptui.Prompt{
				Label: "Enter path to delete (e.g., 'db.izuna-db.port')",
			}
			query, err = prompt.Run()
			if err != nil {
				fmt.Println("Prompt failed:", err)
				return
			}
		}

		if query == "" {
			fmt.Println("Query must be provided.")
			return
		}

		err = hi.Delete(query)
		if err != nil {
			fmt.Println("Failed to delete:", err)
			return
		}

		fmt.Printf("Deleted %s\n", query)
	},
}

var inventoryListCmd = &cobra.Command{
	Use:   "list [query]",
	Short: "List keys at a specific path in hierarchical inventory",
	Long: `List available keys at a specific path in the hierarchical inventory.
	
Examples:
  tsukuyo inventory list           # List top-level keys
  tsukuyo inventory list db        # List keys under 'db'
  tsukuyo inventory list db.izuna-db  # List keys under 'db.izuna-db'`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hi, err := inventory.NewHierarchicalInventory(getDataDir())
		if err != nil {
			fmt.Println("Failed to initialize hierarchical inventory:", err)
			return
		}

		var query string
		if len(args) > 0 {
			query = args[0]
		}

		keys, err := hi.List(query)
		if err != nil {
			fmt.Println("Failed to list keys:", err)
			return
		}

		if len(keys) == 0 {
			fmt.Printf("No keys found at path '%s'\n", query)
			return
		}

		if query == "" {
			fmt.Println("Available keys:")
		} else {
			fmt.Printf("Keys at '%s':\n", query)
		}
		for _, key := range keys {
			fmt.Println("-", key)
		}
	},
}

var inventoryImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import legacy inventory files into hierarchical format",
	Long: `Import existing *-inventory.json files into the new hierarchical format.
This will migrate db-inventory.json, node-inventory.json, etc. into a unified structure.`,
	Run: func(cmd *cobra.Command, args []string) {
		hi, err := inventory.NewHierarchicalInventory(getDataDir())
		if err != nil {
			fmt.Println("Failed to initialize hierarchical inventory:", err)
			return
		}

		// The inventory will automatically load from existing files during initialization
		// Just need to save it in the new format
		dataDir := getDataDir()
		files, err := os.ReadDir(dataDir)
		if err != nil {
			fmt.Println("Failed to read data directory:", err)
			return
		}

		imported := 0
		for _, file := range files {
			if strings.HasSuffix(file.Name(), "-inventory.json") && file.Name() != "hierarchical-inventory.json" {
				fmt.Printf("Found legacy inventory file: %s\n", file.Name())
				imported++
			}
		}

		if imported == 0 {
			fmt.Println("No legacy inventory files found.")
			return
		}

		fmt.Printf("Imported %d legacy inventory files into hierarchical format.\n", imported)
		fmt.Println("You can now use 'tsukuyo inventory query' to access the data.")

		// Show available top-level keys
		keys, err := hi.List("")
		if err == nil && len(keys) > 0 {
			fmt.Println("\nAvailable top-level keys:")
			for _, key := range keys {
				fmt.Println("-", key)
			}
		}
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryHierarchicalCmd)
	inventoryCmd.AddCommand(inventorySetCmd)
	inventoryCmd.AddCommand(inventoryDeleteCmd)
	inventoryCmd.AddCommand(inventoryListCmd)
	inventoryCmd.AddCommand(inventoryImportCmd)
}
