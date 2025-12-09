#!/usr/bin/env bash

# DAG Feature Testing for matlas-cli
# Tests analyze, visualize, and optimize commands on real infrastructure
# WARNING: Creates real Atlas resources - use only in test environments

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
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/dag-feature"
REGION="${TEST_REGION:-US_EAST_1}"

# Test state
CLEANUP_REQUIRED=false
CLUSTER_NAME=""
CONFIG_FILE=""

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

# Cleanup function
cleanup() {
    if [[ "$CLEANUP_REQUIRED" == "true" ]] && [[ -n "$CONFIG_FILE" ]] && [[ -f "$CONFIG_FILE" ]]; then
        print_header "Cleanup"
        print_info "Cleaning up test resources..."
        
        if "$PROJECT_ROOT/matlas" infra destroy -f "$CONFIG_FILE" \
            --project-id "$ATLAS_PROJECT_ID" \
            --auto-approve \
            --force 2>&1 | tee "$TEST_REPORTS_DIR/cleanup.log"; then
            print_success "Cleanup completed"
        else
            print_warning "Cleanup may have failed - check logs"
        fi
    fi
    
    # Clean up temporary files
    if [[ -n "$CONFIG_FILE" ]] && [[ -f "$CONFIG_FILE" ]]; then
        rm -f "$CONFIG_FILE"
    fi
}

trap cleanup EXIT

# Environment validation
check_environment() {
    print_info "Validating DAG feature test environment..."
    
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
    
    print_success "Environment validation completed"
    return 0
}

# Generate test configuration YAML
generate_test_config() {
    local timestamp=$(date +%s | tail -c 6)
    CLUSTER_NAME="dag-test-${timestamp}"
    CONFIG_FILE="$TEST_REPORTS_DIR/dag-test-config.yaml"
    
    print_info "Generating test configuration with cluster: $CLUSTER_NAME"
    
    cat > "$CONFIG_FILE" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: dag-feature-test
  labels:
    matlas-mongodb-com-project-id: "$ATLAS_PROJECT_ID"
    test: dag-feature

resources:
  # Network access entries (no dependencies - can run in parallel)
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: dag-test-network-1
      labels:
        atlas.mongodb.com/project-id: "$ATLAS_PROJECT_ID"
    spec:
      ipAddress: "203.0.113.0/24"
      comment: "DAG Test Network 1"
      
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: dag-test-network-2
      labels:
        atlas.mongodb.com/project-id: "$ATLAS_PROJECT_ID"
    spec:
      ipAddress: "198.51.100.0/24"
      comment: "DAG Test Network 2"
  
  # Cluster (depends on project, blocks users)
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: $CLUSTER_NAME
      labels:
        atlas.mongodb.com/project-id: "$ATLAS_PROJECT_ID"
    spec:
      name: "$CLUSTER_NAME"
      clusterType: "REPLICASET"
      providerSettings:
        providerName: "AWS"
        regionName: "$REGION"
        instanceSizeName: "M10"
      diskSizeGB: 10
      
  # Database user (depends on cluster)
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: dag-test-user
      labels:
        atlas.mongodb.com/project-id: "$ATLAS_PROJECT_ID"
    spec:
      username: "dagtestuser"
      password: "DagTest123!"
      databaseName: "admin"
      roles:
        - roleName: "readWrite"
          databaseName: "test"
        - roleName: "read"
          databaseName: "admin"
EOF

    print_success "Test configuration generated: $CONFIG_FILE"
}

