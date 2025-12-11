# Squash Merge Setup - Quick Start Guide

This guide will help you configure GitHub for squash merge and verify the setup works with semantic-release.

## ✅ What Was Done

Created comprehensive documentation and templates for squash merge workflow:

1. **`.github/pull_request_template.md`** - PR template guiding contributors to:
   - Select commit type (feat, fix, docs, etc.)
   - Specify scope (atlas, database, infra, etc.)
   - Write conventional commit message for squash
   - Mark breaking changes
   - Complete checklist

2. **`.github/CONTRIBUTING.md`** - Full contributing guide covering:
   - Development setup and workflow
   - Conventional Commits specification
   - Squash merge process explanation
   - Code style and testing requirements
   - Feature development guidelines

3. **`.github/SQUASH_MERGE.md`** - Repository configuration instructions:
   - Step-by-step GitHub settings configuration
   - CLI commands for automation
   - Explanation of why squash merge is required
   - Troubleshooting guide

4. **`README.md`** - Added contributing section with links

5. **`tracking/documentation.md`** - Tracked this work

## 🔧 Next Steps: Configure GitHub Repository

### Option 1: Via GitHub Web UI (Recommended)

1. Go to: https://github.com/teabranch/matlas-cli/settings

2. Scroll to **"Pull Requests"** section

3. Configure merge options:
   - ✅ **Allow squash merging** - ENABLED
   - ❌ **Allow merge commits** - DISABLED  
   - ❌ **Allow rebase merging** - DISABLED

4. Under "Allow squash merging", select:
   - **Default commit message**: "Pull request title and description"
   - ✅ **Suggest including pull request description** - ENABLED

5. Recommended: Enable **"Automatically delete head branches"**

6. Click **"Save changes"**

### Option 2: Via GitHub CLI (Fast)

```bash
gh repo edit teabranch/matlas-cli \
  --allow-merge-commit=false \
  --allow-rebase-merge=false \
  --allow-squash-merge=true \
  --delete-branch-on-merge=true
```

### Option 3: Via GitHub API

```bash
curl -X PATCH \
  -H "Accept: application/vnd.github.v3+json" \
  -H "Authorization: token YOUR_GITHUB_TOKEN" \
  https://api.github.com/repos/teabranch/matlas-cli \
  -d '{
    "allow_merge_commit": false,
    "allow_rebase_merge": false,
    "allow_squash_merge": true,
    "delete_branch_on_merge": true
  }'
```

## 🧪 Testing the Configuration

### 1. Push This Branch and Create a PR

```bash
# Push current branch
git push origin fix/release-11122025

# Create PR using the new template
gh pr create --title "docs(ci): enforce squash merge with conventional commits" \
  --body "$(cat <<'EOF'
## Description

Configure repository for squash merge only to ensure proper semantic versioning and automated releases.

## Type of Change

- [x] docs: Documentation updates

## Scope

**Scope**: ci

## Breaking Changes

- [ ] This PR contains breaking changes

## Commit Message

```
docs(ci): enforce squash merge with conventional commits for automated releases

Add comprehensive documentation and templates to enforce squash merge workflow
and conventional commits, fixing semantic-release integration.

Fixes the issue where merge commit format prevented semantic-release from
triggering releases. Now all PRs will be squashed with proper conventional
commit messages.

Refs: #13
```

## Checklist

- [x] Code follows project style guidelines
- [x] Self-review completed
- [x] Documentation updated (PR template, contributing guide, squash merge guide)
- [x] CHANGELOG.md updated under \`## [Unreleased]\` section
- [x] Examples not needed (documentation only)

## Related Issues

Refs: #13

## Additional Context

This solves the problem where PR #13 merged with message "Fix/security patches (#13)"
which didn't follow Conventional Commits format, causing semantic-release to skip the release.
EOF
)"
```

### 2. Verify PR Template Appears

When you create the PR, GitHub should show the PR template with all sections pre-filled.

