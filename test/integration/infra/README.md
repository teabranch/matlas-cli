# Integration Tests for Apply Engine

This directory contains integration tests for the apply engine that test against real Atlas APIs.

## Setup

### Prerequisites

1. Atlas organization and project access
2. Atlas API credentials (public/private key pair)
3. Go 1.21+ installed

### Configuration Options

#### Option 1: Using .env File (Recommended)

Create a `.env` file in the project root with your Atlas credentials:

```bash
# .env file
ATLAS_PUB_KEY=your-atlas-public-key
ATLAS_API_KEY=your-atlas-private-key
PROJECT_ID=your-atlas-project-id
ORG_ID=your-atlas-org-id
```

The integration test framework will automatically:
- 🔍 **Auto-discover** the `.env` file (searches current and parent directories)
- 🔄 **Auto-load** environment variables
- 🗺️ **Auto-map** variable names to the expected format

#### Option 2: Environment Variables

Set environment variables manually:

```bash
export ATLAS_PUBLIC_KEY="your-atlas-public-key"
export ATLAS_PRIVATE_KEY="your-atlas-private-key" 
export ATLAS_PROJECT_ID="your-test-atlas-project-id"
export ATLAS_ORG_ID="your-atlas-org-id"
```

### Optional Configuration

```bash
# Optional settings
export ATLAS_TEST_TIMEOUT="10m"        # Default: 5m
export ATLAS_CLEANUP_TIMEOUT="5m"      # Default: 2m
export ATLAS_SKIP_CLEANUP="false"      # Default: false
export ATLAS_VERBOSE="true"            # Default: false
```

## Running Tests

### Using the Test Runner Script (Recommended)

```bash
# Check environment status
./scripts/run-tests.sh env-status

# Run integration tests (auto-loads .env if present)
./scripts/run-tests.sh integration

# Run all tests including integration
./scripts/run-tests.sh all
```

### Direct Go Commands

```bash
# Run all integration tests
go test -tags=integration ./test/integration/apply/...

# Run specific test
go test -tags=integration -run TestAtlasExecutor_Integration_BasicWorkflow ./test/integration/apply/

# Run tests in short mode (skips integration tests)
go test -short ./test/integration/apply/...

# Run with verbose output
go test -tags=integration -v ./test/integration/apply/...
```

## Environment Status Check

Check if your environment is properly configured:

```bash
./scripts/run-tests.sh env-status
```

This will show:
- ✅ Whether `.env` file is found
- 🔑 Which Atlas credentials are available
- 📋 Current environment variable status

## Test Structure

### Test Environment (`setup_test.go`)

The `TestEnvironment` struct provides:
- **Atlas service clients** - Connection to real Atlas API
- **Test resource management** - Automatic resource tracking
- **Automatic cleanup** - Prevents resource leaks
- **Configuration management** - Environment and .env file handling

### Test Categories

1. **Basic Workflow** - Single resource creation and validation
2. **Multi-Resource Plans** - Complex plans with dependencies
3. **Error Handling** - Invalid configurations and error recovery
4. **Context Cancellation** - Timeout and cancellation behavior

## Resource Management

### Automatic Cleanup

All test resources are automatically cleaned up after test completion:
- 🗑️ Test clusters are deleted
- 👤 Test database users are removed
- 🌐 Test network access rules are deleted

### Resource Naming

Test resources use timestamped names to avoid conflicts:
- **Format**: `test-{name}-{timestamp}`
- **Example**: `test-cluster-1641234567`

### Cleanup Behavior

- 🧹 Cleanup runs in `t.Cleanup()` handlers
- 🔄 Cleanup continues even if individual resource deletion fails
- 📝 Cleanup errors are logged but don't fail the test
- 🚫 Set `ATLAS_SKIP_CLEANUP=true` to disable cleanup for debugging

## Test Isolation

### Project Isolation

- Each test should use a dedicated Atlas project
- Tests should not assume specific project state
- Resources should be created fresh for each test

### Parallel Execution

- Tests can be run in parallel within the same project
- Resource names include timestamps to avoid conflicts
- Each test manages its own resources

### Data Management

- Tests should not rely on existing data
- Tests should create all required test data
- Tests should validate their own assertions

