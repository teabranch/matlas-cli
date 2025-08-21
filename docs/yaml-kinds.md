---
layout: default
title: Reference
nav_order: 7
has_children: true
description: Reference documentation for YAML kinds, configuration, and development.
permalink: /reference/
---

# Reference Documentation

Complete reference materials for matlas CLI including YAML kinds, configuration options, and development guides.

# YAML Kinds Reference

This reference covers all supported YAML kinds in matlas configuration files. Each kind represents a different type of MongoDB Atlas resource that can be managed declaratively.

---

## Overview

matlas supports two main configuration approaches:

1. **Standalone kinds** - Individual resource files (e.g., `Project`)
2. **ApplyDocument** - Multi-resource containers for complex configurations

### API Versions

All resources support these API versions:
- `matlas.mongodb.com/v1` (recommended)
- `matlas.mongodb.com/v1beta1`
- `matlas.mongodb.com/v1alpha1`

### Common Structure

Every YAML kind follows this basic structure:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: <ResourceKind>
metadata:
  name: <resource-name>
  labels:
    key: value
  annotations:
    key: value
spec:
  # Resource-specific configuration
```

---

## ApplyDocument

**Purpose**: Container for multiple resources with dependency management  
**Use case**: Complex configurations, infrastructure as code

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: my-infrastructure
  labels:
    environment: production
  annotations:
    description: "Complete MongoDB Atlas infrastructure"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: production-cluster
    spec:
      # Cluster configuration
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-user
    spec:
      # User configuration
```

### Features

- **Dependency management**: Resources are applied in dependency order
- **Bulk operations**: Apply multiple resources in one command
- **Validation**: Cross-resource validation and conflict detection
- **Rollback**: Atomic operations with automatic rollback on failure

### Example

See {{ '/examples/discovery-basic.yaml' | relative_url }} for a complete example.

---

## Project

**Purpose**: MongoDB Atlas project configuration  
**Use case**: Project-centric infrastructure management

### Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: example-project
spec:
  name: "My Atlas Project"
  organizationId: "5f1d7f3a9d1e8b1234567890"
  databaseUsers:
    - metadata:
        name: project-user
      username: project-user
      authDatabase: admin
      password: "${USER_PASSWORD}"
      roles:
        - roleName: readWrite
          databaseName: myapp
  networkAccess:
    - metadata:
        name: office-access
      cidr: "192.0.2.0/24"
      comment: "Office network access"
```

### Required Fields

- `spec.name`: Project display name
- `spec.organizationId`: Atlas organization ID

### Optional Fields

- `spec.databaseUsers`: Embedded user configurations
- `spec.networkAccess`: Embedded network access rules

### Example

See {{ '/examples/project-format.yaml' | relative_url }} for a complete example.

---

## Cluster

**Purpose**: MongoDB cluster configuration  
**Use case**: Database infrastructure provisioning

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: Cluster
metadata:
  name: production-cluster
  labels:
    environment: production
spec:
  projectName: "My Project"
  provider: AWS
  region: US_EAST_1
  instanceSize: M30
  clusterType: REPLICASET
  mongodbVersion: "7.0"
  backupEnabled: true
```

### Required Fields

- `spec.projectName`: Target Atlas project name
- `spec.provider`: Cloud provider (`AWS`, `GCP`, `AZURE`)
- `spec.region`: Cloud region
- `spec.instanceSize`: Instance size tier

### Advanced Configuration

```yaml
spec:
  # Auto-scaling
  autoScaling:
    diskGBEnabled: true
    compute:
      enabled: true
      scaleDownEnabled: true
      minInstanceSize: M30
      maxInstanceSize: M40
  
  # Multi-region replication
  replicationSpecs:
    - numShards: 1
      regionConfigs:
        - regionName: US_EAST_1
          electableNodes: 3
          priority: 7
          readOnlyNodes: 0
  
  # Security
  encryption:
    encryptionAtRestProvider: AWS
    awsKmsConfig:
      enabled: true
      customerMasterKeyID: "alias/my-key"
  
  # BI Connector
  biConnector:
    enabled: true
    readPreference: secondary
  
  # Tags
  tags:
    environment: production
    team: platform
    cost-center: engineering
```

