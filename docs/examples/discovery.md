---
layout: default
title: Discovery Examples
parent: Examples
nav_order: 1
description: Convert existing Atlas resources to infrastructure-as-code format
---

# Discovery Examples

Convert existing MongoDB Atlas resources to infrastructure-as-code format using the discovery feature.

## discovery-basic.yaml

Basic discovered project converted to ApplyDocument format with cluster, user, and network access.

{% raw %}
```yaml
# Basic discovered project converted to ApplyDocument format
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: discovered-infrastructure
  labels:
    source: discovery
    environment: production
  annotations:
    discovery.matlas.mongodb.com/source-project: "64abc123def456789"
    discovery.matlas.mongodb.com/discovered-at: "2024-01-15T10:30:00Z"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: production-cluster
      labels:
        environment: production
        source: discovery
    spec:
      projectName: "My Production Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M30
      clusterType: REPLICASET
      mongodbVersion: "7.0"
      backupEnabled: true
      diskSizeGB: 40
      tags:
        environment: production
        team: platform
        cost-center: engineering

  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-user
      labels:
        purpose: application
        source: discovery
    spec:
      projectName: "My Production Project"
      username: app-user
      authDatabase: admin
      password: "${APP_USER_PASSWORD}"
      roles:
        - roleName: readWrite
          databaseName: myapp
        - roleName: read
          databaseName: logs
      scopes:
        - name: "production-cluster"
          type: CLUSTER

  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: office-access
      labels:
        source: discovery
    spec:
      projectName: "My Production Project"
      cidr: "203.0.113.0/24"
      comment: "Office network access"
```
{% endraw %}

## discovery-with-databases.yaml  

Comprehensive discovery including database-level resources (databases, collections, indexes, custom roles).

{% raw %}
```yaml
# Comprehensive discovery with database-level resources
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: complete-discovered-infrastructure
  labels:
    source: discovery
    includes-databases: "true"
  annotations:
    discovery.matlas.mongodb.com/source-project: "64abc123def456789"
    discovery.matlas.mongodb.com/cluster-scanned: "production-cluster"
    discovery.matlas.mongodb.com/discovered-at: "2024-01-15T10:30:00Z"
resources:
  # Infrastructure resources
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: production-cluster
    spec:
      projectName: "My Production Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M30
      clusterType: REPLICASET
      mongodbVersion: "7.0"
      backupEnabled: true

  # Custom database roles discovered
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: app-data-access
      labels:
        source: discovery
        database: myapp
    spec:
      roleName: appDataAccess
      databaseName: myapp
      privileges:
        - actions: ["find", "insert", "update", "remove"]
          resource:
            database: myapp
            collection: users
        - actions: ["find", "insert", "update", "remove"] 
          resource:
            database: myapp
            collection: products
        - actions: ["find"]
          resource:
            database: myapp
            collection: config
      inheritedRoles:
        - roleName: read
          databaseName: analytics

  # Users referencing discovered custom roles
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-service-user
    spec:
      projectName: "My Production Project"
      username: app-service-user
      authDatabase: admin
      password: "${SERVICE_ACCOUNT_PASSWORD}"
      roles:
        - roleName: appDataAccess
          databaseName: myapp
        - roleName: read
          databaseName: logs
      scopes:
        - name: "production-cluster"
          type: CLUSTER

  # Search indexes discovered
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: product-search
      labels:
        source: discovery
    spec:
      projectName: "My Production Project"
      clusterName: "production-cluster"
      databaseName: "myapp"
      collectionName: "products"
      indexName: "product-search-index"
      indexType: "search"
      definition:
        mappings:
          dynamic: false
          fields:
            name:
              type: "string"
            description:
              type: "string"
            category:
              type: "stringFacet"
            price:
              type: "number"

  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: office-network
    spec:
      projectName: "My Production Project"
      cidr: "203.0.113.0/24"
      comment: "Office network - discovered"
```  
{% endraw %}

## Usage

### Basic Discovery

Discover existing project resources:

```bash
# Basic discovery
matlas discover --project-id <project-id> --output-file discovered.yaml

# Convert to ApplyDocument format
matlas discover --project-id <project-id> --convert-to-apply --output-file infrastructure.yaml
```

### Comprehensive Discovery with Databases

Include database-level resources in discovery:

```bash
# Full discovery including database resources
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --use-temp-user \
  --convert-to-apply \
  --output-file complete-infrastructure.yaml
```

### Working with Discovered Resources

```bash
# Validate discovered configuration
matlas infra validate -f discovered.yaml

# Preview what would be applied
matlas infra plan -f discovered.yaml

# Apply safely (won't modify existing resources)
matlas infra apply -f discovered.yaml --preserve-existing
```

## Key Benefits

- **Infrastructure as Code**: Convert existing resources to version-controlled YAML
- **Database Resources**: Include collections, indexes, and custom roles
- **Dependency Awareness**: Proper resource ordering and references
- **Safe Application**: Use `--preserve-existing` to protect existing resources
- **Complete Documentation**: Automatically document your current infrastructure

## Related Examples

- [Infrastructure Patterns]({{ '/examples/infrastructure/' | relative_url }}) - Building on discovered resources
- [Custom Roles]({{ '/examples/roles/' | relative_url }}) - Managing discovered custom roles
- [Users]({{ '/examples/users/' | relative_url }}) - User management patterns