# Test DAG analyze command
test_dag_analyze() {
    print_header "Testing DAG Analyze Command"
    
    # Test 1: Basic text analysis
    print_subheader "Test 1: Basic Text Analysis"
    if "$PROJECT_ROOT/matlas" infra analyze \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        2>&1 | tee "$TEST_REPORTS_DIR/analyze-text.log"; then
        print_success "Text analysis completed"
    else
        print_error "Text analysis failed"
        return 1
    fi
    
    # Test 2: JSON output
    print_subheader "Test 2: JSON Output"
    if "$PROJECT_ROOT/matlas" infra analyze \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --format json \
        --output-file "$TEST_REPORTS_DIR/analyze.json" 2>&1; then
        print_success "JSON analysis saved"
        
        # Validate JSON
        if command -v jq >/dev/null 2>&1; then
            if jq empty "$TEST_REPORTS_DIR/analyze.json" 2>/dev/null; then
                print_success "JSON is valid"
                
                # Extract key metrics
                local node_count=$(jq -r '.nodeCount' "$TEST_REPORTS_DIR/analyze.json")
                local critical_path_duration=$(jq -r '.criticalPathDuration' "$TEST_REPORTS_DIR/analyze.json")
                local has_cycles=$(jq -r '.hasCycles' "$TEST_REPORTS_DIR/analyze.json")
                
                print_info "Node count: $node_count"
                print_info "Critical path duration: $critical_path_duration ns"
                print_info "Has cycles: $has_cycles"
                
                # Verify expected values
                if [[ "$node_count" -ge 4 ]]; then
                    print_success "Node count is correct (expected >= 4, got $node_count)"
                else
                    print_error "Node count is incorrect (expected >= 4, got $node_count)"
                    return 1
                fi
                
                if [[ "$has_cycles" == "false" ]]; then
                    print_success "No cycles detected (as expected)"
                else
                    print_error "Unexpected cycles detected"
                    return 1
                fi
            else
                print_error "Invalid JSON output"
                return 1
            fi
        else
            print_warning "jq not available, skipping JSON validation"
        fi
    else
        print_error "JSON analysis failed"
        return 1
    fi
    
    # Test 3: Markdown output
    print_subheader "Test 3: Markdown Output"
    if "$PROJECT_ROOT/matlas" infra analyze \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --format markdown \
        --output-file "$TEST_REPORTS_DIR/analyze.md" 2>&1; then
        print_success "Markdown analysis saved"
        
        # Check for expected sections
        if grep -q "# Dependency Analysis Report" "$TEST_REPORTS_DIR/analyze.md" && \
           grep -q "## Overview" "$TEST_REPORTS_DIR/analyze.md" && \
           grep -q "## Critical Path" "$TEST_REPORTS_DIR/analyze.md"; then
            print_success "Markdown has expected sections"
        else
            print_error "Markdown is missing expected sections"
            return 1
        fi
    else
        print_error "Markdown analysis failed"
        return 1
    fi
    
    # Test 4: Risk analysis
    print_subheader "Test 4: Risk Analysis"
    if "$PROJECT_ROOT/matlas" infra analyze \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --show-risk \
        2>&1 | tee "$TEST_REPORTS_DIR/analyze-risk.log"; then
        print_success "Risk analysis completed"
    else
        print_error "Risk analysis failed"
        return 1
    fi
    
    print_success "All analyze tests passed"
    return 0
}

# Test DAG visualize command
test_dag_visualize() {
    print_header "Testing DAG Visualize Command"
    
    # Test 1: ASCII visualization
    print_subheader "Test 1: ASCII Visualization"
    if "$PROJECT_ROOT/matlas" infra visualize \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --output-file "$TEST_REPORTS_DIR/visualize-ascii.txt" \
        2>&1 | tee "$TEST_REPORTS_DIR/visualize-ascii.log"; then
        print_success "ASCII visualization saved"
        
        # Check content
        if [[ -f "$TEST_REPORTS_DIR/visualize-ascii.txt" ]] && \
           grep -q "Dependency Graph" "$TEST_REPORTS_DIR/visualize-ascii.txt"; then
            print_success "ASCII visualization has expected content"
        else
            print_error "ASCII visualization is invalid"
            return 1
        fi
    else
        print_error "ASCII visualization failed"
        return 1
    fi
    
    # Test 2: DOT format
    print_subheader "Test 2: DOT (Graphviz) Format"
    if "$PROJECT_ROOT/matlas" infra visualize \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --format dot \
        --output-file "$TEST_REPORTS_DIR/visualize.dot" \
        2>&1 | tee "$TEST_REPORTS_DIR/visualize-dot.log"; then
        print_success "DOT visualization saved"
        
        # Validate DOT format
        if [[ -f "$TEST_REPORTS_DIR/visualize.dot" ]] && \
           grep -q "digraph G" "$TEST_REPORTS_DIR/visualize.dot"; then
            print_success "DOT file has valid format"
            
            # Try to render if graphviz is available
            if command -v dot >/dev/null 2>&1; then
                if dot -Tpng "$TEST_REPORTS_DIR/visualize.dot" \
                    -o "$TEST_REPORTS_DIR/visualize.png" 2>/dev/null; then
                    print_success "DOT rendered to PNG successfully"
                else
                    print_warning "Failed to render DOT to PNG"
                fi
            else
                print_info "Graphviz not available, skipping PNG rendering"
            fi
        else
            print_error "DOT file is invalid"
            return 1
        fi
    else
        print_error "DOT visualization failed"
        return 1
    fi
    
    # Test 3: Mermaid format
    print_subheader "Test 3: Mermaid Format"
    if "$PROJECT_ROOT/matlas" infra visualize \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --format mermaid \
        --output-file "$TEST_REPORTS_DIR/visualize.mmd" \
        2>&1 | tee "$TEST_REPORTS_DIR/visualize-mermaid.log"; then
        print_success "Mermaid visualization saved"
        
        # Validate Mermaid format
        if [[ -f "$TEST_REPORTS_DIR/visualize.mmd" ]] && \
           grep -q "graph" "$TEST_REPORTS_DIR/visualize.mmd"; then
            print_success "Mermaid file has valid format"
        else
            print_error "Mermaid file is invalid"
            return 1
        fi
    else
        print_error "Mermaid visualization failed"
        return 1
    fi
    
    # Test 4: JSON format
    print_subheader "Test 4: JSON Format"
    if "$PROJECT_ROOT/matlas" infra visualize \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --format json \
        --output-file "$TEST_REPORTS_DIR/visualize.json" \
        2>&1 | tee "$TEST_REPORTS_DIR/visualize-json.log"; then
        print_success "JSON visualization saved"
        
        # Validate JSON
        if command -v jq >/dev/null 2>&1; then
            if jq empty "$TEST_REPORTS_DIR/visualize.json" 2>/dev/null; then
                print_success "JSON is valid"
            else
                print_error "Invalid JSON output"
                return 1
            fi
        fi
    else
        print_error "JSON visualization failed"
        return 1
    fi
    
    # Test 5: Options (highlight critical path, show levels)
    print_subheader "Test 5: Visualization Options"
    if "$PROJECT_ROOT/matlas" infra visualize \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --highlight-critical-path \
        --show-levels \
        --output-file "$TEST_REPORTS_DIR/visualize-options.txt" \
        2>&1; then
        print_success "Visualization with options completed"
    else
        print_error "Visualization with options failed"
        return 1
    fi
    
    print_success "All visualize tests passed"
    return 0
}

