---
layout: default
title: DAG Analysis Examples
parent: Examples
nav_order: 8
---

# DAG Analysis Examples

Examples showing how to use the DAG (Directed Acyclic Graph) engine for analyzing, visualizing, and optimizing infrastructure deployments.

---

## Basic Analysis

Analyze dependencies and identify the critical path:

```bash
matlas infra analyze -f infrastructure.yaml --project-id <project-id>
```

**Example Output:**

```
Dependency Analysis Report
======================================================================

Generated: 2025-12-09T14:11:24+02:00

OVERVIEW
----------------------------------------------------------------------
Total Operations:      8
Dependencies:          2
Dependency Levels:     2
Has Cycles:            false

CRITICAL PATH
----------------------------------------------------------------------
Length:   2 operations
Duration: 10m30s

Operations on Critical Path:
  1. op-1
  2. op-3

BOTTLENECKS
----------------------------------------------------------------------

1. op-1 (production-cluster)
   Blocks:     1 operations (12.5% impact)
   Reason:     Bottleneck because: [on critical path]

2. op-3 (app-user)
   Blocks:     0 operations (0.0% impact)
   Reason:     Bottleneck because: [on critical path]

RISK ANALYSIS
----------------------------------------------------------------------
Total Risk Score:      62.5
Average Risk Level:    high
High-Risk Operations:  4
Critical-Risk Ops:     2 (on critical path)

Risk Distribution:
  high      : 4 operations
  medium    : 2 operations
  low       : 2 operations

OPTIMIZATION SUGGESTIONS
----------------------------------------------------------------------

1. Low parallelization factor (4.00). Consider reducing dependencies 
   to enable more parallel execution

2. Bottleneck detected: 'production-cluster' blocks 1 operations (12.5% of total)

3. 2 high-risk operations on critical path. Consider moving them earlier 
   (fail-fast) or adding validation steps

4. Critical path is 10m30s. Consider optimizing these operations:
   - production-cluster (10m)
   - app-user (30s)
```

**Key Metrics Explained:**

- **Total Operations**: Number of infrastructure changes to be made
- **Dependency Levels**: Maximum depth of dependency chain (2 = operations can run in 2 sequential stages)
- **Critical Path**: Longest sequence of dependent operations that determines minimum execution time
- **Parallelization Factor**: 4.00x means operations can run 4x faster with sufficient parallelization
- **Risk Score**: 0-100 aggregate risk (higher = more risky)

---

## JSON Output for CI/CD

Export analysis as JSON for programmatic use:

```bash
matlas infra analyze -f infrastructure.yaml \
  --project-id <project-id> \
  --format json \
  --output-file analysis.json
```

**Example JSON Output:**

```json
{
  "nodeCount": 8,
  "edgeCount": 2,
  "hasCycles": false,
  "levels": {
    "op-0": 0,
    "op-1": 0,
    "op-2": 0,
    "op-3": 1,
    "op-4": 0,
    "op-5": 0,
    "op-6": 0,
    "op-7": 0
  },
  "maxLevel": 1,
  "criticalPath": [
    "op-1",
    "op-3"
  ],
  "criticalPathDuration": 630000000000,
  "parallelGroups": [
    [
      {
        "id": "op-1",
        "name": "production-cluster",
        "resourceType": "Cluster",
        "properties": {
          "estimatedDuration": 600000000000,
          "riskLevel": "high",
          "isDestructive": false
        }
      }
    ],
    [
      {
        "id": "op-3",
        "name": "app-user",
        "resourceType": "DatabaseUser",
        "properties": {
          "estimatedDuration": 30000000000,
          "riskLevel": "medium"
        }
      }
    ]
  ],
  "parallelizationFactor": 4.0,
  "bottlenecks": [
    {
      "nodeID": "op-1",
      "nodeName": "production-cluster",
      "blockedNodes": ["op-3"],
      "blockedCount": 1,
      "impact": 0.125,
      "reason": "Bottleneck because: [on critical path]"
    }
  ],
  "riskAnalysis": {
    "totalRiskScore": 62.5,
    "averageRiskLevel": "high",
    "highRiskOperations": 4,
    "criticalRiskOperations": 2
  }
}
```

**CI/CD Integration:**

