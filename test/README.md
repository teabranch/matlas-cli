# matlas-cli Testing Infrastructure

This directory contains comprehensive testing infrastructure for matlas-cli, including unit tests, integration tests, infrastructure tests, and test automation tools.

## 🚀 Quick Start

```bash
# Run all tests
./scripts/test-comprehensive.sh

# Run only unit tests
./scripts/test-comprehensive.sh unit

# Run integration tests with verbose output
./scripts/test-comprehensive.sh --verbose integration

# Run with custom coverage threshold
./scripts/test-comprehensive.sh --coverage-threshold 90 all
```

## 📁 Directory Structure

```
test/
├── README.md                    # This file
├── integration/                 # Integration tests
│   ├── atlas/                  # Atlas API integration tests
│   │   ├── projects_integration_test.go
│   │   └── clusters_integration_test.go
│   ├── database/               # Database integration tests
│   │   └── database_integration_test.go
│   └── apply/                  # Apply engine integration tests
│       ├── executor_integration_test.go
│       └── setup_test.go
├── infrastructure/             # Infrastructure tests (existing)
│   ├── performance/
│   │   └── scale_test.go
│   └── reliability/
│       └── resilience_test.go
└── unit/                       # Additional unit test helpers
    └── apply/
        └── executor_unit_test.go
```

## 🧪 Test Categories

### 1. Unit Tests (`internal/`)
- **Location**: `internal/`
- **Purpose**: Test individual components in isolation
- **Coverage**: Services, validation, configuration, apply engine
- **Dependencies**: Mocked/stubbed
- **Runtime**: Fast (< 1 minute)

```bash
# Run unit tests only
./scripts/test-comprehensive.sh unit

# Run with coverage
go test -cover ./internal/...
```

### 2. Integration Tests (`test/integration/`)
- **Location**: `test/integration/`
- **Purpose**: Test component interactions with real services
- **Coverage**: Atlas API, Database operations, Apply workflows
- **Dependencies**: Live Atlas/MongoDB instances
- **Runtime**: Medium (2-10 minutes)

#### Atlas Integration Tests
Tests Atlas commands against live Atlas API:
- Project management (list, get)
- Cluster operations (list, get)
- User management (list, get, delete)
- Error scenarios and edge cases

```bash
# Set up credentials
export ATLAS_PUB_KEY="your-public-key"
export ATLAS_API_KEY="your-private-key"
export PROJECT_ID="your-project-id"
export ATLAS_ORG_ID="your-org-id"

# Run Atlas integration tests
./scripts/test-comprehensive.sh atlas
```

#### Database Integration Tests
Tests database commands against live MongoDB:
- Database listing and inspection
- Collection management (create, list, delete)
- Connection string handling
- Error scenarios

```bash
# Set up MongoDB connection
export MONGODB_CONNECTION_STRING="mongodb+srv://user:pass@cluster.mongodb.net/"

# Run database integration tests
./scripts/test-comprehensive.sh database
```

### 3. Infrastructure Tests (`test/infrastructure/`)
- **Location**: `test/infrastructure/`
- **Purpose**: Test system behavior under operational conditions
- **Coverage**: Performance, reliability, scale
- **Dependencies**: Live Atlas resources (may incur costs)
- **Runtime**: Long (10-60 minutes)

```bash
# Run infrastructure tests (requires confirmation)
./scripts/test-comprehensive.sh infrastructure
```

## 🔧 Test Configuration

### Environment Variables

#### Required for Atlas Tests
```bash
export ATLAS_PUB_KEY="your-atlas-public-key"
export ATLAS_API_KEY="your-atlas-private-key"
```

#### Optional for Atlas Tests
```bash
export PROJECT_ID="your-test-project-id"      # For project-specific tests
export ATLAS_ORG_ID="your-atlas-org-id"      # For organization tests
```

#### Required for Database Tests
```bash
export MONGODB_CONNECTION_STRING="mongodb+srv://..."
```

#### Test Configuration
```bash
export VERBOSE=true                    # Enable verbose output
export TEST_TIMEOUT=30m               # Set test timeout
export COVERAGE_THRESHOLD=85          # Set coverage threshold
```

### Using .env File

Create a `.env` file in the project root:

```bash
# Atlas credentials
ATLAS_PUB_KEY=your-public-key
ATLAS_API_KEY=your-private-key
PROJECT_ID=your-project-id
ATLAS_ORG_ID=your-org-id

# Database credentials
MONGODB_CONNECTION_STRING=mongodb+srv://user:pass@cluster.mongodb.net/

# Test configuration
VERBOSE=false
TEST_TIMEOUT=60m
COVERAGE_THRESHOLD=80
```

## 📊 Test Reports

All tests generate comprehensive reports in the `test-reports/` directory:

```
test-reports/
├── test-summary.md              # Comprehensive test summary
├── unit-coverage.html           # HTML coverage report
├── unit-coverage.txt            # Text coverage summary
├── unit-tests.json             # Detailed unit test results
├── atlas-integration-tests.json # Atlas integration results
├── database-integration-tests.json # Database integration results
├── infrastructure-tests.json   # Infrastructure test results
└── benchmarks.txt              # Benchmark results
```

