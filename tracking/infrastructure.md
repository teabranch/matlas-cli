# Infrastructure Tracking

This file tracks build system, CI/CD, deployment, and infrastructure changes.

## Template for New Entries

```markdown
## [YYYY-MM-DD] [Brief Description]

**Status**: [In Progress | Completed | Cancelled]  
**Developer**: [Name/Username]  
**Related Issues**: [#123, #456]  

### Summary
Brief description of the infrastructure work.

### Tasks
- [x] Task 1 description
- [x] Task 2 description  
- [ ] Task 3 description

### Files Modified
- `path/to/file1.yml` - Description of changes
- `path/to/file2.yaml` - Description of changes

### Notes
Any important decisions, blockers, or context for future developers.

---
```

## [2025-01-07] Release Workflow Improvement

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: Release workflow best practices  

### Summary
Completely redesigned the release workflow to follow best practices by separating pre-releases from official releases, eliminating duplicate artifact building, and creating a clear branching strategy for different release types.

### Tasks
- [x] Analyze current workflow problems and design better release strategy
- [x] Update semantic-release configuration to create proper pre-releases for main branch
- [x] Refactor CI workflow to properly integrate with release process
- [x] Update release workflow to use CI-built artifacts instead of rebuilding
- [x] Implement proper branching strategy with main for pre-releases and tags for official releases
- [x] Update workflow documentation to explain the new release process

### Files Modified
- `.releaserc.json` - Updated semantic-release config for dual-branch strategy (main=pre-release, release=official)
- `package.json` - Removed unnecessary @semantic-release/exec dependency
- `.github/workflows/semantic-release.yml` - Updated to support main and release branches
- `.github/workflows/ci.yml` - Added consolidated artifact creation with checksums
- `.github/workflows/release.yml` - Redesigned to download CI artifacts instead of rebuilding
- `docs/release-process.md` - Created comprehensive release process documentation
- `docs/_config.yml` - Added release process to navigation
- `CHANGELOG.md` - Documented workflow improvements

### Notes
**Major Improvements:**
1. **Branching Strategy**: Main branch creates `-main.X` pre-releases, release branch creates official releases
2. **Artifact Efficiency**: CI builds artifacts once, release workflow reuses them
3. **Clear Separation**: Development builds vs stable releases are clearly differentiated
4. **Documentation**: Complete guide for developers on how to create releases

**Technical Changes:**
- Semantic-release now supports dual branches with different release types
- CI workflow consolidates all build artifacts with checksums
- Release workflow downloads existing CI artifacts instead of rebuilding
- Added comprehensive documentation with troubleshooting guide

This fixes the confusing release process where every main branch push created incomplete releases.

---

## [2025-01-07] Release Workflow Critical Fix

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: Manual tag artifact issue, checksums not created  

### Summary
Fixed critical issues with the release workflow where manual tags couldn't find CI artifacts because checksums were only created for specific branches, and simplified the overly complex dual-branch strategy.

### Tasks
- [x] Fix CI workflow so checksums are created for all builds, not just specific branches
- [x] Simplify semantic-release to only work with main branch and normal releases
- [x] Fix release workflow to properly find and download CI artifacts for any commit
- [x] Ensure manual tag creation properly triggers release with artifacts
- [x] Update documentation to reflect simplified single-branch strategy

### Files Modified
- `.github/workflows/ci.yml` - Removed branch-specific condition from create-checksums job
- `.releaserc.json` - Simplified to single main branch strategy  
- `.github/workflows/semantic-release.yml` - Removed release branch support
- `.github/workflows/release.yml` - Enhanced artifact finding and extraction with fallbacks
- `docs/release-process.md` - Updated to reflect simplified single-branch strategy
- `CHANGELOG.md` - Documented critical fixes

### Notes
**Critical Issues Fixed:**
1. **Checksum Creation**: CI now creates checksums for ALL builds, not just main/release branches
2. **Artifact Download**: Release workflow can now find artifacts for any commit with successful CI
3. **Manual Tags**: Manual tag creation now works properly with artifact attachment
4. **Simplified Strategy**: Eliminated confusing dual-branch pre-release approach

**Technical Improvements:**
- Better error handling in release workflow with detailed logging
- Fallback to individual platform artifacts if consolidated artifacts aren't found
- Robust commit SHA lookup for artifact matching
- Enhanced artifact extraction handling

This resolves the issue where manual tags from main commits couldn't find their CI artifacts.

---

## [2025-01-07] Release Timing Fix

**Status**: Completed  
**Developer**: Assistant  
**Related Issues**: Releases created before CI artifacts ready  

### Summary
Fixed critical timing issue where semantic-release workflow was creating releases in parallel with CI workflow, resulting in empty releases without binary attachments.

### Tasks
- [x] Add wait-for-ci job to semantic-release workflow
- [x] Make semantic-release depend on CI completion
- [x] Update documentation to reflect sequential workflow
- [x] Ensure releases are only created after artifacts are ready

### Files Modified
- `.github/workflows/semantic-release.yml` - Added wait-for-ci job with polling for CI completion
- `docs/release-process.md` - Updated process documentation to show CI wait step

### Notes
**Problem**: When pushing to main, both CI and semantic-release workflows started simultaneously. Semantic-release would create a release immediately while CI was still building artifacts, resulting in releases without binaries.

**Solution**: Added a `wait-for-ci` job that polls the GitHub API every 30 seconds to check if the CI workflow for the same commit has completed successfully. Only after CI succeeds does semantic-release proceed to create the release.

**Technical Details**:
- 40-minute timeout for CI completion
- 30-second polling interval
- Checks CI workflow status for the exact commit SHA
- Fails if CI workflow fails or times out
- Provides detailed logging of CI workflow status

This ensures that releases always have their binary artifacts attached from the start.

---

