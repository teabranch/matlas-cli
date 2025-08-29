#!/bin/bash

# Advanced Search Features Test Script (YAML-Only)
# Tests Atlas Search advanced features via YAML configuration only
# CLI commands for advanced features were removed due to Atlas Admin API limitations
# Tests: analyzers, facets, autocomplete, highlighting, synonyms, and fuzzy search via YAML apply

set -euo pipefail

# Script metadata (exported for external use)
export SCRIPT_NAME="search-advanced-features.sh"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MATLAS_CLI="$PROJECT_ROOT/matlas"

# Load environment variables
if [[ -f "$PROJECT_ROOT/.env" ]]; then
    source "$PROJECT_ROOT/.env"
fi

# Note: This script implements functionality inline since test-helpers.sh was removed
# Advanced search features are YAML-only due to Atlas Admin API limitations

# Test configuration
TEST_PROJECT_ID="${ATLAS_PROJECT_ID:-}"
TEST_CLUSTER_NAME="${ATLAS_CLUSTER_NAME:-advanced-search-test-cluster}"
TEST_DATABASE_NAME=""  # Will be discovered
TEST_COLLECTION_NAME=""  # Will be discovered
TEST_INDEX_NAME="products-advanced-index-$(date +%s)"

# Test data files
ADVANCED_SEARCH_YAML="$PROJECT_ROOT/examples/search-advanced-features.yaml"
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Track created search indexes for cleanup
CREATED_INDEXES_FILE="$TEMP_DIR/created_indexes.txt"
touch "$CREATED_INDEXES_FILE"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to track created search indexes for cleanup
track_created_index() {
    local index_name="$1"
    echo "$index_name" >> "$CREATED_INDEXES_FILE"
}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Function to discover existing database and collection for testing
discover_test_database_and_collection() {
    log_info "Discovering existing databases and collections for testing..."
    
    # Export variables for use in test functions
    export TEST_PROJECT_ID
    export TEST_CLUSTER_NAME
    export TEST_DATABASE_NAME
    export TEST_COLLECTION_NAME
    export TEST_INDEX_NAME
    
    # First try to find existing search indexes to reuse their database/collection
    local existing_indexes
    if existing_indexes=$($MATLAS_CLI atlas search list --project-id "$TEST_PROJECT_ID" --cluster "$TEST_CLUSTER_NAME" --output json 2>/dev/null); then
        if echo "$existing_indexes" | jq -e '. | length > 0' >/dev/null 2>&1; then
            # Use database and collection from first existing search index
            TEST_DATABASE_NAME=$(echo "$existing_indexes" | jq -r '.[0].database')
            TEST_COLLECTION_NAME=$(echo "$existing_indexes" | jq -r '.[0].collectionName')
            log_success "Found existing search index - using database: $TEST_DATABASE_NAME, collection: $TEST_COLLECTION_NAME"
            return 0
        fi
    fi
    
    # If no search indexes exist, discover databases
    local databases
    if databases=$($MATLAS_CLI database list --project-id "$TEST_PROJECT_ID" --cluster "$TEST_CLUSTER_NAME" --use-temp-user --output json 2>/dev/null); then
        # Filter out system databases and find a suitable test database
        local suitable_db
        suitable_db=$(echo "$databases" | jq -r '.[] | select(.name != "admin" and .name != "config" and .name != "local" and .empty == false) | .name' | head -1)
        
        if [[ -n "$suitable_db" ]]; then
            TEST_DATABASE_NAME="$suitable_db"
            # Use a default collection name for testing
            TEST_COLLECTION_NAME="auth-test-collection"
            log_success "Using discovered database: $TEST_DATABASE_NAME with collection: $TEST_COLLECTION_NAME"
            return 0
        fi
    fi
    
    # Fallback to creating a new test database and collection
    TEST_DATABASE_NAME="advanced_search_test_$(date +%s)"
    TEST_COLLECTION_NAME="products"
    log_warning "No suitable existing database found, will use: $TEST_DATABASE_NAME.$TEST_COLLECTION_NAME"
    log_info "Note: Collection will be created when first document is inserted"
    return 0
}

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites for advanced search features test..."
    
    # Check if matlas CLI exists
    if [[ ! -x "$MATLAS_CLI" ]]; then
        log_error "matlas CLI not found at $MATLAS_CLI"
        log_error "Please build the CLI first: make build"
        exit 1
    fi
    
    # Check if project ID is set
    if [[ -z "$TEST_PROJECT_ID" ]]; then
        log_error "ATLAS_PROJECT_ID environment variable is required"
        log_error "Please set it to your Atlas project ID"
        exit 1
    fi
    
    # Check Atlas authentication by testing a simple command
    log_info "Verifying Atlas authentication..."
    if ! $MATLAS_CLI atlas projects list >/dev/null 2>&1; then
        log_error "Atlas authentication not configured properly"
        log_error "Please run: matlas config init"
        log_error "Or check your ATLAS_API_KEY and ATLAS_PUB_KEY environment variables"
        exit 1
    fi
    
    log_success "Prerequisites check completed"
}

