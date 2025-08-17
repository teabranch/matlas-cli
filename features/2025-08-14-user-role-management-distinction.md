# Feature: User and Role Management Distinction

## Summary

Implemented comprehensive separation between Atlas-managed users/roles (API-based) and MongoDB database-level users/roles (connection-based) to eliminate confusion and provide clear architectural boundaries.

## Context

The matlas-cli previously had unclear separation between Atlas database users managed via API and MongoDB custom roles created via direct database connections. Users were confused about when to use `atlas users` vs `database roles` commands, and there was no equivalent `database users` functionality.

## Implementation

### CLI Architecture

**Atlas Level Commands (API-Based)**:
- `matlas atlas users` - Centralized project-level user management
- Built-in MongoDB roles only (read, readWrite, dbAdmin, etc.)  
- Uses Atlas Admin API and API key authentication
- Manages users across multiple databases/clusters

**Database Level Commands (Connection-Based)**:
- `matlas database users` - Database-specific user management (NEW)
- `matlas database roles` - Custom roles with granular privileges (EXISTING)
- Uses MongoDB database connections or temporary Atlas users
- Supports both built-in and custom roles

### Key Components

#### Database Users Implementation
- **Commands**: list, create, get, update, delete
- **MongoDB Commands**: `createUser`, `updateUser`, `dropUser`, `usersInfo`
- **Features**: Password management, incremental role updates, retry logic
- **Connection**: Direct connection string or Atlas cluster with temp user

#### Connection Resolution
- Reuses Atlas cluster resolution with temporary user creation
- Enhanced user propagation timing for user management operations
- Proper credential encoding and connection string formatting

#### Enhanced Testing
- Comprehensive test suite in `users-lifecycle.sh`
- Tests full CRUD operations for database users
- Retry logic for Atlas user propagation delays
- Integration with existing role and database tests

### Files Modified

#### Implementation
- `cmd/database/users/users.go` - New database users command implementation
- `cmd/database/database.go` - Added users subcommand
- `cmd/atlas/users/users.go` - Added clarification text

#### Documentation  
- `docs/database.md` - Added database users section and distinction explanation
- `docs/atlas.md` - Added Atlas users clarification
- `examples/atlas-vs-database-users-roles.yaml` - Comprehensive comparison example

#### Testing
- `scripts/test/users-lifecycle.sh` - Added database users lifecycle tests

#### Tracking
- `tracking/features.md` - Updated with implementation details

## Breaking Changes

None. All existing commands work as before.

## Migration Path

- Existing `atlas users` commands unchanged
- Existing `database roles` commands unchanged  
- New `database users` commands available immediately
- Documentation clarifies when to use each approach

## Use Cases

### Use Atlas Commands When:
- Managing users across multiple databases/clusters
- Need centralized user management via Atlas dashboard
- Working with Atlas projects and organizations
- Using built-in MongoDB roles
- Need Atlas-level authentication and auditing

### Use Database Commands When:
- Need custom roles with granular privileges
- Working with specific database-level operations
- Need collection-level permissions
- Working with on-premises or non-Atlas MongoDB
- Creating application-specific roles and users

## Future Enhancements

### Planned
- YAML ApplyDocument support for DatabaseDirectUser kind
- Optional clarification flags (--management-level, --auth-method)

### Considered
- Unified user management interface
- Role inheritance visualization
- Automated migration between Atlas and Database users

## Testing Strategy

- Unit tests for all database user operations
- Integration tests with Atlas clusters
- Retry logic testing for user propagation
- Error handling for authentication failures
- YAML configuration validation

## Documentation Strategy

- Clear distinction in all help text
- Comprehensive examples for both approaches
- Architecture documentation for developers
- Migration guidance for existing users

## Success Metrics

- ✅ Zero breaking changes to existing functionality
- ✅ Clear separation of concerns in CLI structure
- ✅ Comprehensive test coverage for new functionality
- ✅ Documentation clarity improvements
- ✅ Feature-complete database user management

## Related Issues

- Original confusion between Atlas and Database role concepts
- Missing database-level user management functionality
- Need for granular permission management in applications
- Test coverage gaps in user lifecycle operations

