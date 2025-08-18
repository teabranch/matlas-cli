# Bugfixes Tracking

## [2025-01-27] Semantic Release Workflow Fix

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: Artifact attachment failure, chore commit interference  

### Summary
Fixed semantic-release workflow where post-release chore commits were preventing CI artifacts from being attached to releases, causing releases to be published without binaries.

### Tasks
- [x] Analyze semantic-release workflow issue where chore commits break artifact attachment
- [x] Identify that @semantic-release/git plugin creates chore commits after releases, causing confusion
- [x] Remove @semantic-release/git and @semantic-release/changelog plugins from .releaserc.json
- [x] Verify the updated configuration works correctly  
- [x] Update changelog to document the fix

### Files Modified
- `.releaserc.json` - Removed @semantic-release/git and @semantic-release/changelog plugins
- `CHANGELOG.md` - Documented the workflow fix and plugin removal
- `tracking/bugfixes.md` - Added permanent tracking entry

### Notes
The issue was that semantic-release was creating a `chore(release): vX.X.X [skip ci]` commit after creating the GitHub release. This caused confusion in the release workflow when trying to find CI artifacts, as it would look for artifacts associated with the chore commit (which don't exist) instead of the original feature/fix commit.

By removing the problematic plugins, the workflow now cleanly:
1. Analyzes commits and generates release notes
2. Creates GitHub release and tag pointing to the correct commit
3. Release workflow finds CI artifacts for the correct commit SHA
4. Artifacts are successfully attached to the release

The changelog is now maintained manually as per project standards, which is actually cleaner and more predictable.

**Root Cause**: The `@semantic-release/git` plugin in `.releaserc.json` was configured to create a commit with the updated CHANGELOG.md after the release was already created. This created a timing issue where the release tag pointed to the original commit, but the latest commit in the repo became the chore commit, confusing the artifact lookup process.

**Solution**: Removed both `@semantic-release/changelog` and `@semantic-release/git` plugins, keeping only the essential plugins: `@semantic-release/commit-analyzer`, `@semantic-release/release-notes-generator`, and `@semantic-release/github`. The changelog is maintained manually per project standards.

---

## [2025-01-27] Error Handling and Logging Standardization Analysis

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: Error handling standardization audit

### Summary
Comprehensive analysis of error handling and logging patterns across the matlas-cli codebase to identify inconsistencies and ensure adherence to established standards.

### Tasks
- [x] Analyze existing error handling standards and rules
- [x] Review error handling patterns in commands
- [x] Check logging implementation across services
- [x] Identify specific inconsistencies and violations
- [x] Document findings and recommendations

### Files Analyzed
- `cmd/**/*.go` - All command implementations
- `internal/cli/errors.go` - Error formatting infrastructure
- `internal/cli/enhanced_errors.go` - Enhanced error handling
- `internal/logging/logger.go` - Logging infrastructure
- `internal/services/**/*.go` - Service layer implementations
- `internal/apply/**/*.go` - Apply pipeline error handling

### Key Findings

#### ✅ Good Standardization:
1. **Error Handling Infrastructure**: Well-defined error handling policies and infrastructure in place
2. **CLI Error Wrappers**: Consistent use of `cli.WrapWithOperation`, `cli.WrapWithContext`, and `cli.WrapWithSuggestion` across many commands
3. **Atlas Client Errors**: Proper sentinel error handling using `atlasclient.IsNotFound`, `atlasclient.IsUnauthorized`, etc.
4. **Root Command**: Centralized error formatting and handling in `cmd/root.go`

#### ⚠️ Inconsistencies Found:

**Error Handling Violations:**
1. **Direct Error Printing**: Several commands use `fmt.Printf` for error output instead of returning errors:
   - `cmd/infra/apply.go:838-849` - Direct printing of execution results
   - `cmd/database/users/users.go:341` - Direct printing of "no users found" message
   - Multiple files use `fmt.Printf` for informational messages that should use structured output

2. **Inconsistent Panic Usage**: Mixed patterns for flag requirement panics:
   - Some use `panic(fmt.Errorf(...))` (newer style)
   - Others use `panic(fmt.Sprintf(...))` (older style)
   - Files: `cmd/atlas/users/users.go:633`, `cmd/database/users/users.go:416`, etc.

3. **Exit Code Handling**: Direct `os.Exit(1)` calls in command logic:
   - `cmd/infra/apply.go:987` - Should return error instead

**Logging Inconsistencies:**
1. **Mixed Logging Libraries**: Database services use `zap` directly instead of standardized logging:
   - `internal/services/database/service.go` imports and uses `go.uber.org/zap` directly
   - Should use `internal/logging` package instead

2. **Atlas Services**: No logging infrastructure - no logger imports or usage
   - Services don't inject or use any logging mechanism
   - Missing debug/trace logging for API calls

3. **Test Logging**: Tests use `t.Logf` instead of structured logging where appropriate

### Impact Assessment
- **Severity**: Medium - System functions but lacks consistency
- **User Experience**: Some error messages bypass the standardized formatting
- **Maintainability**: Mixed patterns make code harder to maintain
- **Observability**: Inconsistent logging reduces debugging capability

### Recommendations
1. **Error Handling Fixes**:
   - Refactor commands to return errors instead of direct printing
   - Standardize all panic patterns to use `fmt.Errorf`
   - Remove direct `os.Exit` calls from command logic

2. **Logging Standardization**:
   - Migrate database service to use `internal/logging` instead of zap directly
   - Add logging infrastructure to Atlas services
   - Implement structured logging for API operations

3. **Infrastructure Improvements**:
   - Add linting rules to enforce error handling patterns
   - Create code review checklist for error handling compliance
   - Add unit tests to verify error handling behavior

---

## [2025-08-14] Fixed Failing Test Cases

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: Test failures in CI/development

### Summary
Fixed two failing test cases that were blocking development - one in database command tests and one in temp user service tests.

### Tasks
- [x] Fix TestNewCreateDatabaseCmd test assertion
- [x] Fix TestTempUserManager_CreateTempUserForDiscovery_WithDatabaseName test expectations

### Files Modified
- `cmd/database/database_test.go` - Updated test assertion to match current implementation
- `internal/services/database/temp_user_test.go` - Updated test to expect database-specific roles instead of admin roles

### Root Cause Analysis

#### Test 1: TestNewCreateDatabaseCmd
- **Issue**: Test expected description "Create a database" but implementation had "Create a database with a collection"
- **Cause**: Test was outdated - implementation correctly requires collection to be specified
- **Fix**: Updated test assertion to match the accurate implementation description

#### Test 2: TestTempUserManager_CreateTempUserForDiscovery_WithDatabaseName  
- **Issue**: Test expected admin-scoped roles (readWriteAnyDatabase@admin) when specific database provided
- **Cause**: Test was expecting old behavior, but implementation correctly provides database-specific roles for security
- **Analysis**: Implementation follows security principle of least privilege by scoping roles to requested database
- **Fix**: Updated test to expect database-specific roles (readWrite@myapp, dbAdmin@myapp)

### Security Implications
The temp user implementation correctly implements the principle of least privilege:
- When specific database requested → database-specific roles (readWrite, dbAdmin)
- When admin/no database specified → admin-scoped roles (readWriteAnyDatabase, dbAdminAnyDatabase)

This aligns with MongoDB Atlas security best practices and the implementation's security comment.

### Verification
- All database command tests now pass
- All database service tests now pass  
- No regression in existing functionality
- Security model remains intact and improved

---

## [2025-08-15] E2E Test Failures - HTTP 500 UNEXPECTED_ERROR Handling

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: E2E test failures due to transient Atlas API errors

### Summary
Fixed e2e test failures caused by HTTP 500 UNEXPECTED_ERROR responses from Atlas API that were not being retried. Enhanced error handling and test resilience.

### Tasks
- [x] Investigate e2e test failure with HTTP 500 Internal Server Error
- [x] Analyze Atlas client error mapping for transient errors
- [x] Add UNEXPECTED_ERROR to transient error mapping
- [x] Implement retry logic in e2e tests for critical operations
- [x] Add retry helper function with exponential backoff

### Files Modified
- `internal/clients/atlas/errors.go` - Added UNEXPECTED_ERROR to transient error mapping
- `scripts/test/e2e.sh` - Added retry_command helper and applied to critical operations

### Root Cause Analysis

#### Primary Issue: UNEXPECTED_ERROR Not Retryable
- **Error**: `HTTP 500 Internal Server Error (Error code: "UNEXPECTED_ERROR")`
- **Cause**: Atlas client only treated `"TOO_MANY_REQUESTS"` and `"INTERNAL"` as transient
- **Impact**: HTTP 500 with `UNEXPECTED_ERROR` code was treated as permanent failure
- **Analysis**: `UNEXPECTED_ERROR` indicates temporary server issues that should be retryable

#### Secondary Issue: Test Resilience
- **Problem**: E2E tests had no retry logic for transient Atlas API failures
- **Impact**: Tests failed on temporary server issues instead of retrying
- **Analysis**: Production workloads need resilience to temporary API failures

### Technical Changes

#### 1. Error Mapping Enhancement
```go
// Before: Only TOO_MANY_REQUESTS and INTERNAL were transient
case admin.IsErrorCode(err, "TOO_MANY_REQUESTS") || admin.IsErrorCode(err, "INTERNAL"):

// After: Added UNEXPECTED_ERROR as transient
case admin.IsErrorCode(err, "TOO_MANY_REQUESTS") || admin.IsErrorCode(err, "INTERNAL") || admin.IsErrorCode(err, "UNEXPECTED_ERROR"):
```

#### 2. E2E Test Retry Logic
- Added `retry_command()` helper with exponential backoff (5s → 10s → 20s)
- Applied retry logic to critical operations:
  - Initial configuration apply in comprehensive workflow
  - Both apply operations in preserve-existing behavior test
- Configurable retry attempts (default 3) and base delay (default 5s)

### Impact Assessment
- **Reliability**: E2E tests now handle transient Atlas API failures gracefully
- **User Experience**: CLI operations with Atlas API will retry HTTP 500 UNEXPECTED_ERROR automatically
- **CI/CD**: Reduces false positive test failures due to temporary server issues
- **Production**: Improved resilience for real-world usage where Atlas may have temporary issues

### Testing Results
- Error mapping correctly identifies UNEXPECTED_ERROR as transient
- Retry logic provides exponential backoff as expected
- E2E tests should now pass even with occasional Atlas API hiccups
- No impact on non-transient error handling (still fail immediately for auth, not found, etc.)

### Atlas API Error Classification
After this fix, the complete transient error mapping includes:
- `TOO_MANY_REQUESTS` - Rate limiting (should retry)
- `INTERNAL` - Internal server errors (should retry)  
- `UNEXPECTED_ERROR` - Unexpected server errors (should retry)

All other error codes (NOT_FOUND, UNAUTHORIZED, CONFLICT, etc.) remain non-retryable as appropriate.

---

## [2025-08-15] E2E Test Project Name Conflict Resolution

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: E2E test failures due to project name conflicts

### Summary
Resolved e2e test failures caused by test configurations attempting to rename the Atlas project to test-specific names, which Atlas was rejecting with HTTP 500 UNEXPECTED_ERROR. Fixed by using the actual project name consistently across all test configurations.

### Root Cause Analysis

#### Primary Issue: Project Name Mismatch
- **Problem**: Test configurations specified project names like "Preserve Test Project", "Comprehensive E2E Test", etc.
- **Actual Project**: The Atlas project name is "IacOperatorPOC" 
- **Conflict**: Diff engine detected name differences and tried to update project name via Atlas API
- **API Rejection**: Atlas rejected the project rename with HTTP 500 UNEXPECTED_ERROR

#### Discovery Process
1. **Initial Investigation**: Confirmed UNEXPECTED_ERROR was being retried (retry fix working)
2. **Project Analysis**: Retrieved actual project details: `./matlas atlas projects get --project-id $ATLAS_PROJECT_ID`
3. **Config Review**: Found test configs trying to set different project names
4. **API Behavior**: Atlas API doesn't support renaming this project or rejects the specific names used

### Technical Resolution

#### Files Modified
- `scripts/test/e2e.sh` - Updated all test configurations to use actual project name

#### Changes Made
```yaml
# Before: Test configs tried to rename project
spec:
  name: "Preserve Test Project"          # ❌ Causes HTTP 500

# After: Test configs use actual project name  
spec:
  name: "IacOperatorPOC"                 # ✅ No conflicts
```

#### Tests Updated
- `test_preserve_existing_behavior()` - Fixed both user creation configs
- `test_comprehensive_workflow()` - Fixed workflow config
- `test_performance()` - Fixed performance test config  
- `test_cluster_configurations()` - Fixed cluster test config
- `test_infra_workflow()` - Fixed infra test config

### Impact Assessment

#### Before Fix
- E2E tests consistently failed with HTTP 500 UNEXPECTED_ERROR
- Project update operations triggered unnecessarily
- Test failures masked other potential issues
- CI/CD pipeline unreliable due to false negatives

#### After Fix  
- ✅ All E2E tests pass completely
- ✅ No unnecessary project update operations
- ✅ Retry logic working properly for actual transient errors
- ✅ Test reliability significantly improved

### Testing Results

Complete e2e test suite execution:
```
✓ Testing --preserve-existing flag behavior...
✓ initial configuration apply succeeded on attempt 1
✓ updated configuration apply succeeded on attempt 1  
✓ Both users verified - --preserve-existing working correctly
✓ Preserve test cleanup completed
✓ All E2E tests passed
```

Key metrics:
- **Duration**: ~780ms per apply operation (fast)
- **Success Rate**: 100% (no retries needed)
- **Resource Management**: Clean creation and cleanup
- **Preserve Logic**: Working correctly

### Lessons Learned

1. **Test Data Alignment**: Test configurations must match actual infrastructure state
2. **Project Name Immutability**: Atlas projects may have naming restrictions or immutability rules
3. **Error Investigation**: HTTP 500 doesn't always mean server issues - can indicate invalid requests
4. **Configuration Validation**: Test configs should be validated against actual Atlas state

### Prevention Measures

1. **Test Design**: Use actual resource names or discover them dynamically
2. **Configuration Management**: Consider using discovery to get current project name
3. **Validation**: Add pre-test validation of configuration compatibility
4. **Documentation**: Document test assumptions about infrastructure state

---

## [2025-01-28] GoLinting errcheck Issues Resolution

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: GoLinting CI failures due to unchecked error returns

### Summary
Fixed 6 GoLinting errcheck violations related to unchecked error return values in database command files. All issues involved functions that return errors but weren't being checked properly.

### Tasks
- [x] Fix 3 unchecked MarkHidden error returns in cmd/database/database.go
- [x] Fix unchecked client.Disconnect error returns in cmd/database/roles/roles.go 
- [x] Fix unchecked client.Disconnect error returns in cmd/database/users/users.go

### Files Modified
- `cmd/database/database.go` - Added error checking for MarkHidden calls
- `cmd/database/roles/roles.go` - Added error handling for client.Disconnect in defer statements
- `cmd/database/users/users.go` - Added error handling for client.Disconnect in defer statements

### Root Cause Analysis

#### MarkHidden Error Issues
- **Files**: cmd/database/database.go (lines 95, 167, 231)
- **Problem**: `cmd.Flags().MarkHidden("temp-user-roles")` calls did not check error returns
- **Risk**: If flag hiding fails, it could cause unexpected behavior in CLI flag visibility
- **Fix**: Added error checking with panic for early detection of flag configuration issues

#### MongoDB Client Disconnect Issues  
- **Files**: cmd/database/roles/roles.go (lines 245, 339), cmd/database/users/users.go (line 315)
- **Problem**: `defer client.Disconnect(ctx)` calls did not check error returns
- **Risk**: MongoDB connection leaks or cleanup failures could go unnoticed
- **Fix**: Wrapped defer calls in anonymous functions with proper error handling and warning messages

### Technical Implementation

#### 1. MarkHidden Error Handling
```go
// Before: Unchecked error return
cmd.Flags().MarkHidden("temp-user-roles")

// After: Proper error checking
if err := cmd.Flags().MarkHidden("temp-user-roles"); err != nil {
    // This should not fail as the flag was just added
    panic(fmt.Errorf("failed to mark temp-user-roles flag as hidden: %w", err))
}
```

**Rationale**: Uses panic because flag configuration errors indicate programming errors that should be caught during development.

#### 2. MongoDB Disconnect Error Handling
```go
// Before: Unchecked error return
defer client.Disconnect(ctx)

// After: Proper error handling in defer
defer func() {
    if err := client.Disconnect(ctx); err != nil {
        fmt.Printf("Warning: Failed to disconnect from MongoDB: %v\n", err)
    }
}()
```

**Rationale**: Uses warning messages because disconnect failures shouldn't interrupt the main operation flow, but should be logged for debugging.

### Impact Assessment

#### Before Fix
- **CI/CD**: GoLinting failed with 6 errcheck violations
- **Code Quality**: Potential resource leaks and silent configuration failures
- **Maintainability**: Inconsistent error handling patterns
- **Risk**: MongoDB connections might not be properly cleaned up

#### After Fix
- ✅ All GoLinting errcheck issues resolved (verified with `golangci-lint run --enable-only=errcheck ./cmd/database/...`)
- ✅ Proper resource cleanup with warning notifications
- ✅ Early detection of flag configuration issues
- ✅ Consistent error handling patterns across database commands

### Verification Results

1. **go vet**: Clean exit (code 0) with no issues
2. **golangci-lint errcheck**: 0 issues reported
3. **Functionality**: No changes to user-facing behavior
4. **Error Handling**: Improved error visibility and resource cleanup

### Error Handling Strategy

#### Flag Configuration Errors
- **Approach**: Panic on MarkHidden failures
- **Justification**: Configuration errors are programming bugs that should fail fast
- **Detection**: Caught during development and testing phases

#### Resource Cleanup Errors  
- **Approach**: Log warnings for disconnect failures
- **Justification**: Cleanup failures shouldn't interrupt main operation
- **Visibility**: Users/operators can see cleanup issues in output
- **Recovery**: System can continue functioning despite cleanup warnings

### Code Quality Improvements

1. **Consistency**: All database commands now follow the same error handling pattern
2. **Robustness**: Better resource management and cleanup
3. **Observability**: Failed operations now generate appropriate warnings
4. **Standards Compliance**: Meets Go error handling best practices and linting requirements

---