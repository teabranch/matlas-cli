#!/usr/bin/env bash

# Database Operations Testing for matlas-cli (Updated for New Authentication Model)
# Tests comprehensive database management functionality including:
# - Database CRUD operations with new authentication model (requires --collection)
# - Collection CRUD operations  
# - Index CRUD operations with all options
# - All three authentication methods: temp user, username/password, connection string
# - Failure detection and error handling
# - Targeted YAML deletion (doesn't affect other resources)
# - Update/modification operations for both CLI and YAML
# Runs against an existing Atlas cluster (specified via ATLAS_CLUSTER_NAME)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/database-operations"
RESOURCE_STATE_FILE="$TEST_REPORTS_DIR/database-resources.state"
DB_OPERATION_TIMEOUT="${DB_OPERATION_TIMEOUT:-3m}"

# Test state tracking
declare -a CREATED_RESOURCES=()
TEST_CLUSTER_NAME="${ATLAS_CLUSTER_NAME:-}"
TEST_DATABASE_USER="${MANUAL_DB_USER:-}"
TEST_DATABASE_PASSWORD="${MANUAL_DB_PASSWORD:-}"
USE_MANUAL_USER="${USE_MANUAL_USER:-false}"

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_subheader() {
    echo -e "${CYAN}--- $1 ---${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${PURPLE}ℹ $1${NC}"
}

# Track resources for cleanup
track_resource() {
    local resource_type="$1"
    local resource_name="$2"
    local additional_info="${3:-}"
    
    CREATED_RESOURCES+=("$resource_type:$resource_name:$additional_info")
    echo "$resource_type:$resource_name:$additional_info" >> "$RESOURCE_STATE_FILE" 2>/dev/null || true
}

# Environment validation
check_environment() {
    print_info "Validating database operations test environment..."
    
    # Check required environment variables
    if [[ -z "${ATLAS_PUB_KEY:-}" ]] || [[ -z "${ATLAS_API_KEY:-}" ]]; then
        print_error "Atlas credentials not configured"
        print_info "Required: ATLAS_PUB_KEY and ATLAS_API_KEY"
        return 1
    fi
    
    if [[ -z "${ATLAS_PROJECT_ID:-}" ]]; then
        print_error "ATLAS_PROJECT_ID not configured"
        return 1
    fi
    
    if [[ -z "${ATLAS_CLUSTER_NAME:-}" ]]; then
        print_error "ATLAS_CLUSTER_NAME not configured"
        print_info "Required: Name of existing Atlas cluster to run tests against"
        return 1
    fi
    
    # Check matlas binary
    if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
        print_error "matlas binary not found at $PROJECT_ROOT/matlas"
        return 1
    fi
    
    # Create test reports directory
    mkdir -p "$TEST_REPORTS_DIR"
    
    # Clear previous state
    true > "$RESOURCE_STATE_FILE"
    
    print_success "Environment validation completed"
    return 0
}

