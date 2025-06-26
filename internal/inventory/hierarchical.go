package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// HierarchicalInventory manages a jq-like hierarchical data structure
type HierarchicalInventory struct {
	dataDir string
	data    map[string]interface{}
}

// NewHierarchicalInventory creates a new hierarchical inventory instance
func NewHierarchicalInventory(dataDir string) (*HierarchicalInventory, error) {
	hi := &HierarchicalInventory{
		dataDir: dataDir,
		data:    make(map[string]interface{}),
	}

	if err := hi.ensureDataDir(); err != nil {
		return nil, err
	}

	if err := hi.loadData(); err != nil {
		return nil, err
	}

	return hi, nil
}

// ensureDataDir creates the data directory if it doesn't exist
func (hi *HierarchicalInventory) ensureDataDir() error {
	return os.MkdirAll(hi.dataDir, 0755)
}

// loadData loads all inventory data from files
func (hi *HierarchicalInventory) loadData() error {
	// Check if we should use a single file or multiple files
	singleFile := filepath.Join(hi.dataDir, "hierarchical-inventory.json")
	if _, err := os.Stat(singleFile); err == nil {
		return hi.loadFromSingleFile(singleFile)
	}

	// Otherwise, load from multiple *-inventory.json files
	return hi.loadFromMultipleFiles()
}

// loadFromSingleFile loads data from a single hierarchical-inventory.json file
func (hi *HierarchicalInventory) loadFromSingleFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &hi.data)
}

