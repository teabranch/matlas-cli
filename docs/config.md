---
layout: page
title: Configuration Command
description: Manage matlas CLI configuration files and templates.
permalink: /config/
---

# Configuration Command

Manage matlas-cli configuration files and settings.

---

## Validate

```bash
matlas config validate [config-file] [--schema file.json] [--verbose]
```

Validates CLI config at `~/.matlas/config.yaml` by default.

## Templates

List and generate config templates.

```bash
matlas config template list
matlas config template generate <basic|atlas|database|apply|complete> [-o file] [-f yaml|json]
```

Notes:
- The `apply` template generates an Atlas resource configuration document (not CLI config).

## Experimental

These commands are hidden by default and may change:
- `matlas config import`
- `matlas config export`
- `matlas config migrate`


