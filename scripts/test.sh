#!/usr/bin/env bash

# matlas-cli Test Runner - Main Entry Point
# Simple, clean interface to run all types of tests

set -euo pipefail

# Colors
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'
readonly BOLD='\033[1m'

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

print_header() {
    echo -e "${BLUE}${BOLD}════════════════════════════════════════${NC}"
    echo -e "${BLUE}${BOLD} matlas-cli Test Runner${NC}"
    echo -e "${BLUE}${BOLD}════════════════════════════════════════${NC}"
}

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

show_usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

COMMANDS:
    unit        Run unit tests only
    integration Run integration tests only
    e2e         Run end-to-end tests only
    cluster     Run cluster lifecycle tests (creates real clusters)
    users       Run users lifecycle live tests (creates real users)
    network     Run network access lifecycle live tests
    projects    Run projects lifecycle live tests (creates real project)
    applydoc    Run ApplyDocument format tests (comprehensive coverage)
    all         Run all tests (unit + integration + e2e)
    comprehensive Run all tests including cluster and applydoc tests
    clean       Clean test cache and artifacts
    help        Show this help message

OPTIONS:
    --verbose         Enable verbose output
    --coverage        Generate coverage reports
    --dry-run         Show what would be tested without running
    --include-clusters Include real cluster tests in 'all' and 'e2e' commands

EXAMPLES:
    $0 unit                 # Run unit tests
    $0 integration          # Run integration tests
    $0 cluster              # Run cluster lifecycle tests 
    $0 applydoc             # Run ApplyDocument format tests
    $0 e2e                  # Run e2e tests (users/network only)
    $0 e2e --include-clusters  # Run e2e tests with real clusters
    $0 users                # Run live users lifecycle tests
    $0 network              # Run live network lifecycle tests
    $0 projects             # Run live projects lifecycle tests
    $0 all                  # Run all tests (no clusters)
    $0 comprehensive        # Run all tests including cluster and applydoc
    $0 all --coverage       # Run all tests with coverage
    $0 clean                # Clean test cache

EOF
}

load_environment() {
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        print_info "Loading environment from .env file"
        set -o allexport
        source "$PROJECT_ROOT/.env"
        set +o allexport
    fi
}

run_test_type() {
    local test_type="$1"
    local script_path="$SCRIPT_DIR/test/${test_type}.sh"
    
    if [[ ! -f "$script_path" ]]; then
        print_error "Test script not found: $script_path"
        return 1
    fi
    
    print_info "Running $test_type tests..."
    if "$script_path" "$@"; then
        print_success "$test_type tests passed"
        return 0
    else
        print_error "$test_type tests failed"
        return 1
    fi
}

main() {
    local command="${1:-help}"
    shift || true
    
    # Check for --include-clusters flag in arguments
    local include_clusters=false
    local args=()
    for arg in "$@"; do
        if [[ "$arg" == "--include-clusters" ]]; then
            include_clusters=true
        else
            args+=("$arg")
        fi
    done
    
    print_header
    load_environment
    
    case "$command" in
        unit|integration)
            run_test_type "$command" "${args[@]}"
            ;;
        e2e)
            if [[ "$include_clusters" == "true" ]]; then
                print_warning "⚠️  Including real cluster tests - this may incur costs!"
                run_test_type "$command" --include-clusters "${args[@]}"
            else
                run_test_type "$command" "${args[@]}"
            fi
            ;;
        cluster)
            print_info "Running cluster lifecycle tests..."
            print_warning "⚠️  WARNING: Creates real Atlas clusters and may incur costs!"
            if "$SCRIPT_DIR/test/cluster-lifecycle.sh" "${args[@]}"; then
                print_success "Cluster lifecycle tests passed"
            else
                print_error "Cluster lifecycle tests failed"
                return 1
            fi
            ;;
        users)
            print_info "Running users lifecycle tests (live)..."
            if "$SCRIPT_DIR/test/users-lifecycle.sh" "${args[@]}"; then
                print_success "Users lifecycle tests passed"
            else
                print_error "Users lifecycle tests failed"
                return 1
            fi
            ;;
        network)
            print_info "Running network lifecycle tests (live)..."
            if "$SCRIPT_DIR/test/network-lifecycle.sh" "${args[@]}"; then
                print_success "Network lifecycle tests passed"
            else
                print_error "Network lifecycle tests failed"
                return 1
            fi
            ;;
        projects)
            print_info "Running projects lifecycle tests (live)..."
            print_warning "⚠️  WARNING: Creates and deletes a real Atlas project!"
            if "$SCRIPT_DIR/test/projects-lifecycle.sh" "${args[@]}"; then
                print_success "Projects lifecycle tests passed"
            else
                print_error "Projects lifecycle tests failed"
                return 1
            fi
            ;;
        applydoc)
            print_info "Running ApplyDocument format tests..."
            print_info "Testing comprehensive ApplyDocument YAML format coverage"
            if "$SCRIPT_DIR/test/applydocument-test.sh" "${args[@]}"; then
                print_success "ApplyDocument format tests passed"
            else
                print_error "ApplyDocument format tests failed"
                return 1
            fi
            ;;
        comprehensive)
            local failed=0
            print_info "Running comprehensive test suite (all test types)..."
            run_test_type "unit" "${args[@]}" || ((failed++))
            run_test_type "integration" "${args[@]}" || ((failed++))
            run_test_type "e2e" "${args[@]}" || ((failed++))
            
            print_warning "⚠️  Including ApplyDocument and cluster tests - may incur costs!"
            "$SCRIPT_DIR/test/applydocument-test.sh" "${args[@]}" || ((failed++))
            "$SCRIPT_DIR/test/cluster-lifecycle.sh" "${args[@]}" || ((failed++))
            
            if [[ $failed -eq 0 ]]; then
                print_success "All comprehensive tests passed!"
                return 0
            else
                print_error "$failed test type(s) failed"
                return 1
            fi
            ;;
        all)
            local failed=0
            run_test_type "unit" "${args[@]}" || ((failed++))
            run_test_type "integration" "${args[@]}" || ((failed++))
            
            if [[ "$include_clusters" == "true" ]]; then
                print_warning "⚠️  Including real cluster tests - this may incur costs!"
                run_test_type "e2e" --include-clusters "${args[@]}" || ((failed++))
            else
                run_test_type "e2e" "${args[@]}" || ((failed++))
            fi
            
            if [[ $failed -eq 0 ]]; then
                print_success "All tests passed!"
                return 0
            else
                print_error "$failed test type(s) failed"
                return 1
            fi
            ;;
        clean)
            "$SCRIPT_DIR/utils/clean.sh" "${args[@]}"
            ;;
        help|-h|--help)
            show_usage
            ;;
        *)
            print_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"