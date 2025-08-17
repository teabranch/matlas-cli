#!/usr/bin/env bash

# matlas-cli Configuration Command Tests
# Comprehensive testing of config validate, template, and experimental commands

set -euo pipefail

# Colors
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'
readonly BOLD='\033[1m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly PROJECT_ROOT

# Test configuration
TEST_REPORTS_DIR="${TEST_REPORTS_DIR:-$PROJECT_ROOT/test-reports}"
readonly TEST_REPORTS_DIR

# Global state
CREATED_RESOURCES=()
RESOURCE_STATE_FILE="$TEST_REPORTS_DIR/config-test-resources.txt"
TEST_ID="config-test-$(date +%s)"

print_header() {
    echo -e "${BLUE}${BOLD}════════════════════════════════════════${NC}"
    echo -e "${BLUE}${BOLD} matlas-cli Configuration Tests${NC}"
    echo -e "${BLUE}${BOLD}════════════════════════════════════════${NC}"
}

print_subheader() { echo -e "${BLUE}${BOLD}$1${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run comprehensive tests for matlas config commands.

OPTIONS:
    --dry-run       Show what would be tested without running
    --verbose       Enable verbose output
    --timeout N     Set test timeout in seconds (default: 300)
    --help          Show this help message

TESTS COVERED:
    - Template generation (all types: basic, atlas, database, apply, complete)
    - Template listing
    - Configuration validation (valid and invalid configs)
    - Verbose validation output
    - Schema validation
    - Error handling for all commands
    - Output formats (YAML, JSON)
    - Experimental commands (import, export, migrate)

EXAMPLES:
    $0                    # Run all config tests
    $0 --verbose          # Run with verbose output
    $0 --dry-run          # Show test plan without running
    $0 --timeout 600      # Run with 10-minute timeout
EOF
}

track_resource() {
    local type="$1"
    local resource="$2"
    CREATED_RESOURCES+=("$type:$resource")
    echo "[$TEST_ID] Created $type: $resource" >> "$RESOURCE_STATE_FILE" 2>/dev/null || true
}

