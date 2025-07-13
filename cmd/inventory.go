package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/arung-agamani/tsukuyo/internal/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// Command-line flags for db set command
var (
	dbSetType       string
	dbSetRemotePort int
	dbSetLocalPort  int
	dbSetTags       string
)

// ensureDbInventoryInitialized ensures the db inventory is properly initialized
func ensureDbInventoryInitialized(hi *inventory.HierarchicalInventory) error {
	// Check if db key exists
	result, err := hi.Query("db")
	if err != nil {
		// DB key doesn't exist, initialize it as an empty map
		return hi.Set("db", make(map[string]interface{}))
	}

	// Check if it's a map/object type
	if _, ok := result.(map[string]interface{}); !ok {
		// DB key exists but is not a map, reinitialize it
		return hi.Set("db", make(map[string]interface{}))
	}

	// Validate existing entries follow the correct structure
	dbMap := result.(map[string]interface{})
	for entryName, entryValue := range dbMap {
		if err := validateDbEntry(entryName, entryValue); err != nil {
			fmt.Printf("Warning: DB entry '%s' has invalid structure: %v\n", entryName, err)
			// Optionally, you could remove invalid entries or fix them here
		}
	}

	return nil
}

// validateDbEntry validates that a DB entry follows the correct structure
func validateDbEntry(name string, entry interface{}) error {
	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		return fmt.Errorf("entry is not a map/object")
	}

	// Check required fields
	if _, exists := entryMap["host"]; !exists {
		return fmt.Errorf("missing required field 'host'")
	}
	if _, exists := entryMap["type"]; !exists {
		return fmt.Errorf("missing required field 'type'")
	}
	if _, exists := entryMap["remote_port"]; !exists {
		return fmt.Errorf("missing required field 'remote_port'")
	}

	// Validate field types
	if _, ok := entryMap["host"].(string); !ok {
		return fmt.Errorf("field 'host' must be a string")
	}
	if _, ok := entryMap["type"].(string); !ok {
		return fmt.Errorf("field 'type' must be a string")
	}

	// remote_port can be stored as float64 in JSON
	switch rp := entryMap["remote_port"].(type) {
	case float64:
		// Valid
	case int:
		// Valid
	default:
		return fmt.Errorf("field 'remote_port' must be a number, got %T", rp)
	}

	// Optional fields validation
	if localPort, exists := entryMap["local_port"]; exists {
		switch localPort.(type) {
		case float64, int:
			// Valid
		default:
			return fmt.Errorf("field 'local_port' must be a number, got %T", localPort)
		}
	}

	if tags, exists := entryMap["tags"]; exists {
		if _, ok := tags.([]interface{}); !ok {
			return fmt.Errorf("field 'tags' must be an array")
		}
	}

	return nil
}

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

		// Known inventory types that should always be available, even if empty/deleted
		knownTypes := []string{"db", "node", "script"}
		isKnownType := false
		for _, knownType := range knownTypes {
			if typeName == knownType {
				isKnownType = true
				break
			}
		}

		if isKnownType {
			// This is a known type command, handle it even if the key doesn't exist yet
			return handleDynamicTypeCommand(cmd, hi, args)
		}

		// Check if it's an existing dynamic type
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

var getDataDir = func() string {
	dataDirOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "." // fallback
		}
		cachedDataDir = home + "/.tsukuyo"
	})
	return cachedDataDir
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
	// Add flags for db set command
	inventoryCmd.PersistentFlags().StringVar(&dbSetType, "type", "", "Database type (e.g., postgres, redis, mongodb)")
	inventoryCmd.PersistentFlags().IntVar(&dbSetRemotePort, "remote-port", 0, "Remote port number")
	inventoryCmd.PersistentFlags().IntVar(&dbSetLocalPort, "local-port", 0, "Local port number (optional)")
	inventoryCmd.PersistentFlags().StringVar(&dbSetTags, "tags", "", "Comma-separated tags")

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
		titleCase := strings.ToUpper(string(typeName[0])) + typeName[1:]
		fmt.Fprintf(out, "📁 %s Inventory\n", titleCase)
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
		return errors.New(errorMsg)
	}
}

