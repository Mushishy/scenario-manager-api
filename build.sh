#!/bin/bash

# Build for Linux AMD64
echo "Building scenario-manager-api for Linux AMD64..."

cd server

# Clean previous builds
rm -f scenario-manager-api

# Check if we're cross-compiling or building natively
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Building natively on Linux..."
    # Native build on Linux
    export CGO_ENABLED=1
    go build -ldflags="-w -s" -o scenario-manager-api .
else
    echo "Cross-compiling from $OSTYPE to Linux AMD64..."
    # Cross-compilation from macOS/Windows to Linux
    # For SQLite cross-compilation, we need to disable CGO or use a different approach
    export GOOS=linux
    export GOARCH=amd64
    export CGO_ENABLED=0

    go build -ldflags="-w -s" -o scenario-manager-api .
fi

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created: server/scenario-manager-api"
    echo "File size: $(ls -lh scenario-manager-api | awk '{print $5}')"
    
    # Show binary info
    if command -v file >/dev/null 2>&1; then
        echo "Binary info: $(file scenario-manager-api)"
    fi
else
    echo "Build failed!"
    exit 1
fi