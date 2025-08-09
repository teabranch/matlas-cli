# Troubleshooting Guide

This guide covers common issues and solutions when using `matlas infra` commands, with particular focus on destroy operations and resource management.

## Table of Contents

- [Destroy Operation Issues](#destroy-operation-issues)
- [Common Error Messages](#common-error-messages)
- [Resource Management Issues](#resource-management-issues)
- [Performance Issues](#performance-issues)
- [Configuration Issues](#configuration-issues)
- [Recovery Procedures](#recovery-procedures)

## Destroy Operation Issues

### Race Conditions During Destruction

**Symptoms:**
- `HTTP 404 Not Found` errors during destroy operations
- Network access entries showing "not found" errors
- Resources appearing to be deleted before their dependencies

**Root Cause:**
Resources being deleted in the wrong order, causing dependencies to be removed before dependent resources.

**Solution:**
The CLI now automatically handles dependency ordering, but if you still encounter issues:

1. **Use dry-run to verify the deletion plan:**
   ```bash
   matlas infra destroy -f config.yaml --dry-run
   ```

2. **Check the deletion order in the output** - should be:
   - Database Users (first)
   - Network Access Lists (second) 
   - Clusters (last)

3. **If race conditions persist, use targeted deletion:**
   ```bash
   # Manual step-by-step deletion
   matlas infra destroy -f config.yaml --target users
   matlas infra destroy -f config.yaml --target network-access  
   matlas infra destroy -f config.yaml --target clusters
   ```

### Database Users Not Being Deleted

**Symptoms:**
- Users exist in Atlas console but destroy operation doesn't remove them
- "User not found in configuration" type messages

**Root Cause:**
The default destroy mode only destroys resources that exist in BOTH your configuration files AND Atlas. Users not defined in your YAML won't be destroyed.

**Solutions:**

1. **Add missing users to your configuration** (recommended for managed infrastructure):
   ```yaml
   databaseUsers:
     - metadata:
         name: orphaned-user
       username: "orphaned-user"
       databaseName: "admin"
       # ... complete user configuration
   ```

2. **Use discovery-only mode** for comprehensive cleanup:
   ```bash
   # Preview what would be destroyed
   matlas infra destroy --discovery-only --project-id PROJECT_ID --dry-run
   
   # Destroy all discovered resources
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

### Partial Destruction Failures

**Symptoms:**
- Some resources delete successfully while others fail
- Mixed success/failure messages in output

**Troubleshooting Steps:**

1. **Identify failed resources:**
   ```bash
   # Check current project state
   matlas infra show --project-id PROJECT_ID
   ```

2. **Check for resource dependencies:**
   - Look for resources that depend on failed ones
   - Verify network connectivity to Atlas

3. **Use incremental deletion:**
   ```bash
   # Delete by resource type with explicit ordering
   matlas infra destroy -f config.yaml --target users --dry-run
   matlas infra destroy -f config.yaml --target users
   
   matlas infra destroy -f config.yaml --target network-access --dry-run
   matlas infra destroy -f config.yaml --target network-access
   
   matlas infra destroy -f config.yaml --target clusters --dry-run
   matlas infra destroy -f config.yaml --target clusters
   ```

4. **Force deletion if safe:**
   ```bash
   # Use with extreme caution
   matlas infra destroy -f config.yaml --force
   ```

## Common Error Messages

### `HTTP 404 Not Found - ATLAS_NETWORK_PERMISSION_ENTRY_NOT_FOUND`

**Message:**
```
failed to delete network access entry: https://cloud.mongodb.com/api/atlas/v2/groups/PROJECT_ID/accessList/192.168.0.0%2F16 DELETE: HTTP 404 Not Found (Error code: "ATLAS_NETWORK_PERMISSION_ENTRY_NOT_FOUND") Detail: IP Address 192.168.0.0/16 not on Atlas access list
```

**Explanation:**
The network access entry was already deleted (often automatically when a cluster was deleted). This is now handled gracefully and treated as success.

**Action Required:**
None - this is handled automatically in recent versions.

### `project ID not available for X deletion`

**Message:**
```
project ID not available for network access deletion
```

**Cause:**
The project ID isn't properly resolved during the destroy operation.

**Solutions:**

1. **Explicitly provide project ID:**
   ```bash
   matlas infra destroy -f config.yaml --project-id YOUR_PROJECT_ID
   ```

2. **Ensure project ID is in your configuration:**
   ```yaml
   spec:
     organizationId: "YOUR_ORG_ID"
     # Project ID should be discoverable from the configuration
   ```

3. **Use discovery-only mode:**
   ```bash
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

### `at least one configuration file must be specified`

**Message:**
```
at least one configuration file must be specified with --file (or use --discovery-only)
```

**Cause:**
No configuration files provided and not using discovery-only mode.

**Solutions:**

1. **Provide configuration files:**
   ```bash
   matlas infra destroy -f config.yaml
   ```

2. **Use discovery-only mode:**
   ```bash
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

### `circular dependency detected in operations`

**Message:**
```
circular dependency detected in operations
```

**Cause:**
Resource dependencies form a cycle, preventing proper ordering.

**Solutions:**

1. **Review resource dependencies in your configuration**
2. **Remove circular references**
3. **Use manual step-by-step destruction if needed**

## Resource Management Issues

### Orphaned Resources

**Problem:**
Resources exist in Atlas but aren't managed by your configuration files.

**Detection:**
```bash
# Show all resources in project
matlas infra show --project-id PROJECT_ID

# Compare with your configuration
matlas infra diff -f config.yaml
```

**Solutions:**

1. **Add orphaned resources to configuration** (recommended):
   ```yaml
   # Add the discovered resources to your YAML files
   ```

2. **Clean up with discovery-only mode:**
   ```bash
   matlas infra destroy --discovery-only --project-id PROJECT_ID --dry-run
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

### Resource Name Conflicts

**Problem:**
Attempting to create resources with names that already exist.

**Solutions:**

1. **Check existing resources:**
   ```bash
   matlas infra show --project-id PROJECT_ID
   ```

2. **Use unique naming patterns:**
   ```yaml
   metadata:
     name: "${ENVIRONMENT}-${PURPOSE}-${TIMESTAMP}"
   ```

3. **Clean up conflicting resources first**

### Access Permission Issues

**Problem:**
API permissions insufficient for resource operations.

**Symptoms:**
- `HTTP 401 Unauthorized` errors
- `HTTP 403 Forbidden` errors

**Solutions:**

1. **Verify API key permissions**
2. **Check organization and project access**
3. **Ensure API keys are for the correct environment**

## Performance Issues

### Slow Destroy Operations

**Problem:**
Destroy operations taking longer than expected.

**Troubleshooting:**

1. **Check Atlas console** for resource status
2. **Monitor network connectivity**
3. **Use smaller batch sizes:**
   ```bash
   # Process resources incrementally
   matlas infra destroy -f config.yaml --target users
   # Wait for completion before proceeding
   matlas infra destroy -f config.yaml --target clusters
   ```

### Timeout Issues

**Problem:**
Operations timing out before completion.

**Solutions:**

1. **Increase timeout:**
   ```bash
   matlas infra destroy -f config.yaml --timeout 60m
   ```

2. **Break down large operations:**
   ```bash
   # Process in smaller chunks
   matlas infra destroy -f config.yaml --target users
   matlas infra destroy -f config.yaml --target network-access
   matlas infra destroy -f config.yaml --target clusters
   ```

## Configuration Issues

### Environment Variable Resolution

**Problem:**
Environment variables not resolving in configuration files.

**Debugging:**

1. **Check variable definitions:**
   ```bash
   echo $VARIABLE_NAME
   ```

2. **Use strict mode for debugging:**
   ```bash
   matlas infra destroy -f config.yaml --strict-env
   ```

3. **Verify variable syntax in YAML:**
   ```yaml
   # Correct syntax
   name: "${VARIABLE_NAME}"
   
   # With defaults
   name: "${VARIABLE_NAME:-default-value}"
   
   # Required variables
   name: "${VARIABLE_NAME:?Variable is required}"
   ```

### YAML Syntax Issues

**Problem:**
Invalid YAML syntax causing parsing errors.

**Solutions:**

1. **Validate configuration:**
   ```bash
   matlas infra validate -f config.yaml
   ```

2. **Use YAML linting tools:**
   ```bash
   yamllint config.yaml
   ```

3. **Check indentation and quotes**

## Recovery Procedures

### Stuck Destroy Operations

**Symptoms:**
- Operations appear to hang
- Resources in inconsistent state

**Recovery Steps:**

1. **Check Atlas console** for actual resource status
2. **Cancel operation** if safe to do so (Ctrl+C)
3. **Use discovery-only mode** to clean up:
   ```bash
   matlas infra destroy --discovery-only --project-id PROJECT_ID --dry-run
   matlas infra destroy --discovery-only --project-id PROJECT_ID
   ```

### Inconsistent State Recovery

**Problem:**
Atlas resources don't match expected configuration state.

**Recovery:**

1. **Capture current state:**
   ```bash
   matlas infra show --project-id PROJECT_ID --output yaml > current-state.yaml
   ```

2. **Compare with desired state:**
   ```bash
   matlas infra diff -f config.yaml
   ```

3. **Choose recovery approach:**
   - **Bring Atlas to match config:** `matlas infra -f config.yaml`
   - **Update config to match Atlas:** Edit configuration files
   - **Clean slate:** `matlas infra destroy --discovery-only --project-id PROJECT_ID`

### Emergency Resource Cleanup

**When to use:**
- Unrecoverable errors
- Need to clean up test/development environments quickly
- Stuck resources blocking other operations

**Procedure:**

1. **Backup current state:**
   ```bash
   mkdir -p backups/emergency-$(date +%s)
   matlas infra show --project-id PROJECT_ID --output yaml > backups/emergency-$(date +%s)/state.yaml
   ```

2. **Force cleanup:**
   ```bash
   matlas infra destroy --discovery-only --project-id PROJECT_ID --force
   ```

3. **Verify cleanup:**
   ```bash
   matlas infra show --project-id PROJECT_ID
   ```

## Getting Help

If you encounter issues not covered in this guide:

1. **Check the main documentation:** [README.md](../../README.md)
2. **Review best practices:** [best-practices.md](best-practices.md)
3. **Check Atlas console** for additional error details
4. **Enable verbose logging:**
   ```bash
   matlas infra destroy -f config.yaml --verbose
   ```

5. **Capture detailed logs:**
   ```bash
   matlas infra destroy -f config.yaml --verbose 2>&1 | tee destroy.log
   ```