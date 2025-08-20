# Feature: SearchIndex Resource Support

## Summary
Added end-to-end support for Atlas SearchIndex resources, including both full-text and vector search indexes, across CLI commands and YAML ApplyDocument workflows. This feature integrates discovery, diff, validation, and apply pipeline stages to manage SearchIndex resources seamlessly, and introduces non-interactive automation for tests.

## CLI surfaces
- `atlas search create` / `atlas search delete`: Manage search indexes via CLI flags:
  - `--project-id`, `--cluster`, `--database`, `--collection`, `--name`, `--type`
- `infra apply` / `infra destroy`: Apply YAML `SearchIndex` resources with `--auto-approve` to skip prompts.

## YAML ApplyDocument
- Kinds/fields added:
  - Kind: `SearchIndex`
  - Spec fields: `projectName`, `clusterName`, `databaseName`, `collectionName`, `indexName`, `indexType`, `definition`
- Validation/diff/apply behavior notes:
  - `convertSearchIndexToManifest` maps Atlas API responses to manifests.
  - `convertToSearchIndexSpec` and `convertSearchDefinitionToSDK` handle spec conversion for text and vector indexes.

## Service layer
- Packages/functions involved:
  - `internal/services/atlas/search.go`: CRUD operations (`ListAllIndexes`, `CreateSearchIndex`, `UpdateSearchIndex`, `DeleteSearchIndex`).

## Apply pipeline
- Areas touched:
  - `internal/apply/fetchers.go`: `convertSearchIndexToManifest`
  - `internal/apply/cache.go`: `DiscoverSearchIndexes`
  - `internal/apply/executor.go` & `internal/apply/enhanced_executor.go`: Executor support for SearchIndex operations
  - `cmd/infra/apply.go` & `cmd/infra/destroy.go`: Integration points for SearchIndex in apply/destroy commands

## Types/models
- Updated types:
  - `types.SearchIndexManifest`
  - `types.SearchIndexSpec`

## Tests
- Unit: Tests in `internal/apply/executor_test.go` (added mocks for SearchService)
- CLI/E2E: `scripts/test/search-lifecycle.sh` lifecycle tests (basic, vector, multi-resource)

## Docs & examples
- Docs updated: `docs/yaml-kinds.md` (added `SearchIndex` kind)
- Examples added/updated: `examples/search-basic.yaml`, `examples/search-vector.yaml`, `examples/search-multi.yaml`

## Breaking changes / migration
- None. Feature is backward-compatible and additive.

## Links
- PR(s): 
- Issue(s):
