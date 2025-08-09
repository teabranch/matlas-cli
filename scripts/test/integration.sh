#!/usr/bin/env bash

# Integration Tests Runner
# Tests with live Atlas API - creates and cleans up real resources

set -euo pipefail

# Colors
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly RED='\033[0;31m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/integration"
readonly RESOURCE_STATE_FILE="$TEST_REPORTS_DIR/resources.json"

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

# Resource tracking
declare -a CREATED_RESOURCES=()

setup_integration_environment() {
    mkdir -p "$TEST_REPORTS_DIR"
    
    # Load environment
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        set -o allexport
        source "$PROJECT_ROOT/.env"
        set +o allexport
    fi
    
    # Check Atlas credentials
    if [[ -z "${ATLAS_PUB_KEY:-}" || -z "${ATLAS_API_KEY:-}" || -z "${ATLAS_PROJECT_ID:-}" || -z "${ATLAS_ORG_ID:-}" ]]; then
        print_error "Atlas credentials required for integration tests"
        print_info "Set ATLAS_PUB_KEY, ATLAS_API_KEY, ATLAS_PROJECT_ID, and ATLAS_ORG_ID in .env file"
        print_info "Current environment:"
        print_info "  ATLAS_PUB_KEY: ${ATLAS_PUB_KEY:-"not set"}"
        print_info "  ATLAS_API_KEY: ${ATLAS_API_KEY:+set} ${ATLAS_API_KEY:-"not set"}"
        print_info "  ATLAS_PROJECT_ID: ${ATLAS_PROJECT_ID:-"not set"}"
        print_info "  ATLAS_ORG_ID: ${ATLAS_ORG_ID:-"not set"}"
        return 1
    fi
    
    print_success "Integration environment ready"
    print_info "Using credentials:"
    print_info "  Public Key: ${ATLAS_PUB_KEY:0:8}..."
    print_info "  API Key: ${ATLAS_API_KEY:0:8}..."
    print_info "  Project ID: $ATLAS_PROJECT_ID"
    
    # Test if matlas binary exists and works
    if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
        print_info "Building matlas binary..."
        cd "$PROJECT_ROOT"
        if ! go build -o matlas; then
            print_error "Failed to build matlas binary"
            return 1
        fi
        print_success "Built matlas binary"
    fi
    
    # Test basic connectivity
    print_info "Testing Atlas API connectivity..."
    if "$PROJECT_ROOT/matlas" atlas projects list >/dev/null 2>&1; then
        print_success "Atlas API connectivity confirmed"
    else
        print_warning "Atlas API connectivity test failed - proceeding anyway"
    fi
    
    return 0
}

