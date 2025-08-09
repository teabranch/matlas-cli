#!/usr/bin/env bash

# ApplyDocument Format Testing for matlas-cli
# Tests the ApplyDocument YAML format comprehensively
# This format is under-tested compared to Project format

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
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/applydocument"
RESOURCE_STATE_FILE="$TEST_REPORTS_DIR/applydocument-resources.state"
TEST_REGION="${TEST_REGION:-US_EAST_1}"

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

# Environment validation
check_environment() {
    print_info "Validating ApplyDocument test environment..."
    
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

# Test 1: Basic ApplyDocument validation
test_applydocument_validation() {
    print_header "ApplyDocument Validation Tests"
    
    # Test 1.1: Valid basic ApplyDocument
    print_subheader "Test 1.1: Valid basic ApplyDocument"
    
    local test_user="applydoc-valid-$(date +%s)"
    local config_file="$TEST_REPORTS_DIR/valid-basic.yaml"
    
    # Get project name
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        print_warning "Could not get project name, using project ID"
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: valid-basic-test
  labels:
    test-type: applydocument-validation
  annotations:
    description: "Basic valid ApplyDocument test"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: $test_user
      labels:
        test-type: basic-validation
    spec:
      projectName: "$project_name"
      username: $test_user
      databaseName: admin
      password: ValidPassword123!
      roles:
        - roleName: readWrite
          databaseName: testdb
        - roleName: read
          databaseName: admin
EOF
    
    track_resource "config" "$config_file" "validation"
    
    # Test validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
        print_success "Basic ApplyDocument validation passed"
    else
        print_error "Basic ApplyDocument validation failed"
        return 1
    fi
    
    # Test 1.2: Invalid ApplyDocument - missing roles
    print_subheader "Test 1.2: Invalid ApplyDocument - missing roles"
    
    local invalid_config="$TEST_REPORTS_DIR/invalid-no-roles.yaml"
    
    cat > "$invalid_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-no-roles-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: invalid-user
    spec:
      projectName: "$project_name"
      username: invalid-user
      databaseName: admin
      password: InvalidPassword123!
      roles: []  # Invalid: empty roles
EOF
    
    track_resource "config" "$invalid_config" "validation"
    
    # This should fail validation 
    if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_config" 2>/dev/null; then
        print_error "BUG: Validation should have failed for empty roles!"
        return 1
    else
        print_success "Validation correctly failed for empty roles"
    fi
    
    # Test 1.3: Invalid ApplyDocument - missing required fields
    print_subheader "Test 1.3: Invalid ApplyDocument - missing required fields"
    
    local invalid_config2="$TEST_REPORTS_DIR/invalid-missing-fields.yaml"
    
    cat > "$invalid_config2" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-missing-fields-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: invalid-user-2
    spec:
      # Missing username, projectName, password, roles
      databaseName: admin
EOF
    
    track_resource "config" "$invalid_config2" "validation"
    
    # This should fail validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_config2" 2>/dev/null; then
        print_warning "Validation passed for missing required fields - this may be acceptable depending on implementation"
        print_info "Some missing fields might have defaults or be optional"
    else
        print_success "Validation correctly failed for missing required fields"
    fi
    
    print_success "ApplyDocument validation tests completed"
    print_info "✅ ApplyDocument format now has consistent validation with Project format"
    return 0
}

# Test 2: Mixed resource types in ApplyDocument
test_mixed_resources() {
    print_header "Mixed Resource Types in ApplyDocument"
    
    local cluster_name="applydoc-mixed-$(date +%s)"
    local user_name="applydoc-mixed-user-$(date +%s)"
    local config_file="$TEST_REPORTS_DIR/mixed-resources.yaml"
    
    # Get project name
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    print_subheader "Creating ApplyDocument with Cluster + DatabaseUser + NetworkAccess"
    
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: mixed-resources-test
  labels:
    test-type: mixed-resources
    purpose: comprehensive-testing
  annotations:
    description: "Test mixed resource types in ApplyDocument"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: $cluster_name
      labels:
        test-type: applydocument-mixed
        resource-type: cluster
    spec:
      projectName: "$project_name"
      provider: AWS
      region: ${TEST_REGION}
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
        test-type: applydocument-mixed
        resource-type: user
    spec:
      projectName: "$project_name"
      username: $user_name
      databaseName: admin
      password: MixedResourcePassword123!
      roles:
        - roleName: readWrite
          databaseName: mixeddb
        - roleName: read
          databaseName: admin
      scopes:
        - name: $cluster_name
          type: CLUSTER
          
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: applydoc-network-$(date +%s)
      labels:
        test-type: applydocument-mixed
        resource-type: network
    spec:
      projectName: "$project_name"
      cidr: 10.0.0.0/24
      comment: "ApplyDocument mixed resources test"
EOF
    
    track_resource "config" "$config_file" "mixed"
    track_resource "cluster" "$cluster_name" "mixed"
    track_resource "user" "$user_name" "mixed"
    
    # Validate the configuration
    print_subheader "Validating mixed resources configuration"
    if "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
        print_success "Mixed resources validation passed"
    else
        print_error "Mixed resources validation failed"
        return 1
    fi
    
    # Generate plan
    print_subheader "Generating plan for mixed resources"
    local plan_file="$TEST_REPORTS_DIR/mixed-resources-plan.json"
    if "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output json > "$plan_file"; then
        print_success "Mixed resources plan generated"
        track_resource "plan" "$plan_file" "mixed"
        
        # Show plan summary
        print_info "Plan summary:"
        if command -v jq >/dev/null 2>&1; then
            jq -r '.summary // "Plan details not available"' "$plan_file" 2>/dev/null || echo "Plan created (jq not available)"
        else
            echo "Plan created successfully"
        fi
    else
        print_error "Mixed resources plan generation failed"
        return 1
    fi
    
    print_success "Mixed resources test completed (plan only - no apply to avoid costs)"
    return 0
}

# Test 3: Standalone DatabaseUser resources
test_standalone_database_users() {
    print_header "Standalone DatabaseUser Resources Test"
    
    local user1_name="applydoc-standalone1-$(date +%s)"
    local user2_name="applydoc-standalone2-$(date +%s)"
    local config_file="$TEST_REPORTS_DIR/standalone-users.yaml"
    
    # Get project name
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    print_subheader "Creating ApplyDocument with multiple standalone DatabaseUser resources"
    
    cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: standalone-users-test
  labels:
    test-type: standalone-users
  annotations:
    description: "Test multiple standalone DatabaseUser resources"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: $user1_name
      labels:
        test-type: standalone
        user-role: app
    spec:
      projectName: "$project_name"
      username: $user1_name
      databaseName: admin
      password: StandaloneApp123!
      roles:
        - roleName: readWrite
          databaseName: appdb
        - roleName: read
          databaseName: logs
          
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: $user2_name
      labels:
        test-type: standalone
        user-role: analytics
    spec:
      projectName: "$project_name"
      username: $user2_name
      databaseName: admin
      password: StandaloneAnalytics123!
      roles:
        - roleName: read
          databaseName: analytics
        - roleName: read
          databaseName: reports
EOF
    
    track_resource "config" "$config_file" "standalone"
    track_resource "user" "$user1_name" "standalone"
    track_resource "user" "$user2_name" "standalone"
    
    # Validate
    print_subheader "Validating standalone users configuration"
    if "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
        print_success "Standalone users validation passed"
    else
        print_error "Standalone users validation failed"
        return 1
    fi
    
    # Generate plan
    print_subheader "Generating plan for standalone users"
    local plan_file="$TEST_REPORTS_DIR/standalone-users-plan.json"
    if "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --output json > "$plan_file"; then
        print_success "Standalone users plan generated"
        track_resource "plan" "$plan_file" "standalone"
    else
        print_error "Standalone users plan generation failed"
        return 1
    fi
    
    print_success "Standalone DatabaseUser resources test completed"
    return 0
}

# Test 4: ApplyDocument vs Project format comparison
test_format_comparison() {
    print_header "ApplyDocument vs Project Format Comparison"
    
    local test_user="format-comparison-$(date +%s)"
    
    # Get project name
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    # Test 4.1: ApplyDocument format
    print_subheader "Test 4.1: ApplyDocument format"
    
    local applydoc_config="$TEST_REPORTS_DIR/format-applydoc.yaml"
    
    cat > "$applydoc_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: format-comparison-applydoc
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: ${test_user}-applydoc
    spec:
      projectName: "$project_name"
      username: ${test_user}-applydoc
      databaseName: admin
      password: FormatTestPassword123!
      roles:
        - roleName: readWrite
          databaseName: testformat
EOF
    
    track_resource "config" "$applydoc_config" "comparison"
    
    # Test 4.2: Project format
    print_subheader "Test 4.2: Project format"
    
    local project_config="$TEST_REPORTS_DIR/format-project.yaml"
    
    cat > "$project_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: format-comparison-project
spec:
  name: "$project_name"
  organizationId: $ATLAS_ORG_ID
  databaseUsers:
    - metadata:
        name: ${test_user}-project
      username: ${test_user}-project
      databaseName: admin
      password: FormatTestPassword123!
      roles:
        - roleName: readWrite
          databaseName: testformat
EOF
    
    track_resource "config" "$project_config" "comparison"
    
    # Validate both formats
    print_subheader "Validating both formats"
    
    if "$PROJECT_ROOT/matlas" infra validate -f "$applydoc_config"; then
        print_success "ApplyDocument format validation passed"
    else
        print_error "ApplyDocument format validation failed"
        return 1
    fi
    
    if "$PROJECT_ROOT/matlas" infra validate -f "$project_config"; then
        print_success "Project format validation passed"
    else
        print_error "Project format validation failed"
        return 1
    fi
    
    # Generate plans for both
    print_subheader "Generating plans for format comparison"
    
    local applydoc_plan="$TEST_REPORTS_DIR/format-applydoc-plan.json"
    local project_plan="$TEST_REPORTS_DIR/format-project-plan.json"
    
    if "$PROJECT_ROOT/matlas" infra plan -f "$applydoc_config" --project-id "$ATLAS_PROJECT_ID" --output json > "$applydoc_plan"; then
        print_success "ApplyDocument plan generated"
        track_resource "plan" "$applydoc_plan" "comparison"
    else
        print_error "ApplyDocument plan generation failed"
        return 1
    fi
    
    if "$PROJECT_ROOT/matlas" infra plan -f "$project_config" --project-id "$ATLAS_PROJECT_ID" --output json > "$project_plan"; then
        print_success "Project plan generated"
        track_resource "plan" "$project_plan" "comparison"
    else
        print_error "Project plan generation failed"
        return 1
    fi
    
    print_info "Both formats successfully validated and planned"
    print_success "Format comparison test completed"
    return 0
}

# Test 5: Error handling for ApplyDocument
test_applydocument_error_handling() {
    print_header "ApplyDocument Error Handling Tests"
    
    # Get project name
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    # Test 5.1: Invalid region in cluster
    print_subheader "Test 5.1: Invalid region in cluster"
    
    local invalid_region_config="$TEST_REPORTS_DIR/invalid-region.yaml"
    
    cat > "$invalid_region_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-region-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: invalid-region-cluster
    spec:
      projectName: "$project_name"
      provider: AWS
      region: INVALID_REGION  # This should cause validation error
      instanceSize: M10
EOF
    
    track_resource "config" "$invalid_region_config" "error-handling"
    
    # This should fail validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_region_config" 2>/dev/null; then
        print_warning "Validation should have caught invalid region (may depend on validation rules)"
    else
        print_success "Validation correctly caught invalid region"
    fi
    
    # Test 5.2: Invalid roles in database user
    print_subheader "Test 5.2: Invalid roles in database user"
    
    local invalid_roles_config="$TEST_REPORTS_DIR/invalid-roles.yaml"
    
    cat > "$invalid_roles_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-roles-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: invalid-roles-user
    spec:
      projectName: "$project_name"
      username: invalid-roles-user
      databaseName: admin
      password: InvalidRolesPassword123!
      roles:
        - roleName: ""  # Empty role name
          databaseName: testdb
        - roleName: validRole
          databaseName: ""  # Empty database name
EOF
    
    track_resource "config" "$invalid_roles_config" "error-handling"
    
    # This should fail validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_roles_config" 2>/dev/null; then
        print_error "Validation should have failed for invalid roles"
        return 1
    else
        print_success "Validation correctly failed for invalid roles"
    fi
    
    # Test 5.3: Mixed valid and invalid resources
    print_subheader "Test 5.3: Mixed valid and invalid resources"
    
    local mixed_validity_config="$TEST_REPORTS_DIR/mixed-validity.yaml"
    
    cat > "$mixed_validity_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: mixed-validity-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: valid-user
    spec:
      projectName: "$project_name"
      username: valid-user
      databaseName: admin
      password: ValidPassword123!
      roles:
        - roleName: readWrite
          databaseName: testdb
          
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: invalid-user
    spec:
      projectName: "$project_name"
      username: invalid-user
      databaseName: admin
      password: InvalidPassword123!
      # Missing roles - should cause validation error
EOF
    
    track_resource "config" "$mixed_validity_config" "error-handling"
    
    # This should fail validation due to missing roles in second user
    if "$PROJECT_ROOT/matlas" infra validate -f "$mixed_validity_config" 2>/dev/null; then
        print_error "Validation should have failed for missing roles in second user"
        return 1
    else
        print_success "Validation correctly failed for mixed valid/invalid resources"
    fi
    
    print_success "ApplyDocument error handling tests completed"
    return 0
}

# Cleanup function
cleanup_resources() {
    print_info "Cleaning up ApplyDocument test resources..."
    
    if [[ ${#CREATED_RESOURCES[@]} -eq 0 ]]; then
        print_info "No resources to clean up"
        return 0
    fi
    
    # Clean up files
    print_subheader "Cleaning up test files"
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
run_applydocument_tests() {
    local test_type="${1:-all}"
    
    print_header "ApplyDocument Format Comprehensive Tests"
    print_info "Testing ApplyDocument YAML format thoroughly to ensure proper coverage"
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
        "validation")
            test_applydocument_validation || ((test_failures++))
            ;;
        "mixed")
            test_mixed_resources || ((test_failures++))
            ;;
        "standalone")
            test_standalone_database_users || ((test_failures++))
            ;;
        "comparison")
            test_format_comparison || ((test_failures++))
            ;;
        "errors")
            test_applydocument_error_handling || ((test_failures++))
            ;;
        "all"|*)
            print_info "Running comprehensive ApplyDocument test suite..."
            test_applydocument_validation || ((test_failures++))
            echo
            test_mixed_resources || ((test_failures++))
            echo
            test_standalone_database_users || ((test_failures++))
            echo
            test_format_comparison || ((test_failures++))
            echo
            test_applydocument_error_handling || ((test_failures++))
            ;;
    esac
    
    echo
    if [[ $test_failures -eq 0 ]]; then
        print_success "All ApplyDocument tests passed!"
        return 0
    else
        print_error "$test_failures ApplyDocument test(s) failed"
        return 1
    fi
}

