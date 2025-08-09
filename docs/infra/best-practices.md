# Best Practices for Large-Scale Configurations

This document provides guidelines and best practices for managing large-scale Atlas deployments using `matlas infra`. These practices help ensure reliability, maintainability, and security across complex environments.

## Table of Contents

- [Configuration Organization](#configuration-organization)
- [Resource Management](#resource-management)
- [Security Best Practices](#security-best-practices)
- [Performance Optimization](#performance-optimization)
- [Environment Management](#environment-management)
- [Change Management](#change-management)
- [Monitoring and Observability](#monitoring-and-observability)
- [Disaster Recovery](#disaster-recovery)

## Configuration Organization

### File Structure

Organize configurations for maintainability and scalability:

```
atlas-infrastructure/
├── environments/
│   ├── production/
│   │   ├── clusters.yaml
│   │   ├── users.yaml
│   │   ├── network.yaml
│   │   └── main.yaml
│   ├── staging/
│   │   └── ...
│   └── development/
│       └── ...
├── modules/
│   ├── base-cluster/
│   │   └── template.yaml
│   ├── analytics-cluster/
│   │   └── template.yaml
│   └── security-users/
│       └── template.yaml
├── shared/
│   ├── common-labels.yaml
│   ├── security-groups.yaml
│   └── network-ranges.yaml
├── scripts/
│   ├── deploy.sh
│   ├── validate.sh
│   └── rollback.sh
└── docs/
    ├── runbooks/
    └── architecture/
```

### Modular Configuration

Break large configurations into focused, reusable modules:

```yaml
# modules/base-cluster/template.yaml
apiVersion: matlas.mongodb.com/v1
kind: Cluster
metadata:
  name: "${ENVIRONMENT}-${PURPOSE}"
  labels:
    environment: "${ENVIRONMENT}"
    purpose: "${PURPOSE}"
    managed-by: matlas-cli
    module: base-cluster
spec:
  projectName: "${PROJECT_NAME}"
  provider: "${CLOUD_PROVIDER:-AWS}"
  region: "${REGION}"
  instanceSize: "${INSTANCE_SIZE}"
  diskSizeGB: ${DISK_SIZE}
  backupEnabled: ${BACKUP_ENABLED:-true}
  mongodbVersion: "${MONGODB_VERSION:-7.0}"
  
  # Standard auto-scaling for all clusters
  autoScaling:
    compute:
      enabled: true
      scaleDownEnabled: ${SCALE_DOWN_ENABLED:-true}
      minInstanceSize: "${MIN_INSTANCE_SIZE}"
      maxInstanceSize: "${MAX_INSTANCE_SIZE}"
    diskGB:
      enabled: true

  # Security defaults
  encryption:
    encryptionAtRest: ${ENCRYPTION_ENABLED:-true}
    awsKmsKeyId: "${KMS_KEY_ARN}"
```

### Configuration Composition

Compose larger configurations from modules:

```yaml
# environments/production/main.yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: "${PROJECT_NAME}-prod"
  labels:
    environment: production
    tier: enterprise
spec:
  name: "${PROJECT_NAME} Production"
  organizationId: "${ATLAS_ORG_ID}"
  
  clusters:
    # Include base cluster template
    - $include: "../../modules/base-cluster/template.yaml"
      $vars:
        PURPOSE: "primary"
        INSTANCE_SIZE: "M50"
        DISK_SIZE: 1000
        MIN_INSTANCE_SIZE: "M30"
        MAX_INSTANCE_SIZE: "M100"
    
    - $include: "../../modules/analytics-cluster/template.yaml"
      $vars:
        PURPOSE: "analytics"
        INSTANCE_SIZE: "M40"
        DISK_SIZE: 2000
```

## Resource Management

### Naming Conventions

Use consistent, descriptive naming patterns:

```yaml
# Resource naming pattern: {environment}-{purpose}-{sequence}
metadata:
  name: "${ENVIRONMENT}-primary-01"
  name: "${ENVIRONMENT}-analytics-01"
  name: "${ENVIRONMENT}-backup-01"

# User naming pattern: {environment}-{application}-{role}
username: "${ENVIRONMENT}-${APPLICATION}-app"
username: "${ENVIRONMENT}-${APPLICATION}-readonly"
username: "${ENVIRONMENT}-monitoring-collector"

# Network access naming: {environment}-{source}-{purpose}
metadata:
  name: "${ENVIRONMENT}-app-servers"
  name: "${ENVIRONMENT}-vpn-access"
  name: "${ENVIRONMENT}-cicd-pipeline"
```

### Resource Sizing Guidelines

Define sizing standards for different environments:

```yaml
# sizing-standards.yaml
sizing:
  production:
    primary_cluster:
      instanceSize: "M50"
      diskSizeGB: 1000
      minInstanceSize: "M30"
      maxInstanceSize: "M100"
    analytics_cluster:
      instanceSize: "M40"
      diskSizeGB: 2000
      minInstanceSize: "M30"
      maxInstanceSize: "M80"
  
  staging:
    primary_cluster:
      instanceSize: "M30"
      diskSizeGB: 500
      minInstanceSize: "M20"
      maxInstanceSize: "M50"
    analytics_cluster:
      instanceSize: "M20"
      diskSizeGB: 1000
      minInstanceSize: "M10"
      maxInstanceSize: "M40"
  
  development:
    primary_cluster:
      instanceSize: "M10"
      diskSizeGB: 100
      minInstanceSize: "M10"
      maxInstanceSize: "M20"
```

### Resource Dependencies

Explicit dependency management for complex configurations:

```yaml
spec:
  clusters:
    - metadata:
        name: primary-cluster
        labels:
          dependency-order: "1"
      # ... cluster configuration
    
    - metadata:
        name: analytics-cluster
        labels:
          dependency-order: "2"
        dependsOn:
          - primary-cluster
      # ... analytics cluster configuration
  
  databaseUsers:
    - metadata:
        name: app-user
        labels:
          dependency-order: "3"
        dependsOn:
          - primary-cluster
          - analytics-cluster
      # ... user configuration
```

## Security Best Practices

### Secrets Management

Never hardcode secrets in configuration files:

```yaml
# ❌ Never do this
password: "hardcoded-password-123"
awsKmsKeyId: "arn:aws:kms:US_EAST_1:123456789012:key/abcd1234"

# ✅ Use environment variables
password: "${APP_PASSWORD:?Application password is required}"
awsKmsKeyId: "${KMS_KEY_ARN:?KMS key ARN is required}"

# ✅ External secret management
password: "${APP_PASSWORD_FROM_VAULT}"
awsKmsKeyId: "${KMS_KEY_FROM_PARAMETER_STORE}"
```

### Role-Based Access Control

Implement least-privilege access patterns:

```yaml
# Application-specific users with minimal permissions
databaseUsers:
  - metadata:
      name: web-app-user
    username: "${ENVIRONMENT}-web-app"
    roles:
      - roleName: "readWrite"
        databaseName: "webapp"
      - roleName: "read"
        databaseName: "shared"
    scopes:
      - name: "primary-cluster"
        type: "CLUSTER"
  
  # Analytics users with read-only access
  - metadata:
      name: analytics-user
    username: "${ENVIRONMENT}-analytics"
    roles:
      - roleName: "read"
        databaseName: "webapp"
      - roleName: "read"
        databaseName: "analytics"
    scopes:
      - name: "analytics-cluster"
        type: "CLUSTER"
  
  # Monitoring users with cluster-level read access
  - metadata:
      name: monitoring-user
    username: "${ENVIRONMENT}-monitoring"
    roles:
      - roleName: "clusterMonitor"
        databaseName: "admin"
    scopes:
      - name: "primary-cluster"
        type: "CLUSTER"
      - name: "analytics-cluster"
        type: "CLUSTER"
```

### Network Security

Implement defense-in-depth network access:

```yaml
networkAccess:
  # Production - security groups only
  - metadata:
      name: prod-app-servers
      labels:
        security-level: high
    awsSecurityGroup: "${PROD_APP_SG_ID}"
    comment: "Production application servers"
  
  # Monitoring - specific monitoring subnet
  - metadata:
      name: prod-monitoring
      labels:
        security-level: medium
    cidr: "${MONITORING_SUBNET_CIDR}"
    comment: "Monitoring and observability systems"
  
  # Emergency access - time-limited
  - metadata:
      name: emergency-access
      labels:
        security-level: emergency
        temporary: "true"
    ipAddress: "${EMERGENCY_IP}"
    comment: "Emergency access - auto-expires"
    deleteAfterDate: "${EMERGENCY_EXPIRATION}"
```

### Encryption Standards

Mandatory encryption for production environments:

```yaml
# Production encryption requirements
encryption:
  encryptionAtRest: true
  awsKmsKeyId: "${PROD_KMS_KEY_ARN:?Production KMS key is required}"

# Multi-region encryption with region-specific keys
replicationSpecs:
  - numShards: 1
    regionConfigs:
      - regionName: US_EAST_1
        encryption:
          awsKmsKeyId: "${EAST_KMS_KEY_ARN}"
      - regionName: US_WEST_2
        encryption:
          awsKmsKeyId: "${WEST_KMS_KEY_ARN}"
```

## Performance Optimization

### Cluster Sizing Strategy

Right-size clusters based on workload patterns:

```yaml
# High-traffic production cluster
- metadata:
    name: prod-api-cluster
    labels:
      workload: high-traffic
      optimization: performance
  instanceSize: "M50"
  autoScaling:
    compute:
      enabled: true
      scaleDownEnabled: false  # Never scale down during business hours
      minInstanceSize: "M40"
      maxInstanceSize: "M100"

# Analytics cluster optimized for reads
- metadata:
    name: prod-analytics-cluster
    labels:
      workload: analytics
      optimization: cost-performance
  instanceSize: "M40"
  biConnector:
    enabled: true
    readPreference: "secondary"
  autoScaling:
    compute:
      enabled: true
      scaleDownEnabled: true
      minInstanceSize: "M20"
      maxInstanceSize: "M80"
```

### Auto-scaling Configuration

Configure auto-scaling based on usage patterns:

```yaml
# Production auto-scaling - conservative
autoScaling:
  compute:
    enabled: true
    scaleDownEnabled: true
    minInstanceSize: "M30"
    maxInstanceSize: "M100"
  diskGB:
    enabled: true

# Development auto-scaling - aggressive cost optimization
autoScaling:
  compute:
    enabled: true
    scaleDownEnabled: true
    minInstanceSize: "M10"
    maxInstanceSize: "M30"
  diskGB:
    enabled: true
```

### Regional Deployment Strategy

Distribute workloads geographically for performance:

```yaml
# Multi-region deployment for low latency
replicationSpecs:
  - numShards: 1
    regionConfigs:
      # Primary region - East Coast users
      - regionName: US_EAST_1
        priority: 7
        electableNodes: 3
        analyticsNodes: 1
        readOnlyNodes: 0
      
      # Secondary region - West Coast users  
      - regionName: US_WEST_2
        priority: 6
        electableNodes: 2
        analyticsNodes: 0
        readOnlyNodes: 2  # Read replicas for West Coast
      
      # Tertiary region - European users
      - regionName: EU_WEST_1
        priority: 5
        electableNodes: 2
        analyticsNodes: 0
        readOnlyNodes: 1
```

## Environment Management

### Environment Isolation

Maintain strict isolation between environments:

```yaml
# Environment-specific organization isolation
production:
  organizationId: "${PROD_ATLAS_ORG_ID}"
  projectName: "${PROJECT_NAME}-prod"
  kmsKeyArn: "${PROD_KMS_KEY_ARN}"
  securityGroups:
    app: "${PROD_APP_SG_ID}"
    monitoring: "${PROD_MONITORING_SG_ID}"

staging:
  organizationId: "${STAGING_ATLAS_ORG_ID}"
  projectName: "${PROJECT_NAME}-staging"
  kmsKeyArn: "${STAGING_KMS_KEY_ARN}"
  securityGroups:
    app: "${STAGING_APP_SG_ID}"
    monitoring: "${STAGING_MONITORING_SG_ID}"

development:
  organizationId: "${DEV_ATLAS_ORG_ID}"
  projectName: "${PROJECT_NAME}-dev"
  # Development may use shared keys for cost optimization
  kmsKeyArn: "${SHARED_DEV_KMS_KEY_ARN}"
```

### Environment-Specific Configurations

Tailor configurations to environment needs:

```yaml
# Production - high availability and security
production:
  cluster:
    instanceSize: "M50"
    backupEnabled: true
    encryptionEnabled: true
    multiRegion: true
  users:
    x509AuthEnabled: true
    passwordExpiration: "90 days"
  monitoring:
    alertingEnabled: true
    24x7Support: true

# Staging - production-like but cost-optimized
staging:
  cluster:
    instanceSize: "M30"
    backupEnabled: true
    encryptionEnabled: true
    multiRegion: false
  users:
    x509AuthEnabled: false
    passwordExpiration: "180 days"
  monitoring:
    alertingEnabled: true
    24x7Support: false

# Development - minimal cost, maximum flexibility
development:
  cluster:
    instanceSize: "M10"
    backupEnabled: false
    encryptionEnabled: false
    multiRegion: false
  users:
    x509AuthEnabled: false
    passwordExpiration: "365 days"
  monitoring:
    alertingEnabled: false
    24x7Support: false
```

## Change Management

### Version Control Strategy

Implement comprehensive version control:

```bash
# Git workflow for Atlas changes
git flow init
git flow feature start add-analytics-cluster
# Make configuration changes
git add environments/production/clusters.yaml
git commit -m "feat: add analytics cluster to production

- Instance size: M40 
- Disk: 2TB
- Auto-scaling enabled
- BI Connector enabled for Tableau integration"

git flow feature finish add-analytics-cluster
# Create pull request for review
```

### Change Approval Process

Implement review and approval workflows:

```yaml
# .github/workflows/atlas-changes.yml
name: Atlas Configuration Changes
on:
  pull_request:
    paths:
      - 'environments/**'
      - 'modules/**'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Validate configurations
        run: |
          for env in development staging production; do
            matlas infra validate -f environments/$env/main.yaml
          done
  
  plan:
    needs: validate
    runs-on: ubuntu-latest
    steps:
      - name: Generate plans
        run: |
          matlas infra plan -f environments/staging/main.yaml --output-file staging-plan.json
          matlas infra plan -f environments/production/main.yaml --output-file production-plan.json
  
  deploy-staging:
    needs: plan
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to staging
        run: matlas infra -f environments/staging/main.yaml
  
  deploy-production:
    needs: deploy-staging
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    environment: production  # Requires manual approval
    steps:
      - name: Deploy to production
        run: matlas infra -f environments/production/main.yaml
```

### Rollback Procedures

Maintain rollback capabilities:

```bash
#!/bin/bash
# scripts/rollback.sh

set -e

ENVIRONMENT="${1:?Environment is required}"
BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"

echo "Creating backup of current state..."
mkdir -p "$BACKUP_DIR"
matlas infra show --project-id "${PROJECT_ID}" --output yaml > "$BACKUP_DIR/current-state.yaml"

echo "Rolling back to previous configuration..."
git log --oneline -n 10 environments/"$ENVIRONMENT"/
read -p "Enter commit hash to rollback to: " COMMIT_HASH

git checkout "$COMMIT_HASH" -- environments/"$ENVIRONMENT"/

echo "Applying rollback configuration..."
matlas infra -f environments/"$ENVIRONMENT"/main.yaml

echo "Rollback completed. Backup saved to $BACKUP_DIR"
```

### Safe Resource Destruction

Implement safe practices for destroying Atlas resources:

```bash
#!/bin/bash
# scripts/safe-destroy.sh

set -e

ENVIRONMENT="${1:?Environment is required}"
PROJECT_ID="${2:-}"

echo "⚠️  DANGER: This will destroy Atlas resources!"
echo "Environment: $ENVIRONMENT"

# 1. Validate environment
if [[ "$ENVIRONMENT" == "production" ]]; then
    echo "❌ Production destruction requires manual approval"
    read -p "Type 'DESTROY PRODUCTION' to continue: " CONFIRMATION
    if [[ "$CONFIRMATION" != "DESTROY PRODUCTION" ]]; then
        echo "Destruction cancelled"
        exit 1
    fi
fi

# 2. Create backup before destruction
BACKUP_DIR="backups/destroy-$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"
echo "Creating backup..."
matlas infra show --project-id "${PROJECT_ID:-${ATLAS_PROJECT_ID}}" --output yaml > "$BACKUP_DIR/pre-destroy-state.yaml"

# 3. Show what will be destroyed
echo "Resources to be destroyed:"
if [[ -n "$PROJECT_ID" ]]; then
    # Discovery-only mode for complete cleanup
    matlas infra destroy --discovery-only --project-id "$PROJECT_ID" --dry-run
else
    # Configuration-based destruction
    matlas infra destroy -f environments/"$ENVIRONMENT"/main.yaml --dry-run
fi

# 4. Final confirmation
read -p "Proceed with destruction? (yes/no): " FINAL_CONFIRM
if [[ "$FINAL_CONFIRM" != "yes" ]]; then
    echo "Destruction cancelled"
    exit 1
fi

# 5. Execute destruction with proper ordering
echo "Executing destruction..."
if [[ -n "$PROJECT_ID" ]]; then
    # Discovery-only mode - destroys everything
    matlas infra destroy --discovery-only --project-id "$PROJECT_ID"
else
    # Configuration-based destruction
    matlas infra destroy -f environments/"$ENVIRONMENT"/main.yaml
fi

echo "✅ Destruction completed. Backup saved to $BACKUP_DIR"
```

### Resource Cleanup Best Practices

1. **Always use dry-run first**:
   ```bash
   # Preview what will be destroyed
   matlas infra destroy -f config.yaml --dry-run
   ```

2. **Use discovery-only for complete cleanup**:
   ```bash
   # Clean up ALL resources in a project
   matlas infra destroy --discovery-only --project-id PROJECT_ID --dry-run
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

3. **Incremental destruction for complex environments**:
   ```bash
   # Destroy by resource type to control order
   matlas infra destroy -f config.yaml --target users --dry-run
   matlas infra destroy -f config.yaml --target users
   
   matlas infra destroy -f config.yaml --target network-access --dry-run  
   matlas infra destroy -f config.yaml --target network-access
   
   matlas infra destroy -f config.yaml --target clusters --dry-run
   matlas infra destroy -f config.yaml --target clusters
   ```

4. **Handle orphaned resources**:
   ```bash
   # Find orphaned resources not in config
   matlas infra show --project-id PROJECT_ID > current-state.yaml
   matlas infra diff -f config.yaml
   
   # Clean up with discovery-only if needed
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

5. **Emergency cleanup procedures**:
   ```bash
   # In case of stuck destroy operations
   # 1. Check Atlas console for resource status
   # 2. Use discovery-only mode to clean up remaining resources
   matlas infra destroy --discovery-only --project-id PROJECT_ID --force
   ```

## Monitoring and Observability

### Resource Labeling Strategy

Implement comprehensive labeling:

```yaml
metadata:
  name: "${ENVIRONMENT}-primary-cluster"
  labels:
    # Environment classification
    environment: "${ENVIRONMENT}"
    tier: "${TIER}"  # development, staging, production
    
    # Operational metadata
    team: "${TEAM}"
    cost-center: "${COST_CENTER}"
    project: "${PROJECT_NAME}"
    
    # Technical metadata
    purpose: "primary"
    workload-type: "oltp"
    backup-tier: "critical"
    
    # Compliance and security
    data-classification: "${DATA_CLASSIFICATION}"
    compliance-scope: "${COMPLIANCE_SCOPE}"
    encryption-required: "true"
    
    # Monitoring and alerting
    monitoring-level: "enhanced"
    alert-severity: "critical"
    
  annotations:
    # Detailed documentation
    description: "Primary OLTP cluster for ${PROJECT_NAME} ${ENVIRONMENT}"
    owner: "${TEAM}-team@company.com"
    runbook: "https://wiki.company.com/runbooks/${PROJECT_NAME}"
    
    # Operational information
    created-by: "matlas-cli"
    last-updated: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    change-ticket: "${CHANGE_TICKET_ID}"
    
    # Business context
    business-impact: "${BUSINESS_IMPACT}"
    sla-requirement: "${SLA_REQUIREMENT}"
    maintenance-window: "${MAINTENANCE_WINDOW}"
```

### Configuration Drift Detection

Monitor for configuration drift:

```bash
#!/bin/bash
# scripts/drift-detection.sh

for environment in development staging production; do
  echo "Checking drift for $environment..."
  
  # Generate current state snapshot
  matlas infra show --project-id "${PROJECT_ID}" --output yaml > "current-$environment.yaml"
  
  # Compare with desired state
  if ! matlas infra diff -f "environments/$environment/main.yaml" --output json > "drift-$environment.json"; then
    echo "⚠️  Configuration drift detected in $environment"
    # Send alert to monitoring system
    curl -X POST "$SLACK_WEBHOOK" -d "{
      \"text\": \"Atlas configuration drift detected in $environment environment\",
      \"attachments\": [{
        \"color\": \"warning\",
        \"fields\": [{
          \"title\": \"Environment\",
          \"value\": \"$environment\",
          \"short\": true
        }]
      }]
    }"
  else
    echo "✅ No drift detected in $environment"
  fi
done
```

## Disaster Recovery

### Backup Strategy

Implement comprehensive backup strategies:

```yaml
# Multi-tier backup configuration
clusters:
  # Production - maximum protection
  - metadata:
      name: prod-primary
      labels:
        backup-tier: critical
    backupEnabled: true
    # Continuous backup with point-in-time recovery
    # Managed by Atlas, retained for 30 days
  
  # Additional cross-region backup cluster
  - metadata:
      name: prod-backup
      labels:
        backup-tier: critical
        purpose: disaster-recovery
    provider: GCP  # Different provider for DR
    region: us-central1
    instanceSize: M20  # Smaller for cost
    backupEnabled: true
```

### Multi-Region Deployment

Design for geographic redundancy:

```yaml
# Enterprise disaster recovery setup
replicationSpecs:
  - numShards: 2
    regionConfigs:
      # Primary site - East Coast
      - regionName: US_EAST_1
        priority: 7
        electableNodes: 3
        analyticsNodes: 2
        readOnlyNodes: 0
      
      # Hot standby - West Coast  
      - regionName: US_WEST_2
        priority: 6
        electableNodes: 3
        analyticsNodes: 1
        readOnlyNodes: 1
      
      # Cold standby - Europe
      - regionName: EU_WEST_1
        priority: 5
        electableNodes: 2
        analyticsNodes: 0
        readOnlyNodes: 2
```

### Recovery Testing

Regular disaster recovery testing:

```bash
#!/bin/bash
# scripts/dr-test.sh

echo "Starting disaster recovery test..."

# 1. Create test scenario
TEST_PROJECT="dr-test-$(date +%s)"
export TEST_PROJECT

# 2. Deploy to test environment
matlas infra -f environments/production/main.yaml --project-id "$TEST_PROJECT"

# 3. Simulate failure scenarios
echo "Testing primary region failure..."
# Disable primary region nodes
# Test application failover
# Measure RTO and RPO

# 4. Test recovery procedures
echo "Testing recovery procedures..."
# Restore from backup
# Verify data integrity
# Test application connectivity

# 5. Cleanup
echo "Cleaning up test resources..."
matlas infra destroy -f environments/production/main.yaml --project-id "$TEST_PROJECT"

echo "DR test completed successfully"
```

This comprehensive best practices guide provides the foundation for managing large-scale Atlas deployments with confidence and reliability. 