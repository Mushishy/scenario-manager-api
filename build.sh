#!/bin/bash

set -e

echo "Building scenario-manager-api for production..."
cd server

# Clean any previous builds
rm -f scenario-manager-api

# Build
GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o scenario-manager-api .
#GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o scenario-manager-api .

echo "Build complete!"
echo "File size: $(du -h scenario-manager-api | cut -f1)"

chmod +x "scenario-manager-api"