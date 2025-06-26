package cmd

import (
	"fmt"
	"os"
	"os/exec"

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

				// Create node entry in hierarchical inventory
				nodeData := map[string]interface{}{
					"name": name,
					"host": host,
					"type": "ssh",
					"user": user,
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

				fmt.Fprintf(cmd.OutOrStdout(), "%s: host=%s, type=%s, port=%d, user=%s\n", name, host, nodeType, port, user)

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

					fmt.Fprintf(cmd.OutOrStdout(), "- %s: host=%s, type=%s, port=%d, user=%s\n", nodeName, host, nodeType, port, user)
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

func init() {
	sshCmd.Flags().StringVar(&tunnelTarget, "tunnel", "", "Tunnel in format localPort:remoteHost:remotePort (optional)")
	rootCmd.AddCommand(sshCmd)
}
