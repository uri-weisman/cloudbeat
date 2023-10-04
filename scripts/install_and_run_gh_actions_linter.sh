#!/bin/bash

VERSION="v0.5.3"
GITHUB_RELEASE_URL="https://github.com/mpalmer/action-validator/releases/download/$VERSION/"

# Detect the OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Set the binary filename based on the detected OS and architecture
if [ "$OS" == "linux" ]; then
    if [ "$ARCH" == "x86_64" ]; then
        BINARY="action-validator_linux_amd64"
    elif [ "$ARCH" == "aarch64" ]; then
        BINARY="action-validator_linux_arm64"
    else
        echo "Unsupported architecture: $ARCH"
        exit 1
    fi
elif [ "$OS" == "darwin" ]; then
    if [ "$ARCH" == "arm64" ]; then
        BINARY="action-validator_darwin_arm64"
    else
        echo "Unsupported architecture: $ARCH"
        exit 1
    fi
else
    echo "Unsupported OS: $OS"
    exit 1
fi

# Download the appropriate binary
curl -o action_validator -fsSL "${GITHUB_RELEASE_URL}${BINARY}"

# Make the binary executable
chmod +x action_validator

# Run action_validator on GitHub Actions workflow files
bash -c './action_validator .github/workflows/*'
