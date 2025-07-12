package cmd

// DbInventoryEntry represents a database entry in the inventory.
type DbInventoryEntry struct {
	Host       string   `json:"host"`
	Type       string   `json:"type"` // e.g., "postgres", "redis", "mongodb"
	RemotePort int      `json:"remote_port"`
	LocalPort  int      `json:"local_port,omitempty"` // Optional: if not set, a default will be used
	Tags       []string `json:"tags,omitempty"`
}
