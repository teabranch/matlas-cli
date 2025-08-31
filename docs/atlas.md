---
layout: default
title: Atlas Commands
nav_order: 3
has_children: false
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

Manage Atlas database users within Atlas projects.

**Note**: These are Atlas-managed database users created via the Atlas Admin API. They are assigned built-in MongoDB roles and managed centrally at the project level. For database-specific users with custom roles, use `matlas database users` commands.

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
# Create user (password will be prompted)
matlas atlas users create \
  --project-id <id> \
  --username <username> \
  --database-name admin \
  --roles role@db[,role@db]

# Create user and display password
matlas atlas users create \
  --project-id <id> \
  --username <username> \
  --database-name admin \
  --roles role@db[,role@db] \
  --show-password
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

Manage MongoDB Atlas clusters directly via CLI commands.

### List clusters
```bash
matlas atlas clusters list --project-id <id>
```

### Get cluster details
```bash
matlas atlas clusters describe <cluster-name> --project-id <id>
```

### Create cluster
```bash
# Basic cluster creation
matlas atlas clusters create <cluster-name> --project-id <id> --tier M10 --provider AWS --region US_EAST_1

# Create cluster with backup enabled
matlas atlas clusters create <cluster-name> --project-id <id> --tier M10 --provider AWS --region US_EAST_1 --backup

# Create cluster with advanced options
matlas atlas clusters create <cluster-name> \
  --project-id <id> \
  --tier M30 \
  --provider AWS \
  --region US_EAST_1 \
  --disk-size 40 \
  --mongodb-version 7.0 \
  --backup \
  --tag environment=production \
  --tag team=backend
```

### Update cluster configuration
```bash
# Update cluster tier
matlas atlas clusters update <cluster-name> --project-id <id> --tier M20

# Update disk size
matlas atlas clusters update <cluster-name> --project-id <id> --disk-size 80

# Enable/disable backup
matlas atlas clusters update <cluster-name> --project-id <id> --backup

# Enable Point-in-Time Recovery (requires backup to be enabled first)
matlas atlas clusters update <cluster-name> --project-id <id> --pit

# Update multiple settings
matlas atlas clusters update <cluster-name> \
  --project-id <id> \
  --tier M40 \
  --disk-size 100 \
  --backup \
  --pit \
  --tag owner=platform-team
```

### Delete cluster
```bash
matlas atlas clusters delete <cluster-name> --project-id <id> [--yes]
```

### Backup Features

**Important**: Point-in-Time Recovery cannot be enabled during cluster creation. Use the following workflow:

```bash
# ❌ This will fail
matlas atlas clusters create my-cluster --pit

# ✅ Correct workflow
# Step 1: Create cluster with backup
matlas atlas clusters create my-cluster --project-id <id> --backup --tier M10 --provider AWS --region US_EAST_1

# Step 2: Wait for cluster to be ready
matlas atlas clusters describe my-cluster --project-id <id>

# Step 3: Enable Point-in-Time Recovery
matlas atlas clusters update my-cluster --project-id <id> --pit
```

**Backup Features:**
- **Continuous Backup** (`--backup`): Automated snapshots and restore capabilities
- **Point-in-Time Recovery** (`--pit`): Recovery to any specific moment in time (requires backup)
- **Cross-Region Backup**: Use multi-region cluster configurations (see YAML examples)

**Note:** For complex cluster configurations with multi-region setups, use [infrastructure workflows](/infra/) with YAML configurations.

## Atlas Search

Atlas Search provides full-text search capabilities for your MongoDB collections.

### Basic Search Index Management

The CLI provides basic CRUD operations for search indexes:

