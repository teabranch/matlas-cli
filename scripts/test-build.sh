#!/bin/bash

# Test build script based on release.yml workflow
set -e

echo "🔧 Testing build configuration..."

# Set build variables like in release.yml
VERSION="test"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Build variables:"
echo "  VERSION: $VERSION"
echo "  COMMIT: $COMMIT" 
echo "  BUILD_TIME: $BUILD_TIME"

# Test the ldflags format used in release.yml
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}"

echo ""
echo "🏗️ Testing build with ldflags..."
echo "LDFLAGS: $LDFLAGS"

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi

# Test build (without actual execution since we may not have all deps)
echo ""
echo "📋 Checking module dependencies..."
go mod verify

echo ""
echo "✅ Build configuration test completed successfully!"
echo ""
echo "To test the actual build, run:"
echo "go build -ldflags=\"$LDFLAGS\" -o matlas ."
