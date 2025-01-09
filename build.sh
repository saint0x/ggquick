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

# Determine shell profile
case "$SHELL" in
  */zsh) 
    SHELL_PROFILE="$HOME/.zshenv"
    ;;
  */bash)
    SHELL_PROFILE="$HOME/.bash_profile"
    [ -f "$HOME/.bash_profile" ] || SHELL_PROFILE="$HOME/.profile"
    ;;
  *)
    SHELL_PROFILE="$HOME/.profile"
    ;;
esac

# Load environment variables
ENV_FILE=".env"
[ -f ".env.local" ] && ENV_FILE=".env.local"

if [ ! -f "$ENV_FILE" ]; then
    print_status $RED "✗" "Error: No $ENV_FILE file found"
    exit 1
fi

# Source and persist env vars
while IFS='=' read -r key value; do
    if [[ ! $key =~ ^# && -n $key ]]; then
        key=$(echo "$key" | tr -d '"' | tr -d "'" | tr -d " ")
        value=$(echo "$value" | tr -d '"' | tr -d "'" | tr -d " ")
        
        # Export to current shell
        export "$key=$value"
        
        # Update or add to shell profile
        if grep -q "^export $key=" "$SHELL_PROFILE" 2>/dev/null; then
            sed -i.bak "s|^export $key=.*|export $key=$value|" "$SHELL_PROFILE"
            rm -f "$SHELL_PROFILE.bak"
        else
            echo "export $key=$value" >> "$SHELL_PROFILE"
        fi
    fi
done < "$ENV_FILE"

# Verify required variables
if [ -z "$GITHUB_TOKEN" ]; then
    print_status $RED "✗" "Error: GITHUB_TOKEN not found in $ENV_FILE"
    exit 1
fi

if [ -z "$GITHUB_USERNAME" ]; then
    print_status $RED "✗" "Error: GITHUB_USERNAME not found in $ENV_FILE"
    exit 1
fi

if [ -z "$OPENAI_API_KEY" ]; then
    print_status $RED "✗" "Error: OPENAI_API_KEY not found in $ENV_FILE"
    exit 1
fi

print_status $GREEN $CHECK "Prerequisites satisfied"

# Build and install
print_status $BLUE $INFO "Building ggquick..."
INSTALL_DIR="/usr/local/bin"
[ -d "$HOME/.local/bin" ] && INSTALL_DIR="$HOME/.local/bin"

if go build -o /tmp/ggquick ./cmd; then
    print_status $GREEN $CHECK "Build successful"
else
    print_status $RED "✗" "Build failed"
    rm -f /tmp/ggquick
    exit 1
fi

print_status $BLUE $INFO "Installing to $INSTALL_DIR..."
if [ "$INSTALL_DIR" = "/usr/local/bin" ]; then
    if sudo mv /tmp/ggquick "$INSTALL_DIR/"; then
        print_status $GREEN $CHECK "Installation complete"
        print_status $BLUE $INFO "Try running: ggquick --help"
    else
        print_status $RED "✗" "Installation failed"
        rm -f /tmp/ggquick
        exit 1
    fi
else
    mkdir -p "$INSTALL_DIR"
    if mv /tmp/ggquick "$INSTALL_DIR/"; then
        print_status $GREEN $CHECK "Installation complete"
        print_status $BLUE $INFO "Try running: ggquick --help"
    else
        print_status $RED "✗" "Installation failed"
        rm -f /tmp/ggquick
        exit 1
    fi
fi 