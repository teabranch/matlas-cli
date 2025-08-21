---
layout: default
title: Authentication & Configuration
nav_order: 2
description: Configure matlas to authenticate with MongoDB Atlas and set default behaviors.
permalink: /auth/
---

# Authentication & Configuration

Configure matlas to authenticate with MongoDB Atlas and set default behaviors.



---

## Configuration precedence

Matlas uses the following configuration precedence (later sources override earlier ones):

1. **Defaults** - `output=table`, `timeout=30s`
2. **Config file** - `~/.matlas/config.yaml` or via `--config`/`ATLAS_CONFIG_FILE`
3. **Environment variables** - Prefix `ATLAS_`, e.g. `ATLAS_API_KEY`, `ATLAS_PUB_KEY`
4. **Command line flags** - `--api-key`, `--project-id`, etc.

## Config file

Create `~/.matlas/config.yaml` to set default values:

```yaml
output: table
timeout: 30s
projectId: "507f1f77bcf86cd799439011"
apiKey: "<private-key>"
publicKey: "<public-key>"
```

You can override the config file location using `--config` flag or `ATLAS_CONFIG_FILE` environment variable.

## Environment variables

Set these environment variables for authentication:

| Variable | Description | Required |
|:---------|:------------|:---------|
| `ATLAS_API_KEY` | Private API key | ✅ |
| `ATLAS_PUB_KEY` | Public API key | ✅ |
| `ATLAS_PROJECT_ID` | Default project ID | ❌ |
| `ATLAS_ORG_ID` | Default organization ID | ❌ |

```bash
export ATLAS_API_KEY="your-private-key"
export ATLAS_PUB_KEY="your-public-key"
export ATLAS_PROJECT_ID="507f1f77bcf86cd799439011"
```

## Command line flags

Override any configuration using command line flags.

### Global flags

Global flags available on all commands:

| Flag | Description |
|:-----|:------------|
| `--config` | Config file path |
| `--output, -o` | Output format (table, text, json, yaml) |
| `--timeout` | Context timeout for operations |
| `--verbose, -v` | Verbose logging |
| `--quiet, -q` | Suppress non-error output |
| `--log-format` | Log output format: text, json |
| `--api-key` | Atlas private API key (discouraged on CLI; prefer env var) |
| `--pub-key` | Atlas public API key (discouraged on CLI; prefer env var) |

Per-command flags still apply, for example `--project-id` on Atlas, discover, and infra subcommands, or `--cluster` on database commands.

## macOS Keychain integration

On macOS, matlas falls back to Keychain if credentials aren't found in flags/env/config.

Lookups performed using `security find-generic-password`:
- Service: `api-key`, Account: `matlas` → `ATLAS_API_KEY`
- Service: `pub-key`, Account: `matlas` → `ATLAS_PUB_KEY`

## Best practices

**Security recommendations:**
- Use environment variables instead of command line flags for secrets
- Limit API key scope to required permissions only
- For database enumeration, use `--use-temp-user` to create short-lived database users

### Getting API keys

1. Log into [MongoDB Atlas](https://cloud.mongodb.com)
2. Go to **Organization Access Manager** → **API Keys**
3. Create a new API key with appropriate permissions
4. Save the public and private keys securely