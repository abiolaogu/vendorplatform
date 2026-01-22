# Claude Code: GitHub Push Instructions

## Overview

This document contains instructions for pushing the VendorPlatform codebase to GitHub using Claude Code.

## Prerequisites

Before pushing, ensure:
1. You have a GitHub account with repository creation permissions
2. GitHub CLI (`gh`) or Git is configured with your credentials
3. You have the repository URL (or will create a new one)

## Option 1: Create New Repository and Push

### Step 1: Initialize Git Repository

```bash
cd /path/to/vendorplatform

# Initialize git
git init

# Add all files
git add .

# Create initial commit
git commit -m "feat: Initial commit - VendorPlatform contextual commerce orchestration

- Core database schema (PostgreSQL with TimescaleDB, PostGIS)
- Recommendation engine (Go)
- LifeOS: Intelligent life event orchestration platform
- EventGPT: Conversational AI event planner
- VendorNet: B2B partnership network
- HomeRescue: Emergency home services platform
- ML service for recommendations (Python)
- Comprehensive documentation"
```

### Step 2: Create GitHub Repository

```bash
# Using GitHub CLI (recommended)
gh repo create BillyRonksGlobal/vendorplatform --private --description "Contextual Commerce Orchestration Platform"

# Or create manually on GitHub, then add remote:
git remote add origin https://github.com/BillyRonksGlobal/vendorplatform.git
```

### Step 3: Push to GitHub

```bash
# Push to main branch
git branch -M main
git push -u origin main
```

## Option 2: Push to Existing Repository

```bash
cd /path/to/vendorplatform

# Add remote if not already added
git remote add origin https://github.com/BillyRonksGlobal/vendorplatform.git

# Fetch existing content
git fetch origin

# If you need to merge with existing content:
git pull origin main --allow-unrelated-histories

# Add all changes
git add .

# Commit
git commit -m "feat: Add platform specifications and core infrastructure"

# Push
git push origin main
```

## File Structure to Push

```
vendorplatform/
├── .gitignore
├── README.md
├── Makefile
├── go.mod
├── requirements.txt
├── cmd/
│   └── server/
│       └── main.go
├── api/
│   ├── lifeos/
│   │   └── platform.go
│   ├── eventgpt/
│   │   └── platform.go
│   ├── vendornet/
│   │   └── platform.go
│   ├── homerescue/
│   │   └── platform.go
│   ├── server.go
│   └── handlers.go
├── recommendation-engine/
│   ├── engine.go
│   ├── ml_service.py
│   └── api/
├── database/
│   ├── 001_core_schema.sql
│   └── 002_seed_data.sql
├── docs/
│   ├── PLATFORM_CONCEPTS_SUMMARY.md
│   ├── cluster_deep_dive_part1.md
│   ├── cluster_deep_dive_part2.md
│   └── Vendor_Platform_Strategy_Document.docx
└── business-models/
    └── business_model_canvases.md
```

## Verification

After pushing, verify:

```bash
# Check remote
git remote -v

# Check status
git status

# View on GitHub
gh repo view --web
```

## Claude Code Commands

Copy and paste these commands into Claude Code:

```
# Navigate to project
cd vendorplatform

# Initialize and push
git init
git add .
git commit -m "feat: Initial commit - VendorPlatform contextual commerce orchestration"
gh repo create BillyRonksGlobal/vendorplatform --private -y
git push -u origin main
```

## Troubleshooting

### Authentication Issues
```bash
# Configure Git credentials
gh auth login

# Or use SSH
git remote set-url origin git@github.com:BillyRonksGlobal/vendorplatform.git
```

### Large Files
If any files exceed GitHub's 100MB limit:
```bash
# Install Git LFS
git lfs install

# Track large files
git lfs track "*.pkl"
git lfs track "*.h5"

# Add .gitattributes
git add .gitattributes
```

### Branch Issues
```bash
# Rename branch to main
git branch -m master main

# Force push if needed (careful!)
git push -f origin main
```