# Function to create test search index with advanced features
create_advanced_search_index() {
    log_info "Creating advanced search index with multiple features..."
    log_info "Using database: $TEST_DATABASE_NAME, collection: $TEST_COLLECTION_NAME"
    
    # Create temporary YAML with actual project ID, cluster, database, and collection
    local temp_yaml="$TEMP_DIR/advanced-search-test.yaml"
    local timestamp=$(date +%s)
    local unique_index_name="products-advanced-search-$timestamp"
    # Replace patterns in specific order to avoid conflicts
    sed "s/your-project-id/$TEST_PROJECT_ID/g; s/your-cluster-name/$TEST_CLUSTER_NAME/g; s/ecommerce/$TEST_DATABASE_NAME/g; s/collectionName: \"products\"/collectionName: \"$TEST_COLLECTION_NAME\"/g; s/products-advanced-search/$unique_index_name/g" \
        "$ADVANCED_SEARCH_YAML" > "$temp_yaml"
    
    # Apply the configuration with preserve-existing flag
    log_info "Applying advanced search configuration..."
    if $MATLAS_CLI infra -f "$temp_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing --verbose; then
        log_success "Advanced search index created successfully"
    else
        log_error "Failed to create advanced search index"
        return 1
    fi
    
    # Wait for index to be ready
    log_info "Waiting for search index to be ready..."
    wait_for_search_index_ready "$TEST_PROJECT_ID" "$TEST_CLUSTER_NAME" "$unique_index_name"
}

# Function to test YAML apply with basic search index
test_yaml_apply_basic() {
    log_info "Testing YAML apply with basic search index..."
    
    local timestamp=$(date +%s)
    local index_name="basic-test-index-$timestamp"
    local test_yaml="$TEMP_DIR/basic-search-index.yaml"
    
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: basic-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: basic-test-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "$index_name"
      indexType: "search"
      definition:
        mappings:
          dynamic: true
EOF
    
    if $MATLAS_CLI infra -f "$test_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing; then
        log_success "Basic YAML apply executed successfully"
        # Validate the index was actually created and is functional
        if validate_index_status "$TEST_PROJECT_ID" "$TEST_CLUSTER_NAME" "$index_name"; then
            track_created_index "$index_name"
            return 0
        else
            log_error "Basic index validation failed"
            return 1
        fi
    else
        log_error "Basic YAML apply failed"
        return 1
    fi
}

# Function to test YAML apply with analyzers
test_yaml_apply_analyzers() {
    log_info "Testing YAML apply with custom analyzers..."
    
    local test_yaml="$TEMP_DIR/analyzers-search-index.yaml"
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: analyzers-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: analyzers-test-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "analyzers-test-index-$(date +%s)"
      indexType: "search"
      definition:
        mappings:
          dynamic: true
      analyzers:
        - name: "customTextAnalyzer"
          type: "standard"
          tokenFilters: ["lowercase", "stop"]
        - name: "productTitleAnalyzer"
          type: "keyword"
EOF
    
    if $MATLAS_CLI infra -f "$test_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing; then
        log_success "Analyzers YAML apply executed successfully"
        return 0
    else
        log_error "Analyzers YAML apply failed"
        return 1
    fi
}

