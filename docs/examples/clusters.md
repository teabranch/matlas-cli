---
layout: default
title: Cluster Examples
parent: Examples
nav_order: 2
description: MongoDB cluster configurations for different environments
permalink: /examples/clusters/
---

# Cluster Examples

MongoDB cluster configurations for different environments, from basic development setups to production clusters with autoscaling and multi-region replication.

## cluster-basic.yaml

Minimal cluster definition perfect for development environments.

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: cluster-basic
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: basic-cluster
    spec:
      projectName: "My Project"
      provider: AWS
      region: us-west-2
      instanceSize: M10
```
{% endraw %}

## cluster-comprehensive.yaml

Production-ready cluster with autoscaling, multi-region, security features, and comprehensive configuration.

{% raw %}
```yaml
# Comprehensive Cluster Configuration Example
# This example shows advanced cluster configuration with autoscaling, backup, and security features

apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: cluster-comprehensive
  labels:
    environment: production
    tier: advanced
    purpose: example
  annotations:
    description: "Comprehensive cluster configuration with advanced features"
    cost-warning: "This configuration may incur significant costs"
resources:
  # Production cluster with autoscaling and advanced features
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: prod-cluster-advanced
      labels:
        environment: production
        tier: advanced
        backup: enabled
        encryption: enabled
    spec:
      projectName: "Production Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M40
      clusterType: REPLICASET
      tierType: REPLICASET
      mongodbVersion: "7.0"
      
      # Storage configuration
      diskSizeGB: 80
      
      # Backup configuration
      backupEnabled: true
      pitEnabled: true
      
      # Autoscaling configuration
      autoscaling:
        diskGBEnabled: true
        computeEnabled: true
        computeScaleDownEnabled: true
        computeMinInstanceSize: M40
        computeMaxInstanceSize: M80
      
      # Security features
      encryptionAtRestProvider: AWS
      
      # BI Connector
      biConnector:
        enabled: true
        readPreference: secondary
      
      # Tags for resource management
      tags:
        - key: Environment
          value: Production
        - key: Application
          value: MainApp
        - key: CostCenter
          value: Engineering
        - key: Owner
          value: Platform Team

  # Multi-region cluster for high availability
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: global-cluster
      labels:
        environment: production
        type: global
        regions: multi
    spec:
      projectName: "Production Project"
      provider: AWS
      clusterType: REPLICASET
      mongodbVersion: "7.0"
      backupEnabled: true
      
      # Multi-region replication specifications
      replicationSpecs:
        - numShards: 1
          regionConfigs:
            - electableNodes: 3
              priority: 7
              readOnlyNodes: 0
              analyticsNodes: 0
              providerName: AWS
              regionName: US_EAST_1
              instanceSize: M30
            - electableNodes: 2
              priority: 6
              readOnlyNodes: 0
              analyticsNodes: 0
              providerName: AWS
              regionName: US_WEST_2
              instanceSize: M30
            - electableNodes: 2
              priority: 5
              readOnlyNodes: 1
              analyticsNodes: 0
              providerName: AWS
              regionName: EU_WEST_1
              instanceSize: M30

  # Development cluster with minimal configuration
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: dev-cluster
      labels:
        environment: development
        tier: basic
        cost-optimized: "true"
    spec:
      projectName: "Production Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M10
      clusterType: REPLICASET
      tierType: REPLICASET
      mongodbVersion: "7.0"
      diskSizeGB: 10
      backupEnabled: false
      
      tags:
        - key: Environment
          value: Development
        - key: AutoShutdown
          value: "true"

  # Network access for clusters
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: production-vpc
      labels:
        environment: production
        type: aws-sg
    spec:
      projectName: "Production Project"
      awsSecurityGroup: "sg-1234567890abcdef0"
      comment: "Production VPC security group access"

  # Administrative users for cluster management
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: cluster-admin
      labels:
        purpose: administration
        scope: all-clusters
    spec:
      projectName: "Production Project"
      username: cluster-admin
      authDatabase: admin
      password: "${CLUSTER_ADMIN_PASSWORD}"
      roles:
        - roleName: atlasAdmin
          databaseName: admin
