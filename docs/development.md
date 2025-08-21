---
layout: default
title: Development Guide
nav_order: 3
parent: Reference
description: Development guide for contributing to matlas CLI
permalink: /reference/development/
---

# Development Guide

This guide explains how to develop new features in matlas-cli using our established workspace rules, tracking systems, and development workflow.

## Overview

matlas-cli follows a structured development approach that emphasizes:
- **Feature Interface Consistency** - Every feature must provide both CLI and YAML ApplyDocument support
- **Task Tracking** - Comprehensive todo management and permanent tracking for team visibility
- **Documentation Standards** - Consistent documentation and changelog maintenance
- **Conventional Commits** - Automated versioning and release management

---

## Development Workflow

### 1. Planning and Task Setup

**Always start by creating a todo list** using our task tracking system:

```bash
# Example: Development session starts with structured planning
# The development tools will automatically create todos for complex tasks
```

**Key requirements:**
- Break complex tasks into specific, actionable items
- Update todo status in real-time as work progresses
- Add detailed comments explaining what was accomplished
- Mark todos as completed immediately after finishing each task

**Todo Management Guidelines:**
- Only ONE task should be `in_progress` at a time
- Include clear, descriptive task names
- Use comments to document decisions and file changes
- Provide enough detail for another developer to understand progress

### 2. Feature Interface Consistency

Every new user-facing feature MUST provide both:

#### CLI Interface
- Add or extend subcommands in the correct group:
  - `cmd/infra/*` - Infrastructure operations (plan/apply/diff/show/validate)
  - `cmd/atlas/**/*` - Atlas resources (projects/clusters/users/network)
  - `cmd/database/**/*` - MongoDB database operations
  - `cmd/config/*` - CLI configuration and credentials

#### YAML ApplyDocument Support
- Define or extend types in `internal/types/apply.go` and `internal/types/*.go`
- Load and map kinds in `internal/apply/loader.go`
- Validate schema in `internal/apply/validation.go` and `internal/validation/schema.go`
- Wire execution to services in `internal/apply/executor.go` and `internal/apply/fetchers.go`
- Update diff/dry-run output in `internal/apply/diff_formatter.go` and `internal/apply/dryrun_formatter.go`

**Both CLI and YAML paths must use the same service-layer implementations** in `internal/services/*` to avoid divergence.

### 3. Service Layer Architecture

All business logic belongs in `internal/services/*`:
- `internal/services/atlas/` - Atlas SDK operations
- `internal/services/database/` - MongoDB driver operations  
- `internal/services/discovery/` - Resource discovery logic

This ensures consistency between CLI commands and YAML ApplyDocument processing.

### 4. Documentation and Examples

For every new feature:
- **Documentation**: Update relevant pages under `docs/`
- **Examples**: Add minimal YAML examples to `examples/`
- **Feature Tracking**: Create a summary file under `features/` using the template

---

## Task Tracking System

### Session-Level Tracking (todo_write tool)

Used for immediate planning and progress tracking within development sessions:

```markdown
Example todo structure:
1. "Set up database connection pool" [in_progress]
2. "Implement user CRUD operations" [pending] 
3. "Add input validation middleware" [pending]
4. "Write integration tests" [pending]
```

**Completion Comments Format:**
```markdown
"Completed: Set up database connection pool in internal/database/pool.go. 
Configured max connections (50), idle timeout (30s), and connection retry logic. 
Added health check endpoint. Tested with local PostgreSQL instance. 
Ready for user operations implementation."
```

### Permanent Tracking (tracking/ directory)

Create entries in category-based files for team visibility:

- **`tracking/features.md`** - New functionality, enhancements
- **`tracking/bugfixes.md`** - Bug resolutions, patches
- **`tracking/refactoring.md`** - Code improvements without behavior changes
- **`tracking/documentation.md`** - Documentation updates
- **`tracking/infrastructure.md`** - Build, CI/CD, deployment changes
- **`tracking/performance.md`** - Optimization work

**Entry Format:**
```markdown
## [YYYY-MM-DD] [Brief Description]

**Status**: [In Progress | Completed | Cancelled]  
**Developer**: [Name/Username]  
**Related Issues**: [#123, #456]  

### Summary
Brief description of the work being done.

### Tasks
- [x] Task 1 description
- [x] Task 2 description  
- [ ] Task 3 description

### Files Modified
- `path/to/file1.go` - Description of changes
- `path/to/file2.go` - Description of changes

### Notes
Any important decisions, blockers, or context for future developers.

---
```

