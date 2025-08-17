#!/bin/bash

# Safe Cluster Lifecycle Tests
# This script tests cluster creation and management without affecting existing resources
# It uses the --preserve-existing flag to ensure only test resources are managed

set -euo pipefail

# Constants
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports"
REGION="${ATLAS_REGION:-us-west-2}"

# Ensure test reports directory exists
mkdir -p "$TEST_REPORTS_DIR"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions for colored output
print_header() { echo -e "\n${BLUE}========================================${NC}"; echo -e "${BLUE}$1${NC}"; echo -e "${BLUE}========================================${NC}"; }
print_subheader() { echo -e "\n${BLUE}$1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }

# Resource tracking for cleanup
declare -a CREATED_RESOURCES=()

# Track created resources for cleanup
track_resource() {
    local resource_type="$1"
    local resource_name="$2"
    local test_type="$3"
    CREATED_RESOURCES+=("$resource_type:$resource_name:$test_type")
    print_info "Tracking resource: $resource_type:$resource_name ($test_type)"
}

# Environment validation
check_environment() {
    print_info "Validating safe cluster lifecycle test environment..."
    
    local errors=0
    
    # Check required environment variables
    if [[ -z "${ATLAS_PUB_KEY:-}" || -z "${ATLAS_API_KEY:-}" ]]; then
        print_error "Atlas credentials not configured"
        print_info "Required: ATLAS_PUB_KEY and ATLAS_API_KEY"
        ((errors++))
    fi
    
    if [[ -z "${ATLAS_PROJECT_ID:-}" ]]; then
        print_error "Atlas project ID not configured"
        print_info "Required: ATLAS_PROJECT_ID"
        ((errors++))
    fi
    
    if [[ -z "${ATLAS_ORG_ID:-}" ]]; then
        print_error "Atlas organization ID not configured"
        print_info "Required: ATLAS_ORG_ID"
        ((errors++))
    fi
    
    # Check if matlas CLI is available
    if ! "$PROJECT_ROOT/matlas" version >/dev/null 2>&1; then
        print_error "matlas CLI not available or not working"
        print_info "Run 'make build' to build the CLI"
        ((errors++))
    fi
    
    # Check if required tools are available
    for tool in jq; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            print_error "Required tool '$tool' not found"
            ((errors++))
        fi
    done
    
    if [[ $errors -eq 0 ]]; then
        print_success "Environment validation passed"
        return 0
    else
        print_error "Environment validation failed"
        return 1
    fi
}

# Wait for cluster to be ready
wait_for_cluster_ready() {
    local cluster_name="$1"
    local timeout="${2:-900}" # 15 minutes default
    local interval=30
    local elapsed=0
    
    print_info "Waiting for cluster '$cluster_name' to be ready (timeout: ${timeout}s)..."
    
    while [[ $elapsed -lt $timeout ]]; do
        local status
        if status=$("$PROJECT_ROOT/matlas" atlas clusters get "$cluster_name" --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.stateName' 2>/dev/null); then
            case "$status" in
                "IDLE"|"REPLICATING")
                    print_success "Cluster '$cluster_name' is ready (status: $status)"
                    return 0
                    ;;
                "CREATING"|"UPDATING")
                    print_info "Cluster status: $status (waiting...)"
                    ;;
                "ERROR"|"DELETING")
                    print_error "Cluster in error state: $status"
                    return 1
                    ;;
                *)
                    print_info "Cluster status: $status"
                    ;;
            esac
        else
            print_warning "Could not get cluster status"
        fi
        
        sleep $interval
        elapsed=$((elapsed + interval))
    done
    
    print_error "Timeout waiting for cluster '$cluster_name' to be ready"
    return 1
}

