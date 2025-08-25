---
layout: default
title: Examples
nav_order: 8
has_children: true
description: Working YAML examples and comprehensive CLI demonstrations
permalink: /examples/
---

# Examples

Working YAML examples for `ApplyDocument` resources used by `matlas infra` and comprehensive demonstrations of CLI functionality.

{: .note }
These examples demonstrate both ApplyDocument (recommended) and standalone YAML approaches. All examples include detailed comments explaining usage patterns and best practices.

## Quick Start

### Environment Variables Setup

Before running examples, set up required environment variables:

```bash
# Basic user passwords
export APP_USER_PASSWORD='StrongPass123!'
export APP_WRITER_PASSWORD='StrongPass123!'
export ANALYTICS_PASSWORD='StrongPass123!'
export DATABASE_ADMIN_PASSWORD='AdminPass123!'
export SERVICE_ACCOUNT_PASSWORD='ServicePass123!'
```

### Basic Usage

```bash
# Validate any example
matlas infra validate -f examples/cluster-basic.yaml

# Preview changes before applying
matlas infra plan -f examples/users-basic.yaml

# Apply configuration safely
matlas infra apply -f examples/safe-operations-preserve-existing.yaml --preserve-existing
```

## Example Categories

### [Discovery Examples]({{ '/examples/discovery/' | relative_url }})
Convert existing Atlas resources to infrastructure-as-code format
- Basic project discovery
- Comprehensive discovery with databases

### [Cluster Examples]({{ '/examples/clusters/' | relative_url }}) 
MongoDB cluster configurations for different environments
- Basic development clusters
- Production clusters with autoscaling
- Multi-region configurations

### [User Management]({{ '/examples/users/' | relative_url }})
Database user and authentication examples
- Basic user creation
- Scoped users for specific clusters
- Password management patterns

### [Custom Roles]({{ '/examples/roles/' | relative_url }})
Granular permission management with custom database roles
- Basic custom role definitions
- Advanced permission patterns
- Role inheritance examples

### [Network Access]({{ '/examples/network/' | relative_url }})
IP allowlisting and network security configurations
- IP address and CIDR rules
- AWS security group integration
- Temporary access patterns

### [Infrastructure Patterns]({{ '/examples/infrastructure/' | relative_url }})
Complete infrastructure management workflows
- Project-centric configurations
- Safe operations with preserve-existing
- Dependency management

### [Search & VPC]({{ '/examples/advanced/' | relative_url }})
Advanced Atlas features
- Atlas Search index configurations
- VPC endpoint setups
- Vector search for AI applications

## Best Practices

### ApplyDocument vs Standalone Files

{: .important }
**ApplyDocument is recommended** for most use cases as it provides dependency management, cross-resource validation, and atomic operations.

```yaml
# ✅ Recommended: ApplyDocument with multiple resources
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: my-infrastructure
resources:
  - kind: Cluster
    # ...
  - kind: DatabaseUser
    # ...
```

```yaml
# ⚠️ Limited use: Standalone for simple scenarios only
apiVersion: matlas.mongodb.com/v1
kind: Cluster
metadata:
  name: simple-cluster
spec:
  # ...
```

### Safety Patterns

- Use `--preserve-existing` flag to protect existing resources
- Always run `matlas infra plan` before `apply`
- Use environment variables for sensitive data
- Test with `--dry-run` for complex changes

### Common Workflows

```bash
# 1. Discovery-driven development
matlas discover --project-id <id> --include-databases --output-file current.yaml

# 2. Modify the discovered configuration
vim current.yaml

# 3. Preview and apply changes
matlas infra diff -f current.yaml
matlas infra apply -f current.yaml --preserve-existing
```

## Related Documentation

- [YAML Kinds Reference]({{ '/yaml-kinds/' | relative_url }}) - Complete reference for all resource types
- [Infrastructure Commands]({{ '/infra/' | relative_url }}) - `plan`, `apply`, `diff`, and `destroy` operations
- [Atlas Commands]({{ '/atlas/' | relative_url }}) - Direct Atlas resource management
- [Database Commands]({{ '/database/' | relative_url }}) - MongoDB database operations

---

For the complete source files, see the [examples directory](https://github.com/teabranch/matlas-cli/tree/main/examples) in the GitHub repository.