```
{% endraw %}

## cluster-multiregion.yaml

Specialized multi-region cluster using `replicationSpecs` and `regionConfigs` for global distribution.

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: cluster-multiregion
  labels:
    type: global
    deployment: multi-region
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: global-ecommerce-cluster
      labels:
        environment: production
        application: ecommerce
        topology: global
    spec:
      projectName: "Global Ecommerce Project"
      provider: AWS
      clusterType: REPLICASET
      mongodbVersion: "7.0"
      backupEnabled: true
      pitEnabled: true
      
      # Multi-region configuration for global presence
      replicationSpecs:
        - numShards: 1
          regionConfigs:
            # Primary region - US East (highest priority)
            - electableNodes: 3
              priority: 7
              readOnlyNodes: 0
              analyticsNodes: 1
              providerName: AWS
              regionName: US_EAST_1
              instanceSize: M40
              
            # Secondary region - Europe (medium priority)
            - electableNodes: 2
              priority: 6
              readOnlyNodes: 2
              analyticsNodes: 1
              providerName: AWS
              regionName: EU_WEST_1
              instanceSize: M30
              
            # Tertiary region - Asia Pacific (lower priority)
            - electableNodes: 2
              priority: 5
              readOnlyNodes: 1
              analyticsNodes: 0
              providerName: AWS
              regionName: AP_SOUTHEAST_1
              instanceSize: M30
      
      # Autoscaling for global workloads
      autoscaling:
        diskGBEnabled: true
        computeEnabled: true
        computeScaleDownEnabled: true
        computeMinInstanceSize: M30
        computeMaxInstanceSize: M60
      
      tags:
        - key: Environment
          value: Production
        - key: Topology
          value: Global
        - key: Application
          value: Ecommerce
        - key: DataResidency
          value: MultiRegion
```
{% endraw %}

## cluster-backup-comprehensive.yaml

Comprehensive backup features demonstration including continuous backup, point-in-time recovery, and cross-region backup configurations.

{% raw %}
```yaml
# Comprehensive Backup Features Example
# This example demonstrates all backup features: continuous backup, point-in-time recovery, and cross-region backup

apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: backup-features-comprehensive
  labels:
    purpose: backup-demo
    environment: production
  annotations:
    description: "Comprehensive backup features demonstration"
    documentation: "Shows continuous backup, PIT recovery, and cross-region configurations"
resources:
  # Basic backup-enabled cluster
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: backup-basic-cluster
      labels:
        backup: enabled
        tier: standard
        environment: production
    spec:
      projectName: "Backup Demo Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M10
      diskSizeGB: 20
      mongodbVersion: "7.0"
      
      # Continuous backup configuration
      backupEnabled: true
      
      tags:
        - key: BackupPolicy
          value: Standard
        - key: Environment
          value: Production

  # Point-in-Time Recovery enabled cluster
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: backup-pit-cluster
      labels:
        backup: enabled
        pit: enabled
        tier: advanced
        criticality: high
    spec:
      projectName: "Backup Demo Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M20
      diskSizeGB: 40
      mongodbVersion: "7.0"
      
      # Advanced backup configuration with Point-in-Time Recovery
      backupEnabled: true
      pitEnabled: true       # Requires backupEnabled: true
      
      # Enhanced storage for PIT workloads
      autoscaling:
        diskGBEnabled: true
        computeEnabled: false
      
      tags:
        - key: BackupPolicy
          value: PointInTime
        - key: Environment
          value: Production
        - key: DataCriticality
          value: High

  # Cross-region backup cluster (via multi-region configuration)
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: backup-cross-region-cluster
      labels:
        backup: enabled
        regions: multi
        geographic-redundancy: enabled
        disaster-recovery: enabled
    spec:
      projectName: "Backup Demo Project"
      provider: AWS
      clusterType: REPLICASET
      mongodbVersion: "7.0"
      
      # Backup enabled for cross-region redundancy
      backupEnabled: true
      
      # Multi-region configuration for geographic backup redundancy
      replicationSpecs:
        - numShards: 1
          regionConfigs:
            # Primary region - US East
            - electableNodes: 3
              priority: 7
              readOnlyNodes: 0
              analyticsNodes: 0
              providerName: AWS
              regionName: US_EAST_1
              instanceSize: M30
              
            # Backup region - US West (geographic separation)
            - electableNodes: 2
              priority: 6
              readOnlyNodes: 1
              analyticsNodes: 0
              providerName: AWS
              regionName: US_WEST_2
              instanceSize: M20
              
            # International backup region - Europe
            - electableNodes: 2
              priority: 5
              readOnlyNodes: 1
              analyticsNodes: 0
              providerName: AWS
              regionName: EU_WEST_1
              instanceSize: M20
      
      tags:
        - key: BackupPolicy
          value: CrossRegion
        - key: Environment
          value: Production
        - key: DisasterRecovery
          value: Global

  # Development cluster with backup disabled (cost optimization)
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: backup-dev-cluster
      labels:
        backup: disabled
        tier: basic
        environment: development
        cost-optimized: "true"
    spec:
      projectName: "Backup Demo Project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M10
      diskSizeGB: 10
      mongodbVersion: "7.0"
      
      # Backup disabled for development to reduce costs
      backupEnabled: false
      # Note: pitEnabled cannot be true when backupEnabled is false
      
      tags:
        - key: BackupPolicy
          value: None
        - key: Environment
          value: Development
        - key: CostOptimization
          value: Enabled
```
{% endraw %}

