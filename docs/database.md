---
title: Database Commands
nav_order: 4
---

# Database Commands
{: .no_toc }

Work directly with MongoDB databases, collections, and indexes through Atlas clusters or direct connections.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Connection methods

Matlas supports two ways to connect to MongoDB databases:

### Direct connection
```bash
--connection-string "mongodb+srv://user:pass@cluster.mongodb.net/"
```

### Via Atlas cluster  
```bash
--cluster <cluster-name> --project-id <project-id>
```

{: .highlight }
When using Atlas cluster connection, you can optionally use `--use-temp-user` to create a temporary database user for the operation.

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
# Direct connection
matlas database create <database-name> --connection-string "mongodb+srv://user:pass@host/"

# Via Atlas cluster  
matlas database create <database-name> --cluster <name> --project-id <id>
```

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
| `text` | Text index |
| `2d` | 2D index |
| `2dsphere` | 2D sphere index |
| `hashed` | Hashed index |

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

