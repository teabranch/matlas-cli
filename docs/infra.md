Infrastructure workflows

File formats
- DiscoveredProject: output from `matlas discover`
- ApplyDocument / ApplyConfig: inputs for `matlas infra`

Discover
- Enumerate Atlas resources for a project
- Flags: `--project-id`, `--include [project,clusters,users,network,databases]`, `--exclude`, `--mask-secrets`, `--include-databases`, `--use-temp-user`, `--resource-type`, `--resource-name`, `--convert-to-apply`, `--output [yaml|json]`, `-o file`
- Example: `matlas discover --project-id <id> --include-databases --use-temp-user --output yaml -o project.yaml`

Plan
- Generate an execution plan without applying
- Example: `matlas infra plan -f config.yaml --output table`

Diff
- Show differences between desired config and current Atlas state
- Example: `matlas infra diff -f config.yaml --detailed --show-context 3`

Apply
- Reconcile desired state
- Key flags: `--dry-run`, `--dry-run-mode [quick|thorough|detailed]`, `--auto-approve`, `--preserve-existing`, `--watch`
- Example: `matlas infra -f config.yaml --dry-run --output summary`

Show
- Display current Atlas project state
- Example: `matlas infra show --project-id <id> --output table`

Destroy
- Delete resources defined in config or everything discovered (`--discovery-only`)
- Confirmation required unless `--auto-approve`/`--force`
- Example: `matlas infra destroy -f config.yaml` or `matlas infra destroy --discovery-only --project-id <id>`

