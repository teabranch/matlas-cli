#!/usr/bin/env bash

# matlas-cli uninstallation script
# Supports macOS and Linux

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Configuration
readonly BINARY_NAME="matlas"
readonly DEFAULT_INSTALL_DIR="/usr/local/bin"

# Helper functions
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

# Find all installed binaries
find_installations() {
    local installations=()
    
    # Common installation directories
    local search_dirs=(
        "/usr/local/bin"
        "/usr/bin"
        "$HOME/.local/bin"
        "$HOME/bin"
    )
    
    # Add custom directory if specified
    if [[ -n "${MATLAS_INSTALL_DIR:-}" ]]; then
        search_dirs+=("$MATLAS_INSTALL_DIR")
    fi
    
    # Search PATH directories
    IFS=':' read -ra path_dirs <<< "$PATH"
    for dir in "${path_dirs[@]}"; do
        if [[ -n "$dir" && -d "$dir" ]]; then
            search_dirs+=("$dir")
        fi
    done
    
    # Remove duplicates and search for binary
    local unique_dirs=($(printf "%s\n" "${search_dirs[@]}" | sort -u))
    
    for dir in "${unique_dirs[@]}"; do
        local binary_path="$dir/$BINARY_NAME"
        if [[ -f "$binary_path" && -x "$binary_path" ]]; then
            installations+=("$binary_path")
        fi
    done
    
    printf "%s\n" "${installations[@]}"
}

# Remove binary from specified location
remove_binary() {
    local binary_path="$1"
    local dir=$(dirname "$binary_path")
    
    if [[ ! -w "$dir" && $EUID -ne 0 ]]; then
        print_warning "Removal from '$dir' requires root access"
        print_info "You may need to run: sudo $0"
        return 1
    fi
    
    if [[ -f "$binary_path" ]]; then
        print_info "Removing $binary_path..."
        rm -f "$binary_path"
        print_success "Removed $binary_path"
        return 0
    else
        print_warning "$binary_path not found"
        return 1
    fi
}

# Remove PATH entries from shell configuration files
remove_from_path() {
    local install_dir="$1"
    local modified=false
    
    # Shell configuration files to check
    local config_files=(
        "$HOME/.bashrc"
        "$HOME/.bash_profile"
        "$HOME/.zshrc"
        "$HOME/.config/fish/config.fish"
    )
    
    for config_file in "${config_files[@]}"; do
        if [[ -f "$config_file" ]]; then
            # Check if the install directory is mentioned in the config
            if grep -q "$install_dir" "$config_file"; then
                print_info "Checking $config_file for PATH entries..."
                
                # Create backup
                cp "$config_file" "${config_file}.bak.$(date +%s)"
                
                # Remove lines containing the install directory
                if [[ "$config_file" == *"config.fish" ]]; then
                    # Fish shell syntax
                    sed -i.tmp "/set.*PATH.*$install_dir/d" "$config_file" && rm "${config_file}.tmp"
                else
                    # Bash/Zsh syntax
                    sed -i.tmp "/export.*PATH.*$install_dir/d" "$config_file" && rm "${config_file}.tmp"
                fi
                
                print_success "Removed PATH entries from $config_file"
                print_info "Backup created: ${config_file}.bak.$(date +%s)"
                modified=true
            fi
        fi
    done
    
    if [[ "$modified" == true ]]; then
        print_warning "Please restart your shell or source your shell configuration"
    fi
}

# Remove configuration directory
remove_config() {
    local config_dir="$HOME/.matlas"
    
    if [[ -d "$config_dir" ]]; then
        print_info "Configuration directory found: $config_dir"
        read -p "Remove configuration directory? [y/N]: " -n 1 -r
        echo
        
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_info "Removing configuration directory..."
            rm -rf "$config_dir"
            print_success "Removed configuration directory"
        else
            print_info "Keeping configuration directory"
        fi
    else
        print_info "No configuration directory found"
    fi
}

# Show usage information
show_usage() {
    cat << EOF
matlas-cli Uninstallation Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --keep-config           Keep configuration directory (~/.matlas)
    --force                 Remove all installations without confirmation
    -h, --help             Show this help

EXAMPLES:
    $0                     # Interactive uninstallation
    $0 --force            # Remove all without confirmation
    $0 --keep-config      # Remove binary but keep config

EOF
}

# Main uninstallation function
main() {
    local keep_config=false
    local force=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --keep-config)
                keep_config=true
                shift
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
    
    print_info "matlas-cli Uninstallation Script"
    print_info "================================="
    
    # Find all installations
    local installations
    mapfile -t installations < <(find_installations)
    
    if [[ ${#installations[@]} -eq 0 ]]; then
        print_info "No $BINARY_NAME installations found"
        
        # Still offer to clean up config
        if [[ "$keep_config" == false ]]; then
            remove_config
        fi
        
        return 0
    fi
    
    print_info "Found ${#installations[@]} installation(s):"
    for installation in "${installations[@]}"; do
        echo "  - $installation"
    done
    echo
    
    # Confirmation
    if [[ "$force" == false ]]; then
        read -p "Remove all installations? [y/N]: " -n 1 -r
        echo
        
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Uninstallation cancelled"
            exit 0
        fi
    fi
    
    # Remove all installations
    local removed=0
    local failed=0
    
    for installation in "${installations[@]}"; do
        if remove_binary "$installation"; then
            ((removed++))
            
            # Remove from PATH if it was the last binary in that directory
            local install_dir=$(dirname "$installation")
            if [[ ! -f "$install_dir/$BINARY_NAME" ]]; then
                remove_from_path "$install_dir"
            fi
        else
            ((failed++))
        fi
    done
    
    # Summary
    if [[ $removed -gt 0 ]]; then
        print_success "Removed $removed installation(s)"
    fi
    
    if [[ $failed -gt 0 ]]; then
        print_warning "$failed installation(s) could not be removed (permission denied?)"
    fi
    
    # Remove configuration
    if [[ "$keep_config" == false ]]; then
        remove_config
    fi
    
    # Final message
    if [[ $removed -gt 0 ]]; then
        print_success "matlas-cli uninstallation completed!"
        
        # Verify removal
        if command -v "$BINARY_NAME" >/dev/null 2>&1; then
            print_warning "$BINARY_NAME command is still available (may be in a different location)"
            print_info "Run: which $BINARY_NAME"
        else
            print_success "$BINARY_NAME command is no longer available"
        fi
    fi
}

# Run main function with all arguments
main "$@"
