---
layout: default
title: Database Commands
nav_order: 4
has_children: false
description: Work directly with MongoDB databases, collections, users, roles, and indexes through Atlas clusters or direct connections.
permalink: /database/
---

# Database Commands

Work directly with MongoDB databases, collections, users, roles, and indexes through Atlas clusters or direct connections.

## Important Distinction: Database vs Atlas Management

**Database Commands** (`matlas database`) operate directly on MongoDB databases via connection strings:
- Create and manage custom roles with granular privileges
- Manage databases, collections, and indexes
- Require database connection (direct connection string or Atlas cluster with temp user)
- Support collection-level permissions and custom role definitions

**Atlas Commands** (`matlas atlas`) operate via Atlas Admin API:
- **All user management** happens through Atlas API (there is no direct MongoDB user creation)
- Users created via Atlas API automatically propagate to MongoDB databases
- Assign built-in MongoDB roles (read, readWrite, dbAdmin, etc.) and custom roles
- Project-level user management with centralized authentication

**User Management**: All database users must be created through `matlas atlas users` commands. Users created in Atlas automatically become available in MongoDB databases after propagation.



---

## Authentication Methods

Matlas supports three authentication methods for database operations:

### Method 1: Temporary User (Recommended)
```bash
--cluster <cluster-name> --project-id <project-id> --use-temp-user
```
- **How it works**: Creates a temporary Atlas database user with required permissions
- **Automatic cleanup**: User is automatically deleted after operation
- **Security**: Uses Atlas API keys, no permanent credentials needed
- **Best for**: Automation, CI/CD pipelines, one-off operations

### Method 2: Manual User Credentials
```bash
--cluster <cluster-name> --project-id <project-id> --username <user> --password <pass>
```
- **How it works**: Uses existing database user credentials
- **Requirements**: Both username and password must be provided
- **Best for**: Using existing database users with specific permissions

### Method 3: Direct Connection String
```bash
--connection-string "mongodb+srv://user:pass@cluster.mongodb.net/"
```
- **How it works**: Direct MongoDB connection with embedded credentials
- **Full control**: Complete control over connection parameters
- **Best for**: Custom connection requirements, external clusters

## Connection Requirements

**Database Creation**: All database creation operations require the `--collection` parameter because MongoDB databases are created lazily when the first collection is added. This ensures the database is immediately visible in Atlas UI.

**Authentication**: You must use exactly one authentication method per command. Mixing methods (e.g., `--use-temp-user` with `--username`) will result in an error.

---

## Databases

### List databases
```bash
# Direct connection
matlas database list --connection-string "mongodb+srv://user:pass@host/"

# Via Atlas cluster
matlas database list --cluster <name> --project-id <id> [--use-temp-user] [--database <db>]
```

### Create database
```bash
# Direct connection (requires collection for immediate visibility)
matlas database create <database-name> \
  --connection-string "mongodb+srv://user:pass@host/" \
  --collection <collection-name>

# Via Atlas cluster with temporary user (recommended)
matlas database create <database-name> \
  --cluster <name> \
  --project-id <id> \
  --collection <collection-name> \
  --use-temp-user

# Via Atlas cluster with manual credentials
matlas database create <database-name> \
  --cluster <name> \
  --project-id <id> \
  --collection <collection-name> \
  --username <db-user> \
  --password <db-password>
```

**Important**: Database creation requires the `--collection` parameter because MongoDB databases are created lazily when the first collection is added. This ensures the database is immediately visible in Atlas UI.

### Delete database
```bash
# Direct connection
matlas database delete <database-name> --connection-string "mongodb+srv://user:pass@host/" [--yes]

# Via Atlas cluster
matlas database delete <database-name> --cluster <name> --project-id <id> [--yes]
```

## Collections

### List collections
```bash
matlas database collections list \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name>
```

### Create collection
```bash
# Basic collection
matlas database collections create <collection-name> \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name>

# Capped collection
matlas database collections create <collection-name> \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name> \
  --capped \
  --size 1048576 \
  --max-documents 1000
```

### Delete collection
```bash
matlas database collections delete <collection-name> \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name> \
  [--yes]
```

## Indexes

### List indexes
```bash
matlas database collections indexes list \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name> \
  --collection <collection-name>
```

### Create indexes
```bash
# Single field index
matlas database collections indexes create field1:1 \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name> \
  --collection <collection-name> \
  [--name <index-name>]

# Compound index
matlas database collections indexes create field1:1 field2:-1 field3:1 \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name> \
  --collection <collection-name> \
  [--name <index-name>]

# Index with options
matlas database collections indexes create email:1 \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name> \
  --collection <collection-name> \
  --name "unique_email_idx" \
  --unique \
  --sparse \
  --background
```

### Delete index
```bash
matlas database collections indexes delete <index-name> \
  [--connection-string "..." | --cluster <name> --project-id <id>] \
  --database <database-name> \
  --collection <collection-name> \
  [--yes]
```

## Index field specifications

When creating indexes, specify field order using these values:

| Value | Description |
|:------|:------------|
| `1` | Ascending order |
| `-1` | Descending order |

Note: Text, 2d/2dsphere, and hashed index types are not supported in this build via the CLI. Use MongoDB drivers or Atlas UI/APIs for those index types.

## Index options

| Flag | Description |
|:-----|:------------|
| `--name` | Custom index name |
| `--unique` | Enforce uniqueness constraint |
| `--sparse` | Only index documents with the field |
| `--background` | Build index in background |

## Examples

