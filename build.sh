#!/bin/bash

# ANSI color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
CHECK="✓"
WARN="⚠"
INFO="ℹ"

# Function to print status messages
print_status() {
    local color=$1
    local symbol=$2
    local message=$3
    echo -e "${color}${symbol} ${message}${NC}"
}

# Function to check if a command exists
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_status $RED "✗" "Error: $1 is not installed"
        exit 1
    fi
}

# Check prerequisites
print_status $BLUE $INFO "Checking prerequisites..."

check_command "go"
check_command "git"

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
if [[ "${GO_VERSION}" < "1.21" ]]; then
    print_status $RED "✗" "Error: Go version must be 1.21 or higher (current: ${GO_VERSION})"
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

# Create default config directory
CONFIG_DIR="$HOME/.ggquick"
print_status $BLUE $INFO "Setting up configuration directory..."
mkdir -p "$CONFIG_DIR"
if [ $? -ne 0 ]; then
    print_status $RED "✗" "Error: Failed to create config directory"
    exit 1
fi
print_status $GREEN $CHECK "Configuration directory created"

# Download dependencies
print_status $BLUE $INFO "Downloading dependencies..."
go mod download
go mod tidy
print_status $GREEN $CHECK "Dependencies installed"

# Run tests
print_status $BLUE $INFO "Running tests..."
go test -v ./...
if [ $? -ne 0 ]; then
    print_status $RED "✗" "Error: Tests failed"
    exit 1
fi
print_status $GREEN $CHECK "Tests completed"

# Build the binary
print_status $BLUE $INFO "Building ggquick..."
go build -o ggquick
if [ $? -ne 0 ]; then
    print_status $RED "✗" "Error: Build failed"
    exit 1
fi
print_status $GREEN $CHECK "Build successful"

# Install binary
INSTALL_DIR="/usr/local/bin"
if [ -d "$INSTALL_DIR" ]; then
    print_status $BLUE $INFO "Installing ggquick to $INSTALL_DIR..."
    sudo mv ggquick "$INSTALL_DIR/"
    if [ $? -ne 0 ]; then
        print_status $RED "✗" "Error: Failed to install ggquick to $INSTALL_DIR"
        print_status $YELLOW $WARN "You can still use ggquick from the current directory"
    else
        print_status $GREEN $CHECK "Installed ggquick to $INSTALL_DIR"
    fi
else
    print_status $YELLOW $WARN "$INSTALL_DIR not found. Keeping ggquick in current directory"
fi

print_status $GREEN $CHECK "Installation complete!"
print_status $BLUE $INFO "Try running: ggquick --help" 