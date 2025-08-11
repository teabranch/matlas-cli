---
title: "Home"
nav_order: 1
permalink: /
---

# matlas
{: .fs-9 }

One CLI for MongoDB Atlas and MongoDB databases
{: .fs-6 .fw-300 }

Matlas bridges the Atlas SDK and the MongoDB Go Driver so you can manage infrastructure and data workflows without switching tools.

[Get started now](#getting-started){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View it on GitHub](https://github.com/teabranch/matlas-cli){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## Core Features

### 🔐 Authentication & Configuration
Flexible authentication with API keys, config files, environment variables, and macOS Keychain integration.
[Learn more](auth){: .btn .btn-outline }

### ☁️ Atlas Commands  
Manage MongoDB Atlas projects, clusters, users, and networking with intuitive commands.
[Learn more](atlas){: .btn .btn-outline }

### 🗄️ Database Commands
Work directly with MongoDB databases, collections, and indexes through Atlas or direct connections.
[Learn more](database){: .btn .btn-outline }

### 🏗️ Infrastructure Workflows
Terraform-inspired infrastructure-as-code workflows: discover, plan, diff, apply, destroy.
[Learn more](infra){: .btn .btn-outline }

---

## Why Choose matlas?

✅ **Single Mental Model** - Unified interface for both Atlas infrastructure and database operations

✅ **Terraform-Inspired** - Familiar workflow: Discover → Plan/Diff → Apply  

✅ **Consistent Interface** - Same flags, output formats, and ergonomics across all commands

✅ **MongoDB Native** - Built on official Atlas SDK and MongoDB Go Driver

## Getting started

### Installation

```bash
# Download from GitHub releases
# Or build from source
go install github.com/teabranch/matlas-cli@latest
```

### Quick setup

```bash
# Set up authentication
export ATLAS_API_KEY="your-private-key"
export ATLAS_PUB_KEY="your-public-key"

# List your projects
matlas atlas projects list

# Discover a project's resources
matlas discover --project-id <id> --include-databases --output yaml
```

### Example workflow

```bash
# 1. Discover current state
matlas discover --project-id abc123 --include-databases -o project.yaml

# 2. Edit the configuration
vim project.yaml

# 3. Preview changes  
matlas infra diff -f project.yaml

# 4. Apply changes
matlas infra apply -f project.yaml
```

