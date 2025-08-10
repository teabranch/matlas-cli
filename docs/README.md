matlas-cli docs

Topics
- Authentication and configuration
- Atlas commands overview
- Database commands overview
- Infra (discover/plan/diff/apply/destroy)
- Output formats and pagination
- Troubleshooting

Authentication and configuration
- Env: `ATLAS_PUB_KEY`, `ATLAS_API_KEY`, optional `ATLAS_PROJECT_ID`, `ATLAS_ORG_ID`
- File: `~/.matlas/config.yaml` with keys: `output`, `timeout`, `projectId`, `orgId`, `apiKey`, `publicKey`
- Flags override env and file

Atlas commands
- Projects: `matlas atlas projects [list|get|create|update|delete]`
- Users: `matlas atlas users [list|get|create|update|delete] --project-id <id>`
- Network access: `matlas atlas network [list|get|create|delete] --project-id <id>`
- Network peering: `matlas atlas network-peering [list|get|create|delete] --project-id <id>`
- Network containers: `matlas atlas network-containers [list|get|create|delete] --project-id <id>`
- Hidden/unsupported: `atlas search`, `atlas vpc-endpoints`

Database commands
- Source via connection string or `--cluster <name> --project-id <id>`
- Databases: `matlas database [list|create|delete]`
- Collections: `matlas database collections [list|create|delete]`
- Indexes: `matlas database collections indexes [list|create|delete]`

Infra workflows
- Discover: enumerate Atlas project into a DiscoveredProject; can include database/collection enumeration and mask secrets. Example:
  `matlas discover --project-id <id> --output yaml --include-databases --use-temp-user`
- Convert to ApplyDocument: `--convert-to-apply` (pipe to `matlas infra`)
- Plan: `matlas infra plan -f <file>`
- Diff: `matlas infra diff -f <file> [--detailed --show-context 3]`
- Apply: `matlas infra -f <file> [--dry-run|--auto-approve|--preserve-existing]`
- Show: `matlas infra show --project-id <id>`
- Destroy: `matlas infra destroy -f <file>` or `--discovery-only --project-id <id>`

Output and pagination
- Output: `-o table|text|json|yaml`
- Pagination flags (where supported): `--page`, `--limit`, `--all`

Troubleshooting
- Unauthorized: verify `ATLAS_PUB_KEY`/`ATLAS_API_KEY` or flags; ensure roles
- Timeouts: increase with `--timeout`
- Validation errors: check flag values; use `--verbose` for detail
- Network: confirm connectivity; Atlas IP access lists

