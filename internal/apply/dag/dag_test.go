package dag

import (
	"fmt"
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestNewGraph(t *testing.T) {
	metadata := GraphMetadata{
		Name:      "test-graph",
		ProjectID: "test-project",
	}

	graph := NewGraph(metadata)

	if graph == nil {
		t.Fatal("NewGraph returned nil")
	}

	if graph.Metadata.Name != "test-graph" {
		t.Errorf("Expected name 'test-graph', got '%s'", graph.Metadata.Name)
	}

	if graph.NodeCount() != 0 {
		t.Errorf("Expected 0 nodes, got %d", graph.NodeCount())
	}
}

func TestAddNode(t *testing.T) {
	graph := NewGraph(GraphMetadata{})

	node := &Node{
		ID:           "node1",
		Name:         "Test Node",
		ResourceType: types.KindCluster,
		Properties: NodeProperties{
			EstimatedDuration: 5 * time.Minute,
			RiskLevel:         RiskLevelLow,
		},
	}

	err := graph.AddNode(node)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	if graph.NodeCount() != 1 {
		t.Errorf("Expected 1 node, got %d", graph.NodeCount())
	}

	// Try adding duplicate
	err = graph.AddNode(node)
	if err == nil {
		t.Error("Expected error when adding duplicate node")
	}
}

func TestAddEdge(t *testing.T) {
	graph := NewGraph(GraphMetadata{})

	node1 := &Node{ID: "node1", Name: "Node 1", ResourceType: types.KindCluster}
	node2 := &Node{ID: "node2", Name: "Node 2", ResourceType: types.KindDatabaseUser}

	_ = graph.AddNode(node1)
	_ = graph.AddNode(node2)

	edge := &Edge{
		From:   "node1",
		To:     "node2",
		Type:   DependencyTypeHard,
		Weight: 1.0,
		Reason: "Database user depends on cluster",
	}

	err := graph.AddEdge(edge)
	if err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	if graph.EdgeCount() != 1 {
		t.Errorf("Expected 1 edge, got %d", graph.EdgeCount())
	}

	// Verify dependencies
	deps := graph.GetDependencies("node1")
	if len(deps) != 1 || deps[0] != "node2" {
		t.Errorf("Expected node1 to depend on node2")
	}
}

func TestTopologicalSort(t *testing.T) {
	graph := NewGraph(GraphMetadata{})

	// Create a simple DAG where node2 depends on node1, node3 depends on node2
	// In our DAG semantics: Edge(From, To) means FROM depends ON TO
	// So node1 should execute first, then node2, then node3
	node1 := &Node{ID: "node1", Name: "Node 1", ResourceType: types.KindProject}
	node2 := &Node{ID: "node2", Name: "Node 2", ResourceType: types.KindCluster}
	node3 := &Node{ID: "node3", Name: "Node 3", ResourceType: types.KindDatabaseUser}

	_ = graph.AddNode(node1)
	_ = graph.AddNode(node2)
	_ = graph.AddNode(node3)

	// node2 depends on node1, node3 depends on node2
	_ = graph.AddEdge(&Edge{From: "node2", To: "node1", Type: DependencyTypeHard})
	_ = graph.AddEdge(&Edge{From: "node3", To: "node2", Type: DependencyTypeHard})

	order, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(order) != 3 {
		t.Errorf("Expected 3 nodes in order, got %d", len(order))
	}

	// In topological order: node1 should come before node2, node2 before node3
	pos1, pos2, pos3 := -1, -1, -1
	for i, id := range order {
		if id == "node1" {
			pos1 = i
		} else if id == "node2" {
			pos2 = i
		} else if id == "node3" {
			pos3 = i
		}
	}

	// node1 has no dependencies, node2 depends on node1, node3 depends on node2
	// So execution order should be: node1, node2, node3
	// In our result positions should be: pos1 < pos2 < pos3
	if !(pos1 < pos2 && pos2 < pos3) {
		t.Errorf("Invalid topological order: %v (positions: %d, %d, %d) - expected node1 before node2 before node3", order, pos1, pos2, pos3)
	}
}

func TestCycleDetection(t *testing.T) {
	graph := NewGraph(GraphMetadata{})

	node1 := &Node{ID: "node1", Name: "Node 1", ResourceType: types.KindCluster}
	node2 := &Node{ID: "node2", Name: "Node 2", ResourceType: types.KindDatabaseUser}
	node3 := &Node{ID: "node3", Name: "Node 3", ResourceType: types.KindNetworkAccess}

	_ = graph.AddNode(node1)
	_ = graph.AddNode(node2)
	_ = graph.AddNode(node3)

	// Create a cycle: 1 -> 2 -> 3 -> 1
	_ = graph.AddEdge(&Edge{From: "node1", To: "node2", Type: DependencyTypeHard})
	_ = graph.AddEdge(&Edge{From: "node2", To: "node3", Type: DependencyTypeHard})
	_ = graph.AddEdge(&Edge{From: "node3", To: "node1", Type: DependencyTypeHard})

	hasCycle, cycle := graph.HasCycle()
	if !hasCycle {
		t.Error("Expected to detect cycle")
	}

	if len(cycle) == 0 {
		t.Error("Expected non-empty cycle path")
	}
}

func TestCriticalPathMethod(t *testing.T) {
	graph := NewGraph(GraphMetadata{})

	// Create nodes with durations
	node1 := &Node{
		ID:           "node1",
		Name:         "Node 1",
		ResourceType: types.KindProject,
		Properties: NodeProperties{
			EstimatedDuration: 10 * time.Minute,
		},
	}
	node2 := &Node{
		ID:           "node2",
		Name:         "Node 2",
		ResourceType: types.KindCluster,
		Properties: NodeProperties{
			EstimatedDuration: 20 * time.Minute,
		},
	}
	node3 := &Node{
		ID:           "node3",
		Name:         "Node 3",
		ResourceType: types.KindDatabaseUser,
		Properties: NodeProperties{
			EstimatedDuration: 15 * time.Minute,
		},
	}

	_ = graph.AddNode(node1)
	_ = graph.AddNode(node2)
	_ = graph.AddNode(node3)

	// node2 depends on node1, node3 depends on node2
	_ = graph.AddEdge(&Edge{From: "node2", To: "node1", Type: DependencyTypeHard})
	_ = graph.AddEdge(&Edge{From: "node3", To: "node2", Type: DependencyTypeHard})

	criticalPath, duration, err := graph.CriticalPathMethod()
	if err != nil {
		t.Fatalf("CriticalPathMethod failed: %v", err)
	}

	expectedDuration := 45 * time.Minute // 10 + 20 + 15
	if duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, duration)
	}

	if len(criticalPath) != 3 {
		t.Errorf("Expected 3 nodes in critical path, got %d", len(criticalPath))
	}
}

