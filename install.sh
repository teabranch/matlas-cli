#!/usr/bin/env bash

# matlas-cli installation script
# Supports macOS and Linux with automatic platform detection

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Configuration
readonly REPO_OWNER="teabranch"
readonly REPO_NAME="matlas-cli"
readonly BINARY_NAME="matlas"
readonly INSTALL_DIR="${MATLAS_INSTALL_DIR:-/usr/local/bin}"

# Helper functions
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ $1${NC}"; }

# Detect platform and architecture
detect_platform() {
    local os
    local arch
    
    # Detect OS
    case "$(uname -s)" in
        Darwin*)    os="darwin" ;;
        Linux*)     os="linux" ;;
        *)          print_error "Unsupported operating system: $(uname -s)"
                    print_info "This script supports macOS and Linux only."
                    print_info "For Windows, please use install.ps1"
                    exit 1 ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        *)              print_error "Unsupported architecture: $(uname -m)"
                        exit 1 ;;
    esac
    
    echo "${os}-${arch}"
}

# Check if running as root (for install dir permissions)
check_permissions() {
    if [[ ! -w "$INSTALL_DIR" ]]; then
        if [[ $EUID -ne 0 ]]; then
            print_warning "Installation directory '$INSTALL_DIR' requires root access"
            print_info "You may need to run: sudo $0"
            print_info "Or set MATLAS_INSTALL_DIR to a user-writable directory"
            return 1
        fi
    fi
    return 0
}

# Create install directory if it doesn't exist
prepare_install_dir() {
    if [[ ! -d "$INSTALL_DIR" ]]; then
        print_info "Creating install directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi
}

# Get the latest release version from GitHub
get_latest_version() {
    print_info "Fetching latest release information..."
    
    if command -v curl >/dev/null 2>&1; then
        curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"([^"]+)".*/\1/'
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
}

# Download and extract the binary
download_binary() {
    local version="$1"
    local platform="$2"
    local temp_dir
    
    temp_dir=$(mktemp -d)
    trap 'rm -rf "$temp_dir"' EXIT
    
    # Remove 'v' prefix if present
    version="${version#v}"
    
    local archive_name="${BINARY_NAME}-${platform}.tar.gz"
    local download_url="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/v${version}/${archive_name}"
    
    print_info "Downloading $archive_name..."
    
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fL "$download_url" -o "$temp_dir/$archive_name"; then
            print_error "Failed to download from $download_url"
            return 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$download_url" -O "$temp_dir/$archive_name"; then
            print_error "Failed to download from $download_url"
            return 1
        fi
    fi
    
    print_info "Extracting binary..."
    tar -xzf "$temp_dir/$archive_name" -C "$temp_dir"
    
    # Find the binary (might be in a subdirectory or with different name)
    local binary_path
    if [[ -f "$temp_dir/$BINARY_NAME" ]]; then
        binary_path="$temp_dir/$BINARY_NAME"
    elif [[ -f "$temp_dir/${BINARY_NAME}-${platform}" ]]; then
        binary_path="$temp_dir/${BINARY_NAME}-${platform}"
    else
        # Try to find any executable file
        binary_path=$(find "$temp_dir" -type f -executable | head -1)
        if [[ -z "$binary_path" ]]; then
            print_error "Could not find binary in downloaded archive"
            return 1
        fi
    fi
    
    # Install the binary
    print_info "Installing to $INSTALL_DIR/$BINARY_NAME..."
    cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    return 0
}

# Check if install directory is in PATH
check_path() {
    if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
        return 0
    else
        return 1
    fi
}

