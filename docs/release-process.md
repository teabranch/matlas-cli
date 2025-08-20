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
2. Release workflow runs automatically and performs:
   - Code quality checks and linting
   - Cross-platform testing (Ubuntu, macOS, Windows)
   - Multi-platform binary builds (Linux, macOS, Windows for multiple architectures)
   - Checksum generation for all artifacts
3. Semantic-release analyzes commits and creates appropriate release with all binaries attached
4. Optional integration and E2E tests run (if credentials available)

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
3. If the commit had a successful release workflow run, artifacts are already available
4. Otherwise, the workflow must be manually triggered or re-run for that commit

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

### Consolidated Release Workflow (`release.yml`)

All release processes are handled by a single consolidated workflow that runs on every push to main and PRs:

#### 1. Code Quality & Linting
- Runs golangci-lint with errcheck, gosec, and ineffassign
- Checks code formatting with gofmt
- Validates Go modules are up to date

#### 2. Cross-Platform Testing
- Runs unit tests on Ubuntu, macOS, and Windows
- Tests against Go versions 1.23 and 1.24.5
- Generates coverage reports and uploads to Codecov

#### 3. Multi-Platform Build
- Builds binaries for all supported platforms:
  - Linux (AMD64, ARM64)
  - macOS (Intel, Apple Silicon)  
  - Windows (AMD64)
- Creates both .tar.gz and .zip archives for each platform
- Uploads build artifacts for the release process

#### 4. Checksum Generation
- Downloads all platform artifacts
- Generates SHA256 checksums for all archives
- Creates consolidated artifact bundle

#### 5. Semantic Release (main branch only)
- Analyzes commit history using conventional commits
- Determines appropriate version bump
- Creates GitHub release with all platform binaries attached
- Updates CHANGELOG.md automatically
- Only runs on pushes to main branch

#### 6. Integration & E2E Testing (conditional)
- Runs integration tests when labeled or on main branch pushes
- Requires Atlas credentials in repository secrets
- Provides comprehensive end-to-end validation

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
- Check GitHub Actions logs for the release workflow
- Verify the push was to the main branch (semantic release only runs on main)

### Missing Binary Assets
- Verify the release workflow completed successfully for the commit
- Check that all build jobs (lint, test, build, create-checksums) passed
- Review release workflow logs for any artifact creation or upload errors
- Ensure the semantic release job successfully attached artifacts

### Manual Tag Releases
- Manual tags create releases but may not have binary assets attached
- Binary assets are only available if the release workflow ran successfully for that commit
- To get assets for a manual tag, the release workflow must have run on that commit previously
- Consider running the workflow manually from the GitHub Actions UI for the specific commit

## Manual Override

If automated release fails, you can manually create a release:

```bash
# Create and push a tag manually
git tag v1.2.3
git push origin v1.2.3

# This creates a release, but artifacts are only attached if the 
# release workflow previously ran successfully for this commit.
# If needed, manually trigger the workflow from GitHub Actions UI.
```

## Version Strategy

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes that require user action
- **MINOR**: New features that are backward compatible  
- **PATCH**: Bug fixes and small improvements

Versions are automatically determined by semantic-release based on conventional commit messages.
