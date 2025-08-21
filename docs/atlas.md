---
layout: default
title: Atlas Commands
nav_order: 3
has_children: false
description: Manage MongoDB Atlas infrastructure including projects, users, clusters, and networking.
permalink: /atlas/
---

# Atlas Commands

Manage MongoDB Atlas infrastructure including projects, users, clusters, and networking.



---

## Projects

Manage MongoDB Atlas projects within your organization.

### List projects
```bash
matlas atlas projects list [--org-id <org>]
```

### Get project details
```bash
matlas atlas projects get --project-id <id>
```

### Create project
```bash
matlas atlas projects create <name> --org-id <org> [--tag k=v]...
```

### Update project
```bash
matlas atlas projects update --project-id <id> [--name new] [--tag k=v]... [--clear-tags]
```

### Delete project
```bash
matlas atlas projects delete <project-id> [--yes]
```

## Users

Manage Atlas database users within Atlas projects.

**Note**: These are Atlas-managed database users created via the Atlas Admin API. They are assigned built-in MongoDB roles and managed centrally at the project level. For database-specific users with custom roles, use `matlas database users` commands.

### List users
```bash
matlas atlas users list --project-id <id> [--page N --limit M --all]
```

### Get user details
```bash
matlas atlas users get <username> --project-id <id> [--database-name admin]
```

### Create user
```bash
# Create user (password will be prompted)
matlas atlas users create \
  --project-id <id> \
  --username <username> \
  --database-name admin \
  --roles role@db[,role@db]

# Create user and display password
matlas atlas users create \
  --project-id <id> \
  --username <username> \
  --database-name admin \
  --roles role@db[,role@db] \
  --show-password
```

### Update user
```bash
matlas atlas users update <username> \
  --project-id <id> \
  [--database-name admin] \
  [--password] \
  [--roles ...]
```

### Delete user
```bash
matlas atlas users delete <username> --project-id <id> [--database-name admin] [--yes]
```

## Network access

Configure IP access lists for your Atlas clusters.

### List network access entries
```bash
matlas atlas network list --project-id <id>
```

### Get network access entry
```bash
matlas atlas network get <ip-or-cidr> --project-id <id>
```

### Create network access entry
```bash
# Allow specific IP
matlas atlas network create --project-id <id> --ip-address x.x.x.x [--comment "Description"]

# Allow CIDR block
matlas atlas network create --project-id <id> --cidr-block x.x.x.x/24 [--comment "Description"]

# Allow AWS security group
matlas atlas network create --project-id <id> --aws-security-group sg-xxxxxxxxx [--comment "Description"]
```

### Delete network access entry
```bash
matlas atlas network delete <ip-or-cidr> --project-id <id> [--yes]
```

## Network peering

Network peering enables private connectivity between your Atlas clusters and cloud infrastructure.

### Available commands
```bash
matlas atlas network-peering list --project-id <id>
matlas atlas network-peering get <peering-id> --project-id <id>
matlas atlas network-peering create --project-id <id> [options]
matlas atlas network-peering delete <peering-id> --project-id <id> [--yes]
```

Use `matlas atlas network-peering <command> --help` for detailed flag information.

## Network containers

Network containers define the CIDR blocks for your Atlas clusters in specific cloud regions.

### Available commands
```bash
matlas atlas network-containers list --project-id <id>
matlas atlas network-containers get <container-id> --project-id <id>
matlas atlas network-containers create --project-id <id> [options]
matlas atlas network-containers delete <container-id> --project-id <id> [--yes]
```

Use `matlas atlas network-containers <command> --help` for detailed flag information.

## Clusters

**Note:** Cluster management is primarily handled through the [infrastructure workflows](/infra/). Use `matlas discover` and `matlas infra` commands for cluster operations.

## Atlas Search

Atlas Search provides full-text search capabilities for your MongoDB collections.

### List Search Indexes

```bash
# List all search indexes in a cluster
matlas atlas search list --project-id <project-id> --cluster <cluster-name>

# List search indexes for a specific collection
matlas atlas search list --project-id <project-id> --cluster <cluster-name> \
  --database sample_mflix --collection movies
```

### Create Search Index

```bash
# Create a basic search index with dynamic mapping
matlas atlas search create \
  --project-id <project-id> \
  --cluster <cluster-name> \
  --database sample_mflix \
  --collection movies \
  --name default

# Create a vector search index (for AI/ML use cases)
matlas atlas search create \
  --project-id <project-id> \
  --cluster <cluster-name> \
  --database sample_mflix \
  --collection movies \
  --name plot_vector_index \
  --type vectorSearch
```

## VPC Endpoints

VPC Endpoints provide secure, private connectivity to your Atlas clusters using AWS PrivateLink, Azure Private Link, or Google Cloud Private Service Connect.

### List VPC Endpoint Services

```bash
# List all VPC endpoint services
matlas atlas vpc-endpoints list --project-id <project-id>

# List VPC endpoints for specific cloud provider
matlas atlas vpc-endpoints list --project-id <project-id> --cloud-provider AWS
```

### Get VPC Endpoint Service Details

```bash
# Get details for a specific VPC endpoint service
matlas atlas vpc-endpoints get \
  --project-id <project-id> \
  --cloud-provider AWS \
  --endpoint-id <service-id>
```

### Create VPC Endpoint Service

```bash
# Create a VPC endpoint service for AWS
matlas atlas vpc-endpoints create \
  --project-id <project-id> \
  --cloud-provider AWS \
  --region us-east-1

# Create a VPC endpoint service for Azure
matlas atlas vpc-endpoints create \
  --project-id <project-id> \
  --cloud-provider AZURE \
  --region eastus

# Create a VPC endpoint service for GCP
matlas atlas vpc-endpoints create \
  --project-id <project-id> \
  --cloud-provider GCP \
  --region us-central1
```

### Update VPC Endpoint Service

```bash
# Update a VPC endpoint service (most properties are immutable)
matlas atlas vpc-endpoints update \
  --project-id <project-id> \
  --cloud-provider AWS \
  --endpoint-id <service-id>
```

### Delete VPC Endpoint Service

```bash
# Delete a VPC endpoint service
matlas atlas vpc-endpoints delete \
  --project-id <project-id> \
  --cloud-provider AWS \
  --endpoint-id <service-id>

# Delete without confirmation prompt
matlas atlas vpc-endpoints delete \
  --project-id <project-id> \
  --cloud-provider AWS \
  --endpoint-id <service-id> \
  --yes
```

### YAML Configuration

VPC endpoints can also be managed declaratively via YAML:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: vpc-endpoint-example
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: production-vpc-endpoint
      labels:
        environment: production
        provider: aws
    spec:
      projectName: "your-project-id"
      cloudProvider: "AWS"
      region: "us-east-1"
```

Apply the configuration:
```bash
# Plan VPC endpoint changes
matlas infra plan -f vpc-endpoint.yaml --preserve-existing

# Apply VPC endpoint configuration
matlas infra apply -f vpc-endpoint.yaml --preserve-existing --auto-approve
```

## Feature availability

**Updated:** The following features are now available:
- ✅ Search indexes: `matlas atlas search list` - Full functionality
- 🚧 Search indexes: `matlas atlas search create` - Basic implementation  
- ✅ VPC endpoints: `matlas atlas vpc-endpoints ...` - Full functionality with CLI and YAML support