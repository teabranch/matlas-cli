package dag

import (
	"fmt"
	"sort"
	"time"
)

// TopologicalSort returns nodes in topological order using Kahn's algorithm
func (g *Graph) TopologicalSort() ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	// Check for cycles first (using internal method to avoid double-locking)
	if hasCycle, cycle := g.hasCycleInternal(); hasCycle {
		return nil, fmt.Errorf("cannot perform topological sort: graph contains cycle: %v", cycle)
	}
	
	// Calculate in-degree for each node
	// In-degree = number of nodes this node depends on = number of outgoing edges
	inDegree := make(map[string]int)
	for nodeID := range g.Nodes {
		inDegree[nodeID] = len(g.Edges[nodeID])
	}
	
	// Find all nodes with no incoming edges (in-degree = 0)
	queue := make([]string, 0)
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}
	
	// Process nodes in order
	result := make([]string, 0, len(g.Nodes))
	for len(queue) > 0 {
		// Remove a node with no dependencies
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		
		// For each node that depends on current (nodes with edges TO current)
		// When current completes, decrement their dependency count
		for _, edge := range g.ReverseEdges[current] {
			dependent := edge.From
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}
	
	// Verify all nodes were processed
	if len(result) != len(g.Nodes) {
		return nil, fmt.Errorf("topological sort failed: processed %d nodes but graph has %d nodes", len(result), len(g.Nodes))
	}
	
	return result, nil
}

// topologicalSortInternal is the internal implementation without locking
func (g *Graph) topologicalSortInternal() ([]string, error) {
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for nodeID := range g.Nodes {
		inDegree[nodeID] = len(g.Edges[nodeID])
	}
	
	// Find all nodes with no incoming edges
	queue := make([]string, 0)
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}
	
	// Process nodes in order
	result := make([]string, 0, len(g.Nodes))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		
		for _, edge := range g.ReverseEdges[current] {
			dependent := edge.From
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}
	
	if len(result) != len(g.Nodes) {
		return nil, fmt.Errorf("topological sort failed: processed %d nodes but graph has %d nodes", len(result), len(g.Nodes))
	}
	
	return result, nil
}

// TopologicalSortDFS returns nodes in topological order using DFS-based algorithm
func (g *Graph) TopologicalSortDFS() ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	// Check for cycles first (using internal method to avoid double-locking)
	if hasCycle, cycle := g.hasCycleInternal(); hasCycle {
		return nil, fmt.Errorf("cannot perform topological sort: graph contains cycle: %v", cycle)
	}
	
	visited := make(map[string]bool)
	stack := make([]string, 0, len(g.Nodes))
	
	// Visit all nodes
	for nodeID := range g.Nodes {
		if !visited[nodeID] {
			g.topologicalSortDFSUtil(nodeID, visited, &stack)
		}
	}
	
	// Reverse the stack to get topological order
	result := make([]string, len(stack))
	for i, j := 0, len(stack)-1; i < len(stack); i, j = i+1, j-1 {
		result[i] = stack[j]
	}
	
	return result, nil
}

// topologicalSortDFSUtil is a recursive helper for DFS-based topological sort
func (g *Graph) topologicalSortDFSUtil(nodeID string, visited map[string]bool, stack *[]string) {
	visited[nodeID] = true
	
	// Visit all dependencies first
	for _, edge := range g.Edges[nodeID] {
		if !visited[edge.To] {
			g.topologicalSortDFSUtil(edge.To, visited, stack)
		}
	}
	
	// Push to stack after visiting all dependencies
	*stack = append(*stack, nodeID)
}

