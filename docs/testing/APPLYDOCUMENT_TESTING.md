# ApplyDocument Format Testing

## Overview

This document describes the comprehensive testing approach for the `ApplyDocument` YAML format in matlas-cli. Previously, this format was under-tested compared to the `Project` format, leading to runtime errors that weren't caught by validation.

## Background

### The Testing Gap

Prior to this enhancement, the testing suite had an imbalance:

- **Project Format**: Well-tested across multiple scenarios (e2e, integration, run-e2e-tests)
- **ApplyDocument Format**: Only tested in cluster lifecycle scenarios

### Key Differences

#### Project Format
```yaml
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-project
spec:
  name: "Test Project"
  organizationId: 5ca37ef6a6f239b2387738cd
  databaseUsers:           # ← Embedded in Project spec
    - metadata:
        name: test-user
      username: test-user
      roles:
        - roleName: readWrite
          databaseName: testapp
```

#### ApplyDocument Format
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: yaml-cluster-test
resources:                # ← Individual resources array
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser     # ← Standalone resource
    metadata:
      name: test-user
    spec:
      projectName: "Test Project"
      username: test-user
      roles:
        - roleName: readWrite
          databaseName: testapp
```

## Errors That Led to This Testing Enhancement

### 1. Invalid regionName Error
```
non-retryable error: failed to create cluster: Invalid attribute regionName specified
```
**Root Cause**: Using `US_EAST_1` instead of `US_EAST_1`

### 2. Missing Database User Roles Error
```
non-retryable error: database user yaml-test-user-1754498797 must have at least one role defined
```
**Root Cause**: Database user validation in ApplyDocument format wasn't properly tested

## New Testing Structure

### 1. Comprehensive ApplyDocument Tests (`applydocument-test.sh`)

**Test Scenarios:**
- **Validation Tests**: Basic ApplyDocument validation, missing roles, missing required fields
- **Mixed Resources**: Cluster + DatabaseUser + NetworkAccess in single ApplyDocument
- **Standalone Users**: Multiple DatabaseUser resources without clusters
- **Format Comparison**: ApplyDocument vs Project format for same resources
- **Error Handling**: Invalid regions, invalid roles, mixed valid/invalid resources

**Usage:**
```bash
# Run all ApplyDocument tests
./scripts/test/applydocument-test.sh

# Run specific test types
./scripts/test/applydocument-test.sh validation
./scripts/test/applydocument-test.sh mixed
./scripts/test/applydocument-test.sh errors
```

### 2. Regression Tests (`applydocument-regression.sh`)

**Specific Test Cases:**
- **Region Format Validation**: Tests `US_EAST_1` vs `US_EAST_1`
- **Role Validation**: Tests missing roles, empty roles, proper roles
- **Combined Resources**: Tests the exact scenario that failed in cluster-lifecycle.sh

**Usage:**
```bash
./scripts/test/applydocument-regression.sh
```

### 3. Integration with Main Test Runner

The main test script (`scripts/test.sh`) now includes:

```bash
# New commands
./scripts/test.sh applydoc        # Run ApplyDocument format tests
./scripts/test.sh comprehensive   # Run all tests including ApplyDocument and cluster tests

# Existing commands (unchanged)
./scripts/test.sh all            # Run basic tests (unit + integration + e2e)
./scripts/test.sh unit           # Run unit tests
./scripts/test.sh e2e            # Run e2e tests
```

## Test Coverage Matrix

| Scenario | Project Format | ApplyDocument Format |
|----------|----------------|---------------------|
| Basic validation | ✅ (e2e.sh) | ✅ (applydocument-test.sh) |
| Database users only | ✅ (integration.sh) | ✅ (applydocument-test.sh) |
| Mixed resources | ✅ (run-e2e-tests.sh) | ✅ (applydocument-test.sh) |
| Error handling | ✅ (run-e2e-tests.sh) | ✅ (applydocument-test.sh) |
| Cluster + users | ❌ (limited) | ✅ (cluster-lifecycle.sh, applydocument-test.sh) |
| Regression cases | ❌ | ✅ (applydocument-regression.sh) |

## Running Tests

### Development Workflow

1. **Quick validation** (no real resources created):
   ```bash
   ./scripts/test.sh applydoc validation
   ```

2. **Comprehensive format testing** (no real resources):
   ```bash
   ./scripts/test.sh applydoc
   ```

3. **Full regression testing** (validates specific fixes):
   ```bash
   ./scripts/test/applydocument-regression.sh
   ```

4. **Complete test suite** (includes real cluster tests):
   ```bash
   ./scripts/test.sh comprehensive
   ```

### CI/CD Integration

The tests are designed to be safe for CI/CD:
- Most tests only validate and plan (no actual resource creation)
- Real resource creation is clearly marked and optional
- Proper cleanup mechanisms in place

## Environment Requirements

All ApplyDocument tests require:
```bash
export ATLAS_PUB_KEY="your-public-key"
export ATLAS_API_KEY="your-private-key"
export ATLAS_PROJECT_ID="your-project-id"
export ATLAS_ORG_ID="your-org-id"  # For some tests
```

## Implementation Details

### Processing Differences

The codebase handles these formats differently:

1. **Project Format** → `types.ApplyConfig` → embedded `DatabaseUserConfig`
2. **ApplyDocument Format** → `types.ApplyDocument` → individual `DatabaseUserManifest`

### Validation Points

- **Loader Level**: `internal/apply/loader.go` (lines 134-161)
- **Validation Level**: `internal/apply/validation.go` (lines 351-354)
- **Executor Level**: `internal/apply/executor.go` (lines 624-626)

## Benefits

1. **Comprehensive Coverage**: ApplyDocument format now has equal test coverage to Project format
2. **Regression Prevention**: Specific test cases prevent the errors that occurred in cluster-lifecycle.sh
3. **Format Parity**: Both YAML formats are equally validated and tested
4. **CI/CD Safe**: Most tests don't create real resources, making them suitable for continuous integration

## Future Enhancements

1. **Performance Testing**: Compare performance between Project and ApplyDocument formats
2. **Complex Scenarios**: Test large ApplyDocuments with many resources
3. **Partial Updates**: Test updating subset of resources in ApplyDocument
4. **Cross-Format Migration**: Test converting between Project and ApplyDocument formats

## Files Added/Modified

### New Files
- `scripts/test/applydocument-test.sh` - Comprehensive ApplyDocument testing
- `scripts/test/applydocument-regression.sh` - Specific regression tests
- `docs/testing/APPLYDOCUMENT_TESTING.md` - This documentation

### Modified Files
- `scripts/test.sh` - Added `applydoc` and `comprehensive` commands
- `scripts/test/cluster-lifecycle.sh` - Fixed region format (US_EAST_1 → US_EAST_1)

This testing enhancement ensures that both YAML formats are equally robust and well-tested, preventing the runtime errors that previously occurred with ApplyDocument format.