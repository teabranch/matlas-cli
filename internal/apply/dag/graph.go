package dag

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// NewGraph creates a new empty graph
func NewGraph(metadata GraphMetadata) *Graph {
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}

	return &Graph{
		Nodes:        make(map[string]*Node),
		Edges:        make(map[string][]*Edge),
		ReverseEdges: make(map[string][]*Edge),
		Metadata:     metadata,
		mu:           sync.RWMutex{},
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *Node) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	// Validate node ID
	if err := validateNodeID(node.ID); err != nil {
		return err
	}

	if _, exists := g.Nodes[node.ID]; exists {
		return fmt.Errorf("node with ID %s already exists", node.ID)
	}

	// Validate duration is non-negative
	if node.Properties.EstimatedDuration < 0 {
		return fmt.Errorf("estimated duration cannot be negative: %v", node.Properties.EstimatedDuration)
	}
	if node.Properties.MinDuration < 0 {
		return fmt.Errorf("min duration cannot be negative: %v", node.Properties.MinDuration)
	}
	if node.Properties.MaxDuration < 0 {
		return fmt.Errorf("max duration cannot be negative: %v", node.Properties.MaxDuration)
	}

	// Initialize maps if needed
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	if node.Dependencies == nil {
		node.Dependencies = make([]*Edge, 0)
	}

	g.Nodes[node.ID] = node
	return nil
}

