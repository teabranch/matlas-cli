# Infrastructure Tests for matlas-cli Apply Engine

This directory contains comprehensive infrastructure tests that validate the apply engine's behavior under real-world operational conditions. These tests exercise the system against live Atlas APIs to ensure performance, reliability, and operational readiness.

## ⚠️ Important Notice

**Infrastructure tests create real Atlas resources and may incur costs.** Always run these tests against a dedicated test project and monitor your Atlas usage.

## 🎯 Test Categories

### 📊 Performance Tests (`performance/`)

**Purpose**: Validate system performance under various load conditions

**Test Coverage**:
- **Large Scale Configuration**: Tests with 5-100 resources
- **Concurrent Operations**: Tests with 1-20 concurrent executors  
- **Resource Lifecycle Performance**: Create, update, delete cycles
- **Memory Usage**: Memory consumption tracking under load
- **Throughput Measurement**: Operations per second analysis

**Key Metrics**:
- Execution time vs resource count
- Memory usage scaling
- Concurrent operation throughput
- Success rates under load

### 🛡️ Reliability Tests (`reliability/`)

**Purpose**: Validate system resilience under adverse conditions

**Test Coverage**:
- **Network Interruptions**: Behavior with latency and timeouts
- **Rate Limit Handling**: Atlas API rate limit responses
- **Partial Failure Recovery**: Mixed success/failure scenarios
- **Idempotency Testing**: Repeated operation safety
- **Error Classification**: Proper error handling and recovery

**Key Scenarios**:
- Network delays (100ms - 5s)
- Rate limit stress testing
- Invalid configuration handling
- Retry mechanism validation

## 🚀 Running Infrastructure Tests

### Prerequisites

1. **Atlas Access**: Organization and project with appropriate permissions
2. **Test Project**: Dedicated Atlas project for testing (recommended)
3. **Credentials**: Atlas API key pair with project access
4. **Go 1.21+**: Required for test execution

### Quick Start

```bash
# Check environment setup
./scripts/run-tests.sh env-status

# Run all infrastructure tests (requires confirmation)
./scripts/run-tests.sh infrastructure

# Run specific test categories
./scripts/run-tests.sh performance
./scripts/run-tests.sh reliability
```

### Configuration

#### Using .env File (Recommended)

```bash
# .env file in project root
ATLAS_PUB_KEY=your-atlas-public-key
ATLAS_API_KEY=your-atlas-private-key
PROJECT_ID=your-test-project-id
ORG_ID=your-atlas-org-id
```

#### Using Environment Variables

```bash
export ATLAS_PUBLIC_KEY="your-atlas-public-key"
export ATLAS_PRIVATE_KEY="your-atlas-private-key"
export ATLAS_PROJECT_ID="your-test-project-id"
export ATLAS_ORG_ID="your-atlas-org-id"
```

### Direct Go Commands

```bash
# All infrastructure tests
go test -tags=infrastructure -v -timeout=60m ./test/infrastructure/...

# Performance tests only
go test -tags=infrastructure -v -timeout=30m ./test/infrastructure/performance/

# Reliability tests only  
go test -tags=infrastructure -v -timeout=30m ./test/infrastructure/reliability/

# Specific test
go test -tags=infrastructure -run TestLargeScaleConfiguration ./test/infrastructure/performance/
```

## 📋 Test Execution Details

### Performance Test Matrix

| Test Scenario | Resources | Concurrency | Expected Duration | Memory Limit |
|---------------|-----------|-------------|-------------------|--------------|
| Small Scale | 5 users, 3 network rules | 5 concurrent | < 2 minutes | < 2.5 MB |
| Medium Scale | 25 users, 15 network rules | 5 concurrent | < 5 minutes | < 12.5 MB |
| Large Scale | 100 users, 50 network rules | 5 concurrent | < 15 minutes | < 50 MB |

### Concurrency Test Matrix

| Concurrency Level | Workers | Expected Throughput | Max Error Rate |
|-------------------|---------|-------------------|----------------|
| Single | 1 | Baseline | < 5% |
| Low | 5 | > 0.5 ops/sec | < 10% |
| Medium | 10 | > 0.5 ops/sec | < 15% |
| High | 20 | > 0.5 ops/sec | < 20% |

### Reliability Test Scenarios

| Test Type | Scenario | Expected Behavior |
|-----------|----------|-------------------|
| Network Latency | 100ms - 5s delays | Graceful handling, appropriate timeouts |
| Rate Limits | High-frequency operations | Backoff and retry logic |
| Partial Failures | Mixed valid/invalid resources | Partial completion tracking |
| Idempotency | Repeated plan execution | Safe re-execution |

## 📊 Test Output and Analysis

### Performance Metrics

```
Scale Test Results for Large Scale:
  Resources: 100 users, 50 network rules
  Execution Time: 12m34s
  Memory Increase: 45.2 MB
  Operations/Second: 2.1
  Success Rate: 95.6%
```

### Reliability Metrics

```
Network Test Results for High Latency:
  Execution Time: 8m12s
  Expected Retries: 2, Actual Retries: 3
  Success Rate: 88.9%
  network failures: 0
  timeout failures: 2
  rate_limit failures: 1
```

### Concurrency Metrics

```
Concurrency Test Results (Level 10):
  Total Time: 5m23s
  Throughput: 1.85 operations/second
  Error Rate: 12.5%
Performance Summary:
  Operations: 50
  Avg Response: 2.1s
  Max Response: 8.3s
  Memory Usage: 28.4 MB
```