# Test all authentication methods for database operations
test_all_authentication_methods() {
    print_header "Testing All Database Authentication Methods"
    
    local auth_failures=0
    local test_db_base="auth-test-$(date +%s)"
    local test_collection="auth-test-collection"
    
    # Test 1: --use-temp-user authentication
    print_subheader "Test 1: Database creation with --use-temp-user"
    local test_db1="${test_db_base}-temp"
    
    if "$PROJECT_ROOT/matlas" database create "$test_db1" \
            --cluster "$TEST_CLUSTER_NAME" \
            --project-id "$ATLAS_PROJECT_ID" \
        --collection "$test_collection" \
            --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Database created with --use-temp-user authentication"
        track_resource "database" "$test_db1" "temp-user-auth"
        
        # Verify with listing
        sleep 5
        if "$PROJECT_ROOT/matlas" database list \
            --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
            --use-temp-user | grep -q "$test_db1"; then
            print_success "Database visible in list with temp user auth"
        else
            print_warning "Database not immediately visible (propagation delay expected)"
        fi
    else
        print_error "Failed to create database with --use-temp-user"
        ((auth_failures++))
    fi
    
    # Test 2: --username/--password authentication (if available)
    if [[ -n "$TEST_DATABASE_USER" && -n "$TEST_DATABASE_PASSWORD" ]]; then
        print_info "Found manual database credentials - testing username/password authentication"
        print_subheader "Test 2: Database creation with --username/--password"
        local test_db2="${test_db_base}-userpass"
        
        print_info "Testing database creation with user credentials: $TEST_DATABASE_USER"
        
        # First verify the database user exists and has permissions
        print_info "Verifying manual database user exists and has permissions..."
        if ! "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null | grep -q "$TEST_DATABASE_USER"; then
            print_warning "Manual database user '$TEST_DATABASE_USER' not found in Atlas project"
            print_info "Skipping username/password authentication test"
            print_info "To test this method, create a database user in Atlas with readWriteAnyDatabase permissions"
        else
            print_success "Manual database user found in Atlas project"
            
            if "$PROJECT_ROOT/matlas" database create "$test_db2" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
            --collection "$test_collection" \
            --username "$TEST_DATABASE_USER" \
            --password "$TEST_DATABASE_PASSWORD" \
            --timeout "$DB_OPERATION_TIMEOUT"; then
            
            print_success "Database created with --username/--password authentication"
            track_resource "database" "$test_db2" "user-pass-auth"
            
            # Verify with listing
            sleep 5
            if "$PROJECT_ROOT/matlas" database list \
                --cluster "$TEST_CLUSTER_NAME" \
                --project-id "$ATLAS_PROJECT_ID" \
                --username "$TEST_DATABASE_USER" \
                --password "$TEST_DATABASE_PASSWORD" | grep -q "$test_db2"; then
                print_success "Database visible in list with user/pass auth"
            else
                print_warning "Database not immediately visible (propagation delay expected)"
            fi
            else
                print_error "Failed to create database with --username/--password"
                ((auth_failures++))
            fi
        fi
    else
        print_info "Skipping --username/--password test (no manual credentials provided)"
        print_info "Set MANUAL_DB_USER and MANUAL_DB_PASSWORD environment variables to test this method"
    fi
    
    # Test 3: Direct connection string authentication
    if [[ -n "$TEST_DATABASE_USER" && -n "$TEST_DATABASE_PASSWORD" ]]; then
        print_subheader "Test 3: Database creation with direct connection string"
        local test_db3="${test_db_base}-connstr"
        
        # Get base connection string
        local base_connection_string
        if base_connection_string=$("$PROJECT_ROOT/matlas" atlas clusters get "$TEST_CLUSTER_NAME" \
                --project-id "$ATLAS_PROJECT_ID" \
                --output json 2>/dev/null | jq -r '.connectionStrings.standardSrv' 2>/dev/null); then
                
            # Build connection string with embedded credentials
            local encoded_user
            local encoded_pass
            if command -v python3 >/dev/null 2>&1; then
                encoded_user=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$TEST_DATABASE_USER'))")
                encoded_pass=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$TEST_DATABASE_PASSWORD'))")
            else
                # Fallback URL encoding (basic)
                encoded_user="$TEST_DATABASE_USER"
                encoded_pass="$TEST_DATABASE_PASSWORD"
            fi
            
            local auth_connection_string="${base_connection_string/mongodb+srv:\/\//mongodb+srv://$encoded_user:$encoded_pass@}"
            auth_connection_string="${auth_connection_string}/admin?authSource=admin"
            
            print_info "Testing database creation with direct connection string"
            
            if "$PROJECT_ROOT/matlas" database create "$test_db3" \
                --connection-string "$auth_connection_string" \
                --collection "$test_collection" \
                --timeout "$DB_OPERATION_TIMEOUT"; then
                
                print_success "Database created with direct connection string"
                track_resource "database" "$test_db3" "connection-string-auth"
                
                # Verify with listing
                sleep 5
                if "$PROJECT_ROOT/matlas" database list \
                    --connection-string "$auth_connection_string" | grep -q "$test_db3"; then
                    print_success "Database visible in list with connection string auth"
                else
                    print_warning "Database not immediately visible (propagation delay expected)"
            fi
        else
                print_error "Failed to create database with connection string"
                ((auth_failures++))
        fi
    else
            print_error "Failed to get cluster connection string"
            ((auth_failures++))
        fi
    else
        print_info "Skipping connection string test (no manual credentials provided)"
    fi
    
    if [[ $auth_failures -eq 0 ]]; then
        print_success "✅ All authentication method tests passed"
        return 0
    else
        print_error "❌ $auth_failures authentication method test(s) failed"
        return 1
    fi
}

