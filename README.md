# Tsukuyo

> A CLI tool to streamline SSH connections and inventory management

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Version](https://img.shields.io/badge/version-0.1.0-green.svg)
![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg)

## 📖 About

Tsukuyo is a command-line tool designed to automate and streamline various operational tasks. It provides a unified interface for managing SSH connections (both standard SSH and Teleport SSH), maintaining inventories of connection details, and executing predefined scripts. The goal is to reduce manual steps in common workflows and improve productivity.

## ✨ Features

### Implemented Features

- **SSH Connection Management**
  - **Standard SSH**: Connect to hosts using standard OpenSSH client with saved configurations
  - **Teleport SSH (TSH)**: Interactive wizard for connecting to hosts via Teleport SSH
  - **SSH Tunneling**: Support for port forwarding with both SSH and TSH

- **Inventory Management**
  - **Node Inventory**: Store SSH connection details (hostname, user, port)
  - **Database Inventory**: Store database connection details
  - **Interactive Selection**: User-friendly prompts for selecting from available options

- **Script Management**
  - **Script Library**: Store, organize, and execute shell scripts
  - **Environment Variable Support**: Run scripts with environment variables from files
  - **Script Metadata**: Add descriptions and tags for organization
  - **Editor Integration**: Edit scripts directly from the CLI
  - **Search & Filter**: Find scripts by name, tag, or description

- **Data Persistence**
  - Local storage in `~/.tsukuyo` directory
  - Individual script files with metadata
  - JSON-based storage format

### Planned Features

- **Hierarchical Inventory**:
  - Support for more complex data structures in inventory
  - Query similar to `jq` for structured data

## 🚀 Installation

### Prerequisites

- Go 1.22 or higher
- For TSH functionality: [Teleport](https://goteleport.com/docs/installation/) client installed and configured

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

### Inventory Management

Manage database inventory:

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

Manage node inventory:

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

- Node inventory: `~/.tsukuyo/inventory/node-inventory.json`
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

- Database inventory: `~/.tsukuyo/inventory/db-inventory.json`
  ```json
  {
    "db-key": "database-hostname"
  }
  ```

- Script storage:
  - Script content: `~/.tsukuyo/scripts/<script-name>` (executable files)
  - Script metadata: `~/.tsukuyo/scripts/<script-name>.meta.json`
    ```json
    {
      "name": "script-name",
      "description": "What this script does",
      "tags": ["tag1", "tag2", "category"]
    }
    ```

## 🧰 Tech Stack

- **[Go](https://golang.org/)**: Core language
- **[Cobra](https://github.com/spf13/cobra)**: CLI framework
- **[promptui](https://github.com/manifoldco/promptui)**: Interactive prompt library
- **Standard Library**:
  - `encoding/json`: For JSON handling
  - `os/exec`: For executing shell commands
  - `net`: For network operations
  - `bufio`: For reading script content
  - `os`: For filesystem operations

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 Roadmap

- Implement hierarchical inventory structure
- Add support for multiple SSH keys
- Add configuration options
- Expand TSH integration capabilities
- Add support for other script languages (Node.js, Python)
- Add tests

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## ✏️ Author

[Arung Agamani](https://github.com/arung-agamani)