# Features Tracking

## [2025-01-28] Comprehensive Discovery Feature Testing

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: Request for comprehensive discovery feature testing including incremental discovery workflows

### Summary
Created comprehensive test suite for the discovery feature covering basic discovery, ApplyDocument conversion, incremental discovery workflows, resource-specific discovery, format conversion, caching, and error handling. Implemented both shell-based lifecycle tests and Go integration tests to ensure robust testing coverage for the discovery functionality.

### Tasks
- [x] Analyze existing discovery functionality and test patterns
- [x] Design comprehensive test scenarios covering all discovery workflows
- [x] Implement Go integration tests for discovery APIs and services
- [x] Create discovery lifecycle shell script for end-to-end testing
- [x] Implement discovery command integration tests
- [x] Update main test runner to include discovery tests
- [x] Create feature tracking documentation

### Test Scenarios Implemented

#### 1. Basic Discovery Flow
- Discover project and existing resources
- Convert to ApplyDocument format
- Apply converted document to verify consistency (no changes)

#### 2. Incremental Discovery Testing  
- Add user to ApplyDocument and apply
- Detect new user in Atlas via discovery
- Run discovery again and verify user in results
- Remove user while retaining other resources

#### 3. Resource-specific Discovery
- Test individual resource discovery (clusters, users, network)
- Test filtering options (include/exclude)
- Verify specific resource conversion to manifest format

#### 4. Format Conversion Testing
- Test DiscoveredProject → ApplyDocument conversion
- Test applying converted documents
- Verify resource consistency after conversion

#### 5. Advanced Features Testing
- Test discovery caching functionality
- Test error handling and partial failures
- Test different output formats (YAML, JSON)
- Test timeout and context cancellation

### Files Created
- `test/integration/discovery/discovery_integration_test.go` - Core Go integration tests
- `test/integration/discovery/discovery_commands_integration_test.go` - CLI command integration tests
- `scripts/test/discovery-lifecycle.sh` - Shell-based lifecycle tests
- `tracking/features.md` - Updated with discovery testing feature entry

### Files Modified
- `scripts/test.sh` - Added discovery test command and comprehensive test suite integration

### Technical Implementation
- **Go Integration Tests**: 
  - `TestDiscovery_BasicFlow_Integration` - Basic discovery and conversion workflow
  - `TestDiscovery_IncrementalFlow_Integration` - User addition/removal lifecycle
  - `TestDiscovery_ResourceSpecific_Integration` - Individual resource discovery
  - `TestDiscovery_FormatConversion_Integration` - Format conversion validation
  - `TestDiscovery_ErrorHandling_Integration` - Error scenarios
  - `BenchmarkDiscovery_ProjectDiscovery` - Performance benchmarking

- **CLI Command Tests**:
  - `TestDiscoveryCommand_BasicDiscovery_Integration` - Basic CLI discovery
  - `TestDiscoveryCommand_ConvertToApplyDocument_Integration` - CLI conversion
  - `TestDiscoveryCommand_ResourceSpecific_Integration` - CLI filtering
  - `TestDiscoveryCommand_Caching_Integration` - CLI caching tests
  - `TestDiscoveryCommand_ErrorHandling_Integration` - CLI error handling

- **Shell Lifecycle Tests**:
  - `test_basic_discovery()` - End-to-end basic workflow
  - `test_incremental_discovery()` - User lifecycle testing
  - `test_resource_specific_discovery()` - Resource filtering
  - `test_discovery_with_filtering()` - Include/exclude testing
  - `test_discovery_formats()` - Output format validation
  - `test_discovery_caching()` - Cache performance testing

### Test Runner Integration
- Added `discovery` command to main test script
- Integrated discovery tests into comprehensive test suite
- Added proper cleanup and resource tracking
- Included in help documentation and examples

### Usage Examples
```bash
# Run all discovery tests
./scripts/test.sh discovery

# Run basic discovery tests only
./scripts/test/discovery-lifecycle.sh --basic-only

# Run incremental tests only  
./scripts/test/discovery-lifecycle.sh --incremental-only

# Run with Go integration tests
go test -tags=integration ./test/integration/discovery/... -v

# Include in comprehensive test suite
./scripts/test.sh comprehensive
```

### Testing Coverage
- ✅ Basic project discovery and ApplyDocument conversion
- ✅ Incremental discovery with resource addition/removal
- ✅ Resource-specific discovery and filtering
- ✅ Format conversion validation and consistency
- ✅ Caching functionality and performance
- ✅ Error handling and edge cases
- ✅ CLI command integration and error paths
- ✅ Timeout and context cancellation
- ✅ Output format validation (YAML, JSON)

---

## [2025-01-27] Password Display Feature for User Creation

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: User request for password display flag during user creation

### Summary
Enhanced user management functionality by adding a `--show-password` flag to user creation commands. This allows administrators to view passwords immediately after user creation for secure credential distribution while maintaining security best practices. Password update capabilities were verified to already exist.

