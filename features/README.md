## Feature tracking

Create one Markdown file per feature under this directory to briefly document what was accomplished.

### Naming convention
- Use: `YYYY-MM-DD-<short-slug>.md` (e.g., `2025-08-13-temporary-users.md`)
- Alternatively, if tracking against an issue or PR, use: `FTR-<id>-<short-slug>.md`

### Minimum required content
- A top-level title: `Feature: <name>`
- A concise "Summary" section describing what was achieved (2–6 sentences)

### Recommended sections (align with our Feature Interface Consistency rule)
- CLI surfaces: new/changed commands and flags
- YAML ApplyDocument: kinds/fields added or changed
- Service layer: packages/functions touched in `internal/services/*`
- Apply pipeline: loader/validation/plan/diff/apply areas impacted
- Types/models: changes under `internal/types/*`
- Tests: unit/integration/e2e added (paths)
- Docs/examples: docs pages and example YAML updated/added
- Breaking/migration notes: if any

### Quick start
Copy `TEMPLATE.md` to a new file and fill it in:

```bash
cp features/TEMPLATE.md features/2025-08-13-your-feature.md
```

Keep it concise and high-signal. Link to code, docs, and examples where useful.


