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
GOOS=windows GOARCH=amd64 go build -o rp-chat-logger.exe
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

gh repo sync --branch main
if [ $? -ne 0 ]; then
    echo "Push failed!"
    exit 1
fi
echo "✓ Tag and commits pushed"
echo ""

# Step 4: Create GitHub release
echo "Step 4: Creating GitHub release..."

# Check if release already exists
if gh release view $TAG >/dev/null 2>&1; then
    echo "Release $TAG already exists on GitHub, skipping release creation"
else
    gh release create $TAG rp-chat-logger.exe \
        --title "Release $VERSION" \
        --notes "Release version $VERSION of RP Chat Logger"

    if [ $? -ne 0 ]; then
        echo "Release creation failed!"
        exit 1
    fi
    echo "✓ Release created on GitHub"
fi
echo ""

echo "=== Release Complete! ==="
echo "Version: $VERSION"
echo "Release URL: https://github.com/ragaz-zo/rp-chat-logger/releases/tag/$TAG"
