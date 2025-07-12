package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/arung-agamani/tsukuyo/internal/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Connect to a node using standard SSH client or manage SSH node inventory",
	Long: `Connect to a node using OpenSSH, or manage SSH node inventory.\n\n\
Direct connect: tsukuyo ssh <node-name>\n\
Manage inventory: tsukuyo ssh set|get|list [args]\n\
Supports SSH tunneling with --tunnel flag.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Usage: tsukuyo ssh <node-name>|set|get|list [args]")
			return
		}

		// Get hierarchical inventory
		hi, err := getHierarchicalInventory()
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to initialize inventory:", err)
			return
		}

		cmds := map[string]bool{"set": true, "get": true, "list": true}
		if cmds[args[0]] {
			switch args[0] {
			case "set":
				var name, host, user string
				if len(args) > 1 {
					name = args[1]
				} else {
					prompt := promptui.Prompt{Label: "Node name (alias)"}
					name, _ = prompt.Run()
				}
				if name == "set" || name == "get" || name == "list" {
					fmt.Fprintln(cmd.OutOrStdout(), "Invalid node name: cannot be 'set', 'get', or 'list'.")
					return
				}
				if len(args) > 2 {
					host = args[2]
				} else {
					prompt := promptui.Prompt{Label: "Node host (hostname or IP)"}
					host, _ = prompt.Run()
				}
				if name == "" || host == "" {
					fmt.Fprintln(cmd.OutOrStdout(), "Name and host must not be empty.")
					return
				}
				// Prompt for user, default to current shell user
				if u := os.Getenv("USER"); u != "" {
					user = u
				}
				prompt := promptui.Prompt{Label: "SSH user", Default: user}
				user, _ = prompt.Run()
				if user == "" {
					fmt.Fprintln(cmd.OutOrStdout(), "User must not be empty.")
					return
				}

				// Prompt for tags
				tagPrompt := promptui.Prompt{Label: "Tags (comma-separated)"}
				tagsStr, _ := tagPrompt.Run()
				var tags []string
				if tagsStr != "" {
					tags = strings.Split(tagsStr, ",")
					for i := range tags {
						tags[i] = strings.TrimSpace(tags[i])
					}
				}

				// Create node entry in hierarchical inventory
				nodeData := map[string]interface{}{
					"name": name,
					"host": host,
					"type": "ssh",
					"user": user,
					"tags": tags,
				}

				path := fmt.Sprintf("node.%s", name)
				err = hi.Set(path, nodeData)
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "Failed to set node:", err)
					return
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Node '%s' set to host '%s' with user '%s'\n", name, host, user)

			case "get":
				nodeKeys, err := hi.List("node")
				if err != nil || len(nodeKeys) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No SSH node inventory found.")
					return
				}

				var name string
				if len(args) > 1 {
					name = args[1]
				} else {
					prompt := promptui.Select{Label: "Select node", Items: nodeKeys}
					_, name, _ = prompt.Run()
				}

				result, err := hi.Query(fmt.Sprintf("node.%s", name))
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "Node not found.")
					return
				}

				// Parse the node data
				nodeData, ok := result.(map[string]interface{})
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Invalid node data format.")
					return
				}

				host, _ := nodeData["host"].(string)
				nodeType, _ := nodeData["type"].(string)
				user, _ := nodeData["user"].(string)
				port := 22 // default
				if p, ok := nodeData["port"].(float64); ok {
					port = int(p)
				}
				tags := getNodeTags(nodeData)

				fmt.Fprintf(cmd.OutOrStdout(), "%s: host=%s, type=%s, port=%d, user=%s, tags=%s\n", name, host, nodeType, port, user, strings.Join(tags, ","))

			case "list":
				nodeKeys, err := hi.List("node")
				if err != nil || len(nodeKeys) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No SSH node inventory found.")
					return
				}

				fmt.Fprintln(cmd.OutOrStdout(), "Available SSH nodes:")
				for _, nodeName := range nodeKeys {
					result, err := hi.Query(fmt.Sprintf("node.%s", nodeName))
					if err != nil {
						continue
					}

					nodeData, ok := result.(map[string]interface{})
					if !ok {
						continue
					}

					host, _ := nodeData["host"].(string)
					nodeType, _ := nodeData["type"].(string)
					user, _ := nodeData["user"].(string)
					port := 22 // default
					if p, ok := nodeData["port"].(float64); ok {
						port = int(p)
					}
					tags := getNodeTags(nodeData)

					fmt.Fprintf(cmd.OutOrStdout(), "- %s: host=%s, type=%s, port=%d, user=%s, tags=[%s]\n", nodeName, host, nodeType, port, user, strings.Join(tags, ", "))
				}
			}
			return
		}

		// Not a command, treat as node name
		name := args[0]
		result, err := hi.Query(fmt.Sprintf("node.%s", name))
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Node or command not found.")
			return
		}

		// Parse the node data
		nodeData, ok := result.(map[string]interface{})
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Invalid node data format.")
			return
		}

		host, _ := nodeData["host"].(string)
		user, _ := nodeData["user"].(string)
		if user == "" {
			user = "ubuntu"
		}

		port := 22 // default
		if p, ok := nodeData["port"].(float64); ok {
			port = int(p)
		}

		sshArgs := []string{}
		if port != 22 {
			sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", user, host), "-p", fmt.Sprintf("%d", port))
		} else {
			sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", user, host))
		}

		if withDbSsh == "__INTERACTIVE__" {
			withDbSsh = ""
		}
		if withDbSsh != "" || cmd.Flags().Changed("with-db") {
			dbEntry, err := selectDbWithTagging(hi, nodeData)
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), err)
				return
			}

			localPort := dbEntry.LocalPort
			if localPort == 0 {
				localPort = dbEntry.RemotePort // Default to same as remote
			}
			tunnel := fmt.Sprintf("%d:%s:%d", localPort, dbEntry.Host, dbEntry.RemotePort)
			sshArgs = append([]string{"-L", tunnel}, sshArgs...)
			fmt.Fprintf(cmd.OutOrStdout(), "Forwarding local port %d to %s:%d\n", localPort, dbEntry.Host, dbEntry.RemotePort)
		}

		if tunnelTarget != "" {
			sshArgs = append([]string{"-L", tunnelTarget}, sshArgs...)
		}

		sshExec := exec.Command("ssh", sshArgs...)
		sshExec.Stdin = cmd.InOrStdin()
		sshExec.Stdout = cmd.OutOrStdout()
		sshExec.Stderr = cmd.ErrOrStderr()
		err = sshExec.Run()
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "SSH exited with error:", err)
		}
	},
}

var tunnelTarget string
var withDbSsh string

func init() {
	sshCmd.Flags().StringVar(&tunnelTarget, "tunnel", "", "Tunnel in format localPort:remoteHost:remotePort (optional)")
	sshCmd.Flags().StringVar(&withDbSsh, "with-db", "", "Tunnel to DB key from inventory (interactive if empty)")
	sshCmd.Flags().Lookup("with-db").NoOptDefVal = "__INTERACTIVE__"
	rootCmd.AddCommand(sshCmd)
}

func selectDbWithTagging(hi *inventory.HierarchicalInventory, nodeData map[string]interface{}) (*DbInventoryEntry, error) {
	dbEntries, err := hi.List("db")
	if err != nil || len(dbEntries) == 0 {
		return nil, fmt.Errorf("no DB inventory found")
	}

	nodeTags := getNodeTags(nodeData)
	var filteredEntries []string
	entryMap := make(map[string]DbInventoryEntry)

	for _, key := range dbEntries {
		entryData, err := hi.Query(fmt.Sprintf("db.%s", key))
		if err != nil {
			continue
		}
		var entry DbInventoryEntry
		// manual parsing from map[string]interface{} to struct
		if raw, ok := entryData.(map[string]interface{}); ok {
			if h, ok := raw["host"].(string); ok {
				entry.Host = h
			}
			if t, ok := raw["type"].(string); ok {
				entry.Type = t
			}
			if rp, ok := raw["remote_port"].(float64); ok {
				entry.RemotePort = int(rp)
			}
			if lp, ok := raw["local_port"].(float64); ok {
				entry.LocalPort = int(lp)
			}
			if tags, ok := raw["tags"].([]interface{}); ok {
				for _, tag := range tags {
					if t, ok := tag.(string); ok {
						entry.Tags = append(entry.Tags, t)
					}
				}
			}
		}

		if len(entry.Tags) == 0 || hasCommonTags(nodeTags, entry.Tags) {
			filteredEntries = append(filteredEntries, key)
			entryMap[key] = entry
		}
	}

	if len(filteredEntries) == 0 {
		return nil, fmt.Errorf("no DB entries with matching tags found")
	}

	prompt := promptui.Select{
		Label: "Select DB key for tunnel",
		Items: filteredEntries,
		Searcher: func(input string, index int) bool {
			return strings.Contains(strings.ToLower(filteredEntries[index]), strings.ToLower(input))
		},
	}
	_, selectedKey, err := prompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %v", err)
	}

	selectedEntry := entryMap[selectedKey]
	return &selectedEntry, nil
}

func getNodeTags(nodeData map[string]interface{}) []string {
	if tags, ok := nodeData["tags"].([]interface{}); ok {
		var stringTags []string
		for _, tag := range tags {
			if t, ok := tag.(string); ok {
				stringTags = append(stringTags, t)
			}
		}
		return stringTags
	}
	return []string{}
}

func hasCommonTags(tags1, tags2 []string) bool {
	for _, t1 := range tags1 {
		for _, t2 := range tags2 {
			if t1 == t2 {
				return true
			}
		}
	}
	return false
}
