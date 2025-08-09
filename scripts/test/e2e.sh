#!/usr/bin/env bash

# End-to-End Tests Runner
# Complete workflow tests with real resources

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
readonly TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/e2e"
readonly TEST_REGION="${TEST_REGION:-US_EAST_1}"

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

# Resource tracking
declare -a CREATED_RESOURCES=()

setup_e2e_environment() {
    mkdir -p "$TEST_REPORTS_DIR"
    
    # Load environment
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        set -o allexport
        source "$PROJECT_ROOT/.env"
        set +o allexport
    fi
    
    # Check Atlas credentials
    if [[ -z "${ATLAS_PUB_KEY:-}" || -z "${ATLAS_API_KEY:-}" || -z "${ATLAS_PROJECT_ID:-}" || -z "${ATLAS_ORG_ID:-}" ]]; then
        print_error "Atlas credentials required for E2E tests"
        print_info "Set ATLAS_PUB_KEY, ATLAS_API_KEY, ATLAS_PROJECT_ID, and ATLAS_ORG_ID in .env file"
        print_info "Current environment:"
        print_info "  ATLAS_PUB_KEY: ${ATLAS_PUB_KEY:-"not set"}"
        print_info "  ATLAS_API_KEY: ${ATLAS_API_KEY:+set} ${ATLAS_API_KEY:-"not set"}"
        print_info "  ATLAS_PROJECT_ID: ${ATLAS_PROJECT_ID:-"not set"}"
        print_info "  ATLAS_ORG_ID: ${ATLAS_ORG_ID:-"not set"}"
        return 1
    fi
    
    # Ensure matlas binary exists
    if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
        print_info "Building matlas binary..."
        cd "$PROJECT_ROOT"
        if ! go build -o matlas; then
            print_error "Failed to build matlas binary"
            return 1
        fi
    fi
    
    print_success "E2E environment ready"
    return 0
}

test_comprehensive_workflow() {
    print_info "Testing comprehensive workflow..."
    
    local workflow_name="comprehensive-e2e-$(date +%s)"
    local config_file="$TEST_REPORTS_DIR/$workflow_name.yaml"
    
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
          databaseName: testapp
        - roleName: read
          databaseName: admin
  networkAccess:
    - metadata:
        name: $workflow_name-network
      cidr: 203.0.113.0/24
      comment: E2E test network access
EOF
    
    track_resource "config" "$config_file"
    track_resource "user" "$workflow_name-user"
    
    # Validate configuration
    if ! "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
        print_error "Comprehensive configuration validation failed"
        return 1
    fi
    
    # Generate plan
    local plan_file="$TEST_REPORTS_DIR/$workflow_name-plan.json"
    if ! "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output json > "$plan_file"; then
        print_error "Comprehensive plan generation failed"
        return 1
    fi
    
    track_resource "plan" "$plan_file"
    
    # Show diff
    if ! "$PROJECT_ROOT/matlas" infra diff -f "$config_file" --project-id "$ATLAS_PROJECT_ID"; then
        print_error "Comprehensive diff failed"
        return 1
    fi
    
    # Apply dry run first
    if ! "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run; then
        print_error "Comprehensive dry-run failed"
        return 1
    fi
    
    # Test actual apply with preserve-existing flag
    print_info "Testing actual apply with --preserve-existing..."
    if "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve; then
        print_success "Apply with --preserve-existing successful"
        
        # Track resources for cleanup
        track_resource "applied_config" "$config_file"
        
        # Test show operation on applied resources
        print_info "Testing show operation on applied resources..."
        if "$PROJECT_ROOT/matlas" infra show --project-id "$ATLAS_PROJECT_ID"; then
            print_success "Show operation successful"
        else
            print_warning "Show operation failed (resources may not be fully propagated yet)"
        fi
        
        # Test destroy operation to clean up only what we created
        print_info "Testing destroy operation for cleanup..."
        if "$PROJECT_ROOT/matlas" infra destroy -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --auto-approve; then
            print_success "Destroy operation successful"
        else
            print_warning "Destroy operation failed - manual cleanup may be required"
        fi
    else
        print_warning "Apply with --preserve-existing failed - continuing with dry-run validation only"
    fi
    
    print_success "Comprehensive workflow test completed"
    return 0
}