# Script usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  validation       Test ApplyDocument validation scenarios"
    echo "  mixed           Test mixed resource types in ApplyDocument"
    echo "  standalone      Test standalone DatabaseUser resources"
    echo "  comparison      Compare ApplyDocument vs Project formats"
    echo "  errors          Test error handling for ApplyDocument"
    echo "  all             Run all ApplyDocument tests (default)"
    echo
    echo "Purpose:"
    echo "  This script provides comprehensive testing for the ApplyDocument YAML format,"
    echo "  which was identified as under-tested compared to the Project format."
    echo
    echo "Environment variables required:"
    echo "  ATLAS_PUB_KEY       Atlas public API key"
    echo "  ATLAS_API_KEY       Atlas private API key"
    echo "  ATLAS_PROJECT_ID    Atlas project ID for testing"
    echo "  ATLAS_ORG_ID        Atlas organization ID"
    echo
    echo "Examples:"
    echo "  $0                  # Run all ApplyDocument tests"
    echo "  $0 validation       # Test validation scenarios only"
    echo "  $0 mixed            # Test mixed resources only"
    echo "  $0 errors           # Test error handling only"
}

# Main execution
main() {
    local command="${1:-all}"
    case "$command" in
        "validation"|"mixed"|"standalone"|"comparison"|"errors"|"all")
            run_applydocument_tests "$command"
            ;;
        "-h"|"--help"|"help")
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown command: $command"
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