// Handler functions for dynamic type commands

func handleTypeList(cmd *cobra.Command, hi *inventory.HierarchicalInventory, typeName string) error {
	out := cmd.OutOrStdout()

	// Ensure DB inventory is initialized if we're working with DB type
	if typeName == "db" {
		if err := ensureDbInventoryInitialized(hi); err != nil {
			return fmt.Errorf("failed to initialize db inventory: %v", err)
		}
	}

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

	// Ensure DB inventory is initialized if we're working with DB type
	if typeName == "db" {
		if err := ensureDbInventoryInitialized(hi); err != nil {
			return fmt.Errorf("failed to initialize db inventory: %v", err)
		}
	}

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

	if typeName == "db" {
		return handleDbSet(cmd, hi, args)
	}

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

func handleDbSet(cmd *cobra.Command, hi *inventory.HierarchicalInventory, args []string) error {
	out := cmd.OutOrStdout()

	// Ensure DB inventory is properly initialized
	if err := ensureDbInventoryInitialized(hi); err != nil {
		return fmt.Errorf("failed to initialize db inventory: %v", err)
	}

	var name, host string
	var err error

	// Check if we have enough arguments for non-interactive mode
	hasName := len(args) > 0
	hasHost := len(args) > 1

	// Only go interactive if we don't have both name and host
	if !hasName || !hasHost {
		// Interactive mode for missing arguments
		if !hasName {
			prompt := promptui.Prompt{Label: "Enter DB entry name"}
			name, err = prompt.Run()
			if err != nil {
				return fmt.Errorf("input failed: %v", err)
			}
		} else {
			name = args[0]
		}

		if !hasHost {
			prompt := promptui.Prompt{Label: "Host"}
			host, err = prompt.Run()
			if err != nil {
				return fmt.Errorf("input failed: %v", err)
			}
		} else {
			host = args[1]
		}
	} else {
		// Non-interactive mode: use arguments
		name = args[0]
		host = args[1]
	}

	// Get values from flags or defaults
	dbType := dbSetType
	if dbType == "" {
		dbType = "postgres" // default
	}

	remotePort := dbSetRemotePort
	if remotePort == 0 {
		remotePort = 5432 // default
	}

	localPort := dbSetLocalPort
	// localPort can be 0 (optional)

	var tags []string
	if dbSetTags != "" {
		tags = strings.Split(dbSetTags, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	// If we're missing critical values and not provided via flags, go interactive for the rest
	if (!hasName || !hasHost) && (dbSetType == "" || dbSetRemotePort == 0) {
		if dbSetType == "" {
			prompt := promptui.Prompt{Label: "Type (e.g., postgres, redis)", Default: "postgres"}
			dbType, _ = prompt.Run()
		}

		if dbSetRemotePort == 0 {
			prompt := promptui.Prompt{Label: "Remote Port", Default: "5432"}
			remotePortStr, _ := prompt.Run()
			remotePort, _ = strconv.Atoi(remotePortStr)
		}

		if dbSetLocalPort == 0 && localPort == 0 {
			prompt := promptui.Prompt{Label: "Local Port (optional)"}
			localPortStr, _ := prompt.Run()
			if localPortStr != "" {
				localPort, _ = strconv.Atoi(localPortStr)
			}
		}

		if dbSetTags == "" && len(tags) == 0 {
			prompt := promptui.Prompt{Label: "Tags (comma-separated)"}
			tagsStr, _ := prompt.Run()
			if tagsStr != "" {
				tags = strings.Split(tagsStr, ",")
				for i := range tags {
					tags[i] = strings.TrimSpace(tags[i])
				}
			}
		}
	}

	entry := DbInventoryEntry{
		Host:       host,
		Type:       dbType,
		RemotePort: remotePort,
		LocalPort:  localPort,
		Tags:       tags,
	}

	path := fmt.Sprintf("db.%s", name)
	err = hi.Set(path, entry)
	if err != nil {
		return fmt.Errorf("failed to set db entry: %v", err)
	}

	fmt.Fprintf(out, "Set db.%s = %+v\n", name, entry)
	return nil
}
