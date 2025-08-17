# Discovery Integration Tests

This directory contains comprehensive integration tests for the discovery feature in matlas-cli.

## Overview

The discovery feature allows users to:
- Discover existing Atlas project resources (clusters, users, network access, etc.)
- Convert discovered projects to ApplyDocument format
- Support resource-specific discovery with filtering
- Include database enumeration capabilities
- Utilize caching for improved performance

## Test Structure

### Go Integration Tests

#### `discovery_integration_test.go`
Core API and service-level integration tests:

- **TestDiscovery_BasicFlow_Integration**: Tests basic project discovery and ApplyDocument conversion
- **TestDiscovery_IncrementalFlow_Integration**: Tests incremental discovery with user addition/removal lifecycle
- **TestDiscovery_ResourceSpecific_Integration**: Tests individual resource type discovery
- **TestDiscovery_FormatConversion_Integration**: Tests format conversion validation
- **TestDiscovery_ErrorHandling_Integration**: Tests error scenarios and edge cases
- **BenchmarkDiscovery_ProjectDiscovery**: Performance benchmarking

#### `discovery_commands_integration_test.go`
CLI command-level integration tests:

- **TestDiscoveryCommand_BasicDiscovery_Integration**: Basic CLI discovery functionality
- **TestDiscoveryCommand_ConvertToApplyDocument_Integration**: CLI conversion workflows
- **TestDiscoveryCommand_ResourceSpecific_Integration**: CLI filtering and resource-specific discovery
- **TestDiscoveryCommand_Caching_Integration**: CLI caching functionality
- **TestDiscoveryCommand_ErrorHandling_Integration**: CLI error handling and validation

## Test Scenarios

### 1. Basic Discovery Flow
```
Discover Project → Convert to ApplyDocument → Apply (verify no changes)
```
- Discovers an existing Atlas project and all its resources
- Converts the discovered state to ApplyDocument format
- Applies the converted document to verify consistency

### 2. Incremental Discovery
```
Initial Discovery → Add User → Detect in Atlas → New Discovery → Remove User
```
- Gets baseline discovery state
- Adds a new user via ApplyDocument
- Verifies user is discoverable in Atlas
- Runs discovery again to capture the new user
- Removes user while retaining other resources

### 3. Resource-Specific Discovery
- Tests discovery of individual resource types (clusters, users, network)
- Tests filtering with `--include` and `--exclude` options
- Verifies resource-specific manifest conversion

### 4. Format Conversion Testing
- Tests DiscoveredProject → ApplyDocument conversion
- Validates converted document structure and consistency
- Tests applying converted documents

### 5. Advanced Features
- Caching functionality and performance
- Error handling and partial failures
- Different output formats (YAML, JSON)
- Timeout and context cancellation

## Shell-based Lifecycle Tests

The `scripts/test/discovery-lifecycle.sh` script provides end-to-end testing:

```bash
# Run all discovery tests
./scripts/test.sh discovery

# Run specific test types
./scripts/test/discovery-lifecycle.sh --basic-only
./scripts/test/discovery-lifecycle.sh --incremental-only
```

## Prerequisites

### Environment Setup
1. Atlas organization and project access
2. Atlas API credentials (public/private key pair)
3. Go 1.21+ installed

### Configuration
Create a `.env` file in the project root:
```bash
ATLAS_PUB_KEY=your-atlas-public-key
ATLAS_API_KEY=your-atlas-private-key
ATLAS_PROJECT_ID=your-test-atlas-project-id
ATLAS_ORG_ID=your-atlas-org-id
```

### Optional Configuration
```bash
ATLAS_TEST_TIMEOUT=10m
ATLAS_CLEANUP_TIMEOUT=5m
ATLAS_SKIP_CLEANUP=false
ATLAS_VERBOSE=true
TEST_CLUSTER_NAME=test-cluster-discovery
```

## Running Tests

### Go Integration Tests
```bash
# Run all discovery integration tests
go test -tags=integration ./test/integration/discovery/... -v

# Run specific test
go test -tags=integration -run TestDiscovery_BasicFlow_Integration ./test/integration/discovery/ -v

# Run with timeout
go test -tags=integration ./test/integration/discovery/... -timeout=10m -v
```

### Shell Lifecycle Tests
```bash
# Via main test runner
./scripts/test.sh discovery

# Direct execution
./scripts/test/discovery-lifecycle.sh

# With options
./scripts/test/discovery-lifecycle.sh --basic-only --verbose
./scripts/test/discovery-lifecycle.sh --incremental-only
./scripts/test/discovery-lifecycle.sh --skip-integration
```

## Test Environment

### Automatic Cleanup
- Test resources are automatically tracked and cleaned up
- Created users are removed after tests
- Created network access entries are removed
- Temporary files are cleaned up
- Set `ATLAS_SKIP_CLEANUP=true` to disable for debugging

### Resource Naming
Test resources use timestamped names to avoid conflicts:
- **Format**: `discovery-test-{type}-{timestamp}`
- **Example**: `discovery-test-user-1641234567`

### Isolation
- Tests use a dedicated Atlas project
- Resources are created fresh for each test run
- Cleanup ensures no resource leaks between runs

## Test Reports

Test results and artifacts are saved to:
- `test-reports/discovery/` - Shell test reports and temporary files
- Standard Go test output for integration tests

## Troubleshooting

### Common Issues

1. **Atlas Credentials**: Ensure all required environment variables are set
2. **Project Permissions**: Verify the Atlas project has appropriate permissions
3. **Network Connectivity**: Ensure connectivity to Atlas APIs
4. **Resource Limits**: Check Atlas project quotas and limits

### Debug Mode
```bash
# Skip cleanup for debugging
./scripts/test/discovery-lifecycle.sh --skip-cleanup

# Enable verbose output
./scripts/test/discovery-lifecycle.sh --verbose

# Check environment status
./scripts/test.sh help
```

### Manual Cleanup
If automatic cleanup fails, manually remove test resources:
```bash
# List and remove test users
matlas database users list --project-id $ATLAS_PROJECT_ID | grep discovery-test

# Remove specific user
matlas database users delete discovery-test-user-123 --project-id $ATLAS_PROJECT_ID
```

## Architecture

### Test Environment Setup
The `DiscoveryTestEnvironment` and `DiscoveryCommandTestEnvironment` structs provide:
- Atlas service clients
- Test resource management
- Automatic cleanup
- Configuration management

### Resource Tracking
All test resources are tracked for cleanup:
- `CreatedUsers[]` - Database users created during tests
- `CreatedNetwork[]` - Network access entries created
- `TempFiles[]` - Temporary files for cleanup

### Error Handling
Tests are designed to:
- Continue with partial failures where appropriate
- Provide detailed error reporting
- Clean up resources even if tests fail
- Log warnings for non-critical cleanup failures

## Contributing

When adding new discovery tests:

1. **Follow Naming Conventions**: Use descriptive test names with `_Integration` suffix
2. **Resource Cleanup**: Always track and clean up created resources
3. **Error Handling**: Provide meaningful error messages and handle partial failures
4. **Documentation**: Update this README with new test scenarios
5. **Integration**: Add new tests to the shell lifecycle script if appropriate

### Test Categories

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test API interactions and service integrations
- **Lifecycle Tests**: Test complete workflows end-to-end
- **Performance Tests**: Benchmark critical operations
- **Error Tests**: Test error conditions and edge cases



