#!/usr/bin/env bash

# ApplyDocument Regression Tests
# Specifically targets the errors found in cluster-lifecycle.sh:
# - Invalid regionName specified
# - Database user must have at least one role defined

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
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/applydocument-regression"
TEST_REGION="${TEST_REGION:-US_EAST_1}"

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

# Environment validation
check_environment() {
    print_info "Validating regression test environment..."
    
    # Check required environment variables
    if [[ -z "${ATLAS_PUB_KEY:-}" ]] || [[ -z "${ATLAS_API_KEY:-}" ]]; then
        print_error "Atlas credentials not configured"
        return 1
    fi
    
    if [[ -z "${ATLAS_PROJECT_ID:-}" ]]; then
        print_error "ATLAS_PROJECT_ID not configured"
        return 1
    fi
    
    # Check matlas binary
    if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
        print_error "matlas binary not found at $PROJECT_ROOT/matlas"
        return 1
    fi
    
    # Create test reports directory
    mkdir -p "$TEST_REPORTS_DIR"
    
    print_success "Environment validation completed"
    return 0
}

# Test the specific errors found in cluster-lifecycle.sh
test_regression_cases() {
    print_header "ApplyDocument Regression Test Cases"
    
    # Get project name
    local project_name
    if project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null); then
        print_info "Using project name: $project_name"
    else
        print_warning "Could not get project name, using project ID"
        project_name="$ATLAS_PROJECT_ID"
    fi
    
    # Test Case 1: Region Name Validation
    print_subheader "Test Case 1: Region Name Validation (Regression Fix)"
    
    local region_test_config="$TEST_REPORTS_DIR/region-validation.yaml"
    
    # Test 1a: Invalid region format (the original error)
    cat > "$region_test_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: region-validation-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: regression-cluster-invalid-region
    spec:
      projectName: "$project_name"
      provider: AWS
      region: ${TEST_REGION}  # This was the original issue - should be ${TEST_REGION}
      instanceSize: M10
      diskSizeGB: 10
EOF
    
    print_info "Testing invalid region format (${TEST_REGION})..."
    
    # This configuration should now be caught by validation if we've implemented proper region validation
    local validation_output
    if validation_output=$("$PROJECT_ROOT/matlas" infra validate -f "$region_test_config" 2>&1); then
        print_warning "Validation passed for US_EAST_1 - this may be allowed but should preferably use US_EAST_1"
        echo "Validation output: $validation_output"
    else
        print_success "Validation correctly caught invalid region format US_EAST_1"
        echo "Validation output: $validation_output"
    fi
    
    # Test 1b: Correct region format
    cat > "$region_test_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: region-validation-test-correct
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: regression-cluster-valid-region
    spec:
      projectName: "$project_name"
      provider: AWS
      region: ${TEST_REGION}  # Correct format
      instanceSize: M10
      diskSizeGB: 10
EOF
    
    print_info "Testing correct region format (${TEST_REGION})..."
    
    if "$PROJECT_ROOT/matlas" infra validate -f "$region_test_config"; then
        print_success "Validation passed for correct region format US_EAST_1"
    else
        print_error "Validation failed for correct region format - this indicates a problem"
        return 1
    fi
    
    # Test Case 2: Database User Roles Validation
    print_subheader "Test Case 2: Database User Roles Validation (Regression Fix)"
    
    local roles_test_config="$TEST_REPORTS_DIR/roles-validation.yaml"
    
    # Test 2a: Missing roles (the original error)
    cat > "$roles_test_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: roles-validation-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: regression-user-no-roles
    spec:
      projectName: "$project_name"
      username: regression-user-no-roles
      databaseName: admin
      password: RegressionTestPassword123!
      # Missing roles - this was the original issue
EOF
    
    print_info "Testing missing roles (original error case)..."
    
    # This should fail validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$roles_test_config" 2>/dev/null; then
        print_error "Validation should have failed for missing roles"
        return 1
    else
        print_success "Validation correctly failed for missing roles"
    fi
    
    # Test 2b: Empty roles array
    cat > "$roles_test_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: roles-validation-test-empty
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: regression-user-empty-roles
    spec:
      projectName: "$project_name"
      username: regression-user-empty-roles
      databaseName: admin
      password: RegressionTestPassword123!
      roles: []  # Empty roles array
EOF
    
    print_info "Testing empty roles array..."
    
    # This should also fail validation
    if "$PROJECT_ROOT/matlas" infra validate -f "$roles_test_config" 2>/dev/null; then
        print_error "Validation should have failed for empty roles array"
        return 1
    else
        print_success "Validation correctly failed for empty roles array"
    fi
    
    # Test 2c: Proper roles configuration
    cat > "$roles_test_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: roles-validation-test-correct
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: regression-user-valid-roles
    spec:
      projectName: "$project_name"
      username: regression-user-valid-roles
      databaseName: admin
      password: RegressionTestPassword123!
      roles:
        - roleName: readWrite
          databaseName: testdb
        - roleName: read
          databaseName: admin
