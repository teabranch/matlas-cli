# Examples

Working YAML examples for `ApplyDocument` resources used by `matlas infra` and comprehensive demonstrations of CLI functionality.

## Discovery Examples

- **`discovery-basic.yaml`**: Basic discovered project converted to ApplyDocument format with cluster, user, and network access
- **`discovery-with-databases.yaml`**: Comprehensive discovery including database-level resources (databases, collections, indexes, custom roles)

## Cluster Examples

- **`cluster-basic.yaml`**: Minimal cluster definition for development environments
- **`cluster-advanced.yaml`**: Cluster with autoscaling and replication specifications
- **`cluster-comprehensive.yaml`**: Production-ready cluster with autoscaling, multi-region, security features, and proper tagging
- **`cluster-multiregion.yaml`**: Multi-region cluster using `replicationSpecs` and `regionConfigs`
- **`cluster-security-and-tags.yaml`**: Cluster with encryption, BI Connector, and comprehensive tags

## User Management Examples

- **`users-basic.yaml`**: Single DatabaseUser with basic read role
- **`users-standalone-multiple.yaml`**: Multiple users with different roles and labels
- **`users-scoped.yaml`**: Users scoped to specific clusters via `scopes`
- **`user-password-management.yaml`**: Comprehensive user management demonstrating password display features and different user types
- **`users-with-password-display.yaml`**: Users configured for password display during creation

## Authentication and Database Operations

- **`database-operations-authentication.yaml`**: Examples demonstrating the three authentication methods for database operations
- **`atlas-vs-database-users-roles.yaml`**: Comparison between Atlas-managed and database-level users

## Custom Roles Examples

- **`custom-roles-and-users.yaml`**: Basic custom role definition with associated user
- **`custom-roles-example.yaml`**: Comprehensive custom roles example
- **`custom-roles-comprehensive.yaml`**: Advanced custom roles with granular collection-level permissions

## Network Access Examples

- **`network-access.yaml`**: Basic network access configuration
- **`network-variants.yaml`**: Multiple NetworkAccess types (CIDR, IP with expiration, AWS security groups)
- **`overlay-network-and-user.yaml`**: Overlay-style addition of user and network access

## Infrastructure Management

- **`project-format.yaml`**: Project-format configuration for infrastructure commands
- **`project-with-cluster-and-users.yaml`**: Complete project with cluster and users in one document
- **`safe-operations-preserve-existing.yaml`**: Demonstrates safe operations using `--preserve-existing` flag
- **`dependencies-and-deletion.yaml`**: Resource dependencies and deletion policies

## Usage

### Environment Variables

Replace placeholders like "My Project" and provide environment variables for passwords before running:

```bash
# Basic user passwords
export APP_USER_PASSWORD='StrongPass123!'
export APP_WRITER_PASSWORD='StrongPass123!'
export ANALYTICS_PASSWORD='StrongPass123!'
export OVERLAY_USER_PASSWORD='StrongPass123!'
export ROLE_USER_PASSWORD='StrongPass123!'

# Advanced user passwords for comprehensive examples
export SERVICE_ACCOUNT_PASSWORD='ServicePass123!'
export DATABASE_ADMIN_PASSWORD='AdminPass123!'
export CLUSTER_USER_PASSWORD='ClusterPass123!'
export ECOMMERCE_APP_PASSWORD='EcommercePass123!'
export DATABASE_MANAGER_PASSWORD='ManagerPass123!'

# Database operations passwords
export DB_OPERATIONS_PASSWORD='DbOpsPass123!'
export SAFE_TEST_PASSWORD='SafeTestPass123!'
```

### Discovery Workflows

```bash
# Discover existing project
matlas discover --project-id <project-id> --output-file discovered.yaml

# Discover with database resources
matlas discover \
  --project-id <project-id> \
  --include-databases \
  --use-temp-user \
  --convert-to-apply \
  --output-file complete-discovery.yaml

# Filter discovery by resource type
matlas discover --project-id <project-id> --include clusters,users --output-file filtered.yaml
```

### Infrastructure Operations

```bash
# Validate configurations
matlas infra validate -f examples/users-basic.yaml
matlas infra validate -f examples/cluster-comprehensive.yaml

# Preview changes
matlas infra diff -f examples/custom-roles-comprehensive.yaml --detailed
matlas infra plan -f examples/discovery-basic.yaml --output table

# Apply changes safely
matlas infra apply -f examples/safe-operations-preserve-existing.yaml --preserve-existing --auto-approve
matlas infra apply -f examples/overlay-network-and-user.yaml --dry-run --dry-run-mode thorough
```

### Database Operations with New Authentication

```bash
# Create database with temporary user (recommended)
matlas database create inventory \
  --cluster my-cluster \
  --project-id <project-id> \
  --collection products \
  --use-temp-user

# Create database with manual credentials
matlas database create inventory \
  --cluster my-cluster \
  --project-id <project-id> \
  --collection products \
  --username dbuser \
  --password dbpass

# Create database with direct connection
matlas database create inventory \
  --connection-string "mongodb+srv://user:pass@cluster/" \
  --collection products
```

### User Management with Password Display

```bash
# Create user and display password
matlas atlas users create \
  --project-id <project-id> \
  --username myuser \
  --roles "readWrite@myapp" \
  --show-password

# Create database user with custom role
matlas database users create appuser \
  --cluster my-cluster \
  --project-id <project-id> \
  --database myapp \
  --use-temp-user \
  --password "SecurePass123!" \
  --roles "customRole@myapp" \
  --show-password
```

## Implementation Notes

- **Database Creation**: All database creation now requires `--collection` parameter for immediate visibility
- **Authentication Methods**: Choose one of three methods: `--use-temp-user`, `--username/--password`, or `--connection-string`
- **Safety Features**: Use `--preserve-existing` flag to protect existing resources during apply operations
- **Discovery**: Include `--include-databases` for comprehensive resource enumeration
- **Custom Roles**: Use `matlas database roles create` commands for role implementation

These examples mirror structures used in the test scripts under `scripts/test/` and adhere to the types in `internal/types/`.
