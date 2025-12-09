package dag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// VisualizationFormat defines the output format for visualization
type VisualizationFormat string

const (
	// FormatDOT outputs Graphviz DOT format
	FormatDOT VisualizationFormat = "dot"

	// FormatMermaid outputs Mermaid diagram format
	FormatMermaid VisualizationFormat = "mermaid"

	// FormatASCII outputs ASCII art diagram
	FormatASCII VisualizationFormat = "ascii"

	// FormatJSON outputs structured JSON
	FormatJSON VisualizationFormat = "json"
)

// Visualizer generates visual representations of graphs
type Visualizer struct {
	format  VisualizationFormat
	options VisualizerOptions
}

// VisualizerOptions contains visualization options
type VisualizerOptions struct {
	// ShowDurations includes operation durations
	ShowDurations bool

	// ShowRisk includes risk levels
	ShowRisk bool

	// HighlightCriticalPath highlights the critical path
	HighlightCriticalPath bool

	// ShowLevels shows dependency levels
	ShowLevels bool

	// CompactMode reduces visual clutter
	CompactMode bool

	// ColorScheme defines color scheme (for DOT/Mermaid)
	ColorScheme string // "default", "bw", "colorblind"
}

// NewVisualizer creates a new visualizer
func NewVisualizer(format VisualizationFormat, options VisualizerOptions) *Visualizer {
	return &Visualizer{
		format:  format,
		options: options,
	}
}

// Visualize generates a visualization of the graph
func (v *Visualizer) Visualize(graph *Graph) (string, error) {
	if graph == nil {
		return "", fmt.Errorf("graph cannot be nil")
	}

	switch v.format {
	case FormatDOT:
		return v.visualizeDOT(graph)
	case FormatMermaid:
		return v.visualizeMermaid(graph)
	case FormatASCII:
		return v.visualizeASCII(graph)
	case FormatJSON:
		return v.visualizeJSON(graph)
	default:
		return "", fmt.Errorf("unknown visualization format: %s", v.format)
	}
}

// visualizeDOT generates Graphviz DOT format
func (v *Visualizer) visualizeDOT(graph *Graph) (string, error) {
	var buf bytes.Buffer

	// Graph header
	buf.WriteString("digraph G {\n")
	buf.WriteString("  rankdir=LR;\n")
	buf.WriteString("  node [shape=box, style=rounded];\n\n")

	// Define color scheme
	criticalColor := "red"
	normalColor := "black"
	softDepColor := "gray"

	if v.options.ColorScheme == "bw" {
		criticalColor = "black"
		softDepColor = "gray"
	}

	// Add nodes
	for _, node := range graph.Nodes {
		label := v.buildNodeLabel(node)
		color := normalColor
		style := "rounded"

		if node.IsCritical && v.options.HighlightCriticalPath {
			color = criticalColor
			style = "rounded,bold"
		}

		buf.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", color=%s, style=\"%s\"];\n",
			node.ID, label, color, style))
	}

	buf.WriteString("\n")

	// Add edges
	for _, edges := range graph.Edges {
		for _, edge := range edges {
			style := "solid"
			color := normalColor

			if edge.Type == DependencyTypeSoft {
				style = "dashed"
				color = softDepColor
			}

			if edge.IsCritical && v.options.HighlightCriticalPath {
				color = criticalColor
			}

			label := ""
			if !v.options.CompactMode && edge.Reason != "" {
				label = fmt.Sprintf(" [label=\"%s\"]", edge.Reason)
			}

			buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [style=%s, color=%s%s];\n",
				edge.From, edge.To, style, color, label))
		}
	}

	// Add level-based ranking if requested
	if v.options.ShowLevels {
		buf.WriteString("\n  // Level-based ranking\n")
		levelMap := make(map[int][]string)
		for _, node := range graph.Nodes {
			levelMap[node.Level] = append(levelMap[node.Level], node.ID)
		}

		levels := make([]int, 0, len(levelMap))
		for level := range levelMap {
			levels = append(levels, level)
		}
		sort.Ints(levels)

		for _, level := range levels {
			nodes := levelMap[level]
			buf.WriteString(fmt.Sprintf("  { rank=same; "))
			for _, nodeID := range nodes {
				buf.WriteString(fmt.Sprintf("\"%s\"; ", nodeID))
			}
			buf.WriteString("}\n")
		}
	}

	buf.WriteString("}\n")

	return buf.String(), nil
}

