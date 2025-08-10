---
title: Atlas commands
nav_order: 3
---

# Atlas commands

Projects
- List: `matlas atlas projects list [--org-id <org>]`
- Get: `matlas atlas projects get --project-id <id>`
- Create: `matlas atlas projects create <name> --org-id <org> [--tag k=v]...`
- Update: `matlas atlas projects update --project-id <id> [--name new] [--tag k=v]... [--clear-tags]`
- Delete: `matlas atlas projects delete <project-id> [--yes]`

Users
- List: `matlas atlas users list --project-id <id> [--page N --limit M --all]`
- Get: `matlas atlas users get <username> --project-id <id> [--database-name admin]`
- Create: `matlas atlas users create --project-id <id> --username u --database-name admin --roles role@db[,role@db]`
- Update: `matlas atlas users update <username> --project-id <id> [--database-name admin] [--password] [--roles ...]`
- Delete: `matlas atlas users delete <username> --project-id <id> [--database-name admin] [--yes]`

Network access
- List: `matlas atlas network list --project-id <id>`
- Get: `matlas atlas network get <ip-or-cidr> --project-id <id>`
- Create: `matlas atlas network create --project-id <id> [--ip-address x.x.x.x | --cidr-block x/x | --aws-security-group sg-...] [--comment]`
- Delete: `matlas atlas network delete <ip-or-cidr> --project-id <id> [--yes]`

Network peering
- List/Get/Create/Delete with `matlas atlas network-peering ...` (see `--help` for required flags)

Network containers
- List/Get/Create/Delete with `matlas atlas network-containers ...` (see `--help`)

Unsupported/hidden in this build
- `matlas atlas search ...`
- `matlas atlas vpc-endpoints ...`