// validateNodeID validates that a node ID is safe and doesn't contain malicious patterns
func validateNodeID(id string) error {
	if id == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	// Check length limits
	if len(id) > 256 {
		return fmt.Errorf("node ID too long: max 256 characters")
	}

	// Reject path traversal patterns
	if strings.Contains(id, "..") {
		return fmt.Errorf("node ID cannot contain path traversal (..)")
	}

	// Reject command injection patterns
	if strings.ContainsAny(id, ";|&$`\n\r") {
		return fmt.Errorf("node ID contains invalid characters")
	}

	// Reject null bytes
	if strings.Contains(id, "\x00") {
		return fmt.Errorf("node ID cannot contain null bytes")
	}

	// Must be printable ASCII or Unicode letters/numbers/common symbols
	// Allow: alphanumeric, underscore, hyphen, dot, colon, forward slash
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_\-.:/@]+$`)
	if !validPattern.MatchString(id) {
		return fmt.Errorf("node ID contains invalid characters: must be alphanumeric with _-.:/@ only")
	}

	return nil
}

// RemoveNode removes a node and all its associated edges
func (g *Graph) RemoveNode(nodeID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.Nodes[nodeID]; !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// Remove all edges from this node
	delete(g.Edges, nodeID)

	// Remove all edges to this node
	delete(g.ReverseEdges, nodeID)

	// Remove references from other nodes' edge lists
	for fromID := range g.Edges {
		g.Edges[fromID] = filterEdges(g.Edges[fromID], func(e *Edge) bool {
			return e.To != nodeID
		})
	}

	for toID := range g.ReverseEdges {
		g.ReverseEdges[toID] = filterEdges(g.ReverseEdges[toID], func(e *Edge) bool {
			return e.From != nodeID
		})
	}

	// Remove the node
	delete(g.Nodes, nodeID)

	return nil
}

// AddEdge adds a directed edge from one node to another
func (g *Graph) AddEdge(edge *Edge) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if edge == nil {
		return fmt.Errorf("edge cannot be nil")
	}

	if edge.From == "" || edge.To == "" {
		return fmt.Errorf("edge from and to cannot be empty")
	}

	// Verify nodes exist
	if _, exists := g.Nodes[edge.From]; !exists {
		return fmt.Errorf("source node %s not found", edge.From)
	}
	if _, exists := g.Nodes[edge.To]; !exists {
		return fmt.Errorf("target node %s not found", edge.To)
	}

	// Prevent self-loops
	if edge.From == edge.To {
		return fmt.Errorf("self-loops are not allowed: %s", edge.From)
	}

	// Default weight
	if edge.Weight == 0 {
		edge.Weight = 1.0
	}

	// Default type
	if edge.Type == "" {
		edge.Type = DependencyTypeHard
	}

	// Add to forward edges
	if g.Edges[edge.From] == nil {
		g.Edges[edge.From] = make([]*Edge, 0)
	}
	g.Edges[edge.From] = append(g.Edges[edge.From], edge)

	// Add to reverse edges
	if g.ReverseEdges[edge.To] == nil {
		g.ReverseEdges[edge.To] = make([]*Edge, 0)
	}
	g.ReverseEdges[edge.To] = append(g.ReverseEdges[edge.To], edge)

	// Add to node's dependencies
	fromNode := g.Nodes[edge.From]
	fromNode.Dependencies = append(fromNode.Dependencies, edge)

	return nil
}

// RemoveEdge removes an edge between two nodes
func (g *Graph) RemoveEdge(from, to string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Edges[from] == nil {
		return fmt.Errorf("no edges from node %s", from)
	}

	found := false
	g.Edges[from] = filterEdges(g.Edges[from], func(e *Edge) bool {
		if e.To == to {
			found = true
			return false
		}
		return true
	})

	if !found {
		return fmt.Errorf("edge from %s to %s not found", from, to)
	}

	// Remove from reverse edges
	if g.ReverseEdges[to] != nil {
		g.ReverseEdges[to] = filterEdges(g.ReverseEdges[to], func(e *Edge) bool {
			return e.From != from
		})
	}

	// Remove from node's dependencies
	fromNode := g.Nodes[from]
	fromNode.Dependencies = filterEdges(fromNode.Dependencies, func(e *Edge) bool {
		return e.To != to
	})

	return nil
}

// GetNode retrieves a node by ID
func (g *Graph) GetNode(nodeID string) (*Node, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, exists := g.Nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}
	return node, nil
}

// GetEdges returns all edges from a node
func (g *Graph) GetEdges(nodeID string) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Edges[nodeID]
}

// GetIncomingEdges returns all edges to a node
func (g *Graph) GetIncomingEdges(nodeID string) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.ReverseEdges[nodeID]
}

// GetDependencies returns the IDs of all nodes that a node depends on
func (g *Graph) GetDependencies(nodeID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edges := g.Edges[nodeID]
	deps := make([]string, len(edges))
	for i, edge := range edges {
		deps[i] = edge.To
	}
	return deps
}

// GetDependents returns the IDs of all nodes that depend on a node
func (g *Graph) GetDependents(nodeID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edges := g.ReverseEdges[nodeID]
	deps := make([]string, len(edges))
	for i, edge := range edges {
		deps[i] = edge.From
	}
	return deps
}

// NodeCount returns the number of nodes in the graph
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.Nodes)
}

// EdgeCount returns the number of edges in the graph
func (g *Graph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	count := 0
	for _, edges := range g.Edges {
		count += len(edges)
	}
	return count
}

// HasCycle detects if the graph contains any cycles
func (g *Graph) HasCycle() (bool, []string) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.hasCycleInternal()
}

// hasCycleInternal is the internal implementation without locking
func (g *Graph) hasCycleInternal() (bool, []string) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	parent := make(map[string]string)

	for nodeID := range g.Nodes {
		if !visited[nodeID] {
			if hasCycle, cycle := g.hasCycleUtil(nodeID, visited, recStack, parent); hasCycle {
				return true, cycle
			}
		}
	}

	return false, nil
}

// hasCycleUtil is a recursive helper for cycle detection using DFS
func (g *Graph) hasCycleUtil(nodeID string, visited, recStack map[string]bool, parent map[string]string) (bool, []string) {
	visited[nodeID] = true
	recStack[nodeID] = true

	// Visit all dependencies
	for _, edge := range g.Edges[nodeID] {
		dep := edge.To
		parent[dep] = nodeID

		if !visited[dep] {
			if hasCycle, cycle := g.hasCycleUtil(dep, visited, recStack, parent); hasCycle {
				return true, cycle
			}
		} else if recStack[dep] {
			// Found a cycle, reconstruct it
			cycle := []string{dep}
			current := nodeID
			for current != dep {
				cycle = append([]string{current}, cycle...)
				current = parent[current]
			}
			cycle = append([]string{current}, cycle...)
			return true, cycle
		}
	}

	recStack[nodeID] = false
	return false, nil
}

// Clone creates a deep copy of the graph
func (g *Graph) Clone() *Graph {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clone := NewGraph(g.Metadata)

	// Clone nodes
	for id, node := range g.Nodes {
		nodeClone := &Node{
			ID:            node.ID,
			Name:          node.Name,
			ResourceType:  node.ResourceType,
			Properties:    node.Properties,
			Labels:        make(map[string]string),
			Level:         node.Level,
			EarliestStart: node.EarliestStart,
			LatestStart:   node.LatestStart,
			Slack:         node.Slack,
			IsCritical:    node.IsCritical,
		}

		// Clone labels
		for k, v := range node.Labels {
			nodeClone.Labels[k] = v
		}

		clone.Nodes[id] = nodeClone
	}

	// Clone edges
	for _, edges := range g.Edges {
		for _, edge := range edges {
			edgeClone := &Edge{
				From:       edge.From,
				To:         edge.To,
				Type:       edge.Type,
				Weight:     edge.Weight,
				Reason:     edge.Reason,
				IsCritical: edge.IsCritical,
				Metadata:   make(map[string]string),
			}

			// Clone metadata
			if edge.Metadata != nil {
				for k, v := range edge.Metadata {
					edgeClone.Metadata[k] = v
				}
			}

			// Note: Condition is not cloned as it may contain function pointers
			if edge.Condition != nil {
				edgeClone.Condition = edge.Condition
			}

			clone.AddEdge(edgeClone)
		}
	}

	// Copy computed properties
	clone.CriticalPath = append([]string(nil), g.CriticalPath...)
	clone.TotalDuration = g.TotalDuration
	clone.MaxLevel = g.MaxLevel

	return clone
}

// ToJSON serializes the graph to JSON with sensitive data sanitization
func (g *Graph) ToJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Create a sanitized copy for serialization
	sanitized := g.sanitizeForExport()
	return json.MarshalIndent(sanitized, "", "  ")
}

// sanitizeForExport creates a copy of the graph with sensitive data redacted
func (g *Graph) sanitizeForExport() *Graph {
	// Create shallow copy
	export := &Graph{
		Nodes:         make(map[string]*Node),
		Edges:         g.Edges,
		ReverseEdges:  g.ReverseEdges,
		Metadata:      g.Metadata,
		CriticalPath:  g.CriticalPath,
		TotalDuration: g.TotalDuration,
		MaxLevel:      g.MaxLevel,
	}

	// Sanitize sensitive fields in node labels
	sensitiveKeys := []string{"password", "api_key", "apiKey", "token", "secret", "credential", "auth"}

	for id, node := range g.Nodes {
		nodeCopy := *node
		nodeCopy.Labels = make(map[string]string)

		// Copy labels, redacting sensitive ones
		for k, v := range node.Labels {
			isSensitive := false
			keyLower := strings.ToLower(k)
			for _, sensitiveKey := range sensitiveKeys {
				if strings.Contains(keyLower, sensitiveKey) {
					isSensitive = true
					break
				}
			}

			if isSensitive {
				nodeCopy.Labels[k] = "[REDACTED]"
			} else {
				nodeCopy.Labels[k] = v
			}
		}

		export.Nodes[id] = &nodeCopy
	}

	return export
}

// FromJSON deserializes a graph from JSON
func FromJSON(data []byte) (*Graph, error) {
	var graph Graph
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph: %w", err)
	}
	return &graph, nil
}

// GetRootNodes returns all nodes with no dependencies (level 0)
func (g *Graph) GetRootNodes() []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	roots := make([]*Node, 0)
	for _, node := range g.Nodes {
		if len(g.ReverseEdges[node.ID]) == 0 {
			roots = append(roots, node)
		}
	}
	return roots
}

// GetLeafNodes returns all nodes with no dependents
func (g *Graph) GetLeafNodes() []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	leaves := make([]*Node, 0)
	for _, node := range g.Nodes {
		if len(g.Edges[node.ID]) == 0 {
			leaves = append(leaves, node)
		}
	}
	return leaves
}

// GetNodesByLevel returns nodes grouped by their dependency level
func (g *Graph) GetNodesByLevel() map[int][]*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	levels := make(map[int][]*Node)
	for _, node := range g.Nodes {
		level := node.Level
		if levels[level] == nil {
			levels[level] = make([]*Node, 0)
		}
		levels[level] = append(levels[level], node)
	}
	return levels
}

// GetNodesByType returns nodes grouped by their resource type
func (g *Graph) GetNodesByType() map[string][]*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	types := make(map[string][]*Node)
	for _, node := range g.Nodes {
		resourceType := string(node.ResourceType)
		if types[resourceType] == nil {
			types[resourceType] = make([]*Node, 0)
		}
		types[resourceType] = append(types[resourceType], node)
	}
	return types
}

// IsReachable checks if there's a path from source to target
func (g *Graph) IsReachable(from, to string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if from == to {
		return true
	}

	visited := make(map[string]bool)
	return g.isReachableUtil(from, to, visited)
}

// isReachableUtil is a recursive helper for reachability check
func (g *Graph) isReachableUtil(from, to string, visited map[string]bool) bool {
	if from == to {
		return true
	}

	visited[from] = true

	for _, edge := range g.Edges[from] {
		if !visited[edge.To] {
			if g.isReachableUtil(edge.To, to, visited) {
				return true
			}
		}
	}

	return false
}

// GetPath finds a path between two nodes (returns empty if no path exists)
func (g *Graph) GetPath(from, to string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if from == to {
		return []string{from}
	}

	visited := make(map[string]bool)
	parent := make(map[string]string)
	queue := []string{from}
	visited[from] = true

	// BFS to find shortest path
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == to {
			// Reconstruct path
			path := []string{to}
			node := to
			for node != from {
				node = parent[node]
				path = append([]string{node}, path...)
			}
			return path
		}

		for _, edge := range g.Edges[current] {
			if !visited[edge.To] {
				visited[edge.To] = true
				parent[edge.To] = current
				queue = append(queue, edge.To)
			}
		}
	}

	return nil
}

// Validate validates the graph structure
func (g *Graph) Validate() error {
	// Check for cycles (HasCycle already acquires lock)
	if hasCycle, cycle := g.HasCycle(); hasCycle {
		return fmt.Errorf("graph contains cycle: %v", cycle)
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	// Validate all edges reference existing nodes
	for fromID, edges := range g.Edges {
		if _, exists := g.Nodes[fromID]; !exists {
			return fmt.Errorf("edge references non-existent source node: %s", fromID)
		}

		for _, edge := range edges {
			if _, exists := g.Nodes[edge.To]; !exists {
				return fmt.Errorf("edge from %s references non-existent target node: %s", fromID, edge.To)
			}
		}
	}

	// Validate reverse edges match forward edges
	for nodeID := range g.Nodes {
		// Check forward->reverse consistency
		for _, edge := range g.Edges[nodeID] {
			found := false
			for _, revEdge := range g.ReverseEdges[edge.To] {
				if revEdge.From == nodeID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("inconsistency: forward edge %s->%s has no reverse edge", nodeID, edge.To)
			}
		}

		// Check reverse->forward consistency
		for _, edge := range g.ReverseEdges[nodeID] {
			found := false
			for _, fwdEdge := range g.Edges[edge.From] {
				if fwdEdge.To == nodeID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("inconsistency: reverse edge %s->%s has no forward edge", edge.From, nodeID)
			}
		}
	}

	return nil
}

// Helper function to filter edges
func filterEdges(edges []*Edge, predicate func(*Edge) bool) []*Edge {
	filtered := make([]*Edge, 0)
	for _, edge := range edges {
		if predicate(edge) {
			filtered = append(filtered, edge)
		}
	}
	return filtered
}

// ComputeLevels computes and assigns dependency levels to all nodes
func (g *Graph) ComputeLevels() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.computeLevelsInternal()
}

// computeLevelsInternal is the internal implementation without locking
func (g *Graph) computeLevelsInternal() error {
	// Reset levels
	for _, node := range g.Nodes {
		node.Level = 0
	}

	// Get topological order (using internal method)
	order, err := g.topologicalSortInternal()
	if err != nil {
		return fmt.Errorf("cannot compute levels: %w", err)
	}

	// Assign levels based on topological order
	for _, nodeID := range order {
		node := g.Nodes[nodeID]
		maxDepLevel := -1

		// Find maximum level of dependencies (nodes this node depends on)
		// Edges[nodeID] contains edges FROM this node TO its dependencies
		for _, edge := range g.Edges[nodeID] {
			depNode := g.Nodes[edge.To]
			if depNode.Level > maxDepLevel {
				maxDepLevel = depNode.Level
			}
		}

		node.Level = maxDepLevel + 1
		if node.Level > g.MaxLevel {
			g.MaxLevel = node.Level
		}
	}

	return nil
}
