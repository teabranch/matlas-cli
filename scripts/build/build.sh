#!/usr/bin/env bash

# Build Script
# Builds the matlas binary with proper versioning

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

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

get_version() {
    # Try to get version from git
    if git describe --tags --abbrev=0 2>/dev/null; then
        return 0
    elif git rev-parse --short HEAD 2>/dev/null; then
        return 0
    else
        echo "dev"
        return 0
    fi
}

build_binary() {
    local output_name="${1:-matlas}"
    local version
    version=$(get_version)
    
    print_info "Building $output_name (version: $version)..."
    
    cd "$PROJECT_ROOT"
    
    # Build flags
    local ldflags="-X main.version=$version -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    
    if go build -ldflags "$ldflags" -o "$output_name"; then
        print_success "Build completed: $output_name"
        
        # Show binary info
        local size=$(ls -lah "$output_name" | awk '{print $5}')
        print_info "Binary size: $size"
        
        # Test the binary
        if "./$output_name" version &> /dev/null; then
            print_success "Binary is functional"
        else
            print_warning "Binary may not be functional"
        fi
        
        return 0
    else
        print_error "Build failed"
        return 1
    fi
}

cross_compile() {
    local version
    version=$(get_version)
    
    print_info "Cross-compiling for multiple platforms..."
    
    cd "$PROJECT_ROOT"
    mkdir -p dist
    
    local platforms=(
        "linux/amd64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )
    
    for platform in "${platforms[@]}"; do
        local os=$(echo "$platform" | cut -d/ -f1)
        local arch=$(echo "$platform" | cut -d/ -f2)
        local output="dist/matlas-${os}-${arch}"
        
        if [[ "$os" == "windows" ]]; then
            output="${output}.exe"
        fi
        
        print_info "Building for $os/$arch..."
        
        local ldflags="-X main.version=$version -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        
        if GOOS="$os" GOARCH="$arch" go build -ldflags "$ldflags" -o "$output"; then
            print_success "Built: $output"
        else
            print_error "Failed to build for $os/$arch"
        fi
    done
    
    print_success "Cross-compilation completed"
    print_info "Binaries in: dist/"
}

show_usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

Build the matlas binary.

COMMANDS:
    build       Build binary for current platform (default)
    cross       Cross-compile for multiple platforms
    clean       Clean build artifacts
    help        Show this help

OPTIONS:
    -o NAME     Output binary name (default: matlas)

EXAMPLES:
    $0                    # Build for current platform
    $0 build -o mymatlas  # Build with custom name
    $0 cross              # Cross-compile for all platforms
    $0 clean              # Clean build artifacts

EOF
}

clean_build() {
    print_info "Cleaning build artifacts..."
    
    cd "$PROJECT_ROOT"
    
    # Remove binary
    if [[ -f "matlas" ]]; then
        rm -f matlas
        print_success "Removed matlas binary"
    fi
    
    # Remove dist directory
    if [[ -d "dist" ]]; then
        rm -rf dist
        print_success "Removed dist directory"
    fi
    
    print_success "Build artifacts cleaned"
}

main() {
    local command="${1:-build}"
    shift || true
    
    case "$command" in
        build)
            local output_name="matlas"
            
            # Parse options
            while [[ $# -gt 0 ]]; do
                case $1 in
                    -o)
                        output_name="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            build_binary "$output_name"
            ;;
        cross)
            cross_compile
            ;;
        clean)
            clean_build
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