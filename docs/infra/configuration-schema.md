# Configuration Schema Reference

This document provides a comprehensive reference for the YAML configuration schema used by `matlas infra`. It covers all supported resource types, fields, validation rules, and examples.

## Table of Contents

- [Schema Overview](#schema-overview)
- [Common Fields](#common-fields)
- [Resource Types](#resource-types)
- [Validation Rules](#validation-rules)
- [Advanced Configuration](#advanced-configuration)
- [Schema Versions](#schema-versions)

## Schema Overview

All configuration files follow this basic structure:

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

### Required Fields

- `apiVersion`: API version (see [Schema Versions](#schema-versions))
- `kind`: Resource type (Project, Cluster, DatabaseUser, NetworkAccess)
- `metadata.name`: Unique resource identifier
- `spec`: Resource specification (varies by type)

## Common Fields

### Metadata

All resources support metadata for identification and organization:

```yaml
metadata:
  name: my-resource           # Required: Resource identifier (1-64 chars, hostname format)
  labels:                     # Optional: Key-value pairs for categorization
    environment: production
    team: backend
    cost-center: "12345"
  annotations:                # Optional: Extended metadata
    description: "Primary production cluster"
    created-by: "john.doe@company.com"
    last-modified: "2024-01-15T10:30:00Z"
  deletionPolicy: Delete      # Optional: Delete, Retain, Snapshot
  dependsOn:                  # Optional: Resource dependencies
    - other-resource-name
```

#### Deletion Policies

- `Delete` (default): Remove resource when deleted from configuration
- `Retain`: Keep resource but stop managing it
- `Snapshot`: Take snapshot before deletion (clusters only)

### Labels and Annotations

**Labels** (max 63 chars each):
- Used for selection and filtering
- Must be valid Kubernetes label format
- Common patterns: `environment`, `team`, `version`, `cost-center`

**Annotations** (max 512 chars each):
- Extended metadata and documentation
- Can contain any string data
- Common patterns: `description`, `created-by`, `external-id`

### Atlas Resource Tags

**Atlas Resource Tags** are native MongoDB Atlas tags that appear in the Atlas UI, billing invoices, and monitoring integrations.

**Tag Requirements:**
- Maximum 50 tags per resource
- Tag keys: 1-255 characters, must be unique
- Tag values: 1-255 characters
- Allowed characters: letters, numbers, spaces, `;@_-.+`
- Case-sensitive keys and values

**Use Cases:**
- **Cost allocation**: Group resources by department, project, or billing code
- **Resource organization**: Categorize by environment, application, or team
- **Monitoring**: Tag resources for automated alerts and dashboards
- **Compliance**: Mark resources with security or compliance requirements

**Common Tag Patterns:**
```yaml
tags:
  environment: "production"       # Environment classification
  application: "user-service"     # Application or service name
  team: "backend"                 # Owning team
  cost-center: "engineering"      # Billing department
  tier: "critical"                # Service tier/priority
  backup-required: "true"         # Backup policy
  compliance-level: "high"        # Security/compliance level
```

**Note:** Atlas Resource Tags are different from labels/annotations:
- Labels/annotations: Internal to matlas-cli for resource management
- Tags: Native Atlas feature for billing, monitoring, and organization

## Resource Types

### Project

Complete Atlas project configuration with all associated resources.

```yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: my-project
  labels:
    environment: production
    region: us-east
spec:
  name: "My Production Project"                    # Required: Atlas project name
  organizationId: "507f1f77bcf86cd799439011"      # Required: 24-char organization ID
  
  # Optional: Clusters in this project
  clusters:
    - metadata:
        name: primary-cluster
      provider: AWS                                # Required: AWS, GCP, AZURE
      region: US_EAST_1                           # Required: Provider region
      instanceSize: M30                           # Required: Cluster tier
      diskSizeGB: 100                            # Optional: Disk size
      backupEnabled: true                        # Optional: Enable backups
      mongodbVersion: "7.0"                      # Optional: MongoDB version
      clusterType: REPLICASET                    # Optional: REPLICASET, SHARDED
      
      # Optional: Advanced cluster configuration
      autoScaling:
        compute:
          enabled: true
          scaleDownEnabled: true
          minInstanceSize: M20
          maxInstanceSize: M50
        diskGB:
          enabled: true
      
      encryption:
        encryptionAtRest: true
        awsKmsKeyId: "arn:aws:kms:US_EAST_1:123456789012:key/..."
      
      biConnector:
        enabled: true
        readPreference: secondary
      
      # Optional: Multi-region configuration
      replicationSpecs:
        - numShards: 1
          regionConfigs:
            - regionName: US_EAST_1
              priority: 7
              analyticsNodes: 1
              electableNodes: 3
              readOnlyNodes: 0
  
  # Optional: Database users
  databaseUsers:
    - metadata:
        name: app-user
      username: "myapp"                          # Required: Username
      databaseName: "admin"                      # Required: Auth database
      password: "${APP_PASSWORD}"               # Required: Use env vars for passwords
      roles:                                    # Required: At least one role
        - roleName: "readWrite"
          databaseName: "mydb"
        - roleName: "read"
          databaseName: "analytics"
      scopes:                                   # Optional: Scope to specific clusters
        - name: "primary-cluster"
          type: "CLUSTER"
      
      # Optional: Advanced user configuration
      deleteAfterDate: "2024-12-31T23:59:59Z"  # Auto-deletion date
      labels:
        - key: "purpose"
          value: "application"
      
      # Optional: X.509 authentication
      x509Type: "MANAGED"                       # NONE, MANAGED, CUSTOMER
      
      # Optional: LDAP authentication
      ldapAuthType: "USER"                      # USER, GROUP
  
  # Optional: Network access rules
  networkAccess:
    - metadata:
        name: office-access
      ipAddress: "203.0.113.50"                # Specific IP address
      comment: "Office static IP"
      deleteAfterDate: "2024-06-01T00:00:00Z"
    
    - metadata:
        name: office-network
      cidr: "203.0.113.0/24"                   # CIDR block
      comment: "Office network range"
    
    - metadata:
        name: aws-sg-access
      awsSecurityGroup: "sg-0123456789abcdef0"  # AWS Security Group
      comment: "Production AWS security group"
      deleteAfterDate: "2025-01-01T00:00:00Z"
```

### Individual Resource Types

#### Cluster

```yaml
apiVersion: matlas.mongodb.com/v1
kind: Cluster
metadata:
  name: analytics-cluster
  labels:
    purpose: analytics
    environment: production
spec:
  projectName: "my-project"                     # Required: Reference to project
  provider: GCP                                # Required: AWS, GCP, AZURE
  region: us-central1                          # Required: Provider region
  instanceSize: M40                            # Required: Cluster tier
  
  # Required cluster configuration
  mongodbVersion: "7.0"
  clusterType: REPLICASET
  diskSizeGB: 500
  backupEnabled: true
  
  # Optional: Performance optimization
  autoScaling:
    compute:
      enabled: true
      scaleDownEnabled: true
      minInstanceSize: M30
      maxInstanceSize: M80
    diskGB:
      enabled: true
      
  # Optional: Security
  encryption:
    encryptionAtRest: true
    gcpKmsKeyId: "projects/my-project/locations/global/keyRings/..."
    
  # Optional: Multi-region sharded cluster
  replicationSpecs:
    - numShards: 2
      zoneName: "Zone 1"
      regionConfigs:
        - regionName: US_CENTRAL_1
          priority: 7
          analyticsNodes: 1
          electableNodes: 3
          readOnlyNodes: 1
        - regionName: US_EAST_1
          priority: 6
          analyticsNodes: 0
          electableNodes: 2
          readOnlyNodes: 1
```

#### DatabaseUser

```yaml
apiVersion: matlas.mongodb.com/v1
kind: DatabaseUser
metadata:
  name: analytics-readonly
  labels:
    access-level: readonly
    team: analytics
spec:
  projectName: "my-project"                     # Required: Reference to project
  username: "analytics-reader"                 # Required: Username
  databaseName: "admin"                        # Required: Auth database
  password: "${ANALYTICS_PASSWORD}"            # Required: Password (use env vars)
  
  # Required: User permissions
  roles:
    - roleName: "read"
      databaseName: "analytics"
    - roleName: "read"
      databaseName: "logs"
  
  # Optional: Restrict to specific clusters
  scopes:
    - name: "analytics-cluster"
      type: "CLUSTER"
  
  # Optional: User lifecycle
  deleteAfterDate: "2024-12-31T23:59:59Z"
  
  # Optional: Additional labels
  labels:
    - key: "department"
      value: "engineering"
    - key: "access-reviewed"
      value: "2024-01-15"
```

#### NetworkAccess

```yaml
apiVersion: matlas.mongodb.com/v1
kind: NetworkAccess
metadata:
  name: vpc-peering-access
  labels:
    connection-type: vpc-peering
    environment: production
spec:
  projectName: "my-project"                     # Required: Reference to project
  
  # Choose one access method:
  ipAddress: "10.0.1.100"                      # Specific IP address
  # OR
  cidr: "10.0.0.0/16"                          # CIDR block
  # OR  
  awsSecurityGroup: "sg-0123456789abcdef0"     # AWS Security Group ID
  
  comment: "VPC peering connection"             # Optional: Description
  deleteAfterDate: "2025-06-01T00:00:00Z"     # Optional: Auto-deletion
```

## Validation Rules

### Resource Names

- **Length**: 1-64 characters
- **Format**: Valid hostname (alphanumeric, hyphens, dots)
- **Case**: Case-sensitive
- **Uniqueness**: Must be unique within the configuration file

### Organization and Project IDs

- **Format**: 24-character hexadecimal string
- **Example**: `507f1f77bcf86cd799439011`

### Atlas Resource Names

- **Cluster names**: 1-64 characters, alphanumeric and hyphens
- **Database names**: MongoDB naming rules apply
- **Usernames**: 1-256 characters, valid MongoDB username

### Network Access

- **IP addresses**: Valid IPv4 or IPv6 format
- **CIDR blocks**: Valid CIDR notation (e.g., `192.168.1.0/24`)
- **Security groups**: Valid AWS security group ID format

### Instance Sizes

Valid cluster tiers:
- **Shared**: M0, M2, M5
- **Dedicated**: M10, M20, M30, M40, M50, M60, M80, M100, M140, M200, M300, M400, M700
- **Local NVMe**: M40_NVME, M50_NVME, M60_NVME, M80_NVME

### Providers and Regions

**AWS Regions**:
- US_EAST_1, us-west-1, us-west-2, eu-west-1, eu-central-1, ap-southeast-1, etc.

**GCP Regions**:
- us-central1, us-east1, us-west1, europe-west1, asia-east1, etc.

**Azure Regions**:
- East US, West US, North Europe, West Europe, Southeast Asia, etc.

## Advanced Configuration

### Environment Variable Substitution

Support for complex templating:

```yaml
spec:
  name: "${PROJECT_PREFIX}_${ENVIRONMENT}_primary"
  
  # Conditional inclusion
  encryption:
    encryptionAtRest: "${ENABLE_ENCRYPTION:+true}"
    awsKmsKeyId: "${KMS_KEY_ID:?KMS key required for encryption}"
  
  # Default values
  mongodbVersion: "${MONGODB_VERSION:-7.0}"
  diskSizeGB: ${DISK_SIZE:-100}
  
  # Nested variable expansion
  tags:
    owner: "${TEAM_${ENVIRONMENT}_OWNER}"
    cost-center: "${COST_CENTER_${BUSINESS_UNIT}}"
```

### Multi-Document Files

Support multiple resources in one file:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: Cluster
metadata:
  name: web-cluster
spec:
  # ... cluster configuration

---
apiVersion: matlas.mongodb.com/v1
kind: DatabaseUser
metadata:
  name: web-user
  dependsOn:
    - web-cluster
spec:
  # ... user configuration
```

### Resource Dependencies

Explicit dependency management:

```yaml
# User depends on cluster
apiVersion: matlas.mongodb.com/v1
kind: DatabaseUser
metadata:
  name: app-user
  dependsOn:
    - primary-cluster
    - secondary-cluster
spec:
  # ... user configuration
```

## Schema Versions

### v1 (Recommended)

- **Stability**: Production ready
- **Features**: All documented features
- **Compatibility**: Long-term support

### v1beta1

- **Stability**: Feature complete, API may change
- **Features**: Latest features, some experimental
- **Compatibility**: Upgrade path to v1

### v1alpha1

- **Stability**: Experimental
- **Features**: Cutting-edge features
- **Compatibility**: No guarantees

### Version Migration

```yaml
# Upgrade from v1beta1 to v1
apiVersion: matlas.mongodb.com/v1      # Changed from v1beta1
kind: Project
metadata:
  name: my-project
spec:
  # No other changes required for basic configurations
```

## Common Patterns

### Environment-Specific Configuration

```yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: "${ENVIRONMENT}-project"
  labels:
    environment: "${ENVIRONMENT}"
spec:
  name: "${PROJECT_NAME} (${ENVIRONMENT})"
  clusters:
    - metadata:
        name: "${ENVIRONMENT}-cluster"
      instanceSize: "${CLUSTER_SIZE}"
      diskSizeGB: ${DISK_SIZE}
      backupEnabled: ${BACKUP_ENABLED}
```

### High Availability Setup

```yaml
spec:
  clusters:
    - metadata:
        name: primary-cluster
      provider: AWS
      region: US_EAST_1
      instanceSize: M50
      replicationSpecs:
        - numShards: 2
          regionConfigs:
            - regionName: US_EAST_1
              priority: 7
              electableNodes: 3
            - regionName: US_WEST_2
              priority: 6
              electableNodes: 2
```

### Security-First Configuration

```yaml
spec:
  clusters:
    - metadata:
        name: secure-cluster
      encryption:
        encryptionAtRest: true
        awsKmsKeyId: "${KMS_KEY_ARN}"
      
  databaseUsers:
    - metadata:
        name: app-user
      x509Type: "MANAGED"
      roles:
        - roleName: "readWrite"
          databaseName: "app"
      scopes:
        - name: "secure-cluster"
          type: "CLUSTER"
  
  networkAccess:
    - metadata:
        name: application-sg
      awsSecurityGroup: "${APP_SECURITY_GROUP_ID}"
      comment: "Application security group only"
``` 