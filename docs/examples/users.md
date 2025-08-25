---
layout: default
title: User Management
parent: Examples
nav_order: 3
description: Database user and authentication examples
---

# User Management Examples

Database user configurations covering basic user creation, cluster-scoped access, and password management patterns.

## users-basic.yaml

Simple DatabaseUser with basic read role - perfect for getting started.

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: users-basic
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-user
    spec:
      projectName: "My Project"  # Replace with your Atlas project name
      username: app-user
      authDatabase: admin
      password: "${APP_USER_PASSWORD}"  # Set env var before running
      roles:
        - roleName: read
          databaseName: admin
```
{% endraw %}

## users-standalone-multiple.yaml

Multiple users with different roles and team labels for organizational management.

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: users-standalone-multiple
  labels:
    example: standalone-users
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-writer
      labels:
        team: app
    spec:
      projectName: "My Project"
      username: app-writer
      authDatabase: admin
      password: "${APP_WRITER_PASSWORD}"
      roles:
        - roleName: readWrite
          databaseName: appdb
        - roleName: read
          databaseName: logs

  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: analytics-reader
      labels:
        team: data
    spec:
      projectName: "My Project"
      username: analytics-reader
      authDatabase: admin
      password: "${ANALYTICS_PASSWORD}"
      roles:
        - roleName: read
          databaseName: analytics
        - roleName: read
          databaseName: reports
```
{% endraw %}

## users-scoped.yaml

Users scoped to specific clusters for enhanced security and access control.

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: users-scoped
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-user-scoped
      labels:
        team: app
    spec:
      projectName: "My Project"
      username: app-user-scoped
      authDatabase: admin
      password: "${APP_USER_SCOPED_PASSWORD}"
      roles:
        - roleName: readWrite
          databaseName: appdb
      scopes:
        - name: "production-cluster"
          type: CLUSTER

  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: analytics-user-scoped
      labels:
        team: data
    spec:
      projectName: "My Project"
      username: analytics-user-scoped
      authDatabase: admin
      password: "${ANALYTICS_SCOPED_PASSWORD}"
      roles:
        - roleName: read
          databaseName: analytics
      scopes:
        - name: "analytics-cluster"
          type: CLUSTER
```
{% endraw %}

## Environment Variables Setup

Set these environment variables before applying user configurations:

```bash
# Basic user passwords
export APP_USER_PASSWORD='SecureAppPass123!'
export APP_WRITER_PASSWORD='SecureWriterPass123!'
export ANALYTICS_PASSWORD='SecureAnalyticsPass123!'

# Scoped user passwords
export APP_USER_SCOPED_PASSWORD='SecureScopedPass123!'
export ANALYTICS_SCOPED_PASSWORD='SecureAnalyticsScopedPass123!'
```

## Usage Examples

### Basic User Management

```bash
# Create basic user
matlas infra validate -f users-basic.yaml
matlas infra apply -f users-basic.yaml

# Create multiple users at once
matlas infra apply -f users-standalone-multiple.yaml --preserve-existing
```

### Scoped User Management

```bash
# Apply scoped users (requires existing clusters)
matlas infra plan -f users-scoped.yaml --show-dependencies
matlas infra apply -f users-scoped.yaml --preserve-existing
```

### CLI User Management

```bash
# Create user directly via CLI
matlas atlas users create \
  --project-id <project-id> \
  --username myuser \
  --roles "readWrite@myapp,read@logs" \
  --show-password

# List existing users
matlas atlas users list --project-id <project-id> --output table

# Update user roles
matlas atlas users update myuser \
  --project-id <project-id> \
  --roles "readWrite@myapp,read@logs,read@analytics"
```

## Role Types

### Built-in MongoDB Roles

```yaml
roles:
  # Database-specific roles
  - roleName: read
    databaseName: myapp
  - roleName: readWrite
    databaseName: myapp
  - roleName: dbAdmin
    databaseName: myapp
  - roleName: userAdmin
    databaseName: myapp
  
  # Cluster-level roles (use admin database)
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

### Custom Roles

Reference custom roles created via `DatabaseRole` kind:

```yaml
roles:
  - roleName: myCustomRole
    databaseName: myapp
```

## Security Best Practices

### Cluster Scoping

Limit user access to specific clusters:

```yaml
spec:
  scopes:
    - name: "production-cluster"
      type: CLUSTER
    - name: "staging-cluster"
      type: CLUSTER
```

### Environment Variables

Always use environment variables for passwords:

```yaml
password: "${USER_PASSWORD}"  # ✅ Secure
password: "hardcoded-password"  # ❌ Insecure
```

### Minimal Permissions

Grant only required permissions:

```yaml
roles:
  - roleName: read              # ✅ Read-only when possible
    databaseName: analytics
  - roleName: readWriteAnyDatabase  # ❌ Avoid broad permissions
    databaseName: admin
```

## Related Examples

- [Custom Roles]({{ '/examples/roles/' | relative_url }}) - Create custom roles for users
- [Clusters]({{ '/examples/clusters/' | relative_url }}) - Clusters for user scoping
- [Infrastructure Patterns]({{ '/examples/infrastructure/' | relative_url }}) - Complete user management workflows