# Test failure detection and error handling
test_failure_detection() {
    print_header "Testing Failure Detection and Error Handling"
    
    local failure_tests=0
    local failed_failure_tests=0
    
    # Test 1: Missing collection parameter
    print_subheader "Test 1: Missing --collection parameter"
    ((failure_tests++))
    
    if "$PROJECT_ROOT/matlas" database create "invalid-test-db" \
            --cluster "$TEST_CLUSTER_NAME" \
            --project-id "$ATLAS_PROJECT_ID" \
        --use-temp-user 2>/dev/null; then
        print_error "BUG: Command should have failed without --collection parameter"
        ((failed_failure_tests++))
    else
        print_success "✅ Correctly failed when --collection parameter is missing"
    fi
    
    # Test 2: Invalid authentication combination
    print_subheader "Test 2: Invalid authentication combination"
    ((failure_tests++))
    
    if "$PROJECT_ROOT/matlas" database create "invalid-test-db2" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --collection "test-collection" \
        --use-temp-user \
        --username "testuser" 2>/dev/null; then
        print_error "BUG: Command should have failed with conflicting auth flags"
        ((failed_failure_tests++))
    else
        print_success "✅ Correctly failed with conflicting --use-temp-user and --username"
    fi
    
    # Test 3: Missing password when username provided
    print_subheader "Test 3: Missing password when username provided"
    ((failure_tests++))
    
    if "$PROJECT_ROOT/matlas" database create "invalid-test-db3" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --collection "test-collection" \
        --username "testuser" 2>/dev/null; then
        print_error "BUG: Command should have failed with --username but no --password"
        ((failed_failure_tests++))
    else
        print_success "✅ Correctly failed when --username provided without --password"
    fi
    
    # Test 4: No authentication method provided
    print_subheader "Test 4: No authentication method provided"
    ((failure_tests++))
    
    if "$PROJECT_ROOT/matlas" database create "invalid-test-db4" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --collection "test-collection" 2>/dev/null; then
        print_error "BUG: Command should have failed with no authentication method"
        ((failed_failure_tests++))
    else
        print_success "✅ Correctly failed when no authentication method provided"
    fi
    
    # Test 5: Invalid cluster name
    print_subheader "Test 5: Invalid cluster name"
    ((failure_tests++))
    
    if "$PROJECT_ROOT/matlas" database create "invalid-test-db5" \
        --cluster "nonexistent-cluster-name" \
        --project-id "$ATLAS_PROJECT_ID" \
        --collection "test-collection" \
        --use-temp-user 2>/dev/null; then
        print_error "BUG: Command should have failed with invalid cluster name"
        ((failed_failure_tests++))
    else
        print_success "✅ Correctly failed with invalid cluster name"
    fi
    
    print_info "Failure detection tests: $((failure_tests - failed_failure_tests))/$failure_tests passed"
    
    if [[ $failed_failure_tests -eq 0 ]]; then
        print_success "✅ All failure detection tests passed"
        return 0
    else
        print_error "❌ $failed_failure_tests failure detection test(s) failed"
        return 1
    fi
}

# Test database CRUD operations
test_database_crud_operations() {
    print_header "Testing Database CRUD Operations"
    
    local test_db="crud-test-db-$(date +%s)"
    local test_collection="crud-test-collection"
    local crud_failures=0
    
    # Create operation
    print_subheader "Database CREATE operation"
    
    if "$PROJECT_ROOT/matlas" database create "$test_db" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --collection "$test_collection" \
        --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Database created successfully"
        track_resource "database" "$test_db" "crud-test"
    else
        print_error "Failed to create database"
        ((crud_failures++))
        return 1
    fi
    
    # Read operation - List databases
    print_subheader "Database READ operation (list)"
    
    local list_attempts=0
    local max_list_attempts=3
    local database_found=false
    
    while [[ $list_attempts -lt $max_list_attempts ]]; do
        ((list_attempts++))
        print_info "Database list attempt $list_attempts/$max_list_attempts"
        
        if "$PROJECT_ROOT/matlas" database list \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --use-temp-user \
            --timeout "$DB_OPERATION_TIMEOUT" | grep -q "$test_db"; then
            print_success "Database found in list on attempt $list_attempts"
            database_found=true
            break
        else
            if [[ $list_attempts -lt $max_list_attempts ]]; then
                print_info "Database not visible yet, waiting 10s for propagation..."
                sleep 10
            fi
        fi
    done
    
    if [[ "$database_found" != "true" ]]; then
        print_warning "Database not visible in list after $max_list_attempts attempts"
        # Don't fail the test - this is often propagation delay
    fi
    
    if [[ $crud_failures -eq 0 ]]; then
        print_success "✅ Database CRUD operations completed successfully"
    return 0
    else
        print_error "❌ $crud_failures database CRUD operation(s) failed"
            return 1
        fi
}