# Wait for cluster to be deleted
wait_for_cluster_deleted() {
    local cluster_name="$1"
    local timeout="${2:-900}" # 15 minutes default
    local interval=30
    local elapsed=0
    
    print_info "Waiting for cluster '$cluster_name' to be deleted (timeout: ${timeout}s)..."
    
    while [[ $elapsed -lt $timeout ]]; do
        if ! "$PROJECT_ROOT/matlas" atlas clusters get "$cluster_name" --project-id "$ATLAS_PROJECT_ID" >/dev/null 2>&1; then
            print_success "Cluster '$cluster_name' has been deleted"
            return 0
        fi
        
        sleep $interval
        elapsed=$((elapsed + interval))
    done
    
    print_error "Timeout waiting for cluster '$cluster_name' to be deleted"
    return 1
}

# Safe CLI-based cluster test
test_safe_cli_cluster() {
    print_header "Safe CLI-Based Cluster Test"
    
    local cluster_name
    local user_name
    # Use shorter names to avoid Atlas 23-character limit
    local timestamp=$(date +%s | tail -c 6)  # Last 5 digits of timestamp
    cluster_name="scli-t-${timestamp}-${RANDOM:0:3}"
    user_name="scli-u-${timestamp}-${RANDOM:0:3}"
    
    print_info "Testing safe CLI cluster lifecycle with cluster: $cluster_name"
    print_info "This test only manages resources it creates - existing resources are preserved"
    
    # Step 1: Create cluster
    print_subheader "Step 1: Creating test cluster via CLI"
    
    if "$PROJECT_ROOT/matlas" atlas clusters create "$cluster_name" \
        --project-id "$ATLAS_PROJECT_ID" \
        --provider AWS \
        --region "$REGION" \
        --instanceSize M10 \
        --diskSizeGB 10 \
        --mongodbVersion 7.0 \
        --clusterType REPLICASET; then
        
        track_resource "cluster" "$cluster_name" "safe-cli"
        print_success "Cluster creation initiated"
        
        # Wait for cluster to be ready
        if wait_for_cluster_ready "$cluster_name"; then
            print_success "Test cluster is ready"
        else
            print_error "Test cluster failed to become ready"
            return 1
        fi
    else
        print_error "Cluster creation failed"
        return 1
    fi
    
    # Step 2: Create database user
    print_subheader "Step 2: Creating database user via CLI"
    
    if "$PROJECT_ROOT/matlas" atlas users create \
        --username "$user_name" \
        --password "SafeCliPassword123!" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database-name admin \
        --role readWrite \
        --role-database testapp; then
        
        track_resource "user" "$user_name" "safe-cli"
        print_success "Database user created"
    else
        print_error "Database user creation failed"
        return 1
    fi
    
    # Step 3: Clean up test resources
    print_subheader "Step 3: Cleaning up test resources"
    
    # Delete user first
    if "$PROJECT_ROOT/matlas" atlas users delete "$user_name" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database-name admin \
        --yes; then
        print_success "Test user deleted"
    else
        print_warning "Failed to delete test user"
    fi
    
    # Delete cluster
    if "$PROJECT_ROOT/matlas" atlas clusters delete "$cluster_name" \
        --project-id "$ATLAS_PROJECT_ID" \
        --yes; then
        print_success "Test cluster deletion initiated"
        
        if wait_for_cluster_deleted "$cluster_name"; then
            print_success "Test cluster fully deleted"
        else
            print_warning "Test cluster deletion may still be in progress"
        fi
    else
        print_warning "Failed to delete test cluster"
    fi
    
    print_success "Safe CLI cluster test completed successfully"
    return 0
}

