# Feature: User Password Display and Management Enhancement

## Summary

Enhanced user management functionality to support password display during user creation and ensure robust password update capabilities. This feature adds a `--show-password` flag to user creation commands, allowing administrators to view generated or provided passwords for secure credential distribution while maintaining security best practices.

## Motivation

Administrators often need to view user passwords immediately after creation for:
- Secure credential distribution to application teams
- Testing and validation of new user accounts
- Integration with external credential management systems
- Automation scenarios where passwords need to be captured

The existing system created users but never displayed passwords for security reasons, requiring administrators to either:
- Pre-generate passwords and track them separately
- Update passwords immediately after creation to known values

## Implementation

### CLI Interface Changes

#### Atlas Users (`matlas atlas users`)

**Create Command Enhancement:**
- Added `--show-password` flag to `matlas atlas users create`
- When flag is present, password is displayed in the output after successful user creation
- Without flag, maintains existing security behavior (password not shown)
- Works with both prompted passwords and provided passwords

**Password Update (Already Supported):**
- `matlas atlas users update <username> --password` prompts for new password
- `matlas atlas users update <username> --password "newpass"` sets specific password

#### Database Users (`matlas database users`)

**Create Command Enhancement:**
- Added `--show-password` flag to `matlas database users create`
- Note: Database user creation via direct MongoDB connection is not yet fully implemented
- Framework prepared for when implementation is completed

**Password Update (Already Supported):**
- `matlas database users update <username> --password "newpass"` updates password

### Output Formatting Changes

**Enhanced CreateResultFormatter:**
- New `FormatCreateResultWithPassword()` method
- Conditionally displays password field when requested
- Shows security warning when password is displayed
- Maintains backward compatibility with existing formatters

**Security Considerations:**
- Password display is opt-in via explicit flag
- Warning message shown when password is displayed
- Passwords excluded from diff comparisons in apply pipeline
- YAML passwords still sourced from environment variables

### YAML ApplyDocument Support

**Existing Password Support (No Changes Needed):**
- `DatabaseUserSpec` and `DatabaseDirectUserSpec` already support password fields
- Password fields properly excluded from diff comparisons for security
- Password field optional for updates to avoid unintended changes
- Environment variable substitution supported for YAML passwords

## Files Modified

### CLI Commands
- `cmd/atlas/users/users.go` - Added `--show-password` flag and updated function signatures
- `cmd/database/users/users.go` - Added `--show-password` flag (for future implementation)

### Output Formatting
- `internal/output/create_formatters.go` - Enhanced with password display capability

### Documentation
- `docs/atlas.md` - Added examples with `--show-password` flag
- `docs/database.md` - Added examples with `--show-password` flag
- `examples/users-with-password-display.yaml` - New comprehensive example

### Feature Tracking
- `features/2025-01-27-password-display-feature.md` - This file

## Usage Examples

### CLI Usage

```bash
# Create Atlas user and display password
matlas atlas users create \
  --project-id <project-id> \
  --username myuser \
  --roles "readWrite@myapp" \
  --show-password

# Update Atlas user password (no display by default)
matlas atlas users update myuser \
  --project-id <project-id> \
  --password "newpassword"

# Database user creation (when implemented)
matlas database users create dbuser \
  --cluster my-cluster \
  --project-id <project-id> \
  --database myapp \
  --password "secure123" \
  --roles "readWrite@myapp" \
  --show-password
```

### YAML Usage

```yaml
apiVersion: matlas.mongodb.com/v1
kind: DatabaseUser
metadata:
  name: api-user
spec:
  projectName: "My Project"
  username: api-user
  password: "${API_USER_PASSWORD}"  # Environment variable
  roles:
    - roleName: readWrite
      databaseName: myapp
```

## Security Model

### Password Display
- **Opt-in only**: Password display requires explicit `--show-password` flag
- **Clear warning**: Users see warning when password is displayed
- **No logging**: Passwords not logged in verbose output or error messages

### Password Storage
- **Environment variables**: YAML passwords sourced from env vars
- **No defaults**: No default passwords generated or stored
- **No persistence**: Display-only feature, no password storage changes

### Password Updates
- **Optional field**: Password field optional in updates to avoid accidents
- **Diff exclusion**: Passwords excluded from diff comparisons
- **Existing user protection**: Discovered users have password fields removed

## Breaking Changes

None. This is a purely additive feature with opt-in behavior.

## Migration Notes

None required. Existing workflows continue to work unchanged.

## Testing Considerations

### Manual Testing
- Verify `--show-password` flag shows password in output
- Verify password warning message appears
- Verify behavior without flag remains unchanged
- Test with both prompted and provided passwords

### Security Testing
- Verify passwords not shown in verbose logs
- Verify passwords excluded from diff output
- Verify environment variable substitution in YAML

## Future Enhancements

1. **Database User Implementation**: Complete the direct MongoDB user creation functionality
2. **Password Generation**: Add automatic secure password generation option
3. **Integration**: Consider integration with external credential management systems
4. **Audit**: Consider password operation audit logging

## Related Issues

This feature addresses user requests for password visibility during automation and credential distribution scenarios while maintaining security best practices established in the existing codebase.
