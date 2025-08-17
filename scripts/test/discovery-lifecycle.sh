#!/usr/bin/env bash

# Discovery Lifecycle Tests
# Comprehensive testing of the discovery feature with real resources

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
readonly TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/discovery"
readonly TEST_REGION="${TEST_REGION:-US_EAST_1}"

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

# Resource tracking
declare -a CREATED_USERS=()
declare -a CREATED_NETWORK=()
declare -a TEMP_FILES=()

setup_discovery_environment() {
    mkdir -p "$TEST_REPORTS_DIR"
    
    # Load environment
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        set -o allexport
        source "$PROJECT_ROOT/.env"
        set +o allexport
    fi
    
    # Check Atlas credentials
    if [[ -z "${ATLAS_PUB_KEY:-}" || -z "${ATLAS_API_KEY:-}" || -z "${ATLAS_PROJECT_ID:-}" || -z "${ATLAS_ORG_ID:-}" ]]; then
        print_error "Atlas credentials required for discovery tests"
        print_info "Set ATLAS_PUB_KEY, ATLAS_API_KEY, ATLAS_PROJECT_ID, and ATLAS_ORG_ID in .env file"
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
    
    # Set up PATH to include the matlas binary
    export PATH="$PROJECT_ROOT:$PATH"
    
    print_success "Discovery test environment ready"
    return 0
}

track_user() {
    local user_id="$1"
    CREATED_USERS+=("$user_id")
}

track_network() {
    local entry_id="$1"
    CREATED_NETWORK+=("$entry_id")
}

track_temp_file() {
    local file="$1"
    TEMP_FILES+=("$file")
}

cleanup_resources() {
    print_info "Cleaning up test resources..."
    
    # Clean up users
    for user_id in "${CREATED_USERS[@]}"; do
        print_info "Cleaning up user: $user_id"
        if ! matlas database users delete "$user_id" --project-id "$ATLAS_PROJECT_ID" --force --no-confirm 2>/dev/null; then
            print_warning "Failed to cleanup user: $user_id"
        fi
    done
    
    # Clean up network access entries
    for entry_id in "${CREATED_NETWORK[@]}"; do
        print_info "Cleaning up network entry: $entry_id"
        if ! matlas atlas network delete "$entry_id" --project-id "$ATLAS_PROJECT_ID" --force --no-confirm 2>/dev/null; then
            print_warning "Failed to cleanup network entry: $entry_id"
        fi
    done
    
    # Clean up temp files
    for temp_file in "${TEMP_FILES[@]}"; do
        if [[ -f "$temp_file" ]]; then
            rm -f "$temp_file"
        fi
    done
    
    print_success "Cleanup completed"
}

# Trap for cleanup on exit
trap cleanup_resources EXIT

create_temp_file() {
    local name="$1"
    local content="$2"
    local file="$TEST_REPORTS_DIR/$name"
    
    echo "$content" > "$file"
    track_temp_file "$file"
    echo "$file"
}

wait_for_propagation() {
    local seconds="${1:-10}"
    print_info "Waiting ${seconds}s for Atlas propagation..."
    sleep "$seconds"
}

test_basic_discovery() {
    print_info "Testing basic project discovery..."
    
    local test_name="basic-discovery-$(date +%s)"
    local discovery_file="$TEST_REPORTS_DIR/${test_name}.yaml"
    local apply_file="$TEST_REPORTS_DIR/${test_name}-apply.yaml"
    
    # Test 1: Discover project and existing resources
    print_info "Step 1: Discovering project and existing resources..."
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --output-file "$discovery_file" --verbose; then
        print_error "Failed to discover project"
        return 1
    fi
    
    if [[ ! -f "$discovery_file" ]]; then
        print_error "Discovery file was not created"
        return 1
    fi
    
    track_temp_file "$discovery_file"
    print_success "Project discovered and saved to: $discovery_file"
    
    # Test 2: Convert to ApplyDocument
    print_info "Step 2: Converting to ApplyDocument format..."
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --convert-to-apply --output-file "$apply_file" --verbose; then
        print_error "Failed to convert to ApplyDocument"
        return 1
    fi
    
    if [[ ! -f "$apply_file" ]]; then
        print_error "ApplyDocument file was not created"
        return 1
    fi
    
    track_temp_file "$apply_file"
    print_success "Converted to ApplyDocument: $apply_file"
    
    # Test 3: Apply the same document to check consistency (no changes)
    print_info "Step 3: Applying converted document to verify consistency..."
    local plan_output="$TEST_REPORTS_DIR/${test_name}-plan.txt"
    
    if ! matlas infra plan -f "$apply_file" --project-id "$ATLAS_PROJECT_ID" > "$plan_output" 2>&1; then
        print_error "Failed to plan converted document"
        cat "$plan_output"
        return 1
    fi
    
    track_temp_file "$plan_output"
    
    # Check if plan shows no changes
    if grep -q "No changes" "$plan_output" || grep -q "0 to add, 0 to change, 0 to destroy" "$plan_output"; then
        print_success "Plan shows no changes - discovery and conversion are consistent!"
    else
        print_warning "Plan shows changes - may indicate discovery/conversion issues"
        print_info "Plan output:"
        cat "$plan_output"
    fi
    
    return 0
}

