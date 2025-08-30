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

1. **ApplyDocument** - Multi-resource containers for complex configurations (**RECOMMENDED**)
2. **Standalone kinds** - Individual resource files for simple use cases

### Recommended Approach: ApplyDocument

**ApplyDocument is the preferred method** for most infrastructure scenarios because it provides:
- **Dependency management**: Automatic resource ordering and validation
- **Atomic operations**: All-or-nothing deployments with rollback
- **Cross-resource validation**: Ensures references between resources are valid
- **Bulk operations**: Deploy entire infrastructure stacks in one command

Use **standalone kinds** only for simple, single-resource scenarios or when integrating with existing workflows.

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

See the [Discovery Examples]({{ '/examples/discovery/' | relative_url }}) for complete examples.

---

## Project

**Purpose**: MongoDB Atlas project configuration  
**Use case**: Project-centric infrastructure management  
**Usage**: Can be used standalone or within ApplyDocument

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

See the [Infrastructure Patterns]({{ '/examples/infrastructure/' | relative_url }}) for project-format examples.

---

## Cluster

**Purpose**: MongoDB cluster configuration  
**Use case**: Database infrastructure provisioning  
**Usage**: Typically used within ApplyDocument for dependency management

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

See the [Cluster Examples]({{ '/examples/clusters/' | relative_url }}) for:
- Basic development clusters
- Production clusters with autoscaling
- Multi-region configurations

---

## DatabaseUser

**Purpose**: Atlas-managed database user configuration  
**Use case**: Centralized user management via Atlas API  
**Usage**: Recommended for use within ApplyDocument for role dependency validation

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

See the [User Management Examples]({{ '/examples/users/' | relative_url }}) for:
- Basic user creation patterns
- Cluster-scoped user access
- Password management workflows


## DatabaseRole

**Purpose**: Custom database role definition  
**Use case**: Granular permission management  
**Usage**: Best used within ApplyDocument with DatabaseUser resources for proper dependency ordering

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

See the [Custom Roles Examples]({{ '/examples/roles/' | relative_url }}) for:
- Basic custom role definitions
- Advanced permission patterns
- Role inheritance examples

---

## NetworkAccess

**Purpose**: Network access rule configuration  
**Use case**: IP allowlisting and security  
**Usage**: Can be standalone for simple rules, or within ApplyDocument for coordinated infrastructure

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

See the [Network Access Examples]({{ '/examples/network/' | relative_url }}) for:
- Basic IP and CIDR configurations
- AWS security group integration
- Temporary access patterns

---

## Usage Patterns

### ApplyDocument for Infrastructure Management (RECOMMENDED)

**ApplyDocument is the recommended approach** for most infrastructure scenarios. Use it for comprehensive infrastructure management:

```bash
# Apply complete infrastructure
matlas infra apply -f infrastructure.yaml

# Plan before applying
matlas infra plan -f infrastructure.yaml

# Show current state
matlas infra show -f infrastructure.yaml
```

### Single Resource Files (Simple Use Cases)

For simple, single-resource scenarios or legacy integrations, you can use individual kind files:

```bash
# Apply a single cluster (only for simple scenarios)
matlas infra apply -f cluster.yaml

# Apply multiple individual files (less efficient than ApplyDocument)
matlas infra apply -f users.yaml -f network.yaml
```

**Note**: Individual resource files lack dependency management and cross-resource validation. ApplyDocument is recommended for production use.

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

All patterns below are recommended for use within **ApplyDocument** for proper dependency management and validation.

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
- [Examples]({{ '/examples/' | relative_url }}) - Working examples for all kinds

---

## SearchIndex

**Purpose**: Atlas Search index configuration  
**Use case**: Full-text search and vector search capabilities  
**Usage**: Recommended within ApplyDocument to ensure cluster dependencies are met

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

### Advanced Search Index with Features

