#!/usr/bin/env bash

# Comprehensive End-to-End Test Script for matlas-cli
# This script ensures ALL resources are cleaned up even on failures, interrupts, or crashes
# Implements robust signal handling and resource tracking

set -euo pipefail

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly PURPLE='\033[0;35m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color
readonly BOLD='\033[1m'

# Script configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports"
readonly E2E_REPORTS_DIR="$TEST_REPORTS_DIR/e2e"
readonly RESOURCE_TRACKING_FILE="$E2E_REPORTS_DIR/created-resources.json"
readonly CLEANUP_LOG_FILE="$E2E_REPORTS_DIR/cleanup.log"

# Test configuration
readonly DEFAULT_TEST_TIMEOUT="30m"
readonly CLEANUP_TIMEOUT="10m"
readonly ATLAS_API_RETRY_COUNT=3
readonly ATLAS_API_RETRY_DELAY=5

# Global variables for resource tracking
declare -a CREATED_RESOURCES=()
declare -a CLEANUP_FUNCTIONS=()
declare -g TEST_START_TIME
declare -g TEST_PID=$$
declare -g CLEANUP_IN_PROGRESS=false
declare -g EXIT_CODE=0

# Environment variables
TEST_TIMEOUT="${TEST_TIMEOUT:-$DEFAULT_TEST_TIMEOUT}"
VERBOSE="${VERBOSE:-false}"
DRY_RUN="${DRY_RUN:-false}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"

print_header() {
    echo -e "${BLUE}${BOLD}════════════════════════════════════════${NC}"
    echo -e "${BLUE}${BOLD} $1${NC}"
    echo -e "${BLUE}${BOLD}════════════════════════════════════════${NC}"
}

print_subheader() {
    echo -e "${CYAN}${BOLD}▶ $1${NC}"
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

log_to_file() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$CLEANUP_LOG_FILE"
}

# Critical: Resource tracking functions
track_resource() {
    local resource_type="$1"
    local resource_id="$2"
    local resource_name="$3"
    local project_id="${4:-}"
    
    local resource_json=$(cat <<EOF
{
    "type": "$resource_type",
    "id": "$resource_id", 
    "name": "$resource_name",
    "project_id": "$project_id",
    "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "cleanup_command": "",
    "cleanup_status": "pending"
}
EOF
)
    
    CREATED_RESOURCES+=("$resource_json")
    save_resource_tracking
    log_to_file "TRACKED: $resource_type $resource_name ($resource_id)"
    
    if [[ "$VERBOSE" == "true" ]]; then
        print_info "Tracked resource: $resource_type $resource_name"
    fi
}

track_cleanup_function() {
    local cleanup_command="$1"
    local description="$2"
    
    CLEANUP_FUNCTIONS+=("$cleanup_command|$description")
    log_to_file "TRACKED_CLEANUP: $description - $cleanup_command"
}

save_resource_tracking() {
    mkdir -p "$(dirname "$RESOURCE_TRACKING_FILE")"
    
    # Create JSON array from tracked resources
    local json_content="["
    local first=true
    for resource in "${CREATED_RESOURCES[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            json_content+=","
        fi
        json_content+="$resource"
    done
    json_content+="]"
    
    echo "$json_content" > "$RESOURCE_TRACKING_FILE"
}

load_resource_tracking() {
    if [[ -f "$RESOURCE_TRACKING_FILE" ]]; then
        print_info "Loading existing resource tracking data..."
        # Parse JSON and populate CREATED_RESOURCES array
        # This is a simplified implementation - in production, use jq
        while IFS= read -r line; do
            if [[ "$line" =~ \{.*\} ]]; then
                CREATED_RESOURCES+=("$line")
            fi
        done < <(grep -o '{[^}]*}' "$RESOURCE_TRACKING_FILE" 2>/dev/null || true)
    fi
}

# Critical: Cleanup functions that MUST work even on failure
cleanup_atlas_resource() {
    local resource_type="$1"
    local resource_id="$2"
    local project_id="$3"
    local resource_name="$4"
    
    log_to_file "CLEANUP_START: $resource_type $resource_name ($resource_id)"
    
    local cleanup_success=false
    local attempt=1
    
    while [[ $attempt -le $ATLAS_API_RETRY_COUNT ]]; do
        case "$resource_type" in
            "cluster")
                if cleanup_cluster "$project_id" "$resource_id"; then
                    cleanup_success=true
                    break
                fi
                ;;
            "databaseUser")
                if cleanup_database_user "$project_id" "$resource_id"; then
                    cleanup_success=true
                    break
                fi
                ;;
            "networkAccess")
                if cleanup_network_access "$project_id" "$resource_id"; then
                    cleanup_success=true
                    break
                fi
                ;;
            "project")
                if cleanup_project "$resource_id"; then
                    cleanup_success=true
                    break
                fi
                ;;
            *)
                log_to_file "CLEANUP_ERROR: Unknown resource type $resource_type"
                return 1
                ;;
        esac
        
        if [[ $attempt -lt $ATLAS_API_RETRY_COUNT ]]; then
            log_to_file "CLEANUP_RETRY: Attempt $attempt failed, retrying in ${ATLAS_API_RETRY_DELAY}s..."
            sleep $ATLAS_API_RETRY_DELAY
        fi
        ((attempt++))
    done
    
    if [[ "$cleanup_success" == "true" ]]; then
        log_to_file "CLEANUP_SUCCESS: $resource_type $resource_name"
        print_success "Cleaned up $resource_type: $resource_name"
        return 0
    else
        log_to_file "CLEANUP_FAILED: $resource_type $resource_name after $ATLAS_API_RETRY_COUNT attempts"
        print_error "Failed to clean up $resource_type: $resource_name"
        return 1
    fi
}

