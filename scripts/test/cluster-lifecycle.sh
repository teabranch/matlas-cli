#!/usr/bin/env bash

# Cluster Lifecycle Testing for matlas-cli
# Tests both CLI and YAML approaches for complete cluster management workflows
# WARNING: Creates real Atlas clusters - use only in test environments

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
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/cluster-lifecycle"
RESOURCE_STATE_FILE="$TEST_REPORTS_DIR/cluster-resources.state"
REGION="${TEST_REGION:-US_EAST_1}"

# Test state tracking
declare -a CREATED_RESOURCES=()
declare -a CLEANUP_FUNCTIONS=()

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

# Track cleanup functions
track_cleanup_function() {
    local cleanup_cmd="$1"
    local description="$2"
    
    CLEANUP_FUNCTIONS+=("$cleanup_cmd # $description")
}

# Environment validation
check_environment() {
    print_info "Validating cluster lifecycle test environment..."
    
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
    
    if [[ -z "${ATLAS_ORG_ID:-}" ]]; then
        print_error "ATLAS_ORG_ID not configured"
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

# Wait for cluster to be available
wait_for_cluster_ready() {
    local cluster_name="$1"
    local max_wait_time=${2:-1800}  # 30 minutes default
    local wait_interval=30
    local elapsed_time=0
    
    print_info "Waiting for cluster '$cluster_name' to be ready (max ${max_wait_time}s)..."
    
    while [[ $elapsed_time -lt $max_wait_time ]]; do
        local cluster_status
        cluster_status=$("$PROJECT_ROOT/matlas" atlas clusters get "$cluster_name" \
            --project-id "$ATLAS_PROJECT_ID" \
            --output json 2>/dev/null | jq -r '.stateName // "UNKNOWN"' 2>/dev/null || echo "UNKNOWN")
        
        case "$cluster_status" in
            "IDLE")
                print_success "Cluster '$cluster_name' is ready"
                return 0
                ;;
            "CREATING"|"UPDATING"|"REPAIRING")
                print_info "Cluster status: $cluster_status (${elapsed_time}/${max_wait_time}s)"
                ;;
            "ERROR"|"UNKNOWN")
                print_error "Cluster in error state: $cluster_status"
                return 1
                ;;
            *)
                print_info "Cluster status: $cluster_status (${elapsed_time}/${max_wait_time}s)"
                ;;
        esac
        
        sleep $wait_interval
        ((elapsed_time += wait_interval))
    done
    
    print_error "Timeout waiting for cluster '$cluster_name' to be ready"
    return 1
}

# Wait for cluster deletion
wait_for_cluster_deleted() {
    local cluster_name="$1"
    local max_wait_time=${2:-1800}  # 30 minutes default
    local wait_interval=30
    local elapsed_time=0
    
    print_info "Waiting for cluster '$cluster_name' to be deleted (max ${max_wait_time}s)..."
    
    while [[ $elapsed_time -lt $max_wait_time ]]; do
        if ! "$PROJECT_ROOT/matlas" atlas clusters get "$cluster_name" \
            --project-id "$ATLAS_PROJECT_ID" >/dev/null 2>&1; then
            print_success "Cluster '$cluster_name' has been deleted"
            return 0
        fi
        
        print_info "Cluster still exists (${elapsed_time}/${max_wait_time}s)"
        sleep $wait_interval
        ((elapsed_time += wait_interval))
    done
    
    print_error "Timeout waiting for cluster '$cluster_name' to be deleted"
    return 1
}

