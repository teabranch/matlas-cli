---
layout: page
title: Discovery
description: Discover and export existing MongoDB Atlas resources for infrastructure-as-code workflows.
permalink: /discovery/
---

# Discovery

Discover and export existing MongoDB Atlas resources for infrastructure-as-code workflows.

---

## Overview

The discovery feature allows you to:
- **Enumerate existing Atlas resources** (projects, clusters, users, network access)
- **Include database-level resources** (databases, collections, indexes, custom roles)
- **Convert to ApplyDocument format** for infrastructure-as-code workflows
- **Filter and select specific resources** by type or name
- **Cache discovery results** for improved performance
- **Support multiple output formats** (YAML, JSON)

Discovery is the foundation for GitOps workflows and infrastructure migration.

---

## Basic Usage

### Discover project resources
```bash
# Basic project discovery
matlas discover --project-id <project-id>

# Save to file
matlas discover --project-id <project-id> --output-file project.yaml
```

### Include database resources
```bash
# Include databases, collections, and indexes
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --use-temp-user \
  --output-file complete-project.yaml
```

### Convert to ApplyDocument
```bash
# Convert discovered resources to ApplyDocument format
matlas discover \
  --project-id <project-id> \
  --convert-to-apply \
  --mask-secrets \
  --output-file infrastructure.yaml
```

---

## Resource Selection

### Include specific resources
```bash
# Include only specific resource types
matlas discover --project-id <project-id> --include clusters,users,network

# Available types: project,clusters,users,network,databases
```

### Exclude specific resources
```bash
# Exclude network access and databases
matlas discover --project-id <project-id> --exclude network,databases
```

### Filter by resource name
```bash
# Filter clusters by name pattern
matlas discover \
  --project-id <project-id> \
  --resource-type clusters \
  --resource-name "prod-*"

# Discover specific user
matlas discover \
  --project-id <project-id> \
  --resource-type user \
  --resource-name "my-service-user"
```

---

## Database Discovery

### Include database resources
```bash
# Discover databases, collections, and indexes
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --use-temp-user
```

### Database authentication methods

**Temporary user (recommended):**
```bash
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --use-temp-user \
  --temp-user-database myapp
```

**Manual credentials:**
```bash
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --mongo-username myuser \
  --mongo-password mypass
```

**Direct connection string:**
```bash
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --mongo-uri "mongodb+srv://user:pass@cluster.mongodb.net/"
```

---

## Output Formats

### YAML output (default)
```bash
matlas discover --project-id <project-id> --output yaml
```

### JSON output
```bash
matlas discover --project-id <project-id> --output json --output-file project.json
```

### ApplyDocument conversion
```bash
# Convert to infrastructure-as-code format
matlas discover \
  --project-id <project-id> \
  --convert-to-apply \
  --output-file infrastructure.yaml
```

---

## Performance and Caching

### Enable caching (default)
```bash
# Discovery results are cached for improved performance
matlas discover --project-id <project-id> --cache-stats
```

### Disable caching
```bash
# Force fresh discovery without cache
matlas discover --project-id <project-id> --no-cache
```

### Parallel discovery
```bash
# Enable parallel resource discovery
matlas discover \
  --project-id <project-id> \
  --parallel \
  --max-concurrency 5
```

---

## Advanced Options

### Security and secrets
```bash
# Mask sensitive values in output
matlas discover \
  --project-id <project-id> \
  --mask-secrets \
  --convert-to-apply
```

### Preserve existing resources
```bash
# When applying discovered configuration, preserve resources not in the file
matlas discover \
  --project-id <project-id> \
  --convert-to-apply \
  --output-file config.yaml

# Later, apply with preservation
matlas infra apply -f config.yaml --preserve-existing
```

---

## Discovery Flags Reference

| Flag | Description |
|:-----|:------------|
| `--project-id` | Atlas project ID to discover |
| `--include` | Resource types to include (project,clusters,users,network,databases) |
| `--exclude` | Resource types to exclude |
| `--include-databases` | Include database-level resources (databases, collections, indexes) |
| `--resource-type` | Filter by specific resource type |
| `--resource-name` | Filter by resource name pattern |
| `--convert-to-apply` | Convert to ApplyDocument format |
| `--mask-secrets` | Hide sensitive values in output |
| `--output` | Output format (yaml, json) |
| `--output-file, -o` | Save output to file |
| `--use-temp-user` | Create temporary user for database access |
| `--temp-user-database` | Scope temporary user to specific database |
| `--mongo-uri` | Override MongoDB connection for database enumeration |
| `--mongo-username` | Username for database enumeration |
| `--mongo-password` | Password for database enumeration |
| `--no-cache` | Disable discovery caching |
| `--cache-stats` | Print cache statistics |
| `--parallel` | Enable parallel resource discovery |
| `--max-concurrency` | Maximum concurrent API calls |
| `--verbose` | Enable verbose output for debugging |