# Add install directory to PATH
setup_path() {
    local shell_config
    local shell_name
    
    # Detect user's shell
    shell_name=$(basename "$SHELL")
    
    case "$shell_name" in
        bash)
            if [[ "$(uname -s)" == "Darwin" ]]; then
                shell_config="$HOME/.bash_profile"
            else
                shell_config="$HOME/.bashrc"
            fi
            ;;
        zsh)
            shell_config="$HOME/.zshrc"
            ;;
        fish)
            shell_config="$HOME/.config/fish/config.fish"
            ;;
        *)
            print_warning "Unknown shell: $shell_name"
            print_info "Please manually add $INSTALL_DIR to your PATH"
            return 0
            ;;
    esac
    
    # Check if already in config
    if [[ -f "$shell_config" ]] && grep -q "$INSTALL_DIR" "$shell_config"; then
        print_info "PATH already configured in $shell_config"
        return 0
    fi
    
    print_info "Adding $INSTALL_DIR to PATH in $shell_config"
    
    # Create config file if it doesn't exist
    mkdir -p "$(dirname "$shell_config")"
    touch "$shell_config"
    
    # Add PATH export
    case "$shell_name" in
        fish)
            echo "set -gx PATH $INSTALL_DIR \$PATH" >> "$shell_config"
            ;;
        *)
            echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$shell_config"
            ;;
    esac
    
    print_success "Added $INSTALL_DIR to PATH in $shell_config"
    print_warning "Please restart your shell or run: source $shell_config"
}

# Verify installation
verify_installation() {
    if [[ -x "$INSTALL_DIR/$BINARY_NAME" ]]; then
        print_success "Installation successful!"
        
        # Try to get version
        if "$INSTALL_DIR/$BINARY_NAME" version >/dev/null 2>&1; then
            local version_info
            version_info=$("$INSTALL_DIR/$BINARY_NAME" version 2>/dev/null | head -1)
            print_info "Installed version: $version_info"
        fi
        
        print_info "Binary location: $INSTALL_DIR/$BINARY_NAME"
        
        if check_path; then
            print_success "Installation directory is in PATH"
            print_info "You can now run: $BINARY_NAME --help"
        else
            print_warning "Installation directory is not in PATH"
            print_info "Run the following to add it to your PATH:"
            print_info "  export PATH=\"$INSTALL_DIR:\$PATH\""
        fi
        
        return 0
    else
        print_error "Installation failed - binary not found"
        return 1
    fi
}

# Show usage information
show_usage() {
    cat << EOF
matlas-cli Installation Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -v, --version VERSION    Install specific version (default: latest)
    -d, --dir DIRECTORY      Install directory (default: /usr/local/bin)
    --no-path-setup          Skip automatic PATH setup
    -h, --help              Show this help

ENVIRONMENT VARIABLES:
    MATLAS_INSTALL_DIR      Custom installation directory

EXAMPLES:
    $0                      # Install latest version
    $0 -v v1.2.3           # Install specific version
    $0 -d ~/.local/bin     # Install to custom directory
    
    # Install to user directory without sudo
    MATLAS_INSTALL_DIR=~/.local/bin $0

EOF
}

# Main installation function
main() {
    local version=""
    local setup_path_flag=true
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                version="$2"
                shift 2
                ;;
            -d|--dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --no-path-setup)
                setup_path_flag=false
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
    
    print_info "matlas-cli Installation Script"
    print_info "==============================="
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    print_info "Detected platform: $platform"
    
    # Check permissions
    if ! check_permissions; then
        exit 1
    fi
    
    # Prepare install directory
    prepare_install_dir
    
    # Get version to install
    if [[ -z "$version" ]]; then
        version=$(get_latest_version)
        if [[ -z "$version" ]]; then
            print_error "Failed to fetch latest version"
            exit 1
        fi
        print_info "Latest version: $version"
    else
        print_info "Installing version: $version"
    fi
    
    # Download and install
    if ! download_binary "$version" "$platform"; then
        print_error "Installation failed"
        exit 1
    fi
    
    # Setup PATH if requested
    if [[ "$setup_path_flag" == true ]] && ! check_path; then
        setup_path
    fi
    
    # Verify installation
    if verify_installation; then
        print_success "matlas-cli installed successfully!"
        
        # Show next steps
        echo
        print_info "Next steps:"
        print_info "1. Restart your shell or run: source ~/.bashrc (or ~/.zshrc)"
        print_info "2. Run: $BINARY_NAME --help"
        print_info "3. Configure authentication: $BINARY_NAME config --help"
        
    else
        exit 1
    fi
}

# Run main function with all arguments
main "$@"
