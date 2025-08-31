# Feature: Comprehensive Backup Features

## Summary

Complete implementation of MongoDB Atlas backup features including continuous backup, Point-in-Time Recovery (PIT), and cross-region backup support. This feature provides both CLI and YAML interfaces with proper validation to ensure backup requirements are met according to MongoDB Atlas API constraints.

The implementation enforces the correct workflow where Point-in-Time Recovery can only be enabled after cluster creation and requires continuous backup to be enabled first.

## Implementation Details

### CLI Interface

- **Cluster Creation**: `matlas atlas clusters create` with `--backup` flag
- **Cluster Updates**: `matlas atlas clusters update` with `--backup` and `--pit` flags
- **Validation**: Prevents `--pit` during cluster creation with clear error messages
- **Workflow**: Two-step process for PIT (backup first, then PIT via update)

### YAML Interface

- **Backup Configuration**: `backupEnabled: true` field in cluster specs
- **PIT Configuration**: `pitEnabled: true` field in cluster specs (requires backupEnabled)
- **Cross-field Validation**: Ensures PIT requires backup to be enabled
- **Multi-region Support**: Cross-region backup via replicationSpecs configuration

### Validation Rules

1. **PIT Requires Backup**: `pitEnabled: true` without `backupEnabled: true` fails validation
2. **Creation Restriction**: `--pit` flag cannot be used during cluster creation
3. **Update Validation**: PIT can only be enabled if backup is already active
4. **Instance Size**: Backup features require M10+ instance sizes

## Code Changes

### Types
- `internal/types/config.go`: Added `PitEnabled *bool` to ClusterConfig
- `internal/types/apply.go`: Added `PitEnabled *bool` to ClusterSpec

### CLI Commands
- `cmd/atlas/clusters/clusters.go`: 
  - Added PIT validation during cluster creation
  - Implemented proper update workflow with backup validation
  - Added `--pit` flag with appropriate help text

### Apply Pipeline
- `cmd/infra/apply.go`: Added PIT parsing from YAML specifications
- `internal/apply/fetchers.go`: Added PIT status retrieval from Atlas clusters

### Validation
- `internal/apply/validation.go`: 
  - Added cross-field validation for ApplyConfig
  - Added cross-document validation for ApplyDocument
  - Enforces PIT backup requirements

## Testing

### Test Coverage
- `scripts/test/backup-features.sh`: Comprehensive test suite including:
  - Continuous backup CLI testing
  - Point-in-Time Recovery workflow testing
  - Backup update operations testing
  - YAML backup configuration testing
  - CLI validation testing (PIT during creation)
  - YAML validation testing (PIT without backup)

### Test Integration
- Added to `scripts/test.sh` as `backup` command
- Integrated into comprehensive test suite
- Proper cleanup and resource management

## Documentation

### Updated Files
- `docs/yaml-kinds-reference.md`: Added backup features section with validation rules
- `docs/examples/clusters.md`: Added comprehensive backup example and CLI workflows
- `docs/atlas.md`: Updated cluster commands with backup features and correct workflow
- `examples/cluster-backup-comprehensive.yaml`: New comprehensive backup example
- `CHANGELOG.md`: Added feature entries

### Example Configurations
- Basic backup-enabled cluster
- Point-in-Time Recovery cluster configuration
- Cross-region backup via multi-region setup
- Development cluster with backup disabled
- Complete backup workflow examples

## Breaking Changes

None. This is a new feature addition that enhances existing cluster management.

## Migration Notes

Existing clusters are not affected. Users can enable backup features on existing clusters using the update commands:

```bash
# Enable backup on existing cluster
matlas atlas clusters update my-cluster --backup

# Enable PIT on existing cluster (after backup is active)  
matlas atlas clusters update my-cluster --pit
```

## Related Issues/PRs

- Addresses MongoDB Atlas API constraints for Point-in-Time Recovery
- Follows repository standards for CLI + YAML feature consistency
- Implements proper validation according to Atlas backup requirements

## Examples

### CLI Workflow
```bash
# Create cluster with backup
matlas atlas clusters create my-cluster --backup --tier M10 --provider AWS --region US_EAST_1

# Enable PIT after cluster is ready
matlas atlas clusters update my-cluster --pit
```

### YAML Configuration
```yaml
apiVersion: matlas.mongodb.com/v1
kind: Cluster
metadata:
  name: backup-cluster
spec:
  projectName: "My Project"
  provider: AWS
  region: US_EAST_1
  instanceSize: M10
  backupEnabled: true
  pitEnabled: true      # Requires backupEnabled: true
```

This feature provides complete backup functionality while ensuring users follow the correct MongoDB Atlas workflow for Point-in-Time Recovery configuration.
