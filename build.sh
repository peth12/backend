#!/bin/bash

# Build script for SpendWise Pro Backend

set -e  # Exit on error

echo "============================"
echo "SpendWise Pro - Build Script"
echo "============================"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

echo "Go version: $(go version)"
echo ""

# Clean previous builds
echo "Cleaning previous builds..."
rm -rf bin/
mkdir -p bin
echo ""

# Install dependencies
echo "Installing dependencies..."
go mod download
go mod tidy
echo ""

# Build the application
echo "Building application..."
CGO_ENABLED=1 go build -ldflags="-w -s" -o bin/server cmd/api/main.go
echo ""

# Check if build was successful
if [ -f "bin/server" ]; then
    echo "✓ Build successful!"
    echo "Binary created: bin/server"
    
    # Make it executable
    chmod +x bin/server
    
    # Show file info
    ls -lh bin/server
else
    echo "✗ Build failed!"
    exit 1
fi

echo ""
echo "To run the server:"
echo "  ./bin/server"
