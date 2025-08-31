[![Deploy Documentation](https://github.com/teabranch/matlas-cli/actions/workflows/docs.yml/badge.svg?branch=main)](https://github.com/teabranch/matlas-cli/actions/workflows/docs.yml)
[![Release](https://github.com/teabranch/matlas-cli/actions/workflows/release.yml/badge.svg)](https://github.com/teabranch/matlas-cli/actions/workflows/release.yml)
## 🚀 matlas — one CLI for Atlas and MongoDB

> 🌟 **The unified, Go-powered CLI that bridges MongoDB Atlas management and database operations**

Matlas is your **all-in-one command center** for MongoDB Atlas and database management. Think of it as the friendly bridge between the Atlas SDK and the MongoDB Go Driver — spin up projects and clusters, configure network access, and seamlessly dive into database tasks like listing collections or inspecting indexes, all from a single, powerful tool! ⚡

### 💡 Why we built it
- 🔄 **Context switching hurts**: Cloud management lives in one world (Atlas APIs), while day‑to‑day database tasks live in another (drivers and shells). We wanted both, together.
- 🧠 **A single mental model**: One set of flags, one config, one output style for both Atlas and database operations.
- ⚡ **Grease the path**: From "create a cluster" to "inspect collections" in seconds — no hunting for another CLI.

### 🎯 Inspired by Terraform and kubectl: meet Discover
We love how Terraform plans changes before applying, and how kubectl lets you declare desired state. Matlas combines those vibes:

- 📸 **Discover**: Snapshot your Atlas org/project (and optionally databases) into clean YAML:
  ```bash
  matlas discover --project-id <id> -o atlas.yaml --include-databases
  ```
- 🔍 **Plan/Diff**: Treat that YAML as your desired state, compare it to reality:
  ```bash
  matlas infra plan -f atlas.yaml
  matlas infra diff -f atlas.yaml
  ```
- 🚀 **Apply (dry-run first)**: Roll changes out, Terraform‑style:
  ```bash
  matlas infra -f atlas.yaml --dry-run
  matlas infra -f atlas.yaml --auto-approve
  ```

✨ Matlas doesn't try to be Terraform or kubectl. It borrows the best ideas so Atlas work feels **safe**, **reviewable**, and **repeatable** — and it keeps database tasks close at hand.

## 🛠️ What you can do

| Feature | Description |
|---------|-------------|
| 🌐 **Atlas** | List/get/create/update/delete projects, clusters, users, network access, peering, and network containers |
| 🔍 **Atlas Search** | Create and manage Atlas Search indexes for full-text and vector search capabilities |
| 🔗 **VPC Endpoints** | Configure VPC endpoints and Private Link connections for secure Atlas connectivity |
| 🗄️ **Databases** | List/create/delete databases, collections, and indexes — either via connection string or Atlas cluster reference |
| 👥 **Database Users** | Create/list/update/delete database-specific users with custom roles — via connection string or Atlas cluster reference |
| 📋 **Infra** | Discover current state, plan/diff/apply/destroy via declarative YAML |

## 📦 Installation

### ⚡ Quick Install (Recommended)

**macOS & Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/teabranch/matlas-cli/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/teabranch/matlas-cli/main/install.ps1" -OutFile "install.ps1"; .\install.ps1
```

### 📋 Platform-Specific Installation

#### macOS & Linux

**Using the installation script:**
```bash
# Download and run installer
curl -fsSL https://raw.githubusercontent.com/teabranch/matlas-cli/main/install.sh -o install.sh
chmod +x install.sh
./install.sh

# Install specific version
./install.sh --version v1.2.3

# Install to custom directory (no sudo required)
./install.sh --dir ~/.local/bin

# Install to user directory via environment variable
MATLAS_INSTALL_DIR=~/.local/bin ./install.sh
```

**Manual installation:**
1. Download the latest release from [GitHub Releases](https://github.com/teabranch/matlas-cli/releases)
2. Extract the archive: `tar -xzf matlas-*.tar.gz`
3. Move binary to PATH: `sudo mv matlas /usr/local/bin/`
4. Make executable: `sudo chmod +x /usr/local/bin/matlas`

#### Windows

**Using PowerShell (Recommended):**
```powershell
# Download installer
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/teabranch/matlas-cli/main/install.ps1" -OutFile "install.ps1"

# Run installer (may require Administrator privileges)
.\install.ps1

# Install specific version
.\install.ps1 -Version v1.2.3

# Install to user directory (no admin required)
.\install.ps1 -InstallDir "$env:USERPROFILE\matlas"

# Skip automatic PATH setup
.\install.ps1 -NoPathSetup
```

**Manual installation:**
1. Download the Windows release from [GitHub Releases](https://github.com/teabranch/matlas-cli/releases)
2. Extract the ZIP file
3. Move `matlas.exe` to a directory in your PATH (e.g., `C:\Program Files\matlas\`)
4. Add the directory to your system PATH if needed

### 🛠️ Build from Source

**Prerequisites:**
- 🐹 **Go 1.24+** required

```bash
# Clone the repository
git clone https://github.com/teabranch/matlas-cli.git
cd matlas-cli

# Quick build
make build

# Or manually
go build -o bin/matlas ./...

# Cross-compile for all platforms
./scripts/build/build.sh cross
```

### 🗑️ Uninstallation

**macOS & Linux:**
```bash
# Download and run uninstaller
curl -fsSL https://raw.githubusercontent.com/teabranch/matlas-cli/main/uninstall.sh | bash

# Or manually
sudo rm /usr/local/bin/matlas
rm -rf ~/.matlas  # Remove config directory
```

**Windows:**
```powershell
# Download and run uninstaller
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/teabranch/matlas-cli/main/uninstall.ps1" -OutFile "uninstall.ps1"; .\uninstall.ps1

# Or manually remove
Remove-Item "C:\Program Files\matlas\matlas.exe"  # Adjust path as needed
Remove-Item -Recurse -Force "$env:USERPROFILE\.matlas"  # Remove config
```

### ✅ Verify Installation

After installation, verify that matlas is working:

```bash
# Check if matlas is in PATH
matlas --version

# Show help
matlas --help

# Check configuration
matlas config --help
```

### 🔄 Upgrading

**Quick upgrade to latest version:**
```bash
curl -fsSL https://raw.githubusercontent.com/teabranch/matlas-cli/main/upgrade.sh | bash
```

**Advanced upgrade options:**
```bash
# Download upgrade script
curl -fsSL https://raw.githubusercontent.com/teabranch/matlas-cli/main/upgrade.sh -o upgrade.sh
chmod +x upgrade.sh

# Upgrade to latest version
./upgrade.sh

# Upgrade to specific version
./upgrade.sh --version v1.2.3

# Force reinstall current version
./upgrade.sh --force
```

### 🔧 Installation Options

| Method | Pros | Cons | Requires Admin |
|--------|------|------|----------------|
| **Quick Install** | Easy, automatic PATH setup | Requires internet | System dirs: Yes |
| **Installation Script** | Customizable, version selection | Requires download | System dirs: Yes |
| **Manual Install** | Full control | Manual PATH setup | System dirs: Yes |
| **Build from Source** | Latest features, customizable | Requires Go toolchain | No |

### 📁 Installation Directories

**Default locations:**
- **macOS/Linux**: `/usr/local/bin` (system-wide) or `~/.local/bin` (user)
- **Windows**: `C:\Program Files\matlas` (system-wide) or `%USERPROFILE%\matlas` (user)

**Custom installation:**
- Set `MATLAS_INSTALL_DIR` environment variable
- Use `--dir` (Unix) or `-InstallDir` (Windows) flag

### 🐚 Shell Integration

The installer automatically adds matlas to your PATH and detects your shell:

- **Bash**: Updates `~/.bashrc` or `~/.bash_profile`
- **Zsh**: Updates `~/.zshrc`
- **Fish**: Updates `~/.config/fish/config.fish`
- **PowerShell**: Updates user PATH environment variable

To manually add to PATH:
```bash
# Bash/Zsh
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc

# Fish
echo 'set -gx PATH /usr/local/bin $PATH' >> ~/.config/fish/config.fish

# PowerShell
$env:PATH += ";C:\Program Files\matlas"
```

## 🔐 Authenticate

### Environment Variables
```bash
export ATLAS_PUB_KEY="your-public-key"
export ATLAS_API_KEY="your-api-key"
# Optional
export ATLAS_PROJECT_ID="your-project-id"
export ATLAS_ORG_ID="your-org-id"
```

### YAML Configuration
Create `~/.matlas/config.yaml`:
```yaml
apiKey: your-api-key
publicKey: your-public-key
projectId: your-project-id  # optional
orgId: your-org-id          # optional
output: json                # optional
timeout: 30s                # optional
```

### Command Line Flags
```bash
matlas --api-key <key> --pub-key <key> --project-id <id> --org-id <id> [command]
```

## 🚀 Quick start

### Atlas Management
```bash
# List projects
matlas atlas projects list --org-id <id>

# Get a specific project  
matlas atlas projects get --project-id <id>

# List database users
matlas atlas users list --project-id <id>

# Create Atlas user with password display (optional)
matlas atlas users create --project-id <id> --username testuser --show-password \
  --roles "readWriteAnyDatabase" --scopes "cluster1,cluster2"

# List network access rules
matlas atlas network list --project-id <id>

# List Atlas Search indexes
matlas atlas search list --project-id <id> --cluster <name>

# Create a basic search index
matlas atlas search create --project-id <id> --cluster <name> \
  --database <db> --collection <coll> --name <index-name> --type search

# Create VPC endpoint
matlas atlas vpc-endpoints create --project-id <id> --provider AWS \
  --region us-east-1 --service-name com.amazonaws.vpce.us-east-1.vpce-svc-123
```

### 🗄️ Database Operations
```bash
# List databases (via connection string)
matlas database list --connection-string "mongodb+srv://..."

# List databases (via Atlas cluster)
matlas database list --cluster <name> --project-id <id> --use-temp-user

# List collections in a database
matlas database collections list --connection-string ... --database mydb

# List indexes in a collection
matlas database collections indexes list \
  --connection-string ... \
  --database mydb \
  --collection mycoll

# Database user management
matlas database users list --connection-string ... --database mydb
matlas database users create --cluster <name> --project-id <id> --use-temp-user \
  --database mydb --username testuser --password "securepass" --roles "readWrite"

# Database custom roles management  
matlas database roles list --cluster <name> --project-id <id> --use-temp-user --database mydb
matlas database roles create --cluster <name> --project-id <id> --use-temp-user \
  --database mydb --role-name "customRole" --privileges '...'
```

### 📋 Declarative Infrastructure Workflows
```bash
# 🔍 Discover current state
matlas discover --project-id <id> -o atlas.yaml --include-databases --convert-to-apply

# 📊 Plan and preview changes
matlas infra plan -f config.yaml
matlas infra diff -f config.yaml

# 🚀 Apply changes (with safety checks)
matlas infra -f config.yaml --dry-run              # Preview first
matlas infra -f config.yaml --auto-approve         # Apply changes

# 📸 Show current state
matlas infra show --project-id <id>

# 🗑️ Destroy resources
matlas infra destroy -f config.yaml                # From config
matlas infra destroy --discovery-only --project-id <id>  # Discovery only
```

## ⚙️ Configuration

### Configuration Priority (highest to lowest)
1. 🏗️ **Built-in defaults**
2. 📄 **YAML file**: `~/.matlas/config.yaml` or `--config` / `ATLAS_CONFIG_FILE`
3. 🌍 **Environment variables**: `ATLAS_*` prefix (e.g., `ATLAS_OUTPUT`, `ATLAS_TIMEOUT`)
4. 🚩 **Command flags**: `--api-key`, `--project-id`, etc.

### 🔑 Credentials Resolution
Matlas looks for credentials in this order:
1. Command flags/YAML config
2. Environment variables (`ATLAS_API_KEY`, `ATLAS_PUB_KEY`)
3. Platform-specific secure storage:
   - **macOS**: Keychain (`security` command)
   - **Windows**: Credential Manager (PowerShell)
   - **Linux**: secret-service (`secret-tool` or GNOME Keyring)

## 🐚 Shell Completion
Enable auto-completion for your shell:
```bash
# Choose your shell
matlas completion bash | sudo tee /etc/bash_completion.d/matlas
matlas completion zsh > ~/.zsh/completions/_matlas
matlas completion fish > ~/.config/fish/completions/matlas.fish
matlas completion powershell > matlas.ps1
```

## 📚 Learn More

| Topic | Documentation |
|-------|---------------|
| 🌐 **Atlas Commands** | [`docs/atlas.md`](docs/atlas.md) |
| 🗄️ **Database Commands** | [`docs/database.md`](docs/database.md) |
| 📋 **Infrastructure Workflows** | [`docs/infra.md`](docs/infra.md) |
| 🔐 **Authentication & Config** | [`docs/auth.md`](docs/auth.md) |
| 📖 **YAML Configuration** | [`docs/yaml-kinds.md`](docs/yaml-kinds.md) |
| 🔍 **Discovery Workflows** | [`docs/discovery.md`](docs/discovery.md) |

## 🛠️ Development

### Commands
```bash
# Run tests
make test                    # See scripts/test/*.sh for details

# Code quality  
make lint                    # Lint code
make fmt                     # Format code

# Generate mocks
make generate-mocks         # Update test mocks

# Installation (for development)
make install                # Install to /usr/local/bin (requires sudo)
make install-user          # Install to ~/.local/bin (no sudo)
make uninstall             # Remove installation
```

### Feature tracking
Create a brief, per-feature summary in `features/` using the provided template. This helps reviewers and users understand what was achieved and where it was wired end-to-end (CLI + YAML ApplyDocument).

```bash
cp features/TEMPLATE.md features/$(date +%F)-<short-slug>.md
```

## ⚠️ Current Limitations
- 🔧 **Advanced Configuration**: Some complex cluster configurations (ReplicationSpecs, AutoScaling) require manual Atlas console setup

## 📄 License
**MIT License** - see [`LICENSE`](LICENSE) for details.

---

<div align="center">

**Built with ❤️ for the MongoDB community**

[⭐ Star this repo](https://github.com/teabranch/matlas-cli) • [🐛 Report issues](https://github.com/teabranch/matlas-cli/issues) • [💡 Request features](https://github.com/teabranch/matlas-cli/issues/new)

</div>