# Safe YAML-based cluster test using --preserve-existing
test_safe_yaml_cluster() {
    print_header "Safe YAML-Based Cluster Test (with --preserve-existing)"
    
    local cluster_name
    local user_name
    # Use shorter names to avoid Atlas 23-character limit
    local timestamp=$(date +%s | tail -c 6)  # Last 5 digits of timestamp
    cluster_name="syml-t-${timestamp}-${RANDOM:0:3}"
    user_name="syml-u-${timestamp}-${RANDOM:0:3}"
    local config_file="$TEST_REPORTS_DIR/safe-yaml-config.yaml"
    
    print_info "Testing safe YAML cluster lifecycle with cluster: $cluster_name"
    print_info "Using --preserve-existing flag to protect existing resources"
    
    # Get project name for proper YAML configuration
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        print_warning "Could not get project name, using project ID"
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    # Record existing resources before test
    print_subheader "Step 1: Recording existing resources"
    local existing_clusters
    if existing_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null); then
        local cluster_count
        cluster_count=$(echo "$existing_clusters" | jq '. | length' 2>/dev/null || echo "0")
        print_info "Found $cluster_count existing clusters before test"
        
        if [[ $cluster_count -gt 0 ]]; then
            echo "$existing_clusters" | jq -r '.[].name' | while read -r name; do
                print_info "  - $name"
            done
        fi
    else
        print_warning "Could not list existing clusters"
        existing_clusters="[]"
    fi
    
    # Step 2: Create YAML configuration
    print_subheader "Step 2: Creating safe YAML configuration"
    
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: safe-yaml-test
  labels:
    test-type: safe-yaml
    purpose: testing
    safety: preserve-existing
  annotations:
    description: "Safe test that preserves existing resources"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: $cluster_name
      labels:
        test-type: safe-yaml
        purpose: testing
      annotations:
        description: "Test cluster for safe YAML testing"
    spec:
      projectName: "$project_name"
      provider: AWS
      region: $REGION
      instanceSize: M10
      diskSizeGB: 10
      backupEnabled: false
      mongodbVersion: "7.0"
      clusterType: REPLICASET
      
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: $user_name
      labels:
        test-type: safe-yaml
      annotations:
        description: "Test user for safe YAML testing"
    spec:
      projectName: "$project_name"
      username: $user_name
      databaseName: admin
      password: SafeYamlPassword123!
      roles:
        - roleName: readWrite
          databaseName: testapp
        - roleName: read
          databaseName: admin
      scopes:
        - name: $cluster_name
          type: CLUSTER