cleanup_resources() {
    if [[ ${#CREATED_RESOURCES[@]} -gt 0 ]]; then
        print_info "Cleaning up created test files..."
        for resource in "${CREATED_RESOURCES[@]}"; do
            local type="${resource%%:*}"
            local path="${resource#*:}"
            case "$type" in
                file)
                    if [[ -f "$path" ]]; then
                        rm -f "$path" && print_success "Removed file: $path" || print_warning "Failed to remove: $path"
                    fi
                    ;;
                dir)
                    if [[ -d "$path" ]]; then
                        rm -rf "$path" && print_success "Removed directory: $path" || print_warning "Failed to remove: $path"
                    fi
                    ;;
            esac
        done
    fi
}

setup_test_environment() {
    print_info "Setting up configuration test environment..."
    
    # Create test reports directory
    mkdir -p "$TEST_REPORTS_DIR"
    touch "$RESOURCE_STATE_FILE"
    
    # Verify matlas binary exists
    if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
        print_error "matlas binary not found. Run 'make build' first."
        return 1
    fi
    
    print_success "Test environment ready"
    return 0
}

test_template_list() {
    print_subheader "Testing template list command"
    
    local output_file="$TEST_REPORTS_DIR/template-list-output.txt"
    track_resource "file" "$output_file"
    
    # Test basic list
    print_info "Testing basic template list..."
    if "$PROJECT_ROOT/matlas" config template list > "$output_file" 2>&1; then
        print_success "Template list command succeeded"
    else
        print_error "Template list command failed"
        cat "$output_file"
        return 1
    fi
    
    # Verify expected templates are listed
    local expected_templates=("basic" "atlas" "database" "apply" "complete")
    for template in "${expected_templates[@]}"; do
        if grep -q "$template" "$output_file"; then
            print_success "Found template: $template"
        else
            print_error "Missing template: $template"
            return 1
        fi
    done
    
    # Test JSON output (if supported)
    print_info "Testing template list with JSON output..."
    local json_output="$TEST_REPORTS_DIR/template-list-json.json"
    track_resource "file" "$json_output"
    
    if "$PROJECT_ROOT/matlas" config template list --output json > "$json_output" 2>&1; then
        print_success "Template list JSON output succeeded"
        
        # Verify it's valid JSON
        if command -v jq >/dev/null 2>&1; then
            if jq empty < "$json_output" 2>/dev/null; then
                print_success "Template list JSON output is valid"
            else
                print_warning "Template list JSON output is invalid"
            fi
        else
            print_warning "jq not available, skipping JSON validation"
        fi
    else
        print_warning "Template list JSON output failed (may not be supported)"
    fi
    
    return 0
}

test_template_generation() {
    print_subheader "Testing template generation"
    
    local templates=("basic" "atlas" "database" "apply" "complete")
    local formats=("yaml" "json")
    
    for template in "${templates[@]}"; do
        for format in "${formats[@]}"; do
            print_info "Testing template generation: $template in $format format"
            
            local output_file="$TEST_REPORTS_DIR/template-$template-$format.$format"
            track_resource "file" "$output_file"
            
            # Generate template to file
            if "$PROJECT_ROOT/matlas" config template generate "$template" --format "$format" --file "$output_file"; then
                print_success "Generated $template template in $format format"
                
                # Verify file was created and has content
                if [[ -f "$output_file" && -s "$output_file" ]]; then
                    print_success "Template file created with content"
                    
                    # Basic format validation
                    case "$format" in
                        yaml)
                            # Check if it looks like YAML
                            if grep -q ":" "$output_file"; then
                                print_success "Template appears to be valid YAML format"
                            else
                                print_warning "Template may not be valid YAML format"
                            fi
                            ;;
                        json)
                            # Check if it's valid JSON
                            if command -v jq >/dev/null 2>&1; then
                                if jq empty < "$output_file" 2>/dev/null; then
                                    print_success "Template is valid JSON"
                                else
                                    print_error "Template is invalid JSON"
                                    return 1
                                fi
                            else
                                print_warning "jq not available, skipping JSON validation"
                            fi
                            ;;
                    esac
                else
                    print_error "Template file was not created or is empty"
                    return 1
                fi
            else
                print_error "Failed to generate $template template in $format format"
                return 1
            fi
            
            # Test generation to stdout
            print_info "Testing $template template generation to stdout"
            local stdout_output="$TEST_REPORTS_DIR/template-$template-$format-stdout.txt"
            track_resource "file" "$stdout_output"
            
            if "$PROJECT_ROOT/matlas" config template generate "$template" --format "$format" > "$stdout_output"; then
                print_success "Template generation to stdout succeeded"
                if [[ -s "$stdout_output" ]]; then
                    print_success "Stdout output contains data"
                else
                    print_warning "Stdout output is empty"
                fi
            else
                print_error "Template generation to stdout failed"
                return 1
            fi
        done
    done
    
    # Test invalid template type
    print_info "Testing invalid template type"
    local error_output="$TEST_REPORTS_DIR/template-invalid-error.txt"
    track_resource "file" "$error_output"
    
    if "$PROJECT_ROOT/matlas" config template generate invalid-template 2>"$error_output"; then
        print_error "Invalid template generation should have failed"
        return 1
    else
        print_success "Invalid template generation correctly failed"
        if grep -q "unknown template type" "$error_output"; then
            print_success "Error message contains expected text"
        else
            print_warning "Error message may not be descriptive enough"
        fi
    fi
    
    # Test invalid format
    print_info "Testing invalid format"
    local format_error="$TEST_REPORTS_DIR/template-invalid-format.txt"
    track_resource "file" "$format_error"
    
    if "$PROJECT_ROOT/matlas" config template generate basic --format invalid-format 2>"$format_error"; then
        print_error "Invalid format should have failed"
        return 1
    else
        print_success "Invalid format correctly failed"
        if grep -q "unsupported format" "$format_error"; then
            print_success "Format error message contains expected text"
        else
            print_warning "Format error message may not be descriptive enough"
        fi
    fi
    
    return 0
}

