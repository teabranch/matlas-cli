# Documentation Tracking

## [2025-12-11] Documentation Link Fixes

**Status**: Completed
**Developer**: Assistant
**Related Issues**: User reported broken documentation links

### Summary
Fixed all broken and incorrect links in the documentation site to resolve Jekyll routing issues and 404 errors.

### Tasks
- [x] Add missing permalinks to all example pages
- [x] Fix broken links to non-existent /examples/advanced/ page
- [x] Fix raw links missing Jekyll relative_url filter
- [x] Fix incorrect yaml-kinds permalink references
- [x] Verify all documentation links are correct

### Files Modified

#### Added Permalinks (7 files)
- `docs/examples/clusters.md` - Added permalink: /examples/clusters/
- `docs/examples/discovery.md` - Added permalink: /examples/discovery/
- `docs/examples/users.md` - Added permalink: /examples/users/
- `docs/examples/roles.md` - Added permalink: /examples/roles/
- `docs/examples/network.md` - Added permalink: /examples/network/
- `docs/examples/infrastructure.md` - Added permalink: /examples/infrastructure/
- `docs/examples/dag-analysis.md` - Added permalink: /examples/dag-analysis/

#### Fixed Broken Links (2 files)
- `docs/examples.md` - Changed "Search & VPC" to link to DAG Analysis
- `docs/examples/network.md` - Changed VPC Endpoints to link to YAML Kinds Reference

#### Fixed Raw Links (5 files)
- `docs/infra.md` - Fixed 4 links missing relative_url filter
- `docs/dag-engine.md` - Fixed 3 links in Further Reading
- `docs/atlas.md` - Fixed 1 link to /infra/
- `docs/database.md` - Fixed 1 link to /atlas/
- `docs/examples/dag-analysis.md` - Fixed links in Further Reading

#### Fixed Permalink Paths (3 files)
- `docs/alerts.md` - Updated /yaml-kinds/ to /reference/
- `docs/examples/alerts.md` - Updated /yaml-kinds/ to /reference/
- `docs/yaml-kinds.md` - Fixed malformed Related Documentation links

### Commits
1. `2493b5e` - docs: add missing permalinks to example pages
2. `e22a563` - docs: fix broken links to non-existent /examples/advanced/ page
3. `a44fbc5` - docs: fix raw links missing Jekyll relative_url filter
4. `054daf5` - docs: fix incorrect yaml-kinds permalink references

### Notes
- Main issue: Example pages were accessible at incorrect URLs (e.g., /examples/clusters.html vs /examples/clusters/)
- Root cause: Missing `permalink` frontmatter in Jekyll pages
- Secondary issues: Links using wrong paths and missing relative_url filters
- All changes pushed to `fix/security-patches` branch

---

## [2025-12-11] GitHub Squash Merge Configuration

**Status**: Completed
**Developer**: Danny Teller / Assistant
**Related Issues**: semantic-release not triggering releases due to merge commit format

### Summary
Configured repository for squash merge only to ensure proper semantic versioning and automated releases. Created comprehensive documentation for contributors on conventional commits and squash merge workflow.

### Tasks
- [x] Create PR template with conventional commit format guidance
- [x] Create CONTRIBUTING.md with full development guidelines
- [x] Create SQUASH_MERGE.md with GitHub configuration instructions
- [x] Update README.md with contributing section
- [x] Add tracking entry for documentation changes

### Files Modified
- `.github/pull_request_template.md` - NEW: PR template enforcing conventional commits
- `.github/CONTRIBUTING.md` - NEW: Comprehensive contributing guide with squash merge policy
- `.github/SQUASH_MERGE.md` - NEW: GitHub repository configuration guide
- `README.md` - Added contributing section linking to new documentation
- `tracking/documentation.md` - Added this tracking entry

### Notes

**Problem Identified:**
- Merge commit message "Fix/security patches (#13)" didn't follow Conventional Commits format
- semantic-release couldn't parse commit type, didn't trigger release
- Individual commits in PR were correct, but merge commit format was wrong

**Solution Implemented:**
1. **PR Template**: Guides contributors to provide conventional commit message for squash
2. **Contributing Guide**: Complete documentation on:
   - Conventional Commits specification
   - Commit types and their version impacts
   - Scope guidelines
   - Breaking change syntax
   - Feature development workflow
   - Testing requirements
   
