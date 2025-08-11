---
layout: page
title: Atlas Commands
description: Manage MongoDB Atlas infrastructure including projects, users, clusters, and networking.
permalink: /atlas/
---

# Atlas Commands

Manage MongoDB Atlas infrastructure including projects, users, clusters, and networking.



---

## Projects

Manage MongoDB Atlas projects within your organization.

### List projects
```bash
matlas atlas projects list [--org-id <org>]
```

### Get project details
```bash
matlas atlas projects get --project-id <id>
```

### Create project
```bash
matlas atlas projects create <name> --org-id <org> [--tag k=v]...
```

### Update project
```bash
matlas atlas projects update --project-id <id> [--name new] [--tag k=v]... [--clear-tags]
```

### Delete project
```bash
matlas atlas projects delete <project-id> [--yes]
```

## Users

Manage database users within Atlas projects.

### List users
```bash
matlas atlas users list --project-id <id> [--page N --limit M --all]
```

### Get user details
```bash
matlas atlas users get <username> --project-id <id> [--database-name admin]
```

### Create user
```bash
matlas atlas users create \
  --project-id <id> \
  --username <username> \
  --database-name admin \
  --roles role@db[,role@db]
```

### Update user
```bash
matlas atlas users update <username> \
  --project-id <id> \
  [--database-name admin] \
  [--password] \
  [--roles ...]
```

### Delete user
```bash
matlas atlas users delete <username> --project-id <id> [--database-name admin] [--yes]
```

## Network access

Configure IP access lists for your Atlas clusters.

### List network access entries
```bash
matlas atlas network list --project-id <id>
```

### Get network access entry
```bash
matlas atlas network get <ip-or-cidr> --project-id <id>
```

### Create network access entry
```bash
# Allow specific IP
matlas atlas network create --project-id <id> --ip-address x.x.x.x [--comment "Description"]

# Allow CIDR block
matlas atlas network create --project-id <id> --cidr-block x.x.x.x/24 [--comment "Description"]

# Allow AWS security group
matlas atlas network create --project-id <id> --aws-security-group sg-xxxxxxxxx [--comment "Description"]
```

### Delete network access entry
```bash
matlas atlas network delete <ip-or-cidr> --project-id <id> [--yes]
```

## Network peering

Network peering enables private connectivity between your Atlas clusters and cloud infrastructure.

### Available commands
```bash
matlas atlas network-peering list --project-id <id>
matlas atlas network-peering get <peering-id> --project-id <id>
matlas atlas network-peering create --project-id <id> [options]
matlas atlas network-peering delete <peering-id> --project-id <id> [--yes]
```

Use `matlas atlas network-peering <command> --help` for detailed flag information.

## Network containers

Network containers define the CIDR blocks for your Atlas clusters in specific cloud regions.

### Available commands
```bash
matlas atlas network-containers list --project-id <id>
matlas atlas network-containers get <container-id> --project-id <id>
matlas atlas network-containers create --project-id <id> [options]
matlas atlas network-containers delete <container-id> --project-id <id> [--yes]
```

Use `matlas atlas network-containers <command> --help` for detailed flag information.

## Clusters

**Note:** Cluster management is primarily handled through the [infrastructure workflows](infra). Use `matlas discover` and `matlas infra` commands for cluster operations.

## Feature availability

**Warning:** The following features are not supported in the current build:
- Search indexes: `matlas atlas search ...`
- VPC endpoints: `matlas atlas vpc-endpoints ...`