---
layout: page
title: Infrastructure Workflows
description: Terraform-inspired workflows for managing MongoDB Atlas infrastructure as code.
permalink: /infra/
---

# Infrastructure Workflows

Terraform-inspired workflows for managing MongoDB Atlas infrastructure as code.



---

## Overview

Matlas provides infrastructure-as-code workflows inspired by Terraform and kubectl:

1. **Discover** → Enumerate current Atlas resources
2. **Plan/Diff** → Preview changes before applying
3. **Apply** → Reconcile desired state
4. **Show** → Display current state
5. **Destroy** → Clean up resources

## File formats

| Format | Description | Usage |
|:-------|:------------|:------|
| **DiscoveredProject** | Output from `matlas discover` | Read-only snapshot of current state |
| **ApplyDocument** | Input for `matlas infra` commands | Desired state configuration |

---

## Discover

Enumerate Atlas resources for a project and optionally convert to ApplyDocument format.

### Basic discovery
```bash
matlas discover --project-id <project-id> --output yaml
```

### Include database resources
```bash
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --use-temp-user \
  --output yaml \
  -o project.yaml
```

### Selective discovery
```bash
# Include specific resource types
matlas discover --project-id <project-id> --include project,clusters,users

# Exclude specific resource types
matlas discover --project-id <project-id> --exclude network,databases

# Filter by resource name
matlas discover --project-id <project-id> --resource-type clusters --resource-name "prod-*"
```

### Convert to ApplyDocument
```bash
matlas discover \
  --project-id <project-id> \
  --convert-to-apply \
  --mask-secrets \
  --output yaml \
  -o config.yaml
```

### Discovery flags

| Flag | Description |
|:-----|:------------|
| `--project-id` | Atlas project ID to discover |
| `--include` | Resource types to include (project,clusters,users,network,databases) |
| `--exclude` | Resource types to exclude |
| `--mask-secrets` | Hide sensitive values in output |
| `--include-databases` | Include database-level resources |
| `--use-temp-user` | Create temporary user for database access |
| `--resource-type` | Filter by specific resource type |
| `--resource-name` | Filter by resource name pattern |
| `--convert-to-apply` | Convert to ApplyDocument format |
| `--output` | Output format (yaml, json, table) |
| `-o` | Save to file |

---

## Plan

Generate an execution plan showing what changes would be made without applying them.

```bash
matlas infra plan -f config.yaml --output table
```

The plan shows:
- Resources to be created (+)
- Resources to be updated (~)
- Resources to be deleted (-)
- Resources that will remain unchanged

---

## Diff

Show detailed differences between desired configuration and current Atlas state.

### Basic diff
```bash
matlas infra diff -f config.yaml
```

### Detailed diff with context
```bash
matlas infra diff -f config.yaml --detailed --show-context 3
```

### Diff options

| Flag | Description |
|:-----|:------------|
| `--detailed` | Show detailed field-level changes |
| `--show-context N` | Show N lines of context around changes |
| `--ignore-order` | Ignore array element ordering |

---

## Apply

Reconcile the desired state defined in your configuration.

### Dry run (recommended first)
```bash
# Quick dry run - fast validation
matlas infra apply -f config.yaml --dry-run --dry-run-mode quick

# Thorough dry run - validates API calls
matlas infra apply -f config.yaml --dry-run --dry-run-mode thorough

# Detailed dry run - shows full execution plan
matlas infra apply -f config.yaml --dry-run --dry-run-mode detailed
```

### Apply changes
```bash
# Interactive apply (prompts for confirmation)
matlas infra apply -f config.yaml

# Auto-approve (skip confirmation)
matlas infra apply -f config.yaml --auto-approve

# Watch progress in real-time
matlas infra apply -f config.yaml --watch
```

### Apply flags

| Flag | Description |
|:-----|:------------|
| `--dry-run` | Show what would happen without making changes |
| `--dry-run-mode` | Dry run depth (quick, thorough, detailed) |
| `--auto-approve` | Skip interactive confirmation |
| `--preserve-existing` | Keep resources not defined in config |
| `--watch` | Show real-time progress |
| `--output` | Output format (table, summary, json) |

---

## Show

Display the current state of Atlas project resources.

```bash
# Show all resources in a project
matlas infra show --project-id <project-id> --output table

# Show specific resource types
matlas infra show --project-id <project-id> --resource-type clusters

# Output as YAML for inspection
matlas infra show --project-id <project-id> --output yaml
```

---

## Destroy

Delete resources defined in configuration or discovered from a project.

### Destroy from configuration
```bash
# Interactive destroy (prompts for confirmation)
matlas infra destroy -f config.yaml

# Auto-approve destroy
matlas infra destroy -f config.yaml --auto-approve

# Force destroy (skip safety checks)
matlas infra destroy -f config.yaml --force
```

### Destroy discovered resources
```bash
# Destroy everything in a project
matlas infra destroy --discovery-only --project-id <project-id>

# Destroy with confirmation
matlas infra destroy --discovery-only --project-id <project-id> --auto-approve
```

**Warning:** Destroy is permanent. Always run with `--dry-run` first to preview what will be deleted.

### Destroy flags

| Flag | Description |
|:-----|:------------|
| `--discovery-only` | Destroy all discovered resources in project |
| `--auto-approve` | Skip confirmation prompts |
| `--force` | Skip safety checks |
| `--dry-run` | Preview what would be destroyed |

---

## Complete workflow example

```bash
# 1. Discover current state
matlas discover \
  --project-id abc123 \
  --include-databases \
  --use-temp-user \
  --convert-to-apply \
  --output yaml \
  -o infrastructure.yaml

# 2. Edit the configuration
vim infrastructure.yaml

# 3. Preview changes
matlas infra diff -f infrastructure.yaml --detailed

# 4. Dry run to validate
matlas infra apply -f infrastructure.yaml --dry-run --dry-run-mode thorough

# 5. Apply changes
matlas infra apply -f infrastructure.yaml --watch

# 6. Verify the result
matlas infra show --project-id abc123 --output table
```

## Configuration file structure

A typical ApplyDocument includes:

```yaml
apiVersion: v1
kind: ApplyDocument
metadata:
  projectId: "507f1f77bcf86cd799439011"
  name: "my-infrastructure"
spec:
  project:
    name: "My Project"
    tags:
      environment: "production"
  clusters:
    - name: "main-cluster"
      tier: "M10"
      provider: "AWS"
      region: "US_EAST_1"
  users:
    - username: "app-user"
      databaseName: "admin"
      roles:
        - role: "readWrite"
          database: "myapp"
  networkAccess:
    - ipAddress: "203.0.113.0/24"
      comment: "Office network"
```