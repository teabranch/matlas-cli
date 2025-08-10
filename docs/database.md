Database commands

Connection options
- Direct: `--connection-string "mongodb+srv://user:pass@host/"`
- Via Atlas: `--cluster <name> --project-id <id>` (optionally `--use-temp-user` and `--database <db>`)

Databases
- List: `matlas database list [--connection-string ... | --cluster <name> --project-id <id> [--use-temp-user] [--database <db>]]`
- Create: `matlas database create <db> [--connection-string ... | --cluster <name> --project-id <id>]`
- Delete: `matlas database delete <db> [--connection-string ... | --cluster <name> --project-id <id>] [--yes]`

Collections
- List: `matlas database collections list [--connection-string ... | --cluster <name> --project-id <id>] --database <db>`
- Create: `matlas database collections create <collection> [--connection-string ... | --cluster <name> --project-id <id>] --database <db> [--capped --size BYTES --max-documents N]`
- Delete: `matlas database collections delete <collection> [--connection-string ... | --cluster <name> --project-id <id>] --database <db> [--yes]`

Indexes
- List: `matlas database collections indexes list [--connection-string ... | --cluster <name> --project-id <id>] --database <db> --collection <coll>`
- Create: `matlas database collections indexes create <field:order> [...] [--connection-string ... | --cluster <name> --project-id <id>] --database <db> --collection <coll> [--name NAME] [--unique] [--sparse] [--background]`
- Delete: `matlas database collections indexes delete <name> [--connection-string ... | --cluster <name> --project-id <id>] --database <db> --collection <coll> [--yes]`

