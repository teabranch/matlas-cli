---
layout: default
title: Alerts
nav_order: 9
has_children: false
description: MongoDB Atlas alert management and monitoring configuration
permalink: /alerts/
---

# MongoDB Atlas Alerts

Comprehensive alert management and monitoring configuration for MongoDB Atlas resources.

{: .note }
Atlas alerts provide real-time monitoring and notifications for your MongoDB infrastructure. The CLI supports both alert management (viewing/acknowledging active alerts) and alert configuration management (creating/managing alert rules).

## Overview

MongoDB Atlas alerts help you monitor your database infrastructure and respond quickly to issues. The matlas CLI provides:

- **Alert Management**: View, acknowledge, and monitor active alerts
- **Alert Configuration**: Create, update, and delete alert rules
- **Multi-Channel Notifications**: Email, Slack, PagerDuty, webhooks, and more
- **Advanced Targeting**: Precise resource targeting with matchers
- **YAML Configuration**: Infrastructure-as-code alert management

## Quick Start

### List Active Alerts

```bash
# List all alerts in a project
matlas atlas alerts list --project-id <project-id>

# Get specific alert details
matlas atlas alerts get <alert-id> --project-id <project-id>

# Acknowledge an alert
matlas atlas alerts acknowledge <alert-id> --project-id <project-id>
```

### Manage Alert Configurations

```bash
# List alert configurations
matlas atlas alert-configurations list --project-id <project-id>

# Get configuration details
matlas atlas alert-configurations get <config-id> --project-id <project-id>

# Delete a configuration
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
  metricThreshold:
    metricName: "CPU_USAGE_PERCENT"
    operator: "GREATER_THAN"
    threshold: 80.0
    units: "PERCENT"
    mode: "AVERAGE"
```

## Alert Management

### Viewing Alerts

**List all alerts in a project:**
```bash
matlas atlas alerts list --project-id <project-id>
```

**Pagination support:**
```bash
matlas atlas alerts list --project-id <project-id> --page 2 --limit 10
```

**JSON output for automation:**
```bash
matlas atlas alerts list --project-id <project-id> --output json
```

### Alert Details

**Get comprehensive alert information:**
```bash
matlas atlas alerts get <alert-id> --project-id <project-id>
```

Alert details include:
- Alert ID and status
- Event type and severity
- Current metric values
- Acknowledgment status
- Configuration details
- Trigger conditions

### Acknowledging Alerts

**Acknowledge an alert:**
```bash
matlas atlas alerts acknowledge <alert-id> --project-id <project-id>
```

**Unacknowledge an alert:**
```bash
matlas atlas alerts acknowledge <alert-id> --project-id <project-id> --unacknowledge
```

Acknowledgment helps track which alerts have been reviewed and are being addressed.

## Alert Configuration Management

### Listing Configurations

**List all alert configurations:**
```bash
matlas atlas alert-configurations list --project-id <project-id>
```

**With pagination:**
```bash
matlas atlas alert-configurations list --project-id <project-id> --page 2 --limit 10
```

### Configuration Details

**Get detailed configuration information:**
```bash
matlas atlas alert-configurations get <config-id> --project-id <project-id>
```

Configuration details include:
- Event type and conditions
- Matcher rules
- Notification channels
- Threshold settings
- Enabled status

### Deleting Configurations

**Delete an alert configuration:**
```bash
matlas atlas alert-configurations delete <config-id> --project-id <project-id>
```

**Skip confirmation prompt:**
```bash
matlas atlas alert-configurations delete <config-id> --project-id <project-id> --yes
```

### Matcher Fields

**List available matcher field names:**
```bash
matlas atlas alert-configurations matcher-fields
```

This command shows all available field names that can be used in alert matchers for targeting specific resources.

## YAML Configuration

### Basic Alert Configuration

```yaml
apiVersion: matlas.mongodb.com/v1
kind: AlertConfiguration
metadata:
  name: basic-cpu-alert
  labels:
    severity: high
    category: performance
spec:
  enabled: true
  eventTypeName: "HOST_CPU_USAGE_PERCENT"
  severityOverride: "HIGH"
  
  # Target specific resources
  matchers:
    - fieldName: "HOSTNAME_AND_PORT"
      operator: "CONTAINS"
      value: "production"
  
  # Email notification
  notifications:
    - typeName: "EMAIL"
      emailAddress: "alerts@company.com"
      delayMin: 0
      intervalMin: 15
  
  # Threshold configuration
  metricThreshold:
    metricName: "CPU_USAGE_PERCENT"
    operator: "GREATER_THAN"
    threshold: 80.0
    units: "PERCENT"
    mode: "AVERAGE"
```