### Examples

- {{ '/examples/cluster-basic.yaml' | relative_url }} - Basic cluster
- {{ '/examples/cluster-comprehensive.yaml' | relative_url }} - Production cluster with all features
- {{ '/examples/cluster-multiregion.yaml' | relative_url }} - Multi-region configuration

---

## DatabaseUser

**Purpose**: Atlas-managed database user configuration  
**Use case**: Centralized user management via Atlas API

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: DatabaseUser
metadata:
  name: app-user
  labels:
    purpose: application
spec:
  projectName: "My Project"
  username: app-user
  authDatabase: admin
  password: "${APP_USER_PASSWORD}"
  roles:
    - roleName: readWrite
      databaseName: myapp
    - roleName: read
      databaseName: logs
```

### Required Fields

- `spec.projectName`: Target Atlas project name
- `spec.username`: Database username
- `spec.roles`: Array of database roles

### Role Types

**Built-in MongoDB roles**:
```yaml
roles:
  - roleName: read
    databaseName: myapp
  - roleName: readWrite
    databaseName: myapp
  - roleName: dbAdmin
    databaseName: myapp
  - roleName: userAdmin
    databaseName: myapp
  - roleName: clusterAdmin
    databaseName: admin
  - roleName: readAnyDatabase
    databaseName: admin
  - roleName: readWriteAnyDatabase
    databaseName: admin
  - roleName: userAdminAnyDatabase
    databaseName: admin
  - roleName: dbAdminAnyDatabase
    databaseName: admin
```

**Custom roles** (created via `DatabaseRole` kind):
```yaml
roles:
  - roleName: myCustomRole
    databaseName: myapp
```

### Scoping

Limit user access to specific clusters:

```yaml
spec:
  scopes:
    - name: "production-cluster"
      type: CLUSTER
    - name: "staging-cluster"
      type: CLUSTER
```

### Authentication

- `spec.authDatabase`: Authentication database (default: `admin`)
- `spec.password`: User password (use environment variables)

### Examples

- {{ '/examples/users-basic.yaml' | relative_url }} - Basic users
- {{ '/examples/users-scoped.yaml' | relative_url }} - Cluster-scoped users
- {{ '/examples/user-password-management.yaml' | relative_url }} - Password management

---

## DatabaseDirectUser

**Purpose**: Direct database connection user management  
**Use case**: Database-level operations requiring direct connection

### Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: DatabaseDirectUser
metadata:
  name: direct-user
spec:
  connectionConfig:
    cluster: "production-cluster"
    projectId: "507f1f77bcf86cd799439011"
    # OR use connection string directly:
    # connectionString: "mongodb+srv://..."
    useTempUser: true
    tempUserRole: dbAdmin
  username: direct-user
  password: "${DIRECT_USER_PASSWORD}"
  database: myapp
  roles:
    - roleName: readWrite
      databaseName: myapp
```

### Connection Methods

1. **Atlas cluster reference**:
   ```yaml
   connectionConfig:
     cluster: "my-cluster"
     projectId: "507f1f77bcf86cd799439011"
   ```

2. **Direct connection string**:
   ```yaml
   connectionConfig:
     connectionString: "mongodb+srv://user:pass@cluster.mongodb.net/"
   ```

3. **Temporary user authentication**:
   ```yaml
   connectionConfig:
     cluster: "my-cluster"
     useTempUser: true
     tempUserRole: dbAdmin
   ```

### Required Fields

- `spec.connectionConfig`: Connection configuration
- `spec.username`: Database username
- `spec.password`: User password
- `spec.database`: Target database
- `spec.roles`: Database roles

---

## DatabaseRole

**Purpose**: Custom database role definition  
**Use case**: Granular permission management

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: DatabaseRole
metadata:
  name: custom-app-role
spec:
  roleName: appDataAccess
  databaseName: myapp
  privileges:
    - actions: ["find", "insert", "update", "remove"]
      resource:
        database: myapp
        collection: users
    - actions: ["find"]
      resource:
        database: myapp
        collection: config
  inheritedRoles:
    - roleName: read
      databaseName: reference-data