func TestAnalyzer(t *testing.T) {
	graph := NewGraph(GraphMetadata{Name: "test-analysis"})

	// Create a simple graph
	for i := 1; i <= 5; i++ {
		node := &Node{
			ID:           fmt.Sprintf("node%d", i),
			Name:         fmt.Sprintf("Node %d", i),
			ResourceType: types.KindCluster,
			Properties: NodeProperties{
				EstimatedDuration: time.Duration(i) * time.Minute,
				RiskLevel:         RiskLevelLow,
			},
		}
		_ = graph.AddNode(node)
	}

	// Add some dependencies
	_ = graph.AddEdge(&Edge{From: "node2", To: "node1", Type: DependencyTypeHard})
	_ = graph.AddEdge(&Edge{From: "node3", To: "node1", Type: DependencyTypeHard})
	_ = graph.AddEdge(&Edge{From: "node4", To: "node2", Type: DependencyTypeHard})
	_ = graph.AddEdge(&Edge{From: "node5", To: "node3", Type: DependencyTypeHard})

	analyzer := NewAnalyzer(graph)
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if analysis.NodeCount != 5 {
		t.Errorf("Expected 5 nodes, got %d", analysis.NodeCount)
	}

	if analysis.EdgeCount != 4 {
		t.Errorf("Expected 4 edges, got %d", analysis.EdgeCount)
	}

	if analysis.HasCycles {
		t.Error("Did not expect cycles")
	}

	if len(analysis.CriticalPath) == 0 {
		t.Error("Expected non-empty critical path")
	}

	if analysis.ParallelizationFactor <= 0 {
		t.Error("Expected positive parallelization factor")
	}
}

func TestGraphClone(t *testing.T) {
	original := NewGraph(GraphMetadata{Name: "original"})

	node1 := &Node{ID: "node1", Name: "Node 1", ResourceType: types.KindCluster}
	node2 := &Node{ID: "node2", Name: "Node 2", ResourceType: types.KindDatabaseUser}

	_ = original.AddNode(node1)
	_ = original.AddNode(node2)
	_ = original.AddEdge(&Edge{From: "node1", To: "node2", Type: DependencyTypeHard})

	clone := original.Clone()

	if clone.NodeCount() != original.NodeCount() {
		t.Errorf("Clone has different node count: %d vs %d", clone.NodeCount(), original.NodeCount())
	}

	if clone.EdgeCount() != original.EdgeCount() {
		t.Errorf("Clone has different edge count: %d vs %d", clone.EdgeCount(), original.EdgeCount())
	}

	// Modify clone and ensure original is unchanged
	_ = clone.AddNode(&Node{ID: "node3", Name: "Node 3", ResourceType: types.KindNetworkAccess})

	if original.NodeCount() == clone.NodeCount() {
		t.Error("Modifying clone affected original")
	}
}

func TestValidation(t *testing.T) {
	graph := NewGraph(GraphMetadata{})

	node1 := &Node{ID: "node1", Name: "Node 1", ResourceType: types.KindCluster}
	node2 := &Node{ID: "node2", Name: "Node 2", ResourceType: types.KindDatabaseUser}

	graph.AddNode(node1)
	graph.AddNode(node2)
	graph.AddEdge(&Edge{From: "node1", To: "node2", Type: DependencyTypeHard})

	// Valid graph
	err := graph.Validate()
	if err != nil {
		t.Errorf("Validation failed for valid graph: %v", err)
	}

	// Create a graph with cycle
	graph.AddEdge(&Edge{From: "node2", To: "node1", Type: DependencyTypeHard})
	err = graph.Validate()
	if err == nil {
		t.Error("Expected validation to fail for graph with cycle")
	}
}
