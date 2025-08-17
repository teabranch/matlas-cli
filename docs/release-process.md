---
layout: page
title: Release Process
permalink: /release-process/
---

# Release Process

This document explains the automated release process for matlas-cli, including how pre-releases and official releases work.

## Release Strategy Overview

We use a dual-branch strategy for releases:

- **Main branch** → Creates pre-releases with `-main.X` suffix
- **Release branch** → Creates official stable releases

## Release Types

### Pre-Releases (Development)

**Trigger:** Push to `main` branch with conventional commit messages

**Release Format:** `v1.2.3-main.1`, `v1.2.3-main.2`, etc.

**Purpose:** 
- Continuous delivery from main branch
- Testing new features before official release
- Development builds with all latest changes

**Process:**
1. Developer pushes to main with conventional commits (`feat:`, `fix:`, etc.)
2. CI workflow builds and tests the code
3. Semantic-release creates a pre-release tag
4. Release workflow attaches binary artifacts to the pre-release

### Official Releases (Stable)

**Trigger:** Push to `release` branch or manual release creation

**Release Format:** `v1.2.3`, `v2.0.0`, etc.

**Purpose:**
- Stable releases for production use
- Semantic versioning for breaking changes
- Official distribution releases

**Process:**
1. Create release branch from main: `git checkout -b release && git push origin release`
2. CI workflow builds and tests the code
3. Semantic-release creates an official release tag
4. Release workflow attaches binary artifacts to the release

## Conventional Commits

Release automation is driven by conventional commit messages:

| Commit Type | Effect | Example |
|-------------|--------|---------|
| `feat:` | Minor version bump | `feat: add database backup command` |
| `fix:` | Patch version bump | `fix: resolve connection timeout issue` |
| `feat!:` | Major version bump | `feat!: redesign CLI interface` |
| `BREAKING CHANGE:` | Major version bump | (in commit footer) |
| `docs:`, `test:`, `chore:` | No version bump | `docs: update API reference` |

## Release Workflow Details

### 1. CI Workflow (`ci.yml`)
- Runs on every push to main, release, and PRs
- Builds binaries for Linux, macOS, Windows (both Intel and ARM)
- Creates checksums for all artifacts
- Uploads consolidated artifacts for use by release process

### 2. Semantic Release (`semantic-release.yml`)
- Runs on push to `main` or `release` branches
- Analyzes commit history using conventional commits
- Creates appropriate release (pre-release for main, stable for release)
- Updates CHANGELOG.md automatically

### 3. Release Assets (`release.yml`)
- Triggers when a GitHub release is published
- Downloads CI artifacts for the specific commit
- Attaches binary distributions to the GitHub release
- Provides installation instructions

## How to Create Releases

### Development Pre-Release
```bash
# Just push to main with conventional commits
git checkout main
git add .
git commit -m "feat: add new feature"
git push origin main

# This automatically creates: v1.2.3-main.1
```

### Official Release
```bash
# Option 1: Create release branch
git checkout main
git checkout -b release
git push origin release  # This creates official release v1.2.3

# Option 2: Manual release via GitHub UI
# Go to GitHub → Releases → "Create a new release"
# Choose a tag version following semantic versioning
```

## Artifact Distribution

Each release includes:
- **Linux** (AMD64, ARM64): `.tar.gz` and `.zip` files
- **macOS** (Intel, Apple Silicon): `.tar.gz` and `.zip` files  
- **Windows** (AMD64): `.zip` files
- **Checksums**: `checksums.txt` with SHA256 hashes

## Release Branch Management

### Creating a Release Branch
```bash
# From main branch (ensure it's up to date)
git checkout main
git pull origin main

# Create and push release branch
git checkout -b release
git push origin release
```

### Cleanup After Release
```bash
# Delete local release branch
git branch -d release

# Delete remote release branch (optional)
git push origin --delete release
```

## Troubleshooting

### Release Not Created
- Check commit messages follow conventional commit format
- Ensure commits since last release warrant a version bump
- Check GitHub Actions logs for semantic-release workflow

### Missing Binary Assets
- Verify CI workflow completed successfully
- Check that release workflow found and downloaded CI artifacts
- Review release workflow logs for artifact download errors

### Pre-release vs Official Release
- Pre-releases are created from `main` branch pushes
- Official releases require `release` branch or manual tag creation
- Check which branch the commit was pushed to

## Manual Override

If automated release fails, you can manually create a release:

```bash
# Create and push a tag manually
git tag v1.2.3
git push origin v1.2.3

# This triggers the release workflow to attach artifacts
```

## Version Strategy

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes that require user action
- **MINOR**: New features that are backward compatible  
- **PATCH**: Bug fixes and small improvements

Pre-releases append `-main.X` to indicate development builds.
