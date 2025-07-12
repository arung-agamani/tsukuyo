package cmd

import (
	"strings"
	"testing"
)

func setupStructuredDbData(t *testing.T) {
	t.Helper()
	hi, err := getHierarchicalInventory()
	if err != nil {
		t.Fatalf("Failed to get hierarchical inventory: %v", err)
	}

	// Clean up before setting new data
	keys, err := hi.List("db")
	if err == nil {
		for _, key := range keys {
			hi.Delete("db." + key)
		}
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
			t.Fatalf("Failed to set structured db data for %s: %v", name, err)
		}
	}
}

func TestInventoryDbCommand_StructuredData(t *testing.T) {
	setupStructuredDbData(t)

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