### Multi-Resource Documents

Combine alerts with infrastructure:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: infrastructure-with-monitoring
resources:
  # Infrastructure
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: production-cluster
    spec:
      # ... cluster configuration
  
  # Monitoring
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: cluster-monitoring
    spec:
      enabled: true
      eventTypeName: "HOST_CPU_USAGE_PERCENT"
      # ... alert configuration
```

### Apply Alert Configurations

```bash
# Validate configuration
matlas infra validate -f alerts.yaml

# Preview changes
matlas infra plan -f alerts.yaml

# Apply configuration
matlas infra apply -f alerts.yaml --preserve-existing
```

## Event Types

### Performance Monitoring

Monitor resource usage and performance:

- **HOST_CPU_USAGE_PERCENT** - CPU usage monitoring
- **HOST_MEMORY_USAGE_PERCENT** - Memory usage monitoring
- **HOST_DISK_USAGE_PERCENT** - Disk usage monitoring
- **DATABASE_CONNECTIONS_PERCENT** - Connection usage monitoring
- **REPLICATION_LAG** - Replication lag monitoring

### Cluster Health

Monitor cluster status and availability:

- **CLUSTER_MONGOS_IS_MISSING** - Missing mongos process
- **CLUSTER_PRIMARY_ELECTED** - Primary election events
- **CLUSTER_DISK_USAGE_PERCENT** - Cluster-wide disk usage
- **CLUSTER_OPLOG_WINDOW_RUNNING_OUT** - Oplog window issues

### Database Operations

Monitor database-level activities:

- **DATABASE_OPERATIONS_TOTAL** - Total database operations
- **DATABASE_QUERY_TARGETING_SCANNED_PER_RETURNED** - Query efficiency
- **DATABASE_CURSORS_TOTAL_OPEN** - Open cursor count

## Notification Channels

### Email Notifications

Simple email alerts:

```yaml
notifications:
  - typeName: "EMAIL"
    emailAddress: "alerts@company.com"
    delayMin: 0        # Immediate notification
    intervalMin: 15    # Repeat every 15 minutes
```

### Slack Integration

Slack channel notifications:

```yaml
notifications:
  - typeName: "SLACK"
    apiToken: "${SLACK_TOKEN}"
    channelName: "#alerts"
    username: "Atlas Monitor"
    delayMin: 0
    intervalMin: 5
```

### PagerDuty Integration

Critical incident management:

```yaml
notifications:
  - typeName: "PAGER_DUTY"
    serviceKey: "${PAGERDUTY_SERVICE_KEY}"
    delayMin: 0
    intervalMin: 0     # No repeat (PagerDuty handles escalation)
```

### Webhook Notifications

Custom HTTP integrations:

```yaml
notifications:
  - typeName: "WEBHOOK"
    url: "https://api.company.com/webhooks/atlas"
    secret: "${WEBHOOK_SECRET}"
    delayMin: 0
    intervalMin: 10
```

### Microsoft Teams

Teams channel notifications:

```yaml
notifications:
  - typeName: "MICROSOFT_TEAMS"
    microsoftTeamsWebhookUrl: "${TEAMS_WEBHOOK_URL}"
    delayMin: 0
    intervalMin: 15
```

### Additional Channels

- **SMS** - Mobile phone text messages
- **OPS_GENIE** - OpsGenie alert management
- **DATADOG** - Datadog monitoring integration
- **USER** - Atlas user notifications
- **GROUP** - Project group notifications
- **TEAM** - Atlas team notifications

## Matchers and Targeting

### Matcher Operators

Target specific resources with precise matching:

- **EQUALS** / **NOT_EQUALS** - Exact string matching
- **CONTAINS** / **NOT_CONTAINS** - Substring matching
- **STARTS_WITH** / **ENDS_WITH** - Prefix/suffix matching
- **REGEX** / **NOT_REGEX** - Regular expression matching

### Common Matcher Fields

- **HOSTNAME_AND_PORT** - Target specific hosts
- **REPLICA_SET_NAME** - Target replica sets
- **CLUSTER_NAME** - Target clusters
- **DATABASE_NAME** - Target databases
- **TYPE_NAME** - Target node types (PRIMARY, SECONDARY)

### Matcher Examples

**Target production hosts:**
```yaml
matchers:
  - fieldName: "HOSTNAME_AND_PORT"
    operator: "CONTAINS"
    value: "production"