```yaml
apiVersion: matlas.mongodb.com/v1
kind: SearchIndex
metadata:
  name: products-advanced-search
spec:
  projectName: "My Project"
  clusterName: "production-cluster"
  databaseName: "ecommerce"
  collectionName: "products"
  indexName: "products-advanced"
  indexType: "search"
  definition:
    mappings:
      dynamic: false
      fields:
        title:
          type: string
          analyzer: "titleAnalyzer"
        category:
          type: stringFacet
        price:
          type: numberFacet
  # Advanced search features
  analyzers:
    - name: "titleAnalyzer"
      type: "custom"
      charFilters: []
      tokenizer:
        type: "standard"
      tokenFilters:
        - type: "lowercase"
        - type: "stemmer"
          language: "english"
  facets:
    - field: "category"
      type: "string"
      numBuckets: 20
    - field: "price"
      type: "number"
      boundaries: [0, 25, 50, 100, 250, 500]
  autocomplete:
    - field: "title"
      maxEdits: 2
      prefixLength: 1
  highlighting:
    - field: "title"
      maxCharsToExamine: 500000
      maxNumPassages: 3
  synonyms:
    - name: "productSynonyms"
      input: ["laptop", "notebook", "computer"]
      output: "laptop"
      explicit: false
  fuzzySearch:
    - field: "title"
      maxEdits: 2
      prefixLength: 1
      maxExpansions: 50
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

### Optional Advanced Search Fields

- `spec.analyzers`: Custom analyzer configurations
  - `name`: Analyzer name
  - `type`: Analyzer type (standard, keyword, simple, whitespace, language, custom)
  - `charFilters`: Character filters to apply
  - `tokenizer`: Tokenizer configuration
  - `tokenFilters`: Token filters to apply
- `spec.facets`: Faceted search configurations
  - `field`: Field name to facet on
  - `type`: Facet type (string, number, date)
  - `numBuckets`: Maximum number of buckets
  - `boundaries`: Custom bucket boundaries
- `spec.autocomplete`: Autocomplete configurations
  - `field`: Field name for autocomplete
  - `maxEdits`: Maximum edits for fuzzy matching
  - `prefixLength`: Minimum prefix length
- `spec.highlighting`: Search result highlighting
  - `field`: Field name to highlight
  - `maxCharsToExamine`: Maximum characters to examine
  - `maxNumPassages`: Maximum highlighted passages
- `spec.synonyms`: Synonym configurations
  - `name`: Synonym collection name
  - `input`: Array of input terms
  - `output`: Output term
  - `explicit`: Whether synonyms are explicit
- `spec.fuzzySearch`: Fuzzy search configurations
  - `field`: Field name for fuzzy search
  - `maxEdits`: Maximum character edits
  - `prefixLength`: Exact prefix match length
  - `maxExpansions`: Maximum similar terms

### Index Types

- `search`: Full-text search (default)
- `vectorSearch`: Vector/semantic search for AI/ML applications

### Examples

See the [Examples]({{ '/examples/' | relative_url }}) for:
- Basic text search index configurations
- Vector search for AI applications

---

## SearchMetrics

**Purpose**: Search index performance metrics and analytics  
**Use case**: Monitor search index performance, query patterns, and usage  
**Usage**: Read-only operation for retrieving metrics data

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1alpha1
kind: SearchMetrics
metadata:
  name: product-search-metrics
spec:
  projectName: "My Project"
  clusterName: "my-cluster"
  indexName: "products-search"
  timeRange: "24h"
  metrics:
    - query
    - performance
    - usage
```

### Required Fields

- `spec.projectName`: Atlas project name or ID
- `spec.clusterName`: Atlas cluster name

### Optional Fields

- `spec.indexName`: Specific search index name (omit for all indexes)
- `spec.timeRange`: Time range for metrics (`1h`, `6h`, `24h`, `7d`, `30d`)
- `spec.metrics`: Types of metrics to retrieve (`query`, `performance`, `usage`)

---

## SearchOptimization

**Purpose**: Search index optimization analysis and recommendations  
**Use case**: Analyze search indexes for performance improvements  
**Usage**: Analysis operation providing optimization recommendations

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1alpha1
kind: SearchOptimization
metadata:
  name: index-optimization-analysis
spec:
  projectName: "My Project"
  clusterName: "my-cluster"
  indexName: "products-search"
  analyzeAll: true
  categories:
    - performance
    - mappings
    - analyzers
```

### Required Fields

- `spec.projectName`: Atlas project name or ID
- `spec.clusterName`: Atlas cluster name

### Optional Fields

- `spec.indexName`: Specific search index name (omit for all indexes)
- `spec.analyzeAll`: Enable detailed analysis (default: false)
- `spec.categories`: Analysis categories (`performance`, `mappings`, `analyzers`, `facets`, `synonyms`)

---

## SearchQueryValidation

**Purpose**: Search query syntax validation and optimization  
**Use case**: Validate search queries before execution  
**Usage**: Validation operation for query syntax and performance analysis

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1alpha1
kind: SearchQueryValidation
metadata:
  name: query-validation-test
spec:
  projectName: "My Project"
  clusterName: "my-cluster"
  indexName: "products-search"
  testMode: true
  query:
    text:
      query: "wireless headphones"
      path: "title"
  validate:
    - syntax
    - fields
    - performance
```

### Required Fields

- `spec.projectName`: Atlas project name or ID
- `spec.clusterName`: Atlas cluster name
- `spec.indexName`: Search index name
- `spec.query`: Search query object to validate

### Optional Fields

- `spec.testMode`: Enable detailed analysis and recommendations (default: false)
- `spec.validate`: Validation types (`syntax`, `fields`, `performance`)

### Query Examples

#### Text Search Query
```yaml
query:
  text:
    query: "search term"
    path: "fieldName"
```

#### Compound Query
```yaml
query:
  compound:
    must:
      - text:
          query: "required term"
          path: "title"
    should:
      - text:
          query: "optional term"
          path: "description"
    filter:
      - range:
          path: "price"
          gte: 10
          lte: 100
```

#### Vector Search Query
```yaml
query:
  knnBeta:
    vector: [0.1, 0.2, 0.3, 0.4, 0.5]
    path: "embedding"
    k: 10
```