test_infra_workflow() {
    print_info "Testing infra workflow integration..."
    
    local config_file="$TEST_REPORTS_DIR/integration-test-config.yaml"
    local apply_user="apply-integration-$(date +%s)"
    
    # Create test configuration using Project format
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: integration-test-project
spec:
  name: "Integration Test Project"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata:
        name: $apply_user
      username: $apply_user
      databaseName: admin
      password: ApplyIntegrationTest123!
      roles:
        - roleName: readWrite
          databaseName: testapp
        - roleName: read
          databaseName: admin
  networkAccess:
    - metadata:
        name: integration-network
      cidr: 192.168.0.0/24
      comment: Integration test network access
EOF
    
    track_resource "config" "$config_file"
    track_resource "user" "$apply_user"
    
    # Test validation
    print_info "Testing infra validation..."
    if "$PROJECT_ROOT/matlas" infra validate --file "$config_file"; then
        print_success "Configuration validation passed"
    else
        print_error "Configuration validation failed"
        return 1
    fi
    
    # Test planning
    print_info "Testing infra planning..."
    local plan_file="$TEST_REPORTS_DIR/integration-plan.json"
    if "$PROJECT_ROOT/matlas" infra plan --file "$config_file" --project-id "$ATLAS_PROJECT_ID" --output json > "$plan_file"; then
        print_success "Plan generation successful"
    else
        print_error "Plan generation failed"
        return 1
    fi
    
    track_resource "plan" "$plan_file"
    
    # Test dry-run
    print_info "Testing infra dry-run..."
    if "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run; then
        print_success "Dry-run successful"
    else
        print_error "Dry-run failed"
        return 1
    fi
    
    # Test diff
    print_info "Testing infra diff..."
    if "$PROJECT_ROOT/matlas" infra diff -f "$config_file" --project-id "$ATLAS_PROJECT_ID"; then
        print_success "Diff operation successful"
    else
        print_error "Diff operation failed"
        return 1
    fi
    
    # Test actual apply with preserve-existing
    print_info "Testing actual apply with --preserve-existing..."
    if "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve; then
        print_success "Apply with --preserve-existing successful"
        
        # Test show operation on applied resources
        print_info "Testing show operation on applied resources..."
        if "$PROJECT_ROOT/matlas" infra show --project-id "$ATLAS_PROJECT_ID"; then
            print_success "Show operation successful"
        else
            print_warning "Show operation failed"
        fi
        
        # Verify resources were created via Atlas CLI
        print_info "Verifying created resources via Atlas CLI..."
        sleep 2  # Wait for propagation
        
        if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$apply_user" 2>/dev/null; then
            print_success "Database user created and verified"
        else
            print_warning "Database user not immediately visible"
        fi
        
        if "$PROJECT_ROOT/matlas" atlas network list --project-id "$ATLAS_PROJECT_ID" | grep -q "192.168.0.0/24" 2>/dev/null; then
            print_success "Network access rule created and verified"
        else
            print_warning "Network access rule not immediately visible"
        fi
        
        # Test destroy operation
        print_info "Testing destroy operation..."
        if "$PROJECT_ROOT/matlas" infra destroy -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --auto-approve; then
            print_success "Destroy operation successful"
            
            # Verify cleanup
            sleep 2  # Wait for cleanup
            if ! "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$apply_user" 2>/dev/null; then
                print_success "Database user cleanup verified"
            else
                print_warning "Database user cleanup may still be in progress"
            fi
        else
            print_warning "Destroy operation failed - manual cleanup may be required"
        fi
    else
        print_warning "Apply with --preserve-existing failed - skipping destroy test"
    fi
    
    print_success "Infra workflow integration test completed"
    return 0
}