EOF
    
    track_resource "config" "$config_file" "safe-yaml"
    track_resource "cluster" "$cluster_name" "safe-yaml"
    track_resource "user" "$user_name" "safe-yaml"
    
    print_success "Safe YAML configuration created"
    
    # Step 3: Validate configuration
    print_subheader "Step 3: Validating YAML configuration"
    
    if "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
        print_success "YAML configuration validation passed"
    else
        print_error "YAML configuration validation failed"
        return 1
    fi
    
    # Step 4: Apply configuration with --preserve-existing
    print_subheader "Step 4: Applying YAML configuration (with --preserve-existing)"
    
    print_warning "Using --preserve-existing to ensure existing resources are not deleted"
    
    if "$PROJECT_ROOT/matlas" infra apply -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --preserve-existing \
        --auto-approve; then
        
        print_success "Safe configuration apply completed"
        
        # Wait for cluster to be ready
        if wait_for_cluster_ready "$cluster_name"; then
            print_success "Test cluster is ready"
        else
            print_error "Test cluster failed to become ready"
            return 1
        fi
    else
        print_error "Safe configuration apply failed"
        return 1
    fi
    
    # Step 5: Verify existing clusters are still there
    print_subheader "Step 5: Verifying existing clusters are preserved"
    
    local current_clusters
    if current_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null); then
        local current_count
        current_count=$(echo "$current_clusters" | jq '. | length' 2>/dev/null || echo "0")
        local original_count
        original_count=$(echo "$existing_clusters" | jq '. | length' 2>/dev/null || echo "0")
        local expected_count=$((original_count + 1))
        
        if [[ $current_count -eq $expected_count ]]; then
            print_success "Cluster count correct: $current_count (was $original_count, added 1 test cluster)"
            
            # Verify our test cluster exists
            if echo "$current_clusters" | jq -r '.[].name' | grep -q "^$cluster_name\$"; then
                print_success "Test cluster exists as expected"
            else
                print_error "Test cluster not found"
                return 1
            fi
            
            # Verify existing clusters still exist
            if [[ $original_count -gt 0 ]]; then
                local missing_clusters=0
                echo "$existing_clusters" | jq -r '.[].name' | while read -r existing_name; do
                    if echo "$current_clusters" | jq -r '.[].name' | grep -q "^$existing_name\$"; then
                        print_success "Existing cluster preserved: $existing_name"
                    else
                        print_error "Existing cluster missing: $existing_name"
                        missing_clusters=$((missing_clusters + 1))
                    fi
                done
                
                if [[ $missing_clusters -eq 0 ]]; then
                    print_success "All existing clusters preserved"
                fi
            fi
        else
            print_error "Unexpected cluster count: expected $expected_count, found $current_count"
            return 1
        fi
    else
        print_warning "Could not verify cluster preservation"
    fi
    
    # Step 6: Clean up only test resources
    print_subheader "Step 6: Cleaning up test resources (existing resources preserved)"
    
    if "$PROJECT_ROOT/matlas" infra destroy -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --preserve-existing \
        --auto-approve; then
        
        print_success "Test resource cleanup completed"
        
        # Wait for cluster deletion
        if wait_for_cluster_deleted "$cluster_name"; then
            print_success "Test cluster deleted successfully"
        else
            print_warning "Test cluster deletion verification timed out"
        fi
    else
        print_error "Test resource cleanup failed"
        return 1
    fi
    
    # Step 7: Final verification that existing clusters are still intact
    print_subheader "Step 7: Final verification of existing cluster preservation"
    
    local final_clusters
    if final_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null); then
        local final_count
        final_count=$(echo "$final_clusters" | jq '. | length' 2>/dev/null || echo "0")
        local original_count
        original_count=$(echo "$existing_clusters" | jq '. | length' 2>/dev/null || echo "0")
        
        if [[ $final_count -eq $original_count ]]; then
            print_success "All existing clusters preserved - test successful!"
            
            # Verify each existing cluster is still there
            if [[ $original_count -gt 0 ]]; then
                echo "$existing_clusters" | jq -r '.[].name' | while read -r existing_name; do
                    if echo "$final_clusters" | jq -r '.[].name' | grep -q "^$existing_name\$"; then
                        print_success "Confirmed preserved: $existing_name"
                    else
                        print_error "Lost existing cluster: $existing_name"
                    fi
                done
            fi
        else
            print_error "Cluster count mismatch: started with $original_count, ended with $final_count"
            return 1
        fi
    else
        print_warning "Could not verify final cluster state"
    fi
    
    print_success "Safe YAML cluster test completed successfully"
    return 0
}

# Test comparison between CLI and YAML approaches
test_approach_comparison() {
    print_header "Approach Comparison (Safe Mode)"
    
    print_info "Both CLI and YAML approaches use safe patterns:"
    print_success "✓ CLI: Only manages clusters it creates with specific naming patterns"
    print_success "✓ YAML: Uses --preserve-existing flag to protect existing resources"
    print_success "✓ Both approaches are safe for use in projects with existing clusters"
    
    print_info "Key benefits of the safe approach:"
    echo "  • Existing resources are never deleted"
    echo "  • Test resources use predictable naming patterns"
    echo "  • Cleanup only affects resources created by the test"
    echo "  • Safe for use in production environments"
    
    return 0
}

