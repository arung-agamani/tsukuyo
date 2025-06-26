package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/arung-agamani/tsukuyo/internal/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// inventoryCmd represents the inventory command
var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Manage local hierarchical inventory store",
	Long:  `Manage a local key-value inventory with hierarchical structure, similar to jq queries.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no args, show help
		if len(args) == 0 {
			showInventoryHelp(cmd)
			return nil
		}

		// Check if first arg is a dynamic inventory type
		typeName := args[0]
		hi, err := getHierarchicalInventory()
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to initialize inventory:", err)
			return nil
		}

		// Get available types
		keys, err := hi.List("")
		if err == nil {
			for _, key := range keys {
				if key == typeName {
					// This is a dynamic type command
					return handleDynamicTypeCommand(cmd, hi, args)
				}
			}
		}

		// Not a dynamic type, show help
		showInventoryHelp(cmd)
		return nil
	},
}

var (
	cachedDataDir string
	dataDirOnce   sync.Once
)

func getDataDir() string {
	dataDirOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "." // fallback
		}
		cachedDataDir = home + "/.tsukuyo"
	})
	return cachedDataDir
}

func ensureDataDir() error {
	dir := getDataDir()
	return os.MkdirAll(dir, 0755)
}

// Migration command: copy .data inventory files to ~/.tsukuyo
var inventoryMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate inventory data from .data to ~/.tsukuyo",
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Could not determine home directory:", err)
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
					fmt.Fprintln(cmd.OutOrStdout(), "Failed to read", oldPath, ":", err)
					continue
				}
				err = os.WriteFile(newPath, b, 0644)
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "Failed to write", newPath, ":", err)
					continue
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Migrated", f, "to", newPath)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No", f, "found in", oldDir)
			}
		}
	},
}

func init() {
	inventoryCmd.AddCommand(inventoryMigrateCmd)

	rootCmd.AddCommand(inventoryCmd)
}

// showInventoryHelp displays the main inventory help with dynamic types
func showInventoryHelp(cmd *cobra.Command) {
	out := cmd.OutOrStdout()

	// Get available inventory types dynamically
	hi, err := getHierarchicalInventory()
	if err != nil {
		fmt.Fprintln(out, "Failed to initialize inventory:", err)
		return
	}

	// Get top-level keys (inventory types)
	keys, err := hi.List("")
	if err != nil || len(keys) == 0 {
		fmt.Fprintln(out, "No inventory data found.")
		fmt.Fprintln(out, "\nQuick start:")
		fmt.Fprintln(out, "  tsukuyo inventory set db.server1.host \"example.com\"")
		fmt.Fprintln(out, "  tsukuyo inventory set node.web1.host \"192.168.1.10\"")
		return
	}

	fmt.Fprintln(out, "🗄️  Hierarchical Inventory Management")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Available inventory types:")
	for _, key := range keys {
		fmt.Fprintf(out, "  - %-10s (tsukuyo inventory %s list)\n", key, key)
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "📋 Query Commands:")
	fmt.Fprintln(out, "  tsukuyo inventory query <path>          # Query any data path")
	fmt.Fprintln(out, "  tsukuyo inventory query db.server1.host # Query specific value")
	fmt.Fprintln(out, "  tsukuyo inventory query db.[*].host     # Query with wildcards")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "⚙️  Management Commands:")
	fmt.Fprintln(out, "  tsukuyo inventory set <path> <value>    # Set a value")
	fmt.Fprintln(out, "  tsukuyo inventory delete <path>         # Delete a value")
	fmt.Fprintln(out, "  tsukuyo inventory list [path]           # List keys at path")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "🏷️  Type-specific Commands:")
	for _, key := range keys {
		fmt.Fprintf(out, "  tsukuyo inventory %-8s list         # List all %s entries\n", key, key)
		fmt.Fprintf(out, "  tsukuyo inventory %-8s get <n>   # Get specific %s entry\n", key, key)
		fmt.Fprintf(out, "  tsukuyo inventory %-8s set <n> <value> # Set %s entry\n", key, key)
	}
}

// handleDynamicTypeCommand handles commands for dynamically discovered inventory types
func handleDynamicTypeCommand(cmd *cobra.Command, hi *inventory.HierarchicalInventory, args []string) error {
	out := cmd.OutOrStdout()

	if len(args) == 0 {
		return fmt.Errorf("no type specified")
	}

	typeName := args[0]
	subArgs := args[1:]

	// Handle subcommands
	if len(subArgs) == 0 {
		// Show help for this type
		fmt.Fprintf(out, "📁 %s Inventory\n", strings.Title(typeName))
		fmt.Fprintf(out, "Use 'tsukuyo inventory %s <command>' where <command> is:\n", typeName)
		fmt.Fprintf(out, "  list                    # List all %s entries\n", typeName)
		fmt.Fprintf(out, "  get <n>              # Get specific %s entry\n", typeName)
		fmt.Fprintf(out, "  set <n> <value>      # Set %s entry\n", typeName)
		fmt.Fprintf(out, "\nOr use hierarchical queries:\n")
		fmt.Fprintf(out, "  tsukuyo inventory query %s.<n>.<field>\n", typeName)
		return nil
	}

	subCommand := subArgs[0]
	subSubArgs := subArgs[1:]

	switch subCommand {
	case "list":
		return handleTypeList(cmd, hi, typeName)
	case "get":
		return handleTypeGet(cmd, hi, typeName, subSubArgs)
	case "set":
		return handleTypeSet(cmd, hi, typeName, subSubArgs)
	default:
		errorMsg := fmt.Sprintf("unknown subcommand '%s'. Available: list, get, set", subCommand)
		fmt.Fprintln(out, errorMsg)
		return fmt.Errorf(errorMsg)
	}
}

// Handler functions for dynamic type commands

func handleTypeList(cmd *cobra.Command, hi *inventory.HierarchicalInventory, typeName string) error {
	out := cmd.OutOrStdout()

	keys, err := hi.List(typeName)
	if err != nil {
		fmt.Fprintf(out, "No %s entries found.\n", typeName)
		return nil
	}

	if len(keys) == 0 {
		fmt.Fprintf(out, "No %s entries found.\n", typeName)
		return nil
	}

	fmt.Fprintf(out, "Available %s entries:\n", typeName)
	for _, key := range keys {
		fmt.Fprintf(out, "  - %s\n", key)
	}
	return nil
}

func handleTypeGet(cmd *cobra.Command, hi *inventory.HierarchicalInventory, typeName string, args []string) error {
	out := cmd.OutOrStdout()
	var name string
	var err error

	if len(args) > 0 {
		name = args[0]
	} else {
		// Interactive selection
		keys, err := hi.List(typeName)
		if err != nil || len(keys) == 0 {
			fmt.Fprintf(out, "No %s entries found.\n", typeName)
			return nil
		}

		prompt := promptui.Select{
			Label: fmt.Sprintf("Select %s entry", typeName),
			Items: keys,
		}
		_, name, err = prompt.Run()
		if err != nil {
			return fmt.Errorf("selection failed: %v", err)
		}
	}

	result, err := hi.Query(fmt.Sprintf("%s.%s", typeName, name))
	if err != nil {
		fmt.Fprintf(out, "Entry '%s' not found in %s inventory.\n", name, typeName)
		return nil
	}

	fmt.Fprintf(out, "%s.%s:\n", typeName, name)
	switch v := result.(type) {
	case string:
		fmt.Fprintf(out, "  %s\n", v)
	case map[string]interface{}, []interface{}:
		jsonBytes, err := json.MarshalIndent(v, "  ", "  ")
		if err != nil {
			fmt.Fprintf(out, "  %v\n", v)
		} else {
			fmt.Fprintf(out, "  %s\n", string(jsonBytes))
		}
	default:
		fmt.Fprintf(out, "  %v\n", v)
	}
	return nil
}

func handleTypeSet(cmd *cobra.Command, hi *inventory.HierarchicalInventory, typeName string, args []string) error {
	out := cmd.OutOrStdout()
	var name, valueStr string
	var err error

	if len(args) > 0 {
		name = args[0]
	} else {
		prompt := promptui.Prompt{
			Label: fmt.Sprintf("Enter %s name", typeName),
		}
		name, err = prompt.Run()
		if err != nil {
			return fmt.Errorf("input failed: %v", err)
		}
	}

	if len(args) > 1 {
		valueStr = args[1]
	} else {
		prompt := promptui.Prompt{
			Label: "Enter value (JSON or string)",
		}
		valueStr, err = prompt.Run()
		if err != nil {
			return fmt.Errorf("input failed: %v", err)
		}
	}

	// Try to parse as JSON, fall back to string
	var value interface{}
	if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
		value = valueStr
	}

	path := fmt.Sprintf("%s.%s", typeName, name)
	err = hi.Set(path, value)
	if err != nil {
		return fmt.Errorf("failed to set %s: %v", path, err)
	}

	fmt.Fprintf(out, "Set %s = %v\n", path, value)
	return nil
}