### Tasks
- [x] Analyze current user creation and password handling functionality
- [x] Add `--show-password` flag to Atlas user creation command
- [x] Add `--show-password` flag to Database user creation command (framework ready)
- [x] Enhance output formatters to conditionally display passwords with security warnings
- [x] Verify existing password update functionality works for both user types
- [x] Confirm YAML ApplyDocument password support and security handling
- [x] Update documentation with new flag examples
- [x] Create comprehensive password display example YAML
- [x] Create feature tracking file with implementation details
- [x] Update changelog with password display feature

### Files Modified
- `cmd/atlas/users/users.go` - Added `--show-password` flag and updated function signatures
- `cmd/database/users/users.go` - Added `--show-password` flag (for future implementation)
- `internal/output/create_formatters.go` - Enhanced with password display capability and security warnings
- `docs/atlas.md` - Added examples with `--show-password` flag
- `docs/database.md` - Added examples with `--show-password` flag
- `examples/users-with-password-display.yaml` - New comprehensive password handling example
- `features/2025-01-27-password-display-feature.md` - Detailed feature documentation
- `CHANGELOG.md` - Added password display feature entry

### Technical Implementation
- **CLI Enhancement**: Added boolean `--show-password` flag to user creation commands
- **Output Formatting**: New `FormatCreateResultWithPassword()` method with conditional password display
- **Security Model**: Opt-in password display with clear warnings, passwords excluded from diffs
- **Backward Compatibility**: Existing workflows unchanged, purely additive feature

### Security Considerations
- Password display requires explicit opt-in via `--show-password` flag
- Security warning displayed when password is shown
- Passwords continue to be excluded from diff comparisons in apply pipeline
- YAML passwords still sourced from environment variables
- No changes to password storage or logging mechanisms

---

## [2025-01-27] User and Role Management Distinction

**Status**: Completed (Documentation and Structure) / In Progress (Implementation)  
**Developer**: Assistant  
**Related Issues**: User request for clearer separation between Atlas and Database user/role management

### Summary
Implemented clear architectural separation between Atlas-managed users/roles (API-based) and MongoDB database-level users/roles (connection-based). Added comprehensive documentation, CLI structure, and examples to eliminate confusion between the two distinct concepts.

### Tasks
- [x] Analyze current CLI structure for users and roles
- [x] Design clear separation between Atlas (API) and Database (connection) management
- [x] Update documentation to distinguish concepts clearly
- [x] Add clarification text to existing atlas users commands
- [x] Create database users subcommand structure
- [x] Update examples with both Atlas and Database approaches
- [x] Create architectural documentation
- [x] Implement full database users functionality (MongoDB createUser/updateUser/dropUser commands)
- [x] Add comprehensive database users tests to users-lifecycle.sh
- [x] Move architecture documentation to features/ directory

### Files Modified
- `docs/database.md` - Added distinction explanation and database users documentation
- `docs/atlas.md` - Added clarification for Atlas users
- `cmd/atlas/users/users.go` - Added clarification text
- `cmd/database/database.go` - Added database users subcommand
- `cmd/database/users/users.go` - Full database users command implementation
- `examples/atlas-vs-database-users-roles.yaml` - New comprehensive example
- `scripts/test/users-lifecycle.sh` - Added database users lifecycle tests
- `features/2025-01-27-user-role-management-distinction.md` - Comprehensive feature documentation

### Architecture Decision
**Atlas Level (API-Based)**:
- Command: `matlas atlas users`
- Management: Via Atlas Admin API
- Authentication: Atlas API keys
- Roles: Built-in MongoDB roles only (read, readWrite, dbAdmin, etc.)
- Scope: Project-level, centralized management

**Database Level (Connection-Based)**:
- Commands: `matlas database users` and `matlas database roles`
- Management: Direct MongoDB database connections
- Authentication: Database credentials or temporary Atlas users
- Roles: Both built-in and custom roles with granular privileges
- Scope: Database-specific, granular management

### Implementation Status
- ✅ CLI structure and documentation complete
- ✅ Database roles implementation exists and works correctly
- ✅ Database users implementation complete with full MongoDB operations
- ✅ YAML ApplyDocument support for DatabaseRole exists
- ⚠️ YAML ApplyDocument support for DatabaseDirectUser needs implementation
- ✅ Comprehensive test coverage for database users lifecycle

### Next Steps
1. Add YAML support for DatabaseDirectUser kind
2. Consider adding optional flags for explicit management-level specification
3. Add more advanced role inheritance features
4. Improve error messages and user guidance

### User Impact
- **No breaking changes** - all existing commands work as before
- **Clear documentation** distinguishing the two approaches
- **Better user experience** with explicit help text
- **Comprehensive examples** showing both management approaches
- **Future-proof architecture** for both centralized and granular user management

---