// CriticalPathMethod computes the critical path using forward and backward pass
func (g *Graph) CriticalPathMethod() ([]string, time.Duration, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// Verify no cycles (using internal method)
	if hasCycle, cycle := g.hasCycleInternal(); hasCycle {
		return nil, 0, fmt.Errorf("cannot compute critical path: graph contains cycle: %v", cycle)
	}
	
	// Get topological order (using internal method)
	topoOrder, err := g.topologicalSortInternal()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get topological order: %w", err)
	}
	
	// Forward pass: compute earliest start times
	for _, nodeID := range topoOrder {
		node := g.Nodes[nodeID]
		node.EarliestStart = 0
		
		// Find maximum earliest start + duration of dependencies
		// Dependencies are in Edges[nodeID] (nodes this node depends on)
		for _, edge := range g.Edges[nodeID] {
			depNode := g.Nodes[edge.To]
			earliestFinish := depNode.EarliestStart + depNode.Properties.EstimatedDuration
			if earliestFinish > node.EarliestStart {
				node.EarliestStart = earliestFinish
			}
		}
	}
	
	// Find project completion time (maximum earliest start + duration)
	var projectDuration time.Duration
	for _, node := range g.Nodes {
		finishTime := node.EarliestStart + node.Properties.EstimatedDuration
		if finishTime > projectDuration {
			projectDuration = finishTime
		}
	}
	
	// Backward pass: compute latest start times
	// Initialize all latest start times to project duration
	for _, node := range g.Nodes {
		node.LatestStart = projectDuration - node.Properties.EstimatedDuration
	}
	
	// Process in reverse topological order
	for i := len(topoOrder) - 1; i >= 0; i-- {
		nodeID := topoOrder[i]
		node := g.Nodes[nodeID]
		
		// Find minimum latest start of dependents
		// Dependents are nodes that depend on this node (in ReverseEdges)
		minLatestStart := projectDuration
		for _, edge := range g.ReverseEdges[nodeID] {
			depNode := g.Nodes[edge.From]
			if depNode.LatestStart < minLatestStart {
				minLatestStart = depNode.LatestStart
			}
		}
		
		// Adjust if we have dependents
		if len(g.ReverseEdges[nodeID]) > 0 {
			node.LatestStart = minLatestStart - node.Properties.EstimatedDuration
		}
	}
	
	// Compute slack and identify critical path
	criticalPath := make([]string, 0)
	for _, node := range g.Nodes {
		node.Slack = node.LatestStart - node.EarliestStart
		if node.Slack == 0 {
			node.IsCritical = true
			criticalPath = append(criticalPath, node.ID)
		} else {
			node.IsCritical = false
		}
	}
	
	// Sort critical path by earliest start time
	sort.Slice(criticalPath, func(i, j int) bool {
		return g.Nodes[criticalPath[i]].EarliestStart < g.Nodes[criticalPath[j]].EarliestStart
	})
	
	// Mark critical edges
	for i := 0; i < len(criticalPath)-1; i++ {
		from := criticalPath[i]
		to := criticalPath[i+1]
		
		// Mark edge as critical if it exists
		for _, edge := range g.Edges[from] {
			if edge.To == to {
				edge.IsCritical = true
			}
		}
	}
	
	// Store in graph
	g.CriticalPath = criticalPath
	g.TotalDuration = projectDuration
	
	return criticalPath, projectDuration, nil
}

// LongestPath finds the longest path from any source to any sink
func (g *Graph) LongestPath() ([]string, time.Duration, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	// Get topological order (using internal method)
	topoOrder, err := g.topologicalSortInternal()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get topological order: %w", err)
	}
	
	// Initialize distances to negative infinity
	dist := make(map[string]time.Duration)
	parent := make(map[string]string)
	for nodeID := range g.Nodes {
		dist[nodeID] = -1
	}
	
	// Set distance of root nodes to their duration
	roots := make([]*Node, 0)
	for _, node := range g.Nodes {
		if len(g.ReverseEdges[node.ID]) == 0 {
			roots = append(roots, node)
		}
	}
	for _, root := range roots {
		dist[root.ID] = root.Properties.EstimatedDuration
	}
	
	// Process nodes in topological order
	for _, nodeID := range topoOrder {
		if dist[nodeID] == -1 {
			continue
		}
		
		// Update distances of dependents
		for _, edge := range g.Edges[nodeID] {
			depNode := g.Nodes[edge.To]
			newDist := dist[nodeID] + depNode.Properties.EstimatedDuration
			if newDist > dist[edge.To] {
				dist[edge.To] = newDist
				parent[edge.To] = nodeID
			}
		}
	}
	
	// Find node with maximum distance
	var maxDist time.Duration
	var endNode string
	for nodeID, d := range dist {
		if d > maxDist {
			maxDist = d
			endNode = nodeID
		}
	}
	
	// Reconstruct path
	path := make([]string, 0)
	current := endNode
	for current != "" {
		path = append([]string{current}, path...)
		current = parent[current]
	}
	
	return path, maxDist, nil
}

