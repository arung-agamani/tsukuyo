package cmd

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupIsolatedInventory creates a temporary directory for inventory tests
// and overrides the getDataDir function to point to the temp directory.
func setupIsolatedInventory(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "tsukuyo-test-inventory-")
	assert.NoError(t, err)

	// Override the function that determines the data directory
	originalGetDataDir := getDataDir
	getDataDir = func() string {
		return tmpDir
	}

	// Reset the global inventory cache to force using the new directory
	// We can't copy sync.Once, so we just reset the cache to nil
	originalCache := globalInventoryCache
	globalInventoryCache = nil

	// Return a cleanup function to be called via defer
	cleanup := func() {
		getDataDir = originalGetDataDir
		globalInventoryCache = originalCache
		// Reset the once to allow it to be called again for the original cache
		inventoryCacheOnce = sync.Once{}
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func setupStructuredDbData(t *testing.T) func() {
	t.Helper()

	_, cleanup := setupIsolatedInventory(t)

	hi, err := getHierarchicalInventory()
	if err != nil {
		cleanup()
		t.Fatalf("Failed to get hierarchical inventory: %v", err)
	}

	// Set up structured test data
	dbEntries := map[string]DbInventoryEntry{
		"redis-prod": {
			Host:       "redis-prod.example.com",
			Type:       "redis",
			RemotePort: 6379,
			Tags:       []string{"prod", "cache"},
		},
		"mongo-dev": {
			Host:       "mongo-dev.internal",
			Type:       "mongodb",
			RemotePort: 27017,
			LocalPort:  27018,
			Tags:       []string{"dev", "document"},
		},
		"postgres-staging": {
			Host:       "10.0.1.50",
			Type:       "postgres",
			RemotePort: 5432,
			Tags:       []string{"staging", "sql"},
		},
	}

	for name, entry := range dbEntries {
		err := hi.Set("db."+name, entry)
		if err != nil {
			cleanup()
			t.Fatalf("Failed to set structured db data for %s: %v", name, err)
		}
	}

	return cleanup
}

func TestInventoryDbCommand_StructuredData(t *testing.T) {
	cleanup := setupStructuredDbData(t)
	defer cleanup()

	tests := []struct {
		name        string
		args        []string
		contains    []string
		notcontains []string
	}{
		{
			name:     "list structured dbs",
			args:     []string{"db", "list"},
			contains: []string{"redis-prod", "mongo-dev", "postgres-staging"},
		},
		{
			name:     "get structured db entry",
			args:     []string{"db", "get", "mongo-dev"},
			contains: []string{"mongo-dev.internal", "mongodb", "27017", "27018", "[dev document]"},
		},
		{
			name:     "get non-existent entry",
			args:     []string{"db", "get", "non-existent"},
			contains: []string{"not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We use simpleCommandTest from inventory_test.go
			output, err := simpleCommandTest(t, tt.args)
			if err != nil {
				// some commands are expected to fail
				if !strings.Contains(err.Error(), "not found") && !strings.Contains(output, "not found") {
					t.Errorf("Command failed unexpectedly: %v, output: %s", err, output)
				}
			}

			for _, c := range tt.contains {
				if !strings.Contains(output, c) {
					t.Errorf("Output for '%s' did not contain '%s'.\nGot: %s", tt.name, c, output)
				}
			}
			for _, nc := range tt.notcontains {
				if strings.Contains(output, nc) {
					t.Errorf("Output for '%s' should not contain '%s'.\nGot: %s", tt.name, nc, output)
				}
			}
		})
	}
}

// Test the new DB initialization and enhanced set command functionality
func TestDbInventoryInitialization(t *testing.T) {
	_, cleanup := setupIsolatedInventory(t)
	defer cleanup()

	hi, err := getHierarchicalInventory()
	assert.NoError(t, err)

	// Test that ensureDbInventoryInitialized creates the db key
	err = ensureDbInventoryInitialized(hi)
	assert.NoError(t, err)

	// Verify db key exists and is a map
	result, err := hi.Query("db")
	assert.NoError(t, err)
	_, ok := result.(map[string]interface{})
	assert.True(t, ok, "db key should be a map")
}

func TestDbInventorySetWithFlags(t *testing.T) {
	_, cleanup := setupIsolatedInventory(t)
	defer cleanup()

	// Test setting flags
	dbSetType = "mongodb"
	dbSetRemotePort = 27017
	dbSetLocalPort = 27018
	dbSetTags = "dev,nosql"

	// Reset flags after test
	defer func() {
		dbSetType = ""
		dbSetRemotePort = 0
		dbSetLocalPort = 0
		dbSetTags = ""
	}()

	hi, err := getHierarchicalInventory()
	assert.NoError(t, err)

	// Test handleDbSet with arguments (non-interactive)
	err = handleDbSet(rootCmd, hi, []string{"test-mongo", "mongo.example.com"})
	assert.NoError(t, err)

	// Verify the entry was created correctly
	result, err := hi.Query("db.test-mongo")
	assert.NoError(t, err)

	// The result should be a DbInventoryEntry struct
	entry, ok := result.(DbInventoryEntry)
	assert.True(t, ok, "DB entry should be a DbInventoryEntry struct")

	assert.Equal(t, "mongo.example.com", entry.Host)
	assert.Equal(t, "mongodb", entry.Type)
	assert.Equal(t, 27017, entry.RemotePort)
	assert.Equal(t, 27018, entry.LocalPort)

	// Verify tags
	assert.Len(t, entry.Tags, 2)
	assert.Equal(t, "dev", entry.Tags[0])
	assert.Equal(t, "nosql", entry.Tags[1])
}

func TestDbInventorySetDefaults(t *testing.T) {
	_, cleanup := setupIsolatedInventory(t)
	defer cleanup()

	hi, err := getHierarchicalInventory()
	assert.NoError(t, err)

	// Test handleDbSet with only name and host (should use defaults)
	err = handleDbSet(rootCmd, hi, []string{"test-pg", "postgres.example.com"})
	assert.NoError(t, err)

	// Verify the entry was created with defaults
	result, err := hi.Query("db.test-pg")
	assert.NoError(t, err)

	entry, ok := result.(DbInventoryEntry)
	assert.True(t, ok, "DB entry should be a DbInventoryEntry struct")

	assert.Equal(t, "postgres.example.com", entry.Host)
	assert.Equal(t, "postgres", entry.Type) // default
	assert.Equal(t, 5432, entry.RemotePort) // default
	assert.Equal(t, 0, entry.LocalPort)     // default (0)
}

func TestValidateDbEntry(t *testing.T) {
	tests := []struct {
		name        string
		entry       interface{}
		expectError bool
	}{
		{
			name: "valid entry",
			entry: map[string]interface{}{
				"host":        "test.com",
				"type":        "postgres",
				"remote_port": float64(5432),
				"local_port":  float64(5433),
				"tags":        []interface{}{"dev", "test"},
			},
			expectError: false,
		},
		{
			name:        "not a map",
			entry:       "invalid",
			expectError: true,
		},
		{
			name: "missing host",
			entry: map[string]interface{}{
				"type":        "postgres",
				"remote_port": float64(5432),
			},
			expectError: true,
		},
		{
			name: "missing type",
			entry: map[string]interface{}{
				"host":        "test.com",
				"remote_port": float64(5432),
			},
			expectError: true,
		},
		{
			name: "missing remote_port",
			entry: map[string]interface{}{
				"host": "test.com",
				"type": "postgres",
			},
			expectError: true,
		},
		{
			name: "invalid host type",
			entry: map[string]interface{}{
				"host":        123,
				"type":        "postgres",
				"remote_port": float64(5432),
			},
			expectError: true,
		},
		{
			name: "invalid tags type",
			entry: map[string]interface{}{
				"host":        "test.com",
				"type":        "postgres",
				"remote_port": float64(5432),
				"tags":        "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDbEntry("test", tt.entry)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDebugDbStorage(t *testing.T) {
	_, cleanup := setupIsolatedInventory(t)
	defer cleanup()

	hi, err := getHierarchicalInventory()
	assert.NoError(t, err)

	// Create a test entry directly
	entry := DbInventoryEntry{
		Host:       "test.com",
		Type:       "postgres",
		RemotePort: 5432,
		LocalPort:  5433,
		Tags:       []string{"test"},
	}

	err = hi.Set("db.test", entry)
	assert.NoError(t, err)

	// Query it back and see what we get
	result, err := hi.Query("db.test")
	assert.NoError(t, err)

	t.Logf("Result type: %T", result)
	t.Logf("Result value: %+v", result)

	// Also check the whole db structure
	dbResult, err := hi.Query("db")
	assert.NoError(t, err)
	t.Logf("DB structure type: %T", dbResult)
	t.Logf("DB structure value: %+v", dbResult)
}

func TestDbInventoryRecovery(t *testing.T) {
	_, cleanup := setupIsolatedInventory(t)
	defer cleanup()

	hi, err := getHierarchicalInventory()
	assert.NoError(t, err)

	// Create a test entry first
	err = handleDbSet(rootCmd, hi, []string{"test-entry", "test.example.com"})
	assert.NoError(t, err)

	// Verify it exists
	result, err := hi.Query("db.test-entry")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test 1: Delete the entire db key
	err = hi.Delete("db")
	assert.NoError(t, err)

	// Verify db key is gone
	_, err = hi.Query("db")
	assert.Error(t, err)

	// Now use a db command which should trigger recovery
	err = handleTypeList(rootCmd, hi, "db")
	assert.NoError(t, err)

	// Verify db key is recreated as empty map
	result, err = hi.Query("db")
	assert.NoError(t, err)
	dbMap, ok := result.(map[string]interface{})
	assert.True(t, ok, "db key should be a map after recovery")
	assert.Empty(t, dbMap, "db should be empty after recovery")

	// Test 2: Set db to invalid type (string)
	err = hi.Set("db", "this is not a map")
	assert.NoError(t, err)

	// Verify it's set to string
	result, err = hi.Query("db")
	assert.NoError(t, err)
	_, ok = result.(string)
	assert.True(t, ok, "db should be a string before recovery")

	// Use a db command which should trigger recovery
	err = handleTypeList(rootCmd, hi, "db")
	assert.NoError(t, err)

	// Verify db key is fixed
	result, err = hi.Query("db")
	assert.NoError(t, err)
	dbMap, ok = result.(map[string]interface{})
	assert.True(t, ok, "db key should be a map after recovery from invalid type")
	assert.Empty(t, dbMap, "db should be empty after recovery from invalid type")
}
