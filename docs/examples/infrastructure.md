---
layout: default
title: Infrastructure Patterns
parent: Examples
nav_order: 6
description: Complete infrastructure management workflows and patterns
permalink: /examples/infrastructure/
---

# Infrastructure Patterns

Complete infrastructure management workflows demonstrating project-centric configurations, safe operations, and dependency management.

## Project Format Configuration

Complete project configuration with embedded resources:

{% raw %}
```yaml
# Project-format configuration (standalone Project kind)
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: example-project-config
spec:
  name: "Example Project"
  organizationId: "5f1d7f3a9d1e8b1234567890"
  databaseUsers:
    - metadata:
        name: project-user
      username: project-user
      authDatabase: admin
      password: "${PROJECT_USER_PASSWORD}"
      roles:
        - roleName: read
          databaseName: admin
  networkAccess:
    - metadata:
        name: project-network
      cidr: "192.0.2.0/24"
      comment: "project format example"
```
{% endraw %}

## Safe Operations with Preserve Existing

Demonstrates safe operations using `--preserve-existing` flag to protect existing resources:

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: safe-operations-preserve-existing
  labels:
    operation-mode: safe
    preserve-existing: "true"
  annotations:
    safety-notice: "This configuration uses preserve-existing patterns"
    usage: "Apply with --preserve-existing flag"
resources:
  # Cluster that may already exist
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: existing-production-cluster
      labels:
        environment: production
        safety-mode: preserve-existing
    spec:
      projectName: "My Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M30
      clusterType: REPLICASET
      mongodbVersion: "7.0"
      backupEnabled: true

  # User that may already exist - will be updated if different
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: existing-app-user
      labels:
        purpose: application
        safety-mode: preserve-existing
    spec:
      projectName: "My Project"
      username: existing-app-user
      authDatabase: admin
      password: "${SAFE_TEST_PASSWORD}"
      roles:
        - roleName: readWrite
          databaseName: myapp
        - roleName: read
          databaseName: logs

  # Network access that may conflict with existing rules
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: office-access-safe
      labels:
        type: cidr
        safety-mode: preserve-existing
    spec:
      projectName: "My Project"
      cidr: "203.0.113.0/24"
      comment: "Safe operations - office network"

  # New resources that will be created
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: new-service-user
      labels:
        purpose: service-account
        creation-mode: new
    spec:
      projectName: "My Project"
      username: new-service-user
      authDatabase: admin
      password: "${NEW_SERVICE_PASSWORD}"
      roles:
        - roleName: read
          databaseName: analytics
      scopes:
        - name: "existing-production-cluster"
          type: CLUSTER
```
{% endraw %}

## Complete Infrastructure Stack

Comprehensive infrastructure configuration with dependencies and full resource management:

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: complete-infrastructure-stack
  labels:
    environment: production
    stack-type: complete
  annotations:
    description: "Complete production infrastructure stack"
    dependency-order: "clusters -> roles -> users -> network -> search"
resources:
  # Primary production cluster
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: production-primary
      labels:
        tier: primary
        environment: production
    spec:
      projectName: "Production Infrastructure"
      provider: AWS
      region: US_EAST_1
      instanceSize: M40
      clusterType: REPLICASET
      mongodbVersion: "7.0"
      backupEnabled: true
      autoscaling:
        diskGBEnabled: true
        computeEnabled: true
        computeMinInstanceSize: M40
        computeMaxInstanceSize: M80

  # Analytics cluster
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: analytics-cluster
      labels:
        tier: analytics
        environment: production
    spec:
      projectName: "Production Infrastructure"
      provider: AWS
      region: US_WEST_2
      instanceSize: M30
      clusterType: REPLICASET
      mongodbVersion: "7.0"
      backupEnabled: true

  # Custom roles for granular access control
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: application-role
    spec:
      roleName: applicationAccess
      databaseName: appdb
      privileges:
        - actions: ["find", "insert", "update", "remove"]
          resource:
            database: appdb
            collection: users
        - actions: ["find", "insert", "update", "remove"]
          resource:
            database: appdb
            collection: orders
        - actions: ["find"]
          resource:
            database: appdb
            collection: config

  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: analytics-role
    spec:
      roleName: analyticsReader
      databaseName: analytics
      privileges:
        - actions: ["find", "listIndexes"]
          resource:
            database: analytics
        - actions: ["listCollections"]
          resource:
            database: analytics
      inheritedRoles:
        - roleName: read
          databaseName: reference

  # Application users with scoped access
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: application-service
      labels:
        purpose: application
        tier: service
    spec:
      projectName: "Production Infrastructure"
      username: application-service
      authDatabase: admin
      password: "${APPLICATION_SERVICE_PASSWORD}"
      roles:
        - roleName: applicationAccess
          databaseName: appdb
      scopes:
        - name: "production-primary"
          type: CLUSTER

  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: analytics-service
      labels:
        purpose: analytics
        tier: service
    spec:
      projectName: "Production Infrastructure"
      username: analytics-service
      authDatabase: admin
      password: "${ANALYTICS_SERVICE_PASSWORD}"
      roles:
        - roleName: analyticsReader
          databaseName: analytics
      scopes:
        - name: "analytics-cluster"
          type: CLUSTER

  # Administrative users
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: database-admin
      labels:
        purpose: administration
        tier: admin
    spec:
      projectName: "Production Infrastructure"
      username: database-admin
      authDatabase: admin
      password: "${DATABASE_ADMIN_PASSWORD}"
      roles:
        - roleName: dbAdminAnyDatabase
          databaseName: admin
        - roleName: userAdminAnyDatabase
          databaseName: admin

  # Network access configuration
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: production-vpc
    spec:
      projectName: "Production Infrastructure"
      awsSecurityGroup: "sg-prod-app-servers"
      comment: "Production application servers"

  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: analytics-vpc
    spec:
      projectName: "Production Infrastructure"
      awsSecurityGroup: "sg-analytics-servers"
      comment: "Analytics processing servers"

  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: admin-access
    spec:
      projectName: "Production Infrastructure"
      cidr: "10.0.1.0/24"
      comment: "Database administration subnet"

  # Search indexes for application
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: user-search-index
    spec:
      projectName: "Production Infrastructure"
      clusterName: "production-primary"
      databaseName: "appdb"
      collectionName: "users"
      indexName: "user-search"
      indexType: "search"
      definition:
        mappings:
          dynamic: false
          fields:
            name:
              type: "string"
            email:
              type: "string"
            status:
              type: "stringFacet"

  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: order-search-index
    spec:
      projectName: "Production Infrastructure"
      clusterName: "production-primary"
      databaseName: "appdb"
      collectionName: "orders"
      indexName: "order-search"
      indexType: "search"
      definition:
        mappings:
          dynamic: true
```
{% endraw %}