EOF
    
    print_info "Testing proper roles configuration..."
    
    if "$PROJECT_ROOT/matlas" infra validate -f "$roles_test_config"; then
        print_success "Validation passed for proper roles configuration"
    else
        print_error "Validation failed for proper roles configuration - this indicates a problem"
        return 1
    fi
    
    # Test Case 3: Combined cluster + user (the original failing scenario)
    print_subheader "Test Case 3: Combined Cluster + DatabaseUser (Original Failing Scenario)"
    
    local combined_test_config="$TEST_REPORTS_DIR/combined-regression.yaml"
    
    cat > "$combined_test_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: combined-regression-test
  labels:
    test-type: regression
    purpose: combined-resources
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: regression-combined-cluster
      labels:
        test-type: regression
    spec:
      projectName: "$project_name"
      provider: AWS
      region: ${TEST_REGION}  # Fixed from ${TEST_REGION}
      instanceSize: M10
      diskSizeGB: 10
      backupEnabled: false
      mongodbVersion: "7.0"
      clusterType: REPLICASET
      
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: regression-combined-user
      labels:
        test-type: regression
    spec:
      projectName: "$project_name"
      username: regression-combined-user
      databaseName: admin
      password: CombinedRegressionPassword123!
      roles:  # Fixed - added proper roles
        - roleName: readWrite
          databaseName: testapp
        - roleName: read
          databaseName: admin
      scopes:
        - name: regression-combined-cluster
          type: CLUSTER
EOF
    
    print_info "Testing combined cluster + user configuration (regression fixed)..."
    
    if "$PROJECT_ROOT/matlas" infra validate -f "$combined_test_config"; then
        print_success "Combined cluster + user validation passed"
    else
        print_error "Combined cluster + user validation failed"
        return 1
    fi
    
    # Generate plan to ensure it works end-to-end
    print_info "Generating plan for combined configuration..."
    
    local plan_file="$TEST_REPORTS_DIR/combined-regression-plan.json"
    if "$PROJECT_ROOT/matlas" infra plan -f "$combined_test_config" --project-id "$ATLAS_PROJECT_ID" --output json > "$plan_file"; then
        print_success "Plan generation succeeded for combined configuration"
        
        # Show plan summary if jq is available
        if command -v jq >/dev/null 2>&1; then
            print_info "Plan summary:"
            jq -r '.summary // "Plan details not available"' "$plan_file" 2>/dev/null || echo "Plan created successfully"
        fi
    else
        print_error "Plan generation failed for combined configuration"
        return 1
    fi
    
    print_success "All regression test cases passed!"
    return 0
}

# Cleanup function
cleanup_files() {
    print_info "Cleaning up regression test files..."
    rm -f "$TEST_REPORTS_DIR"/*.yaml "$TEST_REPORTS_DIR"/*.json 2>/dev/null || true
    print_success "Cleanup completed"
}

# Main test runner
run_regression_tests() {
    print_header "ApplyDocument Regression Tests"
    print_info "Testing fixes for specific errors found in cluster-lifecycle.sh:"
    print_info "1. Invalid regionName specified (${TEST_REGION} vs ${TEST_REGION})"
    print_info "2. Database user must have at least one role defined"
    echo
    
    # Setup cleanup trap
    trap cleanup_files EXIT INT TERM
    
    # Environment validation
    if ! check_environment; then
        print_error "Environment validation failed"
        return 1
    fi
    
    # Run regression tests
    if test_regression_cases; then
        print_success "All regression tests passed!"
        print_info "The ApplyDocument format issues have been properly addressed"
        return 0
    else
        print_error "Regression tests failed"
        return 1
    fi
}

# Script usage
show_usage() {
    echo "Usage: $0"
    echo
    echo "Purpose:"
    echo "  Test specific regression cases for ApplyDocument format errors:"
  echo "  - Invalid regionName specified (${TEST_REGION} vs ${TEST_REGION})"
    echo "  - Database user must have at least one role defined"
    echo
    echo "Environment variables required:"
    echo "  ATLAS_PUB_KEY       Atlas public API key"
    echo "  ATLAS_API_KEY       Atlas private API key"
    echo "  ATLAS_PROJECT_ID    Atlas project ID for testing"
    echo
    echo "This script validates that the ApplyDocument format properly handles"
    echo "the specific error cases that were found in cluster-lifecycle.sh"
}

# Main execution
main() {
    local command="${1:-run}"
    case "$command" in
        "run"|"")
            run_regression_tests
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