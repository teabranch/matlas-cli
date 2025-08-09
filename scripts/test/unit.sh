#!/usr/bin/env bash

# Unit Tests Runner
# Fast, isolated tests with no external dependencies

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
readonly TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/unit"

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

run_unit_tests() {
    local coverage=false
    local verbose=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --coverage) coverage=true; shift ;;
            --verbose) verbose=true; shift ;;
            *) shift ;;
        esac
    done
    
    print_info "Running unit tests..."
    
    # Setup
    mkdir -p "$TEST_REPORTS_DIR"
    cd "$PROJECT_ROOT"
    
    # Clean cache
    print_info "Cleaning test cache..."
    go clean -testcache
    
    # Build test command
    local test_cmd="go test"
    local test_args=("-timeout=10m" "-race")
    
    if [[ "$verbose" == "true" ]]; then
        test_args+=("-v")
    fi
    
    if [[ "$coverage" == "true" ]]; then
        test_args+=("-coverprofile=$TEST_REPORTS_DIR/coverage.out" "-covermode=atomic")
    fi
    
    test_args+=("./internal/..." "./cmd/...")
    
    # Run tests
    if $test_cmd "${test_args[@]}" 2>&1 | tee "$TEST_REPORTS_DIR/output.log"; then
        print_success "Unit tests passed"
        
        # Generate coverage report if requested
        if [[ "$coverage" == "true" && -f "$TEST_REPORTS_DIR/coverage.out" ]]; then
            go tool cover -html="$TEST_REPORTS_DIR/coverage.out" -o "$TEST_REPORTS_DIR/coverage.html"
            local coverage_pct=$(go tool cover -func="$TEST_REPORTS_DIR/coverage.out" | grep total | awk '{print $3}')
            print_info "Coverage: $coverage_pct"
            print_success "Coverage report: $TEST_REPORTS_DIR/coverage.html"
        fi
        
        return 0
    else
        print_error "Unit tests failed"
        print_info "Output saved to: $TEST_REPORTS_DIR/output.log"
        return 1
    fi
}

run_unit_tests "$@"