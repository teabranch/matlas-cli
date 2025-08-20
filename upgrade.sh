#!/usr/bin/env bash

# matlas-cli upgrade script
# Updates existing installation to the latest version

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Configuration
readonly BINARY_NAME="matlas"

# Helper functions
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

# Find installed binary
find_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        which "$BINARY_NAME"
    else
        return 1
    fi
}

# Get current version
get_current_version() {
    local binary_path="$1"
    if [[ -x "$binary_path" ]]; then
        "$binary_path" version 2>/dev/null | head -1 | awk '{print $NF}' || echo "unknown"
    else
        echo "unknown"
    fi
}

# Get latest version from GitHub
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -s "https://api.github.com/repos/teabranch/matlas-cli/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/teabranch/matlas-cli/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"([^"]+)".*/\1/'
    else
        print_error "Neither curl nor wget found. Please install one of them."
        return 1
    fi
}

# Compare versions
compare_versions() {
    local current="$1"
    local latest="$2"
    
    # Remove 'v' prefix if present
    current="${current#v}"
    latest="${latest#v}"
    
    if [[ "$current" == "$latest" ]]; then
        return 0  # Same version
    else
        return 1  # Different version
    fi
}

# Show usage information
show_usage() {
    cat << EOF
matlas-cli Upgrade Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -v, --version VERSION    Upgrade to specific version (default: latest)
    --force                 Force upgrade even if versions match
    -h, --help             Show this help

EXAMPLES:
    $0                     # Upgrade to latest version
    $0 -v v1.2.3          # Upgrade to specific version
    $0 --force            # Force reinstall current version

EOF
}

# Main upgrade function
main() {
    local target_version=""
    local force=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                target_version="$2"
                shift 2
                ;;
            --force)
                force=true
                shift
                ;;
            -h|--help)
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
    
    print_info "matlas-cli Upgrade Script"
    print_info "========================="
    
    # Find existing installation
    local binary_path
    if ! binary_path=$(find_installation); then
        print_error "$BINARY_NAME is not installed or not in PATH"
        print_info "Please run the installation script first:"
        print_info "  curl -fsSL https://raw.githubusercontent.com/teabranch/matlas-cli/main/install.sh | bash"
        exit 1
    fi
    
    print_info "Found installation: $binary_path"
    
    # Get current version
    local current_version
    current_version=$(get_current_version "$binary_path")
    print_info "Current version: $current_version"
    
    # Get target version
    if [[ -z "$target_version" ]]; then
        print_info "Fetching latest version..."
        if ! target_version=$(get_latest_version); then
            print_error "Failed to fetch latest version"
            exit 1
        fi
    fi
    
    print_info "Target version: $target_version"
    
    # Check if upgrade is needed
    if [[ "$force" == false ]] && compare_versions "$current_version" "$target_version"; then
        print_success "Already running the latest version ($current_version)"
        print_info "Use --force to reinstall"
        exit 0
    fi
    
    # Determine install directory
    local install_dir
    install_dir=$(dirname "$binary_path")
    print_info "Install directory: $install_dir"
    
    # Download and run installation script with appropriate flags
    print_info "Downloading installation script..."
    
    local install_flags="--version $target_version --dir $install_dir"
    
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://raw.githubusercontent.com/teabranch/matlas-cli/main/install.sh" | bash -s -- $install_flags
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://raw.githubusercontent.com/teabranch/matlas-cli/main/install.sh" | bash -s -- $install_flags
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
    
    # Verify upgrade
    local new_version
    new_version=$(get_current_version "$binary_path")
    
    if [[ "$new_version" == "${target_version#v}" ]] || [[ "$new_version" == "$target_version" ]]; then
        print_success "Upgrade completed successfully!"
        print_info "Updated from $current_version to $new_version"
    else
        print_warning "Upgrade may not have completed successfully"
        print_info "Expected: $target_version, Got: $new_version"
    fi
}

# Run main function with all arguments
main "$@"
