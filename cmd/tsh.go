package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"sort"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// tshCmd represents the tsh command (Teleport SSH)
var tshCmd = &cobra.Command{
	Use:   "tsh",
	Short: "Connect to a VM using TSH (Teleport SSH)",
	Long:  `Connect to a VM instance using Teleport SSH, with automated node selection.`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		// Step 1: Ensure tsh login
		loginCmd := exec.Command("tsh", "status")
		if err := loginCmd.Run(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "You are not logged in to Teleport. Please run 'tsh login' first.")
			return
		}

		// Step 2: Get nodes list in JSON
		lsCmd := exec.Command("tsh", "ls", "--format=json")
		var out bytes.Buffer
		lsCmd.Stdout = &out
		if err := lsCmd.Run(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to list nodes with 'tsh ls'. Is tsh installed and configured?")
			return
		}

		// Step 3: Parse JSON nodes and labels
		type Node struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Spec struct {
				Hostname string `json:"hostname"`
			} `json:"spec"`
		}
		var nodes []Node
		if err := json.Unmarshal(out.Bytes(), &nodes); err != nil || len(nodes) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to parse tsh ls output.")
			return
		}

		// Step 4: Wizard for label pair selection (app_namespace + environment)
		type labelPair struct {
			AppNamespace string
			Environment  string
		}
		pairSet := map[labelPair]struct{}{}
		pairToNodes := map[labelPair][]Node{}
		for _, n := range nodes {
			appns := n.Metadata.Labels["app_namespace"]
			env := n.Metadata.Labels["environment"]
			pair := labelPair{AppNamespace: appns, Environment: env}
			pairSet[pair] = struct{}{}
			pairToNodes[pair] = append(pairToNodes[pair], n)
		}
		pairs := make([]labelPair, 0, len(pairSet))
		for p := range pairSet {
			pairs = append(pairs, p)
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].AppNamespace == pairs[j].AppNamespace {
				return pairs[i].Environment < pairs[j].Environment
			}
			return pairs[i].AppNamespace < pairs[j].AppNamespace
		})
		pairLabels := make([]string, len(pairs))
		for i, p := range pairs {
			pairLabels[i] = fmt.Sprintf("%s | %s", p.AppNamespace, p.Environment)
		}
		prompt := promptui.Select{
			Label: "Select app_namespace | environment",
			Items: pairLabels,
		}
		_, pairLabel, err := prompt.Run()
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Prompt failed:", err)
			return
		}
		selectedPair := pairs[0]
		for i, lbl := range pairLabels {
			if lbl == pairLabel {
				selectedPair = pairs[i]
				break
			}
		}
		filtered := pairToNodes[selectedPair]
		if len(filtered) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No nodes found with that label pair.")
			return
		}

		// Step 5: Select node by spec.hostname ONLY
		hostToNode := map[string]Node{}
		hostnames := make([]string, 0, len(filtered))
		for _, n := range filtered {
			host := n.Spec.Hostname
			if host == "" {
				continue // skip nodes without a hostname
			}
			hostToNode[host] = n
			hostnames = append(hostnames, host)
		}
		if len(hostnames) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No nodes with a valid hostname found.")
			return
		}
		sort.Strings(hostnames)
		prompt = promptui.Select{
			Label: "Select node (hostname)",
			Items: hostnames,
		}
		_, hostname, err := prompt.Run()
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Prompt failed:", err)
			return
		}

		if withDb == "__INTERACTIVE__" {
			withDb = ""
		}
		if withDb != "" || cmd.Flags().Changed("with-db") {
			// Use hierarchical inventory for DB entries
			hi, err := getHierarchicalInventory()
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Failed to initialize inventory:", err)
				return
			}

			var dbKey, dbHost string
			if withDb != "" {
				dbKey = withDb
				// Query the hierarchical inventory for the DB entry
				result, err := hi.Query(fmt.Sprintf("db.%s", dbKey))
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "DB key not found in inventory.")
					return
				}
				// Handle different value types - could be a string or object with host field
				switch v := result.(type) {
				case string:
					dbHost = v
				case map[string]interface{}:
					if host, ok := v["host"].(string); ok {
						dbHost = host
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), "DB entry missing host field.")
						return
					}
				default:
					fmt.Fprintln(cmd.OutOrStdout(), "Invalid DB entry format.")
					return
				}
			} else {
				// Interactive selection - get all DB keys
				dbKeys, err := hi.List("db")
				if err != nil || len(dbKeys) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No DB inventory found.")
					return
				}
				prompt := promptui.Select{Label: "Select DB key for tunnel", Items: dbKeys}
				_, dbKey, err = prompt.Run()
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "Prompt failed:", err)
					return
				}
				// Query the selected DB entry
				result, err := hi.Query(fmt.Sprintf("db.%s", dbKey))
				if err != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "Failed to get DB entry:", err)
					return
				}
				// Handle different value types
				switch v := result.(type) {
				case string:
					dbHost = v
				case map[string]interface{}:
					if host, ok := v["host"].(string); ok {
						dbHost = host
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), "DB entry missing host field.")
						return
					}
				default:
					fmt.Fprintln(cmd.OutOrStdout(), "Invalid DB entry format.")
					return
				}
			}
			// Find available local port (start at 5432, skip if in use)
			localPort := 5432
			for ; localPort < 5500; localPort++ {
				ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
				if err == nil {
					ln.Close()
					break
				}
			}
			if localPort >= 5500 {
				fmt.Fprintln(cmd.OutOrStdout(), "No available local port found for tunnel.")
				return
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Forwarding local port %d to %s:5432\n", localPort, dbHost)
			sshCmd := exec.Command("tsh", "ssh", "-L", fmt.Sprintf("127.0.0.1:%d:%s:5432", localPort, dbHost), fmt.Sprintf("ubuntu@%s", hostname))
			sshCmd.Stdin = cmd.InOrStdin()
			sshCmd.Stdout = cmd.OutOrStdout()
			sshCmd.Stderr = cmd.ErrOrStderr()
			err = sshCmd.Run()
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
				// Suppress status 130 (SIGINT/Ctrl+C)
				return
			}
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "SSH tunnel exited with error:", err)
			}
			return
		}
		sshCmd := exec.Command("tsh", "ssh", fmt.Sprintf("ubuntu@%s", hostname))
		sshCmd.Stdin = cmd.InOrStdin()
		sshCmd.Stdout = cmd.OutOrStdout()
		sshCmd.Stderr = cmd.ErrOrStderr()
		err = sshCmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			// Suppress status 130 (SIGINT/Ctrl+C)
			return
		}
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "SSH exited with error:", err)
		}
	},
}

var withDb string

func init() {
	tshCmd.Flags().StringVar(&withDb, "with-db", "", "Tunnel to DB key from inventory (interactive if empty)")
	tshCmd.Flags().Lookup("with-db").NoOptDefVal = "__INTERACTIVE__"
	rootCmd.AddCommand(tshCmd)
}