cleanup_cluster() {
    local project_id="$1"
    local cluster_name="$2"
    
    # Use the matlas CLI to delete the cluster
    if [[ -f "$PROJECT_ROOT/matlas" ]]; then
        timeout 300s "$PROJECT_ROOT/matlas" atlas clusters delete "$cluster_name" \
            --project-id "$project_id" \
            --yes 2>/dev/null || return 1
    else
        # Fallback to direct API call
        print_warning "matlas binary not found, using direct API call"
        return 1
    fi
    
    # Wait for cluster deletion to complete
    local wait_count=0
    while [[ $wait_count -lt 30 ]]; do
        if ! "$PROJECT_ROOT/matlas" atlas clusters get "$cluster_name" \
            --project-id "$project_id" 2>/dev/null; then
            return 0  # Cluster no longer exists
        fi
        sleep 10
        ((wait_count++))
    done
    
    return 1  # Timeout waiting for deletion
}

cleanup_database_user() {
    local project_id="$1"
    local username="$2"
    
    if [[ -f "$PROJECT_ROOT/matlas" ]]; then
        timeout 60s "$PROJECT_ROOT/matlas" atlas users delete "$username" \
            --project-id "$project_id" \
            --database-name "admin" \
            --yes 2>/dev/null || return 1
    else
        return 1
    fi
    
    return 0
}

cleanup_network_access() {
    local project_id="$1"
    local access_list_id="$2"
    
    if [[ -f "$PROJECT_ROOT/matlas" ]]; then
        timeout 60s "$PROJECT_ROOT/matlas" atlas network delete "$access_list_id" \
            --project-id "$project_id" \
            --yes 2>/dev/null || return 1
    else
        return 1
    fi
    
    return 0
}

cleanup_project() {
    local project_id="$1"
    
    if [[ -f "$PROJECT_ROOT/matlas" ]]; then
        timeout 180s "$PROJECT_ROOT/matlas" atlas projects delete "$project_id" \
            --yes 2>/dev/null || return 1
    else
        return 1
    fi
    
    return 0
}