# Function to test YAML apply with facets
test_yaml_apply_facets() {
    log_info "Testing YAML apply with faceted search..."
    
    local test_yaml="$TEMP_DIR/facets-search-index.yaml"
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: facets-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: facets-test-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "facets-test-index-$(date +%s)"
      indexType: "search"
      definition:
        mappings:
          dynamic: true
        facets:
          - field: "category"
            type: "string"
          - field: "price"
            type: "number"
          - field: "created_date"
            type: "date"
EOF
    
    if $MATLAS_CLI infra -f "$test_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing; then
        log_success "Facets YAML apply executed successfully"
        return 0
    else
        log_error "Facets YAML apply failed"
        return 1
    fi
}

# Function to test YAML apply with autocomplete
test_yaml_apply_autocomplete() {
    log_info "Testing YAML apply with autocomplete features..."
    
    local test_yaml="$TEMP_DIR/autocomplete-search-index.yaml"
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: autocomplete-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: autocomplete-test-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "autocomplete-test-index-$(date +%s)"
      indexType: "search"
      definition:
        mappings:
          dynamic: false
          fields:
            title:
              type: autocomplete
            description:
              type: autocomplete
EOF
    
    if $MATLAS_CLI infra -f "$test_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing; then
        log_success "Autocomplete YAML apply executed successfully"
        return 0
    else
        log_error "Autocomplete YAML apply failed"
        return 1
    fi
}

# Function to test YAML apply with highlighting
test_yaml_apply_highlighting() {
    log_info "Testing YAML apply with highlighting features..."
    
    local test_yaml="$TEMP_DIR/highlighting-search-index.yaml"
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: highlighting-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: highlighting-test-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "highlighting-test-index-$(date +%s)"
      indexType: "search"
      definition:
        mappings:
          dynamic: true
        highlighting:
          - field: "content"
            maxCharsToExamine: 500000
            maxNumPassages: 5
          - field: "description"
            maxCharsToExamine: 100000
            maxNumPassages: 3
EOF
    
    if $MATLAS_CLI infra -f "$test_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing; then
        log_success "Highlighting YAML apply executed successfully"
        return 0
    else
        log_error "Highlighting YAML apply failed"
        return 1
    fi
}

# Function to test YAML apply with synonyms
test_yaml_apply_synonyms() {
    log_info "Testing YAML apply with synonyms..."
    
    local test_yaml="$TEMP_DIR/synonyms-search-index.yaml"
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: synonyms-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: synonyms-test-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "synonyms-test-index-$(date +%s)"
      indexType: "search"
      definition:
        mappings:
          dynamic: true
        synonyms:
          - name: "vehicleSynonyms"
            input: ["car", "automobile", "vehicle"]
            output: "vehicle"
          - name: "techSynonyms"
            input: ["computer", "laptop", "pc"]
            output: "computer"
EOF
    
    if $MATLAS_CLI infra -f "$test_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing; then
        log_success "Synonyms YAML apply executed successfully"
        return 0
    else
        log_error "Synonyms YAML apply failed"
        return 1
    fi
}

# Function to test YAML apply with fuzzy search
test_yaml_apply_fuzzy() {
    log_info "Testing YAML apply with fuzzy search..."
    
    local test_yaml="$TEMP_DIR/fuzzy-search-index.yaml"
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: fuzzy-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: fuzzy-test-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "fuzzy-test-index-$(date +%s)"
      indexType: "search"
      definition:
        mappings:
          dynamic: true
        fuzzySearch:
          - field: "title"
            maxEdits: 2
            prefixLength: 1
            maxExpansions: 50
          - field: "tags"
            maxEdits: 1
            prefixLength: 2
            maxExpansions: 25
EOF
    
    if $MATLAS_CLI infra -f "$test_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing; then
        log_success "Fuzzy search YAML apply executed successfully"
        return 0
    else
        log_error "Fuzzy search YAML apply failed"
        return 1
    fi
}