# Test DAG optimize command
test_dag_optimize() {
    print_header "Testing DAG Optimize Command"
    
    # Test 1: Basic optimization
    print_subheader "Test 1: Basic Optimization"
    if "$PROJECT_ROOT/matlas" infra optimize \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        2>&1 | tee "$TEST_REPORTS_DIR/optimize.log"; then
        print_success "Optimization analysis completed"
        
        # Check for expected content
        if grep -q "Optimization Suggestions Report" "$TEST_REPORTS_DIR/optimize.log"; then
            print_success "Optimization report has expected format"
        else
            print_error "Optimization report is invalid"
            return 1
        fi
    else
        print_error "Optimization analysis failed"
        return 1
    fi
    
    print_success "All optimize tests passed"
    return 0
}

# Test actual infrastructure apply
test_apply_infrastructure() {
    print_header "Testing Infrastructure Apply"
    
    print_subheader "Applying Configuration"
    print_warning "This will create real Atlas resources (cluster, users, network access)"
    
    if "$PROJECT_ROOT/matlas" infra apply \
        -f "$CONFIG_FILE" \
        --project-id "$ATLAS_PROJECT_ID" \
        --auto-approve \
        2>&1 | tee "$TEST_REPORTS_DIR/apply.log"; then
        print_success "Infrastructure apply completed"
        CLEANUP_REQUIRED=true
        
        # Verify cluster was created
        print_subheader "Verifying Cluster Creation"
        if "$PROJECT_ROOT/matlas" atlas clusters get "$CLUSTER_NAME" \
            --project-id "$ATLAS_PROJECT_ID" \
            --output json > "$TEST_REPORTS_DIR/cluster-state.json" 2>&1; then
            print_success "Cluster verified: $CLUSTER_NAME"
            
            local cluster_status=$(jq -r '.stateName // "UNKNOWN"' "$TEST_REPORTS_DIR/cluster-state.json")
            print_info "Cluster status: $cluster_status"
        else
            print_warning "Could not verify cluster (may still be creating)"
        fi
        
        return 0
    else
        print_error "Infrastructure apply failed"
        return 1
    fi
}

# Main test execution
main() {
    print_header "DAG Feature Test Suite"
    echo
    
    # Check environment
    if ! check_environment; then
        print_error "Environment validation failed"
        exit 1
    fi
    echo
    
    # Generate test configuration
    if ! generate_test_config; then
        print_error "Failed to generate test configuration"
        exit 1
    fi
    echo
    
    # Test DAG commands (without creating resources)
    local all_passed=true
    
    if ! test_dag_analyze; then
        all_passed=false
        print_error "Analyze tests failed"
    fi
    echo
    
    if ! test_dag_visualize; then
        all_passed=false
        print_error "Visualize tests failed"
    fi
    echo
    
    if ! test_dag_optimize; then
        all_passed=false
        print_error "Optimize tests failed"
    fi
    echo
    
    if [[ "$all_passed" != "true" ]]; then
        print_error "Some DAG tests failed"
        exit 1
    fi
    
    # Optionally apply infrastructure (requires confirmation)
    if [[ "${SKIP_APPLY:-false}" != "true" ]]; then
        print_header "Infrastructure Apply"
        print_warning "The following step will create real Atlas resources"
        print_info "To skip this step, set SKIP_APPLY=true"
        echo
        
        if ! test_apply_infrastructure; then
            print_error "Infrastructure apply test failed"
            exit 1
        fi
    else
        print_info "Skipping infrastructure apply (SKIP_APPLY=true)"
    fi
    
    # Summary
    echo
    print_header "Test Summary"
    print_success "All DAG feature tests passed!"
    echo
    print_info "Test reports saved to: $TEST_REPORTS_DIR"
    print_info "Files generated:"
    ls -lh "$TEST_REPORTS_DIR" 2>/dev/null || true
    
    return 0
}

# Run main function
main "$@"
