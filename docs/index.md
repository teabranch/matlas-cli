---
layout: home
title: Home
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
    <a href="auth" class="btn btn-outline">Learn more</a>
  </div>
  
  <div class="card feature">
    <div class="feature-icon">☁️</div>
    <h3 class="feature-title">Atlas Commands</h3>
    <p class="feature-description">Manage MongoDB Atlas projects, clusters, users, and networking with intuitive commands.</p>
    <a href="atlas" class="btn btn-outline">Learn more</a>
  </div>
  
  <div class="card feature">
    <div class="feature-icon">🗄️</div>
    <h3 class="feature-title">Database Commands</h3>
    <p class="feature-description">Work directly with MongoDB databases, collections, and indexes through Atlas or direct connections.</p>
    <a href="database" class="btn btn-outline">Learn more</a>
  </div>
  
  <div class="card feature">
    <div class="feature-icon">🏗️</div>
    <h3 class="feature-title">Infrastructure Workflows</h3>
    <p class="feature-description">Terraform-inspired infrastructure-as-code workflows: discover, plan, diff, apply, destroy.</p>
    <a href="infra" class="btn btn-outline">Learn more</a>
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
```