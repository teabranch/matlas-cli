# Feature: MongoDB Atlas Alerting System

**Status**: ✅ **COMPLETED**  
**Impact**: High - Critical monitoring and notification capabilities  
**Complexity**: High - Full CRUD operations, multiple notification channels, validation  

## Summary

Implemented comprehensive alerting system for MongoDB Atlas with full CLI and YAML support. The feature provides complete alert configuration management, alert monitoring, and multi-channel notifications through both command-line interface and declarative YAML configurations.

## Implementation Details

### Core Components

#### 1. Type System (`internal/types/`)
- **AlertConfig**: Complete alert configuration structure with matchers, notifications, thresholds
- **AlertMatcher**: Rule-based targeting with 8 operator types (EQUALS, CONTAINS, REGEX, etc.)
- **AlertNotification**: 11 notification channel types (EMAIL, SLACK, PAGERDUTY, WEBHOOK, etc.)
- **AlertMetricThreshold**: Metric-based thresholds with AVERAGE/TOTAL modes
- **AlertThreshold**: General thresholds for non-metric alerts
- **AlertStatus**: Read-only alert status information
- **AlertCurrentValue**: Current metric values that triggered alerts
- **AlertConfigurationManifest**: YAML manifest for alert configurations
- **AlertManifest**: YAML manifest for alert status (read-only)

#### 2. Service Layer (`internal/services/atlas/`)
- **AlertsService**: Alert operations (list, get, acknowledge, list by configuration)
- **AlertConfigurationsService**: Full CRUD operations for alert configurations
- **Conversion Methods**: Bidirectional conversion between internal types and Atlas SDK types
- **Matcher Field Names**: Dynamic field name discovery for alert matchers

#### 3. CLI Commands (`cmd/atlas/alerts/`)
- **alerts.go**: Alert management commands
  - `matlas atlas alerts list` - List all alerts in project
  - `matlas atlas alerts get <id>` - Get specific alert details
  - `matlas atlas alerts acknowledge <id>` - Acknowledge/unacknowledge alerts
- **alert_configurations.go**: Alert configuration management
  - `matlas atlas alert-configurations list` - List all alert configurations
  - `matlas atlas alert-configurations get <id>` - Get configuration details
  - `matlas atlas alert-configurations delete <id>` - Delete configuration
  - `matlas atlas alert-configurations matcher-fields` - List available matcher fields

#### 4. YAML Support (`internal/apply/`)
- **Validation**: Comprehensive validation for AlertConfiguration and Alert kinds
- **Type Conversion**: Map-to-struct conversion for YAML parsing
- **ApplyDocument Integration**: Full support in multi-resource documents
- **Schema Validation**: Field validation, enum checking, range validation

### Notification Channels Supported

1. **EMAIL** - Direct email notifications
2. **SMS** - Mobile phone text messages
3. **SLACK** - Slack channel notifications
4. **PAGER_DUTY** - PagerDuty service integration
5. **OPS_GENIE** - OpsGenie alert management
6. **DATADOG** - Datadog monitoring integration
7. **MICROSOFT_TEAMS** - Microsoft Teams webhook
8. **WEBHOOK** - Custom HTTP webhooks
9. **USER** - Atlas user notifications
10. **GROUP** - Project group notifications
11. **TEAM** - Atlas team notifications

### Alert Matcher Operators

- **EQUALS** / **NOT_EQUALS** - Exact string matching
- **CONTAINS** / **NOT_CONTAINS** - Substring matching
- **STARTS_WITH** / **ENDS_WITH** - Prefix/suffix matching
- **REGEX** / **NOT_REGEX** - Regular expression matching

### Threshold Types

- **MetricThreshold**: Metric-specific thresholds with units and modes
- **GeneralThreshold**: Simple numeric thresholds for non-metric events

## Files Modified/Created