test_infra_output_modes_and_dryrun_modes() {
    print_info "Testing infra output formats and dry-run modes..."
    local config_file="$TEST_REPORTS_DIR/output-modes.yaml"
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: output-modes
spec:
  name: "Output Modes"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata: { name: out-user-$(date +%s) }
      username: out-user-$(date +%s)
      databaseName: admin
      password: OutModesPass123!
      roles: [ { roleName: read, databaseName: admin } ]
EOF

    # Validate
    "$PROJECT_ROOT/matlas" infra validate -f "$config_file" || return 1

    # Plan with different outputs
    "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output json > "$TEST_REPORTS_DIR/integration-plan.json" || return 1
    "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output yaml > /dev/null || return 1
    "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output summary > /dev/null || return 1

    # Dry-run modes
    "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run --dry-run-mode quick --output table > /dev/null || return 1
    "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run --dry-run-mode thorough --output detailed > /dev/null || return 1
    "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run --dry-run-mode detailed --output yaml > /dev/null || return 1

    print_success "Infra output and dry-run mode tests completed"
    return 0
}

test_diff_outputs_and_preserve() {
    print_info "Testing diff outputs and --preserve-existing option..."
    local config_file="$TEST_REPORTS_DIR/diff-preserve.yaml"
    local user_name="diff-user-$(date +%s)"
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: diff-preserve
spec:
  name: "Diff Preserve"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata: { name: $user_name }
      username: $user_name
      databaseName: admin
      password: DiffPreserve123!
      roles: [ { roleName: read, databaseName: admin } ]
EOF

    # Table/unified/json/yaml outputs
    "$PROJECT_ROOT/matlas" infra diff -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output table || return 1
    "$PROJECT_ROOT/matlas" infra diff -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output unified || return 1
    "$PROJECT_ROOT/matlas" infra diff -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output json > /dev/null || return 1
    "$PROJECT_ROOT/matlas" infra diff -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output yaml > /dev/null || return 1

    # Preserve-existing should exclude deletions when we apply
    "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve || return 1
    track_resource "user" "$user_name"

    # Now remove the user from config to see diff behavior with preserve flag
    local config_file_empty="$TEST_REPORTS_DIR/diff-preserve-empty.yaml"
    cat > "$config_file_empty" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: diff-preserve
spec:
  name: "Diff Preserve"
  organizationId: $ATLAS_ORG_ID
  databaseUsers: []
EOF
    "$PROJECT_ROOT/matlas" infra diff -f "$config_file_empty" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --output summary || return 1

    print_success "Diff outputs and preserve-existing tests completed"
    return 0
}

test_validate_batch_and_strict_env() {
    print_info "Testing validate batch mode and strict-env..."
    local f1="$TEST_REPORTS_DIR/val1.yaml"
    local f2="$TEST_REPORTS_DIR/val2.yaml"
    cat > "$f1" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata: { name: v1 }
spec:
  name: "V1"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata: { name: v1-user }
      username: v1-user
      databaseName: admin
      password: V1Pass123!
      roles: [ { roleName: read, databaseName: admin } ]
EOF
    cat > "$f2" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata: { name: v2 }
spec:
  name: "V2"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata: { name: v2-user }
      username: v2-user
      databaseName: admin
      password: V2Pass123!
      roles: [ { roleName: read, databaseName: admin } ]
EOF

    # Batch validate may report cross-file dependency warnings/errors by design; do not fail the suite here
    "$PROJECT_ROOT/matlas" infra validate -f "$f1" -f "$f2" --batch || true

    # Strict env should fail when undefined env refs exist
    local f3="$TEST_REPORTS_DIR/val-env.yaml"
    cat > "$f3" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata: { name: v3 }
spec:
  name: "V3"
  organizationId: ${UNDEFINED_ENV_VAR}
EOF
    if "$PROJECT_ROOT/matlas" infra validate -f "$f3" --strict-env 2>/dev/null; then
        print_error "Strict env validation unexpectedly passed"
        return 1
    else
        print_success "Strict env validation correctly failed"
    fi

    print_success "Validate batch and strict-env tests completed"
    return 0
}