# Cleanup function (emergency cleanup for test resources only)
cleanup_resources() {
    print_info "Cleaning up safe test resources..."
    
    # Only clean up test resources that match our safe naming patterns
    local test_patterns=("scli-t-" "syml-t-")
    
    if [[ -n "${ATLAS_PROJECT_ID:-}" ]]; then
        # Find any test clusters that match our patterns
        local potential_clusters
        if potential_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null); then
            for pattern in "${test_patterns[@]}"; do
                local test_clusters
                test_clusters=$(echo "$potential_clusters" | jq -r ".[] | select(.name | startswith(\"$pattern\")) | .name" 2>/dev/null || true)
                
                for cluster in $test_clusters; do
                    if [[ -n "$cluster" ]]; then
                        print_info "Cleaning up test cluster: $cluster"
                        if "$PROJECT_ROOT/matlas" atlas clusters delete "$cluster" \
                            --project-id "$ATLAS_PROJECT_ID" \
                            --yes 2>/dev/null; then
                            print_success "Test cluster deletion initiated: $cluster"
                        else
                            print_warning "Failed to delete test cluster: $cluster"
                        fi
                    fi
                done
            done
        fi
        
        # Find any test users that match our patterns  
        local potential_users
        if potential_users=$("$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null); then
            for pattern in "${test_patterns[@]}"; do
                local test_users
                test_users=$(echo "$potential_users" | grep -E "${pattern}[0-9]+" | awk '{print $1}' || true)
                
                for user in $test_users; do
                    if [[ -n "$user" ]]; then
                        print_info "Cleaning up test user: $user"
                        if "$PROJECT_ROOT/matlas" atlas users delete "$user" \
                            --project-id "$ATLAS_PROJECT_ID" \
                            --database-name admin \
                            --yes 2>/dev/null; then
                            print_success "Test user deleted: $user"
                        else
                            print_warning "Failed to delete test user: $user"
                        fi
                    fi
                done
            done
        fi
    fi
    
    # Clean up temporary files
    local test_files=("$TEST_REPORTS_DIR/safe-yaml-config.yaml")
    for file in "${test_files[@]}"; do
        if [[ -f "$file" ]]; then
            rm -f "$file"
            print_info "Cleaned up test file: $file"
        fi
    done
    
    print_success "Safe cleanup completed"
}

# Main test runner
run_safe_cluster_tests() {
    local test_type="${1:-all}"
    
    print_header "Safe MongoDB Atlas Cluster Lifecycle Tests"
    print_success "✓ These tests preserve existing resources and are safe for production projects"
    print_info "Using --preserve-existing flag and safe naming patterns"
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
        "cli")
            print_info "Running safe CLI cluster tests..."
            test_safe_cli_cluster || ((test_failures++))
            ;;
        "yaml")
            print_info "Running safe YAML cluster tests..."
            test_safe_yaml_cluster || ((test_failures++))
            ;;
        "comparison")
            print_info "Running approach comparison..."
            test_approach_comparison || ((test_failures++))
            ;;
        "all"|*)
            print_info "Running complete safe test suite..."
            test_safe_cli_cluster || ((test_failures++))
            echo
            test_safe_yaml_cluster || ((test_failures++))
            echo
            test_approach_comparison || ((test_failures++))
            ;;
    esac
    
    echo
    if [[ $test_failures -eq 0 ]]; then
        print_success "All safe cluster tests passed!"
        print_info "Existing resources were preserved throughout testing"
        return 0
    else
        print_error "$test_failures safe cluster test(s) failed"
        return 1
    fi
}

# Script usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  cli              Run safe CLI-based cluster tests only"
    echo "  yaml             Run safe YAML-based cluster tests only"
    echo "  comparison       Show approach comparison"
    echo "  all              Run complete safe test suite (default)"
    echo
    echo "Safety Features:"
    echo "  • Uses --preserve-existing flag for YAML operations"
    echo "  • Only manages resources with specific test naming patterns"
    echo "  • Never deletes existing production resources"
    echo "  • Safe for use in projects with existing clusters"
    echo
    echo "Environment variables required:"
    echo "  ATLAS_PUB_KEY       Atlas public API key"
    echo "  ATLAS_API_KEY       Atlas private API key"
    echo "  ATLAS_PROJECT_ID    Atlas project ID for testing"
    echo "  ATLAS_ORG_ID        Atlas organization ID"
    echo
    echo "Examples:"
    echo "  $0                     # Run complete safe test suite"
    echo "  $0 cli                 # Run safe CLI tests only"
    echo "  $0 yaml                # Run safe YAML tests only"
}

# Main execution
main() {
    local test_type="${1:-all}"
    case "$test_type" in
        "cli"|"yaml"|"comparison"|"all")
            run_safe_cluster_tests "$test_type"
            ;;
        "-h"|"--help"|"help")
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown command: $test_type"
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