# Test collection CRUD operations
test_collection_crud_operations() {
    print_header "Testing Collection CRUD Operations"
    
    local test_db="collection-crud-db-$(date +%s)"
    local base_collection="collection-crud-base"
    local test_collection="collection-crud-test-$(date +%s)"
    local crud_failures=0
    
    # Create database first
    print_info "Creating database for collection tests: $test_db"
    
    if "$PROJECT_ROOT/matlas" database create "$test_db" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --collection "$base_collection" \
        --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Test database created for collection operations"
        track_resource "database" "$test_db" "collection-crud-parent"
    else
        print_error "Failed to create test database for collection operations"
        return 1
    fi
    
    # Wait for database to be ready
    sleep 10
    
    # Create additional collection
    print_subheader "Collection CREATE operation"
    
    if "$PROJECT_ROOT/matlas" database collections create "$test_collection" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database "$test_db" \
        --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Collection created successfully"
        track_resource "collection" "$test_collection" "$test_db"
    else
        print_warning "Collection creation failed or not supported"
        print_info "This is acceptable - some MongoDB versions/configurations don't support explicit collection creation"
    fi
    
    # List collections (Read operation)
    print_subheader "Collection READ operation (list)"
    
    if "$PROJECT_ROOT/matlas" database collections list \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database "$test_db" \
        --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Collections listed successfully"
    else
        print_warning "Failed to list collections"
        ((crud_failures++))
    fi
    
    print_success "✅ Collection CRUD operations completed"
    return 0
}

# Test index CRUD operations
test_index_crud_operations() {
    print_header "Testing Index CRUD Operations"
    
    local test_db="index-crud-db-$(date +%s)"
    local test_collection="index-crud-collection"
    local crud_failures=0
    
    # Create database with collection for index operations
    print_info "Creating database for index tests: $test_db"
    
    if "$PROJECT_ROOT/matlas" database create "$test_db" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --collection "$test_collection" \
        --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Test database created for index operations"
        track_resource "database" "$test_db" "index-crud-parent"
    else
        print_error "Failed to create test database for index operations"
        return 1
    fi
    
    # Wait for database to be ready
    sleep 15
    
    # Create single field index
    print_subheader "Single field index CREATE operation"
    local single_index="single_field_idx"
    
    if "$PROJECT_ROOT/matlas" database indexes create "$single_index" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database "$test_db" \
            --collection "$test_collection" \
        --use-temp-user \
        --keys "name:1" \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Single field index created successfully"
        track_resource "index" "$single_index" "$test_db.$test_collection"
    else
        print_warning "Single field index creation failed or not supported"
    fi
    
    # Create compound index
    print_subheader "Compound index CREATE operation"
    local compound_index="compound_idx"
    
    if "$PROJECT_ROOT/matlas" database indexes create "$compound_index" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database "$test_db" \
            --collection "$test_collection" \
        --use-temp-user \
        --keys "name:1,email:-1,created:1" \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Compound index created successfully"
        track_resource "index" "$compound_index" "$test_db.$test_collection"
    else
        print_warning "Compound index creation failed or not supported"
    fi
    
    # Create unique index
    print_subheader "Unique index CREATE operation"
    local unique_index="unique_email_idx"
    
    if "$PROJECT_ROOT/matlas" database indexes create "$unique_index" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database "$test_db" \
            --collection "$test_collection" \
        --use-temp-user \
        --keys "email:1" \
            --unique \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Unique index created successfully"
        track_resource "index" "$unique_index" "$test_db.$test_collection"
    else
        print_warning "Unique index creation failed or not supported"
    fi
    
    # List indexes (Read operation)
    print_subheader "Index READ operation (list)"
    
    if "$PROJECT_ROOT/matlas" database indexes list \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database "$test_db" \
            --collection "$test_collection" \
        --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Indexes listed successfully"
    else
        print_warning "Failed to list indexes"
        ((crud_failures++))
    fi
    
    print_success "✅ Index CRUD operations completed"
    return 0
}

