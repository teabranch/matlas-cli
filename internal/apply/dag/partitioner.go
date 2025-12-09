package dag

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// PartitionStrategy defines how to partition the graph
type PartitionStrategy string

const (
	// PartitionByLevel partitions based on dependency levels
	PartitionByLevel PartitionStrategy = "level"

	// PartitionByRegion partitions based on resource region/location
	PartitionByRegion PartitionStrategy = "region"

	// PartitionByResourceType partitions based on resource types
	PartitionByResourceType PartitionStrategy = "resource_type"

	// PartitionBalanced creates balanced partitions by node count
	PartitionBalanced PartitionStrategy = "balanced"

	// PartitionMinCut minimizes cross-partition dependencies
	PartitionMinCut PartitionStrategy = "min_cut"
)

// Partitioner partitions graphs for distributed execution
type Partitioner struct {
	strategy      PartitionStrategy
	numPartitions int
}

// NewPartitioner creates a new partitioner
func NewPartitioner(strategy PartitionStrategy, numPartitions int) *Partitioner {
	if numPartitions < 1 {
		numPartitions = 1
	}
	return &Partitioner{
		strategy:      strategy,
		numPartitions: numPartitions,
	}
}

// Partition partitions a graph into independent subgraphs
func (p *Partitioner) Partition(ctx context.Context, graph *Graph) ([]*GraphPartition, error) {
	if graph == nil {
		return nil, fmt.Errorf("graph cannot be nil")
	}

	switch p.strategy {
	case PartitionByLevel:
		return p.partitionByLevel(ctx, graph)
	case PartitionByRegion:
		return p.partitionByRegion(ctx, graph)
	case PartitionByResourceType:
		return p.partitionByResourceType(ctx, graph)
	case PartitionBalanced:
		return p.partitionBalanced(ctx, graph)
	case PartitionMinCut:
		return p.partitionMinCut(ctx, graph)
	default:
		return nil, fmt.Errorf("unknown partition strategy: %s", p.strategy)
	}
}

// partitionByLevel partitions based on dependency levels
func (p *Partitioner) partitionByLevel(ctx context.Context, graph *Graph) ([]*GraphPartition, error) {
	// Compute levels
	if err := graph.ComputeLevels(); err != nil {
		return nil, fmt.Errorf("failed to compute levels: %w", err)
	}

	// Group nodes by level
	levelMap := make(map[int][]*Node)
	for _, node := range graph.Nodes {
		levelMap[node.Level] = append(levelMap[node.Level], node)
	}

	// Distribute levels across partitions
	levels := make([]int, 0, len(levelMap))
	for level := range levelMap {
		levels = append(levels, level)
	}
	sort.Ints(levels)

	partitions := make([]*GraphPartition, p.numPartitions)
	for i := 0; i < p.numPartitions; i++ {
		partitions[i] = &GraphPartition{
			ID:    fmt.Sprintf("partition-%d", i),
			Nodes: make([]*Node, 0),
			Graph: NewGraph(GraphMetadata{Name: fmt.Sprintf("partition-%d", i)}),
		}
	}

	// Distribute levels round-robin
	for idx, level := range levels {
		partitionIdx := idx % p.numPartitions
		partition := partitions[partitionIdx]

		for _, node := range levelMap[level] {
			partition.Nodes = append(partition.Nodes, node)
			_ = partition.Graph.AddNode(node)
		}
	}

	// Add edges within partitions
	for _, partition := range partitions {
		nodeSet := make(map[string]bool)
		for _, node := range partition.Nodes {
			nodeSet[node.ID] = true
		}

		for _, node := range partition.Nodes {
			for _, edge := range graph.Edges[node.ID] {
				// Only add edges where both nodes are in the same partition
				if nodeSet[edge.To] {
					_ = partition.Graph.AddEdge(edge)
				} else {
					// Cross-partition dependency
					partition.CrossPartitionDeps = append(partition.CrossPartitionDeps, edge)
				}
			}
		}
	}

	return partitions, nil
}

// partitionByRegion partitions based on resource region
func (p *Partitioner) partitionByRegion(ctx context.Context, graph *Graph) ([]*GraphPartition, error) {
	// Group nodes by region (from labels or metadata)
	regionMap := make(map[string][]*Node)
	defaultRegion := "default"

	for _, node := range graph.Nodes {
		region := defaultRegion

		// Check for region in labels
		if regionLabel, ok := node.Labels["region"]; ok {
			region = regionLabel
		} else if regionLabel, ok := node.Labels["location"]; ok {
			region = regionLabel
		}

		regionMap[region] = append(regionMap[region], node)
	}

	// Create partitions for each region
	partitions := make([]*GraphPartition, 0, len(regionMap))
	for region, nodes := range regionMap {
		partition := &GraphPartition{
			ID:     fmt.Sprintf("region-%s", region),
			Region: region,
			Nodes:  nodes,
			Graph:  NewGraph(GraphMetadata{Name: fmt.Sprintf("region-%s", region)}),
		}

		// Add nodes to partition graph
		for _, node := range nodes {
			_ = partition.Graph.AddNode(node)
		}

		// Add edges
		nodeSet := make(map[string]bool)
		for _, node := range nodes {
			nodeSet[node.ID] = true
		}

		for _, node := range nodes {
			for _, edge := range graph.Edges[node.ID] {
				if nodeSet[edge.To] {
					_ = partition.Graph.AddEdge(edge)
				} else {
					partition.CrossPartitionDeps = append(partition.CrossPartitionDeps, edge)
				}
			}
		}

		partitions = append(partitions, partition)
	}

	return partitions, nil
}

