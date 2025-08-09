#!/bin/bash
set -euo pipefail

# Script to regenerate all gomock mocks for Atlas services
# Usage: ./scripts/generate-mocks.sh

echo "Regenerating Atlas service mocks..."

# Ensure mockgen is available
if ! command -v mockgen &> /dev/null; then
    echo "Installing mockgen..."
    go install go.uber.org/mock/mockgen@latest
fi

# Create mocks directory if it doesn't exist
mkdir -p internal/atlas/mocks

# Generate mocks using go:generate directives
cd internal/atlas
go generate ./...

echo "Mock generation complete!"
echo "Generated mocks:"
ls -la mocks/ 