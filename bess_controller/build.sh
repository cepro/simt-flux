#!/bin/bash
# Build script for BESS Controller with version information

# Get current timestamp
BUILD_TIME=$(date -u +"%Y-%m-%d_%H:%M:%S_UTC")
BUILD_VERSION="${1:-idle-off-v2}"  # Use first argument or default

echo "Building BESS Controller..."
echo "Version: $BUILD_VERSION"
echo "Build Time: $BUILD_TIME"

cd src

# Build for 64-bit ARM with version info injected
env GOARCH=arm64 GOOS=linux go build \
    -ldflags "-X main.buildVersion=$BUILD_VERSION -X main.buildTime=$BUILD_TIME" \
    -o ../bess_controller_rpi_64 \
    main.go

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Binary: bess_controller_rpi_64"
    ls -lh ../bess_controller_rpi_64
else
    echo "Build failed!"
    exit 1
fi