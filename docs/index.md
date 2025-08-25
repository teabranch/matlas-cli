---
layout: default
title: Home
nav_order: 1
description: "One CLI for MongoDB Atlas and MongoDB databases"
permalink: /
---

<div class="hero">
  <h1 class="hero-title">matlas</h1>
  <p class="hero-subtitle">One CLI for MongoDB Atlas and MongoDB databases. Matlas bridges the Atlas SDK and the MongoDB Go Driver so you can manage infrastructure and data workflows without switching tools.</p>
  <div class="hero-actions">
    <a href="#getting-started" class="btn btn-primary">Get Started</a>
    <a href="https://github.com/teabranch/matlas-cli" class="btn btn-outline" target="_blank">View on GitHub</a>
  </div>
</div>

## Core Features

<div class="features grid grid-cols-2">
  <div class="card feature">
    <div class="feature-icon">🔐</div>
    <h3 class="feature-title">Authentication & Configuration</h3>
    <p class="feature-description">Flexible authentication with API keys, config files, environment variables, and macOS Keychain integration.</p>
    <a href="{{ '/auth/' | relative_url }}" class="btn btn-outline">Learn more</a>
  </div>
  
  <div class="card feature">
    <div class="feature-icon">☁️</div>
    <h3 class="feature-title">Atlas Commands</h3>
    <p class="feature-description">Manage MongoDB Atlas projects, clusters, users, and networking with intuitive commands.</p>
    <a href="{{ '/atlas/' | relative_url }}" class="btn btn-outline">Learn more</a>
  </div>
  
  <div class="card feature">
    <div class="feature-icon">🗄️</div>
    <h3 class="feature-title">Database Commands</h3>
    <p class="feature-description">Work directly with MongoDB databases, collections, and indexes through Atlas or direct connections.</p>
    <a href="{{ '/database/' | relative_url }}" class="btn btn-outline">Learn more</a>
  </div>
  
  <div class="card feature">
    <div class="feature-icon">🔍</div>
    <h3 class="feature-title">Discovery & Export</h3>
    <p class="feature-description">Discover existing Atlas resources and convert to infrastructure-as-code format with database-level resource enumeration.</p>
    <a href="{{ '/discovery/' | relative_url }}" class="btn btn-outline">Learn more</a>
  </div>
  
  <div class="card feature">
    <div class="feature-icon">🏗️</div>
    <h3 class="feature-title">Infrastructure Workflows</h3>
    <p class="feature-description">Terraform-inspired infrastructure-as-code workflows: discover, plan, diff, apply, destroy.</p>
    <a href="{{ '/infra/' | relative_url }}" class="btn btn-outline">Learn more</a>
  </div>
  
  <div class="card feature">
    <div class="feature-icon">📚</div>
    <h3 class="feature-title">Examples & Patterns</h3>
    <p class="feature-description">Comprehensive YAML examples and usage patterns for all resource types and infrastructure scenarios.</p>
    <a href="{{ '/examples/' | relative_url }}" class="btn btn-outline">View examples</a>
  </div>
</div>

---

## Why Choose matlas?

✅ **Single Mental Model** - Unified interface for both Atlas infrastructure and database operations

✅ **Terraform-Inspired** - Familiar workflow: Discover → Plan/Diff → Apply

✅ **Consistent Interface** - Same flags, output formats, and ergonomics across all commands

✅ **MongoDB Native** - Built on official Atlas SDK and MongoDB Go Driver

---

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

# 5. Explore examples for more patterns
matlas examples --help
```

### 📚 Ready-to-use Examples

Jump-start your infrastructure with our comprehensive [examples collection]({{ '/examples/' | relative_url }}):

- **[Discovery Examples]({{ '/examples/discovery/' | relative_url }})** - Convert existing Atlas resources to code
- **[Cluster Examples]({{ '/examples/clusters/' | relative_url }})** - Development to production cluster configurations  
- **[User Management]({{ '/examples/users/' | relative_url }})** - Database users and authentication patterns
- **[Custom Roles]({{ '/examples/roles/' | relative_url }})** - Granular permission management
- **[Network Access]({{ '/examples/network/' | relative_url }})** - IP allowlisting and security rules
- **[Infrastructure Patterns]({{ '/examples/infrastructure/' | relative_url }})** - Complete infrastructure workflows