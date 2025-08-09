# Enhanced MongoDB Atlas Cluster Creation Guide

This guide covers the comprehensive cluster creation capabilities of the `matlas` CLI tool, which provides full support for MongoDB Atlas cluster configuration including advanced features like autoscaling, encryption, multi-region deployment, and YAML configuration files.

## Table of Contents

1. [Basic Cluster Creation](#basic-cluster-creation)
2. [MongoDB Version Selection](#mongodb-version-selection)
3. [Advanced Configuration](#advanced-configuration)
4. [Multi-Region Clusters](#multi-region-clusters)
5. [Sharded Clusters](#sharded-clusters)
6. [YAML Configuration Files](#yaml-configuration-files)
7. [Security and Encryption](#security-and-encryption)
8. [Autoscaling](#autoscaling)
9. [Storage Configuration](#storage-configuration)
10. [BI Connector](#bi-connector)
11. [Examples](#examples)

## Basic Cluster Creation

Create a basic MongoDB Atlas cluster with minimal configuration:

```bash
matlas atlas clusters create \
  --name my-cluster \
  --project-id 507f1f77bcf86cd799439011
```

## MongoDB Version Selection

The CLI supports multiple MongoDB versions:

- `7.0` - MongoDB 7.0 (stable)
- `8.0` - MongoDB 8.0 (latest stable)
- `latest` - Latest release with auto-upgrades

```bash
matlas atlas clusters create \
  --name my-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --mongodb-version 8.0
```

## Advanced Configuration

### Available Cluster Tiers

- **Free/Development**: `M0` (free), `M10`, `M20`
- **Production**: `M30`, `M40`, `M50`, `M60`, `M80`, `M140`, `M200`, `M300`, `M400`, `M700`
- **Low-CPU variants**: `R40`, `R50`, `R60`, `R80`, `R200`, `R300`, `R400`, `R700`

### Cloud Providers and Regions

- **AWS**: `US_EAST_1`, `US_WEST_2`, `EU_WEST_1`, etc.
- **GCP**: `EASTERN_US`, `WESTERN_US`, `EUROPE_WEST_1`, etc.
- **Azure**: `US_EAST`, `US_WEST`, `EUROPE_NORTH`, etc.

```bash
matlas atlas clusters create \
  --name production-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M30 \
  --provider AWS \
  --region US_EAST_1
```

## Multi-Region Clusters

Deploy clusters across multiple regions for high availability and reduced latency:

```bash
matlas atlas clusters create \
  --name global-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M40 \
  --provider AWS \
  --region US_EAST_1 \
  --additional-regions US_WEST_2,EU_WEST_1
```

## Sharded Clusters

Create sharded clusters for horizontal scaling (requires M30 or higher):

```bash
matlas atlas clusters create \
  --name sharded-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M30 \
  --cluster-type SHARDED \
  --num-shards 3 \
  --replication-factor 3
```

### Cluster Types

- `REPLICASET` - Standard replica set (default)
- `SHARDED` - Sharded cluster (1-70 shards)

## YAML Configuration Files

The CLI uses the API specification format (`apiVersion: matlas.mongodb.com/v1`) for YAML configuration files, ensuring consistency with the project's infrastructure-as-code approach:

```yaml
# cluster-api-spec.yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: production-project
spec:
  name: "Production Project"
  organizationId: "507f1f77bcf86cd799439011"
  clusters:
    - metadata:
        name: production-cluster
        labels:
          environment: production
          team: backend
      provider: AWS
      region: US_EAST_1
      instanceSize: M30
      mongodbVersion: "8.0"
      clusterType: REPLICASET
      diskSizeGB: 100
      backupEnabled: true
      tierType: dedicated
      autoScaling:
        compute:
          enabled: true
          minInstanceSize: M20
          maxInstanceSize: M60
        diskGB:
          enabled: true
      encryption:
        encryptionAtRestProvider: "AWS"
        awsKms:
          enabled: true
          customerMasterKeyId: "your-kms-key-id"
      biConnector:
        enabled: false
        readPreference: "secondary"
```

### Using YAML Configuration

Create from YAML configuration:

```bash
# Create cluster from API specification YAML
matlas atlas clusters create --config cluster-spec.yaml

# Override specific values with CLI flags (flags take precedence over YAML)
matlas atlas clusters create --config cluster-spec.yaml --tier M40 --mongodb-version 8.0
```

### Benefits of API Specification Format

- **🔄 Consistency**: Aligns with project's infrastructure-as-code standards
- **📋 Comprehensive**: Supports projects, clusters, database users, and network access in one file
- **🔒 Validation**: Built-in validation for API version and resource kinds
- **🚀 Scalability**: Easily manage complex multi-resource deployments
- **🛠️ Integration**: Works seamlessly with existing project tooling

## Security and Encryption

Enable encryption at rest and other security features:

```bash
matlas atlas clusters create \
  --name secure-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M30 \
  --encryption-at-rest \
  --aws-kms-key-id "your-kms-key-id" \
  --termination-protection \
  --backup
```

### Security Options

- `--encryption-at-rest` - Enable encryption at rest
- `--aws-kms-key-id` - AWS KMS key for encryption (AWS only)
- `--azure-key-vault-key-id` - Azure Key Vault key (Azure only)
- `--gcp-kms-key-id` - GCP KMS key (GCP only)
- `--termination-protection` - Prevent accidental deletion

## Autoscaling

Configure automatic scaling for storage and compute resources:

```bash
matlas atlas clusters create \
  --name autoscaling-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M30 \
  --autoscaling \
  --autoscaling-disk \
  --autoscaling-compute \
  --min-instance-size M20 \
  --max-instance-size M80
```

### Autoscaling Options

- `--autoscaling` - Enable cluster tier autoscaling
- `--autoscaling-disk` - Enable disk autoscaling (default: enabled)
- `--autoscaling-compute` - Enable compute autoscaling
- `--min-instance-size` - Minimum tier for autoscaling
- `--max-instance-size` - Maximum tier for autoscaling

## Storage Configuration

Customize storage capacity and performance:

```bash
matlas atlas clusters create \
  --name high-performance-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M60 \
  --disk-size 500 \
  --disk-iops 3000 \
  --ebs-volume-type gp3
```

### Storage Options

- `--disk-size` - Disk size in GB (varies by tier)
- `--disk-iops` - Provisioned IOPS (AWS only)
- `--ebs-volume-type` - EBS volume type: `STANDARD`, `PROVISIONED`, `gp3` (AWS only)

## BI Connector

Enable BI Connector for Atlas (SQL access to MongoDB):

```bash
matlas atlas clusters create \
  --name analytics-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M30 \
  --bi-connector \
  --bi-connector-read-preference secondary
```

### BI Connector Options

- `--bi-connector` - Enable BI Connector
- `--bi-connector-read-preference` - Read preference: `primary`, `secondary`, `analytics`

## Examples

### Example 1: Development Cluster

```bash
matlas atlas clusters create \
  --name dev-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M10 \
  --provider AWS \
  --region US_EAST_1 \
  --mongodb-version 8.0
```

### Example 2: Production Cluster with Encryption

```bash
matlas atlas clusters create \
  --name production-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M30 \
  --provider AWS \
  --region US_EAST_1 \
  --mongodb-version 8.0 \
  --backup \
  --encryption-at-rest \
  --termination-protection \
  --disk-size 100
```

### Example 3: Multi-Region Cluster with Autoscaling

```bash
matlas atlas clusters create \
  --name global-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M40 \
  --provider AWS \
  --region US_EAST_1 \
  --additional-regions US_WEST_2,EU_WEST_1 \
  --autoscaling \
  --min-instance-size M30 \
  --max-instance-size M80 \
  --mongodb-version 8.0
```

### Example 4: Sharded Cluster

```bash
matlas atlas clusters create \
  --name sharded-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M30 \
  --cluster-type SHARDED \
  --num-shards 3 \
  --replication-factor 3 \
  --provider AWS \
  --region US_EAST_1 \
  --mongodb-version 8.0 \
  --backup
```

### Example 5: High-Performance Cluster

```bash
matlas atlas clusters create \
  --name performance-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M60 \
  --provider AWS \
  --region US_EAST_1 \
  --disk-size 500 \
  --disk-iops 3000 \
  --ebs-volume-type gp3 \
  --mongodb-version 8.0
```

## Flag Reference

### Required Flags

- `--name` - Cluster name (can be specified in YAML)

### Optional Flags

#### Basic Configuration
- `--project-id` - Project ID (from ATLAS_PROJECT_ID env var if not specified)
- `--tier` - Cluster tier (default: M10)
- `--provider` - Cloud provider (default: AWS)
- `--region` - Provider region (default: US_EAST_1)

#### MongoDB Configuration
- `--mongodb-version` - MongoDB version (default: 7.0)
- `--cluster-type` - Cluster type (default: REPLICASET)
- `--num-shards` - Number of shards for sharded clusters (default: 1)
- `--replication-factor` - Number of replica set members (default: 3)

#### Storage Configuration
- `--disk-size` - Disk size in GB (default: tier default)
- `--disk-iops` - Provisioned IOPS (default: 0 for auto)
- `--ebs-volume-type` - EBS volume type (default: STANDARD)

#### Backup and Security
- `--backup` - Enable continuous backup (default: true)
- `--encryption-at-rest` - Enable encryption at rest
- `--aws-kms-key-id` - AWS KMS key ID
- `--azure-key-vault-key-id` - Azure Key Vault key ID
- `--gcp-kms-key-id` - GCP KMS key ID
- `--termination-protection` - Enable termination protection

#### Autoscaling
- `--autoscaling` - Enable cluster tier autoscaling
- `--autoscaling-disk` - Enable disk autoscaling (default: true)
- `--autoscaling-compute` - Enable compute autoscaling
- `--min-instance-size` - Minimum instance size for autoscaling
- `--max-instance-size` - Maximum instance size for autoscaling

#### Multi-Region
- `--additional-regions` - Additional regions (comma-separated)

#### BI Connector
- `--bi-connector` - Enable BI Connector
- `--bi-connector-read-preference` - Read preference (default: secondary)

#### Configuration File
- `--config` - Path to YAML configuration file

## Validation Rules

1. **Cluster name** must be valid (alphanumeric, hyphens, 1-64 characters)
2. **Project ID** must be a valid MongoDB ObjectId
3. **Tier** must be from the supported list
4. **Provider** must be AWS, GCP, or AZURE
5. **MongoDB version** must be 7.0, 8.0, or latest
6. **Cluster type** must be REPLICASET or SHARDED
7. **Sharded clusters** require M30 or higher tier
8. **Number of shards** must be 1-70 for sharded clusters
9. **Replication factor** must be 3 or 5
10. **Autoscaling** requires min/max instance sizes when compute autoscaling is enabled

## Error Handling

The CLI provides detailed error messages with suggestions for common issues:

- **Validation errors** - Clear messages about invalid parameters
- **API errors** - Helpful hints for common API issues
- **Configuration errors** - Specific guidance for YAML file issues
- **Authentication errors** - Suggestions for API key problems

## Environment Variables

- `ATLAS_PROJECT_ID` - Default project ID
- `ATLAS_PUBLIC_KEY` - Atlas API public key
- `ATLAS_PRIVATE_KEY` - Atlas API private key

## Best Practices

1. **Use YAML files** for complex configurations to ensure consistency
2. **Enable backup** for production clusters
3. **Use encryption at rest** for sensitive data
4. **Enable termination protection** for critical clusters
5. **Use autoscaling** to optimize costs and performance
6. **Deploy multi-region** clusters for high availability
7. **Choose appropriate tiers** based on workload requirements
8. **Use environment variables** for sensitive information

## Troubleshooting

### Common Issues

1. **"Bad request" errors** - Check parameter validation and API limits
2. **Authentication failures** - Verify API keys and permissions
3. **YAML parsing errors** - Validate YAML syntax and structure
4. **Region/tier combinations** - Some tiers aren't available in all regions
5. **Quota limits** - Check your Atlas organization limits

### Getting Help

Use the `--verbose` flag for detailed logging and error information:

```bash
matlas atlas clusters create --name test --verbose
```

Check the help for specific flags:

```bash
matlas atlas clusters create --help
```