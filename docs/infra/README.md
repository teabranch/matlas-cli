# matlas infra - Declarative Configuration Management

The `matlas infra` command provides declarative configuration management for MongoDB Atlas resources. It allows you to define your Atlas infrastructure as code using YAML configuration files and apply those configurations to create, update, or delete resources.

## Table of Contents

- [Overview](#overview)
- [Configuration Format](#configuration-format)
- [Commands](#commands)
- [Environment Variable Substitution](#environment-variable-substitution)
- [Output Formats](#output-formats)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

The apply system follows a declarative approach where you describe the desired state of your Atlas resources, and the CLI ensures that state is achieved. This includes:

- **Idempotent operations**: Running the same configuration multiple times produces the same result
- **Diff detection**: Shows exactly what will change before applying
- **Dependency resolution**: Automatically handles resource dependencies
- **Template processing**: Support for environment variable substitution
- **Rollback capabilities**: Safely undo changes when needed

## Configuration Format

Configuration files use YAML format with a specific schema:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: my-project
  labels:
    environment: production
    team: backend
spec:
  name: "My Production Project"
  organizationId: "507f1f77bcf86cd799439011"
  clusters:
    - metadata:
        name: prod-cluster
      provider: AWS
      region: US_EAST_1
      instanceSize: M30
      diskSizeGB: 100
      backupEnabled: true
      mongodbVersion: "7.0"
  databaseUsers:
    - metadata:
        name: app-user
      username: "myapp"
      databaseName: "admin" 
      roles:
        - roleName: "readWrite"
          databaseName: "mydb"
  networkAccess:
    - metadata:
        name: office-access
      cidr: "203.0.113.0/24"
      comment: "Office network access"
```

### Supported Resource Types

- **Project**: Atlas projects with associated clusters, users, and network rules
- **Cluster**: Individual cluster configurations
- **DatabaseUser**: Database user accounts with roles
- **NetworkAccess**: IP access list entries

### API Versions

- `matlas.mongodb.com/v1alpha1`: Early preview (experimental features)
- `matlas.mongodb.com/v1beta1`: Beta release (stable API, may have minor changes)
- `matlas.mongodb.com/v1`: Stable release (recommended for production)

## Commands

### apply

Apply configuration files to Atlas resources.

```bash
# Apply a single configuration file
matlas infra -f config.yaml

# Apply multiple files with glob pattern
matlas infra -f "configs/*.yaml"

# Apply from stdin
cat config.yaml | matlas infra -f -

# Dry run to preview changes
matlas infra -f config.yaml --dry-run

# Auto-approve without prompts
matlas infra -f config.yaml --auto-approve

# Watch mode for continuous reconciliation
matlas infra -f config.yaml --watch
```

#### Flags

- `-f, --file`: Configuration files to apply (supports glob patterns and stdin with '-')
- `--dry-run`: Show what would be applied without making changes
- `--dry-run-mode`: Dry run validation mode: quick, thorough, detailed
- `-o, --output`: Output format: table, json, yaml, summary, detailed
- `--auto-approve`: Skip interactive approval prompts
- `--timeout`: Timeout for the apply operation (default: 30m)
- `-v, --verbose`: Enable verbose output
- `--no-color`: Disable colored output
- `--project-id`: Atlas project ID (overrides config)
- `--strict-env`: Fail on undefined environment variables
- `--watch`: Enable watch mode for continuous reconciliation
- `--watch-interval`: Interval between reconciliation checks (default: 5m)

### plan

Generate and save execution plans without applying them.

```bash
# Generate a plan
matlas infra plan -f config.yaml

# Save plan to file
matlas infra plan -f config.yaml --output-file plan.json

# Detailed plan output
matlas infra plan -f config.yaml --plan-mode detailed
```

#### Flags

- `-f, --file`: Configuration files to plan
- `--output-file`: Save plan to file (format determined by extension)
- `--plan-mode`: Plan detail level: quick, detailed, comprehensive
- `-o, --output`: Output format: table, json, yaml, summary
- `--timeout`: Timeout for plan generation

### show

Display current state of Atlas resources.

```bash
# Show all resources in a project
matlas infra show --project-id 507f1f77bcf86cd799439011

# Show specific resource type
matlas infra show --project-id 507f1f77bcf86cd799439011 --resource-type clusters

# Show with sensitive information
matlas infra show --project-id 507f1f77bcf86cd799439011 --show-secrets
```

#### Flags

- `--project-id`: Atlas project ID (required)
- `--resource-type`: Filter by resource type: clusters, users, network
- `--resource-name`: Show specific resource by name
- `--show-secrets`: Include sensitive information in output
- `--show-metadata`: Include resource metadata and annotations
- `-o, --output`: Output format: table, json, yaml, summary

### validate

Validate configuration files without applying them.

```bash
# Validate a configuration file
matlas infra validate -f config.yaml

# Validate multiple files
matlas infra validate -f "configs/*.yaml"

# Strict validation with detailed output
matlas infra validate -f config.yaml --strict --verbose
```

#### Flags

- `-f, --file`: Configuration files to validate
- `--strict`: Enable strict validation mode
- `--check-quotes`: Validate Atlas resource quotas
- `--lint`: Enable linting for best practices
- `-o, --output`: Output format: table, json, yaml

### diff

Show differences between current and desired state.

```bash
# Show differences
matlas infra diff -f config.yaml

# Detailed diff with field-level changes
matlas infra diff -f config.yaml --diff-mode detailed

# Save diff to file
matlas infra diff -f config.yaml --output-file diff.txt
```

#### Flags

- `-f, --file`: Configuration files to diff
- `--diff-mode`: Diff detail level: summary, detailed, unified
- `--output-file`: Save diff to file
- `--ignore-metadata`: Ignore metadata changes in diff
- `--no-color`: Disable colored diff output

### destroy

Delete all resources defined in configuration files or discovered in Atlas projects.

```bash
# Destroy resources defined in config files (with confirmation)
matlas infra destroy -f config.yaml

# Auto-approve destruction
matlas infra destroy -f config.yaml --auto-approve

# Dry run to see what would be destroyed
matlas infra destroy -f config.yaml --dry-run

# Destroy ALL discovered resources (ignore config files)
matlas infra destroy --discovery-only --project-id PROJECT_ID

# Destroy only specific resource type
matlas infra destroy -f config.yaml --target clusters

# Force destroy with dependency issues
matlas infra destroy -f config.yaml --force
```

The destroy command supports two modes of operation:

1. **Configuration-based destroy** (default): Only destroys resources that are defined in your configuration files AND exist in Atlas
2. **Discovery-only destroy**: Destroys ALL resources discovered in the specified Atlas project, regardless of configuration files

#### Flags

- `-f, --file`: Configuration files defining resources to destroy
- `--discovery-only`: Destroy all discovered resources, regardless of configuration files (requires --project-id)
- `--project-id`: Atlas project ID (required when using --discovery-only)
- `--target`: Only destroy specific resource type: clusters, users, network-access
- `--auto-approve`: Skip interactive confirmation
- `--dry-run`: Show what would be destroyed without doing it
- `--force`: Force deletion even if resources have dependencies
- `--delete-snapshots`: Also delete any cluster snapshots
- `--timeout`: Timeout for destroy operation (default: 30m)

#### Resource Deletion Order

The destroy command follows a specific order to prevent dependency issues:

1. **Database Users** - Deleted first to remove database access
2. **Network Access Lists** - Deleted second to remove network connectivity  
3. **Clusters** - Deleted last after dependencies are cleared

This ordering prevents race conditions where clusters might be deleted before users or network access, which could leave orphaned resources.

## Environment Variable Substitution

Configuration files support environment variable substitution with various patterns:

### Basic Substitution

```yaml
spec:
  name: "${PROJECT_NAME}"
  organizationId: "${ATLAS_ORG_ID}"
```

### Default Values

```yaml
spec:
  # Use default if variable is unset
  mongodbVersion: "${MONGODB_VERSION:-7.0}"
  diskSizeGB: ${DISK_SIZE:-100}
```

### Conditional Values

```yaml
spec:
  # Only include if variable is set
  biConnector:
    enabled: "${ENABLE_BI_CONNECTOR:+true}"
  
  # Error if variable is not set
  organizationId: "${ATLAS_ORG_ID:?Organization ID is required}"
```

### Nested Substitution

```yaml
spec:
  name: "${PROJECT_PREFIX}_${ENVIRONMENT}_cluster"
  tags:
    environment: "${ENVIRONMENT}"
    owner: "${TEAM_${ENVIRONMENT}_OWNER}"
```

### Escape Sequences

```yaml
spec:
  # Literal ${} text (escaped)
  comment: "Use \${VARIABLE} syntax for templating"
```

## Output Formats

### Table Format (Default)

Human-readable table format with aligned columns:

```
RESOURCE        NAME           STATUS    MESSAGE
Cluster         prod-cluster   Ready     Cluster is healthy
DatabaseUser    app-user       Ready     User configured successfully
NetworkAccess   office-access  Ready     Access rule active
```

### JSON Format

Machine-readable JSON for automation:

```json
{
  "resources": [
    {
      "kind": "Cluster",
      "name": "prod-cluster",
      "status": "Ready",
      "spec": {...}
    }
  ]
}
```

### YAML Format

YAML output compatible with input format:

```yaml
resources:
  - kind: Cluster
    metadata:
      name: prod-cluster
    status:
      phase: Ready
```

### Summary Format

High-level overview:

```
Plan Summary:
  Resources to create: 2
  Resources to update: 1
  Resources to delete: 0
  Estimated time: 5m30s
```

## Examples

See the [examples directory](../../examples/apply/) for complete configuration examples:

- [Basic Setup](../../examples/apply/basic-setup.yaml): Simple project with cluster and user
- [Multi-Environment](../../examples/apply/multi-env/): Development, staging, and production configurations
- [Advanced Dependencies](../../examples/apply/advanced-deps.yaml): Complex resource dependencies
- [Template Variables](../../examples/apply/templates/): Environment variable substitution examples
- [Disaster Recovery](../../examples/apply/disaster-recovery.yaml): High-availability configuration

## Best Practices

### File Organization

- Use separate files for different environments
- Group related resources in the same file
- Use descriptive filenames (e.g., `prod-clusters.yaml`, `staging-users.yaml`)

### Resource Naming

- Use consistent naming conventions
- Include environment prefix (e.g., `prod-`, `staging-`)
- Keep names descriptive but concise

### Repository Naming Conventions

- CLI command directories under `cmd/atlas` use hyphenated names for better UX discoverability (e.g., `network-peering`, `vpc-endpoints`, `network-containers`).
- Service packages under `internal/services/atlas` use underscore-separated Go package/file names matching Atlas domains (e.g., `network_peering.go`, `vpc_endpoints.go`, `network_containers.go`).
- Rationale: hyphens fit CLI ergonomics while underscores align with Go naming and Atlas SDK resources. This dual convention improves discoverability across the CLI and code.

### Security

- Never commit sensitive values to configuration files
- Use environment variables for secrets
- Apply proper role-based access controls
- Regularly rotate database user passwords
\- Encryption at rest:
  - Project-level KMS enablement is supported via `spec.clusters[].encryption` for the provider flag and via project update APIs for key configuration when available.
  - Current release maps `encryption.encryptionAtRestProvider` to the cluster model and exposes project encryption operations via `internal/services/atlas.EncryptionService`.
  - Documentation of exact provider fields is in `docs/infra/configuration-schema.md`. Unsupported combinations will be validated and surfaced clearly.

### Performance

- Use batch operations for multiple resources
- Enable caching for large configurations
- Consider parallel processing for independent resources

### Monitoring

- Use labels and annotations for resource tracking
- Implement health checks for critical resources
- Set up alerting for configuration drift

## Troubleshooting

### Destroy Operation Issues

#### Race Conditions

**Problem**: `HTTP 404 Not Found` errors during destroy operations, especially for network access entries.

**Cause**: Resources being deleted in the wrong order, causing dependencies to be removed before dependents.

**Solution**: The CLI now automatically handles dependency ordering, but if you encounter issues:

```bash
# Use dry-run to verify the deletion plan
matlas infra destroy -f config.yaml --dry-run

# Use discovery-only mode for comprehensive cleanup
matlas infra destroy --discovery-only --project-id PROJECT_ID --dry-run
```

#### Missing User Deletions

**Problem**: Database users exist in Atlas but aren't being destroyed.

**Cause**: Users exist in Atlas but aren't defined in your configuration files.

**Solutions**:

1. **Add users to configuration** (recommended for managed infrastructure):
   ```yaml
   databaseUsers:
     - metadata:
         name: orphaned-user
       username: "orphaned-user"
       # ... other user configuration
   ```

2. **Use discovery-only mode** (for cleanup):
   ```bash
   # Preview what would be destroyed
   matlas infra destroy --discovery-only --project-id PROJECT_ID --dry-run
   
   # Destroy all discovered resources
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

#### Partial Failures

**Problem**: Some resources fail to delete while others succeed.

**Troubleshooting**:

1. **Check resource dependencies**:
   ```bash
   # Show current project state
   matlas infra show --project-id PROJECT_ID
   ```

2. **Use targeted deletion**:
   ```bash
   # Delete only specific resource types
   matlas infra destroy -f config.yaml --target users
   matlas infra destroy -f config.yaml --target network-access
   matlas infra destroy -f config.yaml --target clusters
   ```

3. **Force deletion** (use with caution):
   ```bash
   matlas infra destroy -f config.yaml --force
   ```

### Common Error Messages

#### `HTTP 404 Not Found - ATLAS_NETWORK_PERMISSION_ENTRY_NOT_FOUND`

This indicates a network access entry was already deleted (often by cluster deletion). This is now handled automatically and treated as success.

#### `project ID not available for X deletion`

Ensure your configuration includes a valid project ID or use the `--project-id` flag:

```bash
matlas infra destroy -f config.yaml --project-id YOUR_PROJECT_ID
```

#### `at least one configuration file must be specified`

When not using discovery-only mode, you must provide configuration files:

```bash
# Either provide config files
matlas infra destroy -f config.yaml

# Or use discovery-only mode
matlas infra destroy --discovery-only --project-id PROJECT_ID
```

### Recovery Procedures

#### Stuck Destroy Operations

1. **Check Atlas console** for resource status
2. **Use Atlas CLI directly** for manual cleanup if needed
3. **Re-run with discovery-only** to clean up remaining resources:
   ```bash
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

#### Orphaned Resources

1. **Identify orphaned resources**:
   ```bash
   matlas infra show --project-id PROJECT_ID
   ```

2. **Clean up with discovery-only**:
   ```bash
   matlas infra destroy --discovery-only --project-id PROJECT_ID --dry-run
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ``` 