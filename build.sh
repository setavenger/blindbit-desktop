#!/bin/bash

# BlindBit Desktop Build Script

set -e

echo "Building BlindBit Desktop..."

# Clean previous builds
echo "Cleaning previous builds..."
# rm -f blindbit-desktop blindbit-desktop.exe

# Update dependencies
echo "Updating dependencies..."
go mod tidy

# Build for current platform using cmd directory
echo "Building for current platform..."
# go build -tags=libsecp256k1 -o blindbit-desktop ./cmd/blindbit-desktop
go build -o blindbit-desktop ./cmd/blindbit-desktop

echo "Build complete! Run with: ./blindbit-desktop" 
