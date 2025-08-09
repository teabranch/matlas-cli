#!/usr/bin/env bash

# Setup Development Environment
# Installs dependencies and configures the project

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

check_go() {
    print_info "Checking Go installation..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        print_info "Install Go from: https://golang.org/dl/"
        return 1
    fi
    
    local go_version=$(go version | awk '{print $3}')
    print_success "Go installed: $go_version"
    return 0
}

setup_env_file() {
    print_info "Setting up environment file..."
    
    local env_file="$PROJECT_ROOT/.env"
    
    if [[ -f "$env_file" ]]; then
        print_success ".env file already exists"
        return 0
    fi
    
    cat > "$env_file" << 'EOF'
# Atlas Credentials
# Get these from: https://cloud.mongodb.com/v2#/account/publicApiAccess
export ATLAS_PUB_KEY=your-atlas-public-key
export ATLAS_API_KEY=your-atlas-private-key
export ATLAS_PROJECT_ID=your-atlas-project-id
export ATLAS_ORG_ID=your-atlas-org-id
EOF
    
    print_success "Created .env file template"
    print_warning "Edit .env file with your Atlas credentials"
    return 0
}

install_dependencies() {
    print_info "Installing Go dependencies..."
    
    cd "$PROJECT_ROOT"
    
    if go mod download; then
        print_success "Dependencies installed"
    else
        print_error "Failed to install dependencies"
        return 1
    fi
    
    # Install dev tools if needed
    if ! command -v golangci-lint &> /dev/null; then
        print_info "Installing golangci-lint..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
        print_success "golangci-lint installed"
    fi
    
    return 0
}

build_project() {
    print_info "Building project..."
    
    cd "$PROJECT_ROOT"
    
    if go build -o matlas; then
        print_success "Project built successfully"
        print_info "Binary: ./matlas"
    else
        print_error "Build failed"
        return 1
    fi
    
    return 0
}

setup_git_hooks() {
    print_info "Setting up git hooks..."
    
    if [[ ! -d "$PROJECT_ROOT/.git" ]]; then
        print_warning "Not a git repository - skipping git hooks"
        return 0
    fi
    
    # Copy pre-commit hook if it exists in scripts-backup
    local pre_commit_source="$PROJECT_ROOT/scripts-backup/pre-commit"
    local pre_commit_dest="$PROJECT_ROOT/.git/hooks/pre-commit"
    
    if [[ -f "$pre_commit_source" ]]; then
        cp "$pre_commit_source" "$pre_commit_dest"
        chmod +x "$pre_commit_dest"
        print_success "Git pre-commit hook installed"
    else
        print_info "No pre-commit hook found to install"
    fi
    
    return 0
}

verify_setup() {
    print_info "Verifying setup..."
    
    local errors=0
    
    # Check if binary works
    if [[ -f "$PROJECT_ROOT/matlas" ]]; then
        if "$PROJECT_ROOT/matlas" version &> /dev/null; then
            print_success "matlas binary is functional"
        else
            print_error "matlas binary is not functional"
            ((errors++))
        fi
    else
        print_error "matlas binary not found"
        ((errors++))
    fi
    
    # Check environment file
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        if grep -q "your-atlas" "$PROJECT_ROOT/.env"; then
            print_warning ".env file needs to be configured with your Atlas credentials"
        else
            print_success ".env file appears to be configured"
        fi
    else
        print_error ".env file not found"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        print_success "Setup verification passed"
        return 0
    else
        print_error "$errors setup issue(s) found"
        return 1
    fi
}

show_next_steps() {
    print_info "Setup complete! Next steps:"
    echo
    echo "1. Configure your Atlas credentials in .env:"
    echo "   - Get credentials from: https://cloud.mongodb.com/v2#/account/publicApiAccess"
    echo "   - Edit .env file with your actual keys"
    echo
    echo "2. Run tests:"
    echo "   ./scripts/test.sh unit          # Run unit tests"
    echo "   ./scripts/test.sh integration   # Run integration tests (requires .env)"
    echo "   ./scripts/test.sh all           # Run all tests"
    echo
    echo "3. Use the CLI:"
    echo "   ./matlas --help                 # Show help"
    echo "   ./matlas atlas projects list    # List Atlas projects"
    echo
}

main() {
    print_info "Setting up matlas-cli development environment..."
    echo
    
    local setup_errors=0
    
    check_go || ((setup_errors++))
    setup_env_file || ((setup_errors++))
    install_dependencies || ((setup_errors++))
    build_project || ((setup_errors++))
    setup_git_hooks || true  # Don't fail if this doesn't work
    
    echo
    verify_setup || ((setup_errors++))
    
    echo
    if [[ $setup_errors -eq 0 ]]; then
        print_success "Development environment setup complete!"
        show_next_steps
    else
        print_error "Setup completed with $setup_errors error(s)"
        print_info "Please fix the issues above and run setup again"
        return 1
    fi
}

main "$@"