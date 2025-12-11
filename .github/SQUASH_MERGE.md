# GitHub Squash Merge Configuration

This repository requires **squash merging** for all pull requests to ensure proper semantic versioning and automated releases.

## Repository Settings

Configure the following settings in your GitHub repository:

### 1. Enable Squash Merging Only

Go to: **Settings → General → Pull Requests**

Configure:
- ✅ **Allow squash merging** - ENABLED
- ❌ **Allow merge commits** - DISABLED
- ❌ **Allow rebase merging** - DISABLED

### 2. Configure Squash Merge Commit Message

Under "Allow squash merging", select:
- **Default commit message**: "Pull request title and description"
- ✅ **Suggest including pull request description in commit message** - ENABLED

This ensures the PR title and description are used for the squash commit message.

### 3. Enable Branch Protection (Recommended)

Go to: **Settings → Branches → Branch protection rules**

Add rule for `main` branch:
- ✅ **Require pull request reviews before merging**
- ✅ **Require status checks to pass before merging**
  - Select required checks: Release workflow, tests, linting
- ✅ **Require conversation resolution before merging**
- ✅ **Do not allow bypassing the above settings**

## Why Squash Merge?

### Benefits

1. **Clean Git History**: Each feature/fix is a single commit on main
2. **Semantic Versioning**: Commit messages drive automated releases
3. **Conventional Commits**: PR template enforces proper format
4. **Easy Rollbacks**: Revert an entire feature with single commit revert
5. **Better Changelogs**: One commit per feature in release notes

### The Problem with Merge Commits

When using regular merge commits:
- semantic-release analyzes the **merge commit message**
- Merge commit format is often: "Merge pull request #123 from branch"
- This doesn't match Conventional Commits format
- Result: **No release is triggered** ❌

### Example Comparison

**Before (Merge Commit - BROKEN):**
```
Fix/security patches (#13)
└─ Merge commit message doesn't follow convention
└─ semantic-release: "The commit should not trigger a release"
```

**After (Squash Merge - WORKS):**
```
fix(security): add secure file operations and credential masking (#13)
└─ Follows Conventional Commits format
└─ semantic-release: Triggers patch release 🎉
```

## How to Configure

### Option 1: Via GitHub Web UI

1. Navigate to your repository settings
2. Go to **Settings → General**
3. Scroll to **Pull Requests** section
4. Configure as described above
5. Save changes

### Option 2: Via GitHub CLI

```bash
# Disable merge commits and rebase merging
gh repo edit owner/repo \
  --allow-merge-commit=false \
  --allow-rebase-merge=false \
  --allow-squash-merge=true

# Enable auto-delete head branches (recommended)
gh repo edit owner/repo --delete-branch-on-merge=true
```

### Option 3: Via GitHub API

```bash
curl -X PATCH \
  -H "Accept: application/vnd.github.v3+json" \
  -H "Authorization: token YOUR_TOKEN" \
  https://api.github.com/repos/owner/repo \
  -d '{
    "allow_merge_commit": false,
    "allow_rebase_merge": false,
    "allow_squash_merge": true,
    "delete_branch_on_merge": true
  }'
```

## Workflow for Contributors

### 1. Create Feature Branch

```bash
git checkout -b feature/my-feature
```

### 2. Make Multiple Commits (Normal Development)

```bash
git commit -m "wip: start implementing feature"
git commit -m "add tests"
git commit -m "fix linting issues"
git commit -m "address review comments"
```

These commits can use any format - they'll be squashed!

### 3. Open Pull Request

Use the PR template to provide:
- Clear description
- Type of change (feat, fix, docs, etc.)
- **Conventional commit message** for the squash

### 4. PR Gets Squashed on Merge

All 4 commits become 1 commit with the conventional message:

```
feat(api): add user authentication endpoints (#123)

Implemented JWT-based authentication with refresh tokens.
Added middleware for protecting routes.
Includes comprehensive test coverage.

Closes: #456
```

## PR Template Enforcement

The PR template guides contributors to:

1. **Select commit type** (feat, fix, docs, etc.)
2. **Specify scope** (atlas, database, infra, etc.)
3. **Write conventional commit message** that will be used for squash
4. **Include breaking change info** if applicable

## Verification

After merging a PR with squash merge:

1. Check the commit on `main` branch
2. Verify it follows Conventional Commits format
3. Confirm semantic-release workflow triggers
4. Verify release is created (if applicable)

## Troubleshooting

### Problem: Release Not Triggered

**Check:**
- ✅ Squash merge is enabled
- ✅ PR title/description follows Conventional Commits
- ✅ Commit type triggers release (feat, fix, perf, refactor, docs)
- ✅ Not a hidden type (test, build, ci, chore)

**Fix:**
- Ensure PR template is filled correctly
- Amend commit message on main if needed:
  ```bash
  git commit --amend -m "feat: proper conventional commit message"
  git push --force-with-lease
  ```

### Problem: Wrong Commit Message After Merge

**Cause:** PR title/description didn't follow template

**Prevention:**
- Enable branch protection with required reviews
- Reviewers should verify commit message format
- Add GitHub Action to validate PR titles (optional)

### Problem: Multiple Releases Triggered

**Cause:** Multiple PRs merged with different types

**Expected:** Each merged PR can trigger a release if it contains:
- `feat` → Minor version bump
- `fix`, `perf`, `refactor`, `docs` → Patch version bump

## Additional Resources

- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [semantic-release Documentation](https://semantic-release.gitbook.io/)
- [GitHub Squash Merge Docs](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/about-pull-request-merges#squash-and-merge-your-commits)

## Summary

✅ **Enable squash merge only**
✅ **Use PR template for commit messages**
✅ **Follow Conventional Commits format**
✅ **Enable branch protection**
✅ **Review PR commit messages before merging**

This ensures semantic-release can properly analyze commits and trigger releases automatically! 🚀