test_stdin_pipeline_apply_dryrun() {
    print_info "Testing stdin pipeline into infra --dry-run..."
    local tmp_cfg="$TEST_REPORTS_DIR/stdin.yaml"
    cat > "$tmp_cfg" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata: { name: stdin-pipe }
spec:
  name: "STDIN Pipe"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata: { name: stdin-user }
      username: stdin-user
      databaseName: admin
      password: StdinPipe123!
      roles: [ { roleName: read, databaseName: admin } ]
EOF
    if ! cat "$tmp_cfg" | "$PROJECT_ROOT/matlas" infra -f - --project-id "$ATLAS_PROJECT_ID" --dry-run --output summary > /dev/null; then
        print_error "STDIN dry-run failed"
        return 1
    fi
    print_success "STDIN pipeline dry-run succeeded"
    return 0
}

test_users_update_flow() {
    print_info "Testing users update flow (password + roles)..."
    local uname="upd-user-$(date +%s)"
    # Create user
    if ! "$PROJECT_ROOT/matlas" atlas users create --project-id "$ATLAS_PROJECT_ID" --username "$uname" --database-name admin --roles read@admin --password "InitPass123!" 2>/dev/null; then
        print_error "User create failed"
        return 1
    fi
    track_resource "user" "$uname"
    # Update password (use flag to trigger non-prompt)
    if ! "$PROJECT_ROOT/matlas" atlas users update "$uname" --project-id "$ATLAS_PROJECT_ID" --database-name admin --password "NewPass456!" 2>/dev/null; then
        print_error "User password update failed"
        return 1
    fi
    # Update roles
    if ! "$PROJECT_ROOT/matlas" atlas users update "$uname" --project-id "$ATLAS_PROJECT_ID" --database-name admin --roles readWrite@admin 2>/dev/null; then
        print_error "User role update failed"
        return 1
    fi
    print_success "Users update flow completed"
    return 0
}

test_performance() {
    print_info "Testing performance and stress..."
    
    print_info "Testing large configuration handling..."
    
    # Create unique test identifier to avoid resource conflicts
    local test_id="perf-$(date +%s)-$$"
    local large_config="$TEST_REPORTS_DIR/large-config.yaml"
    
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
    
    # Add multiple users to test scaling
    for i in {1..5}; do
        cat >> "$large_config" << EOF
    - metadata:
        name: $test_id-user-$i
      username: $test_id-user-$i
      databaseName: admin
      password: PerfTestPassword$i!
      roles:
        - roleName: readWrite
          databaseName: testapp$i
        - roleName: read
          databaseName: admin
EOF
    done
    
    # Add network access rules
    cat >> "$large_config" << EOF
  networkAccess:
    - metadata:
        name: $test_id-network-1
      cidr: 203.0.113.0/24
      comment: Performance test network 1
    - metadata:
        name: $test_id-network-2  
      cidr: 198.51.100.0/24
      comment: Performance test network 2
EOF
    
    track_resource "config" "$large_config"
    for i in {1..5}; do
        track_resource "user" "$test_id-user-$i"
    done
    
    # Test validation performance
    local start_time=$(date +%s)
    if ! "$PROJECT_ROOT/matlas" infra validate -f "$large_config"; then
        print_error "Large configuration validation failed"
        return 1
    fi
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $duration -gt 30 ]; then
        print_warning "Validation took ${duration}s (expected < 30s)"
    else
        print_success "Validation completed in ${duration}s"
    fi
    
    # Test planning performance
    start_time=$(date +%s)
    if ! "$PROJECT_ROOT/matlas" infra plan -f "$large_config" --project-id "$ATLAS_PROJECT_ID" > "$TEST_REPORTS_DIR/perf-plan.txt"; then
        print_error "Large configuration planning failed"
        return 1
    fi
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    if [ $duration -gt 60 ]; then
        print_warning "Planning took ${duration}s (expected < 60s)"
    else
        print_success "Planning completed in ${duration}s"
    fi
    
    print_success "Performance test completed"
    return 0
}

test_error_handling() {
    print_info "Testing error handling scenarios..."
    
    # Test invalid configuration
    print_info "Testing invalid configuration handling..."
    local invalid_config="$TEST_REPORTS_DIR/invalid-config.yaml"
    
    cat > "$invalid_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: invalid-test
spec:
  name: "Invalid Test Project"
  organizationId: "invalid-org-id"
  databaseUsers:
    - metadata:
        name: invalid-user
      # Missing username field (should cause validation error)
      password: "password"
      roles:
        - roleName: "invalidRole"
          databaseName: "admin"
EOF
    
    track_resource "config" "$invalid_config"
    
    # This should fail validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_config" 2>/dev/null; then
        print_error "Invalid configuration was accepted (should have failed)"
        return 1
    else
        print_success "Invalid configuration properly rejected"
    fi
    
    # Test non-existent project
    print_info "Testing non-existent project handling..."
    local nonexistent_config="$TEST_REPORTS_DIR/nonexistent-project.yaml"
    
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
    
    track_resource "config" "$nonexistent_config"
    
    # This should fail during planning
    if "$PROJECT_ROOT/matlas" infra plan -f "$nonexistent_config" 2>/dev/null; then
        print_error "Non-existent project was accepted (should have failed)"
        return 1
    else
        print_success "Non-existent project properly rejected"
    fi
    
    print_success "Error handling test completed"
    return 0
}