---

## AlertConfiguration

**Purpose**: Atlas alert configuration for monitoring  
**Use case**: Set up monitoring alerts for Atlas resources  
**Usage**: Recommended within ApplyDocument for comprehensive monitoring setup

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: AlertConfiguration
metadata:
  name: high-cpu-alert
spec:
  enabled: true
  eventTypeName: "HOST_CPU_USAGE_PERCENT"
  severityOverride: "HIGH"
  matchers:
    - fieldName: "HOSTNAME_AND_PORT"
      operator: "CONTAINS"
      value: "production"
  notifications:
    - typeName: "EMAIL"
      emailAddress: "alerts@company.com"
      delayMin: 0
      intervalMin: 15
  metricThreshold:
    metricName: "CPU_USAGE_PERCENT"
    operator: "GREATER_THAN"
    threshold: 80.0
    units: "PERCENT"
    mode: "AVERAGE"
```

### Required Fields

- `spec.enabled`: Whether the alert is active
- `spec.eventTypeName`: Type of event to monitor
- `spec.notifications`: Array of notification channels

### Alert Event Types

Common event types include:
- `HOST_CPU_USAGE_PERCENT` - CPU usage monitoring
- `HOST_MEMORY_USAGE_PERCENT` - Memory usage monitoring
- `HOST_DISK_USAGE_PERCENT` - Disk usage monitoring
- `CLUSTER_MONGOS_IS_MISSING` - Missing mongos process
- `CLUSTER_PRIMARY_ELECTED` - Primary election events
- `DATABASE_CONNECTIONS_PERCENT` - Connection usage monitoring

### Notification Channels

**Email notifications**:
```yaml
notifications:
  - typeName: "EMAIL"
    emailAddress: "alerts@company.com"
    delayMin: 0
    intervalMin: 15
```

**Slack notifications**:
```yaml
notifications:
  - typeName: "SLACK"
    apiToken: "${SLACK_TOKEN}"
    channelName: "#alerts"
    username: "Atlas Alert"
```

**PagerDuty notifications**:
```yaml
notifications:
  - typeName: "PAGER_DUTY"
    serviceKey: "${PAGERDUTY_SERVICE_KEY}"
```

**Webhook notifications**:
```yaml
notifications:
  - typeName: "WEBHOOK"
    url: "https://api.company.com/webhooks/atlas"
    secret: "${WEBHOOK_SECRET}"
```

### Matchers

Target specific resources with matchers:

```yaml
matchers:
  - fieldName: "HOSTNAME_AND_PORT"
    operator: "CONTAINS"
    value: "production"
  - fieldName: "REPLICA_SET_NAME"
    operator: "EQUALS"
    value: "atlas-cluster0-shard-0"
```

**Matcher operators**:
- `EQUALS` / `NOT_EQUALS` - Exact matching
- `CONTAINS` / `NOT_CONTAINS` - Substring matching
- `STARTS_WITH` / `ENDS_WITH` - Prefix/suffix matching
- `REGEX` / `NOT_REGEX` - Regular expression matching

### Thresholds

**Metric thresholds** (for performance metrics):
```yaml
metricThreshold:
  metricName: "CPU_USAGE_PERCENT"
  operator: "GREATER_THAN"
  threshold: 80.0
  units: "PERCENT"
  mode: "AVERAGE"
```

**General thresholds** (for non-metric events):
```yaml
threshold:
  operator: "LESS_THAN"
  threshold: 5
  units: "RAW"
```

### Examples

See the [Alert Examples]({{ '/examples/' | relative_url }}) for:
- Basic CPU and memory monitoring
- Multi-channel notification setups
- Complex matcher configurations

---

## Alert (Read-Only)

**Purpose**: Atlas alert status and details  
**Use case**: Monitor active alerts and acknowledgment status  
**Usage**: Read-only resource for alert status information

### Basic Structure

```yaml
apiVersion: matlas.mongodb.com/v1
kind: Alert
metadata:
  name: alert-status
spec:
  projectName: "My Project"
  alertId: "507f1f77bcf86cd799439011"  # Optional: specific alert ID
```

### Required Fields

- `spec.projectName`: Atlas project name or ID

### Optional Fields

- `spec.alertId`: Specific alert ID (omit to list all alerts)

**Note**: Alert resources are read-only and used for monitoring alert status. Alert configurations are managed through the `AlertConfiguration` kind.

---

## VPCEndpoint

**Purpose**: VPC endpoint service configuration  
**Use case**: Private network connectivity to Atlas clusters  
**Usage**: Typically used within ApplyDocument with cluster and network resources

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

See the [Examples]({{ '/examples/' | relative_url }}) for VPC endpoint setup patterns.

**Note**: VPC endpoint implementation creates the Atlas-side service. You'll need to configure the corresponding endpoint in your cloud provider.

---

For working examples of each kind, see the [Examples]({{ '/examples/' | relative_url }}) section with comprehensive YAML configurations and usage patterns.