```bash
# Extract key metrics
RISK_SCORE=$(jq -r '.riskAnalysis.totalRiskScore' analysis.json)
HAS_CYCLES=$(jq -r '.hasCycles' analysis.json)
CRITICAL_DURATION=$(jq -r '.criticalPathDuration' analysis.json)

# Fail build if risk too high
if [ $(echo "$RISK_SCORE > 70" | bc) -eq 1 ]; then
  echo "Risk score too high: $RISK_SCORE"
  exit 1
fi
```

---

## Markdown Report

Generate markdown reports for documentation:

```bash
matlas infra analyze -f infrastructure.yaml \
  --project-id <project-id> \
  --format markdown \
  --output-file DEPLOYMENT_ANALYSIS.md
```

**Example Markdown Output:**

```markdown
# Dependency Analysis Report

**Generated:** 2025-12-09T14:11:24+02:00

## Overview

| Metric | Value |
|--------|-------|
| Total Operations | 8 |
| Dependencies | 2 |
| Dependency Levels | 2 |
| Has Cycles | false |

## Critical Path

**Length:** 2 operations  
**Duration:** 10m30s

### Operations on Critical Path

1. `op-1`
2. `op-3`

## Bottlenecks

### 1. op-1 (production-cluster)

- **Blocks:** 1 operations (12.5% impact)
- **Reason:** Bottleneck because: [on critical path]

## Risk Analysis

**Total Risk Score:** 62.5  
**Average Risk Level:** high  
**High-Risk Operations:** 4  
**Critical-Risk Ops:** 2 (on critical path)

### Risk Distribution

- high: 4 operations
- medium: 2 operations
- low: 2 operations
```

---

## Visualization

### ASCII Visualization

Terminal-friendly dependency graph:

```bash
matlas infra visualize -f infrastructure.yaml --project-id <project-id>
```

**Output:**

```
Dependency Graph (ASCII)
============================================================

Level 0:
  [production-cluster (10m) [high]]
  [network-access-1 (10s) [low]]
  [network-access-2 (10s) [low]]

Level 1:
  [app-user (30s) [medium]]

Statistics:
  Total nodes: 8
  Total edges: 2
  Max level: 1
```

### DOT Format (Graphviz)

Generate visual diagrams:

```bash
matlas infra visualize -f infrastructure.yaml \
  --project-id <project-id> \
  --format dot \
  --output-file deployment.dot

# Render as PNG
dot -Tpng deployment.dot -o deployment.png

# Render as SVG
dot -Tsvg deployment.dot -o deployment.svg
```

### Mermaid Diagram

For Markdown documentation:

```bash
matlas infra visualize -f infrastructure.yaml \
  --project-id <project-id> \
  --format mermaid \
  --output-file deployment.mmd
```

**Example Mermaid Output:**

```mermaid
graph LR
  op_0[production-cluster (10m) [high]]
  op_1[network-access-1 (10s) [low]]
  op_2[network-access-2 (10s) [low]]
  op_3[app-user (30s) [medium]]
  
  op_3 --> op_0
```

Embed this in GitHub/GitLab markdown for automatic rendering.

### Highlight Critical Path

```bash
matlas infra visualize -f infrastructure.yaml \
  --project-id <project-id> \
  --highlight-critical-path
```

**Output:**

```
Dependency Graph (ASCII)
============================================================

Level 0:
  *[production-cluster (10m) [high]]
  [network-access-1 (10s) [low]]
  [network-access-2 (10s) [low]]

Level 1:
  *[app-user (30s) [medium]]

Legend:
  * = Critical path node

Statistics:
  Total nodes: 8
  Total edges: 2
  Max level: 1
```

---

## Optimization Suggestions

Get actionable recommendations:

```bash
matlas infra optimize -f infrastructure.yaml --project-id <project-id>
```

**Example Output:**

```
Optimization Suggestions Report
======================================================================

Generated: 2025-12-09T14:11:26+02:00

HIGH SEVERITY
----------------------------------------------------------------------

1. Critical path is 10m30s (avg per operation: 1m18.75s)
   Type:   long_critical_path
   Impact: Total execution time dominated by critical path
   Action: Optimize operations on critical path or parallelize dependencies

2. 2 high-risk operations on critical path (25.0% of critical path)
   Impact: Deployment likely to fail if these operations fail
   Action: Move high-risk operations earlier (fail-fast) or add validation

MEDIUM SEVERITY
----------------------------------------------------------------------

1. 4 high-risk operations (50.0% of total)
   Impact: Increased failure probability
   Action: Review high-risk operations, add retry logic, or run with risk-based scheduling

2. Bottleneck detected: 'production-cluster' blocks 1 operations (12.5% of total)
   Impact: Single point of failure affects downstream operations
   Action: Add checkpoints or validation before this operation

LOW SEVERITY
----------------------------------------------------------------------

1. 6 operations have slack time > 30s
   Impact: These operations have buffer time for delays
   Action: Consider reordering to optimize resource usage
```

