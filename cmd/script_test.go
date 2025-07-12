package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// tempScript represents a script to be created in the temporary test environment.
type tempScript struct {
	Meta    ScriptMeta
	Content string
}

// setupTestScripts creates a temporary directory for scripts and populates it.
// It overrides the getTsukuyoDir function to point to the temp directory.
func setupTestScripts(t *testing.T, scripts []tempScript) (string, func()) {
	tmpDir, err := ioutil.TempDir("", "tsukuyo-test-scripts-")
	assert.NoError(t, err)

	// Override the function that determines the .tsukuyo directory.
	originalGetTsukuyoDir := getTsukuyoDir
	getTsukuyoDir = func() string {
		return tmpDir
	}

	scriptsDir := getScriptsDir()
	err = os.MkdirAll(scriptsDir, 0755)
	assert.NoError(t, err)

	for _, s := range scripts {
		// Write script file
		scriptPath := scriptFilePath(s.Meta.Name)
		err := ioutil.WriteFile(scriptPath, []byte(s.Content), 0755)
		assert.NoError(t, err)

		// Write metadata file
		metaPath := scriptMetaPath(s.Meta.Name)
		metaBytes, err := json.MarshalIndent(s.Meta, "", "  ")
		assert.NoError(t, err)
		err = ioutil.WriteFile(metaPath, metaBytes, 0644)
		assert.NoError(t, err)
	}

	// Return a cleanup function to be called via defer.
	cleanup := func() {
		getTsukuyoDir = originalGetTsukuyoDir
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestScriptListCmd(t *testing.T) {
	scriptsToCreate := []tempScript{
		{
			Meta:    ScriptMeta{Name: "hello-world", Description: "Prints hello world", Tags: []string{"test", "example"}},
			Content: `#!/bin/bash
echo "Hello World"`,
		},
		{
			Meta:    ScriptMeta{Name: "another-script", Description: "Another one", Tags: []string{"test", "demo"}},
			Content: `#!/bin/bash
echo "Another one"`,
		},
	}
	_, cleanup := setupTestScripts(t, scriptsToCreate)
	defer cleanup()

	output, err := executeCommand(rootCmd, "script", "list")
	assert.NoError(t, err)

	// Check for table headers
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "DESCRIPTION")
	assert.Contains(t, output, "TAGS")

	// Check for script details
	assert.Contains(t, output, "hello-world")
	assert.Contains(t, output, "Prints hello world")
	assert.Contains(t, output, "test, example")

	assert.Contains(t, output, "another-script")
	assert.Contains(t, output, "Another one")
	assert.Contains(t, output, "test, demo")
}

func TestScriptListEmpty(t *testing.T) {
	_, cleanup := setupTestScripts(t, []tempScript{}) // No scripts
	defer cleanup()

	output, err := executeCommand(rootCmd, "script", "list")
	assert.NoError(t, err)

	assert.Contains(t, output, "NAME")
	assert.NotContains(t, output, "hello-world")
}

func TestScriptAddCmd(t *testing.T) {
	_, cleanup := setupTestScripts(t, []tempScript{})
	defer cleanup()

	// Mock user input
	input := "new-script\nA cool new script\ntest,new\n#!/bin/bash\necho 'new'\n"
	r, w, _ := os.Pipe()
	w.Write([]byte(input))
	w.Close()
	originalStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = originalStdin }()

	output, err := executeCommand(rootCmd, "script", "add")
	assert.NoError(t, err)
	assert.Contains(t, output, "Script added: new-script")

	// Verify the files were created
	scriptsDir := getScriptsDir()
	scriptPath := filepath.Join(scriptsDir, "new-script")
	metaPath := filepath.Join(scriptsDir, "new-script.meta.json")

	assert.FileExists(t, scriptPath)
	assert.FileExists(t, metaPath)

	// Verify content
	content, _ := ioutil.ReadFile(scriptPath)
	assert.Contains(t, string(content), "echo 'new'")

	metaBytes, _ := ioutil.ReadFile(metaPath)
	var meta ScriptMeta
	json.Unmarshal(metaBytes, &meta)
	assert.Equal(t, "new-script", meta.Name)
	assert.Equal(t, "A cool new script", meta.Description)
	assert.Equal(t, []string{"test", "new"}, meta.Tags)
}

func TestScriptDeleteCmd(t *testing.T) {
	scriptsToCreate := []tempScript{
		{
			Meta:    ScriptMeta{Name: "to-delete", Description: "This will be deleted", Tags: []string{"delete"}},
			Content: "echo 'delete me'",
		},
	}
	_, cleanup := setupTestScripts(t, scriptsToCreate)
	defer cleanup()

	scriptPath := scriptFilePath("to-delete")
	metaPath := scriptMetaPath("to-delete")
	assert.FileExists(t, scriptPath)
	assert.FileExists(t, metaPath)

	output, err := executeCommand(rootCmd, "script", "delete", "to-delete")
	assert.NoError(t, err)
	assert.Contains(t, output, "Deleted script: to-delete")

	assert.NoFileExists(t, scriptPath)
	assert.NoFileExists(t, metaPath)
}

