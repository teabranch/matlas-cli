## [Unreleased]

### Fixed
- Fixed critical issue where CI artifacts weren't created for manual tags
- Release workflow now properly finds and downloads CI artifacts for any commit
- Eliminated confusing dual-branch pre-release strategy

### Changed
- Simplified release strategy to use single main branch with semantic versioning
- CI workflow now creates checksums for all builds, not just specific branches
- Release workflow has better error handling and fallback for artifact download
- Updated documentation to reflect simplified release process

## [1.0.3-main.1](https://github.com/teabranch/matlas-cli/compare/v1.0.2...v1.0.3-main.1) (2025-08-17)

### Bug Fixes

* release workflow ([c67325d](https://github.com/teabranch/matlas-cli/commit/c67325d252546969f25b66d8e4d93678d911264e))

## [1.0.2](https://github.com/teabranch/matlas-cli/compare/v1.0.1...v1.0.2) (2025-08-17)

### Bug Fixes

* go.mod ([be3b546](https://github.com/teabranch/matlas-cli/commit/be3b546849aa4875156fa149a01e4dc401b0a8f9))

## [1.0.1](https://github.com/teabranch/matlas-cli/compare/v1.0.0...v1.0.1) (2025-08-17)

### Bug Fixes

* lint and 1 test ([3fe0357](https://github.com/teabranch/matlas-cli/commit/3fe03575c256679389edb5b614cc5a91c0be22a2))

## 1.0.0 (2025-08-17)

### ⚠ BREAKING CHANGES

* **cli:** config template generate and export commands now use --file instead of --output flag

## Error Handling Fixes
- Fix unchecked MarkHidden() error returns in database command (cmd/database/database.go)
- Fix unchecked MongoDB client.Disconnect() error returns in roles command (cmd/database/roles/roles.go)
- Fix unchecked MongoDB client.Disconnect() error returns in users command (cmd/database/users/users.go)
- Add proper error handling with panic for flag configuration failures
- Add warning messages for MongoDB connection cleanup failures

## CLI Improvements
- Standardize config command flags: rename --output/-o to --file for consistency
- Update template generate command examples and help text
- Update export command examples and help text
- Improve flag naming consistency across config subcommands

## Testing Infrastructure
- Add comprehensive config command test suite (scripts/test/config-test.sh)
- Test template generation, validation, experimental commands, error handling
- Add config tests to main test runner (scripts/test.sh)
- Include config tests in comprehensive test suite

## Documentation
- Update config.md to reflect --file flag usage
- Update command help text and examples

## Project Tracking
- Document all bug fixes and improvements in tracking/bugfixes.md
- Update test script tracking documentation

Resolves GoLinting errcheck violations and improves CLI consistency.

Refs: #errcheck-fixes #config-standardization

### Bug Fixes

* **cli:** resolve errcheck violations and standardize config command flags ([ce7924b](https://github.com/teabranch/matlas-cli/commit/ce7924b45a0f1b248376d1921d089eac38bae2fc))

### Documentation

* comprehensive documentation updates for discovery and authentication features ([7bd2a13](https://github.com/teabranch/matlas-cli/commit/7bd2a131eaeb1a4c1e98128b2cb7c186a17c2e2f))

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Password display flag `--show-password` for user creation commands to optionally show passwords after user creation
- Enhanced output formatting for user creation with optional password display and security warnings
- Comprehensive discovery feature testing suite with integration tests and lifecycle workflows
- Discovery command integration tests for CLI functionality validation
- Shell-based discovery lifecycle tests for end-to-end workflow testing
- Discovery test integration with main test runner and comprehensive test suite

### Fixed
- Fixed MongoDB Atlas authorization error by removing unsupported `userAdminAnyDatabase` role and updating temporary users to use only Atlas-supported roles (`readWriteAnyDatabase`, `dbAdminAnyDatabase`)
- Disabled direct database user management commands as MongoDB Atlas requires using Atlas API/UI for user management operations
- Updated test scripts to remove tests for unsupported database user operations and focus on supported functionality (custom roles, database operations, Atlas user management)

### Added
- Automated release workflow with semantic versioning
- Multi-platform binary distribution (Linux, macOS, Windows)
- Docker image distribution via GitHub Container Registry
- Build information injection (version, commit, build time)
- Database roles management commands: `database roles list|get|create|delete` for custom MongoDB roles
- Temporary user support for database operations via `--use-temp-user`, with `--role` and advanced `--temp-user-roles`
- Force-confirmation flag `--yes` for destructive operations in `database delete` and `database roles delete`
- Documentation for database workflows and custom roles under `docs/database.md`
- New command aliases: `db` for `database`; `ls` for `list`; `del|rm|remove` for `delete`
- Enable `DatabaseRole` kind in YAML validation and planning; apply remains unsupported

### Changed
- Enhanced version command with detailed build information
- Updated CI workflow to support release automation
- Improved error messages with actionable hints for authentication, network, timeout, and HTTP failures
- Docker image now includes OCI metadata labels for richer container metadata

### Removed

- Homebrew tap instructions; releases are published on GitHub only for now
- GoReleaser configuration and references; using custom build workflows instead

---

*Note: This changelog will be automatically updated by semantic-release based on conventional commits.*