```bash
# List all search indexes in a cluster
matlas atlas search list --project-id <project-id> --cluster <cluster-name>

# List search indexes for a specific collection
matlas atlas search list --project-id <project-id> --cluster <cluster-name> \
  --database sample_mflix --collection movies

# Get search index details by name
matlas atlas search get --project-id <project-id> --cluster <cluster-name> --name default

# Get search index details by ID
matlas atlas search get --project-id <project-id> --cluster <cluster-name> --index-id <index-id>

# Create a basic search index with dynamic mapping
matlas atlas search create \
  --project-id <project-id> \
  --cluster <cluster-name> \
  --database sample_mflix \
  --collection movies \
  --name default

# Create a vector search index (for AI/ML use cases)
matlas atlas search create \
  --project-id <project-id> \
  --cluster <cluster-name> \
  --database sample_mflix \
  --collection movies \
  --name plot_vector_index \
  --type vectorSearch

# Update search index from definition file
matlas atlas search update --project-id <project-id> --cluster <cluster-name> \
  --index-id <index-id> --index-file updated-definition.json

# Delete search index by name (with confirmation)
matlas atlas search delete --project-id <project-id> --cluster <cluster-name> --name default

# Delete search index by name (skip confirmation)
matlas atlas search delete --project-id <project-id> --cluster <cluster-name> --name default --force
```

### Search Analytics and Optimization

The CLI provides advanced search operations for performance monitoring and optimization:

```bash
# Get search metrics for all indexes in a cluster
matlas atlas search metrics --project-id <project-id> --cluster <cluster-name>

# Get metrics for a specific search index
matlas atlas search metrics --project-id <project-id> --cluster <cluster-name> \
  --index-name products-search --time-range 7d

# Get metrics with JSON output for automation
matlas atlas search metrics --project-id <project-id> --cluster <cluster-name> \
  --index-name products-search --time-range 24h --output json

# Analyze all search indexes for optimization recommendations
matlas atlas search optimize --project-id <project-id> --cluster <cluster-name>

# Analyze a specific index with detailed recommendations
matlas atlas search optimize --project-id <project-id> --cluster <cluster-name> \
  --index-name products-search --analyze-all

# Get optimization results in JSON format
matlas atlas search optimize --project-id <project-id> --cluster <cluster-name> \
  --index-name products-search --output json

# Validate a search query from a file
matlas atlas search validate-query --project-id <project-id> --cluster <cluster-name> \
  --index-name products-search --query-file search-query.json

# Validate an inline search query
matlas atlas search validate-query --project-id <project-id> --cluster <cluster-name> \
  --index-name products-search --query '{"text": {"query": "laptop", "path": "title"}}'

# Validate with detailed analysis and recommendations
matlas atlas search validate-query --project-id <project-id> --cluster <cluster-name> \
  --index-name products-search --query-file complex-query.json --test-mode

# Get validation results in JSON format
matlas atlas search validate-query --project-id <project-id> --cluster <cluster-name> \
  --index-name products-search --query-file query.json --output json
```

#### Available Time Ranges for Metrics
- `1h` - Last hour
- `6h` - Last 6 hours  
- `24h` - Last 24 hours (default)
- `7d` - Last 7 days
- `30d` - Last 30 days

#### Optimization Categories
- `performance` - Performance optimizations
- `mappings` - Field mapping recommendations
- `analyzers` - Analyzer optimization suggestions
- `facets` - Facet configuration improvements
- `synonyms` - Synonym mapping optimizations

#### Query Validation Types
- `syntax` - Query syntax validation
- `fields` - Field existence and mapping validation
- `performance` - Performance optimization suggestions

### Advanced Search Features (YAML Configuration)

Advanced search features like analyzers, facets, autocomplete, highlighting, synonyms, and fuzzy search are configured through **YAML ApplyDocuments only**. These features are embedded within search index definitions due to Atlas Admin API limitations.

```yaml
apiVersion: matlas.mongodb.com/v1
kind: SearchIndex
metadata:
  name: products-advanced-search
spec:
  projectName: "your-project-id"
  clusterName: "your-cluster-name"
  databaseName: "ecommerce"
  collectionName: "products"
  indexName: "products-advanced-search"
  indexType: "search"
  definition:
    mappings:
      dynamic: false
      fields:
        title:
          type: string
          analyzer: "productTitleAnalyzer"
        category:
          type: stringFacet
        price:
          type: numberFacet
  # Advanced features (YAML only)
  analyzers:
    - name: "productTitleAnalyzer"
      type: "custom"
      charFilters: []
      tokenizer:
        type: "standard"
      tokenFilters:
        - type: "lowercase"
        - type: "englishStemmer"
  facets:
    - field: "category"
      type: "string"
      numBuckets: 10
    - field: "price"
      type: "number"
      boundaries: [0, 50, 100, 500]
  autocomplete:
    - field: "title"
      maxEdits: 2
      prefixLength: 1
  highlighting:
    - field: "description"
      maxCharsToExamine: 500
      maxNumPassages: 3
  synonyms:
    - name: "product-synonyms"
      input: ["smartphone", "mobile", "phone"]
      output: "smartphone"
      explicit: false
  fuzzySearch:
    - field: "title"
      maxEdits: 2
      prefixLength: 1
      maxExpansions: 50
```

