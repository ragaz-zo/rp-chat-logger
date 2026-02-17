#!/bin/bash

# Release script for RP Chat Logger
# Usage: ./release.sh 1.0.0

if [ -z "$1" ]; then
    echo "Usage: ./release.sh <version>"
    echo "Example: ./release.sh 1.0.0"
    exit 1
fi

VERSION=$1
TAG="v$VERSION"

echo "=== RP Chat Logger Release Script ==="
echo "Version: $VERSION"
echo "Tag: $TAG"
echo ""

# Step 1: Build for Windows
echo "Step 1: Building for Windows..."

# Try to embed manifest if rsrc is available (reduces antivirus false positives)
if command -v rsrc &> /dev/null; then
    echo "  - Embedding Windows manifest..."
    rsrc -manifest rp-chat-logger.manifest -o rsrc.syso
    if [ $? -eq 0 ]; then
        GOOS=windows GOARCH=amd64 go build -o rp-chat-logger.exe
        rm -f rsrc.syso
    else
        echo "  - Manifest embedding failed, building without it..."
        GOOS=windows GOARCH=amd64 go build -o rp-chat-logger.exe
    fi
else
    GOOS=windows GOARCH=amd64 go build -o rp-chat-logger.exe
fi

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi
echo "✓ Build successful: rp-chat-logger.exe"
echo ""

# Step 2: Commit changes
echo "Step 2: Committing changes..."
git add .
git commit -m "Release version $VERSION"
if [ $? -ne 0 ]; then
    echo "Commit failed! (Note: This is normal if there are no changes)"
fi
echo ""

# Step 3: Create and push tag
echo "Step 3: Creating tag and pushing to GitHub..."

# Check if tag already exists
if git rev-parse $TAG >/dev/null 2>&1; then
    # Tag exists, check if it points to current HEAD
    TAG_COMMIT=$(git rev-list -n 1 $TAG)
    CURRENT_COMMIT=$(git rev-parse HEAD)

    if [ "$TAG_COMMIT" = "$CURRENT_COMMIT" ]; then
        echo "Tag $TAG already exists on current commit, skipping tag creation"
    else
        echo "Tag $TAG already exists but points to a different commit!"
        exit 1
    fi
else
    # Tag doesn't exist, create it
    git tag -a $TAG -m "Release version $VERSION"
    if [ $? -ne 0 ]; then
        echo "Tag creation failed!"
        exit 1
    fi
fi

git push
if [ $? -ne 0 ]; then
    echo "Push failed!"
    exit 1
fi

git push --tags
if [ $? -ne 0 ]; then
    echo "Push tags failed!"
    exit 1
fi
echo "✓ Commits and tags pushed"
echo ""

echo "=== Release Complete! ==="
echo "Version: $VERSION"
echo ""
echo "Next step: Create the GitHub release manually at:"
echo "https://github.com/ragaz-zo/rp-chat-logger/releases/new?tag=$TAG"
echo ""
echo "Or run this command once gh CLI is properly authenticated:"
echo "gh release create $TAG rp-chat-logger.exe --title \"Release $VERSION\" --notes \"Release version $VERSION of RP Chat Logger\""