// FindAllPaths finds all paths from source to target
func (g *Graph) FindAllPaths(from, to string) ([][]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if _, exists := g.Nodes[from]; !exists {
		return nil, fmt.Errorf("source node %s not found", from)
	}
	if _, exists := g.Nodes[to]; !exists {
		return nil, fmt.Errorf("target node %s not found", to)
	}
	
	paths := make([][]string, 0)
	currentPath := make([]string, 0)
	visited := make(map[string]bool)
	
	g.findAllPathsUtil(from, to, visited, currentPath, &paths)
	
	return paths, nil
}

// findAllPathsUtil is a recursive helper for finding all paths
func (g *Graph) findAllPathsUtil(current, target string, visited map[string]bool, currentPath []string, paths *[][]string) {
	visited[current] = true
	currentPath = append(currentPath, current)
	
	if current == target {
		// Found a path, add a copy to results
		pathCopy := make([]string, len(currentPath))
		copy(pathCopy, currentPath)
		*paths = append(*paths, pathCopy)
	} else {
		// Explore all dependents
		for _, edge := range g.Edges[current] {
			if !visited[edge.To] {
				g.findAllPathsUtil(edge.To, target, visited, currentPath, paths)
			}
		}
	}
	
	// Backtrack
	visited[current] = false
}

// StronglyConnectedComponents finds all strongly connected components using Tarjan's algorithm
func (g *Graph) StronglyConnectedComponents() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	index := 0
	stack := make([]string, 0)
	indices := make(map[string]int)
	lowLinks := make(map[string]int)
	onStack := make(map[string]bool)
	sccs := make([][]string, 0)
	
	for nodeID := range g.Nodes {
		if _, visited := indices[nodeID]; !visited {
			g.strongConnectUtil(nodeID, &index, &stack, indices, lowLinks, onStack, &sccs)
		}
	}
	
	return sccs
}

// strongConnectUtil is a recursive helper for Tarjan's SCC algorithm
func (g *Graph) strongConnectUtil(nodeID string, index *int, stack *[]string, indices, lowLinks map[string]int, onStack map[string]bool, sccs *[][]string) {
	indices[nodeID] = *index
	lowLinks[nodeID] = *index
	*index++
	*stack = append(*stack, nodeID)
	onStack[nodeID] = true
	
	// Consider successors
	for _, edge := range g.Edges[nodeID] {
		successor := edge.To
		if _, visited := indices[successor]; !visited {
			g.strongConnectUtil(successor, index, stack, indices, lowLinks, onStack, sccs)
			if lowLinks[successor] < lowLinks[nodeID] {
				lowLinks[nodeID] = lowLinks[successor]
			}
		} else if onStack[successor] {
			if indices[successor] < lowLinks[nodeID] {
				lowLinks[nodeID] = indices[successor]
			}
		}
	}
	
	// If nodeID is a root node, pop the stack to generate an SCC
	if lowLinks[nodeID] == indices[nodeID] {
		scc := make([]string, 0)
		for {
			w := (*stack)[len(*stack)-1]
			*stack = (*stack)[:len(*stack)-1]
			onStack[w] = false
			scc = append(scc, w)
			if w == nodeID {
				break
			}
		}
		*sccs = append(*sccs, scc)
	}
}