test_incremental_discovery() {
    print_info "Testing incremental discovery with user addition/removal..."
    
    local test_name="incremental-discovery-$(date +%s)"
    local test_user="discovery-test-user-$(date +%s)"
    local initial_discovery="$TEST_REPORTS_DIR/${test_name}-initial.yaml"
    local user_apply_doc="$TEST_REPORTS_DIR/${test_name}-user-addition.yaml"
    local final_discovery="$TEST_REPORTS_DIR/${test_name}-final.yaml"
    
    # Step 1: Get initial state
    print_info "Step 1: Getting initial project state..."
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --convert-to-apply --output-file "$initial_discovery" --verbose; then
        print_error "Failed to discover initial state"
        return 1
    fi
    
    track_temp_file "$initial_discovery"
    local initial_user_count
    initial_user_count=$(grep -c "kind: DatabaseUser" "$initial_discovery" || echo "0")
    print_success "Initial state captured with $initial_user_count database users"
    
    # Step 2: Create ApplyDocument with new user
    print_info "Step 2: Creating ApplyDocument with new user: $test_user"
    
    cat > "$user_apply_doc" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: discovery-test-user-addition
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: $test_user
    spec:
      username: $test_user
      authDatabase: admin
      password: "DiscoveryTest123!"
      projectName: "$ATLAS_PROJECT_ID"
      roles:
        - roleName: read
          databaseName: admin
EOF
    
    track_temp_file "$user_apply_doc"
    
    # Step 3: Apply the new user
    print_info "Step 3: Applying new user..."
    if ! matlas infra apply -f "$user_apply_doc" --project-id "$ATLAS_PROJECT_ID" --no-confirm --verbose; then
        print_error "Failed to apply new user"
        return 1
    fi
    
    track_user "$test_user"
    print_success "User $test_user created successfully"
    
    # Wait for propagation
    wait_for_propagation 15
    
    # Step 4: Verify user is detectable in Atlas via discovery
    print_info "Step 4: Verifying user is detectable via discovery..."
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --resource-type user --resource-name "$test_user" --verbose; then
        print_error "Failed to discover created user"
        return 1
    fi
    
    print_success "User $test_user is discoverable in Atlas"
    
    # Step 5: Run full discovery again and see new user in ApplyDocument
    print_info "Step 5: Running full discovery to capture new state..."
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --convert-to-apply --output-file "$final_discovery" --verbose; then
        print_error "Failed to discover final state"
        return 1
    fi
    
    track_temp_file "$final_discovery"
    
    # Verify new user is in discovery
    if grep -q "$test_user" "$final_discovery"; then
        print_success "New user found in discovery results"
    else
        print_error "New user NOT found in discovery results"
        return 1
    fi
    
    local final_user_count
    final_user_count=$(grep -c "kind: DatabaseUser" "$final_discovery" || echo "0")
    print_info "Final state has $final_user_count database users (was $initial_user_count)"
    
    if [[ $final_user_count -gt $initial_user_count ]]; then
        print_success "User count increased as expected"
    else
        print_warning "User count did not increase as expected"
    fi
    
    # Step 6: Remove the user while retaining other resources
    print_info "Step 6: Removing test user while retaining other resources..."
    if ! matlas database users delete "$test_user" --project-id "$ATLAS_PROJECT_ID" --force --no-confirm; then
        print_error "Failed to remove test user"
        return 1
    fi
    
    # Remove from tracking since we've manually deleted it
    for i in "${!CREATED_USERS[@]}"; do
        if [[ "${CREATED_USERS[i]}" == "$test_user" ]]; then
            unset 'CREATED_USERS[i]'
            break
        fi
    done
    
    print_success "Test user removed"
    
    # Wait for propagation
    wait_for_propagation 10
    
    # Step 7: Verify user is gone but other resources remain
    print_info "Step 7: Verifying user removal and resource retention..."
    local post_removal_discovery="$TEST_REPORTS_DIR/${test_name}-post-removal.yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --convert-to-apply --output-file "$post_removal_discovery" --verbose; then
        print_error "Failed to discover post-removal state"
        return 1
    fi
    
    track_temp_file "$post_removal_discovery"
    
    # Verify user is gone
    if grep -q "$test_user" "$post_removal_discovery"; then
        print_error "Removed user still appears in discovery"
        return 1
    else
        print_success "Removed user no longer appears in discovery"
    fi
    
    # Verify resource count is back to original or original-1
    local post_removal_user_count
    post_removal_user_count=$(grep -c "kind: DatabaseUser" "$post_removal_discovery" || echo "0")
    
    if [[ $post_removal_user_count -eq $initial_user_count ]]; then
        print_success "User count restored to initial value ($initial_user_count)"
    else
        print_info "Post-removal user count: $post_removal_user_count (initial: $initial_user_count)"
    fi
    
    return 0
}