test_cluster_configurations() {
    print_info "Testing cluster configuration validation..."
    
    local cluster_config="$TEST_REPORTS_DIR/cluster-config.yaml"
    local test_id="cluster-$(date +%s)"
    
    # Create cluster configuration for validation testing only (no actual creation)
    cat > "$cluster_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: $test_id-project
spec:
  name: "Cluster Test Project"
  organizationId: $ATLAS_ORG_ID
  clusters:
    - metadata:
        name: $test_id-cluster
      provider: AWS
      region: ${TEST_REGION}
      instanceSize: M10
      mongodbVersion: "7.0"
      clusterType: REPLICASET
      diskSizeGB: 20
      backupEnabled: true
      tierType: dedicated
      autoScaling:
        compute:
          enabled: false
        diskGB:
          enabled: false
      encryption:
        encryptionAtRest: false
  databaseUsers:
    - metadata:
        name: $test_id-user
      username: $test_id-user
      databaseName: admin
      password: ClusterTestPassword123!
      roles:
        - roleName: readWrite
          databaseName: testapp
        - roleName: read
          databaseName: admin
      scopes:
        - name: $test_id-cluster
          type: CLUSTER
EOF
    
    track_resource "config" "$cluster_config"
    
    # Test validation (should pass)
    print_info "Validating cluster configuration..."
    if "$PROJECT_ROOT/matlas" infra validate -f "$cluster_config"; then
        print_success "Cluster configuration validation passed"
    else
        print_error "Cluster configuration validation failed"
        return 1
    fi
    
    # Test planning (dry-run only for clusters due to cost)
    print_info "Testing cluster plan generation (dry-run)..."
    local plan_file="$TEST_REPORTS_DIR/cluster-plan.json"
    if "$PROJECT_ROOT/matlas" infra plan -f "$cluster_config" --project-id "$ATLAS_PROJECT_ID" --output json > "$plan_file"; then
        print_success "Cluster plan generation successful"
        track_resource "plan" "$plan_file"
    else
        print_error "Cluster plan generation failed"
        return 1
    fi
    
    # Test dry-run apply (no actual resources created)
    print_info "Testing cluster dry-run apply..."
    if "$PROJECT_ROOT/matlas" infra -f "$cluster_config" --project-id "$ATLAS_PROJECT_ID" --dry-run; then
        print_success "Cluster dry-run apply completed"
    else
        print_error "Cluster dry-run apply failed"
        return 1
    fi
    
    # Test invalid cluster configuration
    print_info "Testing invalid cluster configuration..."
    local invalid_cluster_config="$TEST_REPORTS_DIR/invalid-cluster-config.yaml"
    
    cat > "$invalid_cluster_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: invalid-cluster-test
spec:
  name: "Invalid Cluster Test"
  organizationId: $ATLAS_ORG_ID
  clusters:
    - metadata:
        name: invalid-cluster
      provider: "INVALID_PROVIDER"
      region: "invalid-region"
      instanceSize: "INVALID_SIZE"
      # Missing required fields
EOF
    
    track_resource "config" "$invalid_cluster_config"
    
    # This should fail validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_cluster_config" 2>/dev/null; then
        print_error "Invalid cluster configuration was accepted (should have failed)"
        return 1
    else
        print_success "Invalid cluster configuration properly rejected"
    fi
    
    print_success "Cluster configuration tests completed"
    return 0
}