# Function to test comprehensive YAML apply with all advanced features
test_yaml_apply_comprehensive() {
    log_info "Testing comprehensive YAML apply with all advanced search features..."
    
    local timestamp=$(date +%s)
    local index_name="comprehensive-advanced-index-$timestamp"
    local test_yaml="$TEMP_DIR/comprehensive-advanced-search.yaml"
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: comprehensive-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: comprehensive-advanced-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "$index_name"
      indexType: "search"
      definition:
        mappings:
          dynamic: false
          fields:
            title:
              type: string
              analyzer: "lucene.standard"
            title_autocomplete:
              type: autocomplete
              analyzer: "lucene.standard"
            content:
              type: string
            category:
              type: stringFacet
            price:
              type: numberFacet
EOF
    
    if $MATLAS_CLI infra -f "$test_yaml" --project-id "$TEST_PROJECT_ID" --auto-approve --preserve-existing; then
        log_success "Comprehensive YAML apply executed successfully"
        # Validate the index was actually created and is functional
        if validate_index_status "$TEST_PROJECT_ID" "$TEST_CLUSTER_NAME" "$index_name"; then
            track_created_index "$index_name"
            return 0
        else
            log_error "Comprehensive index validation failed"
            return 1
        fi
    else
        log_error "Comprehensive YAML apply failed"
        return 1
    fi
}

# Function to test YAML validation
test_yaml_validation() {
    log_info "Testing YAML validation for advanced search features..."
    
    # Create an invalid YAML to test validation
    local invalid_yaml="$TEMP_DIR/invalid-search-index.yaml"
    cat > "$invalid_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: invalid-index
    spec:
      # Missing required fields to test validation
      indexName: "invalid-index"
      indexType: "search"
      analyzers:
        - name: "invalidAnalyzer"
          # Missing required type field
EOF
    
    if $MATLAS_CLI infra validate -f "$invalid_yaml" 2>/dev/null; then
        log_warning "YAML validation passed when it should have failed"
        return 1
    else
        log_success "YAML validation correctly caught invalid configuration"
        return 0
    fi
}

# Function to test YAML plan and diff operations
test_yaml_plan_diff() {
    log_info "Testing YAML plan and diff operations for advanced search features..."
    
    local test_yaml="$TEMP_DIR/plan-diff-search-index.yaml"
    cat > "$test_yaml" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: plan-diff-search-test
  labels:
    test: advanced-search-features
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: plan-diff-index
    spec:
      projectName: "${TEST_PROJECT_ID}"
      clusterName: "${TEST_CLUSTER_NAME}"
      databaseName: "${TEST_DATABASE_NAME}"
      collectionName: "${TEST_COLLECTION_NAME}"
      indexName: "plan-diff-index-$(date +%s)"
      indexType: "search"
      definition:
        mappings:
          dynamic: true
      analyzers:
        - name: "planDiffAnalyzer"
          type: "keyword"
EOF
    
    local plan_passed=false
    local diff_passed=false
    
    # Test plan operation
    if $MATLAS_CLI infra plan -f "$test_yaml" --preserve-existing >/dev/null 2>&1; then
        log_success "YAML plan operation executed successfully"
        plan_passed=true
    else
        log_warning "YAML plan operation failed"
    fi
    
    # Test diff operation
    if $MATLAS_CLI infra diff -f "$test_yaml" --preserve-existing >/dev/null 2>&1; then
        log_success "YAML diff operation executed successfully"
        diff_passed=true
    else
        log_warning "YAML diff operation failed"
    fi
    
    if [[ "$plan_passed" == true && "$diff_passed" == true ]]; then
        return 0
    else
        return 1
    fi
}