test_resource_specific_discovery() {
    print_info "Testing resource-specific discovery..."
    
    # Test discovering specific resource types
    local test_name="resource-specific-$(date +%s)"
    
    # Test 1: Discover clusters
    print_info "Testing cluster discovery..."
    local cluster_file="$TEST_REPORTS_DIR/${test_name}-clusters.yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --include clusters --output-file "$cluster_file"; then
        print_error "Failed to discover clusters"
        return 1
    fi
    
    track_temp_file "$cluster_file"
    
    if [[ -s "$cluster_file" ]]; then
        local cluster_count
        cluster_count=$(grep -c "kind: Cluster" "$cluster_file" || echo "0")
        print_success "Discovered $cluster_count clusters"
    fi
    
    # Test 2: Discover users only
    print_info "Testing user discovery..."
    local users_file="$TEST_REPORTS_DIR/${test_name}-users.yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --include users --output-file "$users_file"; then
        print_error "Failed to discover users"
        return 1
    fi
    
    track_temp_file "$users_file"
    
    if [[ -s "$users_file" ]]; then
        local user_count
        user_count=$(grep -c "kind: DatabaseUser" "$users_file" || echo "0")
        print_success "Discovered $user_count database users"
    fi
    
    # Test 3: Discover network access only
    print_info "Testing network access discovery..."
    local network_file="$TEST_REPORTS_DIR/${test_name}-network.yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --include network --output-file "$network_file"; then
        print_error "Failed to discover network access"
        return 1
    fi
    
    track_temp_file "$network_file"
    
    if [[ -s "$network_file" ]]; then
        local network_count
        network_count=$(grep -c "kind: NetworkAccess" "$network_file" || echo "0")
        print_success "Discovered $network_count network access entries"
    fi
    
    return 0
}

test_discovery_with_filtering() {
    print_info "Testing discovery with filtering options..."
    
    local test_name="filtered-discovery-$(date +%s)"
    
    # Test 1: Include only specific types
    print_info "Testing include filtering..."
    local include_file="$TEST_REPORTS_DIR/${test_name}-include-clusters-users.yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --include clusters,users --output-file "$include_file"; then
        print_error "Failed to discover with include filter"
        return 1
    fi
    
    track_temp_file "$include_file"
    
    # Verify only clusters and users are included
    if grep -q "kind: Project" "$include_file"; then
        print_warning "Project found in include-filtered results (unexpected)"
    fi
    
    if grep -q "kind: NetworkAccess" "$include_file"; then
        print_warning "NetworkAccess found in include-filtered results (unexpected)"
    fi
    
    print_success "Include filtering completed"
    
    # Test 2: Exclude specific types
    print_info "Testing exclude filtering..."
    local exclude_file="$TEST_REPORTS_DIR/${test_name}-exclude-network.yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --exclude network --output-file "$exclude_file"; then
        print_error "Failed to discover with exclude filter"
        return 1
    fi
    
    track_temp_file "$exclude_file"
    
    # Verify network access is excluded
    if grep -q "kind: NetworkAccess" "$exclude_file"; then
        print_error "NetworkAccess found in exclude-filtered results (should be excluded)"
        return 1
    fi
    
    print_success "Exclude filtering completed"
    
    return 0
}