**How to Use Suggestions:**

1. **High Severity**: Address immediately before deploying
   - Optimize cluster creation time (consider smaller instance for testing)
   - Move risky operations earlier (fail-fast strategy)

2. **Medium Severity**: Plan improvements for next iteration
   - Add retry logic to high-risk operations
   - Add validation before bottleneck operations

3. **Low Severity**: Track for continuous improvement
   - Rebalance operations across dependency levels

---

## Complete Workflow Example

### Step 1: Discover Current State

```bash
matlas discover --project-id <project-id> \
  --convert-to-apply \
  --output yaml \
  -o infrastructure.yaml
```

### Step 2: Edit Configuration

Make your infrastructure changes:

```bash
vim infrastructure.yaml
```

### Step 3: Analyze Dependencies

```bash
# Run analysis
matlas infra analyze -f infrastructure.yaml \
  --project-id <project-id> \
  --show-risk

# Export for review
matlas infra analyze -f infrastructure.yaml \
  --project-id <project-id> \
  --format markdown \
  --output-file ANALYSIS.md
```

### Step 4: Visualize Changes

```bash
# Generate diagram
matlas infra visualize -f infrastructure.yaml \
  --project-id <project-id> \
  --format dot \
  --output-file deployment.dot \
  --highlight-critical-path

# Render to PNG
dot -Tpng deployment.dot -o deployment.png
```

### Step 5: Get Optimization Recommendations

```bash
matlas infra optimize -f infrastructure.yaml \
  --project-id <project-id>
```

### Step 6: Preview Changes

```bash
matlas infra diff -f infrastructure.yaml --detailed
```

### Step 7: Apply with Confidence

```bash
# Dry run first
matlas infra apply -f infrastructure.yaml --dry-run

# Apply changes
matlas infra apply -f infrastructure.yaml
```

---

## Real-World Use Cases

### Use Case 1: Major Infrastructure Update

**Scenario:** Upgrading cluster tier and adding new users

```bash
# 1. Analyze before making changes
matlas infra analyze -f upgrade-plan.yaml \
  --project-id prod-123 \
  --format json \
  --output-file pre-upgrade-analysis.json

# Check critical path duration
DURATION=$(jq -r '.criticalPathDuration / 1000000000 / 60' pre-upgrade-analysis.json)
echo "Estimated deployment time: ${DURATION} minutes"

# Check risk score
RISK=$(jq -r '.riskAnalysis.totalRiskScore' pre-upgrade-analysis.json)
if [ $(echo "$RISK > 70" | bc) -eq 1 ]; then
  echo "WARNING: High risk deployment - consider staging first"
fi

# 2. Generate visual for team review
matlas infra visualize -f upgrade-plan.yaml \
  --project-id prod-123 \
  --format dot \
  --output-file upgrade-graph.dot
dot -Tpng upgrade-graph.dot -o upgrade-graph.png

# 3. Apply changes
matlas infra apply -f upgrade-plan.yaml --project-id prod-123
```

### Use Case 2: CI/CD Pipeline Integration

```yaml
# .github/workflows/infrastructure.yml
name: Infrastructure Changes

on:
  pull_request:
    paths:
      - 'infrastructure/**'

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Analyze Dependencies
        run: |
          matlas infra analyze \
            -f infrastructure/production.yaml \
            --project-id ${{ secrets.ATLAS_PROJECT_ID }} \
            --format json \
            --output-file analysis.json
      
      - name: Check Risk Score
        run: |
          RISK_SCORE=$(jq -r '.riskAnalysis.totalRiskScore' analysis.json)
          if [ $(echo "$RISK_SCORE > 70" | bc) -eq 1 ]; then
            echo "::warning::High risk score detected: $RISK_SCORE"
          fi
      
      - name: Generate Visualization
        run: |
          matlas infra visualize \
            -f infrastructure/production.yaml \
            --project-id ${{ secrets.ATLAS_PROJECT_ID }} \
            --format mermaid \
            --output-file graph.mmd
      
      - name: Comment PR
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const graph = fs.readFileSync('graph.mmd', 'utf8');
            const analysis = fs.readFileSync('analysis.json', 'utf8');
            const data = JSON.parse(analysis);
            
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## 📊 Infrastructure Analysis\n\n` +
                    `**Critical Path:** ${data.criticalPathDuration / 1000000000 / 60} minutes\n` +
                    `**Risk Score:** ${data.riskAnalysis.totalRiskScore}/100\n` +
                    `**Bottlenecks:** ${data.bottlenecks.length}\n\n` +
                    `### Dependency Graph\n\`\`\`mermaid\n${graph}\n\`\`\``
            });