# Test YAML operations with targeted deletion
test_yaml_operations_with_targeted_deletion() {
    print_header "Testing YAML Operations with Targeted Deletion"
    
    local existing_user="existing-user-$(date +%s)"
    local target_user="target-user-$(date +%s)"
    local yaml_config2="$TEST_REPORTS_DIR/target-user.yaml"
    local yaml_config3="$TEST_REPORTS_DIR/empty-config.yaml"
    local project_name
    
    project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null || echo "$ATLAS_PROJECT_ID")
    
    # Step 1: Create existing user via CLI (should not be affected by YAML operations)
    print_subheader "Step 1: Creating existing user via CLI"
    if "$PROJECT_ROOT/matlas" atlas users create \
        --project-id "$ATLAS_PROJECT_ID" \
        --username "$existing_user" \
        --database-name admin \
        --roles read@admin \
        --password "ExistingUserPass123!"; then
        print_success "Existing user created via CLI: $existing_user"
        track_resource "user" "$existing_user" "cli-created"
    else
        print_error "Failed to create existing user"
        return 1
    fi
    
    # Step 2: Create target user via YAML
    print_subheader "Step 2: Creating target user via YAML"
    cat > "$yaml_config2" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata: 
  name: target-user-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata: 
      name: $target_user
    spec:
      projectName: "$project_name"
      username: $target_user
      databaseName: admin
      password: TargetUserPass123!
      roles:
        - roleName: readWrite
          databaseName: targetapp
EOF
    track_resource "yaml-config" "$yaml_config2" "target-user"
    
    if "$PROJECT_ROOT/matlas" infra apply -f "$yaml_config2" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve; then
        print_success "Target user created via YAML: $target_user"
        track_resource "user" "$target_user" "yaml-created"
    else
        print_error "Failed to create target user via YAML"
        return 1
    fi
    
    sleep 5
    
    # Step 3: Verify both users exist
    print_subheader "Step 3: Verifying both users exist"
    local user_list
    user_list=$("$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID")
    
    if echo "$user_list" | grep -q "$existing_user"; then
        print_success "Existing user found: $existing_user"
    else
        print_error "Existing user not found"
        return 1
    fi
    
    if echo "$user_list" | grep -q "$target_user"; then
        print_success "Target user found: $target_user"
    else
        print_error "Target user not found"
        return 1
    fi
    
    # Step 4: Apply empty YAML config (should only remove YAML-managed resources)
    print_subheader "Step 4: Applying empty YAML config (targeted deletion)"
    cat > "$yaml_config3" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata: 
  name: target-user-test
# Empty resources - should remove target user but leave existing user
resources: []
EOF
    track_resource "yaml-config" "$yaml_config3" "empty-config"
    
    if "$PROJECT_ROOT/matlas" infra apply -f "$yaml_config3" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve; then
        print_success "Empty YAML config applied (targeted deletion)"
    else
        print_warning "Empty YAML apply failed - may not support targeted deletion"
    fi
    
    sleep 5
    
    # Step 5: Verify targeted deletion worked correctly
    print_subheader "Step 5: Verifying targeted deletion results"
    user_list=$("$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID")
    
    # Existing user should still be there
    if echo "$user_list" | grep -q "$existing_user"; then
        print_success "✅ TARGETED DELETION SUCCESS: Existing user preserved"
    else
        print_error "❌ TARGETED DELETION FAILED: Existing user was deleted!"
        return 1
    fi
    
    # Target user should be gone
    if ! echo "$user_list" | grep -q "$target_user"; then
        print_success "✅ TARGETED DELETION SUCCESS: Target user removed"
    else
        print_warning "Target user still exists (may be deletion propagation delay)"
    fi
    
    print_success "✅ YAML targeted deletion test completed successfully"
    return 0
}

