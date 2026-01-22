#!/bin/bash

# =============================================================================
# VendorPlatform - GitHub Push Script
# Run this script with Claude Code to push to GitHub
# =============================================================================

set -e  # Exit on error

# Configuration
REPO_NAME="vendorplatform"
ORG_NAME="BillyRonksGlobal"
REPO_VISIBILITY="private"  # Change to "public" if needed
BRANCH_NAME="main"

echo "🚀 VendorPlatform GitHub Push Script"
echo "======================================"

# Check if we're in the right directory
if [ ! -f "README.md" ] || [ ! -f "go.mod" ]; then
    echo "❌ Error: Please run this script from the vendorplatform directory"
    exit 1
fi

# Check if git is initialized
if [ ! -d ".git" ]; then
    echo "📁 Initializing git repository..."
    git init
else
    echo "✅ Git already initialized"
fi

# Configure git user if not set
if [ -z "$(git config user.email)" ]; then
    echo "📧 Please configure git user:"
    read -p "Enter your email: " git_email
    read -p "Enter your name: " git_name
    git config user.email "$git_email"
    git config user.name "$git_name"
fi

# Add all files
echo "📦 Adding files to git..."
git add .

# Show what will be committed
echo ""
echo "📋 Files to be committed:"
git status --short

# Create commit
echo ""
echo "💾 Creating commit..."
COMMIT_MSG="feat: Initial commit - VendorPlatform contextual commerce orchestration

Platform Components:
- LifeOS: Intelligent life event orchestration
- EventGPT: Conversational AI event planner  
- VendorNet: B2B partnership network
- HomeRescue: Emergency home services

Infrastructure:
- PostgreSQL schema with TimescaleDB, PostGIS, LTREE
- Go recommendation engine with collaborative filtering
- Python ML service for predictions
- Complete API server with health checks

Documentation:
- Platform specifications (~6,000+ lines of Go)
- Business model canvases
- Service cluster deep dives
- Strategy documentation"

git commit -m "$COMMIT_MSG" || echo "⚠️ Nothing to commit or already committed"

# Check if remote exists
if git remote | grep -q "origin"; then
    echo "✅ Remote 'origin' already configured"
    REMOTE_URL=$(git remote get-url origin)
    echo "   URL: $REMOTE_URL"
else
    echo ""
    echo "🔗 Setting up GitHub remote..."
    
    # Try to use GitHub CLI
    if command -v gh &> /dev/null; then
        echo "   Using GitHub CLI..."
        
        # Check if logged in
        if gh auth status &> /dev/null; then
            echo "   ✅ GitHub CLI authenticated"
            
            # Check if repo exists
            if gh repo view "$ORG_NAME/$REPO_NAME" &> /dev/null; then
                echo "   📦 Repository exists, adding remote..."
                git remote add origin "https://github.com/$ORG_NAME/$REPO_NAME.git"
            else
                echo "   📦 Creating new repository..."
                gh repo create "$ORG_NAME/$REPO_NAME" --"$REPO_VISIBILITY" --description "Contextual Commerce Orchestration Platform" -y
            fi
        else
            echo "   ⚠️ GitHub CLI not authenticated. Run: gh auth login"
            echo ""
            echo "   Or manually add remote:"
            echo "   git remote add origin https://github.com/$ORG_NAME/$REPO_NAME.git"
            exit 1
        fi
    else
        echo "   ⚠️ GitHub CLI not found"
        echo ""
        echo "   Please manually:"
        echo "   1. Create repository at: https://github.com/new"
        echo "   2. Run: git remote add origin https://github.com/$ORG_NAME/$REPO_NAME.git"
        exit 1
    fi
fi

# Set branch name
echo ""
echo "🌿 Setting branch to '$BRANCH_NAME'..."
git branch -M "$BRANCH_NAME"

# Push to GitHub
echo ""
echo "⬆️ Pushing to GitHub..."
git push -u origin "$BRANCH_NAME"

# Success!
echo ""
echo "======================================"
echo "✅ Successfully pushed to GitHub!"
echo ""
echo "🔗 Repository URL:"
echo "   https://github.com/$ORG_NAME/$REPO_NAME"
echo ""
echo "📋 Next steps:"
echo "   1. Add collaborators if needed"
echo "   2. Set up branch protection rules"
echo "   3. Configure GitHub Actions for CI/CD"
echo "   4. Add secrets for deployments"
echo "======================================"
