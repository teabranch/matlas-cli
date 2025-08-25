---
layout: default
title: Network Access
parent: Examples
nav_order: 5
description: IP allowlisting and network security configurations
---

# Network Access Examples

Network access rule configurations for IP allowlisting, CIDR blocks, and AWS security group integration.

## Basic Network Access

Simple IP address and CIDR block configurations:

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: network-access-basic
resources:
  # Single IP address access
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: developer-workstation
    spec:
      projectName: "My Project"
      ipAddress: "203.0.113.42"
      comment: "Developer workstation access"

  # CIDR block for office network
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: office-network
    spec:
      projectName: "My Project"
      cidr: "203.0.113.0/24"
      comment: "Corporate office network"
```
{% endraw %}

## Network Access Variants

Multiple NetworkAccess types including temporary access and AWS security groups:

{% raw %}
```yaml
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: network-variants
  labels:
    purpose: comprehensive-network-access
resources:
  # CIDR block access
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: office-cidr
      labels:
        type: cidr
        environment: office
    spec:
      projectName: "My Project"
      cidr: "192.168.1.0/24"
      comment: "Office network CIDR block"

  # Temporary IP access with expiration
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: temporary-contractor
      labels:
        type: temporary-ip
        purpose: contractor-access
    spec:
      projectName: "My Project"
      ipAddress: "203.0.113.100"
      comment: "Temporary contractor access"
      deleteAfterDate: "2024-12-31T23:59:59Z"

  # AWS Security Group access
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: aws-production-sg
      labels:
        type: aws-security-group
        environment: production
    spec:
      projectName: "My Project"
      awsSecurityGroup: "sg-1234567890abcdef0"
      comment: "Production AWS security group"

  # Multiple IP addresses for team members
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: team-lead-home
    spec:
      projectName: "My Project"
      ipAddress: "203.0.113.50"
      comment: "Team lead home office"

  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: senior-dev-home
    spec:
      projectName: "My Project"
      ipAddress: "203.0.113.51"
      comment: "Senior developer home office"

  # VPN endpoint access
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: corporate-vpn
      labels:
        type: vpn-endpoint
    spec:
      projectName: "My Project"
      cidr: "10.0.0.0/8"
      comment: "Corporate VPN network range"
```
{% endraw %}

## Usage Examples

### Basic Network Rules

```bash
# Apply basic network access rules
matlas infra validate -f network-access-basic.yaml
matlas infra apply -f network-access-basic.yaml

# Apply comprehensive network configuration
matlas infra apply -f network-variants.yaml --preserve-existing
```

### CLI Network Management

```bash
# Add IP address access
matlas atlas network create \
  --project-id <project-id> \
  --ip-address "203.0.113.42" \
  --comment "Developer access"

# Add CIDR block access
matlas atlas network create \
  --project-id <project-id> \
  --cidr "192.168.1.0/24" \
  --comment "Office network"

# Add AWS security group
matlas atlas network create \
  --project-id <project-id> \
  --aws-security-group "sg-1234567890abcdef0" \
  --comment "Production security group"

# List network access rules
matlas atlas network list --project-id <project-id> --output table

# Delete network access rule
matlas atlas network delete <rule-id> --project-id <project-id>
```

## Network Access Types

### IP Address Access
```yaml
spec:
  ipAddress: "203.0.113.42"
  comment: "Single IP address"
```

### CIDR Block Access
```yaml
spec:
  cidr: "203.0.113.0/24"
  comment: "Network range"
```

### AWS Security Group
```yaml
spec:
  awsSecurityGroup: "sg-1234567890abcdef0"
  comment: "AWS security group"
```

### Temporary Access
```yaml
spec:
  ipAddress: "203.0.113.42"
  comment: "Temporary access"
  deleteAfterDate: "2024-12-31T23:59:59Z"  # ISO 8601 format
```

## Security Best Practices

### Use Specific CIDR Blocks
Avoid overly broad network ranges:

```yaml
# ✅ Specific office network
cidr: "192.168.1.0/24"

# ⚠️ Too broad
cidr: "0.0.0.0/0"  # Allows all traffic
```

### Implement Temporary Access
Use expiration dates for temporary access:

```yaml
spec:
  ipAddress: "203.0.113.100"
  deleteAfterDate: "2024-12-31T23:59:59Z"  # ✅ Auto-expires
```

### Use Descriptive Comments
Always include meaningful comments:

```yaml
comment: "Production VPC security group"  # ✅ Clear purpose
comment: "temp access"                    # ❌ Too vague
```

### AWS Security Groups
For AWS deployments, prefer security groups over IP addresses:

```yaml
# ✅ Security group (dynamic IP handling)
awsSecurityGroup: "sg-1234567890abcdef0"

# ⚠️ Static IP (may change)
ipAddress: "54.123.45.67"
```

## Common Patterns

### Development Environment
```yaml
# Development team access
- ipAddress: "203.0.113.50"
  comment: "Lead developer home"
- ipAddress: "203.0.113.51"  
  comment: "Senior developer home"
- cidr: "192.168.1.0/24"
  comment: "Office development network"
```

### Production Environment
```yaml
# Production access via security groups
- awsSecurityGroup: "sg-prod-web-tier"
  comment: "Production web servers"
- awsSecurityGroup: "sg-prod-app-tier"
  comment: "Production application servers"
```

### Contractor/Temporary Access
```yaml
# Time-limited contractor access
- ipAddress: "203.0.113.200"
  comment: "External contractor - Project X"
  deleteAfterDate: "2024-06-30T23:59:59Z"
```

## Troubleshooting

### Connection Issues
1. **Verify IP address**: Use `curl ifconfig.me` to check your current IP
2. **Check CIDR notation**: Ensure proper subnet mask (e.g., `/24` not `/255.255.255.0`)
3. **AWS security group**: Verify security group exists and has proper outbound rules

### Access Denied
1. **Rule propagation**: Network access rules may take 1-2 minutes to become active
2. **Multiple rules**: Check if IP is covered by existing CIDR blocks
3. **Expired rules**: Verify `deleteAfterDate` hasn't passed

## Related Examples

- [Clusters]({{ '/examples/clusters/' | relative_url }}) - Cluster configurations needing network access
- [Infrastructure Patterns]({{ '/examples/infrastructure/' | relative_url }}) - Complete infrastructure with network security
- [VPC Endpoints]({{ '/examples/advanced/' | relative_url }}) - Private network connectivity