## Configuration

### Variable Name Mapping

The integration tests automatically map `.env` variable names:

| .env Variable | Maps To | Description |
|---------------|---------|-------------|
| `ATLAS_PUB_KEY` | `ATLAS_PUBLIC_KEY` | Atlas API public key |
| `ATLAS_API_KEY` | `ATLAS_PRIVATE_KEY` | Atlas API private key |
| `PROJECT_ID` | `ATLAS_PROJECT_ID` | Atlas project ID |
| `ORG_ID` | `ATLAS_ORG_ID` | Atlas organization ID |

### Timeouts

- **Test Timeout**: Maximum time for entire test execution
- **Cleanup Timeout**: Maximum time for resource cleanup
- **Operation Timeout**: Maximum time for individual Atlas operations

### Error Handling

- Integration tests should handle Atlas API rate limits
- Tests should retry transient failures
- Tests should fail fast on authentication errors

## Best Practices

### 1. Resource Lifecycle

```go
func TestExample(t *testing.T) {
    env := SetupTestEnvironment(t)
    
    // Create test resources
    cluster, err := env.CreateTestCluster("example")
    // ... test logic
    
    // Cleanup is automatic via t.Cleanup()
}
```

### 2. Error Handling

```go
// Handle expected "not implemented" errors
if err != nil && contains(err.Error(), "not yet implemented") {
    t.Log("Test completed - operations not yet implemented (expected)")
    return
}
```

### 3. Timeouts

```go
// Use reasonable timeouts for Atlas operations
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()
```

### 4. Validation

```go
// Validate test environment before proceeding
ValidateTestEnvironment(t, env)

// Use specific assertions
if result.Status != apply.PlanStatusCompleted {
    t.Errorf("Expected status %v, got %v", apply.PlanStatusCompleted, result.Status)
}
```

## Troubleshooting

### Common Issues

1. **Authentication Errors**
   - ✅ Verify Atlas API credentials in `.env` file
   - ✅ Check organization/project access
   - ✅ Ensure credentials have required permissions

2. **Timeout Errors**
   - ⏰ Increase `ATLAS_TEST_TIMEOUT`
   - 🌐 Check Atlas API service status
   - 🔗 Verify network connectivity

3. **Resource Conflicts**
   - 📋 Enable `ATLAS_VERBOSE=true` for detailed logging
   - 🔍 Check for existing resources with same names
   - 🧹 Verify cleanup completed successfully

4. **Permission Errors**
   - 🔐 Verify Atlas project permissions
   - 🏢 Check organization-level access
   - 🔑 Ensure API key has required scopes

### Debug Mode

```bash
# Enable verbose logging and skip cleanup
export ATLAS_VERBOSE="true"
export ATLAS_SKIP_CLEANUP="true"

./scripts/run-tests.sh integration
```

### Environment Debugging

```bash
# Check what's loaded from .env
./scripts/run-tests.sh env-status

# Show current environment
env | grep ATLAS
```

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run Integration Tests
  run: |
    # Create .env file from secrets
    echo "ATLAS_PUB_KEY=${{ secrets.ATLAS_PUBLIC_KEY }}" >> .env
    echo "ATLAS_API_KEY=${{ secrets.ATLAS_PRIVATE_KEY }}" >> .env  
    echo "PROJECT_ID=${{ secrets.ATLAS_PROJECT_ID }}" >> .env
    echo "ORG_ID=${{ secrets.ATLAS_ORG_ID }}" >> .env
    
    # Run integration tests (will auto-load .env)
    ./scripts/run-tests.sh integration
```

### Test Selection

```bash
# Run only fast integration tests
go test -tags=integration -short ./test/integration/apply/...

# Run all integration tests (requires Atlas credentials)
./scripts/run-tests.sh integration

# Check environment before running
./scripts/run-tests.sh env-status && ./scripts/run-tests.sh integration
```

## Security Notes

- 🔒 **Never commit `.env` files** to version control
- 🛡️ **Use project-specific credentials** for testing
- ⚠️ **Monitor Atlas usage** to avoid unexpected charges
- 🔐 **Rotate credentials** regularly
- 🚫 **Use test/development projects** only 