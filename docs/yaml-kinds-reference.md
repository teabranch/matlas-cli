---
layout: default
title: YAML Kinds
nav_order: 1
parent: Reference
description: Complete reference for all supported YAML kinds in matlas configuration files.
permalink: /reference/yaml-kinds/
---

# YAML Kinds Reference

This reference covers all supported YAML kinds in matlas configuration files. Each kind represents a different type of MongoDB Atlas resource that can be managed declaratively.

## Supported Kinds

| Kind | Description | API Version |
|------|-------------|-------------|
| `Project` | MongoDB Atlas project configuration | `v1` |
| `Cluster` | Atlas cluster (database deployment) | `v1` |
| `DatabaseUser` | Atlas database user | `v1` |
| `DatabaseRole` | Custom database role (direct MongoDB connection required) | `v1` |
| `NetworkAccess` | IP access list entry | `v1` |
| `SearchIndex` | Atlas Search index configuration | `v1` |
| `VPCEndpoint` | Private endpoint for VPC peering | `v1` |
| `ApplyDocument` | Multi-resource document containing multiple kinds | `v1` |

## Common Metadata Fields

All resources share common metadata fields:

```yaml
metadata:
  name: "resource-name"          # Required: unique identifier
  labels:                       # Optional: key-value labels
    environment: "production"
    team: "platform"
  annotations:                  # Optional: extended metadata
    description: "Resource description"
    owner: "team@company.com"
  deletionPolicy: "Delete"      # Optional: Delete|Retain|Snapshot
```

## Project Kind

```yaml
apiVersion: v1
kind: Project
metadata:
  name: my-project
spec:
  name: "My Production Project"
  organizationId: "5e2211c17a3e5a48f5497de3"
  tags:
    environment: production
    cost-center: engineering
  # Resources can be embedded in project spec
  clusters: []
  databaseUsers: []
  networkAccess: []
```

## Cluster Kind

```yaml
apiVersion: v1
kind: Cluster
metadata:
  name: production-cluster
spec:
  projectName: "my-project"     # Reference to project
  provider: "AWS"               # AWS, GCP, or AZURE
  region: "us-east-1"
  instanceSize: "M30"
  diskSizeGB: 40
  backupEnabled: true
  mongodbVersion: "7.0"
  clusterType: "REPLICASET"
  tags:
    purpose: "production-workload"
```

## DatabaseUser Kind

```yaml
apiVersion: v1
kind: DatabaseUser
metadata:
  name: app-user
spec:
  projectName: "my-project"
  username: "application-user"
  password: "secure-password"
  authDatabase: "admin"
  roles:
    - roleName: "readWrite"
      databaseName: "myapp"
    - roleName: "read"
      databaseName: "analytics"
  scopes:
    - name: "production-cluster"
      type: "CLUSTER"
```

## NetworkAccess Kind

```yaml
apiVersion: v1
kind: NetworkAccess
metadata:
  name: office-network
spec:
  projectName: "my-project"
  ipAddress: "203.0.113.0/24"   # CIDR notation supported
  comment: "Office network access"
  deleteAfterDate: "2024-12-31T23:59:59Z"  # Optional expiration
```

## SearchIndex Kind

```yaml
apiVersion: v1
kind: SearchIndex
metadata:
  name: products-search
spec:
  projectName: "my-project"
  clusterName: "production-cluster"
  databaseName: "ecommerce"
  collectionName: "products"
  indexName: "default"
  indexType: "search"           # "search" or "vectorSearch"
  definition:
    mappings:
      dynamic: true
      fields:
        title:
          type: "string"
          analyzer: "standard"
```

## VPCEndpoint Kind

```yaml
apiVersion: v1
kind: VPCEndpoint
metadata:
  name: vpc-endpoint
spec:
  projectName: "my-project"
  cloudProvider: "AWS"          # AWS, AZURE, or GCP
  region: "us-east-1"
  endpointId: "vpce-1234567890abcdef0"  # Set after creation
```

## ApplyDocument Kind

Multi-resource document for managing related resources together:

```yaml
apiVersion: v1
kind: ApplyDocument
metadata:
  name: production-setup
resources:
  - apiVersion: v1
    kind: Cluster
    metadata:
      name: prod-cluster
    spec:
      # ... cluster configuration
  
  - apiVersion: v1
    kind: DatabaseUser
    metadata:
      name: app-user
    spec:
      # ... user configuration
      dependsOn:
        - prod-cluster
```

## Validation Rules

- **Names**: Must be lowercase, alphanumeric, with hyphens/underscores allowed
- **References**: `projectName` fields must reference existing projects
- **Dependencies**: Resources can specify `dependsOn` for ordering
- **Immutable Fields**: Some fields cannot be changed after creation (varies by resource type)
- **Required Fields**: Each kind has specific required fields documented above

## Best Practices

1. **Naming**: Use descriptive, consistent naming conventions
2. **Labels**: Tag resources for organization and cost tracking
3. **Dependencies**: Use explicit `dependsOn` when order matters
4. **Secrets**: Never commit passwords or API keys to version control
5. **Validation**: Use `matlas infra plan` to validate before applying changes