```

### Required Fields

- `spec.roleName`: Custom role name
- `spec.databaseName`: Database where role is defined

### Privileges

Define granular permissions:

```yaml
privileges:
  # Collection-level permissions
  - actions: ["find", "insert", "update", "remove"]
    resource:
      database: myapp
      collection: users
  
  # Database-level permissions
  - actions: ["listCollections", "listIndexes"]
    resource:
      database: analytics
  
  # Cluster-level permissions
  - actions: ["connPoolStats"]
    resource:
      cluster: true
```

### Common Actions

**Read operations**: `find`, `listCollections`, `listIndexes`  
**Write operations**: `insert`, `update`, `remove`, `createCollection`  
**Admin operations**: `dbAdmin`, `userAdmin`, `createIndex`, `dropCollection`  
**Cluster operations**: `connPoolStats`, `serverStatus`

### Inherited Roles

Inherit permissions from existing roles:

```yaml
inheritedRoles:
  - roleName: read
    databaseName: logs
  - roleName: readWrite
    databaseName: cache
```

### Examples

- {{ '/examples/custom-roles-example.yaml' | relative_url }} - Basic custom roles
- {{ '/examples/custom-roles-comprehensive.yaml' | relative_url }} - Advanced role permissions

---

## NetworkAccess

**Purpose**: Network access rule configuration  
**Use case**: IP allowlisting and security

### IP Address Access

```yaml
apiVersion: matlas.mongodb.com/v1
kind: NetworkAccess
metadata:
  name: single-ip
spec:
  projectName: "My Project"
  ipAddress: "203.0.113.42"
  comment: "Developer workstation"
  deleteAfterDate: "2024-12-31T23:59:59Z"
```

### CIDR Block Access

```yaml
apiVersion: matlas.mongodb.com/v1
kind: NetworkAccess
metadata:
  name: office-network
spec:
  projectName: "My Project"
  cidr: "203.0.113.0/24"
  comment: "Office network range"
```

### AWS Security Group

```yaml
apiVersion: matlas.mongodb.com/v1
kind: NetworkAccess
metadata:
  name: aws-security-group
spec:
  projectName: "My Project"
  awsSecurityGroup: "sg-0abc123def456789"
  comment: "Production AWS security group"
```

### Required Fields

- `spec.projectName`: Target Atlas project name
- One of: `spec.ipAddress`, `spec.cidr`, or `spec.awsSecurityGroup`

### Optional Fields

- `spec.comment`: Description (max 80 characters)
- `spec.deleteAfterDate`: Automatic expiration (ISO 8601 format)

### Examples

- {{ '/examples/network-access.yaml' | relative_url }} - Basic network access
- {{ '/examples/network-variants.yaml' | relative_url }} - All access types

---

## Usage Patterns

### Single Resource Files

For simple configurations, use individual kind files:

```bash
# Apply a single cluster
matlas infra apply -f cluster.yaml

# Apply multiple files
matlas infra apply -f users.yaml -f network.yaml
```

### ApplyDocument for Complex Infrastructure

For comprehensive infrastructure, use ApplyDocument:

```bash
# Apply complete infrastructure
matlas infra apply -f infrastructure.yaml

# Plan before applying
matlas infra plan -f infrastructure.yaml

# Show current state
matlas infra show -f infrastructure.yaml
```

### Environment Variables

Use environment variables for sensitive data:

```bash
export APP_USER_PASSWORD='SecurePassword123!'
export DATABASE_ADMIN_PASSWORD='AdminPassword456!'
matlas infra apply -f users.yaml
```

### Dependency Management

Resources are applied in dependency order:

1. **Projects** (if using Project kind)
2. **Clusters** 
3. **DatabaseRoles** (custom roles)
4. **DatabaseUsers** (references roles and clusters)
5. **NetworkAccess** (independent)

---

## Validation and Best Practices

### Validation Rules

- **API versions** must be supported
- **Resource kinds** must be valid
- **Required fields** must be present
- **Cross-references** must be valid (e.g., cluster names in user scopes)
- **Dependencies** must be resolvable

### Best Practices

1. **Use environment variables** for passwords and sensitive data
2. **Label resources** consistently for organization
3. **Use meaningful names** in metadata
4. **Group related resources** in ApplyDocument
5. **Document configurations** with annotations
6. **Version control** your YAML files
7. **Test configurations** with `matlas infra plan` before applying

### Common Patterns

**Service account setup**:
```yaml
# 1. Create custom role
kind: DatabaseRole
spec:
  roleName: serviceRole
  # ... role definition

