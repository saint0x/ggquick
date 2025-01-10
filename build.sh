#!/bin/bash

# ANSI color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
CHECK="✓"
INFO="ℹ"
ERROR="✗"

# Function to print status messages
print_status() {
    local color=$1
    local symbol=$2
    local message=$3
    echo -e "${color}${symbol} ${message}${NC}"
}

# Function to quietly remove old build
cleanup_old_build() {
    local binary="/usr/local/bin/ggquick"
    if [ -f "$binary" ]; then
        sudo rm "$binary" 2>/dev/null || true
    fi
}

# Check prerequisites
print_status $BLUE $INFO "Checking prerequisites..."

# Check Go installation
if ! command -v go &> /dev/null; then
    print_status $RED $ERROR "Go is not installed"
    exit 1
fi
print_status $GREEN $CHECK "Go is installed"

# Load environment variables
ENV_FILE=".env"
[ -f ".env.local" ] && ENV_FILE=".env.local"

if [ ! -f "$ENV_FILE" ]; then
    print_status $RED $ERROR "No $ENV_FILE file found"
    exit 1
fi
print_status $GREEN $CHECK "Found $ENV_FILE"

# Source env vars for current session only
while IFS='=' read -r key value; do
    if [[ ! $key =~ ^# && -n $key ]]; then
        key=$(echo "$key" | tr -d '"' | tr -d "'" | tr -d " ")
        value=$(echo "$value" | tr -d '"' | tr -d "'" | tr -d " ")
        export "$key=$value"
    fi
done < "$ENV_FILE"

# Verify required variables
REQUIRED_VARS=("GITHUB_TOKEN" "OPENAI_API_KEY")
for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        print_status $RED $ERROR "$var not found in $ENV_FILE"
        exit 1
    fi
done
print_status $GREEN $CHECK "Required environment variables verified"

# Quietly clean previous build
cleanup_old_build

# Download dependencies
print_status $BLUE $INFO "Downloading dependencies..."
if ! go mod download; then
    print_status $RED $ERROR "Failed to download dependencies"
    exit 1
fi
print_status $GREEN $CHECK "Dependencies downloaded"

# Run installation tests
print_status $BLUE $INFO "Running installation tests..."
if ! go test ./pkg/hooks >/dev/null 2>&1; then
    print_status $RED $ERROR "Installation tests failed"
    exit 1
fi
print_status $GREEN $CHECK "Installation tests passed"

# Build binary
print_status $BLUE $INFO "Building ggquick..."
if ! go build -o ggquick ./cmd; then
    print_status $RED $ERROR "Build failed"
    exit 1
fi
print_status $GREEN $CHECK "Build successful"

# Install binary
print_status $BLUE $INFO "Installing ggquick..."
if ! sudo mv ggquick /usr/local/bin/; then
    print_status $RED $ERROR "Installation failed"
    exit 1
fi
print_status $GREEN $CHECK "Installation successful"

# Verify installation
if [ -f "/usr/local/bin/ggquick" ]; then
    print_status $GREEN $CHECK "Ready to process Git events"
else
    print_status $RED $ERROR "Installation verification failed"
    exit 1
fi 
