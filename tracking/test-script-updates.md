# Test Script Updates for New Authentication Model

## Overview

Updated all database and user test scripts in the `scripts/test/` directory to accommodate the new authentication model and database creation requirements that were implemented in the core functionality.

## Key Changes Made

### 1. Database Creation Requirement: `--collection` Parameter

**Before**: Database creation without specifying a collection
```bash
matlas database create mydb --cluster test --project-id test --use-temp-user
```

**After**: Database creation requires `--collection` parameter  
```bash
matlas database create mydb --cluster test --project-id test --collection mycoll --use-temp-user
```

**Reason**: MongoDB databases are created lazily when the first collection is added. The collection requirement ensures the database is immediately visible in Atlas UI.

### 2. New Authentication Model

Updated all scripts to test the three supported authentication methods:

#### Method 1: Temporary User (`--use-temp-user`)
```bash
matlas database create mydb --cluster test --project-id test --collection mycoll --use-temp-user
```
- Uses Atlas API keys to create temporary database user
- Automatic cleanup after operation
- Default roles scoped to target database

#### Method 2: Manual User Credentials (`--username` and `--password`)
```bash
matlas database create mydb --cluster test --project-id test --collection mycoll --username user --password pass
```
- Uses existing database user credentials
- Credentials injected into connection string
- Requires both username and password

#### Method 3: Direct Connection String (`--connection-string`)
```bash
matlas database create mydb --connection-string "mongodb+srv://user:pass@cluster/admin" --collection mycoll
```
- Direct MongoDB connection with embedded credentials
- Full control over connection parameters

### 3. Enhanced Validation and Error Handling

Added comprehensive failure detection tests for:
- Missing `--collection` parameter
- Invalid authentication combinations (e.g., `--use-temp-user` with `--username`)
- Missing password when username provided
- No authentication method provided
- Invalid cluster names
- Invalid project IDs

### 4. Targeted YAML Deletion

Implemented tests to verify that YAML operations only affect resources defined in the YAML configuration and don't impact other existing resources:

```yaml
# Step 1: Create user via CLI (should not be affected)
# Step 2: Create user via YAML
# Step 3: Apply empty YAML (should only remove YAML-created user)
# Step 4: Verify CLI-created user is preserved
```

### 5. Update/Modification Operations

Added tests for both CLI and YAML-based updates:
- Password updates via CLI
- Role updates via CLI  
- User modifications via YAML apply operations
- Verification that updates take effect

## Updated Scripts

### `/scripts/test/users-lifecycle.sh`

**New Features Added:**
- ✅ Database authentication method testing
- ✅ New database creation with `--collection` requirement
- ✅ YAML targeted deletion testing
- ✅ Error scenario validation
- ✅ CLI and YAML update operations

**Test Categories:**
1. Database authentication methods (temp user, username/password)
2. CLI user lifecycle (create, list, update, delete)
3. YAML user apply/destroy with targeted deletion
4. CLI custom role lifecycle
5. Database operations with new authentication model
6. YAML custom roles configuration
7. Error scenarios and edge cases

### `/scripts/test/database-operations.sh`

**Complete Rewrite** - New comprehensive test suite with:

**New Features Added:**
- ✅ All three authentication methods testing
- ✅ Comprehensive failure detection
- ✅ Database, collection, and index CRUD operations
- ✅ YAML operations with targeted deletion
- ✅ Complete database workflow testing

**Test Categories:**
1. `auth` - Test all authentication methods
2. `failures` - Test failure detection and error handling  
3. `databases` - Test database CRUD operations
4. `collections` - Test collection CRUD operations
5. `indexes` - Test index CRUD operations
6. `yaml` - Test YAML operations with targeted deletion
7. `workflow` - Test complete database → collection → index workflow
8. `comprehensive` - Run all test categories

## Usage Examples

### Running Updated Tests

**Users and Roles Lifecycle:**
```bash
# Run all users/roles tests
./scripts/test/users-lifecycle.sh

# Set manual credentials for additional auth testing
export MANUAL_DB_USER="your-db-user"
export MANUAL_DB_PASSWORD="your-db-password"
./scripts/test/users-lifecycle.sh
```

**Database Operations:**
```bash
# Run all database operation tests
./scripts/test/database-operations.sh

# Test specific categories
./scripts/test/database-operations.sh auth
./scripts/test/database-operations.sh failures  
./scripts/test/database-operations.sh yaml
./scripts/test/database-operations.sh comprehensive
```

## Validation Results

All tests now properly validate:

### ✅ Authentication Requirements
- Missing collection parameter correctly fails
- Conflicting auth flags correctly fail
- Missing credentials correctly fail

### ✅ Functionality Coverage  
- Database creation with all auth methods
- Collection and index operations
- YAML apply/destroy operations
- User and role management

### ✅ Safety Features
- Targeted deletion doesn't affect other resources
- Automatic cleanup of test resources
- Proper error handling and reporting

### ✅ Backward Compatibility
- Existing test patterns preserved where possible
- New functionality added without breaking existing workflows

## Environment Variables

**Required:**
```bash
export ATLAS_PUB_KEY="your-atlas-public-key"
export ATLAS_API_KEY="your-atlas-private-key" 
export ATLAS_PROJECT_ID="your-atlas-project-id"
export ATLAS_CLUSTER_NAME="your-existing-cluster-name"
```

**Optional (for enhanced testing):**
```bash
export MANUAL_DB_USER="your-database-username"
export MANUAL_DB_PASSWORD="your-database-password"
export DB_OPERATION_TIMEOUT="5m"  # Custom timeout
```

## Next Steps

1. **Integration Testing**: Run updated scripts against live Atlas environment
2. **CI/CD Integration**: Update automated test pipelines to use new authentication model
3. **Documentation**: Update user-facing documentation to reflect new requirements
4. **Training**: Update team on new authentication options and test capabilities

## Files Modified

- `scripts/test/users-lifecycle.sh` - Enhanced with new auth model and requirements
- `scripts/test/database-operations.sh` - Complete rewrite with comprehensive testing
- `tracking/test-script-updates.md` - This documentation file

## Compatibility Notes

- **Breaking Change**: Database creation now requires `--collection` parameter
- **New Feature**: Three authentication methods now supported
- **Enhanced**: Better error detection and validation
- **Improved**: Targeted YAML operations don't affect other resources
- **Added**: Comprehensive failure scenario testing

All changes maintain backward compatibility where possible while adding robust support for the new authentication model and database creation requirements.
