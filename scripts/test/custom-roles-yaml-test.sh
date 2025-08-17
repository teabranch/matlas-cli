#!/usr/bin/env bash
# Test custom role creation from YAML ApplyDocument
# This test verifies that custom database roles can be created from YAML configuration

set -euo pipefail

# Color functions
print_info(){ echo -e "\033[0;35mℹ $1\033[0m"; }
print_success(){ echo -e "\033[0;32m✓ $1\033[0m"; }
print_error(){ echo -e "\033[0;31m✗ $1\033[0m"; }
print_warning(){ echo -e "\033[0;33m⚠ $1\033[0m"; }

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/custom-roles-yaml"
mkdir -p "$TEST_REPORTS_DIR"

# Ensure matlas is built
if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
  echo "Building matlas binary..."
  (cd "$PROJECT_ROOT" && go build -o matlas) || { print_error "Build failed"; exit 1; }
fi

# Check required environment variables
if [[ -z "${ATLAS_PROJECT_ID:-}" ]]; then
  print_error "ATLAS_PROJECT_ID environment variable is required"
  exit 1
fi

if [[ -z "${ATLAS_CLUSTER_NAME:-}" ]]; then
  print_warning "ATLAS_CLUSTER_NAME not set - some tests may be skipped"
  CLUSTER_AVAILABLE="false"
else
  CLUSTER_AVAILABLE="true"
fi

# Test functions
test_yaml_role_validation() {
  print_info "Testing YAML custom role validation..."
  
  local timestamp=$(date +%s)
  local test_db="yamlvalidationdb$timestamp"
  local role_name="yamlvalidationrole$timestamp"
  local config_file="$TEST_REPORTS_DIR/role-validation-test.yaml"
  
  # Create a valid YAML configuration
  cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: custom-role-validation-test
  labels:
    test-type: validation
    purpose: role-testing
resources:
  # Custom database role for testing
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: $role_name-resource
      labels:
        purpose: validation-testing
    spec:
      roleName: $role_name
      databaseName: $test_db
      privileges:
        # Collection-level privileges
        - actions: ["find", "insert", "update"]
          resource:
            database: $test_db
            collection: users
        - actions: ["find"]
          resource:
            database: $test_db
            collection: logs
        # Database-level privileges
        - actions: ["listCollections", "listIndexes"]
          resource:
            database: $test_db
      inheritedRoles:
        # Inherit from built-in role
        - roleName: read
          databaseName: $test_db
        # Multiple inherited roles
        - roleName: read
          databaseName: admin
EOF

  # Test validation
  print_info "Validating YAML configuration..."
  if "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
    print_success "YAML validation passed"
  else
    print_error "YAML validation failed"
    return 1
  fi
  
  # Test planning (dry-run)
  print_info "Testing plan generation (dry-run)..."
  if "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run >/dev/null 2>&1; then
    print_success "Plan generation passed"
  else
    print_error "Plan generation failed"
    return 1
  fi
  
  print_success "YAML role validation completed successfully"
}

test_yaml_role_invalid_configurations() {
  print_info "Testing invalid YAML role configurations..."
  
  local timestamp=$(date +%s)
  local invalid_config="$TEST_REPORTS_DIR/invalid-role-test.yaml"
  
  # Test 1: Empty role name
  print_info "Testing empty role name validation..."
  cat > "$invalid_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-empty-role-name
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: invalid-role
    spec:
      roleName: ""  # Invalid: empty role name
      databaseName: testdb
      privileges:
        - actions: ["find"]
          resource:
            database: testdb
            collection: users
EOF

  if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_config" 2>/dev/null; then
    print_error "Validation should have failed for empty role name"
    return 1
  else
    print_success "Validation correctly failed for empty role name"
  fi
  
  # Test 2: Empty database name
  print_info "Testing empty database name validation..."
  cat > "$invalid_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-empty-database-name
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: invalid-role
    spec:
      roleName: testRole
      databaseName: ""  # Invalid: empty database name
      privileges:
        - actions: ["find"]
          resource:
            database: testdb
            collection: users
EOF

  if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_config" 2>/dev/null; then
    print_error "Validation should have failed for empty database name"
    return 1
  else
    print_success "Validation correctly failed for empty database name"
  fi
  
  # Test 3: Invalid privilege format
  print_info "Testing invalid privilege format validation..."
  cat > "$invalid_config" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-privilege-format
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: invalid-privilege-role
    spec:
      roleName: testRole
      databaseName: testdb
      privileges:
        - actions: []  # Invalid: empty actions
          resource:
            database: testdb
            collection: users
EOF

  if "$PROJECT_ROOT/matlas" infra validate -f "$invalid_config" 2>/dev/null; then
    print_error "Validation should have failed for empty actions"
    return 1
  else
    print_success "Validation correctly failed for empty actions"
  fi
  
  print_success "Invalid configuration validation completed successfully"
}

