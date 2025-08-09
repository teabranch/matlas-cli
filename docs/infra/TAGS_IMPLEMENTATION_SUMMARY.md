# Atlas Resource Tags Implementation Summary

This document summarizes the implementation of Atlas native resource tags support in matlas-cli.

## What Was Implemented

### 1. Configuration Types (`internal/types/config.go`)
- Added `Tags map[string]string` field to `ClusterConfig` struct
- Added `Tags map[string]string` field to `ProjectConfig` struct
- Both include validation rules for Atlas tag requirements (50 tags max, 255 char limits)

### 2. CLI Support (`cmd/atlas/clusters/clusters.go`)
- Added `--tag` flag to cluster create command
- Format: `--tag key=value` (repeatable)
- Added helper functions:
  - `parseTagsFromStrings()`: Converts CLI strings to map[string]string
  - `convertTagsToAtlasFormat()`: Converts to Atlas SDK format `[]admin.ResourceTag`
- Updated `buildClusterConfiguration()` to include tags in Atlas API calls
- Added tag examples to command help text

### 3. YAML Configuration Support
- Updated cluster creation to merge YAML tags with CLI tags (CLI takes precedence)
- Added tags to example YAML files:
  - `examples/clusters/api-specification-cluster.yaml`
  - `examples/infra/templates/project-template.yaml`
- Updated documentation in `docs/infra/configuration-schema.md`

### 4. Validation (`internal/validation/validation.go`)
- Added `ValidateAtlasResourceTags()` function
- Validates according to Atlas requirements:
  - Maximum 50 tags per resource
  - Tag keys/values: 1-255 characters
  - Allowed characters: letters, numbers, spaces, `;@_-.+`
  - Case-sensitive keys and values
- Integrated validation into cluster creation flow

### 5. Atlas SDK Integration
- Confirmed Atlas Go SDK v20250312005 supports tags via:
  - `admin.ClusterDescription20240805.Tags *[]admin.ResourceTag`
  - `admin.Group.Tags *[]admin.ResourceTag` (for projects)
- Tags are passed to Atlas API during cluster creation/updates

### 6. Documentation and Examples
- Added comprehensive tag documentation to configuration schema
- Explained difference between:
  - Labels/Annotations: Internal matlas-cli metadata (Kubernetes-style)
  - Tags: Native Atlas feature for billing, monitoring, organization
- Added example tag patterns for common use cases

### 7. Testing
- Created unit tests for tag parsing and conversion functions
- Tested validation with various invalid inputs
- Verified CLI flag appears in help output
- Confirmed build and compilation success

## Usage Examples

### CLI Usage
```bash
# Create cluster with tags
matlas atlas clusters create \
  --name tagged-cluster \
  --project-id 507f1f77bcf86cd799439011 \
  --tier M30 \
  --tag environment=production \
  --tag team=backend \
  --tag cost-center=engineering
```

### YAML Configuration
```yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
spec:
  name: "Production Project"
  organizationId: "507f1f77bcf86cd799439011"
  
  # Project-level Atlas Resource Tags
  tags:
    project-type: "microservices"
    billing-department: "engineering"
    compliance-level: "high"
    
  clusters:
    - metadata:
        name: production-cluster
        labels:
          environment: production  # matlas-cli labels
      
      # Cluster-level Atlas Resource Tags
      tags:
        environment: "production"
        application: "user-service"
        team: "backend"
        tier: "critical"
        backup-required: "true"
        
      provider: AWS
      region: US_EAST_1
      instanceSize: M30
```

## Benefits

1. **Cost Allocation**: Tags appear in Atlas billing for departmental cost tracking
2. **Resource Organization**: Tags visible in Atlas UI for better resource management
3. **Monitoring Integration**: Tags sent to DataDog, Prometheus for automated monitoring
4. **Compliance**: Tag resources for security and compliance requirements
5. **Automation**: Use tags for automated policies and workflows

## Atlas Features Supported

- ✅ Tags appear in Atlas UI
- ✅ Tags included in billing invoices and reports  
- ✅ Tags sent to monitoring integrations (DataDog, Prometheus)
- ✅ Tags support cost allocation and chargeback
- ✅ Tags available via Atlas Administration API
- ✅ All Atlas tag validation rules enforced

## Future Enhancements

- Add tag support to other CLI commands (update, project operations)
- Add tag support for other Atlas resources (database users, network access)
- Add tag-based filtering and search capabilities
- Add tag import/export functionality

## Files Modified

### Core Implementation
- `internal/types/config.go` - Added Tags fields
- `cmd/atlas/clusters/clusters.go` - CLI support and conversion
- `internal/validation/validation.go` - Tag validation

### Documentation & Examples  
- `examples/clusters/api-specification-cluster.yaml`
- `examples/infra/templates/project-template.yaml`
- `docs/infra/configuration-schema.md`

### Tests
- `cmd/atlas/clusters/clusters_tags_test.go` - Unit tests

All implementation follows Atlas native tagging standards and integrates seamlessly with existing matlas-cli architecture.