// visualizeMermaid generates Mermaid diagram format
func (v *Visualizer) visualizeMermaid(graph *Graph) (string, error) {
	var buf bytes.Buffer

	buf.WriteString("graph LR\n")

	// Add nodes with styling
	for _, node := range graph.Nodes {
		label := v.buildNodeLabel(node)
		shape := "[]" // Rectangle

		if node.IsCritical && v.options.HighlightCriticalPath {
			shape = "{}" // Hexagon for critical nodes
		}

		nodeID := sanitizeMermaidID(node.ID)
		buf.WriteString(fmt.Sprintf("  %s%s%s\n",
			nodeID, shape[0:1], label))
		buf.WriteString(fmt.Sprintf("%s\n", shape[1:2]))
	}

	buf.WriteString("\n")

	// Add edges
	for _, edges := range graph.Edges {
		for _, edge := range edges {
			fromID := sanitizeMermaidID(edge.From)
			toID := sanitizeMermaidID(edge.To)

			arrow := "-->"
			if edge.Type == DependencyTypeSoft {
				arrow = "-..->"
			}

			label := ""
			if !v.options.CompactMode && edge.Reason != "" {
				label = fmt.Sprintf("|%s|", edge.Reason)
			}

			buf.WriteString(fmt.Sprintf("  %s %s%s %s\n",
				fromID, arrow, label, toID))
		}
	}

	// Add styling for critical path
	if v.options.HighlightCriticalPath {
		buf.WriteString("\n  %% Critical path styling\n")
		for _, node := range graph.Nodes {
			if node.IsCritical {
				nodeID := sanitizeMermaidID(node.ID)
				buf.WriteString(fmt.Sprintf("  style %s fill:#ffcccc,stroke:#ff0000,stroke-width:2px\n", nodeID))
			}
		}
	}

	return buf.String(), nil
}

// visualizeASCII generates ASCII art diagram
func (v *Visualizer) visualizeASCII(graph *Graph) (string, error) {
	var buf bytes.Buffer

	// Compute levels if not already done
	_ = graph.ComputeLevels()

	// Group nodes by level
	levelMap := make(map[int][]*Node)
	maxLevel := 0
	for _, node := range graph.Nodes {
		levelMap[node.Level] = append(levelMap[node.Level], node)
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}

	// Header
	buf.WriteString("Dependency Graph (ASCII)\n")
	buf.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Display by level
	for level := 0; level <= maxLevel; level++ {
		nodes := levelMap[level]
		if len(nodes) == 0 {
			continue
		}

		buf.WriteString(fmt.Sprintf("Level %d:\n", level))

		for _, node := range nodes {
			marker := "  "
			if node.IsCritical && v.options.HighlightCriticalPath {
				marker = "* "
			}

			label := v.buildNodeLabel(node)
			buf.WriteString(fmt.Sprintf("%s[%s]\n", marker, label))

			// Show dependencies
			deps := graph.GetDependencies(node.ID)
			if len(deps) > 0 && !v.options.CompactMode {
				buf.WriteString("    └─ depends on: ")
				buf.WriteString(strings.Join(deps, ", "))
				buf.WriteString("\n")
			}
		}

		buf.WriteString("\n")
	}

	// Legend
	if v.options.HighlightCriticalPath {
		buf.WriteString("Legend:\n")
		buf.WriteString("  * = Critical path node\n")
	}

	// Summary statistics
	buf.WriteString("\nStatistics:\n")
	buf.WriteString(fmt.Sprintf("  Total nodes: %d\n", graph.NodeCount()))
	buf.WriteString(fmt.Sprintf("  Total edges: %d\n", graph.EdgeCount()))
	buf.WriteString(fmt.Sprintf("  Max level: %d\n", maxLevel))

	if len(graph.CriticalPath) > 0 {
		buf.WriteString(fmt.Sprintf("  Critical path length: %d nodes\n", len(graph.CriticalPath)))
		buf.WriteString(fmt.Sprintf("  Critical path duration: %v\n", graph.TotalDuration))
	}

	return buf.String(), nil
}

