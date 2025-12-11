# Contributing to matlas-cli

Thank you for your interest in contributing to matlas-cli! This guide will help you get started.

## Table of Contents

- [Development Setup](#development-setup)
- [Pull Request Process](#pull-request-process)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Code Style](#code-style)
- [Testing](#testing)
- [Feature Development](#feature-development)

## Development Setup

### Prerequisites

- **Go 1.24+** required
- Git
- Make (optional, but recommended)

### Getting Started

1. **Fork and clone the repository**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/matlas-cli.git
   cd matlas-cli
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Build the project**:
   ```bash
   make build
   # or
   go build -o bin/matlas ./...
   ```

4. **Run tests**:
   ```bash
   make test
   ```

## Pull Request Process

### Before Submitting

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/my-feature
   # or
   git checkout -b fix/issue-description
   ```

2. **Make your changes** following our code style and guidelines

3. **Test your changes**:
   ```bash
   make test
   make lint
   ```

4. **Update documentation** if needed:
   - Update relevant files in `docs/`
   - Update `CHANGELOG.md` under `## [Unreleased]` section
   - Add examples to `examples/` if introducing new features
   - Create feature tracking file in `features/` using `features/TEMPLATE.md`

### Submitting Your PR

1. **Push your branch**:
   ```bash
   git push origin feature/my-feature
   ```

2. **Open a Pull Request** on GitHub

3. **Fill out the PR template** completely:
   - Provide clear description
   - Select the type of change
   - Specify scope (if applicable)
   - Provide a **conventional commit message** that will be used for squash merge
   - Check all applicable items in the checklist

### Merge Process

**This repository uses SQUASH MERGE ONLY**. When your PR is merged:

- All commits will be squashed into a single commit
- The commit message will be taken from the PR template
- The commit message MUST follow [Conventional Commits](https://www.conventionalcommits.org/) format
- This ensures proper versioning and changelog generation via semantic-release

## Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/) specification for automatic versioning and changelog generation.

### Format

```
<type>(<optional scope>): <short summary>

<optional body>

<optional footer(s)>
```

### Types

| Type | Description | Version Impact | In Changelog |
|------|-------------|----------------|--------------|
| `feat` | New feature | Minor (0.X.0) | ✅ Features |
| `fix` | Bug fix | Patch (0.0.X) | ✅ Bug Fixes |
| `security` | Security improvements | Patch (0.0.X) | ✅ Security |
| `perf` | Performance improvement | Patch (0.0.X) | ✅ Performance |
| `refactor` | Code refactoring | Patch (0.0.X) | ✅ Refactoring |
| `docs` | Documentation only | Patch (0.0.X) | ✅ Documentation |
| `test` | Tests only | None | ❌ Hidden |
| `build` | Build system or deps | None | ❌ Hidden |
| `ci` | CI configuration | None | ❌ Hidden |
| `chore` | Maintenance tasks | None | ❌ Hidden |

### Scopes

Use repository areas for clarity:

- `infra` - Infrastructure/apply workflows
- `atlas` - Atlas API operations
- `database` - Database operations
- `cli` - CLI framework/flags
- `docs` - Documentation
- `types` - Type definitions
- `services` - Service layer
- etc.

### Examples

**Feature:**
```
feat(atlas): add VPC endpoint management commands

Implement create, list, get, and delete operations for Atlas VPC endpoints.
Supports AWS, Azure, and GCP providers.

Closes: #123
```

**Bug fix:**
```
fix(database): correct pagination when limit is provided

The list operation was ignoring the --limit flag when paginating
through results. Now properly respects the limit parameter.

Fixes: #456
```

**Documentation:**
```
docs: update installation instructions for Windows

Added PowerShell examples and troubleshooting section.
```

**Breaking change:**
```
feat(infra)!: remove deprecated --legacy flag from apply command

BREAKING CHANGE: The --legacy flag has been removed. Users should
migrate to the new apply format described in docs/infra.md.

Closes: #789
```

### Breaking Changes

To mark a breaking change, use one of these methods:

1. **Append `!` after type/scope**:
   ```
   feat(api)!: drop support for v1 endpoints
   ```

2. **Include `BREAKING CHANGE:` footer**:
   ```
   feat(infra): update apply pipeline
   
   BREAKING CHANGE: removed --legacy flag, use new format instead
   ```

## Code Style

### Go Code Guidelines

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (automatically applied by `make fmt`)
- Run `make lint` before committing
- Add comments for exported functions and complex logic
- Write meaningful variable and function names

### Error Handling

Follow our error handling standards (see `.cursor/rules/error-handling.mdc`):

- Preserve real causes with wrapped errors
- Support both concise and verbose output modes
- Use consistent error formatting across commands

### Logging

Follow logging guidelines (see `.cursor/rules/logging.mdc`):

- Use appropriate log levels (Debug, Info, Warn, Error)
- Automatically mask sensitive data (credentials, connection strings)
- Provide context in log messages

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific test package
go test ./internal/services/atlas/...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

### Writing Tests

- Write unit tests for new functionality
- Update existing tests when modifying behavior
- Use table-driven tests where appropriate
- Mock external dependencies (Atlas SDK, MongoDB driver)
- Follow existing test patterns in the codebase

### Live Tests

For integration tests with real Atlas/MongoDB instances:

- Place test scripts in `scripts/`
- Follow the live tests policy (`.cursor/rules/live-tests.mdc`)
- Document prerequisites and setup instructions

## Feature Development

When adding new user-facing features, ALWAYS provide:

### 1. CLI Interface

- Add or extend subcommands in the appropriate command group:
  - `cmd/infra/` - Infrastructure workflows
  - `cmd/atlas/` - Atlas resource management
  - `cmd/database/` - Database operations
  - `cmd/config/` - Configuration management

- Use consistent flag naming and patterns
- Update command help text and examples

### 2. YAML ApplyDocument Support

If the feature can be expressed declaratively:

- Define or extend types in `internal/types/`
- Add YAML kind support in `internal/apply/loader.go`
- Implement validation in `internal/apply/validation.go`
- Wire execution in `internal/apply/executor.go`
- Both CLI and YAML must use same `internal/services/*` logic

### 3. Documentation

- Update command documentation in `docs/`
- Add examples to `examples/`
- Update `CHANGELOG.md` under `## [Unreleased]`
- Create feature tracking file in `features/` using template

### 4. Feature Tracking

Create a summary file following `features/TEMPLATE.md`:

```bash
cp features/TEMPLATE.md features/$(date +%F)-<short-slug>.md
```

Minimum content:
- Title: `Feature: <name>`
- Summary: 2-6 sentences describing the feature
- Implementation details (CLI, YAML, services, tests, docs)

See `.cursor/rules/feature-format-support.mdc` for complete requirements.

## Documentation Standards

### Writing Documentation

All documentation must:

- Reside under `docs/` directory
- Use Jekyll frontmatter (layout, title, permalink)
- Follow GitHub Pages Jekyll setup in `docs/_config.yml`
- Include code examples with proper syntax highlighting
- Update navigation in `docs/_config.yml` for new pages

### Preview Documentation Locally

```bash
cd docs
bundle install
bundle exec jekyll serve
```

Visit `http://localhost:4000/matlas-cli/` to preview.

See `.cursor/rules/documentation-standards.mdc` for complete requirements.

## Getting Help

- **Questions?** Open a [Discussion](https://github.com/teabranch/matlas-cli/discussions)
- **Bug Reports:** Open an [Issue](https://github.com/teabranch/matlas-cli/issues)
- **Feature Requests:** Open an [Issue](https://github.com/teabranch/matlas-cli/issues) with the "enhancement" label

## Code of Conduct

Please be respectful and constructive in all interactions. We're building this tool together for the MongoDB community.

---

Thank you for contributing to matlas-cli! 🚀
