## [Unreleased]

### Fixed
- Fixed version command to display proper semantic versions instead of 'dev' or branch names
- Fixed build process to use Git-based version detection that works locally and in CI/CD
- Fixed installation script URL generation bug where info messages were captured in version string causing malformed download URLs
- Fixed installation script artifact naming to use underscores (matlas_darwin_arm64.tar.gz) instead of hyphens to match actual release artifacts
- Fixed installation script cleanup trap to handle unbound variables with set -u option

### Added
- **Comprehensive Backup Features**: Complete backup and Point-in-Time Recovery implementation
- Point-in-Time Recovery (PIT) support with proper validation workflow
- CLI backup management (`--backup` and `--pit` flags for cluster create/update)
- YAML backup configuration (`backupEnabled` and `pitEnabled` fields)
- Cross-field validation ensuring PIT requires backup to be enabled
- Backup workflow validation preventing PIT during cluster creation
- Comprehensive backup test suite with CLI and YAML validation testing
- Cross-region backup support via multi-region cluster configurations
- **MongoDB Atlas Alerting System**: Complete alert configuration and management system
- Alert configuration CLI commands (`matlas atlas alert-configurations list/get/delete/matcher-fields`)
- Alert management CLI commands (`matlas atlas alerts list/get/acknowledge`)
- AlertConfiguration and Alert YAML kinds for ApplyDocument support
- 11 notification channel types: EMAIL, SMS, SLACK, PAGER_DUTY, OPS_GENIE, DATADOG, MICROSOFT_TEAMS, WEBHOOK, USER, GROUP, TEAM
- 8 alert matcher operators: EQUALS, NOT_EQUALS, CONTAINS, NOT_CONTAINS, STARTS_WITH, ENDS_WITH, REGEX, NOT_REGEX
- Metric-based and general threshold configurations with AVERAGE/TOTAL modes
- Comprehensive alert validation with field and type checking
- Alert configuration examples for basic, comprehensive, notification channels, and threshold patterns
- Alert lifecycle test suite with proper cleanup and baseline integrity verification
- Advanced Atlas Search features support with CLI and YAML interfaces

### Changed
- Removed Go module caching from CI pipeline to resolve "Cannot open: File exists" errors on main branch builds
- Search Index Analyzers configuration and management
- Faceted Search functionality for string, number, and date facets
- Autocomplete functionality with fuzzy matching support
- Search Highlighting configuration for result highlighting
- Search Synonyms management for improved search relevance
- Fuzzy Search configuration with configurable edit distance
- Search metrics and performance analytics CLI commands (`matlas atlas search metrics`)
- Index optimization analysis and recommendations CLI commands (`matlas atlas search optimize`)
- Search query validation and testing CLI commands (`matlas atlas search validate-query`)
- SearchMetrics, SearchOptimization, and SearchQueryValidation YAML kinds for ApplyDocument support
- Comprehensive time range support for metrics (1h, 6h, 24h, 7d, 30d)
- Advanced query validation with test mode and performance analysis
- Comprehensive test suite for advanced search features with `--preserve-existing` flag
- Advanced search examples and documentation

### Changed
- Extended SearchIndex YAML kind to support advanced search features
- Enhanced search service with advanced functionality
- Updated documentation with advanced search configuration options

### Fixed
- **Search CLI Commands**: Removed advanced search CLI commands (analyzers, facets, autocomplete, highlighting, synonyms, fuzzy, metrics, optimize, validate) that returned placeholder data due to Atlas Admin API limitations
- Updated search command unit test to validate 5 basic subcommands (list, get, create, update, delete)

### Removed
- **Advanced Search CLI Commands**: Removed misleading CLI commands for advanced search features
  - These features are properly supported via YAML configuration through the apply pipeline
  - Atlas Admin API embeds advanced features within search index definitions, not as separate resources
  - YAML-based configuration remains fully functional and is the recommended approach

## [1.0.4](https://github.com/teabranch/matlas-cli/compare/v1.0.3...v1.0.4) (2025-08-17)

### Bug Fixes