---

## Incremental Discovery Workflows

### Baseline and change detection
```bash
# 1. Create baseline
matlas discover \
  --project-id <project-id> \
  --convert-to-apply \
  --output-file baseline.yaml

# 2. Make changes via Atlas UI or other tools

# 3. Discover current state
matlas discover \
  --project-id <project-id> \
  --convert-to-apply \
  --output-file current.yaml

# 4. Compare changes
matlas infra diff -f baseline.yaml --compare-with current.yaml
```

### Resource lifecycle tracking
```bash
# 1. Discover initial state
matlas discover --project-id <project-id> --convert-to-apply -o initial.yaml

# 2. Add resources via ApplyDocument
matlas infra apply -f new-resources.yaml --preserve-existing

# 3. Verify resources are discoverable
matlas discover --project-id <project-id> --resource-type user --resource-name "new-user"

# 4. Capture updated state
matlas discover --project-id <project-id> --convert-to-apply -o updated.yaml
```

---

## Discovery Output Structure

### DiscoveredProject format
```yaml
# Raw discovery output - read-only snapshot
version: "v1"
kind: DiscoveredProject
metadata:
  discoveredAt: "2024-01-27T10:30:00Z"
  projectId: "507f1f77bcf86cd799439011"
spec:
  project:
    name: "My Project"
    orgId: "507f1f77bcf86cd799439012"
  clusters:
    - name: "main-cluster"
      provider: "AWS"
      region: "US_EAST_1"
      tier: "M10"
  users:
    - username: "app-user"
      authDatabase: "admin"
      roles:
        - roleName: "readWrite"
          databaseName: "myapp"
```

### ApplyDocument format
```yaml
# Converted format for infrastructure-as-code
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: "discovered-infrastructure"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: main-cluster
    spec:
      projectName: "My Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M10
      
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-user
    spec:
      projectName: "My Project"
      username: app-user
      authDatabase: admin
      password: "${APP_USER_PASSWORD}"  # Environment variable
      roles:
        - roleName: readWrite
          databaseName: myapp
```

---

## Integration with Infrastructure Commands

### Complete GitOps workflow
```bash
# 1. Discover current state
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --convert-to-apply \
  --mask-secrets \
  --output-file infrastructure.yaml

# 2. Edit configuration
vim infrastructure.yaml

# 3. Preview changes
matlas infra diff -f infrastructure.yaml --detailed

# 4. Apply changes with preservation
matlas infra apply -f infrastructure.yaml --preserve-existing --auto-approve

# 5. Verify final state
matlas discover --project-id <project-id> --convert-to-apply -o final-state.yaml
```

### Resource-specific workflows
```bash
# Discover and manage clusters only
matlas discover --project-id <project-id> --include clusters -o clusters.yaml
matlas infra plan -f clusters.yaml
matlas infra apply -f clusters.yaml --preserve-existing

# Discover and manage users only
matlas discover --project-id <project-id> --include users -o users.yaml
matlas infra plan -f users.yaml
matlas infra apply -f users.yaml --preserve-existing
```

---

## Best Practices

### Security
- Use `--mask-secrets` when sharing discovery output
- Store credentials as environment variables in ApplyDocuments
- Use `--use-temp-user` for database discovery instead of permanent credentials

### Performance
- Enable `--parallel` for large projects with many resources
- Use `--cache-stats` to monitor discovery performance
- Filter with `--include` or `--exclude` to limit discovery scope

### Version Control
- Commit ApplyDocument outputs to Git for change tracking
- Use descriptive filenames: `infrastructure-prod.yaml`, `users-staging.yaml`
- Document any manual modifications to discovered configurations

### Migration
- Start with `--preserve-existing` to safely transition to infrastructure-as-code
- Test discovery and apply workflows in non-production environments first
- Use incremental discovery to validate resource lifecycle management

---

## Troubleshooting

### Common Issues

**Discovery fails with authentication errors:**
```bash
# Verify Atlas credentials
export ATLAS_PUB_KEY="your-public-key"
export ATLAS_API_KEY="your-private-key"

# Test basic connectivity
matlas atlas projects list
```

**Database discovery fails:**
```bash
# Use temporary user instead of manual credentials
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --use-temp-user \
  --verbose
```

**Large projects discovery is slow:**
```bash
# Enable parallel discovery and caching
matlas discover \
  --project-id <project-id> \
  --parallel \
  --max-concurrency 10 \
  --cache-stats
```

### Debug Mode
```bash
# Enable verbose output for troubleshooting
matlas discover \
  --project-id <project-id> \
  --verbose \
  --cache-stats \
  --output-file debug-discovery.yaml
```

---

## Examples

See the [examples directory](https://github.com/teabranch/matlas-cli/tree/main/examples) for working discovery and ApplyDocument configurations.
