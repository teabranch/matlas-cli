## matlas — one CLI for Atlas and MongoDB

Matlas is a unified, Go-powered CLI that lets you manage MongoDB Atlas and work with your MongoDB databases from the same tool. Think of it as the friendly bridge between the Atlas SDK and the MongoDB Go Driver — so you can spin up projects and clusters, tweak network access, and then hop straight into database tasks like listing collections or inspecting indexes, without changing tools or mental models.

### Why we built it
- **Context switching hurts**: Cloud management lives in one world (Atlas APIs), while day‑to‑day database tasks live in another (drivers and shells). We wanted both, together.
- **A single mental model**: One set of flags, one config, one output style for both Atlas and database operations.
- **Grease the path**: From “create a cluster” to “inspect collections” in seconds — no hunting for another CLI.

### Inspired by Terraform and kubectl: meet Discover
We love how Terraform plans changes before applying, and how kubectl lets you declare desired state. Matlas combines those vibes:

- **Discover**: Snapshot your Atlas org/project (and optionally databases) into clean YAML:
  ```bash
  matlas discover --project-id <id> -o atlas.yaml --include-databases
  ```
- **Plan/Diff**: Treat that YAML as your desired state, compare it to reality:
  ```bash
  matlas infra plan -f atlas.yaml
  matlas infra diff -f atlas.yaml
  ```
- **Apply (dry-run first)**: Roll changes out, Terraform‑style:
  ```bash
  matlas infra -f atlas.yaml --dry-run
  matlas infra -f atlas.yaml --auto-approve
  ```

Matlas doesn’t try to be Terraform or kubectl. It borrows the best ideas so Atlas work feels safe, reviewable, and repeatable — and it keeps database tasks close at hand.

## What you can do
- **Atlas**: list/get/create/update/delete projects, clusters, users, network access, peering, and network containers
- **Databases**: list/create/delete databases, collections, and indexes — either via a connection string or by referencing an Atlas cluster
- **Infra**: discover current state, plan/diff/apply/destroy via declarative YAML

## Install
- **Requirements**: Go 1.22+
- **Build**:
  ```bash
  make build
  # or
  go build -o bin/matlas ./...
  ```

## Authenticate
- **Env vars**: set `ATLAS_PUB_KEY` and `ATLAS_API_KEY`
- **Optional**: `ATLAS_PROJECT_ID`, `ATLAS_ORG_ID`
- **YAML config**: `~/.matlas/config.yaml` supports keys `output`, `timeout`, `projectId`, `orgId`, `apiKey`, `publicKey`
- **Flags** also exist: `--api-key`, `--pub-key`, `--project-id`, `--org-id`

## Quick start
- **List projects**:
  ```bash
  matlas atlas projects list --org-id <id>
  ```
- **Get a project**:
  ```bash
  matlas atlas projects get --project-id <id>
  ```
- **List users**:
  ```bash
  matlas atlas users list --project-id <id>
  ```
- **List network access**:
  ```bash
  matlas atlas network list --project-id <id>
  ```

### Database commands
- **List databases (connection string)**:
  ```bash
  matlas database list --connection-string "mongodb+srv://..."
  ```
- **List databases (via Atlas cluster)**:
  ```bash
  matlas database list --cluster <name> --project-id <id> [--use-temp-user]
  ```
- **Collections**:
  ```bash
  matlas database collections list --connection-string ... --database mydb
  ```
- **Indexes**:
  ```bash
  matlas database collections indexes list --connection-string ... --database mydb --collection mycoll
  ```

### Declarative workflows (infra)
- **Discover**:
  ```bash
  matlas discover --project-id <id> [-o out.yaml] [--include-databases] [--convert-to-apply]
  ```
- **Plan/Diff/Apply**:
  ```bash
  matlas infra plan -f config.yaml
  matlas infra diff -f config.yaml
  matlas infra -f config.yaml [--dry-run] [--auto-approve] [--preserve-existing]
  ```
- **Show current state**:
  ```bash
  matlas infra show --project-id <id>
  ```
- **Destroy**:
  ```bash
  matlas infra destroy -f config.yaml
  # or discovery-only
  matlas infra destroy --discovery-only --project-id <id>
  ```

## Configuration precedence
1. Built‑in defaults
2. YAML file: `~/.matlas/config.yaml` or `--config` / `ATLAS_CONFIG_FILE`
3. Env vars with `ATLAS_` prefix (e.g., `ATLAS_OUTPUT`, `ATLAS_TIMEOUT`, `ATLAS_PROJECT_ID`)
4. Command flags

### Credentials resolution
- Uses flags/YAML first, then `ATLAS_API_KEY` and `ATLAS_PUB_KEY`, then macOS Keychain fallback.

## Shell completion
```bash
matlas completion [bash|zsh|fish|powershell]
```

## Learn more
- Atlas commands: see `docs/atlas.md`
- Database commands: see `docs/database.md`
- Infra workflows: see `docs/infra.md`
- Auth and config: see `docs/auth.md`

## Development
- **Tests**: `make test` (see `scripts/test/*.sh`)
- **Lint/format**: `make lint`, `make fmt`
- **Generate mocks**: `make generate-mocks`

## Not yet supported (in this build)
- Atlas Search (hidden commands exist; return unsupported error)
- VPC Endpoints (hidden)

## License
MIT. See `LICENSE`.
