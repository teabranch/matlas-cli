#!/bin/bash

# Release Setup Script for matlas-cli
# This script helps set up the release automation

set -e

echo "🚀 Setting up release automation for matlas-cli..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    print_error "This script must be run from the matlas-cli root directory"
    exit 1
fi

print_info "Checking prerequisites..."

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed. Please install Node.js to use semantic-release."
    exit 1
fi
print_status "Node.js is installed ($(node --version))"

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    print_error "npm is not installed. Please install npm."
    exit 1
fi
print_status "npm is installed ($(npm --version))"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go."
    exit 1
fi
print_status "Go is installed ($(go version))"

# Install npm dependencies
print_info "Installing semantic-release dependencies..."
if npm install; then
    print_status "npm dependencies installed successfully"
else
    print_error "Failed to install npm dependencies"
    exit 1
fi



# Test Go build
print_info "Testing Go build..."
if go build -o matlas-test .; then
    print_status "Go build successful"
    ./matlas-test version
    rm matlas-test
else
    print_error "Go build failed"
    exit 1
fi

# Check Git configuration
print_info "Checking Git configuration..."
if git config --get user.name > /dev/null && git config --get user.email > /dev/null; then
    print_status "Git is configured with user.name and user.email"
else
    print_warning "Git user.name or user.email not configured. This may cause issues with semantic-release."
    print_info "Configure with:"
    echo "  git config --global user.name 'Your Name'"
    echo "  git config --global user.email 'your.email@example.com'"
fi

# Check if we're in a Git repository
if git rev-parse --git-dir > /dev/null 2>&1; then
    print_status "Running in a Git repository"
else
    print_error "Not in a Git repository"
    exit 1
fi

echo ""
print_info "Setup complete! Next steps:"
echo ""
echo "1. 🔐 Configure GitHub repository secrets (optional):"
echo "   - SEMANTIC_RELEASE_TOKEN (optional, for triggering release workflow)"
echo ""
echo "2. 🧪 Test the setup:"
echo "   - Make a commit with conventional commit format:"
echo "     git commit -m 'feat: initial release setup'"
echo "   - Push to main branch:"
echo "     git push origin main"
echo ""
echo "3. 📖 Read the documentation:"
echo "   - See RELEASE_SETUP.md for detailed instructions"
echo ""
print_status "Release automation is ready to use!"

# Offer to run semantic-release in dry-run mode
echo ""
read -p "Would you like to test semantic-release in dry-run mode now? (y/n): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "Running semantic-release in dry-run mode..."
    if npx semantic-release --dry-run; then
        print_status "Semantic-release dry-run completed successfully"
    else
        print_warning "Semantic-release dry-run had issues. Check the output above."
    fi
fi

echo ""
print_status "Setup script completed!"