## Environment Variables

Set these environment variables for infrastructure examples:

```bash
# Project format
export PROJECT_USER_PASSWORD='SecureProjectPass123!'

# Safe operations
export SAFE_TEST_PASSWORD='SafeTestPass123!'
export NEW_SERVICE_PASSWORD='NewServicePass123!'

# Complete infrastructure
export APPLICATION_SERVICE_PASSWORD='AppServicePass123!'
export ANALYTICS_SERVICE_PASSWORD='AnalyticsServicePass123!'
export DATABASE_ADMIN_PASSWORD='AdminPass123!'
```

## Usage Workflows

### Safe Operations Workflow

Apply infrastructure safely without disrupting existing resources:

```bash
# Always validate first
matlas infra validate -f safe-operations-preserve-existing.yaml

# Preview changes
matlas infra plan -f safe-operations-preserve-existing.yaml

# Apply with preserve-existing flag
matlas infra apply -f safe-operations-preserve-existing.yaml --preserve-existing

# Verify no existing resources were modified
matlas infra show -f safe-operations-preserve-existing.yaml
```

### Complete Infrastructure Deployment

Deploy full production stack with dependency management:

```bash
# Validate complete configuration
matlas infra validate -f complete-infrastructure-stack.yaml

# Plan deployment with dependency visualization
matlas infra plan -f complete-infrastructure-stack.yaml --show-dependencies

# Deploy incrementally for safety
matlas infra apply -f complete-infrastructure-stack.yaml --preserve-existing --auto-approve

# Monitor deployment status
matlas infra show -f complete-infrastructure-stack.yaml --output table
```

### Project-Centric Management

Use Project kind for simple, embedded resource management:

```bash
# Project format configuration
matlas infra validate -f project-format.yaml
matlas infra apply -f project-format.yaml

# Compare with ApplyDocument approach
matlas infra diff -f project-format.yaml -f equivalent-applydocument.yaml
```

## Key Patterns

### Resource Dependencies

Resources are applied in dependency order:
1. **Clusters** - Infrastructure foundation
2. **DatabaseRoles** - Custom role definitions
3. **DatabaseUsers** - Users referencing roles and clusters
4. **NetworkAccess** - Network security rules
5. **SearchIndexes** - Indexes requiring clusters and databases

### Safety Patterns

- **Preserve Existing**: Use `--preserve-existing` to protect existing resources
- **Incremental Deployment**: Deploy in stages for complex infrastructure
- **Validation First**: Always validate before applying
- **Plan Review**: Use `plan` command to preview changes

### Organizational Patterns

- **Labeling**: Consistent labels for resource organization
- **Naming**: Descriptive names indicating purpose and environment
- **Scoping**: Limit user access to specific clusters
- **Environment Variables**: Secure password management

## Related Examples

- [Discovery]({{ '/examples/discovery/' | relative_url }}) - Convert existing infrastructure to code
- [Clusters]({{ '/examples/clusters/' | relative_url }}) - Cluster configurations
- [Users]({{ '/examples/users/' | relative_url }}) - User management patterns
- [Custom Roles]({{ '/examples/roles/' | relative_url }}) - Role definitions
- [Network Access]({{ '/examples/network/' | relative_url }}) - Network security