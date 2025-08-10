---
title: Authentication and configuration
nav_order: 2
---

# Authentication and configuration

Sources and precedence
1) Defaults (output=table, timeout=30s)
2) Config file: `~/.matlas/config.yaml` or `--config`/`ATLAS_CONFIG_FILE`
3) Env vars (prefix `ATLAS_`), e.g. `ATLAS_API_KEY`, `ATLAS_PUB_KEY`, `ATLAS_PROJECT_ID`, `ATLAS_ORG_ID`
4) Flags on the command line

## Config file example (`~/.matlas/config.yaml`)
```
output: table
timeout: 30s
projectId: "507f1f77bcf86cd799439011"
apiKey: "<private>"
publicKey: "<public>"
```

## Environment
- `ATLAS_API_KEY`: private key
- `ATLAS_PUB_KEY`: public key
- Optional: `ATLAS_PROJECT_ID`, `ATLAS_ORG_ID`

## Flags
- `--api-key`, `--pub-key`, `--project-id`, `--org-id`, `--output`, `--timeout`, `--config`

## macOS Keychain fallback
- If keys aren’t found in flags/env/file, the CLI attempts a macOS keychain lookup:
  - service "api-key" account "matlas" for `ATLAS_API_KEY`
  - service "pub-key" account "matlas" for `ATLAS_PUB_KEY`

## Best practices
- Prefer environment variables over flags for secrets
- Limit scope of API keys to required permissions
- For DB enumeration, use `--use-temp-user` to create a short‑lived user