```

**Target specific replica sets:**
```yaml
matchers:
  - fieldName: "REPLICA_SET_NAME"
    operator: "REGEX"
    value: "atlas-.*-shard-[0-9]+"
```

**Exclude staging environments:**
```yaml
matchers:
  - fieldName: "CLUSTER_NAME"
    operator: "NOT_CONTAINS"
    value: "staging"
```

**Multiple matchers (AND logic):**
```yaml
matchers:
  - fieldName: "HOSTNAME_AND_PORT"
    operator: "CONTAINS"
    value: "production"
  - fieldName: "TYPE_NAME"
    operator: "EQUALS"
    value: "SECONDARY"
```

## Thresholds

### Metric Thresholds

For performance metrics with units:

```yaml
metricThreshold:
  metricName: "CPU_USAGE_PERCENT"
  operator: "GREATER_THAN"
  threshold: 80.0
  units: "PERCENT"
  mode: "AVERAGE"      # AVERAGE or TOTAL
```

**Threshold operators:**
- **GREATER_THAN** - Trigger when metric exceeds threshold
- **LESS_THAN** - Trigger when metric falls below threshold

**Threshold modes:**
- **AVERAGE** - Use average metric value over time window
- **TOTAL** - Use total/sum metric value over time window

### General Thresholds

For non-metric events:

```yaml
threshold:
  operator: "GREATER_THAN"
  threshold: 10
  units: "SECONDS"
```

## Best Practices

### Alert Design

1. **Start Simple**: Begin with basic CPU, memory, and disk alerts
2. **Use Appropriate Thresholds**: Set realistic thresholds based on normal usage
3. **Avoid Alert Fatigue**: Don't create too many low-priority alerts
4. **Test Notifications**: Verify notification channels work correctly

### Notification Strategy

1. **Escalation Patterns**: Use different delays for different severity levels
2. **Channel Selection**: Match notification channels to alert severity
3. **Avoid Spam**: Use appropriate intervals to prevent notification flooding
4. **Environment Variables**: Store sensitive tokens in environment variables

### Matcher Strategy

1. **Precise Targeting**: Use specific matchers to avoid false positives
2. **Environment Separation**: Use different matchers for different environments
3. **Regex Patterns**: Use regex for complex matching requirements
4. **Multiple Matchers**: Combine matchers for precise resource targeting

### YAML Organization

1. **ApplyDocument**: Use ApplyDocument for related alerts and infrastructure
2. **Labels**: Use consistent labeling for alert categorization
3. **Comments**: Document complex matcher and threshold logic
4. **Environment Variables**: Externalize sensitive configuration

## Troubleshooting

### Common Issues

**Alert not triggering:**
- Verify matcher conditions match actual resource names
- Check threshold values against actual metrics
- Ensure alert configuration is enabled
- Verify event type name is correct

**Notifications not working:**
- Verify notification channel credentials
- Check API tokens and webhook URLs
- Verify channel names and email addresses
- Test notification channels independently

**Matcher not matching:**
- Use `matlas atlas alert-configurations matcher-fields` to see available fields
- Test matcher patterns with actual resource names
- Use regex testing tools for complex patterns
- Check case sensitivity in matcher values

### Debugging Commands

```bash
# List all alerts to see current status
matlas atlas alerts list --project-id <project-id> --output json

# Get alert configuration details
matlas atlas alert-configurations get <config-id> --project-id <project-id>

# List available matcher fields
matlas atlas alert-configurations matcher-fields

# Validate YAML configuration
matlas infra validate -f alerts.yaml

# Preview alert changes
matlas infra plan -f alerts.yaml
```

## Examples

See the [Alert Examples]({{ '/examples/alerts/' | relative_url }}) for comprehensive examples including:

- Basic CPU and memory monitoring
- Multi-channel notification setups
- Complex matcher configurations
- Comprehensive monitoring stacks
- Environment-specific patterns

## Related Documentation

- [Alert Examples]({{ '/examples/alerts/' | relative_url }}) - Working YAML examples
- [Atlas Commands]({{ '/atlas/#alerts' | relative_url }}) - CLI command reference
- [YAML Kinds Reference]({{ '/yaml-kinds/#alertconfiguration' | relative_url }}) - Complete AlertConfiguration reference
- [Infrastructure Commands]({{ '/infra/' | relative_url }}) - Apply and manage configurations

---

For additional help and advanced configurations, see the [MongoDB Atlas documentation](https://docs.atlas.mongodb.com/reference/api/alerts/) for the underlying Atlas Alert API.