### Core Implementation
- `internal/types/apply.go` - Added KindAlert and KindAlertConfiguration
- `internal/types/config.go` - Added alert configuration types
- `internal/services/atlas/alerts.go` - Alert operations service
- `internal/services/atlas/alert_configurations.go` - Alert configuration service
- `internal/apply/validation.go` - Alert validation functions

### CLI Commands
- `cmd/atlas/alerts/alerts.go` - Alert CLI commands
- `cmd/atlas/alerts/alert_configurations.go` - Alert configuration CLI commands
- `cmd/atlas/atlas.go` - Added alert commands to main atlas command

### Examples and Documentation
- `examples/alert-basic.yaml` - Simple CPU alert example
- `examples/alert-comprehensive.yaml` - Multi-channel, complex configurations
- `examples/alert-notification-channels.yaml` - All notification types demo
- `examples/alert-thresholds-and-matchers.yaml` - Threshold and matcher examples

### Testing
- `scripts/test/alerts-lifecycle.sh` - Comprehensive test script with cleanup

### Feature Tracking
- `features/2025-01-28-alerting-system.md` - This feature documentation

## Usage Examples

### CLI Usage
```bash
# List all alerts
matlas atlas alerts list --project-id <project-id>

# Get specific alert
matlas atlas alerts get <alert-id> --project-id <project-id>

# Acknowledge alert
matlas atlas alerts acknowledge <alert-id> --project-id <project-id>

# List alert configurations
matlas atlas alert-configurations list --project-id <project-id>

# Delete alert configuration
matlas atlas alert-configurations delete <config-id> --project-id <project-id>

# List available matcher fields
matlas atlas alert-configurations matcher-fields
```

### YAML Configuration
```yaml
apiVersion: matlas.mongodb.com/v1
kind: AlertConfiguration
metadata:
  name: high-cpu-alert
spec:
  enabled: true
  eventTypeName: "HOST_CPU_USAGE_PERCENT"
  matchers:
    - fieldName: "HOSTNAME_AND_PORT"
      operator: "CONTAINS"
      value: "production"
  notifications:
    - typeName: "EMAIL"
      emailAddress: "alerts@company.com"
    - typeName: "SLACK"
      apiToken: "${SLACK_TOKEN}"
      channelName: "#alerts"
  metricThreshold:
    metricName: "CPU_USAGE_PERCENT"
    operator: "GREATER_THAN"
    threshold: 80.0
    units: "PERCENT"
    mode: "AVERAGE"
```

## Testing Coverage

- ✅ Alert configuration CRUD operations
- ✅ Alert listing and acknowledgment
- ✅ All notification channel types
- ✅ Threshold and matcher configurations
- ✅ Error handling and validation
- ✅ YAML validation and parsing
- ✅ Baseline integrity verification
- ✅ Output format support (table, JSON, YAML)

## Integration Points

- **Atlas SDK**: Uses `AlertConfigurationsApi` and `AlertsApi` from Atlas Go SDK
- **Apply System**: Full integration with `matlas infra apply` command
- **Validation System**: Comprehensive field and type validation
- **CLI Framework**: Consistent with existing command patterns
- **Output System**: Supports all standard output formats

## Breaking Changes

None - This is a new feature addition.

## Migration Notes

No migration required - new functionality only.

## Future Enhancements

- Alert template system for common configurations
- Bulk alert configuration operations
- Alert configuration import/export
- Advanced alert analytics and reporting
- Integration with external monitoring systems

## Dependencies

- Atlas Go SDK v20250312006 or later
- Existing CLI framework and validation system
- YAML processing infrastructure

## Security Considerations

- API keys and tokens properly handled through environment variables
- Webhook URLs validated for security
- Sensitive notification credentials masked in output
- Proper authentication required for all operations

## Performance Impact

- Minimal - Alert operations are lightweight API calls
- Efficient pagination support for large alert lists
- Optimized conversion between internal and Atlas types
- Proper error handling to avoid unnecessary retries