// partitionByResourceType partitions based on resource types
func (p *Partitioner) partitionByResourceType(ctx context.Context, graph *Graph) ([]*GraphPartition, error) {
	// Group nodes by resource type
	typeMap := make(map[string][]*Node)

	for _, node := range graph.Nodes {
		resourceType := string(node.ResourceType)
		if resourceType == "" {
			resourceType = "unknown"
		}
		typeMap[resourceType] = append(typeMap[resourceType], node)
	}

	// Create partition for each resource type
	partitions := make([]*GraphPartition, 0, len(typeMap))
	for resourceType, nodes := range typeMap {
		partition := &GraphPartition{
			ID:           fmt.Sprintf("type-%s", resourceType),
			ResourceType: resourceType,
			Nodes:        nodes,
			Graph:        NewGraph(GraphMetadata{Name: fmt.Sprintf("type-%s", resourceType)}),
		}

		// Add nodes to partition graph
		for _, node := range nodes {
			_ = partition.Graph.AddNode(node)
		}

		// Add edges
		nodeSet := make(map[string]bool)
		for _, node := range nodes {
			nodeSet[node.ID] = true
		}

		for _, node := range nodes {
			for _, edge := range graph.Edges[node.ID] {
				if nodeSet[edge.To] {
					_ = partition.Graph.AddEdge(edge)
				} else {
					partition.CrossPartitionDeps = append(partition.CrossPartitionDeps, edge)
				}
			}
		}

		partitions = append(partitions, partition)
	}

	return partitions, nil
}

// partitionBalanced creates balanced partitions by node count
func (p *Partitioner) partitionBalanced(ctx context.Context, graph *Graph) ([]*GraphPartition, error) {
	// Get topological order to maintain dependencies
	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to get topological order: %w", err)
	}

	// Create partitions
	partitions := make([]*GraphPartition, p.numPartitions)
	for i := 0; i < p.numPartitions; i++ {
		partitions[i] = &GraphPartition{
			ID:    fmt.Sprintf("partition-%d", i),
			Nodes: make([]*Node, 0),
			Graph: NewGraph(GraphMetadata{Name: fmt.Sprintf("partition-%d", i)}),
		}
	}

	// Distribute nodes round-robin in topological order
	for idx, nodeID := range order {
		partitionIdx := idx % p.numPartitions
		node := graph.Nodes[nodeID]

		partition := partitions[partitionIdx]
		partition.Nodes = append(partition.Nodes, node)
		_ = partition.Graph.AddNode(node)
	}

	// Add edges
	for _, partition := range partitions {
		nodeSet := make(map[string]bool)
		for _, node := range partition.Nodes {
			nodeSet[node.ID] = true
		}

		for _, node := range partition.Nodes {
			for _, edge := range graph.Edges[node.ID] {
				if nodeSet[edge.To] {
					_ = partition.Graph.AddEdge(edge)
				} else {
					partition.CrossPartitionDeps = append(partition.CrossPartitionDeps, edge)
				}
			}
		}
	}

	return partitions, nil
}

// partitionMinCut minimizes cross-partition dependencies using greedy approach
func (p *Partitioner) partitionMinCut(ctx context.Context, graph *Graph) ([]*GraphPartition, error) {
	// Start with topological order
	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to get topological order: %w", err)
	}

	// Create partitions
	partitions := make([]*GraphPartition, p.numPartitions)
	for i := 0; i < p.numPartitions; i++ {
		partitions[i] = &GraphPartition{
			ID:    fmt.Sprintf("partition-%d", i),
			Nodes: make([]*Node, 0),
			Graph: NewGraph(GraphMetadata{Name: fmt.Sprintf("partition-%d", i)}),
		}
	}

	// Assign nodes to partitions greedily
	// Try to keep connected nodes together
	nodeToPartition := make(map[string]int)

	for _, nodeID := range order {
		node := graph.Nodes[nodeID]

		// Count how many dependencies are in each partition
		partitionScores := make([]int, p.numPartitions)

		for _, edge := range graph.Edges[nodeID] {
			if partitionIdx, assigned := nodeToPartition[edge.To]; assigned {
				partitionScores[partitionIdx]++
			}
		}

		// Find partition with highest score (most dependencies already there)
		bestPartition := 0
		bestScore := partitionScores[0]
		bestSize := len(partitions[0].Nodes)

		for i := 1; i < p.numPartitions; i++ {
			score := partitionScores[i]
			size := len(partitions[i].Nodes)

			// Prefer partition with more dependencies, but balance size
			if score > bestScore || (score == bestScore && size < bestSize) {
				bestPartition = i
				bestScore = score
				bestSize = size
			}
		}

		// Assign to best partition
		partition := partitions[bestPartition]
		partition.Nodes = append(partition.Nodes, node)
		partition.Graph.AddNode(node)
		nodeToPartition[nodeID] = bestPartition
	}

	// Add edges
	for _, partition := range partitions {
		nodeSet := make(map[string]bool)
		for _, node := range partition.Nodes {
			nodeSet[node.ID] = true
		}

		for _, node := range partition.Nodes {
			for _, edge := range graph.Edges[node.ID] {
				if nodeSet[edge.To] {
					partition.Graph.AddEdge(edge)
				} else {
					partition.CrossPartitionDeps = append(partition.CrossPartitionDeps, edge)
				}
			}
		}
	}

	return partitions, nil
}