# CLI-based cluster lifecycle test
test_cli_cluster_lifecycle() {
    print_header "CLI-Based Cluster Lifecycle Test"
    
    local cluster_name
    local user_name
    cluster_name="cli-test-cluster-$(date +%s)"
    user_name="cli-test-user-$(date +%s)"
    
    print_info "Testing CLI cluster lifecycle with cluster: $cluster_name"
    
    # Step 1: Create cluster using CLI
    print_subheader "Step 1: Creating cluster via CLI"
    print_info "Creating cluster '$cluster_name'..."
    
    if "$PROJECT_ROOT/matlas" atlas clusters create \
        --name "$cluster_name" \
        --project-id "$ATLAS_PROJECT_ID" \
        --provider AWS \
        --region "$REGION" \
        --tier M10 \
        --disk-size 10; then
        
        print_success "Cluster creation initiated"
        track_resource "cluster" "$cluster_name" "cli"
    else
        print_error "Failed to create cluster"
        return 1
    fi
    
    # Wait for cluster to be ready
    if ! wait_for_cluster_ready "$cluster_name"; then
        print_error "Cluster failed to become ready"
        print_warning "Cluster may still exist and will be cleaned up"
        return 1
    fi
    
    # Step 2: Create database user via CLI
    print_subheader "Step 2: Creating database user via CLI"
    print_info "Creating database user '$user_name'..."
    
    if "$PROJECT_ROOT/matlas" atlas users create \
        --username "$user_name" \
        --password "CliTestPassword123!" \
        --roles readWrite@testapp \
        --database-name admin \
        --project-id "$ATLAS_PROJECT_ID"; then
        
        print_success "Database user created"
        track_resource "user" "$user_name" "cli"
        
        # Wait a moment for user propagation
        sleep 5
        
        # Verify user exists
        if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$user_name"; then
            print_success "Database user verified"
        else
            print_warning "Database user not immediately visible"
        fi
    else
        print_error "Failed to create database user"
        print_warning "Continuing with cleanup of cluster that was created"
        return 1
    fi
    
    # Step 3: Test connection and show cluster info
    print_subheader "Step 3: Verifying cluster and user"
    
    # Show cluster details
    print_info "Cluster details:"
    "$PROJECT_ROOT/matlas" atlas clusters get "$cluster_name" --project-id "$ATLAS_PROJECT_ID" || true
    
    # Show users
    print_info "Database users:"
    "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep "$user_name" || true
    
    # Step 4: Delete database user
    print_subheader "Step 4: Deleting database user via CLI"
    print_info "Deleting database user '$user_name'..."
    
    if "$PROJECT_ROOT/matlas" atlas users delete "$user_name" \
        --project-id "$ATLAS_PROJECT_ID" \
        --database-name admin \
        --yes; then
        
        print_success "Database user deleted"
        
        # Verify user is gone
        sleep 3
        if ! "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$user_name"; then
            print_success "Database user deletion verified"
        else
            print_warning "Database user may still be cleaning up"
        fi
    else
        print_warning "Failed to delete database user via CLI"
    fi
    
    # Step 5: Delete cluster
    print_subheader "Step 5: Deleting cluster via CLI"
    print_info "Deleting cluster '$cluster_name'..."
    
    if "$PROJECT_ROOT/matlas" atlas clusters delete "$cluster_name" \
        --project-id "$ATLAS_PROJECT_ID" \
        --yes; then
        
        print_success "Cluster deletion initiated"
        
        # Wait for cluster deletion
        if wait_for_cluster_deleted "$cluster_name"; then
            print_success "CLI cluster lifecycle test completed successfully"
        else
            print_warning "Cluster deletion verification timed out"
        fi
    else
        print_error "Failed to delete cluster"
        return 1
    fi
    
    return 0
}

