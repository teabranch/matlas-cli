---
layout: default
title: Cluster Examples
parent: Examples
nav_order: 2
description: MongoDB cluster configurations for different environments
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

## Usage Examples

### Environment Variables

Set required environment variables before applying:

```bash
export CLUSTER_ADMIN_PASSWORD='SecureAdminPass123!'
export BACKUP_USER_PASSWORD='SecureBackupPass123!'
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