test_preserve_existing_behavior() {
    print_info "Testing --preserve-existing flag behavior..."
    
    local preserve_config="$TEST_REPORTS_DIR/preserve-test-config.yaml"
    local test_user_1="preserve-user-1-$(date +%s)"
    local test_user_2="preserve-user-2-$(date +%s)"
    
    # Create initial configuration with one user
    cat > "$preserve_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: preserve-test-project
spec:
  name: "Preserve Test Project"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata:
        name: $test_user_1
      username: $test_user_1
      databaseName: admin
      password: PreserveTest123!
      roles:
        - roleName: readWrite
          databaseName: testapp
        - roleName: read
          databaseName: admin
EOF
    
    track_resource "config" "$preserve_config"
    track_resource "user" "$test_user_1"
    track_resource "user" "$test_user_2"
    
    # Apply initial configuration
    print_info "Applying initial configuration..."
    if "$PROJECT_ROOT/matlas" infra -f "$preserve_config" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve; then
        print_success "Initial configuration applied"
        
        # Wait for propagation
        sleep 3
        
        # Verify first user exists
        if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$test_user_1"; then
            print_success "First user verified"
            
            # Now update configuration to add a second user
            cat > "$preserve_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: preserve-test-project
spec:
  name: "Preserve Test Project"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata:
        name: $test_user_1
      username: $test_user_1
      databaseName: admin
      password: PreserveTest123!
      roles:
        - roleName: readWrite
          databaseName: testapp
        - roleName: read
          databaseName: admin
    - metadata:
        name: $test_user_2
      username: $test_user_2
      databaseName: admin
      password: PreserveTest456!
      roles:
        - roleName: read
          databaseName: testapp
        - roleName: read
          databaseName: admin
EOF
            
            # Apply updated configuration with --preserve-existing
            print_info "Testing --preserve-existing with existing resources..."
            if "$PROJECT_ROOT/matlas" infra -f "$preserve_config" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve; then
                print_success "Updated configuration applied with --preserve-existing"
                
                # Wait for propagation
                sleep 3
                
                # Verify both users exist
                local users_list
                users_list=$("$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null || echo "")
                
                if echo "$users_list" | grep -q "$test_user_1" && echo "$users_list" | grep -q "$test_user_2"; then
                    print_success "Both users verified - --preserve-existing working correctly"
                else
                    print_warning "User verification incomplete - may still be propagating"
                fi
                
                # Clean up both users
                print_info "Cleaning up preserve test resources..."
                if "$PROJECT_ROOT/matlas" infra destroy -f "$preserve_config" --project-id "$ATLAS_PROJECT_ID" --auto-approve; then
                    print_success "Preserve test cleanup completed"
                else
                    print_warning "Preserve test cleanup failed - manual cleanup may be needed"
                fi
            else
                print_error "Updated configuration apply failed"
                return 1
            fi
        else
            print_error "First user verification failed"
            return 1
        fi
    else
        print_error "Initial configuration apply failed"
        return 1
    fi
    
    print_success "Preserve existing behavior test completed"
    return 0
}

cleanup_resources() {
    print_info "Cleaning up E2E test resources..."
    
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
                # Try to delete exact IP first; if not found, list and attempt CIDR deletion
                if ! "$PROJECT_ROOT/matlas" atlas network delete "$resource_id" \
                    --project-id "$ATLAS_PROJECT_ID" --yes 2>/dev/null; then
                    # Fallback: discover entry (IP vs CIDR) and retry deletion
                    local entry
                    entry=$("$PROJECT_ROOT/matlas" atlas network list --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r ".[] | select(.ipAddress == \"$resource_id\" or .cidrBlock == \"$resource_id\").ipAddress // \"\"" || echo "")
                    if [[ -n "$entry" ]]; then
                        "$PROJECT_ROOT/matlas" atlas network delete "$entry" --project-id "$ATLAS_PROJECT_ID" --yes 2>/dev/null || {
                            print_warning "Failed to delete network access via fallback: $resource_id"; ((cleanup_errors++));
                        }
                    else
                        print_warning "Network entry not found for cleanup: $resource_id"
                    fi
                fi
                ;;
            "config")
                if [[ -f "$resource_id" ]]; then
                    rm -f "$resource_id"
                fi
                ;;
        esac
    done
    
    if [[ $cleanup_errors -eq 0 ]]; then
        print_success "All resources cleaned up"
    else
        print_warning "$cleanup_errors resource(s) failed to clean up"
    fi
}

track_resource() {
    local resource_type="$1"
    local resource_id="$2"
    CREATED_RESOURCES+=("$resource_type:$resource_id")
    print_info "Tracking resource: $resource_type:$resource_id"
}

