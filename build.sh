#!/bin/bash

# ANSI color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
CHECK="✓"
INFO="ℹ"

# Function to print status messages
print_status() {
    local color=$1
    local symbol=$2
    local message=$3
    echo -e "${color}${symbol} ${message}${NC}"
}

# Check prerequisites
print_status $BLUE $INFO "Checking prerequisites..."

# Check Go installation
if ! command -v go &> /dev/null; then
    print_status $RED "✗" "Error: Go is not installed"
    exit 1
fi

# Check GITHUB_TOKEN
if [ -z "$GITHUB_TOKEN" ]; then
    print_status $RED "✗" "Error: GITHUB_TOKEN environment variable is not set"
    print_status $BLUE $INFO "Please set GITHUB_TOKEN environment variable:"
    print_status $BLUE $INFO "export GITHUB_TOKEN=your_github_token"
    exit 1
fi

print_status $GREEN $CHECK "Prerequisites satisfied"

# Build and install
print_status $BLUE $INFO "Building ggquick..."
if go build -o ggquick ./cmd; then
    print_status $GREEN $CHECK "Build successful"
else
    print_status $RED "✗" "Build failed"
    exit 1
fi

print_status $BLUE $INFO "Installing to /usr/local/bin..."
if sudo mv ggquick /usr/local/bin/; then
    print_status $GREEN $CHECK "Installation complete"
    print_status $BLUE $INFO "Try running: ggquick --help"
else
    print_status $RED "✗" "Installation failed"
    exit 1
fi 