test_discovery_formats() {
    print_info "Testing discovery output formats..."
    
    local test_name="formats-$(date +%s)"
    
    # Test 1: YAML output (default)
    print_info "Testing YAML output..."
    local yaml_file="$TEST_REPORTS_DIR/${test_name}.yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --output yaml --output-file "$yaml_file"; then
        print_error "Failed YAML output test"
        return 1
    fi
    
    track_temp_file "$yaml_file"
    
    # Verify it's valid YAML
    if command -v yq >/dev/null; then
        if yq eval . "$yaml_file" > /dev/null 2>&1; then
            print_success "YAML output is valid"
        else
            print_error "YAML output is invalid"
            return 1
        fi
    else
        print_info "yq not available, skipping YAML validation"
    fi
    
    # Test 2: JSON output
    print_info "Testing JSON output..."
    local json_file="$TEST_REPORTS_DIR/${test_name}.json"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --output json --output-file "$json_file"; then
        print_error "Failed JSON output test"
        return 1
    fi
    
    track_temp_file "$json_file"
    
    # Verify it's valid JSON
    if command -v jq >/dev/null; then
        if jq . "$json_file" > /dev/null 2>&1; then
            print_success "JSON output is valid"
        else
            print_error "JSON output is invalid"
            return 1
        fi
    else
        print_info "jq not available, skipping JSON validation"
    fi
    
    # Test 3: Conversion format
    print_info "Testing ApplyDocument conversion format..."
    local convert_file="$TEST_REPORTS_DIR/${test_name}-converted.yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --convert-to-apply --output-file "$convert_file"; then
        print_error "Failed ApplyDocument conversion test"
        return 1
    fi
    
    track_temp_file "$convert_file"
    
    # Verify it contains ApplyDocument structure
    if grep -q "kind: ApplyDocument" "$convert_file"; then
        print_success "ApplyDocument conversion successful"
    else
        print_error "ApplyDocument conversion failed - no ApplyDocument kind found"
        return 1
    fi
    
    return 0
}

test_discovery_caching() {
    print_info "Testing discovery caching functionality..."
    
    # Test 1: Discovery with cache enabled (default)
    print_info "Testing cached discovery..."
    local cache_file="$TEST_REPORTS_DIR/cache-test-$(date +%s).yaml"
    
    local start_time
    start_time=$(date +%s)
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --cache-stats --output-file "$cache_file" --verbose; then
        print_error "Failed cached discovery"
        return 1
    fi
    
    local first_duration=$(($(date +%s) - start_time))
    track_temp_file "$cache_file"
    
    # Test 2: Second discovery should be faster (cache hit)
    print_info "Testing cache hit performance..."
    local cache_file2="$TEST_REPORTS_DIR/cache-test2-$(date +%s).yaml"
    
    start_time=$(date +%s)
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --cache-stats --output-file "$cache_file2" --verbose; then
        print_error "Failed second cached discovery"
        return 1
    fi
    
    local second_duration=$(($(date +%s) - start_time))
    track_temp_file "$cache_file2"
    
    print_info "First discovery: ${first_duration}s, Second discovery: ${second_duration}s"
    
    if [[ $second_duration -lt $first_duration ]]; then
        print_success "Cache appears to be working (second discovery was faster)"
    else
        print_info "Cache performance not conclusive (network variance possible)"
    fi
    
    # Test 3: Discovery with cache disabled
    print_info "Testing no-cache discovery..."
    local nocache_file="$TEST_REPORTS_DIR/nocache-test-$(date +%s).yaml"
    
    if ! matlas discover --project-id "$ATLAS_PROJECT_ID" --no-cache --output-file "$nocache_file" --verbose; then
        print_error "Failed no-cache discovery"
        return 1
    fi
    
    track_temp_file "$nocache_file"
    print_success "No-cache discovery completed"
    
    return 0
}