* database documentation ([6ff416d](https://github.com/teabranch/matlas-cli/commit/6ff416dcce14c56a96d3f98bc80efbe22dd29a7e))

## [1.0.3](https://github.com/teabranch/matlas-cli/compare/v1.0.2...v1.0.3) (2025-08-17)

### Bug Fixes

* release workflow ([3a19040](https://github.com/teabranch/matlas-cli/commit/3a19040e02f577c225981487310ef3b0321178d9))
* release workflow ([378429b](https://github.com/teabranch/matlas-cli/commit/378429bfa5cd19d5199e9134f284488dd18cfb1c))
* release workflow ([c67325d](https://github.com/teabranch/matlas-cli/commit/c67325d252546969f25b66d8e4d93678d911264e))

## [1.0.3](https://github.com/teabranch/matlas-cli/compare/v1.0.2...v1.0.3) (2025-08-17)

### Bug Fixes

* release workflow ([378429b](https://github.com/teabranch/matlas-cli/commit/378429bfa5cd19d5199e9134f284488dd18cfb1c))
* release workflow ([c67325d](https://github.com/teabranch/matlas-cli/commit/c67325d252546969f25b66d8e4d93678d911264e))

## [Unreleased]

### Fixed
- **Test Script Execution Issues**: Fixed critical test script issues preventing proper execution with environment variables
  - Fixed cluster-lifecycle.sh by removing invalid `--preserve-existing` flag from `infra destroy` command (flag only supported by `infra apply`)
  - Fixed database-operations.sh unbound variable errors using proper `${1:-all}` parameter expansion
  - Enhanced database operations script with user existence validation before attempting username/password authentication
  - Updated documentation strings to clarify actual safety mechanisms (resources only managed if defined in YAML configurations)
  - Both scripts now run successfully when sourced with environment variables from project root directory

### Added
- New `atlas search` command group for Atlas Search index management
  - `matlas atlas search list` - List search indexes in cluster or collection
  - `matlas atlas search create` - Create new search indexes (basic implementation)
  - Support for both full-text search and vector search indexes
- New YAML kinds: `SearchIndex` and `VPCEndpoint` for ApplyDocument support
- Comprehensive examples for search and VPC endpoint configurations
- Multi-cloud provider support for VPC endpoints (AWS, Azure, GCP)

### Fixed
- **VPC Endpoints YAML Project ID Resolution**: Fixed critical project ID parsing error where VPCEndpoint YAML configurations weren't being processed for project resolution, causing all YAML operations to fail with "project '' not found in organization" error
- **VPC Endpoints Multi-Provider Deletion**: Fixed cloud provider mismatch where all deletion operations were hardcoded to use AWS provider, causing GCP and Azure endpoints to fail deletion and creating resource leaks
- **VPC Endpoints Test Verification Logic**: Fixed verification logic that searched for non-existent YAML metadata names in Atlas API responses, replacing with actual endpoint count and cloud provider validation
- Enhanced project ID parsing support for DatabaseUser, NetworkAccess, and Cluster resources in YAML configurations
- Improved VPC endpoints test infrastructure with proper timing mechanisms for Atlas backend processing delays
- Added robust resource cleanup procedures with dynamic cloud provider extraction

### Changed
- Updated documentation with Atlas Search and VPC endpoint examples
- Added support for Atlas Search APIs in SDK v20250312006.1.0
- VPC endpoints testing infrastructure now uses Atlas API-compatible verification methods
- Enhanced test scripts with comprehensive multi-provider cleanup and verification logic

### Fixed
- Updated release process documentation to accurately reflect the consolidated workflow implementation
- Fixed semantic-release not detecting conventional commits by consolidating workflows
- Completely redesigned release process following current best practices
- Consolidated CI/CD, testing, building, and releasing into single workflow
- Semantic-release now directly includes artifacts in GitHub releases with proper filenames
- Fixed artifact labels to show actual filenames (e.g., 'matlas_linux_amd64.zip') instead of generic descriptions
- Eliminated workflow coordination issues that caused empty releases
- Fixed semantic-release workflow creating confusing chore commits that broke artifact attachment
- Removed @semantic-release/git and @semantic-release/changelog plugins to eliminate post-release commits
- Fixed CI workflow running twice (on push and release), causing empty releases without artifacts
- Enhanced semantic-release to verify artifacts exist before creating releases
- Release workflow now consistently finds CI artifacts for the correct commit SHA
- Fixed critical issue where CI artifacts weren't created for manual tags
- Fixed timing issue where releases were created before CI artifacts were ready
- Release workflow now properly finds and downloads CI artifacts for any commit
- Eliminated confusing dual-branch pre-release strategy

### Changed
- Simplified release strategy to use single main branch with semantic versioning
- Semantic-release now waits for CI workflow to complete before creating releases
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
