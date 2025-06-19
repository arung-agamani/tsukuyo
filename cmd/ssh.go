package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NodeInventoryEntry represents a node entry for SSH
type NodeInventoryEntry struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Type string `json:"type"`
	Port int    `json:"port,omitempty"`
	User string `json:"user,omitempty"`
}

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
			fmt.Println("Usage: tsukuyo ssh <node-name>|set|get|list [args]")
			return
		}
		cmds := map[string]bool{"set": true, "get": true, "list": true}
		if cmds[args[0]] {
			switch args[0] {
			case "set":
				items := loadNodeInventory()
				var name, host, user string
				if len(args) > 1 {
					name = args[1]
				} else {
					prompt := promptui.Prompt{Label: "Node name (alias)"}
					name, _ = prompt.Run()
				}
				if name == "set" || name == "get" || name == "list" {
					fmt.Println("Invalid node name: cannot be 'set', 'get', or 'list'.")
					return
				}
				if len(args) > 2 {
					host = args[2]
				} else {
					prompt := promptui.Prompt{Label: "Node host (hostname or IP)"}
					host, _ = prompt.Run()
				}
				if name == "" || host == "" {
					fmt.Println("Name and host must not be empty.")
					return
				}
				// Prompt for user, default to current shell user
				if u := os.Getenv("USER"); u != "" {
					user = u
				}
				prompt := promptui.Prompt{Label: "SSH user", Default: user}
				user, _ = prompt.Run()
				if user == "" {
					fmt.Println("User must not be empty.")
					return
				}
				entry := NodeInventoryEntry{Name: name, Host: host, Type: "ssh", User: user}
				items[name] = entry
				saveNodeInventory(items)
				fmt.Printf("Node '%s' set to host '%s' with user '%s'\n", name, host, user)
			case "get":
				items := loadNodeInventory()
				if len(items) == 0 {
					fmt.Println("No SSH node inventory found.")
					return
				}
				var name string
				if len(args) > 1 {
					name = args[1]
				} else {
					keys := make([]string, 0, len(items))
					for k := range items {
						keys = append(keys, k)
					}
					prompt := promptui.Select{Label: "Select node", Items: keys}
					_, name, _ = prompt.Run()
				}
				entry, ok := items[name]
				if !ok {
					fmt.Println("Node not found.")
					return
				}
				fmt.Printf("%s: host=%s, type=%s, port=%d, user=%s\n", entry.Name, entry.Host, entry.Type, entry.Port, entry.User)
			case "list":
				items := loadNodeInventory()
				if len(items) == 0 {
					fmt.Println("No SSH node inventory found.")
					return
				}
				fmt.Println("Available SSH nodes:")
				for _, entry := range items {
					fmt.Printf("- %s: host=%s, type=%s, port=%d, user=%s\n", entry.Name, entry.Host, entry.Type, entry.Port, entry.User)
				}
			}
			return
		}
		// Not a command, treat as node name
		name := args[0]
		items := loadNodeInventory()
		entry, ok := items[name]
		if !ok {
			fmt.Println("Node or command not found.")
			return
		}
		user := entry.User
		if user == "" {
			user = "ubuntu"
		}
		sshArgs := []string{}
		if entry.Port != 0 {
			sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", user, entry.Host), "-p", fmt.Sprintf("%d", entry.Port))
		} else {
			sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", user, entry.Host))
		}
		if tunnelTarget != "" {
			sshArgs = append([]string{"-L", tunnelTarget}, sshArgs...)
		}
		sshExec := exec.Command("ssh", sshArgs...)
		sshExec.Stdin = cmd.InOrStdin()
		sshExec.Stdout = cmd.OutOrStdout()
		sshExec.Stderr = cmd.ErrOrStderr()
		err := sshExec.Run()
		if err != nil {
			fmt.Println("SSH exited with error:", err)
		}
	},
}

var tunnelTarget string

func init() {
	sshCmd.Flags().StringVar(&tunnelTarget, "tunnel", "", "Tunnel in format localPort:remoteHost:remotePort (optional)")
	rootCmd.AddCommand(sshCmd)
}
