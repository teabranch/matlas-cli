# matlas-cli Scripts

Clean, organized scripts for building and testing matlas-cli.

## 🚀 Quick Start

```bash
# Setup development environment
./scripts/utils/setup.sh

# Run all tests
./scripts/test.sh all

# Build the project
./scripts/build/build.sh
```

## 📁 Structure

```
scripts/
├── test.sh              # Main test runner (entry point)
├── test/                 # Individual test type scripts
│   ├── unit.sh          # Unit tests (fast, no dependencies)
│   ├── integration.sh   # Integration tests (live Atlas API)
│   ├── e2e.sh           # End-to-end tests (complete workflows)
│   └── cluster-lifecycle.sh  # Real cluster lifecycle tests (creates clusters!)
├── build/               # Build scripts
│   └── build.sh         # Build binary with versioning
├── utils/               # Utility scripts
│   ├── clean.sh         # Clean cache and artifacts
│   └── setup.sh         # Setup development environment
└── README.md            # This file
```

## 🧪 Testing

### Main Test Runner
```bash
./scripts/test.sh [COMMAND] [OPTIONS]

# Examples:
./scripts/test.sh unit                 # Unit tests only
./scripts/test.sh integration          # Integration tests only
./scripts/test.sh e2e                  # E2E tests only
./scripts/test.sh all                  # All tests
./scripts/test.sh all --coverage       # All tests with coverage
./scripts/test.sh clean                # Clean test cache
```

### Individual Test Scripts
```bash
# Unit tests (no Atlas connection required)
./scripts/test/unit.sh [--coverage] [--verbose]

# Integration tests (requires Atlas credentials)
./scripts/test/integration.sh [--dry-run]

# E2E tests (requires Atlas credentials)
./scripts/test/e2e.sh [--dry-run] [--include-clusters]
```

## 🏗️ Building

```bash
./scripts/build/build.sh [COMMAND] [OPTIONS]

# Examples:
./scripts/build/build.sh                    # Build for current platform
./scripts/build/build.sh build -o mymatlas  # Custom binary name
./scripts/build/build.sh cross              # Cross-compile all platforms
./scripts/build/build.sh clean              # Clean build artifacts
```

## 🛠️ Utilities

### Setup Development Environment
```bash
./scripts/utils/setup.sh
```
- Checks Go installation
- Creates .env template
- Installs dependencies
- Builds project
- Sets up git hooks

### Clean Cache and Artifacts
```bash
./scripts/utils/clean.sh [TARGET]

# Examples:
./scripts/utils/clean.sh                # Clean everything
./scripts/utils/clean.sh cache          # Clean test cache only
./scripts/utils/clean.sh reports        # Clean test reports only
./scripts/utils/clean.sh resources      # Check for leftover Atlas resources
```

## ⚙️ Configuration

### Environment Variables
Create `.env` file in project root:
```bash
# Atlas credentials (required for integration/e2e tests)
export ATLAS_PUB_KEY=your-atlas-public-key
export ATLAS_API_KEY=your-atlas-private-key
export ATLAS_PROJECT_ID=your-atlas-project-id
export ATLAS_ORG_ID=your-atlas-org-id
```

### Test Configuration
```bash
# Override defaults with environment variables
export TEST_TIMEOUT=30m              # Test timeout
export COVERAGE_THRESHOLD=80         # Coverage threshold %
export VERBOSE=true                  # Enable verbose output
```

## 📊 Test Types

### Unit Tests (`./scripts/test/unit.sh`)
- **Purpose**: Fast, isolated tests
- **Dependencies**: None (no Atlas connection)
- **Duration**: ~1-2 minutes
- **Coverage**: Internal packages, command logic
- **Safety**: Completely safe, no external resources

### Integration Tests (`./scripts/test/integration.sh`)
- **Purpose**: Test live Atlas API interactions
- **Dependencies**: Atlas credentials required
- **Duration**: ~5-10 minutes
- **Coverage**: Database users, network access
- **Safety**: Creates and automatically cleans up test resources

### E2E Tests (`./scripts/test/e2e.sh`)
- **Purpose**: Complete workflow testing
- **Dependencies**: Atlas credentials required
- **Duration**: ~10-15 minutes (default), ~30-45 minutes (with clusters)
- **Coverage**: Full command workflows, infra commands
- **Safety**: Creates and automatically cleans up test resources
- **Note**: By default, only creates users/network access. Use `--include-clusters` for real cluster testing

#### E2E Test Modes:
```bash
# Standard E2E tests (no cluster creation)
./scripts/test/e2e.sh

# Include real cluster lifecycle tests (creates actual clusters!)
./scripts/test/e2e.sh --include-clusters

# Show what tests would run
./scripts/test/e2e.sh --dry-run
./scripts/test/e2e.sh --include-clusters --dry-run
```

### Cluster Lifecycle Tests (`./scripts/test/cluster-lifecycle.sh`)
- **Purpose**: Real cluster creation, modification, and deletion testing
- **Dependencies**: Atlas credentials required
- **Duration**: ~30-45 minutes
- **Coverage**: Complete cluster lifecycle with CLI and YAML approaches
- **Safety**: Creates real clusters (costs money!) but guarantees cleanup
- **⚠️ WARNING**: This creates actual Atlas clusters and incurs real costs!

#### Cluster Test Modes:
```bash
# Run all cluster lifecycle tests (CLI + YAML)
./scripts/test/cluster-lifecycle.sh

# Run CLI tests only
./scripts/test/cluster-lifecycle.sh cli

# Run YAML tests only  
./scripts/test/cluster-lifecycle.sh yaml
```

## 🛡️ Safety Features

### Automatic Cleanup
- All scripts with Atlas resources include automatic cleanup
- Cleanup runs on script exit, interruption (Ctrl+C), or error
- Resources are tracked and cleaned up even if script crashes

### Resource Protection
- All test resources use unique names with timestamps
- Test scripts never modify existing production resources
- Dry-run modes available for testing without creating resources

### Error Handling
- All scripts use `set -euo pipefail` for strict error handling
- Comprehensive error messages and troubleshooting info
- Cleanup verification and reporting

## 🚨 Troubleshooting

### Tests Fail to Start
```bash
# Check environment
./scripts/utils/setup.sh

# Verify Atlas credentials
cat .env
```

### Leftover Resources
```bash
# Check for test resources in Atlas
./scripts/utils/clean.sh resources

# Clean everything
./scripts/utils/clean.sh all
```

### Build Issues
```bash
# Clean and rebuild
./scripts/build/build.sh clean
./scripts/build/build.sh
```

## 📝 Best Practices

### Development Workflow
1. Run `./scripts/utils/setup.sh` after cloning
2. Use `./scripts/test.sh unit` during development
3. Run `./scripts/test.sh all` before committing
4. Use `--dry-run` to test without creating resources

### CI/CD Integration
```bash
# In CI pipeline:
./scripts/utils/setup.sh          # Setup environment
./scripts/test.sh all --coverage  # Run all tests with coverage
./scripts/build/build.sh cross    # Build for all platforms
```

### Resource Management
- Use dedicated Atlas projects for testing
- Never run tests against production projects
- Review cleanup logs after test runs
- Keep Atlas API key permissions minimal

---

**Simple. Clean. Reliable.**

Each script has a single, clear purpose and can be run independently or as part of the main test runner.