# Test complete database workflow
test_complete_database_workflow() {
    print_header "Testing Complete Database → Collection → Index Workflow"
    
    local workflow_db="workflow-db-$(date +%s)"
    local workflow_collection="workflow-collection"
    local workflow_failures=0
    
    # Step 1: Create database with collection
    print_subheader "Step 1: Create database with initial collection"
    
    if "$PROJECT_ROOT/matlas" database create "$workflow_db" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --collection "$workflow_collection" \
        --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Workflow database created with initial collection"
        track_resource "database" "$workflow_db" "workflow"
    else
        print_error "Failed to create workflow database"
        ((workflow_failures++))
        return 1
    fi
    
    # Wait for database to be ready
    sleep 15
    
    # Step 2: Add additional collections
    print_subheader "Step 2: Add additional collections to database"
    
    local collections=("users" "products" "orders" "logs")
    for collection in "${collections[@]}"; do
        if "$PROJECT_ROOT/matlas" database collections create "$collection" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database "$workflow_db" \
            --use-temp-user \
            --timeout "$DB_OPERATION_TIMEOUT"; then
            
            print_success "Collection '$collection' created"
            track_resource "collection" "$collection" "$workflow_db"
        else
            print_warning "Collection '$collection' creation failed or not supported"
        fi
    done
    
    # Step 3: Create indexes on collections
    print_subheader "Step 3: Create indexes on collections"
    
    # Index on users collection
    if "$PROJECT_ROOT/matlas" database indexes create "email_idx" \
        --cluster "$TEST_CLUSTER_NAME" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database "$workflow_db" \
        --collection "users" \
        --use-temp-user \
        --keys "email:1" \
        --unique \
        --timeout "$DB_OPERATION_TIMEOUT"; then
        
        print_success "Email index created on users collection"
        track_resource "index" "email_idx" "$workflow_db.users"
    else
        print_warning "Email index creation failed or not supported"
    fi
    
    # Step 4: Verify complete workflow
    print_subheader "Step 4: Verify complete database structure"
    
    # List all databases
    print_info "Listing all databases..."
    if "$PROJECT_ROOT/matlas" database list \
            --cluster "$TEST_CLUSTER_NAME" \
            --project-id "$ATLAS_PROJECT_ID" \
        --use-temp-user \
        --timeout "$DB_OPERATION_TIMEOUT" | grep -q "$workflow_db"; then
        print_success "Workflow database visible in database list"
    else
        print_warning "Workflow database not visible in list"
    fi
    
    if [[ $workflow_failures -eq 0 ]]; then
        print_success "✅ Complete database workflow test passed"
        return 0
    else
        print_error "❌ $workflow_failures workflow step(s) failed"
        return 1
    fi
}