### Complete workflow
```bash
# 1. List databases
matlas database list --cluster my-cluster --project-id abc123

# 2. Create a new database and collection
matlas database create inventory --cluster my-cluster --project-id abc123
matlas database collections create products --cluster my-cluster --project-id abc123 --database inventory

# 3. Create indexes for the collection
matlas database collections indexes create sku:1 --cluster my-cluster --project-id abc123 --database inventory --collection products --name "sku_idx" --unique
matlas database collections indexes create category:1 price:-1 --cluster my-cluster --project-id abc123 --database inventory --collection products --name "category_price_idx"

# 4. List the created indexes
matlas database collections indexes list --cluster my-cluster --project-id abc123 --database inventory --collection products
```

---

## Custom roles

Define database-level custom roles and manage them directly via CLI or YAML.

### CLI Usage

Custom roles can be created, listed, and managed using the CLI:

### List roles
```bash
# Using connection string
matlas database roles list \
  --connection-string "mongodb+srv://user:pass@host/" \
  --database myapp

# Using Atlas cluster with temporary user (recommended)
matlas database roles list \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user
```

### Create role
```bash
# Using connection string
matlas database roles create myCustomRole \
  --connection-string "mongodb+srv://user:pass@host/" \
  --database myapp \
  --privileges "read@myapp,insert@myapp.logs" \
  --inherited-roles "read@myapp"

# Using Atlas cluster with temporary user (recommended)
matlas database roles create myCustomRole \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --privileges "read@myapp,insert@myapp.logs" \
  --inherited-roles "read@myapp"
```

Privilege format: `action@resource`
- action: `read`, `readWrite`, `insert`, `update`, `remove`, etc.
- resource: `db` or `db.collection` (e.g., `mydb.users`)

**Important Notes:**
- When using `--use-temp-user`, the CLI automatically creates a temporary user with `dbAdminAnyDatabase` and `readWriteAnyDatabase` privileges required for role creation
- Role creation includes enhanced retry logic to handle Atlas user propagation delays
- Use `--verbose` flag to see detailed authentication and retry information

### YAML Role Creation

Custom roles can also be defined in YAML ApplyDocuments for infrastructure-as-code workflows:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: custom-roles-example
resources:
  # Custom database role (created directly in MongoDB)
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: app-role
      labels:
        purpose: application
    spec:
      roleName: appRole
      databaseName: myapp
      privileges:
        # Collection-level privileges
        - actions: ["find", "insert", "update"]
          resource:
            database: myapp
            collection: users
        - actions: ["find"]
          resource:
            database: myapp
            collection: logs
        # Database-level privileges
        - actions: ["listCollections", "listIndexes"]
          resource:
            database: myapp
      inheritedRoles:
        - roleName: read
          databaseName: myapp

  # Atlas database user that uses the custom role
  # Note: All users must be created via Atlas API - they propagate to MongoDB databases
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-user
    spec:
      projectName: "My Project"
      username: app-user
      authDatabase: admin
      password: "${APP_USER_PASSWORD}"
      roles:
        - roleName: appRole
          databaseName: myapp
        - roleName: read
          databaseName: admin
```

Apply the configuration:
```bash
export APP_USER_PASSWORD="SecurePassword123!"
matlas infra apply -f custom-roles.yaml --project-id abc123 --auto-approve
```

### Get role
```bash
# Using connection string
matlas database roles get myCustomRole \
  --connection-string "mongodb+srv://user:pass@host/" \
  --database myapp

# Using Atlas cluster with temporary user
matlas database roles get myCustomRole \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user
```

### Delete role
```bash
# Using connection string
matlas database roles delete myCustomRole \
  --connection-string "mongodb+srv://user:pass@host/" \
  --database myapp \
  --yes

# Using Atlas cluster with temporary user
matlas database roles delete myCustomRole \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --yes
```

## Database Users (Atlas-Managed)

**Important**: In MongoDB Atlas, all database users must be created and managed through the Atlas API. Direct MongoDB `createUser` commands are not supported.

All database user management is handled via `matlas atlas users` commands. Users created through Atlas automatically propagate to MongoDB databases and can access databases according to their assigned roles.

### User Management via Atlas API

For complete user management documentation, see the [Atlas Commands](/atlas/) documentation. Here are the essential commands:

```bash
# Create Atlas database user (propagates to MongoDB databases)
matlas atlas users create \
  --project-id abc123 \
  --username dbuser \
  --database-name admin \
  --password "SecurePass123!" \
  --roles "readWrite@myapp,read@logs"

# List all Atlas database users
matlas atlas users list --project-id abc123

# Update user roles
matlas atlas users update \
  --project-id abc123 \
  --username dbuser \
  --database-name admin \
  --roles "read@myapp,read@logs"

# Get user details
matlas atlas users get \
  --project-id abc123 \
  --username dbuser \
  --database-name admin

# Delete user
matlas atlas users delete \
  --project-id abc123 \
  --username dbuser \
  --database-name admin
```

### Role Assignment with Custom Roles

Users created via Atlas can be assigned both built-in MongoDB roles and custom roles created with `matlas database roles`:

```bash
# Create custom role first
matlas database roles create appRole \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --privileges "read@myapp,insert@myapp.logs"

# Create user with custom role via Atlas
matlas atlas users create \
  --project-id abc123 \
  --username appuser \
  --database-name admin \
  --password "SecurePass123!" \
  --roles "appRole@myapp,read@admin"
```

### Propagation and Access

- Users created via Atlas API automatically propagate to all clusters in the project
- Role assignments determine which databases and collections the user can access
- Custom roles created with `matlas database roles` can be assigned to Atlas users
- Use `--verbose` with Atlas commands to see detailed operation information

**Note**: The `matlas database users` commands are not functional in Atlas environments and will redirect you to use `matlas atlas users` instead.