# YAML-based cluster lifecycle test
test_yaml_cluster_lifecycle() {
    print_header "YAML-Based Cluster Lifecycle Test"
    
    local cluster_name
    local user_name
    cluster_name="yaml-test-cluster-$(date +%s)"
    user_name="yaml-test-user-$(date +%s)"
    local config_file="$TEST_REPORTS_DIR/yaml-cluster-config.yaml"
    
    print_info "Testing YAML cluster lifecycle with cluster: $cluster_name"
    
    # Get project name for proper YAML configuration
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        print_warning "Could not get project name, using project ID"
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    # Step 1: Create configuration with cluster and user
    print_subheader "Step 1: Creating YAML configuration"
    
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: yaml-cluster-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: $cluster_name
      labels:
        test-type: yaml-lifecycle
        purpose: testing
      annotations:
        description: "Test cluster for YAML lifecycle testing"
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
        test-type: yaml-lifecycle
      annotations:
        description: "Test user for YAML lifecycle testing"
    spec:
      projectName: "$project_name"
      username: $user_name
      databaseName: admin
      password: YamlTestPassword123!
      roles:
        - roleName: readWrite
          databaseName: admin
        - roleName: read
          databaseName: admin
      scopes:
        - name: $cluster_name
          type: CLUSTER
EOF
    
    track_resource "config" "$config_file" "yaml"
    track_resource "cluster" "$cluster_name" "yaml"
    track_resource "user" "$user_name" "yaml"
    
    print_success "YAML configuration created"
    
    # Step 2: Validate configuration
    print_subheader "Step 2: Validating YAML configuration"
    
    if "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
        print_success "YAML configuration validation passed"
    else
        print_error "YAML configuration validation failed"
        return 1
    fi
    
    # Step 3: Plan infrastructure
    print_subheader "Step 3: Planning infrastructure changes"
    
    local plan_file="$TEST_REPORTS_DIR/yaml-cluster-plan.json"
    if "$PROJECT_ROOT/matlas" infra plan -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --output json > "$plan_file"; then
        
        print_success "Infrastructure planning completed"
        track_resource "plan" "$plan_file" "yaml"
        
        # Show plan summary
        print_info "Plan summary:"
        cat "$plan_file" | jq -r '.summary // "Plan details not available"' 2>/dev/null || \
            echo "Plan created successfully (details not parseable)"
    else
        print_error "Infrastructure planning failed"
        return 1
    fi
    
    # Step 4: Apply infrastructure
    print_subheader "Step 4: Applying infrastructure via YAML"
    
    print_warning "Creating real cluster - this may take 10-20 minutes and incur costs"
    
    if "$PROJECT_ROOT/matlas" infra -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Infrastructure apply initiated"
        
        # Wait for cluster to be ready
        if wait_for_cluster_ready "$cluster_name"; then
            print_success "Cluster is ready via YAML"
        else
            print_error "Cluster failed to become ready"
            return 1
        fi
    else
        print_error "Infrastructure apply failed"
        return 1
    fi
    
    # Step 5: Verify resources
    print_subheader "Step 5: Verifying created resources"
    
    # Verify cluster exists
    if "$PROJECT_ROOT/matlas" atlas clusters get "$cluster_name" \
        --project-id "$ATLAS_PROJECT_ID" >/dev/null; then
        print_success "Cluster verified via Atlas CLI"
    else
        print_error "Cluster not found via Atlas CLI"
        return 1
    fi
    
    # Verify user exists
    sleep 5  # Wait for user propagation
    if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$user_name"; then
        print_success "Database user verified via Atlas CLI"
    else
        print_warning "Database user not immediately visible"
    fi
    
    # Step 6: Show current state
    print_subheader "Step 6: Showing current infrastructure state"
    
    if "$PROJECT_ROOT/matlas" infra show \
        --project-id "$ATLAS_PROJECT_ID"; then
        print_success "Infrastructure state retrieved"
    else
        print_warning "Failed to retrieve infrastructure state"
    fi
    
    # Step 7: Destroy infrastructure
    print_subheader "Step 7: Destroying infrastructure via YAML"
    
    # Brief pause to allow any pending operations to settle
    print_info "Waiting for infrastructure to stabilize before destruction..."
    sleep 3
    
    print_info "Destroying cluster and users..."
    
    if "$PROJECT_ROOT/matlas" infra destroy -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Infrastructure destroy initiated"
        
        # Wait for cluster deletion
        if wait_for_cluster_deleted "$cluster_name"; then
            print_success "YAML cluster lifecycle test completed successfully"
        else
            print_warning "Cluster deletion verification timed out"
        fi
        
        # Verify user cleanup
        sleep 5
        if ! "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$user_name"; then
            print_success "Database user cleanup verified"
        else
            print_warning "Database user may still be cleaning up"
        fi
    else
        print_error "Infrastructure destroy failed"
        return 1
    fi
    
    return 0
}

# Test YAML with existing clusters (safe individual resource management)
test_yaml_existing_clusters() {
    print_header "YAML Individual Resource Test (With Existing Clusters)"
    
    local cluster_name
    local user_name
    cluster_name="yaml-existing-test-$(date +%s)"
    user_name="yaml-existing-user-$(date +%s)"
    local config_file="$TEST_REPORTS_DIR/yaml-existing-config.yaml"
    
    print_info "Testing YAML individual resource management with cluster: $cluster_name"
    print_info "This test verifies that YAML operations don't affect existing clusters"
    
    # Get project name for proper YAML configuration
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        print_warning "Could not get project name, using project ID"
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    # List existing clusters before test
    print_subheader "Step 1: Recording existing cluster state"
    local existing_clusters
    if existing_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null); then
        local cluster_count
        cluster_count=$(echo "$existing_clusters" | jq '. | length' 2>/dev/null || echo "0")
        print_info "Found $cluster_count existing clusters before test"
    else
        print_warning "Could not list existing clusters"
        existing_clusters="[]"
    fi
    
    # Step 2: Create YAML configuration for single cluster
    print_subheader "Step 2: Creating YAML configuration for individual resources"
    
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: yaml-existing-test
  labels:
    test-type: existing-clusters
    purpose: individual-resource-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: $cluster_name
      labels:
        test-type: yaml-existing
        purpose: testing
      annotations:
        description: "Test cluster for YAML with existing clusters"
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
        test-type: yaml-existing
      annotations:
        description: "Test user for YAML with existing clusters"
    spec:
      projectName: "$project_name"
      username: $user_name
      databaseName: admin
      password: YamlExistingPassword123!
      roles:
        - roleName: readWrite
          databaseName: testapp
        - roleName: read
          databaseName: admin
      scopes:
        - name: $cluster_name
          type: CLUSTER
