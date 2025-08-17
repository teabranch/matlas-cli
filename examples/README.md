# Examples

Working YAML examples for `ApplyDocument` resources used by `matlas infra`.

## Files

- `users-basic.yaml`: Single `DatabaseUser` with a basic read role on `admin`.
- `users-standalone-multiple.yaml`: Two standalone users with different roles and labels.
- `overlay-network-and-user.yaml`: Adds a user and a `NetworkAccess` IP entry (overlay style).
- `custom-roles-and-users.yaml`: Defines a `DatabaseRole` and a user that uses it.
- `cluster-basic.yaml`: Minimal `Cluster` definition.
- `cluster-advanced.yaml`: Cluster with autoscaling and replication spec.
- `project-with-cluster-and-users.yaml`: Cluster plus an app user in one document.
- `custom-roles-example.yaml`: Comprehensive custom roles example including users.
 - `users-scoped.yaml`: Users scoped to specific clusters via `scopes`.
 - `network-variants.yaml`: `NetworkAccess` examples for `cidr`, `ipAddress` with `deleteAfterDate`, and `awsSecurityGroup`.
 - `dependencies-and-deletion.yaml`: Demonstrates `dependsOn` and `deletionPolicy` between resources.
 - `cluster-multiregion.yaml`: Multi-region cluster using `replicationSpecs` and `regionConfigs`.
 - `cluster-security-and-tags.yaml`: Cluster with `encryption`, `biConnector`, and `tags`.
 - `project-format.yaml`: Project-format configuration consumable by infra commands.

## Usage

Replace placeholders like "My Project" and provide environment variables for passwords before running:

```bash
export APP_USER_PASSWORD='StrongPass123!'
export APP_WRITER_PASSWORD='StrongPass123!'
export ANALYTICS_PASSWORD='StrongPass123!'
export OVERLAY_USER_PASSWORD='StrongPass123!'
export ROLE_USER_PASSWORD='StrongPass123!'
```

Validate/diff/apply:

```bash
matlas infra validate -f examples/users-basic.yaml
matlas infra diff -f examples/custom-roles-and-users.yaml
matlas infra apply -f examples/overlay-network-and-user.yaml --auto-approve
```

These examples mirror structures used in the test scripts under `scripts/test/` and adhere to the types in `internal/types/`.