### 3. Merge the PR

1. Get PR reviewed and approved
2. Click "Squash and merge" button
3. Verify the commit message is formatted correctly:
   ```
   docs(ci): enforce squash merge with conventional commits for automated releases (#XX)
   ```
4. Merge the PR

### 4. Verify Semantic-Release Triggers

After merging to `main`:

1. Go to: https://github.com/teabranch/matlas-cli/actions

2. Find the **"Release"** workflow run that triggered after merge

3. Check the logs for semantic-release analyzing the commit:
   ```
   [semantic-release] [@semantic-release/commit-analyzer] › ℹ  Analyzing commit: docs(ci): enforce squash merge...
   [semantic-release] [@semantic-release/commit-analyzer] › ℹ  The release type for the commit is patch
   [semantic-release] › ✔  Completed step "analyzeCommits"
   ```

4. Verify a new release is created (for `docs` type, it triggers a patch version bump)

5. Check GitHub Releases: https://github.com/teabranch/matlas-cli/releases

## 📊 Expected Results

### Before (Broken)
```
Commit: "Fix/security patches (#13)"
Result: semantic-release says "no release"
Reason: Doesn't match Conventional Commits format
```

### After (Fixed)
```
Commit: "docs(ci): enforce squash merge with conventional commits (#XX)"
Result: semantic-release triggers patch release (0.0.X)
Reason: Matches Conventional Commits format
```

## 🎯 Commit Type → Version Mapping

| Commit Type | Triggers Release? | Version Bump | In Changelog? |
|-------------|-------------------|--------------|---------------|
| `feat` | ✅ Yes | Minor (0.X.0) | ✅ Features |
| `fix` | ✅ Yes | Patch (0.0.X) | ✅ Bug Fixes |
| `perf` | ✅ Yes | Patch (0.0.X) | ✅ Performance |
| `refactor` | ✅ Yes | Patch (0.0.X) | ✅ Refactoring |
| `docs` | ✅ Yes | Patch (0.0.X) | ✅ Documentation |
| `test` | ❌ No | None | ❌ Hidden |
| `build` | ❌ No | None | ❌ Hidden |
| `ci` | ❌ No | None | ❌ Hidden |
| `chore` | ❌ No | None | ❌ Hidden |
| `style` | ❌ No | None | ❌ Hidden |

**Note:** `docs` type DOES trigger a patch release according to `.releaserc.json` configuration.

## 🔍 Troubleshooting

### Issue: PR Template Not Showing

**Solution:** Template must be at `.github/pull_request_template.md` (already done)

### Issue: Can't Select Squash Merge

**Solution:** Configure repository settings as shown above

### Issue: Wrong Commit Message After Merge

**Cause:** PR title/description didn't follow template

**Fix:** Amend commit on main:
```bash
git checkout main
git pull
git commit --amend -m "proper conventional commit message"
git push --force-with-lease
```

### Issue: Release Still Not Triggered

**Check:**
1. Commit message follows format: `type(scope): description`
2. Commit type is one that triggers releases (feat, fix, docs, perf, refactor)
3. Commit is on `main` branch
4. GitHub Actions are enabled
5. Check workflow logs for errors

## 📚 Additional Documentation

- **Full contributing guide**: `.github/CONTRIBUTING.md`
- **Squash merge details**: `.github/SQUASH_MERGE.md`
- **Conventional Commits spec**: https://www.conventionalcommits.org/
- **semantic-release docs**: https://semantic-release.gitbook.io/

## 🎉 Success Criteria

✅ Repository configured for squash merge only
✅ PR template guides contributors to conventional commits
✅ Merge creates single commit with proper format
✅ semantic-release analyzes commit and triggers release
✅ Changelog automatically updated
✅ GitHub Release created with binaries

---

**Ready to configure GitHub and test!** 🚀

Start with configuring the repository settings, then test by creating a PR from this branch.
