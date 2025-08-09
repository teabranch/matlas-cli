#!/usr/bin/env bash

# Clean Test Cache and Artifacts
# Removes test cache, reports, and leftover resources

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
readonly TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports"

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

clean_test_cache() {
    print_info "Cleaning Go test cache..."
    cd "$PROJECT_ROOT"
    go clean -testcache
    print_success "Test cache cleaned"
}

clean_test_reports() {
    print_info "Cleaning test reports..."
    if [[ -d "$TEST_REPORTS_DIR" ]]; then
        rm -rf "$TEST_REPORTS_DIR"
        print_success "Test reports cleaned"
    else
        print_info "No test reports to clean"
    fi
}

clean_build_artifacts() {
    print_info "Cleaning build artifacts..."
    cd "$PROJECT_ROOT"
    
    # Remove coverage files
    rm -f coverage.out coverage.html
    
    # Remove binary if it exists
    if [[ -f "matlas" ]]; then
        rm -f matlas
        print_success "Removed matlas binary"
    fi
    
    print_success "Build artifacts cleaned"
}

cleanup_test_resources() {
    print_warning "Checking for leftover test resources..."
    
    # Load environment
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        set -o allexport
        source "$PROJECT_ROOT/.env"
        set +o allexport
    fi
    
    # Check if we have credentials to clean up resources
    if [[ -z "${ATLAS_PUB_KEY:-}" || -z "${ATLAS_API_KEY:-}" || -z "${ATLAS_PROJECT_ID:-}" ]]; then
        print_warning "No Atlas credentials - cannot check for leftover resources"
        return 0
    fi
    
    # Build matlas if needed
    if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
        print_info "Building matlas to check for resources..."
        cd "$PROJECT_ROOT"
        if ! go build -o matlas; then
            print_warning "Cannot build matlas - skipping resource cleanup"
            return 0
        fi
    fi
    
    local found_test_resources=0
    
    # Check for test users
    print_info "Checking for test database users..."
    if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null | grep -E "(test-|e2e-|integration-)" > /dev/null; then
        print_warning "Found test database users"
        echo "Run the following to clean them up:"
        echo "  matlas atlas users list --project-id $ATLAS_PROJECT_ID | grep -E '(test-|e2e-|integration-)'"
        ((found_test_resources++))
    fi
    
    # Check for test network access
    print_info "Checking for test network access entries..."
    if "$PROJECT_ROOT/matlas" atlas network list --project-id "$ATLAS_PROJECT_ID" 2>/dev/null | grep -i test > /dev/null; then
        print_warning "Found test network access entries"
        echo "Check network access entries with test comments:"
        echo "  matlas atlas network list --project-id $ATLAS_PROJECT_ID"
        ((found_test_resources++))
    fi
    
    if [[ $found_test_resources -eq 0 ]]; then
        print_success "No test resources found in Atlas"
    else
        print_warning "$found_test_resources type(s) of test resources found"
        print_info "Consider cleaning them up manually"
    fi
}

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Clean test cache, reports, and artifacts.

OPTIONS:
    cache      Clean Go test cache only
    reports    Clean test reports only  
    build      Clean build artifacts only
    resources  Check for leftover test resources in Atlas
    all        Clean everything (default)
    help       Show this help

EXAMPLES:
    $0              # Clean everything
    $0 cache        # Clean test cache only
    $0 reports      # Clean test reports only
    $0 resources    # Check for test resources

EOF
}

main() {
    local mode="${1:-all}"
    
    case "$mode" in
        cache)
            clean_test_cache
            ;;
        reports)
            clean_test_reports
            ;;
        build)
            clean_build_artifacts
            ;;
        resources)
            cleanup_test_resources
            ;;
        all|*)
            clean_test_cache
            clean_test_reports
            clean_build_artifacts
            cleanup_test_resources
            print_success "All cleanup completed"
            ;;
        help|-h|--help)
            show_usage
            ;;
    esac
}

main "$@"