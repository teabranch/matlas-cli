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
| 🗄️ **Databases** | List/create/delete databases, collections, and indexes — either via connection string or Atlas cluster reference |
| 📋 **Infra** | Discover current state, plan/diff/apply/destroy via declarative YAML |

## 📦 Install

### Prerequisites
- 🐹 **Go 1.22+** required

### Download from GitHub Releases
Download the archive for your OS/architecture from the Releases page, extract, and place `matlas` in your PATH.

### Build from source
```bash
# Quick build
make build

# Or manually
go build -o bin/matlas ./...
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

# List network access rules
matlas atlas network list --project-id <id>
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
3. macOS Keychain (fallback)

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
```

### Feature tracking
Create a brief, per-feature summary in `features/` using the provided template. This helps reviewers and users understand what was achieved and where it was wired end-to-end (CLI + YAML ApplyDocument).

```bash
cp features/TEMPLATE.md features/$(date +%F)-<short-slug>.md
```

## ⚠️ Current Limitations
- 🔍 **Atlas Search**: Commands exist but return unsupported errors
- 🔗 **VPC Endpoints**: Hidden in current build

## 📄 License
**MIT License** - see [`LICENSE`](LICENSE) for details.

---

<div align="center">

**Built with ❤️ for the MongoDB community**

[⭐ Star this repo](https://github.com/mongodb/matlas-cli) • [🐛 Report issues](https://github.com/mongodb/matlas-cli/issues) • [💡 Request features](https://github.com/mongodb/matlas-cli/issues/new)

</div>