func TestScriptSearchCmd(t *testing.T) {
	scriptsToCreate := []tempScript{
		{
			Meta:    ScriptMeta{Name: "search-me", Description: "A searchable script", Tags: []string{"find", "main"}},
			Content: "echo 1",
		},
		{
			Meta:    ScriptMeta{Name: "another", Description: "Not the target", Tags: []string{"other"}},
			Content: "echo 2",
		},
		{
			Meta:    ScriptMeta{Name: "find-this", Description: "A script to find", Tags: []string{"search"}},
			Content: "echo 3",
		},
	}
	_, cleanup := setupTestScripts(t, scriptsToCreate)
	defer cleanup()

	// Search by name
	output, err := executeCommand(rootCmd, "script", "search", "search-me")
	assert.NoError(t, err)
	assert.Contains(t, output, "search-me")
	assert.NotContains(t, output, "another")
	assert.NotContains(t, output, "find-this")

	// Search by description
	output, err = executeCommand(rootCmd, "script", "search", "target")
	assert.NoError(t, err)
	assert.NotContains(t, output, "search-me")
	assert.Contains(t, output, "another") // "Not the target"
	assert.NotContains(t, output, "find-this")

	// Search by tag
	output, err = executeCommand(rootCmd, "script", "search", "find")
	assert.NoError(t, err)
	assert.Contains(t, output, "search-me")
	assert.NotContains(t, output, "another")
	assert.Contains(t, output, "find-this")

	// No results
	output, err = executeCommand(rootCmd, "script", "search", "nonexistent")
	assert.NoError(t, err)
	assert.Contains(t, output, "No scripts found matching query.")
}

func TestScriptRunCmd(t *testing.T) {
	scriptsToCreate := []tempScript{
		{
			Meta:    ScriptMeta{Name: "run-test", Description: "A runnable script", Tags: []string{"run"}},
			Content: `#!/bin/bash
echo "SCRIPT_VAR=${SCRIPT_VAR}"
`,
		},
	}
	tmpDir, cleanup := setupTestScripts(t, scriptsToCreate)
	defer cleanup()

	// Test dry run
	output, err := executeCommand(rootCmd, "script", "run", "--dry-run", "run-test")
	assert.NoError(t, err)
	assert.Contains(t, output, "--- DRY RUN ---")
	assert.Contains(t, output, "Name: run-test")
	assert.Contains(t, output, "Script Content:")
	assert.Contains(t, output, `echo "SCRIPT_VAR=${SCRIPT_VAR}"`)

	// Test dry run with env file
	envFilePath := filepath.Join(tmpDir, ".env")
	ioutil.WriteFile(envFilePath, []byte("SCRIPT_VAR=dry-run-test"), 0644)
	output, err = executeCommand(rootCmd, "script", "run", "--dry-run", "--with-env-file", envFilePath, "run-test")
	assert.NoError(t, err)
	assert.Contains(t, output, "Env Vars:")
	assert.Contains(t, output, "SCRIPT_VAR=dry-run-test")

	// Note: Actually running the script is harder to test in a unit test environment
	// because it involves a real `exec.Command`. The dry-run gives us good coverage
	// of the logic leading up to the execution.
}

func TestLoadEnvFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tsukuyo-test-env-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	envContent := `
# This is a comment
VAR1=value1
VAR2 = value2 # with comment
  VAR3= value3
INVALID_LINE
`
	envFilePath := filepath.Join(tmpDir, ".env")
	err = ioutil.WriteFile(envFilePath, []byte(envContent), 0644)
	assert.NoError(t, err)

	envs := loadEnvFile(envFilePath)

	assert.Equal(t, "value1", envs["VAR1"])
	assert.Equal(t, "value2", envs["VAR2"])
	assert.Equal(t, "value3", envs["VAR3"])
	_, exists := envs["INVALID_LINE"]
	assert.False(t, exists)
	_, exists = envs["# This is a comment"]
	assert.False(t, exists)
}

func TestSanitizeScriptName(t *testing.T) {
	assert.Equal(t, "my_script", sanitizeScriptName("my script"))
	assert.Equal(t, "my_script_name", sanitizeScriptName("my/script/name"))
	assert.Equal(t, "my_complex_script_name", sanitizeScriptName("my complex/script name"))
}

func TestContainsTag(t *testing.T) {
	tags := []string{"Go", "Test", "Example"}
	assert.True(t, containsTag(tags, "test"))
	assert.True(t, containsTag(tags, "go"))
	assert.False(t, containsTag(tags, "java"))
}
