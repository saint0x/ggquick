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

# Load environment variables
ENV_FILE=".env"
[ -f ".env.local" ] && ENV_FILE=".env.local"

if [ ! -f "$ENV_FILE" ]; then
    print_status $RED "✗" "Error: No $ENV_FILE file found"
    exit 1
fi

# Source env vars for current session only
while IFS='=' read -r key value; do
    if [[ ! $key =~ ^# && -n $key ]]; then
        key=$(echo "$key" | tr -d '"' | tr -d "'" | tr -d " ")
        value=$(echo "$value" | tr -d '"' | tr -d "'" | tr -d " ")
        export "$key=$value"
    fi
done < "$ENV_FILE"

# Verify required variables
if [ -z "$GITHUB_TOKEN" ]; then
    print_status $RED "✗" "Error: GITHUB_TOKEN not found in $ENV_FILE"
    exit 1
fi

if [ -z "$OPENAI_API_KEY" ]; then
    print_status $RED "✗" "Error: OPENAI_API_KEY not found in $ENV_FILE"
    exit 1
fi

print_status $GREEN $CHECK "Prerequisites satisfied"

# Build and install
print_status $BLUE $INFO "Building ggquick..."

if go build -o /tmp/ggquick ./cmd; then
    print_status $GREEN $CHECK "Build successful"
else
    print_status $RED "✗" "Build failed"
    rm -f /tmp/ggquick
    exit 1
fi

print_status $BLUE $INFO "Installing to /usr/local/bin..."
if sudo mv /tmp/ggquick "/usr/local/bin/"; then
    print_status $GREEN $CHECK "Installation complete"
    print_status $BLUE $INFO "Try running: ggquick --help"
else
    print_status $RED "✗" "Installation failed"
    rm -f /tmp/ggquick
    exit 1
fi 
