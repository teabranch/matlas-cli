# Migration Guide: From Imperative to Declarative

This guide helps you migrate from using imperative `matlas atlas` commands to declarative `matlas infra` configurations. The declarative approach provides better consistency, reproducibility, and change management for your Atlas infrastructure.

## Table of Contents

- [Why Migrate to Declarative?](#why-migrate-to-declarative)
- [Migration Strategy](#migration-strategy)
- [Command Mapping](#command-mapping)
- [Discovery and Export](#discovery-and-export)
- [Step-by-Step Migration](#step-by-step-migration)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)
- [Troubleshooting](#troubleshooting)

## Why Migrate to Declarative?

### Benefits of Declarative Configuration

| Imperative Approach | Declarative Approach |
|-------------------|---------------------|
| Manual command execution | Automated, repeatable deployments |
| No change tracking | Full audit trail and version control |
| Error-prone manual steps | Validation and dry-run capabilities |
| Difficult to reproduce | Infrastructure as code |
| No rollback capability | Easy rollback and recovery |
| Manual state management | Automatic drift detection |

### When to Migrate

**Ideal candidates for migration**:
- Multiple environments (dev/staging/prod)
- Complex Atlas projects with many resources
- Team-based development
- Need for change tracking and approvals
- Automated CI/CD pipelines
- Compliance requirements

**Consider keeping imperative for**:
- Quick one-off operations
- Exploratory or temporary resources
- Emergency fixes
- Learning and experimentation

## Migration Strategy

### 1. Assessment Phase

```bash
# Discover current Atlas resources
matlas infra show --project-id <your-project-id> --output yaml > current-state.yaml

# Review existing imperative scripts
find . -name "*.sh" -exec grep -l "matlas atlas" {} \;
```

### 2. Planning Phase

1. **Inventory existing resources**:
   - Clusters and their configurations
   - Database users and permissions
   - Network access rules
   - Project settings

2. **Identify environments**:
   - Development, staging, production
   - Different regions or providers
   - Temporary vs permanent resources

3. **Choose migration approach**:
   - **Big Bang**: Migrate everything at once
   - **Incremental**: Migrate by environment or resource type
   - **Hybrid**: Keep some imperative, migrate critical paths

### 3. Implementation Phase

1. **Create configuration files**
2. **Validate configurations**
3. **Test in development**
4. **Gradual rollout to production**

## Command Mapping

### Cluster Operations

#### Creating Clusters

**Imperative**:
```bash
matlas atlas clusters create \
  --projectId "507f1f77bcf86cd799439011" \
  --name "MyCluster" \
  --provider AWS \
  --region US_EAST_1 \
  --instanceSize M30 \
  --diskSizeGB 100 \
  --backup
```

**Declarative**:
```yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: my-project
spec:
  name: "My Project"
  organizationId: "507f1f77bcf86cd799439011"
  clusters:
    - metadata:
        name: my-cluster
      provider: AWS
      region: US_EAST_1
      instanceSize: M30
      diskSizeGB: 100
      backupEnabled: true
```

#### Updating Clusters

**Imperative**:
```bash
matlas atlas clusters update \
  --projectId "507f1f77bcf86cd799439011" \
  --name "MyCluster" \
  --instanceSize M40
```

**Declarative**:
```yaml
# Update instanceSize in configuration file
spec:
  clusters:
    - metadata:
        name: my-cluster
      instanceSize: M40  # Changed from M30
```

```bash
# Apply the change
matlas infra -f config.yaml
```

### Database User Operations

#### Creating Users

**Imperative**:
```bash
matlas atlas users create \
  --projectId "507f1f77bcf86cd799439011" \
  --username "myapp" \
  --password "secure-password" \
  --role "readWrite@mydb"
```

**Declarative**:
```yaml
spec:
  databaseUsers:
    - metadata:
        name: app-user
      username: "myapp"
      databaseName: "admin"
      password: "${APP_PASSWORD}"
      roles:
        - roleName: "readWrite"
          databaseName: "mydb"
```

#### Updating User Roles

**Imperative**:
```bash
matlas atlas users update \
  --projectId "507f1f77bcf86cd799439011" \
  --username "myapp" \
  --role "readWrite@mydb,read@analytics"
```

**Declarative**:
```yaml
spec:
  databaseUsers:
    - metadata:
        name: app-user
      username: "myapp"
      roles:
        - roleName: "readWrite"
          databaseName: "mydb"
        - roleName: "read"
          databaseName: "analytics"  # Added new role
```

### Network Access Operations

#### Adding IP Access

**Imperative**:
```bash
matlas atlas networkAccess create \
  --projectId "507f1f77bcf86cd799439011" \
  --cidr "203.0.113.0/24" \
  --comment "Office network"
```

**Declarative**:
```yaml
spec:
  networkAccess:
    - metadata:
        name: office-access
      cidr: "203.0.113.0/24"
      comment: "Office network"
```

## Discovery and Export

### Export Current Configuration

Use the `show` command to export your current Atlas configuration:

```bash
# Export entire project configuration
matlas infra show --project-id "507f1f77bcf86cd799439011" --output yaml > project-export.yaml

# Export specific resource types
matlas infra show --project-id "507f1f77bcf86cd799439011" --resource-type clusters --output yaml > clusters.yaml
matlas infra show --project-id "507f1f77bcf86cd799439011" --resource-type users --output yaml > users.yaml
matlas infra show --project-id "507f1f77bcf86cd799439011" --resource-type network --output yaml > network.yaml
```

### Clean Up Exported Configuration

The exported configuration may need cleanup:

```yaml
# Remove read-only fields
# ❌ Remove these
status:
  phase: Ready
  lastUpdate: "2024-01-15T10:30:00Z"

# ❌ Remove auto-generated IDs  
id: "507f1f77bcf86cd799439011"

# ✅ Keep configuration fields
metadata:
  name: my-cluster
spec:
  provider: AWS
  region: US_EAST_1
  instanceSize: M30
```

### Identify Configuration Patterns

Look for patterns in your existing setup:

```bash
# Find common instance sizes
matlas atlas clusters list --projectId <id> | jq -r '.[] | .instanceSizeName' | sort | uniq -c

# Find common regions
matlas atlas clusters list --projectId <id> | jq -r '.[] | .regionName' | sort | uniq -c

# Find user patterns
matlas atlas users list --projectId <id> | jq -r '.[] | .roles[].roleName' | sort | uniq -c
```

## Step-by-Step Migration

### Step 1: Create Base Configuration

Start with a minimal working configuration:

```yaml
# base-config.yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: "${PROJECT_NAME}"
  labels:
    environment: "${ENVIRONMENT}"
    migrated-from: imperative
spec:
  name: "${PROJECT_DISPLAY_NAME}"
  organizationId: "${ATLAS_ORG_ID}"
  
  # Start with minimal resources
  clusters: []
  databaseUsers: []
  networkAccess: []
```

### Step 2: Migrate Clusters First

Add cluster configurations one by one:

```yaml
spec:
  clusters:
    # Migrate most critical cluster first
    - metadata:
        name: production-primary
        labels:
          criticality: high
          migration-order: "1"
      provider: AWS
      region: US_EAST_1
      instanceSize: M30
      diskSizeGB: 100
      backupEnabled: true
```

### Step 3: Add Database Users

Migrate users after clusters are stable:

```yaml
spec:
  databaseUsers:
    # Application users first
    - metadata:
        name: prod-app-user
        labels:
          purpose: application
          migration-order: "2"
      username: "myapp"
      databaseName: "admin"
      password: "${APP_PASSWORD}"
      roles:
        - roleName: "readWrite"
          databaseName: "production"
      scopes:
        - name: "production-primary"
          type: "CLUSTER"
```

### Step 4: Configure Network Access

Add network rules last:

```yaml
spec:
  networkAccess:
    # Most restrictive rules first
    - metadata:
        name: production-sg
        labels:
          migration-order: "3"
      awsSecurityGroup: "sg-0123456789abcdef0"
      comment: "Production application servers"
```

### Step 5: Validate and Test

```bash
# Validate configuration
matlas infra validate -f config.yaml

# Test with dry run
matlas infra -f config.yaml --dry-run

# Compare with current state
matlas infra diff -f config.yaml
```

### Step 6: Gradual Migration

```bash
# Apply to development first
ENVIRONMENT=development matlas infra -f config.yaml

# Then staging
ENVIRONMENT=staging matlas infra -f config.yaml

# Finally production (with extra caution)
ENVIRONMENT=production matlas infra -f config.yaml --timeout 60m
```

## Best Practices

### Environment Variables

Replace hardcoded values with environment variables:

```yaml
# ❌ Hardcoded
spec:
  name: "MyApp Production"
  organizationId: "507f1f77bcf86cd799439011"

# ✅ Parameterized
spec:
  name: "${PROJECT_NAME} ${ENVIRONMENT}"
  organizationId: "${ATLAS_ORG_ID}"
```

### Resource Naming

Use consistent naming conventions:

```yaml
# ❌ Inconsistent
metadata:
  name: prodCluster
  name: dev-cluster-01
  name: staging_analytics

# ✅ Consistent
metadata:
  name: "${ENVIRONMENT}-primary"
  name: "${ENVIRONMENT}-analytics"
  name: "${ENVIRONMENT}-backup"
```

### Security Considerations

1. **Passwords**: Use environment variables, never hardcode
   ```yaml
   password: "${APP_PASSWORD:?Password is required}"
   ```

2. **API Keys**: Store in secure environment variables
   ```bash
   export ATLAS_PUBLIC_KEY="your-key"
   export ATLAS_PRIVATE_KEY="your-secret"
   ```

3. **Network Access**: Use least privilege principle
   ```yaml
   # ❌ Too broad
   cidr: "0.0.0.0/0"
   
   # ✅ Specific
   awsSecurityGroup: "sg-specific-id"
   ```

### Version Control

1. **Git Repository Structure**:
   ```
   atlas-config/
   ├── environments/
   │   ├── development.yaml
   │   ├── staging.yaml
   │   └── production.yaml
   ├── templates/
   │   └── base-template.yaml
   └── scripts/
       ├── deploy.sh
       └── validate.sh
   ```

2. **Commit Messages**:
   ```bash
   git commit -m "feat: add analytics cluster to production"
   git commit -m "fix: update user permissions for staging"
   git commit -m "chore: migrate from imperative commands"
   ```

## Common Patterns

### Multi-Environment Setup

Create environment-specific configurations:

```yaml
# environments/production.yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: myapp-prod
spec:
  clusters:
    - metadata:
        name: prod-primary
      instanceSize: M50  # Larger for production
      
# environments/development.yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: myapp-dev
spec:
  clusters:
    - metadata:
        name: dev-primary
      instanceSize: M10  # Smaller for development
```

### Template-Based Approach

Use a single template for all environments:

```yaml
# template.yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: "${PROJECT_NAME}-${ENVIRONMENT}"
spec:
  clusters:
    - metadata:
        name: "${ENVIRONMENT}-primary"
      instanceSize: "${CLUSTER_SIZE}"
      diskSizeGB: ${DISK_SIZE}
```

```bash
# Deploy to different environments
PROJECT_NAME=myapp ENVIRONMENT=dev CLUSTER_SIZE=M10 DISK_SIZE=20 \
  matlas infra -f template.yaml

PROJECT_NAME=myapp ENVIRONMENT=prod CLUSTER_SIZE=M50 DISK_SIZE=500 \
  matlas infra -f template.yaml
```

### Batch Migration Script

Create scripts to automate migration:

```bash
#!/bin/bash
# migrate.sh

set -e

PROJECT_ID="507f1f77bcf86cd799439011"
CONFIG_FILE="project-config.yaml"

echo "Starting migration from imperative to declarative..."

# 1. Export current state
echo "Exporting current configuration..."
matlas infra show --project-id "$PROJECT_ID" --output yaml > current-state.yaml

# 2. Validate new configuration
echo "Validating new configuration..."
matlas infra validate -f "$CONFIG_FILE"

# 3. Show differences
echo "Showing differences..."
matlas infra diff -f "$CONFIG_FILE"

# 4. Confirm before applying
read -p "Apply changes? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Applying configuration..."
    matlas infra -f "$CONFIG_FILE"
else
    echo "Migration cancelled."
    exit 1
fi

echo "Migration completed successfully!"
```

## Troubleshooting

### Common Migration Issues

1. **Resource Already Exists**:
   ```
   Error: cluster 'MyCluster' already exists
   ```
   
   **Solution**: Use existing resource names in configuration
   ```yaml
   metadata:
     name: MyCluster  # Use exact existing name
   ```

2. **Permission Conflicts**:
   ```
   Error: user 'myapp' already exists with different permissions
   ```
   
   **Solution**: Match existing permissions or update gradually
   ```yaml
   # First, match existing permissions exactly
   roles:
     - roleName: "readWrite"
       databaseName: "mydb"
   ```

3. **Network Rule Overlaps**:
   ```
   Error: CIDR range conflicts with existing rule
   ```
   
   **Solution**: Export existing rules and consolidate
   ```bash
   matlas infra show --project-id <id> --resource-type network
   ```

### Rollback Procedures

If migration goes wrong, you can rollback:

```bash
# 1. Restore from exported state
matlas infra -f current-state.yaml

# 2. Or destroy and recreate manually
matlas infra destroy -f new-config.yaml
# Then recreate using imperative commands
```

### Validation Strategies

1. **Pre-migration validation**:
   ```bash
   # Check current state
   matlas infra show --project-id <id> > before-migration.yaml
   
   # Validate new config
   matlas infra validate -f new-config.yaml
   
   # Preview changes
   matlas infra diff -f new-config.yaml
   ```

2. **Post-migration verification**:
   ```bash
   # Check new state
   matlas infra show --project-id <id> > after-migration.yaml
   
   # Compare states
   diff before-migration.yaml after-migration.yaml
   ```

### Getting Help

1. **Start small**: Migrate one resource type at a time
2. **Use development environment**: Test migration process first
3. **Keep backups**: Export current state before changes
4. **Read documentation**: Review [Configuration Schema](configuration-schema.md) and [Troubleshooting Guide](troubleshooting.md)
5. **Dry run everything**: Always test with `--dry-run` first

## Migration Checklist

- [ ] **Assessment**
  - [ ] Export current Atlas configuration
  - [ ] Inventory all resources (clusters, users, network rules)
  - [ ] Identify environments and patterns
  
- [ ] **Planning**
  - [ ] Choose migration strategy (big bang vs incremental)
  - [ ] Create configuration templates
  - [ ] Plan rollback procedures
  
- [ ] **Preparation**
  - [ ] Set up version control for configurations
  - [ ] Create environment variable templates
  - [ ] Write validation and deployment scripts
  
- [ ] **Testing**
  - [ ] Validate configuration syntax
  - [ ] Test with dry run
  - [ ] Test in development environment
  
- [ ] **Migration**
  - [ ] Backup current state
  - [ ] Apply to staging environment
  - [ ] Verify staging results
  - [ ] Apply to production environment
  - [ ] Verify production results
  
- [ ] **Post-Migration**
  - [ ] Update documentation
  - [ ] Train team on new workflows
  - [ ] Retire old imperative scripts
  - [ ] Set up monitoring for configuration drift 