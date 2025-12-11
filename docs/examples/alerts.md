---
layout: default
title: Alert Examples
parent: Examples
nav_order: 7
description: MongoDB Atlas alert configuration examples for monitoring and notifications
permalink: /examples/alerts/
---

# Alert Examples

MongoDB Atlas alert configurations for comprehensive monitoring and multi-channel notifications.

{: .note }
All alert examples use `ApplyDocument` for proper dependency management and validation. Alert configurations can be combined with cluster and user resources in the same document.

## Quick Reference

| Example | Description | Features |
|---------|-------------|----------|
| [alert-basic.yaml](https://github.com/teabranch/matlas-cli/blob/main/examples/alert-basic.yaml) | Simple CPU monitoring | Email notifications, basic threshold |
| [alert-comprehensive.yaml](https://github.com/teabranch/matlas-cli/blob/main/examples/alert-comprehensive.yaml) | Multi-metric monitoring | Multiple alerts, various thresholds |
| [alert-notification-channels.yaml](https://github.com/teabranch/matlas-cli/blob/main/examples/alert-notification-channels.yaml) | All notification types | Email, Slack, PagerDuty, webhooks |
| [alert-thresholds-and-matchers.yaml](https://github.com/teabranch/matlas-cli/blob/main/examples/alert-thresholds-and-matchers.yaml) | Advanced targeting | Complex matchers, multiple thresholds |

## Basic CPU Alert

Simple CPU usage monitoring with email notifications:

```yaml
# examples/alert-basic.yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: basic-alert-config
  labels:
    environment: production
    team: platform
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: high-cpu-alert
      labels:
        severity: high
        category: performance
    spec:
      enabled: true
      eventTypeName: "HOST_CPU_USAGE_PERCENT"
      severityOverride: "HIGH"
      
      # Target production cluster hosts
      matchers:
        - fieldName: "HOSTNAME_AND_PORT"
          operator: "CONTAINS"
          value: "cluster0"
      
      # Email notification
      notifications:
        - typeName: "EMAIL"
          emailAddress: "alerts@company.com"
          delayMin: 0
          intervalMin: 15
      
      # Trigger when CPU > 80%
      metricThreshold:
        metricName: "CPU_USAGE_PERCENT"
        operator: "GREATER_THAN"
        threshold: 80.0
        units: "PERCENT"
        mode: "AVERAGE"
```

## Multi-Channel Notifications

Comprehensive alert with multiple notification channels:

```yaml
# examples/alert-notification-channels.yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: multi-channel-alerts
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: critical-system-alert
    spec:
      enabled: true
      eventTypeName: "HOST_MEMORY_USAGE_PERCENT"
      severityOverride: "CRITICAL"
      
      matchers:
        - fieldName: "HOSTNAME_AND_PORT"
          operator: "REGEX"
          value: ".*production.*"
      
      notifications:
        # Email notification
        - typeName: "EMAIL"
          emailAddress: "alerts@company.com"
          delayMin: 0
          intervalMin: 5
        
        # Slack notification
        - typeName: "SLACK"
          apiToken: "${SLACK_TOKEN}"
          channelName: "#alerts"
          username: "Atlas Monitor"
          delayMin: 0
          intervalMin: 5
        
        # PagerDuty for critical issues
        - typeName: "PAGER_DUTY"
          serviceKey: "${PAGERDUTY_SERVICE_KEY}"
          delayMin: 0
          intervalMin: 0
        
        # Webhook for custom integrations
        - typeName: "WEBHOOK"
          url: "https://api.company.com/webhooks/atlas"
          secret: "${WEBHOOK_SECRET}"
          delayMin: 0
          intervalMin: 10
      
      metricThreshold:
        metricName: "MEMORY_USAGE_PERCENT"
        operator: "GREATER_THAN"
        threshold: 90.0
        units: "PERCENT"
        mode: "AVERAGE"
```

## Comprehensive Monitoring Setup

Multiple alerts for complete infrastructure monitoring:

```yaml
# examples/alert-comprehensive.yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: comprehensive-monitoring
  labels:
    environment: production
resources:
  # CPU Usage Alert
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: cpu-usage-alert
    spec:
      enabled: true
      eventTypeName: "HOST_CPU_USAGE_PERCENT"
      severityOverride: "HIGH"
      matchers:
        - fieldName: "HOSTNAME_AND_PORT"
          operator: "CONTAINS"
          value: "production"
      notifications:
        - typeName: "EMAIL"
          emailAddress: "ops@company.com"
          intervalMin: 15
      metricThreshold:
        metricName: "CPU_USAGE_PERCENT"
        operator: "GREATER_THAN"
        threshold: 75.0
        units: "PERCENT"
        mode: "AVERAGE"

  # Memory Usage Alert
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: memory-usage-alert
    spec:
      enabled: true
      eventTypeName: "HOST_MEMORY_USAGE_PERCENT"
      severityOverride: "HIGH"
      matchers:
        - fieldName: "HOSTNAME_AND_PORT"
          operator: "CONTAINS"
          value: "production"
      notifications:
        - typeName: "SLACK"
          apiToken: "${SLACK_TOKEN}"
          channelName: "#infrastructure"
      metricThreshold:
        metricName: "MEMORY_USAGE_PERCENT"
        operator: "GREATER_THAN"
        threshold: 85.0
        units: "PERCENT"
        mode: "AVERAGE"

  # Disk Usage Alert
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: disk-usage-alert
    spec:
      enabled: true
      eventTypeName: "HOST_DISK_USAGE_PERCENT"
      severityOverride: "CRITICAL"
      matchers:
        - fieldName: "HOSTNAME_AND_PORT"
          operator: "CONTAINS"
          value: "production"
      notifications:
        - typeName: "PAGER_DUTY"
          serviceKey: "${PAGERDUTY_SERVICE_KEY}"
      metricThreshold:
        metricName: "DISK_USAGE_PERCENT"
        operator: "GREATER_THAN"
        threshold: 90.0
        units: "PERCENT"
        mode: "AVERAGE"

  # Connection Usage Alert
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: connection-usage-alert
    spec:
      enabled: true
      eventTypeName: "DATABASE_CONNECTIONS_PERCENT"
      severityOverride: "MEDIUM"
      matchers:
        - fieldName: "REPLICA_SET_NAME"
          operator: "STARTS_WITH"
          value: "atlas-"
      notifications:
        - typeName: "EMAIL"
          emailAddress: "dba@company.com"
          intervalMin: 30
      metricThreshold:
        metricName: "CONNECTIONS_PERCENT"
        operator: "GREATER_THAN"
        threshold: 80.0
        units: "PERCENT"
        mode: "AVERAGE"
```

## Advanced Matchers and Thresholds

Complex targeting and threshold configurations:

```yaml
# examples/alert-thresholds-and-matchers.yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: advanced-alert-patterns
resources:
  # Multi-matcher alert targeting specific shards
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: shard-specific-alert
    spec:
      enabled: true
      eventTypeName: "REPLICATION_LAG"
      severityOverride: "HIGH"
      
      # Multiple matchers for precise targeting
      matchers:
        - fieldName: "REPLICA_SET_NAME"
          operator: "REGEX"
          value: "atlas-.*-shard-[0-9]+"
        - fieldName: "HOSTNAME_AND_PORT"
          operator: "NOT_CONTAINS"
          value: "staging"
        - fieldName: "TYPE_NAME"
          operator: "EQUALS"
          value: "SECONDARY"
      
      notifications:
        - typeName: "SLACK"
          apiToken: "${SLACK_TOKEN}"
          channelName: "#database-alerts"
      
      # General threshold for non-metric events
      threshold:
        operator: "GREATER_THAN"
        threshold: 10
        units: "SECONDS"

  # Cluster-level alert with specific targeting
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: cluster-health-alert
    spec:
      enabled: true
      eventTypeName: "CLUSTER_MONGOS_IS_MISSING"
      severityOverride: "CRITICAL"
      
      matchers:
        - fieldName: "CLUSTER_NAME"
          operator: "ENDS_WITH"
          value: "-production"
      
      notifications:
        - typeName: "PAGER_DUTY"
          serviceKey: "${PAGERDUTY_SERVICE_KEY}"
        - typeName: "EMAIL"
          emailAddress: "oncall@company.com"
          delayMin: 0
          intervalMin: 0

  # Custom metric threshold with TOTAL mode
  - apiVersion: matlas.mongodb.com/v1
    kind: AlertConfiguration
    metadata:
      name: total-operations-alert
    spec:
      enabled: true
      eventTypeName: "DATABASE_OPERATIONS_TOTAL"
      severityOverride: "MEDIUM"
      
      matchers:
        - fieldName: "DATABASE_NAME"
          operator: "NOT_EQUALS"
          value: "admin"
      
      notifications:
        - typeName: "WEBHOOK"
          url: "https://monitoring.company.com/atlas-webhook"
          secret: "${MONITORING_WEBHOOK_SECRET}"
      
      metricThreshold:
        metricName: "OPERATIONS_TOTAL"
        operator: "GREATER_THAN"
        threshold: 10000.0
        units: "RAW"
        mode: "TOTAL"  # Sum instead of average
```

## Usage Examples

### Environment Variables

Set up notification credentials:

```bash
# Slack integration
export SLACK_TOKEN='xoxb-your-slack-bot-token'

# PagerDuty integration
export PAGERDUTY_SERVICE_KEY='your-pagerduty-integration-key'

# Webhook secrets
export WEBHOOK_SECRET='your-webhook-secret-key'
export MONITORING_WEBHOOK_SECRET='your-monitoring-secret'
```

### Apply Alert Configurations

```bash
# Validate alert configuration
matlas infra validate -f examples/alert-basic.yaml

# Preview alert changes
matlas infra plan -f examples/alert-comprehensive.yaml

# Apply basic CPU alert
matlas infra apply -f examples/alert-basic.yaml --auto-approve

# Apply comprehensive monitoring (safe mode)
matlas infra apply -f examples/alert-comprehensive.yaml --preserve-existing

# Apply with specific project context
ATLAS_PROJECT_ID=507f1f77bcf86cd799439011 \
  matlas infra apply -f examples/alert-notification-channels.yaml
```

### Monitor Alert Status

```bash
# List all alerts in project
matlas atlas alerts list --project-id <project-id>

# Get specific alert details
matlas atlas alerts get <alert-id> --project-id <project-id>

# Acknowledge an alert
matlas atlas alerts acknowledge <alert-id> --project-id <project-id>

# List alert configurations
matlas atlas alert-configurations list --project-id <project-id>

# Get available matcher field names
matlas atlas alert-configurations matcher-fields
```

## Common Patterns

### Production Monitoring Stack

Combine alerts with infrastructure:

```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: production-stack-with-monitoring
resources:
  # Infrastructure
  - kind: Cluster
    metadata:
      name: production-cluster
    spec:
      # ... cluster configuration
  
  # Monitoring
  - kind: AlertConfiguration
    metadata:
      name: cluster-cpu-alert
    spec:
      enabled: true
      eventTypeName: "HOST_CPU_USAGE_PERCENT"
      # ... alert configuration
```

### Environment-Specific Alerts

Use labels and matchers for environment targeting:

```yaml
matchers:
  - fieldName: "HOSTNAME_AND_PORT"
    operator: "CONTAINS"
    value: "{{ .Values.environment }}"  # production, staging, dev
```

### Escalation Patterns

Progressive notification delays:

```yaml
notifications:
  # Immediate Slack notification
  - typeName: "SLACK"
    channelName: "#alerts"
    delayMin: 0
    intervalMin: 5
  
  # Email after 5 minutes
  - typeName: "EMAIL"
    emailAddress: "team@company.com"
    delayMin: 5
    intervalMin: 15
  
  # PagerDuty after 15 minutes
  - typeName: "PAGER_DUTY"
    serviceKey: "${PAGERDUTY_SERVICE_KEY}"
    delayMin: 15
    intervalMin: 0
```

## Related Documentation

- [Alert CLI Commands]({{ '/atlas/#alerts' | relative_url }}) - Command-line alert management
- [YAML Kinds Reference]({{ '/reference/#alertconfiguration' | relative_url }}) - Complete AlertConfiguration reference
- [Infrastructure Commands]({{ '/infra/' | relative_url }}) - Apply and manage alert configurations
- [Atlas Documentation]({{ '/atlas/' | relative_url }}) - Atlas resource management

---

For complete source files, see the [examples directory](https://github.com/teabranch/matlas-cli/tree/main/examples) in the GitHub repository.