## 🚀 Running Tests

### Comprehensive Test Runner

The main test runner (`scripts/test-comprehensive.sh`) provides a unified interface:

```bash
# Show help
./scripts/test-comprehensive.sh --help

# Run all tests
./scripts/test-comprehensive.sh all

# Run specific test category
./scripts/test-comprehensive.sh unit
./scripts/test-comprehensive.sh integration
./scripts/test-comprehensive.sh atlas
./scripts/test-comprehensive.sh database
./scripts/test-comprehensive.sh infrastructure

# Run with options
./scripts/test-comprehensive.sh --verbose --coverage-threshold 90 unit

# Generate reports only
./scripts/test-comprehensive.sh report
```

### Direct Go Commands

For development and debugging:

```bash
# Run unit tests with coverage
go test -v -race -cover ./internal/...

# Run specific integration tests
go test -v -tags=integration ./test/integration/atlas/...

# Run with build tags
go test -v -tags=infrastructure ./test/infrastructure/...

# Run benchmarks
go test -bench=. -benchmem ./internal/...
```

## 🎯 Test Coverage

### Current Coverage Targets
- **Unit Tests**: 90%+ coverage
- **Integration Tests**: Critical path coverage
- **Infrastructure Tests**: Operational scenario coverage

### Coverage Reports
- HTML reports: `test-reports/*-coverage.html`
- Text summaries: `test-reports/*-coverage.txt`
- Combined analysis: `test-reports/combined-coverage.html`

### Improving Coverage
1. Check coverage reports for gaps
2. Add unit tests for uncovered functions
3. Add integration tests for untested workflows
4. Use `go test -cover` for quick checks

## 🔍 Test Development Guidelines

### Writing Unit Tests
```go
func TestServiceMethod(t *testing.T) {
    // Arrange
    service := NewTestService()
    
    // Act
    result, err := service.Method(testInput)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expectedResult, result)
}
```

### Writing Integration Tests
```go
//go:build integration
// +build integration

func TestIntegrationScenario(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Setup real dependencies
    // Run test scenario
    // Verify results
}
```

### Test Tags
- `integration`: For tests requiring external services
- `infrastructure`: For tests that may incur costs
- No tags: Unit tests (run by default)

## 🚦 CI/CD Integration

### GitHub Actions (Recommended)
```yaml
name: Tests
on: [push, pull_request]
jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: ./scripts/test-comprehensive.sh unit
      
  integration-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - env:
          ATLAS_PUB_KEY: ${{ secrets.ATLAS_PUB_KEY }}
          ATLAS_API_KEY: ${{ secrets.ATLAS_API_KEY }}
        run: ./scripts/test-comprehensive.sh integration
```

### Local Development Workflow
```bash
# Before committing
./scripts/test-comprehensive.sh unit

# Before pushing
./scripts/test-comprehensive.sh integration

# Before releases
./scripts/test-comprehensive.sh all
```

## 🐛 Troubleshooting

### Common Issues

#### "Atlas credentials not available"
```bash
# Check environment variables
echo $ATLAS_PUB_KEY
echo $ATLAS_API_KEY

# Verify .env file
cat .env
```

#### "MongoDB connection failed"
```bash
# Test connection string
mongosh "$MONGODB_CONNECTION_STRING" --eval "db.runCommand('ping')"
```

#### "Tests timeout"
```bash
# Increase timeout
./scripts/test-comprehensive.sh --timeout 120m all
```

#### "Coverage below threshold"
```bash
# Check detailed coverage
go tool cover -html=test-reports/unit-coverage.out
```

### Test Debugging
```bash
# Run with verbose output
./scripts/test-comprehensive.sh --verbose unit

# Run specific test
go test -v -run TestSpecificFunction ./internal/package

# Debug with dlv
dlv test ./internal/package -- -test.run TestSpecificFunction
```

## 📚 Additional Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Assertions](https://github.com/stretchr/testify)
- [MongoDB Go Driver Testing](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo/integration)
- [Atlas Go SDK](https://github.com/mongodb/atlas-sdk-go)

## 🤝 Contributing

### Adding New Tests
1. Determine test category (unit/integration/infrastructure)
2. Place in appropriate directory
3. Follow naming conventions (`*_test.go`)
4. Add build tags if needed
5. Update test runner if necessary

### Test Naming Conventions
- Test functions: `TestFunctionName_Scenario`
- Benchmark functions: `BenchmarkFunctionName`
- Example functions: `ExampleFunctionName`

### Integration Test Checklist
- [ ] Uses build tags
- [ ] Skips when credentials unavailable
- [ ] Cleans up resources
- [ ] Has timeout protection
- [ ] Logs useful information

## 📈 Metrics and Monitoring

### Key Metrics
- Test execution time
- Coverage percentage
- Success/failure rates
- Resource usage

### Monitoring
- Track test performance over time
- Monitor integration test reliability
- Alert on coverage degradation
- Review infrastructure test costs 