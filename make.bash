#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

cmd="${1:-}"

build() {
    echo "Building slider..."

    # Regenerate embedded filesystem
    statik -src=example -dest=src -f

    # Build binary
    cd src
    go build -o ../slider
    cd ..

    echo "✓ Build complete: slider"
}

build_release() {
    echo "Building for release..."

    # Regenerate embedded filesystem
    statik -src=example -dest=src -f

    # Create .local directory
    mkdir -p .local

    # Build macOS/Linux binary
    echo "Building macOS/Linux binary..."
    cd src
    go build -o ../slider
    cd ..
    cp slider .local/slider
    chmod +x .local/slider
    echo "✓ macOS/Linux: .local/slider"

    # Build Windows binary
    echo "Building Windows binary..."
    cd src
    GOOS=windows GOARCH=amd64 go build -o ../.local/slider.exe
    cd ..
    echo "✓ Windows: .local/slider.exe"

    echo ""
    echo "✓ Release complete"
}

case "$cmd" in
    build)
        build
        ;;
    build-release|release)
        build_release
        ;;
    *)
        echo "Usage: $0 {build|build-release}"
        echo ""
        echo "Commands:"
        echo "  build         - Build slider binary"
        echo "  build-release - Build release binaries in .local/"
        exit 1
        ;;
esac
