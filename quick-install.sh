#!/usr/bin/env bash

# Quick installation script for matlas-cli
# Can be run with: curl -fsSL https://raw.githubusercontent.com/teabranch/matlas-cli/main/quick-install.sh | bash

set -euo pipefail

# Check if we have curl or wget
if command -v curl >/dev/null 2>&1; then
    DOWNLOADER="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER="wget -qO-"
else
    echo "Error: Neither curl nor wget found. Please install one of them." >&2
    exit 1
fi

# Download and run the full installation script
echo "Downloading matlas-cli installation script..."
$DOWNLOADER "https://raw.githubusercontent.com/teabranch/matlas-cli/main/install.sh" | bash "$@"