3. **Squash Merge Guide**: Instructions for repository maintainers to:
   - Configure GitHub settings (disable merge commits and rebase)
   - Enable squash merge only
   - Configure branch protection rules
   - Use PR title/description for commit message

**Benefits:**
- Clean git history (one commit per feature/fix)
- Automatic semantic versioning from commit messages
- Proper changelog generation
- Easy rollbacks (single commit per feature)
- Consistent commit format enforcement

**Next Steps:**
- Configure GitHub repository settings via web UI or CLI
- Test with a sample PR to verify semantic-release triggers properly
- Consider adding GitHub Action to validate PR titles (optional enhancement)

**Configuration Required (Manual):**
```bash
gh repo edit teabranch/matlas-cli \
  --allow-merge-commit=false \
  --allow-rebase-merge=false \
  --allow-squash-merge=true \
  --delete-branch-on-merge=true
```

---

## [2025-12-11] Semantic-Release Configuration Fix

**Status**: Completed
**Developer**: Danny Teller / Assistant
**Related Issues**: #13, #14 - semantic-release not triggering despite properly formatted commits

### Summary
Fixed semantic-release configuration by adding explicit releaseRules to the commit-analyzer plugin. The root cause was that presetConfig.types only controls changelog visibility, not what triggers releases.

### Tasks
- [x] Identify root cause of release failures
- [x] Add releaseRules to @semantic-release/commit-analyzer
- [x] Add security commit type support
- [x] Update PR template with security type
- [x] Update CONTRIBUTING.md with complete commit type table
- [x] Create comprehensive summary documentation

### Files Modified
- `.releaserc.json` - Added complete releaseRules configuration
- `.github/CONTRIBUTING.md` - Added security type to commit types table
- `.github/pull_request_template.md` - Added security checkbox option
- `RELEASE_FIX_SUMMARY.md` - NEW: Complete documentation of root cause and solution
- `tracking/documentation.md` - Added this tracking entry

### Root Cause

The `.releaserc.json` configuration had `presetConfig.types` but no explicit `releaseRules` in the commit-analyzer plugin. This caused semantic-release to use default rules which only trigger releases for `feat` and `fix` commits.

**Key Discovery**: `presetConfig.types` only controls:
- What sections appear in the changelog
- Whether commits are hidden (hidden: true/false)

It does NOT control what triggers releases. That requires explicit `releaseRules`.

### Solution Implemented

Added explicit `releaseRules` to `@semantic-release/commit-analyzer`:

**Commits that trigger releases:**
- feat → minor (0.X.0)
- fix → patch (0.0.X)
- security → patch (0.0.X) [NEW]
- perf → patch (0.0.X) [NEW]
- docs → patch (0.0.X) [NEW]
- refactor → patch (0.0.X) [NEW]
- revert → patch (0.0.X) [NEW]

**Commits that don't trigger releases:**
- style → release: false
- chore → release: false
- test → release: false
- build → release: false
- ci → release: false

### Commits Made

1. `b57ac9a` - fix(ci): add releaseRules to commit-analyzer to trigger releases for docs commits
2. `32b1931` - fix(ci): add security type and explicit release rules for all commit types

### Impact

Now all properly formatted conventional commits will correctly trigger releases:
- docs commits (like PR #14) will trigger patch releases
- security commits (like in PR #13) will trigger patch releases
- perf commits (like in PR #13) will trigger patch releases
- Previous docs/perf/security commits were properly formatted but ignored

### Expected Result

Push to main should trigger v4.0.1 release including:
- 2 CI fixes (releaseRules additions)
- 2 documentation updates (squash merge docs)
- Changelog with both Bug Fixes and Documentation sections

### Notes

This explains why:
1. PR #13 didn't trigger a release (bad merge commit format + missing releaseRules)
2. PR #14 didn't trigger a release (missing releaseRules for docs type)
3. Manual commit amendments didn't help (still missing releaseRules)

The squash merge configuration is still valuable for ensuring clean commit messages, but it couldn't fix the underlying missing releaseRules configuration.

---
