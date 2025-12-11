---
layout: default
title: Custom Roles
parent: Examples
nav_order: 4
description: Granular permission management with custom database roles
permalink: /examples/roles/
---

# Custom Roles Examples

Granular permission management using custom database roles for precise access control beyond built-in MongoDB roles.

## Basic Custom Role

Create a custom role with specific collection-level permissions:

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: custom-role-basic
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: app-data-access
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
          databaseName: logs

  # User that uses the custom role
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-service-user
    spec:
      projectName: "My Project"
      username: app-service-user
      authDatabase: admin
      password: "${SERVICE_USER_PASSWORD}"
      roles:
        - roleName: appDataAccess
          databaseName: myapp
```
{% endraw %}

## Advanced Custom Roles

Complex role definitions with multiple privilege types and inheritance:

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: custom-roles-comprehensive
  labels:
    purpose: advanced-permissions
resources:
  # Analytics role with broad read access
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: analytics-role
      labels:
        purpose: analytics
    spec:
      roleName: analyticsReader
      databaseName: analytics
      privileges:
        # Collection-level read permissions
        - actions: ["find", "listIndexes"]
          resource:
            database: analytics
            collection: events
        - actions: ["find", "listIndexes"]
          resource:
            database: analytics
            collection: metrics
        - actions: ["find"]
          resource:
            database: analytics
            collection: reports
        
        # Database-level permissions
        - actions: ["listCollections"]
          resource:
            database: analytics
      
      inheritedRoles:
        - roleName: read
          databaseName: reference-data

  # Application role with granular write permissions
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: ecommerce-app-role
      labels:
        purpose: application
        domain: ecommerce
    spec:
      roleName: ecommerceApp
      databaseName: ecommerce
      privileges:
        # Product management permissions
        - actions: ["find", "insert", "update", "remove", "createIndex"]
          resource:
            database: ecommerce
            collection: products
        
        # Order management permissions
        - actions: ["find", "insert", "update"]
          resource:
            database: ecommerce
            collection: orders
        
        # Customer data permissions
        - actions: ["find", "update"]
          resource:
            database: ecommerce
            collection: customers
        
        # Read-only access to configuration
        - actions: ["find"]
          resource:
            database: ecommerce
            collection: config
        
        # Database-level permissions
        - actions: ["listCollections", "listIndexes"]
          resource:
            database: ecommerce

  # Administrative role with cluster-level permissions
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: database-manager-role
      labels:
        purpose: administration
    spec:
      roleName: databaseManager
      databaseName: admin
      privileges:
        # Cluster administration
        - actions: ["serverStatus", "connPoolStats"]
          resource:
            cluster: true
        
        # Database creation and management
        - actions: ["listDatabases", "dbStats"]
          resource:
            cluster: true
      
      inheritedRoles:
        - roleName: dbAdmin
          databaseName: ecommerce
        - roleName: dbAdmin
          databaseName: analytics

  # Users with custom roles
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: analytics-service
    spec:
      projectName: "My Project"
      username: analytics-service
      authDatabase: admin
      password: "${ANALYTICS_SERVICE_PASSWORD}"
      roles:
        - roleName: analyticsReader
          databaseName: analytics

  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: ecommerce-api
    spec:
      projectName: "My Project"
      username: ecommerce-api
      authDatabase: admin
      password: "${ECOMMERCE_API_PASSWORD}"
      roles:
        - roleName: ecommerceApp
          databaseName: ecommerce
      scopes:
        - name: "production-cluster"
          type: CLUSTER

  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: database-manager
    spec:
      projectName: "My Project"
      username: database-manager
      authDatabase: admin
      password: "${DATABASE_MANAGER_PASSWORD}"
      roles:
        - roleName: databaseManager
          databaseName: admin
```
{% endraw %}

## Environment Variables

Set these environment variables for custom role examples:

```bash
export SERVICE_USER_PASSWORD='SecureServicePass123!'
export ANALYTICS_SERVICE_PASSWORD='SecureAnalyticsServicePass123!'
export ECOMMERCE_API_PASSWORD='SecureEcommercePass123!'
export DATABASE_MANAGER_PASSWORD='SecureManagerPass123!'
```

## Common Actions Reference

### Read Operations
- `find` - Query documents
- `listCollections` - List collections in database
- `listIndexes` - List indexes on collections

### Write Operations
- `insert` - Insert new documents
- `update` - Modify existing documents
- `remove` - Delete documents
- `createCollection` - Create new collections

### Index Operations
- `createIndex` - Create indexes
- `dropIndex` - Remove indexes
- `reIndex` - Rebuild indexes

### Administrative Operations
- `dbAdmin` - Database administration
- `userAdmin` - User management
- `serverStatus` - Server statistics
- `connPoolStats` - Connection pool statistics

## Usage Examples

### Create Custom Roles

```bash
# Validate role definitions
matlas infra validate -f custom-roles-comprehensive.yaml

# Plan role deployment
matlas infra plan -f custom-roles-comprehensive.yaml --show-dependencies

# Apply roles and users together
matlas infra apply -f custom-roles-comprehensive.yaml --preserve-existing
```

### CLI Role Management

```bash
# Create custom role via CLI
matlas database roles create myCustomRole \
  --project-id <project-id> \
  --cluster <cluster-name> \
  --database myapp \
  --privileges "find,insert,update@myapp.users" \
  --inherited-roles "read@logs"

# List custom roles
matlas database roles list \
  --project-id <project-id> \
  --cluster <cluster-name> \
  --database myapp

# Get role details
matlas database roles get myCustomRole \
  --project-id <project-id> \
  --cluster <cluster-name> \
  --database myapp
```

## Best Practices

### Principle of Least Privilege
Grant only the minimum permissions required:

```yaml
# ✅ Specific actions and resources
privileges:
  - actions: ["find", "update"]
    resource:
      database: myapp
      collection: users

# ❌ Overly broad permissions
privileges:
  - actions: ["*"]
    resource:
      cluster: true
```

### Use Inheritance
Leverage existing roles to reduce complexity:

```yaml
inheritedRoles:
  - roleName: read
    databaseName: reference-data  # ✅ Inherit standard permissions
```

### Resource Scoping
Be specific about resource targets:

```yaml
# ✅ Collection-specific
resource:
  database: myapp
  collection: users

# ✅ Database-specific
resource:
  database: analytics

# ⚠️ Cluster-wide (use sparingly)
resource:
  cluster: true
```

## Related Examples

- [User Management]({{ '/examples/users/' | relative_url }}) - Assign custom roles to users
- [Infrastructure Patterns]({{ '/examples/infrastructure/' | relative_url }}) - Role-based access patterns
- [Discovery]({{ '/examples/discovery/' | relative_url }}) - Discover existing custom roles