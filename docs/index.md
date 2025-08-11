---
title: "Home"
nav_order: 1
permalink: /
layout: default
---

<div class="hero">
  <h1>🚀 matlas</h1>
  <p class="tagline">One CLI for MongoDB Atlas and MongoDB databases</p>
  <p>Matlas bridges the Atlas SDK and the MongoDB Go Driver so you can manage infrastructure and data workflows without switching tools.</p>
  
  <div style="margin-top: 2rem;">
    <a href="#getting-started" class="btn btn-primary" style="margin-right: 1rem;">Get Started Now</a>
    <a href="https://github.com/teabranch/matlas-cli" class="btn" style="background: white; color: #333;">View on GitHub</a>
  </div>
</div>

## 🎯 Core Features

<div class="feature-grid">
  <div class="feature-card">
    <h3>🔐 Authentication & Configuration</h3>
    <p>Flexible authentication with API keys, config files, environment variables, and macOS Keychain integration.</p>
    <a href="auth">Learn more →</a>
  </div>
  
  <div class="feature-card">
    <h3>☁️ Atlas Commands</h3>
    <p>Manage MongoDB Atlas projects, clusters, users, and networking with intuitive commands.</p>
    <a href="atlas">Learn more →</a>
  </div>
  
  <div class="feature-card">
    <h3>🗄️ Database Commands</h3>
    <p>Work directly with MongoDB databases, collections, and indexes through Atlas or direct connections.</p>
    <a href="database">Learn more →</a>
  </div>
  
  <div class="feature-card">
    <h3>🏗️ Infrastructure Workflows</h3>
    <p>Terraform-inspired infrastructure-as-code workflows: discover, plan, diff, apply, destroy.</p>
    <a href="infra">Learn more →</a>
  </div>
</div>

## ✨ Why Choose matlas?

<div class="feature-grid">
  <div class="feature-card">
    <h3><span class="checkmark">✅</span> Single Mental Model</h3>
    <p>Unified interface for both Atlas infrastructure and database operations</p>
  </div>
  
  <div class="feature-card">
    <h3><span class="checkmark">✅</span> Terraform-Inspired</h3>
    <p>Familiar workflow: Discover → Plan/Diff → Apply</p>
  </div>
  
  <div class="feature-card">
    <h3><span class="checkmark">✅</span> Consistent Interface</h3>
    <p>Same flags, output formats, and ergonomics across all commands</p>
  </div>
  
  <div class="feature-card">
    <h3><span class="checkmark">✅</span> MongoDB Native</h3>
    <p>Built on official Atlas SDK and MongoDB Go Driver</p>
  </div>
</div>

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