EOF
    
    track_resource "config" "$config_file" "yaml-existing"
    track_resource "cluster" "$cluster_name" "yaml-existing"
    track_resource "user" "$user_name" "yaml-existing"
    
    print_success "YAML configuration created for individual resources"
    
    # Step 3: Apply the configuration
    print_subheader "Step 3: Applying YAML configuration (individual resources)"
    
    if "$PROJECT_ROOT/matlas" infra -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Individual resource apply completed"
        
        # Wait for cluster to be ready
        if wait_for_cluster_ready "$cluster_name"; then
            print_success "Test cluster is ready"
        else
            print_error "Test cluster failed to become ready"
            return 1
        fi
    else
        print_error "Individual resource apply failed"
        return 1
    fi
    
    # Step 4: Verify existing clusters are untouched
    print_subheader "Step 4: Verifying existing clusters are untouched"
    
    local current_clusters
    if current_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null); then
        local current_count
        current_count=$(echo "$current_clusters" | jq '. | length' 2>/dev/null || echo "0")
        local expected_count=$(($(echo "$existing_clusters" | jq '. | length' 2>/dev/null || echo "0") + 1))
        
        if [[ "$current_count" -eq "$expected_count" ]]; then
            print_success "Cluster count increased by exactly 1 (existing clusters safe)"
            
            # Verify our test cluster exists
            if echo "$current_clusters" | jq -r '.[].name' | grep -q "$cluster_name"; then
                print_success "Test cluster created successfully"
            else
                print_error "Test cluster not found in cluster list"
                return 1
            fi
        else
            print_error "Unexpected cluster count: expected $expected_count, found $current_count"
            return 1
        fi
    else
        print_warning "Could not verify cluster count after creation"
    fi
    
    # Step 5: Remove only the test cluster
    print_subheader "Step 5: Removing only the test cluster via YAML destroy"
    
    if "$PROJECT_ROOT/matlas" infra destroy -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Individual resource destroy completed"
        
        # Wait for cluster deletion
        if wait_for_cluster_deleted "$cluster_name"; then
            print_success "Test cluster deleted successfully"
        else
            print_warning "Test cluster deletion verification timed out"
        fi
    else
        print_error "Individual resource destroy failed"
        return 1
    fi
    
    # Step 6: Final verification that existing clusters are still intact
    print_subheader "Step 6: Final verification of existing cluster preservation"
    
    local final_clusters
    if final_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null); then
        local final_count
        final_count=$(echo "$final_clusters" | jq '. | length' 2>/dev/null || echo "0")
        local original_count
        original_count=$(echo "$existing_clusters" | jq '. | length' 2>/dev/null || echo "0")
        
        if [[ "$final_count" -eq "$original_count" ]]; then
            print_success "All existing clusters preserved - test successful!"
        else
            print_error "Cluster count mismatch: started with $original_count, ended with $final_count"
            return 1
        fi
    else
        print_warning "Could not verify final cluster state"
    fi
    
    print_success "YAML individual resource test completed successfully"
    return 0
}