## 🔧 Test Environment Management

### Resource Management

- **Automatic Cleanup**: All test resources are automatically deleted
- **Unique Naming**: Timestamped resource names prevent conflicts
- **Isolation**: Each test creates independent resources
- **Tracking**: Comprehensive resource lifecycle tracking

### Error Handling

**Acceptable Errors** (tests continue):
- `DUPLICATE_DATABASE_USER` - Expected for idempotency tests
- `rate limit` - Expected under load testing
- `not yet implemented` - Expected during development

**Failure Conditions** (tests fail):
- Authentication errors
- Project access issues
- Unexpected API failures
- Resource creation timeouts

### Cleanup Strategy

```go
// Automatic cleanup registration
t.Cleanup(func() {
    env.cleanup(t)
})

// Resource tracking
env.RegisterResource(TestResource{
    Type: "databaseUser",
    ID:   userName,
    Name: userName,
})
```

## 🎛️ Configuration Options

### Executor Configuration

```go
// Performance-focused configuration
config := apply.DefaultExecutorConfig()
config.MaxConcurrentOperations = 5
config.OperationTimeout = 5 * time.Minute

// Reliability-focused configuration  
config := apply.DefaultExecutorConfig()
config.MaxConcurrentOperations = 2
config.OperationTimeout = 10 * time.Minute
config.RetryConfig.MaxRetries = 10
config.RetryConfig.BaseDelay = 2 * time.Second
```

### Test Timeouts

- **Performance Tests**: 30 minutes per test category
- **Reliability Tests**: 30 minutes per test category
- **Full Infrastructure**: 60 minutes total
- **Individual Operations**: 5-10 minutes per operation

## 🚨 Troubleshooting

### Common Issues

1. **Authentication Failures**
   ```
   Atlas credentials not provided - skipping infrastructure test
   ```
   **Solution**: Set up `.env` file or environment variables

2. **Rate Limit Errors**
   ```
   rate limit exceeded: too many requests
   ```
   **Solution**: Expected behavior, tests should handle gracefully

3. **Timeout Errors**
   ```
   context deadline exceeded
   ```
   **Solution**: Increase test timeout or check network connectivity

4. **Resource Cleanup Failures**
   ```
   Warning: Failed to cleanup resource user-123: NOT_FOUND
   ```
   **Solution**: Usually harmless, resource may have been deleted already

### Debug Mode

```bash
# Enable verbose logging
export ATLAS_VERBOSE="true"

# Skip cleanup for investigation
export ATLAS_SKIP_CLEANUP="true"

# Run with detailed output
go test -tags=infrastructure -v -run TestSpecificTest ./test/infrastructure/performance/
```

### Resource Monitoring

Monitor your Atlas project during infrastructure tests:
- Database user count
- Network access rules
- API usage metrics
- Billing/usage alerts

## 🔒 Security Considerations

### Credentials Management
- **Never commit** `.env` files to version control
- **Use test-specific** API keys with limited permissions
- **Rotate credentials** regularly
- **Monitor API usage** for unexpected activity

### Project Isolation
- **Dedicated test project** recommended
- **Separate from production** environments
- **Limited resource quotas** to prevent runaway costs
- **Regular cleanup** of test artifacts

### Cost Management
- **Monitor billing** during infrastructure tests
- **Set spending alerts** on test projects
- **Use free tier resources** when possible (M0 clusters)
- **Clean up promptly** after test completion

## 📈 Performance Baselines

### Expected Performance (as of current implementation)

| Resource Type | Create Time | Update Time | Delete Time |
|---------------|-------------|-------------|-------------|
| Database User | 1-3 seconds | 2-4 seconds | 1-2 seconds |
| Network Access | 2-5 seconds | 3-6 seconds | 1-3 seconds |
| Cluster (M0) | 2-5 minutes | 5-15 minutes | 1-3 minutes |

### Scaling Characteristics

- **Linear scaling** up to 50 concurrent operations
- **Memory usage** ~0.5MB per resource in memory
- **Network overhead** ~100-500ms per Atlas API call
- **Rate limits** typically ~100 requests per minute per project

## 🔮 Future Enhancements

### Planned Test Additions

1. **Multi-Region Testing**: Cross-region performance validation
2. **Large Cluster Testing**: M10+ cluster lifecycle tests  
3. **Complex Dependencies**: Multi-resource dependency chains
4. **Disaster Recovery**: Failure injection and recovery testing
5. **Security Testing**: Authentication and authorization edge cases

### Monitoring Integration

1. **Metrics Collection**: Prometheus/Grafana integration
2. **Alert Testing**: Monitoring system validation
3. **Performance Regression**: Automated performance tracking
4. **Cost Analysis**: Automated cost impact assessment

---

## 📞 Support

For issues with infrastructure tests:

1. **Check Prerequisites**: Verify Atlas access and credentials
2. **Review Logs**: Enable verbose logging for detailed output
3. **Check Environment**: Run `./scripts/run-tests.sh env-status`
4. **Isolate Tests**: Run individual test categories to isolate issues
5. **Monitor Resources**: Check Atlas project for resource states

**Remember**: Infrastructure tests are designed to validate real-world operational scenarios. Some variance in timing and occasional transient failures are expected and handled gracefully by the test framework. 