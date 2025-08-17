---
layout: page
title: Database Commands
description: Work directly with MongoDB databases, collections, users, roles, and indexes through Atlas clusters or direct connections.
permalink: /database/
---

# Database Commands

Work directly with MongoDB databases, collections, users, roles, and indexes through Atlas clusters or direct connections.

## Important Distinction: Database vs Atlas Management

**Database Commands** (`matlas database`) operate directly on MongoDB databases via connection strings:
- Create database-level users and custom roles directly in MongoDB
- Require database connection (direct connection string or Atlas cluster with temp user)
- Support granular, collection-level permissions
- Use MongoDB's native `createUser`, `createRole` commands

**Atlas Commands** (`matlas atlas`) operate via Atlas Admin API:
- Manage Atlas-level database users with built-in roles
- Use Atlas API authentication (API keys)
- Assign built-in MongoDB roles (read, readWrite, dbAdmin, etc.)
- Project-level user management

Use `database` commands for granular database-specific operations and `atlas` commands for centralized Atlas project management.



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
  # Custom database role
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

  # User that uses the custom role
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

## Database Users

Manage MongoDB database users directly in databases. These users are created using MongoDB's `createUser` command and can be assigned both built-in roles and custom roles created with `matlas database roles`.

**Note**: These are different from Atlas database users managed via `matlas atlas users`. Database users exist only within specific MongoDB databases, while Atlas users are managed centrally via the Atlas API.

### List database users
```bash
# Using connection string
matlas database users list \
  --connection-string "mongodb+srv://user:pass@host/" \
  --database myapp

# Using Atlas cluster with temporary user (recommended)
matlas database users list \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user
```

### Create database user
```bash
# Create user with built-in roles
matlas database users create dbuser \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --password "SecurePass123!" \
  --roles "readWrite@myapp,read@logs"

# Create user with custom roles and display password
matlas database users create appuser \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --password "SecurePass123!" \
  --roles "customRole@myapp" \
  --show-password
```

### Update database user
```bash
# Update password
matlas database users update dbuser \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --password "NewPass123!"

# Replace all roles
matlas database users update dbuser \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --roles "read@myapp,read@logs"

# Add roles incrementally
matlas database users update dbuser \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --add-roles "write@logs"
```

### Get database user details
```bash
matlas database users get dbuser \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user
```

### Delete database user
```bash
matlas database users delete dbuser \
  --cluster my-cluster \
  --project-id abc123 \
  --database myapp \
  --use-temp-user \
  --yes
```

**Important Notes:**
- Database users are scoped to specific databases
- When using `--use-temp-user`, the CLI creates a temporary Atlas user with `userAdmin` privileges for user management
- Database users can be assigned custom roles created with `matlas database roles create`
- Use `--verbose` flag to see detailed operation information