// AnalyzePartitions analyzes partition quality
func (p *Partitioner) AnalyzePartitions(partitions []*GraphPartition) *PartitionAnalysis {
	if len(partitions) == 0 {
		return nil
	}

	totalNodes := 0
	totalInternalEdges := 0
	totalCrossEdges := 0
	minSize := -1
	maxSize := 0

	for _, partition := range partitions {
		size := len(partition.Nodes)
		totalNodes += size

		if minSize == -1 || size < minSize {
			minSize = size
		}
		if size > maxSize {
			maxSize = size
		}

		// Count internal edges
		totalInternalEdges += partition.Graph.EdgeCount()

		// Count cross-partition edges
		totalCrossEdges += len(partition.CrossPartitionDeps)
	}

	avgSize := float64(totalNodes) / float64(len(partitions))

	// Balance metric (0 = perfectly balanced, 1 = completely imbalanced)
	balance := 0.0
	if avgSize > 0 {
		balance = float64(maxSize-minSize) / avgSize
	}

	// Edge cut ratio (lower is better)
	edgeCutRatio := 0.0
	totalEdges := totalInternalEdges + totalCrossEdges
	if totalEdges > 0 {
		edgeCutRatio = float64(totalCrossEdges) / float64(totalEdges)
	}

	// Independence score (0 = completely dependent, 1 = fully independent)
	independence := 1.0 - edgeCutRatio

	return &PartitionAnalysis{
		NumPartitions:       len(partitions),
		TotalNodes:          totalNodes,
		AvgPartitionSize:    avgSize,
		MinPartitionSize:    minSize,
		MaxPartitionSize:    maxSize,
		Balance:             1.0 - balance, // Invert so higher is better
		InternalEdges:       totalInternalEdges,
		CrossPartitionEdges: totalCrossEdges,
		EdgeCutRatio:        edgeCutRatio,
		Independence:        independence,
	}
}

// MergePartitionResults merges results from distributed execution
func MergePartitionResults(results []*PartitionResult) *MergedResult {
	if len(results) == 0 {
		return nil
	}

	merged := &MergedResult{
		PartitionResults: results,
		TotalDuration:    0,
		Success:          true,
		Errors:           make([]string, 0),
	}

	// Find maximum duration (parallel execution time)
	for _, result := range results {
		if result.Duration > merged.TotalDuration {
			merged.TotalDuration = result.Duration
		}

		if !result.Success {
			merged.Success = false
		}

		if result.Error != "" {
			merged.Errors = append(merged.Errors, fmt.Sprintf("[%s] %s", result.PartitionID, result.Error))
		}
	}

	return merged
}

// GraphPartition represents a partition of the graph
type GraphPartition struct {
	ID                 string  `json:"id"`
	Region             string  `json:"region,omitempty"`
	ResourceType       string  `json:"resourceType,omitempty"`
	Nodes              []*Node `json:"nodes"`
	Graph              *Graph  `json:"graph"`
	CrossPartitionDeps []*Edge `json:"crossPartitionDeps,omitempty"`
}

// PartitionAnalysis contains metrics about partition quality
type PartitionAnalysis struct {
	NumPartitions       int     `json:"numPartitions"`
	TotalNodes          int     `json:"totalNodes"`
	AvgPartitionSize    float64 `json:"avgPartitionSize"`
	MinPartitionSize    int     `json:"minPartitionSize"`
	MaxPartitionSize    int     `json:"maxPartitionSize"`
	Balance             float64 `json:"balance"` // 0-1, higher is better
	InternalEdges       int     `json:"internalEdges"`
	CrossPartitionEdges int     `json:"crossPartitionEdges"`
	EdgeCutRatio        float64 `json:"edgeCutRatio"` // 0-1, lower is better
	Independence        float64 `json:"independence"` // 0-1, higher is better
}

// PartitionResult represents the result of executing a partition
type PartitionResult struct {
	PartitionID         string        `json:"partitionId"`
	Success             bool          `json:"success"`
	Duration            time.Duration `json:"duration"`
	OperationsCompleted int           `json:"operationsCompleted"`
	Error               string        `json:"error,omitempty"`
}

// MergedResult represents merged results from all partitions
type MergedResult struct {
	PartitionResults []*PartitionResult `json:"partitionResults"`
	TotalDuration    time.Duration      `json:"totalDuration"`
	Success          bool               `json:"success"`
	Errors           []string           `json:"errors,omitempty"`
}