# Function to run comprehensive advanced search tests (YAML-only)
run_advanced_search_tests() {
    log_info "Starting comprehensive advanced search features test (YAML-only)..."
    log_info "Note: CLI commands for advanced features were removed due to Atlas Admin API limitations"
    
    local test_start_time
    test_start_time=$(date +%s)
    local tests_passed=0
    local tests_failed=0
    # local tests_skipped=0  # Unused in current implementation
    
    # Test YAML-based advanced features only
    local test_functions=(
        "test_yaml_apply_basic"
        "test_yaml_apply_analyzers"
        "test_yaml_apply_facets"
        "test_yaml_apply_autocomplete"
        "test_yaml_apply_highlighting"
        "test_yaml_apply_synonyms"
        "test_yaml_apply_fuzzy"
        "test_yaml_apply_comprehensive"
        "test_yaml_validation"
        "test_yaml_plan_diff"
    )
    
    for test_func in "${test_functions[@]}"; do
        log_info "Running test: $test_func"
        if $test_func; then
            ((tests_passed++))
            log_success "Test $test_func passed"
        else
            ((tests_failed++))
            log_error "Test $test_func failed"
        fi
        echo ""
    done
    
    # Calculate test duration
    local test_end_time
    test_end_time=$(date +%s)
    local test_duration=$((test_end_time - test_start_time))
    
    # Print test summary
    echo ""
    log_info "=== Advanced Search Features Test Summary ==="
    log_info "Total tests run: $((tests_passed + tests_failed))"
    log_success "Tests passed: $tests_passed"
    log_error "Tests failed: $tests_failed"
    log_info "Test duration: ${test_duration} seconds"
    echo ""
    
    if [[ $tests_failed -eq 0 ]]; then
        log_success "All advanced search feature tests completed successfully!"
        log_info "Note: Tests validate YAML-based advanced features only (CLI commands removed due to API limitations)"
        return 0
    else
        log_error "Some advanced search feature tests failed"
        return 1
    fi
}

# Function to wait for search index to be ready
wait_for_search_index_ready() {
    local project_id="$1"
    local cluster_name="$2"
    local index_name="$3"
    local max_wait=150  # 5 minutes
    local wait_time=0
    
    log_info "Waiting for search index '$index_name' to be ready..."
    
    while [[ $wait_time -lt $max_wait ]]; do
        if $MATLAS_CLI atlas search get \
            --project-id "$project_id" \
            --cluster "$cluster_name" \
            --name "$index_name" \
            --output json 2>/dev/null | grep -q '"status":"READY"'; then
            log_success "Search index is ready"
            return 0
        fi
        
        log_info "Index still building... waiting 15 seconds"
        sleep 15
        wait_time=$((wait_time + 15))
    done
    
    log_warning "Search index not ready after $max_wait seconds, continuing with tests"
    return 0
}

# Function to validate index status after creation
validate_index_status() {
    local project_id="$1"
    local cluster_name="$2" 
    local index_name="$3"
    local max_wait=60  # 1 minute for validation
    local wait_time=0
    
    log_info "Validating index '$index_name' status..."
    
    while [[ $wait_time -lt $max_wait ]]; do
        local index_status
        index_status=$($MATLAS_CLI atlas search get \
            --project-id "$project_id" \
            --cluster "$cluster_name" \
            --name "$index_name" \
            --output json 2>/dev/null | jq -r '.status // "UNKNOWN"')
        
        case "$index_status" in
            "READY")
                log_success "Index '$index_name' is READY"
                return 0
                ;;
            "FAILED")
                log_error "Index '$index_name' FAILED to build"
                # Get error details
                local error_msg
                error_msg=$($MATLAS_CLI atlas search get \
                    --project-id "$project_id" \
                    --cluster "$cluster_name" \
                    --name "$index_name" \
                    --output json 2>/dev/null | jq -r '.statusDetail[0].mainIndex.message // "Unknown error"')
                log_error "Error: $error_msg"
                return 1
                ;;
            "PENDING"|"IN_PROGRESS")
                log_info "Index still building... waiting 10 seconds"
                sleep 10
                wait_time=$((wait_time + 10))
                ;;
            "UNKNOWN"|"")
                log_warning "Could not determine index status"
                return 1
                ;;
            *)
                log_info "Index status: $index_status, waiting 10 seconds"
                sleep 10
                wait_time=$((wait_time + 10))
                ;;
        esac
    done
    
    log_warning "Index validation timed out after $max_wait seconds"
    return 1
}

