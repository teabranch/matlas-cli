# DAG Engine - Advanced Dependency Management

## Overview

The DAG (Directed Acyclic Graph) engine provides sophisticated dependency management, scheduling optimization, and execution planning for infrastructure operations. It uses algorithms from operations research to optimize parallel execution and identify critical paths.

## Edge Semantics

**IMPORTANT**: Understanding edge direction is critical:

```
Edge(From: A, To: B) means "A depends on B"
```

- **Edges[A]**: Contains edges showing what A depends on (A's prerequisites)
- **ReverseEdges[B]**: Contains edges showing what depends on B (B's dependents)

### Example
```go
// node2 depends on node1 (node1 must complete before node2)
graph.AddEdge(&Edge{From: "node2", To: "node1"})

// Execution order: node1 → node2

// Edges["node2"] = [edge to node1]  // node2's dependencies
// ReverseEdges["node1"] = [edge from node2]  // node1's dependents
```

### Execution Flow
1. Nodes with no dependencies (Edges[node] is empty) execute first
2. As nodes complete, dependents (ReverseEdges[completed]) become eligible
3. Topological sort returns execution order respecting dependencies

## Core Components

### 1. types.go
Defines core data structures:
- **Node**: Operation with properties (duration, risk, resources)
- **Edge**: Dependency relationship with type, weight, conditions
- **Graph**: Complete DAG with forward/reverse edges
- **DependencyType**: Hard, Soft, Conditional, Mutual Exclusion, etc.
- **Analysis types**: Results, bottlenecks, risk analysis

### 2. graph.go
Graph operations and management:
- CRUD operations for nodes and edges
- Graph validation and cycle detection
- Cloning, serialization (JSON)
- Utility functions (reachability, paths, levels)

### 3. algorithms.go
Advanced graph algorithms:
- **TopologicalSort**: Kahn's algorithm for execution order
- **CriticalPathMethod**: Forward/backward pass for schedule optimization
- **LongestPath**: Find critical bottlenecks
- **StronglyConnectedComponents**: Tarjan's algorithm
- **TransitiveClosure/Reduction**: Dependency optimization

### 4. analyzer.go
Dependency analysis and insights:
- Bottleneck detection with impact analysis
- Risk analysis (high-risk ops on critical path)
- What-if scenario modeling
- Optimization suggestions
- Parallelization metrics

## Key Algorithms

### Critical Path Method (CPM)
Identifies the longest path through dependencies:

**Forward Pass**: Compute earliest start times
```
For each node in topological order:
    ES[node] = max(ES[dep] + duration[dep]) for all dependencies
```

**Backward Pass**: Compute latest start times
```
For each node in reverse topological order:
    LS[node] = min(LS[dependent] - duration[node]) for all dependents
```

**Critical Path**: Nodes where `Slack = LS - ES = 0`

### Topological Sort (Kahn's Algorithm)
Returns execution order:
1. Calculate in-degree (number of dependencies) for each node
2. Start with nodes having in-degree 0 (no dependencies)
3. Process nodes, decrementing in-degree of dependents
4. Add dependents with in-degree 0 to queue

## Usage Examples

### Basic Graph Creation
```go
import "github.com/teabranch/matlas-cli/internal/apply/dag"

// Create graph
graph := dag.NewGraph(dag.GraphMetadata{
    Name: "infrastructure-deployment",
    ProjectID: "project-123",
})

// Add nodes
cluster := &dag.Node{
    ID: "cluster-1",
    Name: "Production Cluster",
    ResourceType: types.KindCluster,
    Properties: dag.NodeProperties{
        EstimatedDuration: 20 * time.Minute,
        RiskLevel: dag.RiskLevelMedium,
    },
}
graph.AddNode(cluster)

user := &dag.Node{
    ID: "user-1",
    Name: "Database User",
    ResourceType: types.KindDatabaseUser,
    Properties: dag.NodeProperties{
        EstimatedDuration: 2 * time.Minute,
        RiskLevel: dag.RiskLevelLow,
    },
}
graph.AddNode(user)

// Add dependency: user depends on cluster
graph.AddEdge(&dag.Edge{
    From: "user-1",
    To: "cluster-1",
    Type: dag.DependencyTypeHard,
    Reason: "User requires cluster to exist",
})
```

### Analysis
```go
analyzer := dag.NewAnalyzer(graph)
analysis, err := analyzer.Analyze()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Critical path: %v\n", analysis.CriticalPath)
fmt.Printf("Duration: %v\n", analysis.CriticalPathDuration)
fmt.Printf("Parallelization factor: %.2f\n", analysis.ParallelizationFactor)

for _, bottleneck := range analysis.Bottlenecks {
    fmt.Printf("Bottleneck: %s blocks %d operations\n", 
        bottleneck.NodeName, bottleneck.BlockedCount)
}
```

### What-If Analysis
```go
// Simulate adding a new operation
scenario := &dag.WhatIfScenario{
    Name: "Add backup configuration",
    AddNodes: []*dag.Node{newBackupNode},
    AddEdges: []*dag.Edge{backupDependsOnCluster},
}

result, err := analyzer.WhatIfAnalysis(scenario)
fmt.Printf("Duration change: %v\n", result.DurationChange)
fmt.Printf("Parallelism change: %.2f\n", result.ParallelismChange)
```

## Dependency Types

### Hard Dependencies
Must complete before dependent can start. Used for required relationships.

### Soft Dependencies
Preferred order but not required. Used for optimization hints.

### Conditional Dependencies
Depends on runtime conditions or resource properties.

### Mutual Exclusion
Cannot run in parallel (e.g., modifying same resource).

### Ordering Constraints
Relative ordering without strict blocking dependencies.

### Resource Dependencies
Limited by resource availability (API quotas, rate limits).

## Performance Characteristics

### Time Complexity
- TopologicalSort: O(V + E)
- CriticalPathMethod: O(V + E)
- Cycle Detection: O(V + E)
- TransitiveClosure: O(V³)
- Bottleneck Detection: O(V * (V + E))

### Space Complexity
- Graph storage: O(V + E)
- Analysis results: O(V + E)

## Testing

### Running Tests
```bash
# Unit tests
go test ./internal/apply/dag/...

# With race detection
go test ./internal/apply/dag/... -race

# With coverage
go test ./internal/apply/dag/... -cover

# Verbose output
go test ./internal/apply/dag/... -v
```

### Test Organization
- **dag_test.go**: Core functionality tests (graph, algorithms, analysis)
- **security_test.go**: Security and concurrency tests

### Common Testing Patterns

#### Thread Safety
All public methods use RWMutex locking:
- Write operations (Add, Remove, Update): `mu.Lock()`
- Read operations (Get, List, Analyze): `mu.RLock()`
- Internal methods (called while holding lock): No additional locking

**Critical**: Never call a locking method from within another locking method to avoid deadlock.

#### Example: Avoiding Deadlock
```go
// WRONG - causes deadlock
func (g *Graph) ComputeParallelGroups() ([][]*Node, error) {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // BAD: GetNodesByLevel() tries to acquire RLock while we hold Lock
    return g.GetNodesByLevel(), nil
}

// CORRECT - inline the logic
func (g *Graph) ComputeParallelGroups() ([][]*Node, error) {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // Inline level grouping - no additional locking needed
    levels := make(map[int][]*Node)
    for _, node := range g.Nodes {
        levels[node.Level] = append(levels[node.Level], node)
    }
    return levels, nil
}
```

### Known Issues and Fixes

#### Fixed: Deadlock in ComputeParallelGroups (v3.0.3)
**Issue**: Calling `GetNodesByLevel()` while holding write lock caused deadlock.
**Fix**: Inlined level grouping logic to avoid nested locking.

#### Fixed: Concurrent Modifications Test (v3.0.3)
**Issue**: Test incorrectly expected no cycles, but circular edge pattern created cycles by design.
**Fix**: Changed test to check for data corruption (forward/reverse edge consistency) instead of cycles.

## Best Practices

### 1. Always Validate Before Analysis
```go
if err := graph.Validate(); err != nil {
    return fmt.Errorf("invalid graph: %w", err)
}
```

### 2. Handle Cycles Gracefully
```go
if hasCycle, cycle := graph.HasCycle(); hasCycle {
    return fmt.Errorf("cycle detected: %v", cycle)
}
```

### 3. Use Internal Methods When Holding Lock
```go
func (g *Graph) PublicMethod() error {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // Use internal methods that don't acquire locks
    return g.internalMethodNoLock()
}
```

### 4. Clone for Concurrent Operations
```go
// Clone graph for analysis while original is being modified
analysisGraph := graph.Clone()
go analyzer.Analyze(analysisGraph)
```

### 5. Estimate Durations Realistically
```go
props := NodeProperties{
    EstimatedDuration: 10 * time.Minute,  // Based on historical data
    MinDuration: 5 * time.Minute,         // Best case
    MaxDuration: 20 * time.Minute,        // Worst case
}
```

Where:
- V = number of nodes (operations)
- E = number of edges (dependencies)

## Testing

Run tests with coverage:
```bash
go test ./internal/apply/dag/... -v -cover
```

Run with race detector:
```bash
go test ./internal/apply/dag/... -race
```

## Integration

### With Existing DependencyResolver
The new DAG engine is designed to work alongside the existing `DependencyResolver` in `internal/apply/dependencies.go`. The plugin-based rule system (Phase 2) will allow migrating existing rules incrementally.

### With Plan Execution
The DAG engine produces optimized schedules that the executor can use for parallel operation execution with proper dependency ordering.

## Future Enhancements

- **Phase 2**: Plugin-based dependency rules
- **Phase 3**: Intelligent scheduling strategies
- **Phase 4**: Multi-format visualization
- **Phase 5**: State management and checkpointing
- **Phase 6**: Comprehensive documentation

## References

- Kahn's Algorithm: [Topological Sorting](https://en.wikipedia.org/wiki/Topological_sorting)
- Critical Path Method: [CPM](https://en.wikipedia.org/wiki/Critical_path_method)
- Tarjan's SCC: [Strongly Connected Components](https://en.wikipedia.org/wiki/Tarjan%27s_strongly_connected_components_algorithm)