Apply the configuration:
```bash
# Plan search index changes
matlas infra plan -f search-index.yaml --preserve-existing

# Apply search index configuration
matlas infra apply -f search-index.yaml --preserve-existing --auto-approve
```

**Note**: Advanced search features are not available as separate CLI commands because the Atlas Admin API manages these features as part of search index definitions, not as independent resources.

## VPC Endpoints

VPC Endpoints provide secure, private connectivity to your Atlas clusters using AWS PrivateLink, Azure Private Link, or Google Cloud Private Service Connect.

### List VPC Endpoint Services

```bash
# List all VPC endpoint services
matlas atlas vpc-endpoints list --project-id <project-id>

# List VPC endpoints for specific cloud provider
matlas atlas vpc-endpoints list --project-id <project-id> --cloud-provider AWS
```

### Get VPC Endpoint Service Details

```bash
# Get details for a specific VPC endpoint service
matlas atlas vpc-endpoints get \
  --project-id <project-id> \
  --cloud-provider AWS \
  --endpoint-id <service-id>
```

### Create VPC Endpoint Service

```bash
# Create a VPC endpoint service for AWS
matlas atlas vpc-endpoints create \
  --project-id <project-id> \
  --cloud-provider AWS \
  --region us-east-1

# Create a VPC endpoint service for Azure
matlas atlas vpc-endpoints create \
  --project-id <project-id> \
  --cloud-provider AZURE \
  --region eastus

# Create a VPC endpoint service for GCP
matlas atlas vpc-endpoints create \
  --project-id <project-id> \
  --cloud-provider GCP \
  --region us-central1
```

### Update VPC Endpoint Service

```bash
# Update a VPC endpoint service (most properties are immutable)
matlas atlas vpc-endpoints update \
  --project-id <project-id> \
  --cloud-provider AWS \
  --endpoint-id <service-id>
```

### Delete VPC Endpoint Service

```bash
# Delete a VPC endpoint service
matlas atlas vpc-endpoints delete \
  --project-id <project-id> \
  --cloud-provider AWS \
  --endpoint-id <service-id>

# Delete without confirmation prompt
matlas atlas vpc-endpoints delete \
  --project-id <project-id> \
  --cloud-provider AWS \
  --endpoint-id <service-id> \
  --yes
```

### YAML Configuration

VPC endpoints can also be managed declaratively via YAML:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: vpc-endpoint-example
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: production-vpc-endpoint
      labels:
        environment: production
        provider: aws
    spec:
      projectName: "your-project-id"
      cloudProvider: "AWS"
      region: "us-east-1"
```

Apply the configuration:
```bash
# Plan VPC endpoint changes
matlas infra plan -f vpc-endpoint.yaml --preserve-existing

# Apply VPC endpoint configuration
matlas infra apply -f vpc-endpoint.yaml --preserve-existing --auto-approve
```

## Alerts

MongoDB Atlas alerts provide monitoring and notification capabilities for your Atlas resources. The CLI supports both alert management and alert configuration management.

### Alert Management

Monitor and acknowledge active alerts in your Atlas projects:

```bash
# List all alerts in a project
matlas atlas alerts list --project-id <project-id>

# List alerts with pagination
matlas atlas alerts list --project-id <project-id> --page 2 --limit 10

# Get specific alert details
matlas atlas alerts get <alert-id> --project-id <project-id>

# Acknowledge an alert
matlas atlas alerts acknowledge <alert-id> --project-id <project-id>