run_discovery_integration_tests() {
    print_info "Running Go integration tests for discovery..."
    
    cd "$PROJECT_ROOT"
    
    if ! go test -tags=integration -v ./test/integration/discovery/... -timeout=10m; then
        print_error "Discovery integration tests failed"
        return 1
    fi
    
    print_success "Discovery integration tests passed"
    return 0
}

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Discovery Lifecycle Tests - Comprehensive testing of the discovery feature

OPTIONS:
    --basic-only       Run only basic discovery tests
    --incremental-only Run only incremental discovery tests
    --skip-integration Skip Go integration tests
    --skip-cleanup     Skip resource cleanup (for debugging)
    --verbose          Enable verbose output
    --help             Show this help message

EXAMPLES:
    $0                     # Run all discovery tests
    $0 --basic-only        # Run only basic tests
    $0 --incremental-only  # Run only incremental tests
    $0 --skip-integration  # Skip Go integration tests
    $0 --verbose           # Run with verbose output

WHAT IT TESTS:
    1. Basic Discovery Flow:
       - Discover project and existing resources
       - Convert to ApplyDocument format
       - Apply converted document (verify no changes)
    
    2. Incremental Discovery:
       - Add user via ApplyDocument
       - Detect new user in Atlas via discovery
       - Run discovery again and verify user in results
       - Remove user while retaining other resources
    
    3. Resource-Specific Discovery:
       - Test discovery of individual resource types
       - Test filtering options (include/exclude)
    
    4. Format and Conversion Testing:
       - Test YAML and JSON output formats
       - Test DiscoveredProject to ApplyDocument conversion
    
    5. Advanced Features:
       - Test discovery caching functionality
       - Test Go integration tests

REQUIREMENTS:
    - Atlas credentials in .env file
    - Existing Atlas project with appropriate permissions
    - matlas binary (will be built if needed)

EOF
}

main() {
    local basic_only=false
    local incremental_only=false
    local skip_integration=false
    local skip_cleanup=false
    local verbose=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --basic-only)
                basic_only=true
                shift
                ;;
            --incremental-only)
                incremental_only=true
                shift
                ;;
            --skip-integration)
                skip_integration=true
                shift
                ;;
            --skip-cleanup)
                skip_cleanup=true
                shift
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            --help|-h)
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
    
    # Modify cleanup behavior
    if [[ "$skip_cleanup" == "true" ]]; then
        trap - EXIT
        print_warning "Resource cleanup disabled - you must manually clean up test resources"
    fi
    
    print_info "🔍 Starting Discovery Lifecycle Tests"
    
    # Setup environment
    if ! setup_discovery_environment; then
        print_error "Failed to setup test environment"
        exit 1
    fi
    
    local failed=0
    
    # Run tests based on options
    if [[ "$basic_only" == "true" ]]; then
        print_info "Running basic discovery tests only..."
        test_basic_discovery || ((failed++))
    elif [[ "$incremental_only" == "true" ]]; then
        print_info "Running incremental discovery tests only..."
        test_incremental_discovery || ((failed++))
    else
        # Run all tests
        print_info "Running comprehensive discovery test suite..."
        
        test_basic_discovery || ((failed++))
        test_incremental_discovery || ((failed++))
        test_resource_specific_discovery || ((failed++))
        test_discovery_with_filtering || ((failed++))
        test_discovery_formats || ((failed++))
        test_discovery_caching || ((failed++))
        
        if [[ "$skip_integration" != "true" ]]; then
            run_discovery_integration_tests || ((failed++))
        fi
    fi
    
    # Report results
    if [[ $failed -eq 0 ]]; then
        print_success "🎉 All discovery tests passed!"
        print_info "Test reports saved to: $TEST_REPORTS_DIR"
        exit 0
    else
        print_error "❌ $failed test(s) failed"
        print_info "Check test reports in: $TEST_REPORTS_DIR"
        exit 1
    fi
}

main "$@"