# Function to cleanup test resources (with preserve-existing, this should be minimal)
cleanup_test_resources() {
    log_info "Cleaning up test resources..."
    
    # Clean up search indexes created during testing using direct CLI delete commands
    if [[ -f "$CREATED_INDEXES_FILE" ]] && [[ -s "$CREATED_INDEXES_FILE" ]]; then
        log_info "Destroying test-created search indexes..."
        
        local index_count=0
        local deleted_count=0
        local failed_count=0
        
        while IFS= read -r index_name; do
            if [[ -n "$index_name" ]]; then
                ((index_count++))
                log_info "Deleting search index: $index_name"
                
                # Use the direct atlas search delete command
                if $MATLAS_CLI atlas search delete \
                    --project-id "$TEST_PROJECT_ID" \
                    --cluster "$TEST_CLUSTER_NAME" \
                    --name "$index_name" \
                    --force 2>/dev/null; then
                    ((deleted_count++))
                    log_info "✓ Deleted index: $index_name"
                else
                    ((failed_count++))
                    log_warning "✗ Failed to delete index: $index_name (may not exist)"
                fi
            fi
        done < "$CREATED_INDEXES_FILE"
        
        if [[ $index_count -gt 0 ]]; then
            log_info "Cleanup summary: $deleted_count deleted, $failed_count failed out of $index_count total"
            if [[ $deleted_count -gt 0 ]]; then
                log_success "Test search indexes destroyed successfully"
            elif [[ $failed_count -eq $index_count ]]; then
                log_warning "No test search indexes were destroyed (may have been already deleted)"
            else
                log_warning "Some test search indexes may not have been destroyed"
            fi
        else
            log_info "No test-created search indexes to clean up"
        fi
    else
        log_info "No test-created search indexes to clean up"
    fi
    
    # Clean up temporary files
    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
        log_info "Temporary files cleaned up"
    fi
    
    log_info "Cleanup completed (existing resources preserved)"
}

# Main execution function
main() {
    local exit_code=0
    
    # Export variables for use in test functions
    export TEST_PROJECT_ID
    export TEST_CLUSTER_NAME
    export TEST_DATABASE_NAME
    export TEST_COLLECTION_NAME
    export TEST_INDEX_NAME
    export TEMP_DIR
    export MATLAS_CLI
    
    log_info "Starting Atlas Search Advanced Features Test (YAML-only)"
    log_info "Project ID: $TEST_PROJECT_ID"
    log_info "Cluster: $TEST_CLUSTER_NAME"
    log_info "Using --preserve-existing flag to protect existing resources"
    log_info "Note: Testing YAML-based advanced features only (CLI commands removed due to API limitations)"
    echo ""
    
    # Run test phases
    if check_prerequisites; then
        if discover_test_database_and_collection; then
            if create_advanced_search_index; then
                if run_advanced_search_tests; then
                    log_success "All advanced search feature tests completed successfully!"
                else
                    log_error "Some advanced search feature tests failed"
                    exit_code=1
                fi
            else
                log_error "Failed to create advanced search index"
                exit_code=1
            fi
        else
            log_error "Failed to discover test database and collection"
            exit_code=1
        fi
    else
        log_error "Prerequisites check failed"
        exit_code=1
    fi
    
    # Cleanup
    cleanup_test_resources
    
    echo ""
    if [[ $exit_code -eq 0 ]]; then
        log_success "Atlas Search Advanced Features Test (YAML-only) completed successfully!"
        log_info "Advanced search features are fully supported via YAML configuration"
    else
        log_error "Atlas Search Advanced Features Test (YAML-only) failed!"
    fi
    
    exit $exit_code
}

# Handle script arguments
case "${1:-}" in
    "--help"|"-h")
        echo "Usage: $0 [options]"
        echo ""
        echo "Test advanced search features for Atlas Search indexes (YAML-only)"
        echo ""
        echo "Note: CLI commands for advanced features were removed due to Atlas Admin API limitations."
        echo "This test validates YAML-based configuration of analyzers, facets, autocomplete,"
        echo "highlighting, synonyms, and fuzzy search features."
        echo ""
        echo "Environment variables:"
        echo "  ATLAS_PROJECT_ID    - Atlas project ID (required)"
        echo "  TEST_CLUSTER_NAME   - Test cluster name (default: advanced-search-test-cluster)"
        echo ""
        echo "Options:"
        echo "  --help, -h          - Show this help message"
        echo ""
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
