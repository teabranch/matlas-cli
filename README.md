matlas-cli

Unified CLI for MongoDB Atlas and MongoDB, written in Go.

What you can do:
- Atlas: list/get/create/update/delete projects, clusters, users, network access, peering, network containers
- Databases: list/create/delete databases, collections, indexes via connection string or Atlas cluster reference
- Infra: discover current state, plan/diff/apply/destroy via declarative YAML

Install
- Go 1.22+
- Build: `make build` or `go build -o bin/matlas ./...`

Authenticate
- Set `ATLAS_PUB_KEY` and `ATLAS_API_KEY`
- Optional: `ATLAS_PROJECT_ID`, `ATLAS_ORG_ID`
- YAML: `~/.matlas/config.yaml` supports keys `output`, `timeout`, `projectId`, `orgId`, `apiKey`, `publicKey`
- Flags also exist: `--api-key`, `--pub-key`, `--project-id`, `--org-id`

Root flags
- `-o, --output`: table|text|json|yaml
- `--timeout`: e.g. 30s, 1m (default 30s)
- `-v/--verbose`, `-q/--quiet`, `--log-format`
- `--config` to point to a config file

Quick start
- List projects: `matlas atlas projects list` (use `--org-id` to filter)
- Get project: `matlas atlas projects get --project-id <id>`
- List users: `matlas atlas users list --project-id <id>`
- List network access: `matlas atlas network list --project-id <id>`

Database commands
- List DBs: `matlas database list --connection-string "mongodb+srv://..."`
- Or via Atlas: `matlas database list --cluster <name> --project-id <id> [--use-temp-user]`
- Collections: `matlas database collections list --connection-string ... --database mydb`
- Indexes: `matlas database collections indexes list --connection-string ... --database mydb --collection mycoll`

Declarative workflows (infra)
- Discover: `matlas discover --project-id <id> [-o out.yaml] [--include-databases] [--convert-to-apply]`
- Plan: `matlas infra plan -f config.yaml`
- Diff: `matlas infra diff -f config.yaml`
- Apply: `matlas infra -f config.yaml [--dry-run] [--auto-approve] [--preserve-existing]`
- Show current state: `matlas infra show --project-id <id>`
- Destroy: `matlas infra destroy -f config.yaml` or `--discovery-only --project-id <id>`

Configuration precedence
1) Built-in defaults
2) YAML file: `~/.matlas/config.yaml` or `--config`/`ATLAS_CONFIG_FILE`
3) Env vars with `ATLAS_` prefix (e.g., `ATLAS_OUTPUT`, `ATLAS_TIMEOUT`, `ATLAS_PROJECT_ID`)
4) Command flags

Credentials resolution
- Uses flags/YAML first, then `ATLAS_API_KEY` and `ATLAS_PUB_KEY`, then macOS Keychain fallback

Shell completion
- Enable: `matlas completion [bash|zsh|fish|powershell]`

Not yet supported in this build
- Atlas Search (hidden commands exist; return unsupported error)
- VPC Endpoints (hidden)

Development
- Tests: `make test` (see scripts/test/*.sh)
- Lint/format: `make lint`, `make fmt`
- Generate mocks: `make generate-mocks`

License
MIT. See `LICENSE`.