---

## Feature Tracking Requirements

Every feature MUST include a per-feature summary file:

- **Location**: `features/`
- **Template**: `features/TEMPLATE.md`
- **Naming**: `YYYY-MM-DD-<short-slug>.md` or `FTR-<id>-<short-slug>.md`
- **Minimum content**: Title and 2-6 sentence summary
- **Recommended content**: Map changes across CLI, YAML, services, tests, docs

Example structure:
```markdown
# Feature: User Role Management Distinction

## Summary
Enhanced user management to distinguish between Atlas and database users, 
with separate role systems for improved security and clarity.

## CLI surfaces
- Commands added: `atlas users`, `database users`, `database roles`

## YAML ApplyDocument  
- Kinds added: `AtlasUser`, `DatabaseUser`, `DatabaseRole`

## Service layer
- `internal/services/atlas/users.go` - Atlas user operations
- `internal/services/database/service.go` - Database user/role operations
```

---

## Code Quality Standards

### Error Handling
- Follow standardized error handling patterns for concise vs verbose output
- Preserve real causes and maintain consistent formatting
- Use error recovery mechanisms where appropriate

### Testing
- **Unit tests**: Test individual functions and components
- **Integration tests**: Test cross-component interactions  
- **E2E tests**: Test complete user workflows

### Linting and Validation
- Address linter errors if clear how to fix them
- Don't make uneducated guesses on fixes
- Don't loop more than 3 times on fixing the same file

---

## Documentation Standards

### Jekyll Site Requirements
- All documentation under `docs/` directory
- Include Jekyll frontmatter with `layout`, `title`, `permalink`
- Update navigation in `docs/_config.yml` when adding pages
- Use fenced code blocks with language tags
- Follow GitHub Pages Jekyll theme (minima)

### Local Preview
```bash
cd docs
bundle install
bundle exec jekyll serve
```

---

## Changelog and Release Management

### Changelog Maintenance
- Only update the `## [Unreleased]` section in `CHANGELOG.md`
- Group entries under standard headings: Added, Changed, Fixed, Deprecated, Removed, Security
- Use concise, imperative bullet points with user-facing language
- Link related issues/PRs using `[#123]` format

### Conventional Commits
Required for automated versioning:

```
<type>(<optional scope>)<!>: <short summary>

<optional body>

<optional footer(s)>
```

**Common types:**
- `feat`: user-facing feature (minor version bump)
- `fix`: bug fix (patch version bump)  
- `perf`: performance improvement
- `refactor`: code refactor without behavior change
- `docs`: documentation only
- `test`: tests only

**Breaking changes:** Append `!` or include `BREAKING CHANGE:` footer

---

## Development Best Practices

### File Organization
- Prefer editing existing files over creating new ones
- Never proactively create documentation files unless requested
- Use absolute paths in tool calls when possible
- Clean up temporary files at the end of tasks

### Parallel Development
- Execute multiple read-only operations simultaneously
- Batch todo updates with other tool calls for efficiency
- Plan searches upfront and execute together
- Minimize sequential tool calls unless truly required

### Team Collaboration
- Check existing tracking entries before starting similar work
- Document decisions and trade-offs in tracking notes
- Reference GitHub issues when applicable
- Provide enough context for seamless handoffs

---

## Acceptance Checklist

Before considering a feature complete:

- [ ] CLI subcommand/flags added in correct command group
- [ ] YAML kind exists and supported by ApplyDocument end-to-end
- [ ] Both interfaces call same `internal/services/*` logic
- [ ] Documentation updated under `docs/`
- [ ] Example YAML added to `examples/`
- [ ] Per-feature tracking file created under `features/`
- [ ] Permanent tracking entry added to appropriate category
- [ ] Tests added (unit/integration as appropriate)
- [ ] Changelog updated in `## [Unreleased]` section
- [ ] Conventional commit messages used

---

## Getting Help

- **Feature template**: See `features/TEMPLATE.md` for structured feature documentation
- **Tracking examples**: Review existing entries in `tracking/` directory
- **Code patterns**: Examine existing implementations in similar command groups
- **Documentation**: Follow patterns established in existing `docs/` pages

This development guide ensures consistent, trackable, and maintainable feature development across the matlas-cli project.
