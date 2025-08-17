---
layout: page
title: Release Process
permalink: /release-process/
---

# Release Process

This document explains the automated release process for matlas-cli, including how pre-releases and official releases work.

## Release Strategy Overview

We use a simplified single-branch strategy for releases:

- **Main branch** → Creates official releases using semantic versioning
- **Manual tags** → Can be created from any commit to create releases

## Release Types

### Automatic Releases (Semantic)

**Trigger:** Push to `main` branch with conventional commit messages

**Release Format:** `v1.2.3`, `v2.0.0`, etc.

**Purpose:** 
- Automatic semantic versioning based on commit messages
- Continuous delivery from main branch
- Official releases for any merge to main

**Process:**
1. Developer pushes to main with conventional commits (`feat:`, `fix:`, etc.)
2. CI workflow builds and tests the code, creates artifacts
3. Semantic-release analyzes commits and creates appropriate release
4. Release workflow downloads CI artifacts and attaches them to the release

### Manual Releases

**Trigger:** Manual tag creation from any commit

**Release Format:** `v1.2.3`, `v2.0.0`, etc.

**Purpose:**
- Create releases from specific commits
- Hotfix releases from non-main branches
- Custom release timing

**Process:**
1. Create tag manually: `git tag v1.2.3 <commit-sha> && git push origin v1.2.3`
2. GitHub automatically creates a release from the tag
3. Release workflow finds CI artifacts for that commit and attaches them

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

### Automatic Release (Recommended)
```bash
# Just push to main with conventional commits
git checkout main
git add .
git commit -m "feat: add new feature"
git push origin main

# This automatically creates: v1.2.3 (based on semantic analysis)
```

### Manual Release
```bash
# Option 1: Create tag from current commit
git tag v1.2.3
git push origin v1.2.3

# Option 2: Create tag from specific commit
git tag v1.2.3 abc1234
git push origin v1.2.3

# Option 3: Manual release via GitHub UI
# Go to GitHub → Releases → "Create a new release"
# Choose a tag version following semantic versioning
```

## Artifact Distribution

Each release includes:
- **Linux** (AMD64, ARM64): `.tar.gz` and `.zip` files
- **macOS** (Intel, Apple Silicon): `.tar.gz` and `.zip` files  
- **Windows** (AMD64): `.zip` files
- **Checksums**: `checksums.txt` with SHA256 hashes

## Release Management Tips

### Creating a Release from Specific Commit
```bash
# Find the commit you want to release
git log --oneline -10

# Create a tag from that commit
git tag v1.2.3 abc1234
git push origin v1.2.3
```

### Listing Existing Releases
```bash
# List all tags/releases
git tag -l

# List tags with commit info
git tag -l --format='%(refname:short) %(objectname:short) %(contents:subject)'
```

## Troubleshooting

### Release Not Created
- Check commit messages follow conventional commit format
- Ensure commits since last release warrant a version bump
- Check GitHub Actions logs for semantic-release workflow

### Missing Binary Assets
- Verify CI workflow completed successfully for the commit
- Check that release workflow found and downloaded CI artifacts
- Review release workflow logs for artifact download errors
- Ensure the commit had a successful CI run before creating the release

### Manual Tag Releases
- Manual tags work from any commit that had a successful CI run
- Release workflow will find and download artifacts from the CI run for that commit
- If CI didn't run for a commit, artifacts won't be available

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

Versions are automatically determined by semantic-release based on conventional commit messages.