test_config_validation() {
    print_subheader "Testing configuration validation"
    
    # Temporarily unset Atlas environment variables to ensure we're testing config file validation
    # not environment variable validation
    local original_atlas_api_key="${ATLAS_API_KEY:-}"
    local original_atlas_pub_key="${ATLAS_PUB_KEY:-}"
    unset ATLAS_API_KEY ATLAS_PUB_KEY
    
    # Create test config directory
    local test_config_dir="$TEST_REPORTS_DIR/test-configs"
    mkdir -p "$test_config_dir"
    track_resource "dir" "$test_config_dir"
    
    # Test 1: Valid basic configuration
    print_info "Testing validation of valid basic configuration"
    local valid_config="$test_config_dir/valid-config.yaml"
    track_resource "file" "$valid_config"
    
    cat > "$valid_config" << EOF
output: text
timeout: 30s
projectId: 507f1f77bcf86cd799439011
apiKey: test-api-key-1234567890abcdef
publicKey: test-public-key-1234567890abcdef
EOF
    
    if "$PROJECT_ROOT/matlas" config validate "$valid_config"; then
        print_success "Valid configuration passed validation"
    else
        print_error "Valid configuration failed validation"
        return 1
    fi
    
    # Test 2: Valid configuration with verbose output
    print_info "Testing validation with verbose output"
    local verbose_output="$TEST_REPORTS_DIR/validation-verbose.txt"
    track_resource "file" "$verbose_output"
    
    if "$PROJECT_ROOT/matlas" config validate "$valid_config" --verbose > "$verbose_output" 2>&1; then
        print_success "Verbose validation succeeded"
        if grep -q "Configuration summary" "$verbose_output"; then
            print_success "Verbose output contains summary"
        else
            print_warning "Verbose output may be missing summary"
        fi
    else
        print_error "Verbose validation failed"
        return 1
    fi
    
    # Test 3: Invalid YAML syntax
    print_info "Testing validation of invalid YAML syntax"
    local invalid_yaml="$test_config_dir/invalid-yaml.yaml"
    track_resource "file" "$invalid_yaml"
    
    cat > "$invalid_yaml" << EOF
output: text
timeout: 30s
invalid_yaml: [unclosed bracket
projectId: test
EOF
    
    local yaml_error="$TEST_REPORTS_DIR/yaml-validation-error.txt"
    track_resource "file" "$yaml_error"
    
    if "$PROJECT_ROOT/matlas" config validate "$invalid_yaml" 2>"$yaml_error"; then
        print_error "Invalid YAML should have failed validation"
        return 1
    else
        print_success "Invalid YAML correctly failed validation"
        if grep -q -i "yaml\|syntax" "$yaml_error"; then
            print_success "Error message mentions YAML/syntax issues"
        else
            print_warning "Error message may not clearly indicate YAML issues"
        fi
    fi
    
    # Test 4: Configuration with validation errors
    print_info "Testing validation of configuration with field errors"
    local invalid_fields="$test_config_dir/invalid-fields.yaml"
    track_resource "file" "$invalid_fields"
    
    cat > "$invalid_fields" << EOF
output: invalid_output_format
timeout: -5s
projectId: invalid-project-id-format
apiKey: short
publicKey: abc
EOF
    
    local field_error="$TEST_REPORTS_DIR/field-validation-error.txt"
    track_resource "file" "$field_error"
    
    if "$PROJECT_ROOT/matlas" config validate "$invalid_fields" 2>"$field_error"; then
        print_error "Invalid fields should have failed validation"
        return 1
    else
        print_success "Invalid fields correctly failed validation"
        # Check if specific field errors are mentioned
        local error_content
        error_content=$(cat "$field_error")
        if echo "$error_content" | grep -q -i "api.*key\|public.*key\|project.*id"; then
            print_success "Error message mentions specific field issues"
        else
            print_warning "Error message may not detail specific field issues"
        fi
    fi
    
    # Test 5: Validation of non-existent file
    print_info "Testing validation of non-existent file"
    local missing_error="$TEST_REPORTS_DIR/missing-file-error.txt"
    track_resource "file" "$missing_error"
    
    if "$PROJECT_ROOT/matlas" config validate "/non/existent/file.yaml" 2>"$missing_error"; then
        print_error "Non-existent file should have failed validation"
        return 1
    else
        print_success "Non-existent file correctly failed validation"
        if grep -q -i "not found\|does not exist" "$missing_error"; then
            print_success "Error message indicates file not found"
        else
            print_warning "Error message may not clearly indicate missing file"
        fi
    fi
    
    # Test 6: Default config validation (if exists)
    print_info "Testing default config validation"
    local default_config_dir="$HOME/.matlas"
    local default_config="$default_config_dir/config.yaml"
    
    if [[ -f "$default_config" ]]; then
        print_info "Default config exists, testing validation"
        if "$PROJECT_ROOT/matlas" config validate; then
            print_success "Default config validation succeeded"
        else
            print_warning "Default config validation failed (may have actual issues)"
        fi
    else
        print_info "No default config found, creating temporary one for test"
        mkdir -p "$default_config_dir"
        local temp_default_created=false
        
        if [[ ! -f "$default_config" ]]; then
            cat > "$default_config" << EOF
output: text
timeout: 30s
EOF
            temp_default_created=true
        fi
        
        local default_error="$TEST_REPORTS_DIR/default-config-error.txt"
        track_resource "file" "$default_error"
        
        if "$PROJECT_ROOT/matlas" config validate 2>"$default_error"; then
            print_success "Default config validation with created config succeeded"
        else
            # This might fail if no credentials are set, which is OK
            print_warning "Default config validation failed (possibly due to missing credentials)"
        fi
        
        # Clean up temporary default config if we created it
        if [[ "$temp_default_created" == "true" ]]; then
            rm -f "$default_config"
        fi
    fi
    
    return 0
}

test_schema_validation() {
    print_subheader "Testing schema validation"
    
    # Test with custom schema would require creating a JSON schema
    # For now, test built-in schema validation behavior
    print_info "Testing built-in schema validation"
    
    local config_file="$TEST_REPORTS_DIR/schema-test-config.yaml"
    track_resource "file" "$config_file"
    
    cat > "$config_file" << EOF
output: json
timeout: 45s
projectId: 507f1f77bcf86cd799439011
EOF
    
    # Test validation (should use built-in schema validation)
    if "$PROJECT_ROOT/matlas" config validate "$config_file"; then
        print_success "Schema validation with built-in schema succeeded"
    else
        print_warning "Schema validation failed (may indicate schema issues)"
    fi
    
    # Test with non-existent custom schema
    print_info "Testing with non-existent custom schema"
    local schema_error="$TEST_REPORTS_DIR/schema-error.txt"
    track_resource "file" "$schema_error"
    
    if "$PROJECT_ROOT/matlas" config validate "$config_file" --schema "/non/existent/schema.json" 2>"$schema_error"; then
        print_error "Non-existent schema should have failed"
        return 1
    else
        print_success "Non-existent schema correctly failed"
        if grep -q -i "schema\|not found" "$schema_error"; then
            print_success "Error message mentions schema issues"
        else
            print_warning "Error message may not clearly indicate schema issues"
        fi
    fi
    
    return 0
}

test_experimental_commands() {
    print_subheader "Testing experimental commands"
    
    # These commands are hidden and return "not yet implemented" errors
    # We test that they exist and return appropriate errors
    
    # Test import command
    print_info "Testing experimental import command"
    local import_error="$TEST_REPORTS_DIR/import-error.txt"
    track_resource "file" "$import_error"
    
    local test_file="$TEST_REPORTS_DIR/test-import.yaml"
    echo "test: value" > "$test_file"
    track_resource "file" "$test_file"
    
    if "$PROJECT_ROOT/matlas" config import "$test_file" 2>"$import_error"; then
        print_warning "Import command succeeded (implementation may have been added)"
    else
        print_success "Import command failed as expected"
        if grep -q -i "not.*implemented\|not yet implemented" "$import_error"; then
            print_success "Import error message indicates not implemented"
        else
            print_warning "Import error message may not indicate implementation status"
        fi
    fi
    
    # Test export command
    print_info "Testing experimental export command"
    local export_error="$TEST_REPORTS_DIR/export-error.txt"
    track_resource "file" "$export_error"
    
    if "$PROJECT_ROOT/matlas" config export 2>"$export_error"; then
        print_warning "Export command succeeded (implementation may have been added)"
    else
        print_success "Export command failed as expected"
        if grep -q -i "not.*implemented\|not yet implemented" "$export_error"; then
            print_success "Export error message indicates not implemented"
        else
            print_warning "Export error message may not indicate implementation status"
        fi
    fi
    
    # Test migrate command
    print_info "Testing experimental migrate command"
    local migrate_error="$TEST_REPORTS_DIR/migrate-error.txt"
    track_resource "file" "$migrate_error"
    
    if "$PROJECT_ROOT/matlas" config migrate 2>"$migrate_error"; then
        print_warning "Migrate command succeeded (implementation may have been added)"
    else
        print_success "Migrate command failed as expected"
        if grep -q -i "not.*implemented\|not yet implemented" "$migrate_error"; then
            print_success "Migrate error message indicates not implemented"
        else
            print_warning "Migrate error message may not indicate implementation status"
        fi
    fi
    
    return 0
}

test_help_and_usage() {
    print_subheader "Testing help and usage output"
    
    # Test main config help
    print_info "Testing main config help"
    local config_help="$TEST_REPORTS_DIR/config-help.txt"
    track_resource "file" "$config_help"
    
    if "$PROJECT_ROOT/matlas" config --help > "$config_help" 2>&1; then
        print_success "Config help command succeeded"
        
        # Check for expected subcommands
        local expected_commands=("validate" "template")
        for cmd in "${expected_commands[@]}"; do
            if grep -q "$cmd" "$config_help"; then
                print_success "Help mentions $cmd subcommand"
            else
                print_warning "Help may not mention $cmd subcommand"
            fi
        done
    else
        print_error "Config help command failed"
        return 1
    fi
    
    # Test validate help
    print_info "Testing validate help"
    local validate_help="$TEST_REPORTS_DIR/validate-help.txt"
    track_resource "file" "$validate_help"
    
    if "$PROJECT_ROOT/matlas" config validate --help > "$validate_help" 2>&1; then
        print_success "Validate help command succeeded"
        
        # Check for expected flags
        local expected_flags=("--config" "--schema" "--verbose")
        for flag in "${expected_flags[@]}"; do
            if grep -q "$flag" "$validate_help"; then
                print_success "Help mentions $flag flag"
            else
                print_warning "Help may not mention $flag flag"
            fi
        done
    else
        print_error "Validate help command failed"
        return 1
    fi
    
    # Test template help
    print_info "Testing template help"
    local template_help="$TEST_REPORTS_DIR/template-help.txt"
    track_resource "file" "$template_help"
    
    if "$PROJECT_ROOT/matlas" config template --help > "$template_help" 2>&1; then
        print_success "Template help command succeeded"
        
        if grep -q -i "generate\|list" "$template_help"; then
            print_success "Template help mentions subcommands"
        else
            print_warning "Template help may not mention subcommands"
        fi
    else
        print_error "Template help command failed"
        return 1
    fi
    
    # Test template generate help
    print_info "Testing template generate help"
    local generate_help="$TEST_REPORTS_DIR/template-generate-help.txt"
    track_resource "file" "$generate_help"
    
    if "$PROJECT_ROOT/matlas" config template generate --help > "$generate_help" 2>&1; then
        print_success "Template generate help command succeeded"
        
        # Check for template types
        local template_types=("basic" "atlas" "database" "apply" "complete")
        for template in "${template_types[@]}"; do
            if grep -q "$template" "$generate_help"; then
                print_success "Help mentions $template template"
            else
                print_warning "Help may not mention $template template"
            fi
        done
        
        # Check for flags
        local expected_flags=("--output" "--format")
        for flag in "${expected_flags[@]}"; do
            if grep -q "$flag" "$generate_help"; then
                print_success "Help mentions $flag flag"
            else
                print_warning "Help may not mention $flag flag"
            fi
        done
    else
        print_error "Template generate help command failed"
        return 1
    fi
    
    return 0
}

test_edge_cases() {
    print_subheader "Testing edge cases and error conditions"
    
    # Test config command with no subcommand
    print_info "Testing config command with no subcommand"
    local no_subcmd="$TEST_REPORTS_DIR/no-subcommand.txt"
    track_resource "file" "$no_subcmd"
    
    if "$PROJECT_ROOT/matlas" config > "$no_subcmd" 2>&1; then
        print_success "Config with no subcommand shows help"
    else
        print_warning "Config with no subcommand behavior may need review"
    fi
    
    # Test template generate with no template type
    print_info "Testing template generate with no template type"
    local no_template="$TEST_REPORTS_DIR/no-template-type.txt"
    track_resource "file" "$no_template"
    
    if "$PROJECT_ROOT/matlas" config template generate 2>"$no_template"; then
        print_error "Template generate with no type should fail"
        return 1
    else
        print_success "Template generate with no type correctly failed"
    fi
    
    # Test template generate with multiple template types
    print_info "Testing template generate with multiple template types"
    local multi_template="$TEST_REPORTS_DIR/multi-template-type.txt"
    track_resource "file" "$multi_template"
    
    if "$PROJECT_ROOT/matlas" config template generate basic atlas 2>"$multi_template"; then
        print_error "Template generate with multiple types should fail"
        return 1
    else
        print_success "Template generate with multiple types correctly failed"
    fi
    
    # Test validate with multiple config files
    print_info "Testing validate with multiple config files"
    local multi_config="$TEST_REPORTS_DIR/multi-config.txt"
    track_resource "file" "$multi_config"
    
    # Create two dummy config files
    local config1="$TEST_REPORTS_DIR/config1.yaml"
    local config2="$TEST_REPORTS_DIR/config2.yaml"
    echo "output: text" > "$config1"
    echo "output: json" > "$config2"
    track_resource "file" "$config1"
    track_resource "file" "$config2"
    
    if "$PROJECT_ROOT/matlas" config validate "$config1" "$config2" 2>"$multi_config"; then
        print_error "Validate with multiple configs should fail"
        return 1
    else
        print_success "Validate with multiple configs correctly failed"
    fi
    
    return 0
}

run_config_tests() {
    local dry_run=false
    local timeout=300
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                dry_run=true
                shift
                ;;
            --verbose)
                # Verbose output is handled automatically by print functions
                shift
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    if [[ "$dry_run" == "true" ]]; then
        print_info "DRY RUN: Configuration command tests"
        print_info "Tests that would run:"
        print_info "  • Template list command"
        print_info "  • Template generation (all types, all formats)"
        print_info "  • Configuration validation (valid and invalid configs)"
        print_info "  • Schema validation"
        print_info "  • Experimental commands (import, export, migrate)"
        print_info "  • Help and usage output"
        print_info "  • Edge cases and error conditions"
        return 0
    fi
    
    print_header
    print_info "Running comprehensive configuration command tests..."
    print_info "Test ID: $TEST_ID"
    print_info "Timeout: ${timeout}s"
    print_info "Reports directory: $TEST_REPORTS_DIR"
    
    # Setup cleanup trap
    trap cleanup_resources EXIT INT TERM
    
    # Setup test environment
    if ! setup_test_environment; then
        print_error "Failed to setup test environment"
        return 1
    fi
    
    local test_failures=0
    
    # Run test suites
    print_info "Starting configuration command tests..."
    
    test_template_list || ((test_failures++))
    test_template_generation || ((test_failures++))
    test_config_validation || ((test_failures++))
    test_schema_validation || ((test_failures++))
    test_experimental_commands || ((test_failures++))
    test_help_and_usage || ((test_failures++))
    test_edge_cases || ((test_failures++))
    
    # Report results
    echo
    if [[ $test_failures -eq 0 ]]; then
        print_success "All configuration tests passed! 🎉"
        print_info "Test artifacts saved to: $TEST_REPORTS_DIR"
        return 0
    else
        print_error "$test_failures test suite(s) failed"
        print_info "Check test artifacts in: $TEST_REPORTS_DIR"
        return 1
    fi
}

run_config_tests "$@"
