# matlas-cli

A command-line interface for MongoDB Atlas built with Go, providing comprehensive management of Atlas resources through a well-structured, tested codebase.

## Features

- **Hierarchical Command Structure**: Organized commands for Atlas (`atlas`), database operations (`database`), and declarative configuration (`apply`)
- **Atlas SDK Integration**: Built on the official Atlas Go SDK with retry logic and error handling
- **Comprehensive Resource Management**: Projects, Organizations, Clusters, Database Users, Network Access Lists
- **Robust Error Handling**: Typed errors with helper predicates for better error classification
- **Input Validation**: Consistent validation across all Atlas resource types
- **Extensive Testing**: >90% test coverage with unit, integration, and stress tests
- **Clean Architecture**: Modular service abstractions with clear separation of concerns
- **Future-Ready**: Structured for growth with dedicated packages for different concerns

## Installation

```bash
# Build from source
go build -o bin/matlas-cli ./main.go

# Or use make
make build
```

## Configuration

### Environment Variables

```bash
# Atlas API credentials (required)
export ATLAS_PUB_KEY="your-public-key"
export ATLAS_API_KEY="your-private-key"

# Optional: Atlas organization and project IDs
export ATLAS_ORG_ID="your-org-id"
export ATLAS_PROJECT_ID="your-project-id"
```

### Configuration File

Create `$HOME/.matlas/config.yaml`:

```yaml
atlas:
  publicKey: "your-public-key"
  privateKey: "your-private-key"
  retryMax: 3
  retryDelay: "250ms"
```

## Usage Examples

### Atlas Projects

```bash
# List all projects
matlas atlas projects list

# Get specific project
matlas atlas projects get --project-id 65f2cfaea1f13d03dc7de067

# List projects by organization
matlas atlas projects list --org-id 5ca37ef6a6f239b2387738cd
```

### Atlas Clusters

```bash
# List clusters in a project
matlas atlas clusters list --project-id 65f2cfaea1f13d03dc7de067
```

### Database Users

```bash
# List database users
matlas atlas users list --project-id 65f2cfaea1f13d03dc7de067
```

### Network Access Lists

```bash
# List IP access list entries
matlas atlas network list --project-id 65f2cfaea1f13d03dc7de067
```

### Infrastructure Management

```bash
# Apply configuration from file
matlas infra -f atlas-config.yaml

# Dry run to see what would change
matlas infra -f atlas-config.yaml --dry-run

# Destroy resources defined in config files
matlas infra destroy -f atlas-config.yaml

# Destroy all discovered resources (cleanup mode)
matlas infra destroy --discovery-only --project-id PROJECT_ID

# Show what would be destroyed
matlas infra destroy -f atlas-config.yaml --dry-run
```

## Architecture

The CLI is organized into several key layers:

### Command Structure (`cmd/`)

```
cmd/
├── atlas/          # Atlas management commands
│   ├── projects/   # Project operations
│   ├── clusters/   # Cluster operations
│   ├── users/      # Database user operations
│   └── network/    # Network access operations
├── database/       # Direct MongoDB operations (planned)
└── apply/          # Declarative configuration
```

### Service Layer (`internal/services/`)

Business logic organized by domain:

```
internal/services/
├── atlas/          # Atlas SDK services
│   ├── projects.go
│   ├── clusters.go
│   ├── users.go
│   └── network_access.go
├── database/       # MongoDB driver services (planned)
└── discovery/      # Project discovery logic (planned)
```

### Client Layer (`internal/clients/`)

SDK and driver wrappers with retry, logging, and error handling:

```
internal/clients/
├── atlas/          # Atlas client wrapper
│   ├── client.go
│   ├── retry.go
│   └── errors.go
└── mongodb/        # MongoDB driver wrapper (planned)
```

### Shared Components

- `internal/types/`: Shared type definitions and configuration schemas
- `internal/validation/`: Cross-cutting validation logic
- `internal/config/`: Configuration management
- `internal/apply/`: Declarative engine (planned)

### Atlas Integration Layer

The core Atlas integration is built around service abstractions that wrap the Atlas SDK:

```go
// Client wrapper with retry and logging
client, err := atlasclient.NewClient(atlasclient.Config{
    PublicKey:  "your-pub-key",
    PrivateKey: "your-private-key",
    RetryMax:   3,
    RetryDelay: 250 * time.Millisecond,
    Logger:     logger,
})

// Service abstractions
projectsService := atlasservice.NewProjectsService(client)
usersService := atlasservice.NewDatabaseUsersService(client)
```

### Error Handling

All services use typed errors for consistent error handling:

```go
projects, err := projectsService.List(ctx)
if err != nil {
    if atlasclient.IsNotFound(err) {
        // Handle not found
    } else if atlasclient.IsTransient(err) {
        // Retry logic
    } else if atlasclient.IsUnauthorized(err) {
        // Authentication issue
    }
}
```

### Input Validation

Consistent validation across all operations:

```go
// Validate Atlas resource IDs
if err := validation.ValidateProjectID(projectID); err != nil {
    return err
}

// Validate cluster names
if err := validation.ValidateClusterName(name); err != nil {
    return err
}
```

## Development

### Testing

```bash
# Run all tests
make test

# Run short tests (skip integration/stress tests)
make test-short

# Generate coverage report
make coverage

# Check Atlas package coverage (enforces ≥90%)
make coverage-atlas
```

### Code Generation

```bash
# Generate mocks for testing
make generate-mocks

# Regenerate specific mocks
cd internal/clients/atlas
go generate ./...
```

### Linting

```bash
# Run linter
make lint

# Format code
make fmt
```

## Contributing

1. **Code Quality**: Maintain >90% test coverage for service packages
2. **Error Handling**: Use typed errors and appropriate helper predicates
3. **Validation**: Add input validation for new resource types
4. **Testing**: Include unit tests, integration tests, and cleanup logic
5. **Documentation**: Update README and code comments

### Adding New Services

1. Create service file: `internal/services/atlas/new_service.go`
2. Add CRUD methods using the client wrapper
3. Include input validation using `internal/validation`
4. Add comprehensive tests: `internal/services/atlas/new_service_test.go`
5. Update `generate.go` with mock generation directive
6. Create command structure in `cmd/atlas/new_resource/`

## Architecture Decision Records

See `design/` directory for detailed architectural decisions:

- `adr-0003-atlas-integration-architecture.md`: Atlas integration layer design

## License

MIT License - see [LICENSE](LICENSE) file for details.
