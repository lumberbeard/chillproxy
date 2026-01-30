#!/bin/bash
# Quick build and push script for chillproxy

set -e

echo "🔨 Building and pushing Chillproxy Docker image..."
echo ""

# Stop on errors
trap 'echo "❌ Build failed"; exit 1' ERR

# Get current directory (works on both Windows and Linux)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Pull latest code if in git repo
if [ -d ".git" ]; then
    echo "📥 Pulling latest code..."
    git fetch origin main
    echo "✅ Latest commit: $(git rev-parse --short HEAD)"
else
    echo "⚠️  Not a git repository"
fi

COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
echo "📦 Building commit: $COMMIT"
echo ""

# Build dependencies first
echo "🔨 Building dashboard..."
pnpm run dash:build

if [ $? -ne 0 ]; then
    echo "❌ Dashboard build failed"
    exit 1
fi
echo ""

# Build image
echo "🔨 Building Docker image..."
docker build -t ghcr.io/lumberbeard/chillproxy:latest -t ghcr.io/lumberbeard/chillproxy:$COMMIT .

if [ $? -eq 0 ]; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi
echo ""

# Login to registry
echo "🔐 Logging into GitHub Container Registry..."
if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ GITHUB_TOKEN not set. Please export it:"
    echo "export GITHUB_TOKEN='ghp_xxxxx'"
    exit 1
fi

echo $GITHUB_TOKEN | docker login ghcr.io -u lumberbeard --password-stdin

if [ $? -eq 0 ]; then
    echo "✅ Logged in"
else
    echo "❌ Login failed"
    exit 1
fi
echo ""

# Push images
echo "📤 Pushing images to registry..."
docker push ghcr.io/lumberbeard/chillproxy:latest
docker push ghcr.io/lumberbeard/chillproxy:$COMMIT

echo ""
echo "✅ Push successful!"
echo ""
echo "📊 Image pushed:"
echo "  ghcr.io/lumberbeard/chillproxy:latest"
echo "  ghcr.io/lumberbeard/chillproxy:$COMMIT"
echo ""
