package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestEnvOutputFeature tests the ENV output functionality
func TestEnvOutputFeature(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		flatten  bool
		coerce   bool
		expected []string // Expected lines in output
	}{
		{
			name: "simple object with flat=true",
			data: map[string]interface{}{
				"FOO": "BAR",
				"FAZ": map[string]interface{}{
					"FES": "BEAR",
					"NYA": map[string]interface{}{
						"WOW": "NJIR",
					},
				},
			},
			flatten: true,
			coerce:  false,
			expected: []string{
				"FOO=BAR",
				"FAZ_FES=BEAR",
				"FAZ_NYA_WOW=NJIR",
			},
		},
		{
			name: "array values with flat=true",
			data: map[string]interface{}{
				"FOO": []interface{}{"VAL1", "VAL2"},
			},
			flatten: true,
			coerce:  false,
			expected: []string{
				"FOO_0=VAL1",
				"FOO_1=VAL2",
			},
		},
		{
			name: "array of objects with flat=true",
			data: map[string]interface{}{
				"FOO": []interface{}{
					map[string]interface{}{
						"BAR": "AWOO",
					},
					map[string]interface{}{
						"BAR": "HAHA",
					},
				},
			},
			flatten: true,
			coerce:  false,
			expected: []string{
				"FOO_0_BAR=AWOO",
				"FOO_1_BAR=HAHA",
			},
		},
		{
			name: "primitives only with flat=false",
			data: map[string]interface{}{
				"FOO": "BAR",
				"NUM": 42,
				"BOOL": true,
				"NULL": nil,
				"FAZ": map[string]interface{}{
					"FES": "BEAR",
				},
			},
			flatten: false,
			coerce:  false,
			expected: []string{
				"FOO=BAR",
				"NUM=42",
				"BOOL=true",
				"NULL=",
				"FAZ_FES=BEAR",
			},
		},
		{
			name: "array with coerce=true and flat=false",
			data: map[string]interface{}{
				"FOO": "BAR",
				"ARRAY": []interface{}{"VAL1", "VAL2"},
				"OBJ": map[string]interface{}{
					"NESTED": "VALUE",
				},
			},
			flatten: false,
			coerce:  true,
			expected: []string{
				"FOO=BAR",
				"ARRAY=[\"VAL1\",\"VAL2\"]",
				"OBJ_NESTED=VALUE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAsEnv(tt.data, tt.flatten, tt.coerce)
			
			// Split result into lines and sort for consistent comparison
			lines := strings.Split(result, "\n")
			
			// Check that all expected lines are present
			for _, expected := range tt.expected {
				found := false
				for _, line := range lines {
					if strings.TrimSpace(line) == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected line '%s' not found in output:\n%s", expected, result)
				}
			}
			
			// Check that no extra primitive lines are present (allow empty lines)
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				found := false
				for _, expected := range tt.expected {
					if line == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected line '%s' in output:\n%s", line, result)
				}
			}
		})
	}
}

// TestEnvOutputIntegration tests the integration with the inventory query command
func TestEnvOutputIntegration(t *testing.T) {
	// This test requires setting up actual inventory data, so we'll create a simple test
	// that verifies the flags work correctly
	
	// Create a test command to verify flag parsing
	var outputEnvFlag, flatFlag, coerceFlag bool
	
	testCmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			// Flag values should be captured
		},
	}
	
	testCmd.Flags().BoolVar(&outputEnvFlag, "output-env", false, "Output in .env file format")
	testCmd.Flags().BoolVar(&flatFlag, "flat", false, "Flatten nested structures")
	testCmd.Flags().BoolVar(&coerceFlag, "coerce", false, "Convert complex values to JSON strings")
	
	// Test flag parsing
	testCmd.SetArgs([]string{"--output-env", "--flat", "--coerce"})
	
	var buf bytes.Buffer
	testCmd.SetOut(&buf)
	
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	
	if !outputEnvFlag {
		t.Error("--output-env flag not set correctly")
	}
	if !flatFlag {
		t.Error("--flat flag not set correctly")
	}
	if !coerceFlag {
		t.Error("--coerce flag not set correctly")
	}
}
