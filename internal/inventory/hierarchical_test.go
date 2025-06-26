package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHierarchicalInventory_BasicQueries(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tsukuyo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hi, err := NewHierarchicalInventory(tempDir)
	if err != nil {
		t.Fatalf("Failed to create hierarchical inventory: %v", err)
	}

	// Test data from the directives
	testData := map[string]interface{}{
		"db": map[string]interface{}{
			"izuna-db": map[string]interface{}{
				"host": "kureya.howlingmoon.dev",
				"port": "2333",
				"user": "abcd",
				"pass": "pass",
			},
		},
	}

	// Set the test data
	hi.data = testData

	tests := []struct {
		name     string
		query    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:  "query root",
			query: "",
			expected: map[string]interface{}{
				"db": map[string]interface{}{
					"izuna-db": map[string]interface{}{
						"host": "kureya.howlingmoon.dev",
						"port": "2333",
						"user": "abcd",
						"pass": "pass",
					},
				},
			},
		},
		{
			name:  "query db",
			query: "db",
			expected: map[string]interface{}{
				"izuna-db": map[string]interface{}{
					"host": "kureya.howlingmoon.dev",
					"port": "2333",
					"user": "abcd",
					"pass": "pass",
				},
			},
		},
		{
			name:  "query db.izuna-db",
			query: "db.izuna-db",
			expected: map[string]interface{}{
				"host": "kureya.howlingmoon.dev",
				"port": "2333",
				"user": "abcd",
				"pass": "pass",
			},
		},
		{
			name:     "query db.izuna-db.port",
			query:    "db.izuna-db.port",
			expected: "2333",
		},
		{
			name:     "query db.izuna-db.host",
			query:    "db.izuna-db.host",
			expected: "kureya.howlingmoon.dev",
		},
		{
			name:    "query non-existent key",
			query:   "db.nonexistent",
			wantErr: true,
		},
		{
			name:    "query on non-object",
			query:   "db.izuna-db.port.something",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hi.Query(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Query() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHierarchicalInventory_ArrayQueries(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tsukuyo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hi, err := NewHierarchicalInventory(tempDir)
	if err != nil {
		t.Fatalf("Failed to create hierarchical inventory: %v", err)
	}

	// Test data with arrays from the directives
	testData := map[string]interface{}{
		"db": map[string]interface{}{
			"izuna-db": []interface{}{
				map[string]interface{}{
					"host": "kureya.howlingmoon.dev",
					"port": "2333",
					"user": "abcd",
					"pass": "pass",
					"env":  "int",
				},
				map[string]interface{}{
					"host": "kureya.howlingmoon.dev",
					"port": "2333",
					"user": "abcd",
					"pass": "pass",
					"env":  "prd",
				},
			},
		},
	}

	hi.data = testData

	tests := []struct {
		name     string
		query    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:  "query array by index [0]",
			query: "db.izuna-db.[0]",
			expected: map[string]interface{}{
				"host": "kureya.howlingmoon.dev",
				"port": "2333",
				"user": "abcd",
				"pass": "pass",
				"env":  "int",
			},
		},
		{
			name:     "query array item property [0].env",
			query:    "db.izuna-db.[0].env",
			expected: "int",
		},
		{
			name:     "query array item property [1].env",
			query:    "db.izuna-db.[1].env",
			expected: "prd",
		},
		{
			name:     "query wildcard [*].env",
			query:    "db.izuna-db.[*].env",
			expected: []interface{}{"int", "prd"},
		},
		{
			name:    "query array out of bounds",
			query:   "db.izuna-db.[5]",
			wantErr: true,
		},
		{
			name:    "query array on non-array",
			query:   "db.[0]",
			wantErr: true,
		},
		{
			name:    "query wildcard on non-array",
			query:   "db.[*]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hi.Query(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Query() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHierarchicalInventory_SetAndDelete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tsukuyo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hi, err := NewHierarchicalInventory(tempDir)
	if err != nil {
		t.Fatalf("Failed to create hierarchical inventory: %v", err)
	}

	// Test setting values
	err = hi.Set("servers.web.host", "nginx.example.com")
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	err = hi.Set("servers.web.port", 80)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	err = hi.Set("servers.db.host", "postgres.example.com")
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Test querying set values
	result, err := hi.Query("servers.web.host")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result != "nginx.example.com" {
		t.Errorf("Expected 'nginx.example.com', got %v", result)
	}

	result, err = hi.Query("servers.web.port")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result != 80 {
		t.Errorf("Expected 80, got %v", result)
	}

	// Test listing keys
	keys, err := hi.List("servers")
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}
	if len(keys) != 2 || !contains(keys, "web") || !contains(keys, "db") {
		t.Errorf("Expected keys [web, db], got %v", keys)
	}

	// Test deleting
	err = hi.Delete("servers.web.port")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify deletion
	_, err = hi.Query("servers.web.port")
	if err == nil {
		t.Error("Expected error when querying deleted key")
	}

	// Test that host still exists
	result, err = hi.Query("servers.web.host")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result != "nginx.example.com" {
		t.Errorf("Expected 'nginx.example.com', got %v", result)
	}
}

func TestHierarchicalInventory_DataPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tsukuyo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create first instance and set data
	hi1, err := NewHierarchicalInventory(tempDir)
	if err != nil {
		t.Fatalf("Failed to create hierarchical inventory: %v", err)
	}

	err = hi1.Set("config.version", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	err = hi1.Set("config.debug", true)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Create second instance and verify data was loaded
	hi2, err := NewHierarchicalInventory(tempDir)
	if err != nil {
		t.Fatalf("Failed to create hierarchical inventory: %v", err)
	}

	result, err := hi2.Query("config.version")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result != "1.0.0" {
		t.Errorf("Expected '1.0.0', got %v", result)
	}

	result, err = hi2.Query("config.debug")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestHierarchicalInventory_LoadFromMultipleFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tsukuyo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create legacy inventory files
	dbData := map[string]string{
		"main": "postgres.example.com",
		"test": "postgres-test.example.com",
	}
	dbJSON, _ := json.Marshal(dbData)
	err = os.WriteFile(filepath.Join(tempDir, "db-inventory.json"), dbJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write db inventory: %v", err)
	}

	nodeData := map[string]interface{}{
		"web1": map[string]interface{}{
			"host": "web1.example.com",
			"port": 22,
			"user": "admin",
		},
	}
	nodeJSON, _ := json.Marshal(nodeData)
	err = os.WriteFile(filepath.Join(tempDir, "node-inventory.json"), nodeJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write node inventory: %v", err)
	}

	// Create hierarchical inventory and verify it loads the legacy files
	hi, err := NewHierarchicalInventory(tempDir)
	if err != nil {
		t.Fatalf("Failed to create hierarchical inventory: %v", err)
	}

	// Test db data
	result, err := hi.Query("db.main")
	if err != nil {
		t.Fatalf("Failed to query db: %v", err)
	}
	if result != "postgres.example.com" {
		t.Errorf("Expected 'postgres.example.com', got %v", result)
	}

	// Test node data
	result, err = hi.Query("node.web1.host")
	if err != nil {
		t.Fatalf("Failed to query node: %v", err)
	}
	if result != "web1.example.com" {
		t.Errorf("Expected 'web1.example.com', got %v", result)
	}
}

func TestHierarchicalInventory_ComplexQueries(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tsukuyo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hi, err := NewHierarchicalInventory(tempDir)
	if err != nil {
		t.Fatalf("Failed to create hierarchical inventory: %v", err)
	}

	// Complex nested data structure
	testData := map[string]interface{}{
		"environments": map[string]interface{}{
			"production": map[string]interface{}{
				"servers": []interface{}{
					map[string]interface{}{
						"name": "web-prod-1",
						"host": "10.0.1.10",
						"role": "web",
					},
					map[string]interface{}{
						"name": "web-prod-2",
						"host": "10.0.1.11",
						"role": "web",
					},
					map[string]interface{}{
						"name": "db-prod-1",
						"host": "10.0.1.20",
						"role": "database",
					},
				},
				"config": map[string]interface{}{
					"debug":   false,
					"workers": 8,
				},
			},
			"staging": map[string]interface{}{
				"servers": []interface{}{
					map[string]interface{}{
						"name": "web-stage-1",
						"host": "10.0.2.10",
						"role": "web",
					},
				},
				"config": map[string]interface{}{
					"debug":   true,
					"workers": 2,
				},
			},
		},
	}

	hi.data = testData

	tests := []struct {
		name     string
		query    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "get all server names in production",
			query:    "environments.production.servers.[*].name",
			expected: []interface{}{"web-prod-1", "web-prod-2", "db-prod-1"},
		},
		{
			name:     "get all server hosts in production",
			query:    "environments.production.servers.[*].host",
			expected: []interface{}{"10.0.1.10", "10.0.1.11", "10.0.1.20"},
		},
		{
			name:     "get second server in production",
			query:    "environments.production.servers.[1].name",
			expected: "web-prod-2",
		},
		{
			name:     "get production workers config",
			query:    "environments.production.config.workers",
			expected: 8,
		},
		{
			name:     "get staging debug config",
			query:    "environments.staging.config.debug",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hi.Query(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Query() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHierarchicalInventory_EdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tsukuyo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hi, err := NewHierarchicalInventory(tempDir)
	if err != nil {
		t.Fatalf("Failed to create hierarchical inventory: %v", err)
	}

	// Test setting root-level key
	err = hi.Set("rootkey", "rootvalue")
	if err != nil {
		t.Fatalf("Failed to set root key: %v", err)
	}

	result, err := hi.Query("rootkey")
	if err != nil {
		t.Fatalf("Failed to query root key: %v", err)
	}
	if result != "rootvalue" {
		t.Errorf("Expected 'rootvalue', got %v", result)
	}

	// Test deleting root-level key
	err = hi.Delete("rootkey")
	if err != nil {
		t.Fatalf("Failed to delete root key: %v", err)
	}

	_, err = hi.Query("rootkey")
	if err == nil {
		t.Error("Expected error when querying deleted root key")
	}

	// Test setting complex nested structures
	complexValue := map[string]interface{}{
		"nested": map[string]interface{}{
			"array": []interface{}{1, 2, 3},
			"bool":  true,
		},
	}
	err = hi.Set("complex", complexValue)
	if err != nil {
		t.Fatalf("Failed to set complex value: %v", err)
	}

	result, err = hi.Query("complex.nested.array.[1]")
	if err != nil {
		t.Fatalf("Failed to query complex nested value: %v", err)
	}
	if result != 2 {
		t.Errorf("Expected 2, got %v", result)
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
