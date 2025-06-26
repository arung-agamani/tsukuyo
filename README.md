# Tsukuyo

> A CLI tool to streamline SSH connections and inventory management

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Version](https://img.shields.io/badge/version-0.1.0-green.svg)
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

    -   **Node Inventory**: Store SSH connection details (hostname, user, port)
    -   **Database Inventory**: Store database connection details
    -   **Hierarchical Inventory**: Query complex nested data structures with jq-like syntax
    -   **Interactive Selection**: User-friendly prompts for selecting from available options

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

### Implemented Features

-   **SSH Connection Management**

    -   **Standard SSH**: Connect to hosts using standard OpenSSH client with saved configurations
    -   **Teleport SSH (TSH)**: Interactive wizard for connecting to hosts via Teleport SSH
    -   **SSH Tunneling**: Support for port forwarding with both SSH and TSH

-   **Inventory Management**

    -   **Node Inventory**: Store SSH connection details (hostname, user, port)
    -   **Database Inventory**: Store database connection details
    -   **Hierarchical Inventory**: Query complex nested data structures with jq-like syntax
    -   **Interactive Selection**: User-friendly prompts for selecting from available options

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
```

Manage SSH node inventory:

```bash
# List all saved nodes
tsukuyo ssh list

# Add new node (interactive)
tsukuyo ssh set

# Add new node (direct)
tsukuyo ssh set <name> <host> [user]

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
tsukuyo tsh --with-db michiru_ch_scroll_db
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

#### Advanced Examples

**Complex server inventory:**

```bash
# Set up environment-based server configuration
tsukuyo inventory set environments.production.servers '[{"name":"web-prod-1","host":"10.0.1.10","role":"web"},{"name":"db-prod-1","host":"10.0.1.20","role":"database"}]'
tsukuyo inventory set environments.staging.servers '[{"name":"web-stage-1","host":"10.0.2.10","role":"web"}]'

# Query all production servers
tsukuyo inventory query environments.production.servers

# Get all server names in production
tsukuyo inventory query environments.production.servers.[*].name

# Get database servers across environments
tsukuyo inventory query environments.[*].servers.[*].host
```

**Configuration management:**

```bash
# Store application configurations
tsukuyo inventory set config.app.debug true
tsukuyo inventory set config.app.workers 8
tsukuyo inventory set config.database.pool_size 20

# Query entire configuration
tsukuyo inventory query config
```

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
            "user": "username"
        }
    }
    ```

-   **Legacy database inventory**: `~/.tsukuyo/db-inventory.json`

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
-   Add support for multiple SSH keys
-   Add configuration options
-   Expand TSH integration capabilities
-   Add support for other script languages (Node.js, Python)
-   Add import/export functionality for inventory data

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## ✏️ Author

[Arung Agamani](https://github.com/arung-agamani)