# Unacknowledge an alert
matlas atlas alerts acknowledge <alert-id> --project-id <project-id> --unacknowledge

# Output as JSON for automation
matlas atlas alerts list --project-id <project-id> --output json
```

### Alert Configuration Management

Create and manage alert configurations for monitoring:

```bash
# List all alert configurations
matlas atlas alert-configurations list --project-id <project-id>

# List with pagination
matlas atlas alert-configurations list --project-id <project-id> --page 2 --limit 10

# Get specific alert configuration details
matlas atlas alert-configurations get <config-id> --project-id <project-id>

# Delete an alert configuration
matlas atlas alert-configurations delete <config-id> --project-id <project-id>

# Delete without confirmation prompt
matlas atlas alert-configurations delete <config-id> --project-id <project-id> --yes

# List available matcher field names for alert rules
matlas atlas alert-configurations matcher-fields

# Output as JSON for automation
matlas atlas alert-configurations list --project-id <project-id> --output json
```

### YAML Configuration

Alerts can be managed declaratively via YAML for infrastructure-as-code workflows:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: monitoring-alerts
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: high-cpu-alert
      labels:
        severity: high
        category: performance
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
        - typeName: "SLACK"
          apiToken: "${SLACK_TOKEN}"
          channelName: "#alerts"
      metricThreshold:
        metricName: "CPU_USAGE_PERCENT"
        operator: "GREATER_THAN"
        threshold: 80.0
        units: "PERCENT"
        mode: "AVERAGE"
```

Apply the configuration:
```bash
# Plan alert changes
matlas infra plan -f alerts.yaml --preserve-existing

# Apply alert configuration
matlas infra apply -f alerts.yaml --preserve-existing --auto-approve
```

### Notification Channels

The CLI supports multiple notification channels:

- **EMAIL** - Direct email notifications
- **SMS** - Mobile phone text messages  
- **SLACK** - Slack channel notifications
- **PAGER_DUTY** - PagerDuty service integration
- **OPS_GENIE** - OpsGenie alert management
- **DATADOG** - Datadog monitoring integration
- **MICROSOFT_TEAMS** - Microsoft Teams webhook
- **WEBHOOK** - Custom HTTP webhooks
- **USER** - Atlas user notifications
- **GROUP** - Project group notifications
- **TEAM** - Atlas team notifications

### Alert Event Types

Common event types for monitoring:

- **HOST_CPU_USAGE_PERCENT** - CPU usage monitoring
- **HOST_MEMORY_USAGE_PERCENT** - Memory usage monitoring
- **HOST_DISK_USAGE_PERCENT** - Disk usage monitoring
- **CLUSTER_MONGOS_IS_MISSING** - Missing mongos process
- **CLUSTER_PRIMARY_ELECTED** - Primary election events
- **DATABASE_CONNECTIONS_PERCENT** - Connection usage monitoring
- **REPLICATION_LAG** - Replication lag monitoring
- **CLUSTER_DISK_USAGE_PERCENT** - Cluster disk usage

### Matcher Operators

Target specific resources with matchers:

- **EQUALS** / **NOT_EQUALS** - Exact string matching
- **CONTAINS** / **NOT_CONTAINS** - Substring matching
- **STARTS_WITH** / **ENDS_WITH** - Prefix/suffix matching
- **REGEX** / **NOT_REGEX** - Regular expression matching

### Examples

See the [Alert Examples]({{ '/examples/' | relative_url }}) for:
- Basic CPU and memory monitoring setups
- Multi-channel notification configurations
- Complex matcher and threshold patterns

## Feature availability

**Updated:** The following features are now available:
- ✅ **Atlas Search**: All commands (`list`, `create`, `get`, `update`, `delete`) - Full functionality with CLI and YAML support
- ✅ **VPC Endpoints**: All commands (`list`, `create`, `get`, `update`, `delete`) - Full functionality with CLI and YAML support
- ✅ **Alerts**: All commands (`list`, `get`, `acknowledge`) and alert configurations (`list`, `get`, `delete`, `matcher-fields`) - Full functionality with CLI and YAML support