// loadFromMultipleFiles loads data from multiple *-inventory.json files
func (hi *HierarchicalInventory) loadFromMultipleFiles() error {
	files, err := filepath.Glob(filepath.Join(hi.dataDir, "*-inventory.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		// Extract the inventory type from filename (e.g., "db-inventory.json" -> "db")
		baseName := filepath.Base(file)
		inventoryType := strings.TrimSuffix(baseName, "-inventory.json")

		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		var fileData interface{}
		if err := json.Unmarshal(data, &fileData); err != nil {
			continue // Skip invalid JSON files
		}

		hi.data[inventoryType] = fileData
	}

	return nil
}

// saveData saves all inventory data to storage
func (hi *HierarchicalInventory) saveData() error {
	// Prefer single file approach for hierarchical data
	singleFile := filepath.Join(hi.dataDir, "hierarchical-inventory.json")

	data, err := json.MarshalIndent(hi.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(singleFile, data, 0644)
}

// Query performs a jq-like query on the hierarchical data
func (hi *HierarchicalInventory) Query(query string) (interface{}, error) {
	if query == "" {
		return hi.data, nil
	}

	// Parse the query into segments
	segments, err := hi.parseQuery(query)
	if err != nil {
		return nil, err
	}

	// Navigate through the data structure
	return hi.navigate(hi.data, segments)
}

// parseQuery parses a jq-like query string into segments
func (hi *HierarchicalInventory) parseQuery(query string) ([]QuerySegment, error) {
	var segments []QuerySegment

	// Split by dots, but handle array notation
	parts := strings.Split(query, ".")

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Check for standalone array notation [index] or [*]
		standaloneArrayRegex := regexp.MustCompile(`^\[(.+)\]$`)
		if matches := standaloneArrayRegex.FindStringSubmatch(part); matches != nil {
			// Handle array index or wildcard
			indexPart := matches[1]
			if indexPart == "*" {
				segments = append(segments, QuerySegment{
					Type: SegmentTypeWildcard,
				})
			} else {
				index, err := strconv.Atoi(indexPart)
				if err != nil {
					return nil, fmt.Errorf("invalid array index: %s", indexPart)
				}
				segments = append(segments, QuerySegment{
					Type:  SegmentTypeIndex,
					Index: index,
				})
			}
			continue
		}

		// Check for key with array notation key[index] or key[*]
		keyArrayRegex := regexp.MustCompile(`^(.+?)\[(.+)\]$`)
		if matches := keyArrayRegex.FindStringSubmatch(part); matches != nil {
			// Handle the base part first
			if matches[1] != "" {
				segments = append(segments, QuerySegment{
					Type: SegmentTypeKey,
					Key:  matches[1],
				})
			}

			// Handle array index or wildcard
			indexPart := matches[2]
			if indexPart == "*" {
				segments = append(segments, QuerySegment{
					Type: SegmentTypeWildcard,
				})
			} else {
				index, err := strconv.Atoi(indexPart)
				if err != nil {
					return nil, fmt.Errorf("invalid array index: %s", indexPart)
				}
				segments = append(segments, QuerySegment{
					Type:  SegmentTypeIndex,
					Index: index,
				})
			}
		} else {
			// Regular key access
			segments = append(segments, QuerySegment{
				Type: SegmentTypeKey,
				Key:  part,
			})
		}
	}

	return segments, nil
}

// QuerySegment represents a single segment of a query
type QuerySegment struct {
	Type  SegmentType
	Key   string
	Index int
}

// SegmentType represents the type of query segment
type SegmentType int

const (
	SegmentTypeKey SegmentType = iota
	SegmentTypeIndex
	SegmentTypeWildcard
)

// navigate recursively navigates through the data structure
func (hi *HierarchicalInventory) navigate(data interface{}, segments []QuerySegment) (interface{}, error) {
	if len(segments) == 0 {
		return data, nil
	}

	segment := segments[0]
	remaining := segments[1:]

	switch segment.Type {
	case SegmentTypeKey:
		return hi.navigateKey(data, segment.Key, remaining)
	case SegmentTypeIndex:
		return hi.navigateIndex(data, segment.Index, remaining)
	case SegmentTypeWildcard:
		return hi.navigateWildcard(data, remaining)
	default:
		return nil, fmt.Errorf("unknown segment type")
	}
}

// navigateKey handles key-based navigation
func (hi *HierarchicalInventory) navigateKey(data interface{}, key string, remaining []QuerySegment) (interface{}, error) {
	switch d := data.(type) {
	case map[string]interface{}:
		value, exists := d[key]
		if !exists {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return hi.navigate(value, remaining)
	default:
		return nil, fmt.Errorf("cannot access key %s on non-object type", key)
	}
}

// navigateIndex handles array index navigation
func (hi *HierarchicalInventory) navigateIndex(data interface{}, index int, remaining []QuerySegment) (interface{}, error) {
	switch d := data.(type) {
	case []interface{}:
		if index < 0 || index >= len(d) {
			return nil, fmt.Errorf("array index out of bounds: %d", index)
		}
		return hi.navigate(d[index], remaining)
	default:
		return nil, fmt.Errorf("cannot access index %d on non-array type", index)
	}
}

// navigateWildcard handles wildcard navigation
func (hi *HierarchicalInventory) navigateWildcard(data interface{}, remaining []QuerySegment) (interface{}, error) {
	switch d := data.(type) {
	case []interface{}:
		var results []interface{}
		for _, item := range d {
			result, err := hi.navigate(item, remaining)
			if err != nil {
				continue // Skip items that don't match the remaining path
			}
			results = append(results, result)
		}
		return results, nil
	default:
		return nil, fmt.Errorf("cannot use wildcard on non-array type")
	}
}

// Set sets a value at the specified query path
func (hi *HierarchicalInventory) Set(query string, value interface{}) error {
	if query == "" {
		return fmt.Errorf("cannot set root level")
	}

	segments, err := hi.parseQuery(query)
	if err != nil {
		return err
	}

	// Navigate to the parent and set the final key
	if len(segments) == 1 {
		// Setting at root level
		segment := segments[0]
		if segment.Type != SegmentTypeKey {
			return fmt.Errorf("can only set keys at root level")
		}
		hi.data[segment.Key] = value
	} else {
		// Navigate to parent
		parent, err := hi.navigate(hi.data, segments[:len(segments)-1])
		if err != nil {
			// Try to create the path if it doesn't exist
			parent, err = hi.createPath(segments[:len(segments)-1])
			if err != nil {
				return err
			}
		}

		// Set the final value
		finalSegment := segments[len(segments)-1]
		switch finalSegment.Type {
		case SegmentTypeKey:
			parentMap, ok := parent.(map[string]interface{})
			if !ok {
				return fmt.Errorf("cannot set key on non-object type")
			}
			parentMap[finalSegment.Key] = value
		default:
			return fmt.Errorf("can only set keys, not array indices or wildcards")
		}
	}

	return hi.saveData()
}

// createPath creates a path in the data structure if it doesn't exist
func (hi *HierarchicalInventory) createPath(segments []QuerySegment) (interface{}, error) {
	current := hi.data

	for _, segment := range segments {
		if segment.Type != SegmentTypeKey {
			return nil, fmt.Errorf("can only create paths with keys")
		}

		if _, exists := current[segment.Key]; !exists {
			current[segment.Key] = make(map[string]interface{})
		}

		next, ok := current[segment.Key].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("path conflict: %s is not an object", segment.Key)
		}
		current = next
	}

	return current, nil
}

// Delete removes a value at the specified query path
func (hi *HierarchicalInventory) Delete(query string) error {
	if query == "" {
		return fmt.Errorf("cannot delete root level")
	}

	segments, err := hi.parseQuery(query)
	if err != nil {
		return err
	}

	if len(segments) == 1 {
		// Deleting at root level
		segment := segments[0]
		if segment.Type != SegmentTypeKey {
			return fmt.Errorf("can only delete keys at root level")
		}
		delete(hi.data, segment.Key)
	} else {
		// Navigate to parent
		parent, err := hi.navigate(hi.data, segments[:len(segments)-1])
		if err != nil {
			return err
		}

		// Delete the final key
		finalSegment := segments[len(segments)-1]
		if finalSegment.Type != SegmentTypeKey {
			return fmt.Errorf("can only delete keys, not array indices")
		}

		parentMap, ok := parent.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot delete key from non-object type")
		}

		delete(parentMap, finalSegment.Key)
	}

	return hi.saveData()
}

// List returns all keys at the specified path level
func (hi *HierarchicalInventory) List(query string) ([]string, error) {
	data, err := hi.Query(query)
	if err != nil {
		return nil, err
	}

	switch d := data.(type) {
	case map[string]interface{}:
		var keys []string
		for key := range d {
			keys = append(keys, key)
		}
		return keys, nil
	default:
		return nil, fmt.Errorf("cannot list keys on non-object type")
	}
}

// GetData returns the raw data for debugging/inspection
func (hi *HierarchicalInventory) GetData() map[string]interface{} {
	return hi.data
}