```

### Use Case 3: Pre-Deployment Validation

```bash
#!/bin/bash
# pre-deploy-check.sh

CONFIG_FILE="$1"
PROJECT_ID="$2"

echo "🔍 Running pre-deployment checks..."

# Analyze
matlas infra analyze -f "$CONFIG_FILE" \
  --project-id "$PROJECT_ID" \
  --format json \
  --output-file /tmp/analysis.json

# Extract metrics
HAS_CYCLES=$(jq -r '.hasCycles' /tmp/analysis.json)
RISK_SCORE=$(jq -r '.riskAnalysis.totalRiskScore' /tmp/analysis.json)
CRITICAL_OPS=$(jq -r '.riskAnalysis.criticalRiskOperations' /tmp/analysis.json)

# Validate
if [ "$HAS_CYCLES" = "true" ]; then
  echo "❌ FAIL: Circular dependencies detected!"
  exit 1
fi

if [ $(echo "$RISK_SCORE > 80" | bc) -eq 1 ]; then
  echo "⚠️  WARNING: High risk score: $RISK_SCORE"
  echo "   Consider staging this deployment"
fi

if [ "$CRITICAL_OPS" -gt 0 ]; then
  echo "⚠️  WARNING: $CRITICAL_OPS critical-risk operations on critical path"
fi

# Get optimization suggestions
echo ""
echo "💡 Optimization Suggestions:"
matlas infra optimize -f "$CONFIG_FILE" --project-id "$PROJECT_ID"

echo ""
echo "✅ Pre-deployment checks complete"
```

---

## Tips & Best Practices

### 1. Always Analyze Before Major Changes

```bash
# Good workflow
matlas infra analyze -f changes.yaml --project-id <id>
matlas infra apply -f changes.yaml --dry-run
matlas infra apply -f changes.yaml

# Poor workflow
matlas infra apply -f changes.yaml --auto-approve  # Skip analysis
```

### 2. Export Visualizations for Documentation

Keep deployment diagrams in your repository:

```bash
# Generate diagram
matlas infra visualize -f infrastructure.yaml \
  --format mermaid \
  --output-file docs/deployment-diagram.mmd

# Commit to repo
git add docs/deployment-diagram.mmd
git commit -m "docs: Update deployment diagram"
```

### 3. Use Risk Scores for Staging Decisions

```bash
RISK=$(matlas infra analyze -f config.yaml --format json | jq -r '.riskAnalysis.totalRiskScore')

if [ $(echo "$RISK > 70" | bc) -eq 1 ]; then
  # Deploy to staging first
  matlas infra apply -f config.yaml --project-id staging-project
  
  # If successful, promote to production
  matlas infra apply -f config.yaml --project-id production-project
else
  # Low risk - deploy directly
  matlas infra apply -f config.yaml --project-id production-project
fi
```

### 4. Monitor Critical Path Operations

Focus monitoring on operations identified in the critical path:

```bash
# Extract critical path operations
matlas infra analyze -f config.yaml --format json | \
  jq -r '.criticalPath[]' > critical-operations.txt

# Add extra logging/monitoring for these operations
```

### 5. Track Metrics Over Time

```bash
# Save analysis results with timestamp
DATE=$(date +%Y%m%d-%H%M%S)
matlas infra analyze -f config.yaml \
  --format json \
  --output-file "metrics/analysis-$DATE.json"

# Track improvements
echo "Tracking critical path duration over time:"
jq -r '.criticalPathDuration' metrics/analysis-*.json
```

---

## Further Reading

- [DAG Engine Documentation](/dag-engine/) - Complete feature guide
- [Infrastructure Workflows](/infra/) - Plan, diff, apply workflows
- [Discovery Documentation](/discovery/) - Enumerating Atlas resources