// visualizeJSON generates structured JSON output
func (v *Visualizer) visualizeJSON(graph *Graph) (string, error) {
	// Create a visualization-friendly structure
	vis := struct {
		Nodes []*NodeVis `json:"nodes"`
		Edges []*EdgeVis `json:"edges"`
		Meta  *MetaVis   `json:"meta"`
	}{
		Nodes: make([]*NodeVis, 0, len(graph.Nodes)),
		Edges: make([]*EdgeVis, 0),
		Meta:  &MetaVis{},
	}

	// Add nodes
	for _, node := range graph.Nodes {
		nodeVis := &NodeVis{
			ID:         node.ID,
			Name:       node.Name,
			Type:       string(node.ResourceType),
			Level:      node.Level,
			IsCritical: node.IsCritical,
		}

		if v.options.ShowDurations {
			nodeVis.Duration = node.Properties.EstimatedDuration.String()
		}

		if v.options.ShowRisk {
			nodeVis.RiskLevel = string(node.Properties.RiskLevel)
		}

		vis.Nodes = append(vis.Nodes, nodeVis)
	}

	// Add edges
	for _, edges := range graph.Edges {
		for _, edge := range edges {
			edgeVis := &EdgeVis{
				From:       edge.From,
				To:         edge.To,
				Type:       string(edge.Type),
				IsCritical: edge.IsCritical,
			}

			if !v.options.CompactMode {
				edgeVis.Reason = edge.Reason
			}

			vis.Edges = append(vis.Edges, edgeVis)
		}
	}

	// Add metadata
	vis.Meta.NodeCount = len(vis.Nodes)
	vis.Meta.EdgeCount = len(vis.Edges)
	vis.Meta.MaxLevel = graph.MaxLevel

	if len(graph.CriticalPath) > 0 {
		vis.Meta.CriticalPath = graph.CriticalPath
		vis.Meta.CriticalPathDuration = graph.TotalDuration.String()
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(vis, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}

// buildNodeLabel builds a label for a node
func (v *Visualizer) buildNodeLabel(node *Node) string {
	parts := make([]string, 0)

	// Name
	name := node.Name
	if name == "" {
		name = node.ID
	}
	parts = append(parts, name)

	// Duration
	if v.options.ShowDurations && node.Properties.EstimatedDuration > 0 {
		parts = append(parts, fmt.Sprintf("(%v)", node.Properties.EstimatedDuration))
	}

	// Risk level
	if v.options.ShowRisk && node.Properties.RiskLevel != "" {
		parts = append(parts, fmt.Sprintf("[%s]", node.Properties.RiskLevel))
	}

	// Level
	if v.options.ShowLevels {
		parts = append(parts, fmt.Sprintf("L%d", node.Level))
	}

	return strings.Join(parts, " ")
}

// sanitizeMermaidID sanitizes node IDs for Mermaid
func sanitizeMermaidID(id string) string {
	// Replace special characters with underscores
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, "/", "_")
	return id
}

// NodeVis represents a node in JSON visualization
type NodeVis struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Level      int    `json:"level"`
	IsCritical bool   `json:"isCritical"`
	Duration   string `json:"duration,omitempty"`
	RiskLevel  string `json:"riskLevel,omitempty"`
}