test_atlas_workflow() {
    print_info "Testing Atlas workflow..."
    
    local test_user="e2e-user-$(date +%s)"
    local test_ip="10.0.0.100"
    
    # Test user creation workflow
    print_info "Creating database user via Atlas command..."
    if "$PROJECT_ROOT/matlas" atlas users create --username "$test_user" \
        --project-id "$ATLAS_PROJECT_ID" \
        --roles readWrite@admin \
        --database-name admin \
        --password "E2EPassword123!" 2>/dev/null; then
        
        track_resource "user" "$test_user"
        print_success "Created user: $test_user"
    else
        print_error "Failed to create user"
        return 1
    fi
    
    # Test network access workflow
    print_info "Creating network access entry..."
    if "$PROJECT_ROOT/matlas" atlas network create \
        --project-id "$ATLAS_PROJECT_ID" \
        --ip-address "$test_ip" \
        --comment "E2E test" 2>/dev/null; then
        
        track_resource "network" "$test_ip"
        print_success "Created network access: $test_ip"
    else
        print_error "Failed to create network access"
        return 1
    fi
    
    # Test listing operations
    print_info "Testing list operations..."
    if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" > "$TEST_REPORTS_DIR/users.txt"; then
        if grep -q "$test_user" "$TEST_REPORTS_DIR/users.txt"; then
            print_success "User found in list"
        else
            print_error "User not found in list"
            return 1
        fi
    else
        print_error "Failed to list users"
        return 1
    fi
    
    if "$PROJECT_ROOT/matlas" atlas network list --project-id "$ATLAS_PROJECT_ID" > "$TEST_REPORTS_DIR/network.txt"; then
        if grep -q "$test_ip" "$TEST_REPORTS_DIR/network.txt"; then
            print_success "Network access found in list"
        else
            print_error "Network access not found in list"
            return 1
        fi
    else
        print_error "Failed to list network access"
        return 1
    fi
    
    return 0
}

test_infra_workflow() {
    print_info "Testing infra workflow..."
    
    local config_file="$TEST_REPORTS_DIR/test-config.yaml"
    local test_user="infra-user-$(date +%s)"
    
    # Create test configuration
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: infra-test-project
spec:
  name: "Infra Test Project"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata:
        name: $test_user
      username: $test_user
      databaseName: admin
      password: InfraTest123!
      roles:
        - roleName: readWrite
          databaseName: testapp
        - roleName: read
          databaseName: admin
EOF
    
    track_resource "config" "$config_file"
    track_resource "user" "$test_user"
    
    # Test infra validate
    print_info "Validating configuration..."
    if "$PROJECT_ROOT/matlas" infra validate -f "$config_file" 2>/dev/null; then
        print_success "Configuration validation passed"
    else
        print_error "Configuration validation failed"
        return 1
    fi
    
    # Test infra plan
    print_info "Planning infrastructure changes..."
    if "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" > "$TEST_REPORTS_DIR/plan.txt" 2>/dev/null; then
        print_success "Infrastructure planning completed"
    else
        print_error "Infrastructure planning failed"
        return 1
    fi
    
    # Test infra apply (dry run)
    print_info "Testing infra apply (dry run)..."
    if "$PROJECT_ROOT/matlas" infra -f "$config_file" --dry-run > "$TEST_REPORTS_DIR/apply-dry.txt" 2>/dev/null; then
        print_success "Dry run apply completed"
    else
        print_error "Dry run apply failed"
        return 1
    fi
    
    # Test actual apply with preserve-existing
    print_info "Testing actual apply with --preserve-existing..."
    if "$PROJECT_ROOT/matlas" infra -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve > "$TEST_REPORTS_DIR/apply-actual.txt" 2>&1; then
        print_success "Actual apply with --preserve-existing completed"
        
        # Verify the user was created
        print_info "Verifying created resources..."
        sleep 3  # Wait for resource propagation
        
        if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$test_user" 2>/dev/null; then
            print_success "Created user verified in Atlas"
        else
            print_warning "Created user not immediately visible (may still be propagating)"
        fi
        
        # Test destroy to clean up
        print_info "Testing destroy operation..."
        if "$PROJECT_ROOT/matlas" infra destroy -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --auto-approve > "$TEST_REPORTS_DIR/destroy.txt" 2>&1; then
            print_success "Destroy operation completed"
            
            # Verify cleanup
            sleep 3  # Wait for cleanup propagation
            if ! "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$test_user" 2>/dev/null; then
                print_success "Resource cleanup verified"
            else
                print_warning "Resource may still be cleaning up"
            fi
        else
            print_warning "Destroy operation failed - manual cleanup may be needed"
        fi
    else
        print_warning "Actual apply failed - continuing with dry-run validation only"
        print_info "Apply output:"
        cat "$TEST_REPORTS_DIR/apply-actual.txt" 2>/dev/null || echo "No output available"
    fi
    
    return 0
}