# Test YAML with multiple clusters in clean project
test_yaml_multi_clusters() {
    print_header "YAML Multi-Cluster Test (Clean Project Scenario)"
    
    local cluster1_name
    local cluster2_name
    local user1_name
    local user2_name
    cluster1_name="yaml-multi-1-$(date +%s)"
    cluster2_name="yaml-multi-2-$(date +%s)"
    user1_name="yaml-multi-user1-$(date +%s)"
    user2_name="yaml-multi-user2-$(date +%s)"
    local config_file="$TEST_REPORTS_DIR/yaml-multi-config.yaml"
    
    print_info "Testing YAML multi-cluster management"
    print_info "Clusters: $cluster1_name, $cluster2_name"
    
    # Get project name
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        print_warning "Could not get project name, using project ID"
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    # Step 1: Create configuration with 2 clusters and users
    print_subheader "Step 1: Creating YAML configuration for 2 clusters"
    
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: yaml-multi-cluster-test
  labels:
    test-type: multi-cluster
    purpose: testing
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: $cluster1_name
      labels:
        test-type: yaml-multi
        cluster-role: primary
      annotations:
        description: "First test cluster for multi-cluster YAML test"
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
    kind: Cluster
    metadata:
      name: $cluster2_name
      labels:
        test-type: yaml-multi
        cluster-role: secondary
      annotations:
        description: "Second test cluster for multi-cluster YAML test"
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
      name: $user1_name
      labels:
        test-type: yaml-multi
        user-role: app
      annotations:
        description: "Application user for cluster 1"
    spec:
      projectName: "$project_name"
      username: $user1_name
      databaseName: admin
      password: YamlMulti1Password123!
      roles:
        - roleName: readWrite
          databaseName: app1
      scopes:
        - name: $cluster1_name
          type: CLUSTER
          
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: $user2_name
      labels:
        test-type: yaml-multi
        user-role: analytics
      annotations:
        description: "Analytics user for cluster 2"
    spec:
      projectName: "$project_name"
      username: $user2_name
      databaseName: admin
      password: YamlMulti2Password123!
      roles:
        - roleName: read
          databaseName: analytics
      scopes:
        - name: $cluster2_name
          type: CLUSTER
EOF
    
    track_resource "config" "$config_file" "yaml-multi"
    track_resource "cluster" "$cluster1_name" "yaml-multi"
    track_resource "cluster" "$cluster2_name" "yaml-multi"
    track_resource "user" "$user1_name" "yaml-multi"
    track_resource "user" "$user2_name" "yaml-multi"
    
    print_success "Multi-cluster YAML configuration created"
    
    # Step 2: Apply the configuration
    print_subheader "Step 2: Applying multi-cluster YAML configuration"
    
    if "$PROJECT_ROOT/matlas" infra -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Multi-cluster apply completed"
        
        # Wait for both clusters to be ready
        print_info "Waiting for clusters to be ready..."
        if wait_for_cluster_ready "$cluster1_name" && wait_for_cluster_ready "$cluster2_name"; then
            print_success "Both test clusters are ready"
        else
            print_error "One or more clusters failed to become ready"
            return 1
        fi
    else
        print_error "Multi-cluster apply failed"
        return 1
    fi
    
    # Step 3: Verify both clusters and users exist
    print_subheader "Step 3: Verifying multi-cluster resources"
    
    # Check clusters
    for cluster in "$cluster1_name" "$cluster2_name"; do
        if "$PROJECT_ROOT/matlas" atlas clusters get "$cluster" --project-id "$ATLAS_PROJECT_ID" >/dev/null; then
            print_success "Cluster $cluster verified"
        else
            print_error "Cluster $cluster not found"
            return 1
        fi
    done
    
    # Check users
    local user_list
    if user_list=$("$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null); then
        for user in "$user1_name" "$user2_name"; do
            if echo "$user_list" | grep -q "$user"; then
                print_success "User $user verified"
            else
                print_warning "User $user not immediately visible"
            fi
        done
    fi
    
    # Step 4: Remove all resources
    print_subheader "Step 4: Removing all multi-cluster resources"
    
    if "$PROJECT_ROOT/matlas" infra destroy -f "$config_file" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Multi-cluster destroy completed"
        
        # Wait for cluster deletions
        print_info "Waiting for cluster deletions..."
        local deletion_success=true
        for cluster in "$cluster1_name" "$cluster2_name"; do
            if wait_for_cluster_deleted "$cluster" 900; then
                print_success "Cluster $cluster deleted successfully"
            else
                print_warning "Cluster $cluster deletion verification timed out"
                deletion_success=false
            fi
        done
        
        if [[ "$deletion_success" == "true" ]]; then
            print_success "All clusters deleted successfully"
        else
            print_warning "Some cluster deletions may still be in progress"
        fi
    else
        print_error "Multi-cluster destroy failed"
        return 1
    fi
    
    print_success "YAML multi-cluster test completed successfully"
    return 0
}

# Test YAML partial removal (add 2, remove 1)
test_yaml_partial_removal() {
    print_header "YAML Partial Removal Test (Add 2 Clusters, Remove 1)"
    
    local cluster1_name
    local cluster2_name
    cluster1_name="yaml-partial-1-$(date +%s)"
    cluster2_name="yaml-partial-2-$(date +%s)"
    local config_file_full="$TEST_REPORTS_DIR/yaml-partial-full.yaml"
    local config_file_partial="$TEST_REPORTS_DIR/yaml-partial-remaining.yaml"
    
    print_info "Testing YAML partial resource removal"
    print_info "Will create 2 clusters, then remove 1 via updated YAML"
    
    # Get project name
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        print_warning "Could not get project name, using project ID"
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    # Step 1: Create configuration with 2 clusters
    print_subheader "Step 1: Creating YAML configuration for 2 clusters"
    
    cat > "$config_file_full" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: yaml-partial-test
  labels:
    test-type: partial-removal
    purpose: testing
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: $cluster1_name
      labels:
        test-type: yaml-partial
        keep: "true"
      annotations:
        description: "Cluster to keep in partial removal test"
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
    kind: Cluster
    metadata:
      name: $cluster2_name
      labels:
        test-type: yaml-partial
        keep: "false"
      annotations:
        description: "Cluster to remove in partial removal test"
    spec:
      projectName: "$project_name"
      provider: AWS
      region: $REGION
      instanceSize: M10
      diskSizeGB: 10
      backupEnabled: false
      mongodbVersion: "7.0"
      clusterType: REPLICASET
EOF
    
    track_resource "config" "$config_file_full" "yaml-partial"
    track_resource "config" "$config_file_partial" "yaml-partial"
    track_resource "cluster" "$cluster1_name" "yaml-partial"
    track_resource "cluster" "$cluster2_name" "yaml-partial"
    
    print_success "Full configuration created"
    
    # Step 2: Apply full configuration
    print_subheader "Step 2: Applying full configuration (2 clusters)"
    
    if "$PROJECT_ROOT/matlas" infra -f "$config_file_full" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Full configuration applied"
        
        # Wait for both clusters
        if wait_for_cluster_ready "$cluster1_name" && wait_for_cluster_ready "$cluster2_name"; then
            print_success "Both clusters are ready"
        else
            print_error "Clusters failed to become ready"
            return 1
        fi
    else
        print_error "Full configuration apply failed"
        return 1
    fi
    
    # Step 3: Create partial configuration (only cluster1)
    print_subheader "Step 3: Creating partial configuration (only 1 cluster)"
    
    cat > "$config_file_partial" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: yaml-partial-test
  labels:
    test-type: partial-removal
    purpose: testing
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: $cluster1_name
      labels:
        test-type: yaml-partial
        keep: "true"
      annotations:
        description: "Cluster to keep in partial removal test"
    spec:
      projectName: "$project_name"
      provider: AWS
      region: US_EAST_1
      instanceSize: M10
      diskSizeGB: 10
      backupEnabled: false
      mongodbVersion: "7.0"
      clusterType: REPLICASET
EOF
    
    print_success "Partial configuration created (only cluster1)"
    
    # Step 4: Apply partial configuration
    print_subheader "Step 4: Applying partial configuration (should remove cluster2)"
    
    print_warning "This tests whether YAML properly removes resources not in the new configuration"
    
    if "$PROJECT_ROOT/matlas" infra -f "$config_file_partial" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Partial configuration applied"
        
        # Verify cluster1 still exists and cluster2 is being deleted
        sleep 10  # Give time for deletion to start
        
        # Check cluster1 (should exist)
        if "$PROJECT_ROOT/matlas" atlas clusters get "$cluster1_name" --project-id "$ATLAS_PROJECT_ID" >/dev/null 2>&1; then
            print_success "Cluster1 ($cluster1_name) still exists - correct"
        else
            print_error "Cluster1 ($cluster1_name) was unexpectedly removed"
            return 1
        fi
        
        # Check cluster2 (should be deleted or deleting)
        if "$PROJECT_ROOT/matlas" atlas clusters get "$cluster2_name" --project-id "$ATLAS_PROJECT_ID" >/dev/null 2>&1; then
            print_info "Cluster2 ($cluster2_name) still exists (may be deleting)"
            # Wait for deletion to complete
            if wait_for_cluster_deleted "$cluster2_name" 900; then
                print_success "Cluster2 ($cluster2_name) was properly removed"
            else
                print_warning "Cluster2 deletion verification timed out"
            fi
        else
            print_success "Cluster2 ($cluster2_name) was properly removed"
        fi
    else
        print_error "Partial configuration apply failed"
        return 1
    fi
    
    # Step 5: Clean up remaining cluster
    print_subheader "Step 5: Cleaning up remaining cluster"
    
    if "$PROJECT_ROOT/matlas" infra destroy -f "$config_file_partial" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve; then
        
        print_success "Cleanup destroy completed"
        
        if wait_for_cluster_deleted "$cluster1_name" 900; then
            print_success "Final cleanup successful"
        else
            print_warning "Final cleanup verification timed out"
        fi
    else
        print_warning "Final cleanup failed - manual cleanup may be needed"
    fi
    
    print_success "YAML partial removal test completed successfully"
    return 0
}

# Compare CLI vs YAML approaches
test_approach_comparison() {
    print_header "CLI vs YAML Approach Comparison"
    
    print_info "Comparison Summary:"
    echo
    echo "CLI Approach:"
    echo "  ✓ Individual command control"
    echo "  ✓ Immediate feedback"
    echo "  ✓ Good for ad-hoc operations"
    echo "  ✗ No declarative state management"
    echo "  ✗ Manual dependency coordination"
    echo
    echo "YAML Approach:"
    echo "  ✓ Declarative infrastructure as code"
    echo "  ✓ Automatic dependency management"
    echo "  ✓ Version controllable configurations"
    echo "  ✓ Idempotent operations"
    echo "  ✗ Less granular control"
    echo "  ✗ Requires configuration file management"
    echo
    
    print_success "All test scenarios successfully completed"
    return 0
}

# Cleanup function
cleanup_resources() {
    print_info "Cleaning up cluster lifecycle test resources..."
    
    # Also check for clusters that might exist but weren't tracked properly
    local potential_clusters
    if potential_clusters=$("$PROJECT_ROOT/matlas" atlas clusters list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null); then
        local cli_clusters yaml_clusters
        cli_clusters=$(echo "$potential_clusters" | jq -r '.[] | select(.name | test("^cli-test-cluster-[0-9]+$")) | .name' 2>/dev/null || true)
        yaml_clusters=$(echo "$potential_clusters" | jq -r '.[] | select(.name | test("^yaml-test-cluster-[0-9]+$")) | .name' 2>/dev/null || true)
        
        # Add any found test clusters to cleanup list if they're not already tracked
        for cluster in $cli_clusters $yaml_clusters; do
            if [[ -n "$cluster" ]]; then
                local already_tracked=false
                for tracked_resource in "${CREATED_RESOURCES[@]}"; do
                    if [[ "$tracked_resource" == "cluster:$cluster:"* ]]; then
                        already_tracked=true
                        break
                    fi
                done
                if [[ "$already_tracked" == "false" ]]; then
                    print_warning "Found untracked test cluster: $cluster"
                    CREATED_RESOURCES+=("cluster:$cluster:discovered")
                fi
            fi
        done
    fi
    
    # Also check for users that might exist but weren't tracked properly
    local potential_users
    if potential_users=$("$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null); then
        local cli_users yaml_users
        cli_users=$(echo "$potential_users" | grep -E "cli-test-user-[0-9]+" | awk '{print $1}' || true)
        yaml_users=$(echo "$potential_users" | grep -E "yaml-test-user-[0-9]+" | awk '{print $1}' || true)
        
        # Add any found test users to cleanup list if they're not already tracked
        for user in $cli_users $yaml_users; do
            if [[ -n "$user" ]]; then
                local already_tracked=false
                for tracked_resource in "${CREATED_RESOURCES[@]}"; do
                    if [[ "$tracked_resource" == "user:$user:"* ]]; then
                        already_tracked=true
                        break
                    fi
                done
                if [[ "$already_tracked" == "false" ]]; then
                    print_warning "Found untracked test user: $user"
                    CREATED_RESOURCES+=("user:$user:discovered")
                fi
            fi
        done
    fi
    
    if [[ ${#CREATED_RESOURCES[@]} -eq 0 ]]; then
        print_info "No resources to clean up"
        return 0
    fi
    
    # Clean up in reverse order (LIFO) - users first, then clusters
    # First pass: delete users
    print_subheader "Cleaning up database users"
    for ((i=${#CREATED_RESOURCES[@]}-1; i>=0; i--)); do
        local resource_info="${CREATED_RESOURCES[i]}"
        IFS=':' read -r resource_type resource_name additional_info <<< "$resource_info"
        
        if [[ "$resource_type" == "user" ]]; then
            print_info "Deleting user: $resource_name"
            if "$PROJECT_ROOT/matlas" atlas users delete "$resource_name" \
                --project-id "$ATLAS_PROJECT_ID" \
                --database-name admin \
                --yes 2>/dev/null; then
                print_success "User $resource_name deleted"
            else
                print_warning "Failed to delete user $resource_name"
            fi
        fi
    done
    
    # Second pass: delete clusters
    print_subheader "Cleaning up clusters"
    for ((i=${#CREATED_RESOURCES[@]}-1; i>=0; i--)); do
        local resource_info="${CREATED_RESOURCES[i]}"
        IFS=':' read -r resource_type resource_name additional_info <<< "$resource_info"
        
        if [[ "$resource_type" == "cluster" ]]; then
            print_info "Deleting cluster: $resource_name"
            if "$PROJECT_ROOT/matlas" atlas clusters delete "$resource_name" \
                --project-id "$ATLAS_PROJECT_ID" \
                --yes 2>/dev/null; then
                print_success "Cluster deletion initiated: $resource_name"
                
                # Wait for cluster deletion to complete (with shorter timeout for cleanup)
                print_info "Waiting for cluster deletion to complete..."
                if wait_for_cluster_deleted "$resource_name" 900; then
                    print_success "Cluster $resource_name fully deleted"
                else
                    print_warning "Cluster $resource_name deletion may still be in progress"
                fi
            else
                print_warning "Failed to delete cluster $resource_name"
            fi
        fi
    done
    
    # Third pass: clean up files
    print_subheader "Cleaning up temporary files"
    for ((i=${#CREATED_RESOURCES[@]}-1; i>=0; i--)); do
        local resource_info="${CREATED_RESOURCES[i]}"
        IFS=':' read -r resource_type resource_name additional_info <<< "$resource_info"
        
        if [[ "$resource_type" == "config" || "$resource_type" == "plan" ]]; then
            print_info "Removing file: $resource_name"
            rm -f "$resource_name" 2>/dev/null || true
        fi
    done
    
    # Execute additional cleanup functions
    if [[ ${#CLEANUP_FUNCTIONS[@]} -gt 0 ]]; then
        print_subheader "Executing additional cleanup functions"
        for cleanup_func in "${CLEANUP_FUNCTIONS[@]}"; do
            print_info "Executing: ${cleanup_func#* # }"
            eval "${cleanup_func% # *}" 2>/dev/null || true
        done
    fi
    
    # Clear state file
    true > "$RESOURCE_STATE_FILE" 2>/dev/null || true
    
    print_success "Cleanup completed"
}

# Main test runner
run_cluster_lifecycle_tests() {
    local test_type="${1:-all}"
    
    print_header "MongoDB Atlas Cluster Lifecycle Tests"
    print_warning "⚠️  WARNING: These tests create real Atlas clusters and may incur costs!"
    print_warning "⚠️  Only run in dedicated test environments!"
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
            print_info "Running CLI-only cluster lifecycle tests..."
            test_cli_cluster_lifecycle || ((test_failures++))
            ;;
        "yaml")
            print_info "Running YAML-only cluster lifecycle tests..."
            test_yaml_cluster_lifecycle || ((test_failures++))
            ;;
        "yaml-existing")
            print_info "Running YAML test with existing clusters (safe individual resource management)..."
            test_yaml_existing_clusters || ((test_failures++))
            ;;
        "yaml-multi")
            print_info "Running YAML multi-cluster tests..."
            test_yaml_multi_clusters || ((test_failures++))
            ;;
        "yaml-partial")
            print_info "Running YAML partial removal tests..."
            test_yaml_partial_removal || ((test_failures++))
            ;;
        "comprehensive")
            print_info "Running comprehensive test suite (all scenarios)..."
            test_cli_cluster_lifecycle || ((test_failures++))
            echo
            test_yaml_existing_clusters || ((test_failures++))
            echo
            test_yaml_multi_clusters || ((test_failures++))
            echo
            test_yaml_partial_removal || ((test_failures++))
            echo
            test_approach_comparison || ((test_failures++))
            ;;
        "all"|*)
            print_info "Running basic cluster lifecycle test suite..."
            test_cli_cluster_lifecycle || ((test_failures++))
            echo
            test_yaml_cluster_lifecycle || ((test_failures++))
            echo
            test_approach_comparison || ((test_failures++))
            ;;
    esac
    
    echo
    if [[ $test_failures -eq 0 ]]; then
        print_success "All cluster lifecycle tests passed!"
        return 0
    else
        print_error "$test_failures cluster lifecycle test(s) failed"
        return 1
    fi
}

# Script usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  cli              Run CLI-based cluster lifecycle tests only"
    echo "  yaml             Run YAML-based cluster lifecycle tests only (legacy)"
    echo "  yaml-existing    Run YAML test with existing clusters (safe individual resources)"
    echo "  yaml-multi       Run YAML multi-cluster test (clean project scenario)"
    echo "  yaml-partial     Run YAML partial removal test (add 2, remove 1)"
    echo "  comprehensive    Run all test scenarios (recommended for validation)"
    echo "  all              Run basic test suite (CLI + legacy YAML) (default)"
    echo
    echo "Test Scenarios:"
    echo "  • Existing clusters:"
    echo "    - CLI: add cluster and delete it ✓"
    echo "    - YAML: add cluster, delete only that cluster (use yaml-existing)"
    echo "  • Clean project:"
    echo "    - CLI: add cluster and remove it ✓"
    echo "    - YAML: add 2 clusters and remove them (use yaml-multi)"
    echo "    - YAML: add 2 clusters, remove 1 (use yaml-partial)"
    echo
    echo "Environment variables required:"
    echo "  ATLAS_PUB_KEY       Atlas public API key"
    echo "  ATLAS_API_KEY       Atlas private API key"
    echo "  ATLAS_PROJECT_ID    Atlas project ID for testing"
    echo "  ATLAS_ORG_ID        Atlas organization ID"
    echo
    echo "Examples:"
    echo "  $0                     # Run basic tests (CLI + legacy YAML)"
    echo "  $0 comprehensive       # Run all test scenarios (recommended)"
    echo "  $0 yaml-existing       # Test YAML with existing clusters (safe)"
    echo "  $0 yaml-multi          # Test YAML multi-cluster management"
    echo "  $0 yaml-partial        # Test YAML partial removal"
    echo "  $0 cli                 # Test CLI approach only"
}

# Main execution
main() {
    case "${1:-all}" in
        "cli"|"yaml"|"yaml-existing"|"yaml-multi"|"yaml-partial"|"comprehensive"|"all")
            run_cluster_lifecycle_tests "$1"
            ;;
        "-h"|"--help"|"help")
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown command: $1"
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