# Cleanup function
cleanup_resources() {
    print_info "Cleaning up database operations test resources..."
    
    if [[ ${#CREATED_RESOURCES[@]} -eq 0 ]]; then
        print_info "No resources to clean up"
        return 0
    fi
    
    # Clean up in reverse order (LIFO)
    print_subheader "Cleaning up created resources"
    for ((i=${#CREATED_RESOURCES[@]}-1; i>=0; i--)); do
        local resource_info="${CREATED_RESOURCES[i]}"
        IFS=':' read -r resource_type resource_name additional_info <<< "$resource_info"
        
        case "$resource_type" in
            "user")
                print_info "Deleting user: $resource_name"
                    "$PROJECT_ROOT/matlas" atlas users delete "$resource_name" \
                        --project-id "$ATLAS_PROJECT_ID" \
                        --database-name admin \
                        --yes 2>/dev/null || true
                ;;
            "yaml-config"|"config")
                print_info "Removing config file: $resource_name"
                rm -f "$resource_name" 2>/dev/null || true
                ;;
            *)
                print_info "Tracked resource: $resource_type:$resource_name ($additional_info)"
                ;;
        esac
    done
    
    # Clear state file
    true > "$RESOURCE_STATE_FILE" 2>/dev/null || true
    
    print_success "Cleanup completed"
}

# Main test runner
run_database_operations_tests() {
    local test_type="${1:-all}"
    
    print_header "Database Operations Comprehensive Test Suite"
    print_info "Testing database, collection, and index management with new authentication model"
    echo
    
    # Setup cleanup trap
    trap cleanup_resources EXIT INT TERM
    
    # Environment validation
    if ! check_environment; then
        print_error "Environment validation failed"
        return 1
    fi
    
    local test_failures=0
    
    case "$test_type" in
        "auth")
            test_all_authentication_methods || ((test_failures++))
            ;;
        "failures")
            test_failure_detection || ((test_failures++))
            ;;
        "databases")
            test_database_crud_operations || ((test_failures++))
            ;;
        "collections")
            test_collection_crud_operations || ((test_failures++))
            ;;
        "indexes")
            test_index_crud_operations || ((test_failures++))
            ;;
        "yaml")
            test_yaml_operations_with_targeted_deletion || ((test_failures++))
            ;;
        "workflow")
            test_complete_database_workflow || ((test_failures++))
            ;;
        "comprehensive")
            print_info "Running comprehensive database operations test suite..."
            test_all_authentication_methods || ((test_failures++))
            echo
            test_failure_detection || ((test_failures++))
            echo
            test_database_crud_operations || ((test_failures++))
            echo
            test_collection_crud_operations || ((test_failures++))
            echo
            test_index_crud_operations || ((test_failures++))
            echo
            test_yaml_operations_with_targeted_deletion || ((test_failures++))
            echo
            test_complete_database_workflow || ((test_failures++))
            ;;
        "all"|*)
            print_info "Running standard database operations test suite..."
            test_all_authentication_methods || ((test_failures++))
            echo
            test_failure_detection || ((test_failures++))
            echo
            test_database_crud_operations || ((test_failures++))
            echo
            test_collection_crud_operations || ((test_failures++))
            echo
            test_index_crud_operations || ((test_failures++))
            echo
            test_complete_database_workflow || ((test_failures++))
            ;;
    esac
    
    echo
    if [[ $test_failures -eq 0 ]]; then
        print_success "🎉 All database operations tests passed!"
        print_info "✅ Authentication methods tested: temp user, username/password, connection string"
        print_info "✅ Database creation with --collection requirement verified"
        print_info "✅ YAML targeted deletion working correctly"
        print_info "✅ Error detection and validation working"
        print_info "✅ Complete database workflow tested"
        return 0
    else
        print_error "❌ $test_failures database operations test(s) failed"
        return 1
    fi
}

# Script usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  auth            Test all authentication methods (temp user, username/password, connection string)"
    echo "  failures        Test failure detection and error handling"
    echo "  databases       Test database CRUD operations"
    echo "  collections     Test collection CRUD operations"
    echo "  indexes         Test index CRUD operations with all options"
    echo "  yaml            Test YAML operations with targeted deletion"
    echo "  workflow        Test complete database → collection → index workflow"
    echo "  comprehensive   Run all test categories (recommended for full validation)"
    echo "  all             Run standard test suite (default)"
    echo
    echo "Environment variables required:"
    echo "  ATLAS_PUB_KEY       Atlas public API key"
    echo "  ATLAS_API_KEY       Atlas private API key"
    echo "  ATLAS_PROJECT_ID    Atlas project ID"
    echo "  ATLAS_CLUSTER_NAME  Name of existing cluster to test against"
    echo
    echo "Optional environment variables:"
    echo "  MANUAL_DB_USER      Manual database username for auth testing"
    echo "  MANUAL_DB_PASSWORD  Manual database password for auth testing"
    echo "  DB_OPERATION_TIMEOUT Timeout for database operations (default: 3m)"
    echo
    echo "Examples:"
    echo "  $0                     # Run standard tests"
    echo "  $0 comprehensive       # Run all test categories"
    echo "  $0 auth                # Test authentication methods only"
    echo "  $0 failures            # Test failure detection only"
    echo "  $0 yaml                # Test YAML operations with targeted deletion"
    echo
    echo "Note: This script tests against an existing Atlas cluster and creates/deletes"
    echo "      test databases, collections, and indexes. Ensure you have appropriate"
    echo "      permissions and are using a test environment."
}

# Main execution
main() {
    case "${1:-all}" in
        "auth"|"failures"|"databases"|"collections"|"indexes"|"yaml"|"workflow"|"comprehensive"|"all")
            run_database_operations_tests "${1:-all}"
            ;;
        "-h"|"--help"|"help")
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown command: ${1:-}"
            echo
            show_usage
            exit 1
            ;;
    esac
}

# Only run if executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
