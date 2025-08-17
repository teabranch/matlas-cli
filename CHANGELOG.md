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