# 2. Create user with custom role
kind: DatabaseUser
spec:
  roles:
    - roleName: serviceRole
      databaseName: myapp
```

**Environment promotion**:
```yaml
# Use labels for environment management
metadata:
  labels:
    environment: "{{ .Values.environment }}"
    team: platform
```

**Security-first approach**:
```yaml
# Scope users to specific clusters
spec:
  scopes:
    - name: "production-cluster"
      type: CLUSTER

# Use temporary access for network rules
spec:
  deleteAfterDate: "2024-12-31T23:59:59Z"
```

---

## Related Documentation

- {{ '/infra/' | relative_url }} - Infrastructure commands (`apply`, `plan`, `diff`)
- {{ '/atlas/' | relative_url }} - Atlas resource management
- {{ '/database/' | relative_url }} - Database operations
- {{ '/auth/' | relative_url }} - Authentication and configuration
- [Examples directory]({{ '/examples/README.html' | relative_url }}) - Working examples for all kinds

---

## SearchIndex

**Purpose**: Atlas Search index configuration  
**Use case**: Full-text search and vector search capabilities

### Basic Search Index

```yaml
apiVersion: matlas.mongodb.com/v1
kind: SearchIndex
metadata:
  name: movies-text-search
spec:
  projectName: "My Project"
  clusterName: "production-cluster"
  databaseName: "sample_mflix"
  collectionName: "movies"
  indexName: "default"
  indexType: "search"
  definition:
    mappings:
      dynamic: true
```

### Vector Search Index

```yaml
apiVersion: matlas.mongodb.com/v1
kind: SearchIndex
metadata:
  name: movie-plot-embeddings
spec:
  projectName: "My Project"
  clusterName: "production-cluster"  
  databaseName: "sample_mflix"
  collectionName: "movies"
  indexName: "plot_vector_index"
  indexType: "vectorSearch"
  definition:
    fields:
      - type: "vector"
        path: "plot_embedding"
        numDimensions: 1536
        similarity: "cosine"
```

### Required Fields

- `spec.projectName`: Target Atlas project name
- `spec.clusterName`: Target cluster name
- `spec.databaseName`: Database name
- `spec.collectionName`: Collection name
- `spec.indexName`: Search index name
- `spec.definition`: Index definition object

### Index Types

- `search`: Full-text search (default)
- `vectorSearch`: Vector/semantic search for AI/ML applications

### Examples

- {{ '/examples/search-basic.yaml' | relative_url }} - Basic text search index
- {{ '/examples/search-vector.yaml' | relative_url }} - Vector search for AI applications

---

## VPCEndpoint

**Purpose**: VPC endpoint service configuration  
**Use case**: Private network connectivity to Atlas clusters

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: VPCEndpoint
metadata:
  name: production-vpc-endpoint
spec:
  projectName: "My Project"
  cloudProvider: "AWS"
  region: "us-east-1"
```

### Required Fields

- `spec.projectName`: Target Atlas project name
- `spec.cloudProvider`: Cloud provider (`AWS`, `AZURE`, `GCP`)
- `spec.region`: Cloud provider region

### Optional Fields

- `spec.endpointId`: Specific endpoint ID (for existing endpoints)

### Cloud Provider Support

**AWS**: Full support for VPC endpoints  
**Azure**: Private Link endpoints  
**GCP**: Private Service Connect

### Examples

- {{ '/examples/vpc-endpoint-basic.yaml' | relative_url }} - Basic VPC endpoint setup

**Note**: VPC endpoint implementation creates the Atlas-side service. You'll need to configure the corresponding endpoint in your cloud provider.

---

For working examples of each kind, see the `examples/` directory in the repository.