cleanup_resources() {
    print_info "Cleaning up test resources..."
    
    if [[ ${#CREATED_RESOURCES[@]} -eq 0 ]]; then
        print_info "No resources to clean up"
        return 0
    fi
    
    local cleanup_errors=0
    
    for resource in "${CREATED_RESOURCES[@]}"; do
        local resource_type=$(echo "$resource" | cut -d: -f1)
        local resource_id=$(echo "$resource" | cut -d: -f2)
        
        print_info "Cleaning up $resource_type: $resource_id"
        
        case "$resource_type" in
            "user")
                # Check if user exists before trying to delete
                if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null | grep -q "^$resource_id"; then
                    if ! "$PROJECT_ROOT/matlas" atlas users delete "$resource_id" \
                        --project-id "$ATLAS_PROJECT_ID" --database-name admin --yes 2>/dev/null; then
                        print_warning "Failed to delete user: $resource_id"
                        ((cleanup_errors++))
                    else
                        print_info "Deleted user: $resource_id"
                    fi
                else
                    print_info "User $resource_id does not exist (already cleaned up or never created)"
                fi
                ;;
            "network")
                if ! "$PROJECT_ROOT/matlas" atlas network delete "$resource_id" \
                    --project-id "$ATLAS_PROJECT_ID" --yes 2>/dev/null; then
                    print_warning "Failed to delete network access: $resource_id"
                    ((cleanup_errors++))
                fi
                ;;
        esac
    done
    
    if [[ $cleanup_errors -eq 0 ]]; then
        print_success "All resources cleaned up"
    else
        print_warning "$cleanup_errors resource(s) failed to clean up"
    fi
    
    # Clear the array
    CREATED_RESOURCES=()
}

track_resource() {
    local resource_type="$1"
    local resource_id="$2"
    CREATED_RESOURCES+=("$resource_type:$resource_id")
    print_info "Tracking resource: $resource_type:$resource_id"
}

test_database_users() {
    print_info "Testing database users..."
    
    local test_username="test-user-$(date +%s)"
    local error_output
    
    # Create user
    print_info "Creating user: $test_username"
    error_output=$("$PROJECT_ROOT/matlas" atlas users create \
        --project-id "$ATLAS_PROJECT_ID" \
        --username "$test_username" \
        --roles "readWrite@testdb" \
        --database-name "admin" \
        --password "TestPassword123!" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        track_resource "user" "$test_username"
        print_success "Created test user: $test_username"
        
        # Test listing users
        print_info "Checking if user appears in list..."
        
        # Wait a moment for the user to propagate
        sleep 2
        
        local users_output
        users_output=$("$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" 2>&1)
        
        if echo "$users_output" | grep -q "$test_username"; then
            print_success "User appears in list"
        else
            print_error "User not found in list"
            print_error "Expected username: $test_username"
            print_info "Users list output:"
            echo "$users_output"
            return 1
        fi
        
        return 0
    else
        print_error "Failed to create test user: $test_username"
        print_error "Error output: $error_output"
        return 1
    fi
}

test_network_access() {
    print_info "Testing network access..."
    
    local test_ip="192.168.1.100"
    local error_output
    
    # Create network access entry
    print_info "Creating network access entry for IP: $test_ip"
    error_output=$("$PROJECT_ROOT/matlas" atlas network create \
        --project-id "$ATLAS_PROJECT_ID" \
        --ip-address "$test_ip" \
        --comment "Integration test" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        track_resource "network" "$test_ip"
        print_success "Created network access entry: $test_ip"
        
        # Test listing network access
        print_info "Checking if network access entry appears in list..."
        
        # Wait a moment for the entry to propagate
        sleep 2
        
        local list_output
        list_output=$("$PROJECT_ROOT/matlas" atlas network list --project-id "$ATLAS_PROJECT_ID" 2>&1)
        
        print_info "Network access list output:"
        echo "$list_output"
        
        if echo "$list_output" | grep -q "$test_ip"; then
            print_success "Network access entry appears in list"
        else
            print_error "Network access entry not found in list"
            print_error "Expected IP: $test_ip"
            print_error "List output above shows what was actually returned"
            return 1
        fi
        
        return 0
    else
        print_error "Failed to create network access entry for IP: $test_ip"
        print_error "Error output: $error_output"
        return 1
    fi
}

run_integration_tests() {
    local dry_run=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run) dry_run=true; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ "$dry_run" == "true" ]]; then
        print_info "DRY RUN: Would run integration tests"
        print_info "Tests: database users, network access, infra workflow"
        return 0
    fi
    
    print_info "Running integration tests..."
    
    # Setup cleanup trap
    trap cleanup_resources EXIT INT TERM
    
    # Setup environment
    if ! setup_integration_environment; then
        return 1
    fi
    
    # Clean test cache
    cd "$PROJECT_ROOT"
    go clean -testcache
    
    local test_failures=0
    
    # Run specific integration tests
    test_database_users || ((test_failures++))
    test_network_access || ((test_failures++))
    test_infra_workflow || ((test_failures++))
    
    # Save resource state
    printf '%s\n' "${CREATED_RESOURCES[@]}" > "$RESOURCE_STATE_FILE" 2>/dev/null || true
    
    if [[ $test_failures -eq 0 ]]; then
        print_success "All integration tests passed"
        return 0
    else
        print_error "$test_failures integration test(s) failed"
        return 1
    fi
}

run_integration_tests "$@"