package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/arung-agamani/tsukuyo/internal/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// TshNode defines the structure for a Teleport node.
type TshNode struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Hostname string `json:"hostname"`
	} `json:"spec"`
}

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
		var nodes []TshNode
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
		pairToNodes := map[labelPair][]TshNode{}
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
		hostToNode := map[string]TshNode{}
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
		selectedNode := hostToNode[hostname]

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

			dbEntry, err := selectDbWithTaggingForTsh(hi, selectedNode)
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), err)
				return
			}

			localPort := dbEntry.LocalPort
			if localPort == 0 {
				localPort = dbEntry.RemotePort // Default to same as remote
			}
			tunnel := fmt.Sprintf("%d:%s:%d", localPort, dbEntry.Host, dbEntry.RemotePort)

			fmt.Fprintf(cmd.OutOrStdout(), "Forwarding local port %d to %s:%d\n", localPort, dbEntry.Host, dbEntry.RemotePort)
			sshCmd := exec.Command("tsh", "ssh", "-L", tunnel, fmt.Sprintf("ubuntu@%s", hostname))
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

func selectDbWithTaggingForTsh(hi *inventory.HierarchicalInventory, node TshNode) (*DbInventoryEntry, error) {
	dbEntries, err := hi.List("db")
	if err != nil || len(dbEntries) == 0 {
		return nil, fmt.Errorf("no DB inventory found")
	}

	nodeTags := getTshNodeTags(node)
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

func getTshNodeTags(node TshNode) []string {
	var tags []string
	for _, value := range node.Metadata.Labels {
		tags = append(tags, value)
	}
	// also add the node name to the tags
	tags = append(tags, node.Metadata.Name)
	return tags
}