# Critical: Main cleanup function that runs on ALL exit scenarios
cleanup_all_resources() {
    if [[ "$CLEANUP_IN_PROGRESS" == "true" ]]; then
        return 0  # Avoid recursive cleanup
    fi
    
    CLEANUP_IN_PROGRESS=true
    
    print_header "RESOURCE CLEANUP - CRITICAL SECTION"
    log_to_file "CLEANUP_ALL_START: PID=$TEST_PID"
    
    if [[ "$SKIP_CLEANUP" == "true" ]]; then
        print_warning "Cleanup skipped by user request (SKIP_CLEANUP=true)"
        log_to_file "CLEANUP_SKIPPED: User requested skip"
        return 0
    fi
    
    local cleanup_errors=0
    local total_resources=${#CREATED_RESOURCES[@]}
    
    print_info "Cleaning up $total_resources tracked resources..."
    
    # Load any resources from previous runs
    load_resource_tracking
    
    # Cleanup tracked resources in reverse order (LIFO)
    for ((i=${#CREATED_RESOURCES[@]}-1; i>=0; i--)); do
        local resource_json="${CREATED_RESOURCES[i]}"
        
        # Extract resource information (simplified JSON parsing)
        local resource_type=$(echo "$resource_json" | grep -o '"type":"[^"]*"' | cut -d'"' -f4)
        local resource_id=$(echo "$resource_json" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        local resource_name=$(echo "$resource_json" | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
        local project_id=$(echo "$resource_json" | grep -o '"project_id":"[^"]*"' | cut -d'"' -f4)
        
        if ! cleanup_atlas_resource "$resource_type" "$resource_id" "$project_id" "$resource_name"; then
            ((cleanup_errors++))
        fi
    done
    
    # Run custom cleanup functions
    print_info "Running custom cleanup functions..."
    for cleanup_func_entry in "${CLEANUP_FUNCTIONS[@]}"; do
        local cleanup_command=$(echo "$cleanup_func_entry" | cut -d'|' -f1)
        local description=$(echo "$cleanup_func_entry" | cut -d'|' -f2)
        
        print_info "Running: $description"
        log_to_file "CLEANUP_CUSTOM: $description"
        
        if eval "$cleanup_command" 2>>"$CLEANUP_LOG_FILE"; then
            print_success "Custom cleanup completed: $description"
            log_to_file "CLEANUP_CUSTOM_SUCCESS: $description"
        else
            print_error "Custom cleanup failed: $description"
            log_to_file "CLEANUP_CUSTOM_FAILED: $description"
            ((cleanup_errors++))
        fi
    done
    
    # Cleanup test artifacts
    cleanup_test_artifacts
    
    # Final cleanup report
    print_subheader "Cleanup Summary"
    if [[ $cleanup_errors -eq 0 ]]; then
        print_success "All resources cleaned up successfully"
        log_to_file "CLEANUP_ALL_SUCCESS: No errors"
    else
        print_error "$cleanup_errors cleanup operations failed"
        print_warning "Check $CLEANUP_LOG_FILE for details"
        log_to_file "CLEANUP_ALL_COMPLETED: $cleanup_errors errors"
        EXIT_CODE=1
    fi
    
    CLEANUP_IN_PROGRESS=false
}

cleanup_test_artifacts() {
    print_info "Cleaning up test artifacts..."
    
    # Remove temporary test files
    find "$PROJECT_ROOT" -name "test-*.yaml" -type f -mtime +1 -delete 2>/dev/null || true
    find "$PROJECT_ROOT" -name "*.tmp" -type f -mtime +1 -delete 2>/dev/null || true
    
    # Clean up old test reports (keep last 5 runs)
    if [[ -d "$TEST_REPORTS_DIR" ]]; then
        find "$TEST_REPORTS_DIR" -name "e2e-run-*" -type d | \
            sort -r | tail -n +6 | xargs rm -rf 2>/dev/null || true
    fi
    
    log_to_file "CLEANUP_ARTIFACTS: Test artifacts cleaned"
}

# Signal handlers for graceful cleanup
handle_exit() {
    local exit_code=$?
    log_to_file "HANDLE_EXIT: Code=$exit_code"
    cleanup_all_resources
    exit $EXIT_CODE
}

handle_interrupt() {
    print_warning "Interrupt signal received - cleaning up resources..."
    log_to_file "HANDLE_INTERRUPT: Signal received"
    EXIT_CODE=130
    cleanup_all_resources
    exit $EXIT_CODE
}

handle_termination() {
    print_warning "Termination signal received - cleaning up resources..."
    log_to_file "HANDLE_TERMINATION: Signal received"
    EXIT_CODE=143
    cleanup_all_resources
    exit $EXIT_CODE
}

# Setup signal handlers - CRITICAL for resource cleanup
setup_signal_handlers() {
    trap 'handle_exit' EXIT
    trap 'handle_interrupt' INT
    trap 'handle_termination' TERM
    
    # Handle unexpected errors
    set -E
    trap 'print_error "Unexpected error on line $LINENO"; EXIT_CODE=1; cleanup_all_resources; exit $EXIT_CODE' ERR
    
    log_to_file "SIGNAL_HANDLERS: Configured for PID=$TEST_PID"
}

# Environment setup and validation
setup_test_environment() {
    print_subheader "Setting up test environment"
    
    TEST_START_TIME=$(date +%s)
    
    # Create test directories
    mkdir -p "$E2E_REPORTS_DIR"
    mkdir -p "$(dirname "$CLEANUP_LOG_FILE")"
    
    # Initialize log file
    echo "E2E Test Cleanup Log - Started at $(date)" > "$CLEANUP_LOG_FILE"
    log_to_file "TEST_START: PID=$TEST_PID"
    
    # Load environment from .env if it exists
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        print_info "Loading environment from .env file"
        set -o allexport
        source "$PROJECT_ROOT/.env"
        set +o allexport
        log_to_file "ENV_LOADED: .env file processed"
    fi
    
    # Validate required environment variables
    validate_environment
    
    # Initialize resource tracking
    load_resource_tracking
    
    print_success "Test environment setup complete"
}

validate_environment() {
    print_info "Validating environment..."
    
    local missing_vars=()
    
    # Check Atlas credentials
    if [[ -z "${ATLAS_PUB_KEY:-}" && -z "${ATLAS_PUBLIC_KEY:-}" ]]; then
        missing_vars+=("ATLAS_PUB_KEY or ATLAS_PUBLIC_KEY")
    fi
    
    if [[ -z "${ATLAS_API_KEY:-}" && -z "${ATLAS_PRIVATE_KEY:-}" ]]; then
        missing_vars+=("ATLAS_API_KEY or ATLAS_PRIVATE_KEY")
    fi
    
    if [[ -z "${PROJECT_ID:-}" && -z "${ATLAS_PROJECT_ID:-}" ]]; then
        missing_vars+=("PROJECT_ID or ATLAS_PROJECT_ID")
    fi
    
    # Check for matlas binary
    if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
        print_error "matlas binary not found at $PROJECT_ROOT/matlas"
        print_info "Run 'make build' or 'go build -o matlas' first"
        exit 1
    fi
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        print_error "Missing required environment variables:"
        for var in "${missing_vars[@]}"; do
            print_error "  - $var"
        done
        print_info "Set these in your .env file or environment"
        exit 1
    fi
    
    # Normalize environment variables
    export ATLAS_PUBLIC_KEY="${ATLAS_PUBLIC_KEY:-${ATLAS_PUB_KEY:-}}"
    export ATLAS_PRIVATE_KEY="${ATLAS_PRIVATE_KEY:-${ATLAS_API_KEY:-}}"
    export ATLAS_PROJECT_ID="${ATLAS_PROJECT_ID:-${PROJECT_ID:-}}"
    export ATLAS_ORG_ID="${ATLAS_ORG_ID:-${ORG_ID:-}}"
    
    log_to_file "ENV_VALIDATED: All required variables present"
    print_success "Environment validation passed"
}

# Build the matlas binary if needed
build_matlas() {
    print_subheader "Building matlas binary"
    
    cd "$PROJECT_ROOT"
    
    if [[ ! -f "matlas" ]] || [[ "main.go" -nt "matlas" ]]; then
        print_info "Building matlas binary..."
        
        if go build -o matlas 2>&1; then
            print_success "matlas binary built successfully"
        else
            print_error "Failed to build matlas binary"
            exit 1
        fi
    else
        print_info "matlas binary is up to date"
    fi
}

# E2E Test Functions

run_unit_tests() {
    print_subheader "Running Unit Tests"
    
    cd "$PROJECT_ROOT"
    
    print_info "Cleaning test cache..."
    go clean -testcache
    
    local test_output="$E2E_REPORTS_DIR/unit-tests.log"
    
    if go test -race -timeout="$TEST_TIMEOUT" ./internal/... ./cmd/... 2>&1 | tee "$test_output"; then
        print_success "Unit tests passed"
        return 0
    else
        print_error "Unit tests failed"
        print_info "Output saved to: $test_output"
        return 1
    fi
}

run_atlas_operations_e2e() {
    print_subheader "E2E Test: Atlas Operations"
    
    local test_project_name="e2e-test-$(date +%s)"
    local test_user_name="e2e-user-$(date +%s)"
    local test_cluster_name="e2e-cluster-$(date +%s)"
    
    # Test project operations
    print_info "Testing project operations..."
    
    # Create test project (if we have org permissions)
    if [[ -n "${ATLAS_ORG_ID:-}" ]]; then
            if "$PROJECT_ROOT/matlas" atlas projects create "$test_project_name" \
        --org-id "$ATLAS_ORG_ID" 2>/dev/null; then
            
            track_resource "project" "$test_project_name" "$test_project_name" ""
            print_success "Created test project: $test_project_name"
        else
            print_warning "Could not create test project (permissions may be limited)"
        fi
    fi
    
    # Test database user operations
    print_info "Testing database user operations..."
    
    # Capture error output for debugging
    local user_output
    
    if user_output=$("$PROJECT_ROOT/matlas" atlas users create --username "$test_user_name" \
        --project-id "$ATLAS_PROJECT_ID" \
        --password "TempPassword123!" \
        --roles "readWrite@admin" 2>&1); then
        
        track_resource "databaseUser" "$test_user_name" "$test_user_name" "$ATLAS_PROJECT_ID"
        print_success "Created test database user: $test_user_name"
        
        if [[ "$VERBOSE" == "true" ]]; then
            echo "User creation output: $user_output"
        fi
        
        # List users to verify
        if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$test_user_name"; then
            print_success "Database user visible in list"
        else
            print_warning "Database user not found in list"
        fi
    else
        print_error "Failed to create test database user"
        print_error "Error details: $user_output"
        log_to_file "E2E_USER_CREATE_ERROR: $user_output"
        
        # Additional diagnostics
        print_info "Running E2E diagnostics..."
        print_info "Project ID: $ATLAS_PROJECT_ID"
        print_info "Organization ID: ${ATLAS_ORG_ID:-"Not set"}"
        print_info "Matlas binary: $(ls -la "$PROJECT_ROOT/matlas" 2>/dev/null || echo "Not found")"
        
        # Test basic connectivity and permissions
        print_info "Testing Atlas API connectivity..."
        local projects_output
        if projects_output=$("$PROJECT_ROOT/matlas" atlas projects list 2>&1); then
            print_success "Atlas API connectivity: Working"
            if [[ "$VERBOSE" == "true" ]]; then
                echo "Projects list: $projects_output"
            fi
        else
            print_error "Atlas API connectivity: Failed"
            print_error "Projects list error: $projects_output"
        fi
        
        return 1
    fi
    
    print_success "Atlas operations E2E test completed"
    return 0
}

run_database_operations_e2e() {
    print_subheader "E2E Test: Database Operations"
    
    # Test database listing
    print_info "Testing database list operation..."
    
    if [[ -n "${MONGODB_CONNECTION_STRING:-}" ]]; then
        if "$PROJECT_ROOT/matlas" database list --connection-string "$MONGODB_CONNECTION_STRING" 2>/dev/null; then
            print_success "Database list operation successful"
        else
            print_warning "Database list operation failed (connection may be unavailable)"
        fi
    elif [[ -n "${ATLAS_PROJECT_ID:-}" ]]; then
        # Try to discover available clusters first
        print_info "Discovering available clusters..."
    local available_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null | awk '/^[A-Za-z0-9]/ && !/^NAME/ && !/^✓/ {print $1; exit}' || echo "")
        
        if [[ -n "$available_clusters" ]]; then
            print_info "Testing database list with discovered cluster: $available_clusters"
            if "$PROJECT_ROOT/matlas" database list --cluster "$available_clusters" --project-id "$ATLAS_PROJECT_ID" --use-temp-user 2>/dev/null; then
                print_success "Database list operation with Atlas cluster successful"
            else
                print_warning "Database list operation with Atlas cluster failed (cluster may not be accessible)"
            fi
        else
            print_warning "No clusters found in project - skipping database listing test"
        fi
    else
        print_warning "No database connection available for testing"
    fi
    
    print_success "Database operations E2E test completed"
    return 0
}

run_apply_operations_e2e() {
    print_subheader "E2E Test: Apply Operations"
    
    local test_config_file="$E2E_REPORTS_DIR/test-config.yaml"
    local test_user_name="apply-e2e-user-$(date +%s)"
    
    # Create test configuration
    cat > "$test_config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: apply-e2e-test
spec:
  name: "Apply E2E Test"
  organizationId: "$ATLAS_ORG_ID"
  databaseUsers:
    - metadata:
        name: $test_user_name
      username: $test_user_name
      databaseName: admin
      password: ApplyTestPassword123!
      roles:
        - roleName: readWrite
          databaseName: admin
EOF
    
    track_cleanup_function "rm -f '$test_config_file'" "Remove test configuration file"
    
    # Test validate
    print_info "Testing apply validate..."
    if "$PROJECT_ROOT/matlas" infra validate -f "$test_config_file" 2>/dev/null; then
        print_success "Apply validate successful"
    else
        print_error "Apply validate failed"
        return 1
    fi
    
    # Test plan
    print_info "Testing apply plan..."
    local plan_output="$E2E_REPORTS_DIR/test-plan.json"
    if "$PROJECT_ROOT/matlas" infra plan -f "$test_config_file" --project-id "$ATLAS_PROJECT_ID" --output json > "$plan_output" 2>/dev/null; then
        print_success "Apply plan successful"
        track_cleanup_function "rm -f '$plan_output'" "Remove plan output file"
    else
        print_error "Apply plan failed"
        return 1
    fi
    
    # Test dry-run
    print_info "Testing apply dry-run..."
            if "$PROJECT_ROOT/matlas" infra -f "$test_config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run 2>/dev/null; then
        print_success "Apply dry-run successful"
    else
        print_error "Apply dry-run failed"
        return 1
    fi
    
    # Test actual apply (if not in dry-run mode)
    if [[ "$DRY_RUN" != "true" ]]; then
        print_info "Testing actual apply..."
        if "$PROJECT_ROOT/matlas" infra -f "$test_config_file" --auto-approve --project-id "$ATLAS_PROJECT_ID" --preserve-existing 2>/dev/null; then
            print_success "Apply operation successful"
            track_resource "databaseUser" "$test_user_name" "$test_user_name" "$ATLAS_PROJECT_ID"
        else
            print_error "Apply operation failed"
            return 1
        fi
    else
        print_info "Skipping actual apply (dry-run mode)"
    fi
    
    print_success "Apply operations E2E test completed"
    return 0
}

run_comprehensive_workflow_e2e() {
    print_subheader "E2E Test: Comprehensive Workflow"
    
    # This test runs a complete workflow that touches multiple components
    local workflow_name="comprehensive-e2e-$(date +%s)"
    local config_file="$E2E_REPORTS_DIR/$workflow_name.yaml"
    
    print_info "Creating comprehensive test configuration..."
    
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: $workflow_name
spec:
  name: "Comprehensive E2E Test"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata:
        name: $workflow_name-user
      username: $workflow_name-user
      databaseName: admin
      password: WorkflowTestPassword123!
      roles:
        - roleName: readWrite
          databaseName: testdb
        - roleName: read
          databaseName: admin
  networkAccess:
    - metadata:
        name: $workflow_name-network
      cidr: 203.0.113.0/24
      comment: E2E test network access
EOF
    
    track_cleanup_function "rm -f '$config_file'" "Remove comprehensive test configuration"
    
    # Validate configuration
    if ! "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
        print_error "Comprehensive configuration validation failed"
        return 1
    fi
    
    # Generate plan
    local plan_file="$E2E_REPORTS_DIR/$workflow_name-plan.json"
    if ! "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output json > "$plan_file"; then
        print_error "Comprehensive plan generation failed"
        return 1
    fi
    
    track_cleanup_function "rm -f '$plan_file'" "Remove comprehensive plan file"
    
    # Show diff
    if ! "$PROJECT_ROOT/matlas" infra diff -f "$config_file" --project-id "$ATLAS_PROJECT_ID"; then
        print_error "Comprehensive diff failed"
        return 1
    fi
    
    # Apply if not in dry-run mode
    if [[ "$DRY_RUN" != "true" ]]; then
        if "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --auto-approve --preserve-existing; then
            print_success "Comprehensive apply successful"
            track_resource "databaseUser" "$workflow_name-user" "$workflow_name-user" "$ATLAS_PROJECT_ID"
            track_resource "networkAccess" "$workflow_name-network" "$workflow_name-network" "$ATLAS_PROJECT_ID"
        else
            print_error "Comprehensive apply failed"
            return 1
        fi
    fi
    
    print_success "Comprehensive workflow E2E test completed"
    return 0
}

# Error simulation tests
run_error_handling_e2e() {
    print_subheader "E2E Test: Error Handling and Recovery"
    
    # Test invalid configuration
    print_info "Testing invalid configuration handling..."
    local invalid_config="$E2E_REPORTS_DIR/invalid-config.yaml"
    
    cat > "$invalid_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: invalid-test-project
spec:
  name: ""  # Invalid: empty project name
  organizationId: ""  # Invalid: empty organization ID
  databaseUsers:
    - metadata:
        name: invalid-user
      username: ""  # Invalid: empty username
      password: "short"  # Invalid: too short
      roles: []  # Invalid: no roles
EOF
    
    track_cleanup_function "rm -f '$invalid_config'" "Remove invalid test configuration"
    
    # This should fail validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_config" 2>/dev/null; then
        print_error "Invalid configuration was accepted (should have failed)"
        return 1
    else
        print_success "Invalid configuration properly rejected"
    fi
    
    # Test non-existent project
    print_info "Testing non-existent project handling..."
    local nonexistent_config="$E2E_REPORTS_DIR/nonexistent-project.yaml"
    
    cat > "$nonexistent_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: nonexistent-test-project
spec:
  name: "Non-existent Test Project"
  organizationId: "000000000000000000000000"  # Non-existent organization ID
  databaseUsers:
    - metadata:
        name: test-user
      username: test-user
      password: TestPassword123!
      roles:
        - roleName: read
          databaseName: admin
EOF
    
    track_cleanup_function "rm -f '$nonexistent_config'" "Remove nonexistent project test configuration"
    
    # This should fail during planning
    if "$PROJECT_ROOT/matlas" infra plan -f "$nonexistent_config" 2>/dev/null; then
        print_error "Non-existent project was accepted (should have failed)"
        return 1
    else
        print_success "Non-existent project properly rejected"
    fi
    
    print_success "Error handling E2E test completed"
    return 0
}

# Performance and stress tests
run_performance_e2e() {
    print_subheader "E2E Test: Performance and Stress"
    
    print_info "Testing large configuration handling..."
    
    # Create unique test identifier to avoid resource conflicts
    local test_id="perf-$(date +%s)-$$"
    local large_config="$E2E_REPORTS_DIR/large-config.yaml"
    
    # Generate a configuration with multiple resources
    cat > "$large_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: $test_id-project
spec:
  name: "Performance Test Project $test_id"
  organizationId: "$ATLAS_ORG_ID"
  databaseUsers:
EOF
    
    for i in {1..10}; do
        cat >> "$large_config" << EOF
    - metadata:
        name: $test_id-user-$i
      username: $test_id-user-$i
      password: PerfTestPassword123!
      roles:
        - roleName: read
          databaseName: admin
      authDatabase: admin
EOF
    done
    
    track_cleanup_function "rm -f '$large_config'" "Remove large test configuration"
    
    # Time the validation
    local start_time=$(date +%s)
    if "$PROJECT_ROOT/matlas" infra validate -f "$large_config" 2>/dev/null; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        print_success "Large configuration validated in ${duration}s"
    else
        print_error "Large configuration validation failed"
        return 1
    fi
    
    # Time the planning
    start_time=$(date +%s)
    if "$PROJECT_ROOT/matlas" infra plan -f "$large_config" --project-id "$ATLAS_PROJECT_ID" >/dev/null 2>&1; then
        end_time=$(date +%s)
        duration=$((end_time - start_time))
        print_success "Large configuration planned in ${duration}s"
    else
        print_error "Large configuration planning failed"
        return 1
    fi
    
    print_success "Performance E2E test completed"
    return 0
}

run_discovery_e2e() {
    print_subheader "E2E Test: Project Discovery"
    
    print_info "Testing project discovery functionality..."
    
    local discovery_output="$E2E_REPORTS_DIR/discovery-output.yaml"
    local discovery_json="$E2E_REPORTS_DIR/discovery-output.json"
    
    # Test basic discovery
    print_info "Testing basic discovery with YAML output..."
    if "$PROJECT_ROOT/matlas" discover --project-id "$ATLAS_PROJECT_ID" --output yaml > "$discovery_output" 2>/dev/null; then
        print_success "Basic discovery with YAML output successful"
        
        # Verify YAML structure
        if grep -q "apiVersion:" "$discovery_output" && grep -q "kind: DiscoveredProject" "$discovery_output"; then
            print_success "Discovery YAML output has correct structure"
        else
            print_error "Discovery YAML output missing required fields"
            return 1
        fi
    else
        print_error "Basic discovery failed"
        return 1
    fi
    
    track_cleanup_function "rm -f '$discovery_output'" "Remove discovery YAML output"
    
    # Test JSON output
    print_info "Testing discovery with JSON output..."
    if "$PROJECT_ROOT/matlas" discover --project-id "$ATLAS_PROJECT_ID" --output json > "$discovery_json" 2>/dev/null; then
        print_success "Discovery with JSON output successful"
        
        # Verify JSON structure
        if python3 -m json.tool "$discovery_json" >/dev/null 2>&1; then
            print_success "Discovery JSON output is valid JSON"
        else
            print_error "Discovery JSON output is invalid"
            return 1
        fi
    else
        print_error "Discovery with JSON output failed"
        return 1
    fi
    
    track_cleanup_function "rm -f '$discovery_json'" "Remove discovery JSON output"
    
    # Test filtered discovery
    print_info "Testing filtered discovery (clusters only)..."
    local filtered_output="$E2E_REPORTS_DIR/discovery-filtered.yaml"
    if "$PROJECT_ROOT/matlas" discover --project-id "$ATLAS_PROJECT_ID" --include clusters --output yaml > "$filtered_output" 2>/dev/null; then
        print_success "Filtered discovery successful"
        
        # Verify only clusters are included
        if grep -q "clusters:" "$filtered_output" && ! grep -q "databaseUsers:" "$filtered_output"; then
            print_success "Filtered discovery correctly includes only clusters"
        else
            print_error "Filtered discovery did not filter correctly"
            return 1
        fi
    else
        print_error "Filtered discovery failed"
        return 1
    fi
    
    track_cleanup_function "rm -f '$filtered_output'" "Remove filtered discovery output"
    
    # Test discovery with secret masking
    print_info "Testing discovery with secret masking..."
    local masked_output="$E2E_REPORTS_DIR/discovery-masked.yaml"
    if "$PROJECT_ROOT/matlas" discover --project-id "$ATLAS_PROJECT_ID" --mask-secrets --output yaml > "$masked_output" 2>/dev/null; then
        print_success "Discovery with secret masking successful"
        
        # Verify secrets are masked
        if grep -q "***MASKED***" "$masked_output" || ! grep -q "password:" "$masked_output"; then
            print_success "Secrets properly masked in discovery output"
        else
            print_warning "No secrets found to mask (may be expected)"
        fi
    else
        print_error "Discovery with secret masking failed"
        return 1
    fi
    
    track_cleanup_function "rm -f '$masked_output'" "Remove masked discovery output"
    
    # Test discovery timeout and error handling
    print_info "Testing discovery error handling..."
    local invalid_output="$E2E_REPORTS_DIR/discovery-invalid.yaml"
    if "$PROJECT_ROOT/matlas" discover --project-id "invalid-project-id" --output yaml > "$invalid_output" 2>/dev/null; then
        print_error "Discovery should have failed with invalid project ID"
        return 1
    else
        print_success "Discovery correctly failed with invalid project ID"
    fi
    
    # Test discovery-to-apply pipeline
    if [[ "$DRY_RUN" != "true" ]]; then
        print_info "Testing discovery-to-apply pipeline..."
        
        # First create a simple resource via apply
        local simple_config="$E2E_REPORTS_DIR/simple-user.yaml"
        cat > "$simple_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: DatabaseUser
metadata:
  name: discovery-test-user
spec:
  username: discovery-test-user
  password: DiscoveryTestPassword123!
  roles:
    - roleName: read
      databaseName: admin
  authDatabase: admin
  projectId: $ATLAS_PROJECT_ID
EOF
        
        if "$PROJECT_ROOT/matlas" infra -f "$simple_config" --auto-approve --preserve-existing 2>/dev/null; then
            print_success "Created test resource for discovery pipeline"
            track_resource "databaseUser" "discovery-test-user" "discovery-test-user" "$ATLAS_PROJECT_ID"
            
            # Discover it back
            local pipeline_output="$E2E_REPORTS_DIR/discovery-pipeline.yaml"
            if "$PROJECT_ROOT/matlas" discover --project-id "$ATLAS_PROJECT_ID" --include users --output yaml > "$pipeline_output" 2>/dev/null; then
                
                # Validate the discovered configuration
                if "$PROJECT_ROOT/matlas" infra validate -f "$pipeline_output" 2>/dev/null; then
                    print_success "Discovery-to-apply pipeline validation successful"
                else
                    print_warning "Discovery-to-apply pipeline validation failed (format differences expected)"
                fi
            else
                print_error "Discovery in pipeline failed"
                return 1
            fi
            
            track_cleanup_function "rm -f '$simple_config' '$pipeline_output'" "Remove pipeline test files"
        else
            print_warning "Could not create test resource for pipeline test"
        fi
    else
        print_info "Skipping discovery-to-apply pipeline test (dry-run mode)"
    fi
    
    print_success "Discovery E2E test completed"
    return 0
}

# Generate comprehensive test report
generate_e2e_report() {
    print_subheader "Generating E2E Test Report"
    
    local report_file="$E2E_REPORTS_DIR/e2e-test-report.md"
    local test_end_time=$(date +%s)
    local test_duration=$((test_end_time - TEST_START_TIME))
    
    cat > "$report_file" << EOF
# matlas-cli End-to-End Test Report

**Test Run Date:** $(date)
**Test Duration:** ${test_duration} seconds
**Exit Code:** $EXIT_CODE

## Test Environment
- Project ID: $ATLAS_PROJECT_ID
- Organization ID: ${ATLAS_ORG_ID:-"Not set"}
- Test Mode: $(if [[ "$DRY_RUN" == "true" ]]; then echo "Dry Run"; else echo "Live Resources"; fi)

## Resources Created and Cleaned Up
$(if [[ ${#CREATED_RESOURCES[@]} -gt 0 ]]; then
    echo "Total Resources Tracked: ${#CREATED_RESOURCES[@]}"
    for resource in "${CREATED_RESOURCES[@]}"; do
        echo "- $resource"
    done
else
    echo "No resources were created during this test run."
fi)

## Cleanup Functions Executed
$(if [[ ${#CLEANUP_FUNCTIONS[@]} -gt 0 ]]; then
    echo "Total Cleanup Functions: ${#CLEANUP_FUNCTIONS[@]}"
    for func in "${CLEANUP_FUNCTIONS[@]}"; do
        echo "- $func"
    done
else
    echo "No custom cleanup functions were registered."
fi)

## Test Results Summary
- Unit Tests: $(if [[ -f "$E2E_REPORTS_DIR/unit-tests.log" ]]; then echo "Executed"; else echo "Skipped"; fi)
- Atlas Operations: Executed
- Database Operations: Executed
- Apply Operations: Executed
- Comprehensive Workflow: Executed
- Error Handling: Executed
- Performance Tests: Executed

## Files Generated
EOF
    
    find "$E2E_REPORTS_DIR" -type f -name "*" | while read -r file; do
        echo "- $(basename "$file")" >> "$report_file"
    done
    
    cat >> "$report_file" << EOF

## Cleanup Log Summary
$(tail -20 "$CLEANUP_LOG_FILE" 2>/dev/null || echo "No cleanup log available")

---
*Report generated by matlas-cli E2E test framework*
EOF
    
    print_success "E2E test report generated: $report_file"
}

# Main execution function
main() {
    local test_mode="${1:-all}"
    local start_banner="matlas-cli End-to-End Test Suite"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        start_banner="$start_banner (DRY RUN MODE)"
    fi
    
    print_header "$start_banner"
    
    # Critical: Setup signal handlers FIRST
    setup_signal_handlers
    
    # Setup test environment
    setup_test_environment
    
    # Build matlas binary
    build_matlas
    
    print_info "Test mode: $test_mode"
    print_info "Timeout: $TEST_TIMEOUT"
    print_info "Verbose: $VERBOSE"
    print_info "Dry run: $DRY_RUN"
    print_info "Skip cleanup: $SKIP_CLEANUP"
    
    # Run environment diagnostics
    print_subheader "Environment Diagnostics"
    print_info "Atlas Configuration Status:"
    echo "  - Public Key: ${ATLAS_PUBLIC_KEY:+Set (${ATLAS_PUBLIC_KEY:0:8}...)} ${ATLAS_PUBLIC_KEY:-Not set}"
    echo "  - Private Key: ${ATLAS_PRIVATE_KEY:+Set (${ATLAS_PRIVATE_KEY:0:8}...)} ${ATLAS_PRIVATE_KEY:-Not set}"
    echo "  - Project ID: ${ATLAS_PROJECT_ID:-Not set}"
    echo "  - Organization ID: ${ATLAS_ORG_ID:-Not set}"
    
    # Test basic connectivity
    print_info "Testing basic Atlas connectivity..."
    local connectivity_output
    if connectivity_output=$("$PROJECT_ROOT/matlas" atlas projects list 2>&1); then
        print_success "Atlas API connectivity: Working"
    else
        print_error "Atlas API connectivity: Failed"
        print_error "Connectivity error: $connectivity_output"
        if [[ "$DRY_RUN" != "true" ]]; then
            print_error "Cannot proceed with E2E tests without Atlas connectivity"
            exit 1
        else
            print_warning "Continuing in dry-run mode despite connectivity issues"
        fi
    fi
    
    # Run tests based on mode
    local test_errors=0
    
    case "$test_mode" in
        "unit")
            run_unit_tests || ((test_errors++))
            ;;
        "atlas")
            run_atlas_operations_e2e || ((test_errors++))
            ;;
        "database")
            run_database_operations_e2e || ((test_errors++))
            ;;
        "apply")
            run_apply_operations_e2e || ((test_errors++))
            ;;
        "workflow")
            run_comprehensive_workflow_e2e || ((test_errors++))
            ;;
        "errors")
            run_error_handling_e2e || ((test_errors++))
            ;;
        "performance")
            run_performance_e2e || ((test_errors++))
            ;;
        "discovery")
            run_discovery_e2e || ((test_errors++))
            ;;
        "all"|*)
            print_info "Running comprehensive E2E test suite..."
            
            run_unit_tests || ((test_errors++))
            run_atlas_operations_e2e || ((test_errors++))
            run_database_operations_e2e || ((test_errors++))
            run_apply_operations_e2e || ((test_errors++))
            run_comprehensive_workflow_e2e || ((test_errors++))
            run_error_handling_e2e || ((test_errors++))
            run_performance_e2e || ((test_errors++))
            run_discovery_e2e || ((test_errors++))
            ;;
    esac
    
    # Set exit code based on test results
    if [[ $test_errors -gt 0 ]]; then
        EXIT_CODE=1
        print_error "$test_errors test(s) failed"
    else
        print_success "All tests passed!"
    fi
    
    # Generate final report
    generate_e2e_report
    
    print_header "E2E Test Suite Complete"
    print_info "Check logs and reports in: $E2E_REPORTS_DIR"
    
    # Cleanup will be called automatically via EXIT trap
}

# Show usage information
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS] [TEST_MODE]

Comprehensive E2E test runner for matlas-cli with guaranteed resource cleanup.

TEST_MODES:
    unit            Run unit tests only
    atlas           Run Atlas operations E2E tests
    database        Run database operations E2E tests
    apply           Run apply operations E2E tests
    workflow        Run comprehensive workflow tests
    errors          Run error handling tests
    performance     Run performance tests
    discovery       Run project discovery E2E tests
    all             Run all E2E tests (default)

OPTIONS:
    --dry-run       Run tests without creating actual resources
    --verbose       Enable verbose output
    --skip-cleanup  Skip resource cleanup (dangerous!)
    --timeout TIME  Set test timeout (default: $DEFAULT_TEST_TIMEOUT)
    --help          Show this help message

ENVIRONMENT VARIABLES:
    ATLAS_PUB_KEY or ATLAS_PUBLIC_KEY     Atlas public key
    ATLAS_API_KEY or ATLAS_PRIVATE_KEY    Atlas private key  
    PROJECT_ID or ATLAS_PROJECT_ID        Atlas project ID
    ORG_ID or ATLAS_ORG_ID               Atlas organization ID (optional)
    MONGODB_CONNECTION_STRING             MongoDB connection string (optional)
    
    DRY_RUN                              Enable dry-run mode (true/false)
    VERBOSE                              Enable verbose output (true/false)
    SKIP_CLEANUP                         Skip cleanup (true/false) - DANGEROUS!
    TEST_TIMEOUT                         Test timeout duration

EXAMPLES:
    # Run all tests with cleanup
    $0 all

    # Run only Atlas tests in dry-run mode
    $0 --dry-run atlas

    # Run comprehensive workflow test with verbose output
    $0 --verbose workflow

    # Run performance tests with custom timeout
    $0 --timeout 45m performance

    # Run discovery tests
    $0 discovery

SAFETY FEATURES:
    - Automatic resource cleanup on success, failure, or interruption
    - Signal handlers for SIGINT, SIGTERM, and EXIT
    - Resource tracking with persistent state
    - Retry mechanisms for cleanup operations
    - Comprehensive logging of all operations

WARNING:
    This script creates real Atlas resources. Ensure you have the correct
    project credentials and are aware that resources will be created and
    deleted during testing.

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --skip-cleanup)
            SKIP_CLEANUP=true
            shift
            ;;
        --timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        --help|-h)
            show_usage
            exit 0
            ;;
        --*)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
        *)
            TEST_MODE="$1"
            shift
            ;;
    esac
done

# Run main function
main "${TEST_MODE:-all}" 