// TransitiveClosure computes the transitive closure of the graph
func (g *Graph) TransitiveClosure() map[string]map[string]bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	closure := make(map[string]map[string]bool)
	
	// Initialize closure with direct edges
	for nodeID := range g.Nodes {
		closure[nodeID] = make(map[string]bool)
		closure[nodeID][nodeID] = true // Reflexive
		
		for _, edge := range g.Edges[nodeID] {
			closure[nodeID][edge.To] = true
		}
	}
	
	// Floyd-Warshall algorithm
	for k := range g.Nodes {
		for i := range g.Nodes {
			for j := range g.Nodes {
				if closure[i][k] && closure[k][j] {
					closure[i][j] = true
				}
			}
		}
	}
	
	return closure
}

// TransitiveReduction computes the transitive reduction of the graph
func (g *Graph) TransitiveReduction() *Graph {
	g.mu.RLock()
	// Clone the graph (Clone acquires its own lock)
	reduced := g.Clone()
	g.mu.RUnlock()
	
	// Get transitive closure
	closure := g.TransitiveClosure()
	
	// Remove redundant edges
	edgesToRemove := make([][2]string, 0)
	for from := range reduced.Nodes {
		for _, edge := range reduced.Edges[from] {
			to := edge.To
			
			// Check if there's an alternative path from 'from' to 'to'
			for intermediate := range reduced.Nodes {
				if intermediate != from && intermediate != to {
					// If there's a path from -> intermediate -> to, this edge is redundant
					if closure[from][intermediate] && closure[intermediate][to] {
						edgesToRemove = append(edgesToRemove, [2]string{from, to})
						break
					}
				}
			}
		}
	}
	
	// Remove redundant edges
	for _, edge := range edgesToRemove {
		reduced.RemoveEdge(edge[0], edge[1])
	}
	
	return reduced
}

// GetCriticalNodes returns nodes that, if removed, would disconnect the graph
func (g *Graph) GetCriticalNodes() []string {
	g.mu.RLock()
	nodeIDs := make([]string, 0, len(g.Nodes))
	for nodeID := range g.Nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	g.mu.RUnlock()
	
	critical := make([]string, 0)
	
	for _, nodeID := range nodeIDs {
		// Try removing the node temporarily
		clone := g.Clone()
		clone.RemoveNode(nodeID)
		
		// Check if graph is still connected (for root to leaves)
		roots := clone.GetRootNodes()
		leaves := clone.GetLeafNodes()
		
		// If any leaf is no longer reachable from any root, the node is critical
		for _, root := range roots {
			for _, leaf := range leaves {
				if !clone.IsReachable(root.ID, leaf.ID) {
					critical = append(critical, nodeID)
					break
				}
			}
		}
	}
	
	return critical
}

// ComputeParallelGroups groups nodes that can execute in parallel
func (g *Graph) ComputeParallelGroups() ([][]*Node, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// Compute levels first (internal method)
	if err := g.computeLevelsInternal(); err != nil {
		return nil, err
	}
	
	// Group by level (internal - no locking needed since we already hold the lock)
	levelGroups := make(map[int][]*Node)
	for _, node := range g.Nodes {
		level := node.Level
		if levelGroups[level] == nil {
			levelGroups[level] = make([]*Node, 0)
		}
		levelGroups[level] = append(levelGroups[level], node)
	}
	
	// Convert to array of arrays
	groups := make([][]*Node, g.MaxLevel+1)
	for level := 0; level <= g.MaxLevel; level++ {
		groups[level] = levelGroups[level]
	}
	
	return groups, nil
}

// EstimateTotalDuration estimates the total execution time
func (g *Graph) EstimateTotalDuration() (time.Duration, error) {
	_, duration, err := g.CriticalPathMethod()
	if err != nil {
		return 0, fmt.Errorf("failed to estimate duration: %w", err)
	}
	return duration, nil
}

// ComputeSlackDistribution returns a distribution of slack times
func (g *Graph) ComputeSlackDistribution() map[time.Duration]int {
	distribution := make(map[time.Duration]int)
	
	for _, node := range g.Nodes {
		distribution[node.Slack]++
	}
	
	return distribution
}
