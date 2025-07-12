package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const (
	tsukuyoDirName   = ".tsukuyo"
	scriptsDirName   = "scripts"
	scriptMetaSuffix = ".meta.json"
)

type ScriptMeta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

var getTsukuyoDir = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, tsukuyoDirName)
}

func getScriptsDir() string {
	return filepath.Join(getTsukuyoDir(), scriptsDirName)
}

func ensureScriptDirs() error {
	return os.MkdirAll(getScriptsDir(), 0755)
}

func sanitizeScriptName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, " ", "_"), "/", "_")
}

func scriptFilePath(name string) string {
	return filepath.Join(getScriptsDir(), sanitizeScriptName(name))
}

func scriptMetaPath(name string) string {
	return scriptFilePath(name) + scriptMetaSuffix
}

// Add script subcommands
var scriptAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new script",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureScriptDirs(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to create scripts dir:", err)
			return
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprint(cmd.OutOrStdout(), "Script name: ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "Name required.")
			return
		}
		fmt.Fprint(cmd.OutOrStdout(), "Description: ")
		desc, _ := reader.ReadString('\n')
		desc = strings.TrimSpace(desc)
		fmt.Fprint(cmd.OutOrStdout(), "Tags (comma separated): ")
		tagsStr, _ := reader.ReadString('\n')
		tags := strings.Split(strings.TrimSpace(tagsStr), ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Enter script content (end with EOF/Ctrl+D):")
		var content strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			content.WriteString(line)
		}
		if err := os.WriteFile(scriptFilePath(name), []byte(content.String()), 0755); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to write script:", err)
			return
		}
		meta := ScriptMeta{Name: name, Description: desc, Tags: tags}
		metaBytes, _ := json.MarshalIndent(meta, "", "  ")
		_ = os.WriteFile(scriptMetaPath(name), metaBytes, 0644)
		fmt.Fprintln(cmd.OutOrStdout(), "Script added:", name)
	},
}

var scriptListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scripts",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureScriptDirs(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to access scripts dir:", err)
			return
		}
		entries, _ := os.ReadDir(getScriptsDir())
		scripts := []ScriptMeta{}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), scriptMetaSuffix) {
				metaBytes, _ := os.ReadFile(filepath.Join(getScriptsDir(), e.Name()))
				var meta ScriptMeta
				_ = json.Unmarshal(metaBytes, &meta)
				scripts = append(scripts, meta)
			}
		}
		sort.Slice(scripts, func(i, j int) bool { return scripts[i].Name < scripts[j].Name })
		fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-40s %-20s\n", "NAME", "DESCRIPTION", "TAGS")
		for _, s := range scripts {
			fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-40s %-20s\n", s.Name, s.Description, strings.Join(s.Tags, ", "))
		}
	},
}

var (
	runWithEnvFile string
	runEdit        bool
	runDryRun      bool
)

var scriptRunCmd = &cobra.Command{
	Use:   "run [script name]",
	Short: "Run a script",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureScriptDirs(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to access scripts dir:", err)
			return
		}
		name := args[0]
		scriptPath := scriptFilePath(name)
		metaPath := scriptMetaPath(name)
		if _, err := os.Stat(scriptPath); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Script not found:", name)
			return
		}
		if runEdit {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			c := exec.Command(editor, scriptPath)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			_ = c.Run()
			return
		}
		content, _ := os.ReadFile(scriptPath)
		var envs map[string]string
		if runWithEnvFile != "" {
			envs = loadEnvFile(runWithEnvFile)
		}
		if runDryRun {
			fmt.Fprintln(cmd.OutOrStdout(), "--- DRY RUN ---")
			if metaBytes, err := os.ReadFile(metaPath); err == nil {
				var meta ScriptMeta
				_ = json.Unmarshal(metaBytes, &meta)
				fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\nDescription: %s\nTags: %s\n", meta.Name, meta.Description, strings.Join(meta.Tags, ", "))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Env Vars:")
			for k, v := range envs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", k, v)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Script Content:")
			fmt.Fprintln(cmd.OutOrStdout(), string(content))
			return
		}
		cmdExec := exec.Command("/bin/bash", scriptPath)
		cmdExec.Stdin = os.Stdin
		cmdExec.Stdout = os.Stdout
		cmdExec.Stderr = os.Stderr
		for k, v := range envs {
			cmdExec.Env = append(cmdExec.Env, fmt.Sprintf("%s=%s", k, v))
		}
		err := cmdExec.Run()
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Script exited with error:", err)
		}
	},
}

