---
title: "Home"
nav_order: 1
permalink: /
---

# matlas
{: .fs-9 }

One CLI for MongoDB Atlas and MongoDB databases. Matlas bridges the Atlas SDK and the MongoDB Go Driver so you can manage infrastructure and data workflows without switching tools.
{: .fs-6 .fw-300 }

[Get started now](#getting-started){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/teabranch/matlas-cli){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## Quick navigation

<div class="code-example" markdown="1">

**Core features**
- [Authentication and configuration]({{ site.baseurl }}{% link auth.md %}) - Set up API keys and config
- [Atlas commands]({{ site.baseurl }}{% link atlas.md %}) - Manage clusters, users, and networking  
- [Database commands]({{ site.baseurl }}{% link database.md %}) - Work with databases and collections
- [Infrastructure workflows]({{ site.baseurl }}{% link infra.md %}) - Discover, plan, diff, apply, destroy

</div>

## Why matlas?

<div class="code-example" markdown="1">

✅ **Single mental model** for both Atlas and database operations  
✅ **Terraform-inspired workflow**: Discover → Plan/Diff → Apply  
✅ **Consistent interface**: Same flags, output formats, and ergonomics across all commands  
✅ **MongoDB native**: Built on Atlas SDK and MongoDB Go Driver  

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

