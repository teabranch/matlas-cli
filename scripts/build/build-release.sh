#!/bin/bash

# Build script for semantic-release
# Called by semantic-release with the version as argument
set -e

VERSION="$1"

if [ -z "$VERSION" ]; then
    echo "❌ Error: Version parameter is required"
    echo "Usage: $0 <version>"
    exit 1
fi

# Add 'v' prefix if not present
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v${VERSION}"
fi

echo "🏗️ Building matlas-cli binaries for release ${VERSION}"

# Build variables
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILT_BY="semantic-release"

echo "Build variables:"
echo "  VERSION: $VERSION"
echo "  COMMIT: $COMMIT" 
echo "  BUILD_TIME: $BUILD_TIME"
echo "  BUILT_BY: $BUILT_BY"

# Clean and create dist directory
rm -rf dist/
mkdir -p dist/

# Build for all platforms
declare -a platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    echo "🔨 Building for ${GOOS}/${GOARCH}..."
    
    # Set binary name
    BINARY_NAME="matlas"
    if [ "$GOOS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi
    
    # Build binary
    env GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME} -X main.builtBy=${BUILT_BY}" \
        -o "dist/${BINARY_NAME}" \
        .
    
    # Create release archives
    cd dist/
    
    if [ "$GOOS" = "windows" ]; then
        zip "matlas_${GOOS}_${GOARCH}.zip" "${BINARY_NAME}"
    else
        tar -czf "matlas_${GOOS}_${GOARCH}.tar.gz" "${BINARY_NAME}"
        # Also create zip for consistency
        zip "matlas_${GOOS}_${GOARCH}.zip" "${BINARY_NAME}"
    fi
    
    # Remove binary to avoid conflicts with next build
    rm "${BINARY_NAME}"
    
    cd ..
    
    echo "✅ Created archives for ${GOOS}/${GOARCH}"
done

# Generate checksums
echo "🔐 Generating checksums..."
cd dist/
sha256sum *.zip *.tar.gz > checksums.txt || shasum -a 256 *.zip *.tar.gz > checksums.txt

echo ""
echo "📋 Release artifacts created:"
ls -la
echo ""
echo "✅ Build completed successfully for release ${VERSION}"