func loadEnvFile(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		// Note: This function is called from Cobra commands but doesn't have access to cmd
		// For now keeping fmt.Println since it's a utility function
		fmt.Println("Failed to open env file:", err)
		return nil
	}
	defer f.Close()
	envs := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if idx := strings.Index(value, " #"); idx != -1 {
				value = strings.TrimSpace(value[:idx])
			}
			envs[key] = value
		}
	}
	return envs
}

var scriptEditCmd = &cobra.Command{
	Use:   "edit [script name]",
	Short: "Edit a script",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureScriptDirs(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to access scripts dir:", err)
			return
		}
		name := args[0]
		scriptPath := scriptFilePath(name)
		if _, err := os.Stat(scriptPath); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Script not found:", name)
			return
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		c := exec.Command(editor, scriptPath)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		_ = c.Run()
	},
}

var scriptDeleteCmd = &cobra.Command{
	Use:   "delete [script name]",
	Short: "Delete a script",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureScriptDirs(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to access scripts dir:", err)
			return
		}
		name := args[0]
		if err := os.Remove(scriptFilePath(name)); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to delete script:", err)
		}
		_ = os.Remove(scriptMetaPath(name))
		fmt.Fprintln(cmd.OutOrStdout(), "Deleted script:", name)
	},
}

var scriptSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search scripts by name, tag, or description",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureScriptDirs(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Failed to access scripts dir:", err)
			return
		}
		query := strings.ToLower(strings.Join(args, " "))
		entries, _ := os.ReadDir(getScriptsDir())
		scripts := []ScriptMeta{}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), scriptMetaSuffix) {
				metaBytes, _ := os.ReadFile(filepath.Join(getScriptsDir(), e.Name()))
				var meta ScriptMeta
				_ = json.Unmarshal(metaBytes, &meta)
				if strings.Contains(strings.ToLower(meta.Name), query) ||
					strings.Contains(strings.ToLower(meta.Description), query) ||
					containsTag(meta.Tags, query) {
					scripts = append(scripts, meta)
				}
			}
		}
		if len(scripts) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No scripts found matching query.")
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-40s %-20s\n", "NAME", "DESCRIPTION", "TAGS")
		for _, s := range scripts {
			fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-40s %-20s\n", s.Name, s.Description, strings.Join(s.Tags, ", "))
		}
	},
}

func containsTag(tags []string, query string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), query) {
			return true
		}
	}
	return false
}

// scriptCmd represents the script command
var scriptCmd = &cobra.Command{
	Use:   "script",
	Short: "Manage and execute script inventory",
	Long:  `Conveniently execute, view, and edit predefined scripts (bash for now, later node/deno/python).`,
}

func init() {
	scriptRunCmd.Flags().StringVar(&runWithEnvFile, "with-env-file", "", "Path to env file")
	scriptRunCmd.Flags().BoolVar(&runEdit, "edit", false, "Edit script before running")
	scriptRunCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Show env and script content without executing")

	scriptCmd.AddCommand(scriptAddCmd)
	scriptCmd.AddCommand(scriptListCmd)
	scriptCmd.AddCommand(scriptRunCmd)
	scriptCmd.AddCommand(scriptEditCmd)
	scriptCmd.AddCommand(scriptDeleteCmd)
	scriptCmd.AddCommand(scriptSearchCmd)

	rootCmd.AddCommand(scriptCmd)
}