## Usage Examples

### Environment Variables

Set required environment variables before applying:

```bash
export CLUSTER_ADMIN_PASSWORD='SecureAdminPass123!'
export BACKUP_ADMIN_PASSWORD='SecureBackupPass123!'
```

### Development Workflow

```bash
# Start with basic cluster for development
matlas infra validate -f cluster-basic.yaml
matlas infra apply -f cluster-basic.yaml

# Preview comprehensive production setup
matlas infra plan -f cluster-comprehensive.yaml --output table

# Apply production cluster safely
matlas infra apply -f cluster-comprehensive.yaml --preserve-existing
```

### Multi-Region Deployment

```bash
# Validate multi-region configuration
matlas infra validate -f cluster-multiregion.yaml

# Check resource dependencies
matlas infra plan -f cluster-multiregion.yaml --show-dependencies

# Deploy global infrastructure
matlas infra apply -f cluster-multiregion.yaml --auto-approve
```

### Backup Features Workflow

```bash
# Deploy backup-enabled clusters
matlas infra validate -f cluster-backup-comprehensive.yaml
matlas infra apply -f cluster-backup-comprehensive.yaml

# CLI backup management (alternative to YAML)
# Create cluster with backup first
matlas atlas clusters create my-cluster --backup --tier M10 --provider AWS --region US_EAST_1

# Enable Point-in-Time Recovery after cluster is ready
# Note: PIT cannot be enabled during cluster creation
matlas atlas clusters update my-cluster --pit

# Check backup status
matlas atlas clusters describe my-cluster --output json | jq '.backupEnabled, .pitEnabled'
```

### Point-in-Time Recovery Workflow

**Important**: PIT recovery must be enabled AFTER cluster creation, not during creation.

```bash
# ❌ This will fail - PIT cannot be enabled during creation
matlas atlas clusters create my-cluster --pit

# ✅ Correct workflow
# Step 1: Create cluster with backup
matlas atlas clusters create my-cluster --backup --tier M10 --provider AWS --region US_EAST_1

# Step 2: Wait for cluster to be ready (check status)
matlas atlas clusters describe my-cluster

# Step 3: Enable PIT via update
matlas atlas clusters update my-cluster --pit
```

## Key Features Demonstrated

### Basic Cluster (Development)
- **Minimal configuration** for cost optimization
- **M10 instance size** suitable for development
- **Single region** deployment
- **Basic backup** disabled for cost savings

### Comprehensive Cluster (Production)
- **Autoscaling** with compute and storage scaling
- **Advanced backup** with point-in-time recovery
- **Security features** with encryption at rest
- **BI Connector** for analytics workloads
- **Resource tagging** for management and billing
- **Multiple environments** in single document

### Backup Features (cluster-backup-comprehensive.yaml)
- **Continuous backup** for automated snapshots
- **Point-in-Time Recovery** for precise data recovery
- **Cross-region backup** via multi-region cluster topology
- **Cost optimization** examples for development environments
- **Backup validation** with enforced configuration rules

### Multi-Region Cluster (Global)
- **Geographic distribution** across US, Europe, and Asia
- **Priority-based replica** configuration
- **Read replicas** for local read performance
- **Analytics nodes** for dedicated workloads
- **Global autoscaling** policies

## Best Practices

### Cost Optimization
- Use **M10-M20** for development environments
- Disable **backups** for non-critical workloads
- Enable **autoscaling** to handle variable loads
- Use **appropriate instance sizes** for workload requirements

### Backup Strategy
- **Always enable backup** for production clusters (`backupEnabled: true`)
- **Enable PIT recovery** for critical data (`pitEnabled: true`)
- **Use multi-region clusters** for geographic backup redundancy
- **Validate backup configuration** before applying changes
- **Test restore procedures** regularly to ensure backup integrity
- **Consider backup costs** when planning cluster configurations

### Security
- Always enable **encryption at rest** for production
- Use **network access controls** with security groups
- Implement **proper user management** with scoped access
- Enable **audit logging** for compliance requirements

### High Availability
- Use **multi-region** deployment for global applications
- Configure **appropriate replica** counts per region
- Enable **point-in-time recovery** for critical data
- Implement **monitoring and alerting**

## Related Examples

- [User Management]({{ '/examples/users/' | relative_url }}) - Users for cluster access
- [Network Access]({{ '/examples/network/' | relative_url }}) - Network security configuration  
- [Infrastructure Patterns]({{ '/examples/infrastructure/' | relative_url }}) - Complete infrastructure setups