// EdgeVis represents an edge in JSON visualization
type EdgeVis struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	IsCritical bool   `json:"isCritical"`
	Reason     string `json:"reason,omitempty"`
}

// MetaVis contains metadata for JSON visualization
type MetaVis struct {
	NodeCount            int      `json:"nodeCount"`
	EdgeCount            int      `json:"edgeCount"`
	MaxLevel             int      `json:"maxLevel"`
	CriticalPath         []string `json:"criticalPath,omitempty"`
	CriticalPathDuration string   `json:"criticalPathDuration,omitempty"`
}

// VisualizeSchedule visualizes a schedule (stages)
func VisualizeSchedule(schedule *Schedule, format VisualizationFormat) (string, error) {
	if schedule == nil {
		return "", fmt.Errorf("schedule cannot be nil")
	}

	var buf bytes.Buffer

	switch format {
	case FormatASCII:
		buf.WriteString("Execution Schedule\n")
		buf.WriteString(strings.Repeat("=", 60) + "\n\n")

		for i, stage := range schedule.Stages {
			buf.WriteString(fmt.Sprintf("Stage %d (%d operations in parallel):\n", i+1, len(stage)))

			// Calculate stage duration (max of all ops in stage)
			stageDuration := stage[0].Properties.EstimatedDuration
			for _, node := range stage {
				if node.Properties.EstimatedDuration > stageDuration {
					stageDuration = node.Properties.EstimatedDuration
				}
			}

			buf.WriteString(fmt.Sprintf("  Duration: %v\n", stageDuration))

			for _, node := range stage {
				marker := "  "
				if node.IsCritical {
					marker = "* "
				}
				buf.WriteString(fmt.Sprintf("  %s- %s (%v)\n", marker, node.Name, node.Properties.EstimatedDuration))
			}

			buf.WriteString("\n")
		}

		buf.WriteString(fmt.Sprintf("Total estimated duration: %v\n", schedule.EstimatedDuration))
		buf.WriteString(fmt.Sprintf("Strategy: %s\n", schedule.Strategy))

		return buf.String(), nil

	case FormatJSON:
		data, err := json.MarshalIndent(schedule, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal schedule: %w", err)
		}
		return string(data), nil

	default:
		return "", fmt.Errorf("unsupported format for schedule visualization: %s", format)
	}
}

// CompareVisualization generates a side-by-side comparison visualization
func CompareVisualization(graph1, graph2 *Graph, label1, label2 string) (string, error) {
	var buf bytes.Buffer

	buf.WriteString("Graph Comparison\n")
	buf.WriteString(strings.Repeat("=", 80) + "\n\n")

	buf.WriteString(fmt.Sprintf("%-40s | %s\n", label1, label2))
	buf.WriteString(strings.Repeat("-", 80) + "\n")

	buf.WriteString(fmt.Sprintf("%-40s | %s\n",
		fmt.Sprintf("Nodes: %d", graph1.NodeCount()),
		fmt.Sprintf("Nodes: %d", graph2.NodeCount())))

	buf.WriteString(fmt.Sprintf("%-40s | %s\n",
		fmt.Sprintf("Edges: %d", graph1.EdgeCount()),
		fmt.Sprintf("Edges: %d", graph2.EdgeCount())))

	if len(graph1.CriticalPath) > 0 && len(graph2.CriticalPath) > 0 {
		buf.WriteString(fmt.Sprintf("%-40s | %s\n",
			fmt.Sprintf("Critical Path: %d nodes", len(graph1.CriticalPath)),
			fmt.Sprintf("Critical Path: %d nodes", len(graph2.CriticalPath))))

		buf.WriteString(fmt.Sprintf("%-40s | %s\n",
			fmt.Sprintf("Duration: %v", graph1.TotalDuration),
			fmt.Sprintf("Duration: %v", graph2.TotalDuration)))
	}

	return buf.String(), nil
}