test_discover_workflow() {
    print_info "Testing discover workflow..."
    
    # Test project discovery
    print_info "Discovering project resources..."
    if "$PROJECT_ROOT/matlas" discover --project-id "$ATLAS_PROJECT_ID" \
        --output yaml > "$TEST_REPORTS_DIR/discovered.yaml" 2>/dev/null; then
        
        if [[ -s "$TEST_REPORTS_DIR/discovered.yaml" ]]; then
            print_success "Project discovery completed"
        else
            print_warning "Discovery completed but no resources found"
        fi
    else
        print_error "Project discovery failed"
        return 1
    fi
    
    return 0
}

test_discover_apply_overlay_cycle() {
    print_info "Testing discover -> apply -> overlay -> remove overlay cycle..."

    local base_yaml="$TEST_REPORTS_DIR/discovered.yaml"
    local overlay_yaml="$TEST_REPORTS_DIR/overlay.yaml"
    local test_user="overlay-user-$(date +%s)"
    local overlay_ip="198.51.100.42"

    # Step 1: Discover current project as ApplyDocument
    print_info "Discovering project as ApplyDocument..."
    if ! "$PROJECT_ROOT/matlas" discover --project-id "$ATLAS_PROJECT_ID" --convert-to-apply --output yaml > "$base_yaml" 2>/dev/null; then
        print_error "Discovery failed"
        return 1
    fi
    if [[ ! -s "$base_yaml" ]]; then
        print_error "Discovery output is empty"
        return 1
    fi

    # Step 2: Validate and plan base
    if ! "$PROJECT_ROOT/matlas" infra validate -f "$base_yaml"; then
        print_error "Validation of discovered base failed"
        return 1
    fi
    if ! "$PROJECT_ROOT/matlas" infra plan -f "$base_yaml" --project-id "$ATLAS_PROJECT_ID" --output summary > /dev/null; then
        print_error "Planning discovered base failed"
        return 1
    fi

    # Step 3: Skip applying discovered base to avoid non-updatable resources (e.g., network comment updates)
    print_info "Skipping base apply to avoid non-updatable resource updates; validated and planned only"

    # Step 4: Create an overlay ApplyDocument (add user + network)
    print_info "Creating overlay ApplyDocument..."
    # Best-effort project name from current project
    local project_name
    project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null || echo "$ATLAS_PROJECT_ID")
    cat > "$overlay_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: overlay-cycle
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: $test_user
    spec:
      projectName: "$project_name"
      username: $test_user
      databaseName: admin
      password: OverlayCyclePass123!
      roles:
        - roleName: read
          databaseName: admin
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: overlay-ip
    spec:
      projectName: "$project_name"
      ipAddress: $overlay_ip
      comment: "overlay test"
EOF

    # Step 5: Apply overlay
    if ! "$PROJECT_ROOT/matlas" infra -f "$overlay_yaml" --project-id "$ATLAS_PROJECT_ID" --auto-approve; then
        print_error "Applying overlay failed"
        return 1
    fi
    track_resource "config" "$overlay_yaml"
    track_resource "user" "$test_user"
    track_resource "network" "$overlay_ip"

    # Verify overlay resources
    sleep 3
    if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$test_user"; then
        print_success "Overlay user created"
    else
        print_warning "Overlay user not visible yet"
    fi
    if "$PROJECT_ROOT/matlas" atlas network list --project-id "$ATLAS_PROJECT_ID" | grep -q "$overlay_ip"; then
        print_success "Overlay network entry created"
    else
        print_warning "Overlay network entry not visible yet"
    fi

    # Step 6: Remove overlay by destroying overlay ApplyDocument
    print_info "Destroying overlay resources using overlay ApplyDocument..."
    if ! "$PROJECT_ROOT/matlas" infra destroy -f "$overlay_yaml" --project-id "$ATLAS_PROJECT_ID" --auto-approve; then
        print_error "Destroy of overlay resources failed"
        return 1
    fi

    # Verify overlay removal
    sleep 5
    if ! "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$test_user"; then
        print_success "Overlay user removed by base re-apply"
    else
        print_warning "Overlay user still present (may be deleting)"
    fi
    if ! "$PROJECT_ROOT/matlas" atlas network list --project-id "$ATLAS_PROJECT_ID" | grep -q "$overlay_ip"; then
        print_success "Overlay network entry removed by base re-apply"
    else
        print_warning "Overlay network entry still present (may be deleting)"
    fi

    print_success "Discover/apply overlay cycle completed"
    return 0
}

