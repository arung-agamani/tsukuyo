# Tsukuyo

> A CLI tool to streamline SSH connections and inventory management

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Version](https://img.shields.io/badge/version-0.2.0-green.svg)
![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg)

## 📖 About

Tsukuyo is a command-line tool designed to automate and streamline various operational tasks. It provides a unified interface for managing SSH connections (both standard SSH and Teleport SSH), maintaining inventories of connection details, and executing predefined scripts. The goal is to reduce manual steps in common workflows and improve productivity.

## ✨ Features

### Implemented Features

-   **SSH Connection Management**

    -   **Standard SSH**: Connect to hosts using standard OpenSSH client with saved configurations
    -   **Teleport SSH (TSH)**: Interactive wizard for connecting to hosts via Teleport SSH
    -   **SSH Tunneling**: Support for port forwarding with both SSH and TSH

-   **Inventory Management**

    -   **Node Inventory**: Store SSH connection details (hostname, user, port, and tags)
    -   **Enhanced Database Inventory**: Store structured database connection details with modern CLI interface
    -   **Automatic Recovery**: Self-healing database inventory that recovers from deletion or corruption
    -   **Command-Line Flags**: Modern interface with `--type`, `--remote-port`, `--local-port`, `--tags` flags
    -   **Smart Defaults**: Sensible defaults (postgres/5432) with intelligent fallback to interactive mode
    -   **Tag-Based Filtering**: Automatically filter database connections based on node tags when using `--with-db`.
    -   **Hierarchical Inventory**: Query complex nested data structures with jq-like syntax
    -   **Interactive Selection**: User-friendly searchable prompts for selecting from available options

-   **Script Management**

    -   **Script Library**: Store, organize, and execute shell scripts
    -   **Environment Variable Support**: Run scripts with environment variables from files
    -   **Script Metadata**: Add descriptions and tags for organization
    -   **Editor Integration**: Edit scripts directly from the CLI
    -   **Search & Filter**: Find scripts by name, tag, or description

-   **Data Persistence**
    -   Local storage in `~/.tsukuyo` directory
    -   Individual script files with metadata
    -   JSON-based storage format
    -   Hierarchical inventory with flexible data structures

## 🔄 Recent Changes (v0.2.0)

### Enhanced Database Inventory System

We've significantly improved the database inventory management with robust recovery capabilities and an enhanced command-line interface.

#### 🛡️ **Automatic Recovery System**

The DB inventory now features a self-healing architecture:

-   **Recovery from Deletion**: If the `db` key is accidentally deleted (`tsukuyo inventory delete db`), any database command automatically recreates it as an empty structure
-   **Recovery from Corruption**: If the `db` key is set to an invalid type (e.g., a string instead of an object), it's automatically fixed on the next database operation
-   **Smart Type Recognition**: Known inventory types (`db`, `node`, `script`) are always available as commands, even when they don't exist in the inventory yet

**Before**: Users had to manually run `tsukuyo inventory set db {}` after accidental deletion
**After**: Database commands work seamlessly - no manual intervention required

```bash
# This sequence now works seamlessly:
tsukuyo inventory delete db           # Accidentally delete DB inventory
tsukuyo inventory db list            # Automatically recreates and shows empty list
tsukuyo inventory db set my-db host  # Works immediately
```

#### 🚀 **Enhanced Command-Line Interface**

The `tsukuyo inventory db set` command now supports a modern CLI experience:

**New Syntax:**
```bash
tsukuyo inventory db set <name> <host> [flags]
```

**Available Flags:**
-   `--type <string>`: Database type (e.g., postgres, redis, mongodb)
-   `--remote-port <int>`: Remote port number  
-   `--local-port <int>`: Local port number (optional)
-   `--tags <string>`: Comma-separated tags

**Smart Defaults:**
-   Database type: `postgres`
-   Remote port: `5432`
-   Local port: Not set (uses default tunneling behavior)

**Examples:**

```bash
# Minimal usage with defaults (postgres, port 5432)
tsukuyo inventory db set my-postgres postgres.example.com

# Full specification with all options
tsukuyo inventory db set prod-db postgres.prod.com \
  --type postgres \
  --remote-port 5432 \
  --local-port 15432 \
  --tags "production,primary,postgresql"

# Redis with custom port
tsukuyo inventory db set cache-db redis.example.com \
  --type redis \
  --remote-port 6379 \
  --tags "cache,redis"

# MongoDB development database
tsukuyo inventory db set dev-mongo mongo.dev.com \
  --type mongodb \
  --remote-port 27017 \
  --local-port 27018 \
  --tags "development,mongodb"
```

#### 🔄 **Seamless Fallback to Interactive Mode**

When arguments or flags are missing, the command intelligently falls back to interactive mode:

```bash
# Missing arguments - prompts for name and host
tsukuyo inventory db set

# Missing flags - uses defaults or prompts for critical values
tsukuyo inventory db set my-db postgres.example.com
# Uses: type=postgres, remote_port=5432, local_port=unset, tags=empty
```

#### 🧪 **Comprehensive Testing**

The enhanced system includes extensive test coverage:

-   **Recovery Testing**: Verification of automatic recovery from deletion and corruption
-   **Command Interface Testing**: Validation of argument parsing, flag handling, and defaults
-   **Integration Testing**: End-to-end testing of the complete workflow
-   **Validation Testing**: Structure validation and error handling

**Test Results**: All 20+ test cases pass, ensuring reliability and backwards compatibility.

#### 📈 **Migration & Compatibility**

-   **Backwards Compatible**: Existing database entries continue to work without modification
-   **No Breaking Changes**: All existing commands and workflows remain functional
-   **Gradual Migration**: New entries use the enhanced structure, while legacy entries are preserved
-   **Validation Warnings**: Helpful warnings for entries that don't follow the new structure

This update significantly improves the user experience by eliminating manual recovery steps and providing a more intuitive command-line interface while maintaining full compatibility with existing data.

## 🚀 Installation

### Prerequisites

-   Go 1.22 or higher
-   For TSH functionality: [Teleport](https://goteleport.com/docs/installation/) client installed and configured

### Build from Source

```bash
# Clone the repository
git clone https://github.com/arung-agamani/tsukuyo.git
cd tsukuyo

# Build the binary
go build -o tsukuyo

# Make it available in your PATH
cp tsukuyo /usr/local/bin/ # or ~/bin/ if in your PATH
```

## 📋 Usage Guide

### Standard SSH

Connect to a saved node:

```bash
tsukuyo ssh <node-name>
# Example:
tsukuyo ssh izuna
```

SSH with tunneling:

```bash
tsukuyo ssh <node-name> --tunnel <local-port>:<remote-host>:<remote-port>
# Example:
tsukuyo ssh izuna --tunnel 8080:localhost:80

# SSH with database tunneling (interactive selection)
tsukuyo ssh <node-name> --with-db
```

Manage SSH node inventory:

```bash
# List all saved nodes
tsukuyo ssh list

# Add new node (interactive)
tsukuyo ssh set

# Add new node (direct) with tags
tsukuyo ssh set <name> <host> [user]
# You will be prompted to add tags

# Get node details (interactive)
tsukuyo ssh get

# Get node details (direct)
tsukuyo ssh get <name>
```

### Teleport SSH (TSH)

Connect to a node with interactive selection:

```bash
tsukuyo tsh
```

Connect with database tunneling:

```bash
tsukuyo tsh --with-db
# or specify a specific DB from inventory:
tsukuyo tsh --with-db my-db-key
```

### Script Management

Create and add a new script:

```bash
# Add new script (interactive)
tsukuyo script add
```

List all available scripts:

```bash
# List all scripts with descriptions and tags
tsukuyo script list
```

Search for scripts:

```bash
# Search by name, description, or tag
tsukuyo script search <query>

# Examples:
tsukuyo script search backup
tsukuyo script search deploy
```

Run a script:

```bash
# Basic execution
tsukuyo script run <script-name>

# Run with environment variables from file
tsukuyo script run <script-name> --with-env-file path/to/.env

# Preview script contents without executing (dry run)
tsukuyo script run <script-name> --dry-run

# Edit script before running
tsukuyo script run <script-name> --edit
```

Edit a script:

```bash
# Opens script in your preferred editor ($EDITOR or vi)
tsukuyo script edit <script-name>
```

Delete a script:

```bash
# Remove a script and its metadata
tsukuyo script delete <script-name>
```

#### Example Script Workflows

**1. Create a backup script:**

```bash
$ tsukuyo script add
Script name: backup-postgres
Description: Backup PostgreSQL database to S3
Tags (comma separated): backup, postgres, database
Enter script content (end with EOF/Ctrl+D):
#!/bin/bash
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DB_NAME=$1
BACKUP_PATH="/tmp/${DB_NAME}_${TIMESTAMP}.sql"

echo "Backing up $DB_NAME to $BACKUP_PATH..."
pg_dump -U postgres $DB_NAME > $BACKUP_PATH

echo "Uploading to S3..."
aws s3 cp $BACKUP_PATH s3://my-backups/postgres/
^D
Script added: backup-postgres
```

**2. Run with environment variables:**

```bash
$ cat .env
AWS_ACCESS_KEY_ID=AKIAXXXXXXXX
AWS_SECRET_ACCESS_KEY=xxxxxxxxxx
AWS_DEFAULT_REGION=us-west-2

$ tsukuyo script run backup-postgres --with-env-file .env
```

**3. Find scripts for deployment:**

```bash
$ tsukuyo script search deploy
NAME                 DESCRIPTION                                  TAGS
deploy-frontend      Deploy frontend app to production            deploy, frontend
deploy-api           Deploy API server to staging                 deploy, backend
```

### Hierarchical Inventory

The hierarchical inventory system provides a powerful way to store and query complex nested data structures using jq-like syntax.

#### Basic Usage

**Set values:**

```bash
# Set simple values
tsukuyo inventory set db.izuna-db.host "kureya.howlingmoon.dev"
tsukuyo inventory set db.izuna-db.port 2333
tsukuyo inventory set db.izuna-db.user "admin"

# Set complex JSON structures
tsukuyo inventory set servers.web '[{"name":"web-1","host":"192.168.1.10"},{"name":"web-2","host":"192.168.1.11"}]'
```

**Query values:**

```bash
# Query specific values
tsukuyo inventory query db.izuna-db.port
# Output: 2333

# Query entire objects
tsukuyo inventory query db.izuna-db
# Output: {"host":"kureya.howlingmoon.dev","port":2333,"user":"admin"}

# Query top-level categories
tsukuyo inventory query db
# Output: {"izuna-db":{"host":"kureya.howlingmoon.dev","port":2333,"user":"admin"}}
```

**Array queries:**

```bash
# Access array elements by index
tsukuyo inventory query servers.web.[0].name
# Output: web-1

# Use wildcards to query all elements
tsukuyo inventory query servers.web.[*].host
# Output: ["192.168.1.10","192.168.1.11"]
```

**List and delete:**

```bash
# List keys at any level
tsukuyo inventory list db
# Shows: izuna-db

# Delete values
tsukuyo inventory delete db.izuna-db.port
```

#### Database Inventory

The database inventory supports a structured format with enhanced command-line interface and automatic recovery capabilities.

**Enhanced Set Command with Flags:**

```bash
# Modern CLI interface with arguments and flags
tsukuyo inventory db set <name> <host> [flags]

# Examples:
tsukuyo inventory db set prod-postgres postgres.prod.com \
  --type postgres \
  --remote-port 5432 \
  --local-port 15432 \
  --tags "production,primary"

# With smart defaults (postgres, port 5432)
tsukuyo inventory db set dev-redis redis.dev.com \
  --type redis \
  --remote-port 6379

# Minimal usage (uses postgres defaults)
tsukuyo inventory db set simple-db postgres.example.com
```

**Available Flags:**
- `--type`: Database type (default: postgres)
- `--remote-port`: Remote port (default: 5432)  
- `--local-port`: Local port for tunneling (optional)
- `--tags`: Comma-separated tags for filtering

**Interactive Mode (Legacy/Fallback):**

```bash
# Add a new database entry interactively (when args/flags missing)
tsukuyo inventory db set
# You will be prompted for:
# - Name (e.g., redis-prod)
# - Host (e.g., cache.example.com)
# - Type (e.g., redis)
# - Remote Port (e.g., 6379)
# - Local Port (optional, defaults to remote port)
# - Tags (comma-separated, e.g., prod,cache)
```

**List and Get Operations:**

```bash
# List all database entries
tsukuyo inventory db list
# Output:
# Available db entries:
#   - redis-prod
#   - postgres-main
#   - mongo-dev

# Get specific database entry details
tsukuyo inventory db get redis-prod
# Output: JSON formatted entry with all fields
```

**Automatic Recovery:**

The system automatically recovers from common issues:
- If `db` inventory is deleted, it's recreated on first use
- If `db` key becomes corrupted, it's automatically fixed
- No manual intervention required

### Legacy Inventory Management

Manage database inventory (legacy):

```bash
# List all database entries
tsukuyo inventory db list

# Add new database (interactive)
tsukuyo inventory db set

# Add new database (direct)
tsukuyo inventory db set <key> <value>

# Get database details (interactive)
tsukuyo inventory db get
```

Manage node inventory (legacy):

```bash
# List all nodes
tsukuyo inventory node list

# Add new node (interactive)
tsukuyo inventory node set

# Add new node (direct)
tsukuyo inventory node set <name> <host> [user]

# Get node details (interactive)
tsukuyo inventory node get

# Get node details (direct)
tsukuyo inventory node get <name>
```

## 🛠️ Architecture

Tsukuyo is built with the following components:

1. **Command Structure**: Using [Cobra](https://github.com/spf13/cobra) for command-line interface
2. **Interactive UI**: Using [promptui](https://github.com/manifoldco/promptui) for interactive prompts
3. **Data Storage**: JSON files in `.data/` directory for persistence
4. **SSH Integration**: Wrapper around system SSH and TSH clients

### Data Storage Schema

-   **Hierarchical inventory**: `~/.tsukuyo/hierarchical-inventory.json`

    ```json
    {
        "db": {
            "izuna-db": {
                "host": "kureya.howlingmoon.dev",
                "port": "2333",
                "user": "abcd",
                "pass": "pass"
            },
            "redis-prod": {
                "host": "cache.example.com",
                "type": "redis",
                "remote_port": 6379,
                "local_port": 6379,
                "tags": ["prod", "cache"]
            }
        },
        "servers": {
            "web": [
                {
                    "name": "web-1",
                    "host": "192.168.1.10",
                    "env": "production"
                },
                {
                    "name": "web-2",
                    "host": "192.168.1.11",
                    "env": "production"
                }
            ]
        }
    }
    ```

-   **Legacy node inventory**: `~/.tsukuyo/node-inventory.json`

    ```json
    {
        "node-name": {
            "name": "node-name",
            "host": "hostname",
            "type": "ssh",
            "port": 22,
            "user": "username",
            "tags": ["prod", "web"]
        }
    }
    ```

-   **Legacy database inventory**: `~/.tsukuyo/db-inventory.json` (still supported but new entries use the hierarchical store)

    ```json
    {
        "db-key": "database-hostname"
    }
    ```

-   **Script storage**:
    -   Script content: `~/.tsukuyo/scripts/<script-name>` (executable files)
    -   Script metadata: `~/.tsukuyo/scripts/<script-name>.meta.json`
        ```json
        {
            "name": "script-name",
            "description": "What this script does",
            "tags": ["tag1", "tag2", "category"]
        }
        ```

## 🧰 Tech Stack

-   **[Go](https://golang.org/)**: Core language
-   **[Cobra](https://github.com/spf13/cobra)**: CLI framework
-   **[promptui](https://github.com/manifoldco/promptui)**: Interactive prompt library
-   **Standard Library**:
    -   `encoding/json`: For JSON handling
    -   `os/exec`: For executing shell commands
    -   `net`: For network operations
    -   `bufio`: For reading script content
    -   `os`: For filesystem operations

## 🧪 Testing

The project includes comprehensive tests for the hierarchical inventory system.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run only hierarchical inventory tests
go test ./internal/inventory/

# Run tests with coverage
go test -cover ./...
```

### Test Coverage

The hierarchical inventory tests cover:

-   **Basic Queries**: Simple key-value access (`db.izuna-db.port`)
-   **Structured Data**: Handling of complex `DbInventoryEntry` objects.
-   **Nested Object Queries**: Access to nested structures (`db.izuna-db`)
-   **Array Access**: Index-based access (`servers.[0].name`)
-   **Wildcard Queries**: Bulk operations (`servers.[*].name`)
-   **Data Persistence**: File-based storage and loading
-   **Legacy Compatibility**: Migration from old inventory formats
-   **Error Handling**: Invalid queries and malformed data
-   **Edge Cases**: Root-level operations, complex nested structures

### Example Test Data

The tests use data structures matching the examples in the directives:

```json
{
    "db": {
        "izuna-db": [
            {
                "host": "kureya.howlingmoon.dev",
                "port": "2333",
                "user": "abcd",
                "pass": "pass",
                "env": "int"
            },
            {
                "host": "kureya.howlingmoon.dev",
                "port": "2333",
                "user": "abcd",
                "pass": "pass",
                "env": "prd"
            }
        ]
    }
}
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 Roadmap

-   ✅ Implement hierarchical inventory structure with jq-like queries
-   ✅ Add comprehensive test coverage for hierarchical inventory
-   ✅ Rework DB inventory to be structured with types, ports, and tags.
-   ✅ Implement tag-based filtering for DB tunnels in `ssh` and `tsh`.
-   ✅ Enhanced DB inventory with automatic recovery and modern CLI interface
-   ✅ Command-line flags support for database management
-   ✅ Automatic recovery system for corrupted/deleted DB inventory
-   Add support for multiple SSH keys
-   Add configuration options
-   Expand TSH integration capabilities
-   Add support for other script languages (Node.js, Python)
-   Add import/export functionality for inventory data

## Tsukuyo?

![](https://cdn.howlingmoon.dev/101568022_p0.png)
_[Tsukuyo](https://www.pixiv.net/en/artworks/101568022)_

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## Not so Important Notes

Almost the whole entirity of this project is done through
vibe coding, though I know what I'm doing, because in some 
moments, the LLM models are just plain dumb.

## ✏️ Author

[Arung Agamani](https://github.com/arung-agamani)