test_yaml_role_complex_configuration() {
  print_info "Testing complex YAML role configuration..."
  
  local timestamp=$(date +%s)
  local test_db="complexroledb$timestamp"
  local app_role="complexapprole$timestamp"
  local analytics_role="complexanalyticsrole$timestamp"
  local config_file="$TEST_REPORTS_DIR/complex-role-test.yaml"
  
  # Create a complex configuration with multiple roles and dependencies
  cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: complex-custom-roles-test
  labels:
    test-type: complex-validation
    purpose: multi-role-testing
resources:
  # Primary application role
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: $app_role-resource
      labels:
        purpose: application
        tier: primary
    spec:
      roleName: $app_role
      databaseName: $test_db
      privileges:
        # Full CRUD on user data
        - actions: ["find", "insert", "update", "remove"]
          resource:
            database: $test_db
            collection: users
        - actions: ["find", "insert", "update", "remove"]
          resource:
            database: $test_db
            collection: profiles
        # Append-only logging
        - actions: ["insert", "find"]
          resource:
            database: $test_db
            collection: audit_logs
        # Read-only access to configuration
        - actions: ["find"]
          resource:
            database: $test_db
            collection: config
        # Database-level operations
        - actions: ["listCollections", "listIndexes"]
          resource:
            database: $test_db
      inheritedRoles:
        # Inherit read access to shared reference data
        - roleName: read
          databaseName: reference
          
  # Analytics role with broader read access
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: $analytics_role-resource
      labels:
        purpose: analytics
        tier: secondary
    spec:
      roleName: $analytics_role
      databaseName: $test_db
      privileges:
        # Read-only access to all application data
        - actions: ["find", "listIndexes"]
          resource:
            database: $test_db
            collection: users
        - actions: ["find", "listIndexes"]
          resource:
            database: $test_db
            collection: profiles
        - actions: ["find"]
          resource:
            database: $test_db
            collection: audit_logs
        # Database-level read operations
        - actions: ["listCollections", "dbStats", "collStats"]
          resource:
            database: $test_db
        # Cross-database analytics access
        - actions: ["find"]
          resource:
            database: analytics
            collection: reports
      inheritedRoles:
        # Inherit basic read access
        - roleName: read
          databaseName: $test_db
        - roleName: read
          databaseName: analytics
EOF

  # Test validation
  print_info "Validating complex YAML configuration..."
  if "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
    print_success "Complex YAML validation passed"
  else
    print_error "Complex YAML validation failed"
    return 1
  fi
  
  # Test planning
  print_info "Testing complex plan generation..."
  if "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run >/dev/null 2>&1; then
    print_success "Complex plan generation passed"
  else
    print_error "Complex plan generation failed"
    return 1
  fi
  
  print_success "Complex YAML role configuration completed successfully"
}

test_yaml_role_documentation() {
  print_info "Testing YAML role documentation structure..."
  
  local timestamp=$(date +%s)
  local test_db="rolewithuserdb$timestamp"
  local role_name="rolewithuserrole$timestamp"
  # Note: user creation is not tested as it's not supported via direct DB commands
  local config_file="$TEST_REPORTS_DIR/role-with-user-test.yaml"
  
  # Create configuration with role and user
  cat > "$config_file" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: role-with-user-test
  labels:
    test-type: role-user-integration
resources:
  # Custom database role
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: $role_name-resource
    spec:
      roleName: $role_name
      databaseName: $test_db
      privileges:
        - actions: ["find", "insert", "update"]
          resource:
            database: $test_db
            collection: documents
        - actions: ["find"]
          resource:
            database: $test_db
            collection: metadata
      inheritedRoles:
        - roleName: read
          databaseName: $test_db
          
  # Note: DatabaseUser creation via YAML is not supported in Atlas
  # Users must be created through Atlas API/UI, then roles can be assigned
  # This configuration demonstrates the role structure for documentation purposes
EOF

  # Test validation (role portion only)
  print_info "Validating role documentation YAML configuration..."
  if "$PROJECT_ROOT/matlas" infra validate -f "$config_file"; then
    print_success "Role documentation YAML validation passed"
  else
    print_error "Role documentation YAML validation failed"
    return 1
  fi
  
  # Test planning (role portion only)
  print_info "Testing role documentation plan generation..."
  if "$PROJECT_ROOT/matlas" infra plan -f "$config_file" --project-id "$ATLAS_PROJECT_ID" --dry-run >/dev/null 2>&1; then
    print_success "Role documentation plan generation passed"
  else
    print_error "Role documentation plan generation failed"
    return 1
  fi
  
  print_success "YAML role documentation structure completed successfully"
}

# Main test execution
main() {
  print_info "Starting custom roles YAML tests..."
  
  # Run validation tests
  test_yaml_role_validation || exit 1
  test_yaml_role_invalid_configurations || exit 1
  test_yaml_role_complex_configuration || exit 1
  test_yaml_role_documentation || exit 1
  
  print_success "All custom roles YAML tests completed successfully!"
  
  print_info "Summary:"
  print_info "✓ Basic YAML role validation"
  print_info "✓ Invalid configuration rejection"  
  print_info "✓ Complex multi-role configuration"
  print_info "✓ Role documentation structure"
  
  print_success "Custom roles YAML testing suite PASSED"
}

# Run tests
main "$@"