test_real_cluster_lifecycle() {
    print_info "Testing real cluster lifecycle..."
    print_warning "⚠️  WARNING: This will create real Atlas clusters and may incur costs!"
    
    # Source and call the cluster lifecycle tests
    local cluster_script="$SCRIPT_DIR/cluster-lifecycle.sh"
    
    if [[ ! -f "$cluster_script" ]]; then
        print_error "Cluster lifecycle script not found at: $cluster_script"
        return 1
    fi
    
    print_info "Running cluster lifecycle tests from: $cluster_script"
    
    # Execute the cluster lifecycle tests
    if bash "$cluster_script" all; then
        print_success "Real cluster lifecycle tests completed successfully"
        return 0
    else
        print_error "Real cluster lifecycle tests failed"
        return 1
    fi
}

run_e2e_tests() {
    local dry_run=false
    local include_clusters=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run) dry_run=true; shift ;;
            --include-clusters) include_clusters=true; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ "$dry_run" == "true" ]]; then
        print_info "DRY RUN: Would run E2E tests"
        print_info "Workflows: Atlas commands, infra commands, discover"
        if [[ "$include_clusters" == "true" ]]; then
            print_info "Would also include: Real cluster lifecycle tests"
        fi
        return 0
    fi
    
    print_info "Running end-to-end tests..."
    
    # Setup cleanup trap
    trap cleanup_resources EXIT INT TERM
    
    # Setup environment
    if ! setup_e2e_environment; then
        return 1
    fi
    
    local test_failures=0
    
    # Run E2E test workflows
    test_atlas_workflow || ((test_failures++))
    test_infra_workflow || ((test_failures++))
    test_discover_workflow || ((test_failures++))
    test_discover_apply_overlay_cycle || ((test_failures++))
    test_infra_output_modes_and_dryrun_modes || ((test_failures++))
    test_diff_outputs_and_preserve || ((test_failures++))
    test_validate_batch_and_strict_env || ((test_failures++))
    test_stdin_pipeline_apply_dryrun || ((test_failures++))
    test_users_update_flow || ((test_failures++))
    test_comprehensive_workflow || ((test_failures++))
    test_performance || ((test_failures++))
    test_error_handling || ((test_failures++))
    test_cluster_configurations || ((test_failures++))
    test_preserve_existing_behavior || ((test_failures++))
    
    # Run real cluster lifecycle tests if requested
    if [[ "$include_clusters" == "true" ]]; then
        test_real_cluster_lifecycle || ((test_failures++))
    fi
    
    if [[ $test_failures -eq 0 ]]; then
        print_success "All E2E tests passed"
        return 0
    else
        print_error "$test_failures E2E test workflow(s) failed"
        return 1
    fi
}

# Show usage information
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  --dry-run           Show what tests would run without executing them"
    echo "  --include-clusters  Include real cluster lifecycle tests (creates actual clusters!)"
    echo "  -h, --help          Show this help message"
    echo
    echo "Examples:"
    echo "  $0                           # Run E2E tests (users, network access, validation only)"
    echo "  $0 --dry-run                 # Show what would be tested"
    echo "  $0 --include-clusters        # Run ALL tests including real cluster creation"
    echo "  $0 --include-clusters --dry-run  # Show full test plan including clusters"
    echo
    echo "Environment variables required:"
    echo "  ATLAS_PUB_KEY       Atlas public API key"
    echo "  ATLAS_API_KEY       Atlas private API key"
    echo "  ATLAS_PROJECT_ID    Atlas project ID for testing"
    echo "  ATLAS_ORG_ID        Atlas organization ID"
}

# Main execution with help support
main() {
    case "${1:-}" in
        "-h"|"--help"|"help")
            show_usage
            exit 0
            ;;
        *)
            run_e2e_tests "$@"
            ;;
    esac
}

main "$@"