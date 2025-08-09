package apply

import (
	"fmt"
	"sort"
	"strings"

	"github.com/teabranch/matlas-cli/internal/types"
)

// DependencyResolver handles dependency resolution for execution plans
type DependencyResolver struct {
	graph          *types.DependencyGraph
	explicitDeps   map[string][]string // Resource name -> dependencies
	automaticRules []DependencyRule
	resourceSpecs  map[string]interface{} // Resource name -> spec for extracting dependencies
}

// DependencyRule defines automatic dependency detection rules
type DependencyRule struct {
	Name        string
	Description string
	SourceKind  types.ResourceKind
	TargetKind  types.ResourceKind
	Condition   func(source, target interface{}) bool
	Priority    int  // Higher priority rules are applied first
	IsRequired  bool // Whether this dependency is required or optional
}

// DependencyInfo contains information about a specific dependency
type DependencyInfo struct {
	Source     string         `json:"source"`
	Target     string         `json:"target"`
	Type       DependencyType `json:"type"`
	Reason     string         `json:"reason"`
	IsRequired bool           `json:"isRequired"`
	Priority   int            `json:"priority"`
}

// DependencyType categorizes the type of dependency
type DependencyType string

const (
	DependencyTypeAutomatic DependencyType = "automatic" // Automatically detected
	DependencyTypeExplicit  DependencyType = "explicit"  // Explicitly defined in config
	DependencyTypeImplied   DependencyType = "implied"   // Inferred from configuration
)

// DependencyAnalysis provides detailed analysis of dependencies
type DependencyAnalysis struct {
	TotalDependencies int               `json:"totalDependencies"`
	Dependencies      []DependencyInfo  `json:"dependencies"`
	Cycles            []DependencyCycle `json:"cycles,omitempty"`
	Levels            map[string]int    `json:"levels"` // Resource -> dependency level
	CriticalPath      []string          `json:"criticalPath"`
	ParallelGroups    [][]string        `json:"parallelGroups"`
}

// DependencyCycle represents a circular dependency
type DependencyCycle struct {
	Resources []string `json:"resources"`
	Path      []string `json:"path"`
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver() *DependencyResolver {
	dr := &DependencyResolver{
		graph:         types.NewDependencyGraph(),
		explicitDeps:  make(map[string][]string),
		resourceSpecs: make(map[string]interface{}),
	}

	// Initialize automatic dependency rules
	dr.automaticRules = dr.getDefaultDependencyRules()

	return dr
}

// AddResource adds a resource to the dependency resolver
func (dr *DependencyResolver) AddResource(name string, kind types.ResourceKind, spec interface{}) {
	dr.resourceSpecs[name] = spec

	// Extract explicit dependencies from spec
	if explicitDeps := dr.extractExplicitDependencies(spec); len(explicitDeps) > 0 {
		dr.explicitDeps[name] = explicitDeps
	}

	// Create resource node
	node := &types.ResourceNode{
		Name: name,
		Kind: kind,
	}

	dr.graph.AddResource(node)
}

// ResolveDependencies resolves all dependencies and returns the analysis
func (dr *DependencyResolver) ResolveDependencies() (*DependencyAnalysis, error) {
	// Clear existing dependencies
	dr.graph = types.NewDependencyGraph()

	// Re-add all resources
	for name, spec := range dr.resourceSpecs {
		kind := dr.getResourceKind(spec)
		node := &types.ResourceNode{
			Name: name,
			Kind: kind,
		}
		dr.graph.AddResource(node)
	}

	// Apply automatic dependency rules
	if err := dr.applyAutomaticRules(); err != nil {
		return nil, fmt.Errorf("failed to apply automatic rules: %w", err)
	}

	// Apply explicit dependencies
	if err := dr.applyExplicitDependencies(); err != nil {
		return nil, fmt.Errorf("failed to apply explicit dependencies: %w", err)
	}

	// Generate analysis
	analysis, err := dr.analyzeGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze dependency graph: %w", err)
	}

	return analysis, nil
}

// applyAutomaticRules applies automatic dependency detection rules
func (dr *DependencyResolver) applyAutomaticRules() error {
	// Sort rules by priority (higher first)
	sort.Slice(dr.automaticRules, func(i, j int) bool {
		return dr.automaticRules[i].Priority > dr.automaticRules[j].Priority
	})

	for _, rule := range dr.automaticRules {
		if err := dr.applyRule(rule); err != nil {
			return fmt.Errorf("failed to apply rule %s: %w", rule.Name, err)
		}
	}

	return nil
}

// applyRule applies a single dependency rule
func (dr *DependencyResolver) applyRule(rule DependencyRule) error {
	for sourceName, sourceSpec := range dr.resourceSpecs {
		sourceKind := dr.getResourceKind(sourceSpec)
		if sourceKind != rule.SourceKind {
			continue
		}

		for targetName, targetSpec := range dr.resourceSpecs {
			if sourceName == targetName {
				continue
			}

			targetKind := dr.getResourceKind(targetSpec)
			if targetKind != rule.TargetKind {
				continue
			}

			// Check if rule condition is met
			if rule.Condition(sourceSpec, targetSpec) {
				dr.addDependency(sourceName, targetName)
			}
		}
	}

	return nil
}

// applyExplicitDependencies applies explicitly defined dependencies
func (dr *DependencyResolver) applyExplicitDependencies() error {
	for source, targets := range dr.explicitDeps {
		for _, target := range targets {
			// Validate that target resource exists
			if _, exists := dr.resourceSpecs[target]; !exists {
				return fmt.Errorf("explicit dependency target %s does not exist for resource %s", target, source)
			}

			dr.addDependency(source, target)
		}
	}

	return nil
}

// addDependency adds a dependency relationship
func (dr *DependencyResolver) addDependency(source, target string) {
	sourceNode := dr.graph.Resources[source]
	if sourceNode == nil {
		return
	}

	// Add to dependencies list if not already present
	for _, dep := range sourceNode.Dependencies {
		if dep == target {
			return // Already exists
		}
	}

	sourceNode.Dependencies = append(sourceNode.Dependencies, target)
	dr.graph.Dependencies[source] = sourceNode.Dependencies
}

// analyzeGraph performs comprehensive analysis of the dependency graph
func (dr *DependencyResolver) analyzeGraph() (*DependencyAnalysis, error) {
	analysis := &DependencyAnalysis{
		Dependencies:   make([]DependencyInfo, 0),
		Levels:         make(map[string]int),
		ParallelGroups: make([][]string, 0),
	}

	// Check for cycles
	if hasCycles, cyclePaths := dr.graph.HasCycles(); hasCycles {
		cycles := make([]DependencyCycle, 1)
		cycles[0] = DependencyCycle{
			Resources: dr.extractUniqueResources(cyclePaths),
			Path:      cyclePaths,
		}
		analysis.Cycles = cycles
		return analysis, fmt.Errorf("circular dependencies detected: %v", cyclePaths)
	}

	// Get topological order
	topoOrder, err := dr.graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to get topological order: %w", err)
	}

	// Calculate dependency levels
	analysis.Levels = dr.calculateLevels(topoOrder)

	// Build dependency info
	analysis.Dependencies = dr.buildDependencyInfo()
	analysis.TotalDependencies = len(analysis.Dependencies)

	// Find critical path
	analysis.CriticalPath = dr.findCriticalPath(analysis.Levels)

	// Group operations by level for parallel execution
	analysis.ParallelGroups = dr.groupByLevel(analysis.Levels)

	return analysis, nil
}

// calculateLevels assigns dependency levels to resources
func (dr *DependencyResolver) calculateLevels(topoOrder []string) map[string]int {
	levels := make(map[string]int)

	for _, resource := range topoOrder {
		maxDepLevel := -1

		// Find maximum level of dependencies
		for _, dep := range dr.graph.Dependencies[resource] {
			if depLevel, exists := levels[dep]; exists && depLevel > maxDepLevel {
				maxDepLevel = depLevel
			}
		}

		levels[resource] = maxDepLevel + 1
	}

	return levels
}

// buildDependencyInfo creates detailed dependency information
func (dr *DependencyResolver) buildDependencyInfo() []DependencyInfo {
	var deps []DependencyInfo

	for source, targets := range dr.graph.Dependencies {
		for _, target := range targets {
			depInfo := DependencyInfo{
				Source: source,
				Target: target,
			}

			// Determine dependency type and reason
			if dr.isExplicitDependency(source, target) {
				depInfo.Type = DependencyTypeExplicit
				depInfo.Reason = "Explicitly defined in configuration"
			} else {
				depInfo.Type = DependencyTypeAutomatic
				depInfo.Reason = dr.getAutomaticDependencyReason(source, target)
			}

			deps = append(deps, depInfo)
		}
	}

	return deps
}

// findCriticalPath identifies the longest dependency chain
func (dr *DependencyResolver) findCriticalPath(levels map[string]int) []string {
	var criticalPath []string
	var maxLevel int
	var endResource string

	// Find resource with maximum level
	for resource, level := range levels {
		if level > maxLevel {
			maxLevel = level
			endResource = resource
		}
	}

	// Build path backwards from end resource
	current := endResource
	for current != "" {
		criticalPath = append([]string{current}, criticalPath...)

		// Find dependency with highest level
		var nextResource string
		var nextLevel = -1

		for _, dep := range dr.graph.Dependencies[current] {
			if depLevel := levels[dep]; depLevel > nextLevel {
				nextLevel = depLevel
				nextResource = dep
			}
		}

		current = nextResource
	}

	return criticalPath
}

// groupByLevel groups resources by their dependency level for parallel execution
func (dr *DependencyResolver) groupByLevel(levels map[string]int) [][]string {
	levelMap := make(map[int][]string)
	maxLevel := 0

	for resource, level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
		levelMap[level] = append(levelMap[level], resource)
	}

	groups := make([][]string, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		groups[level] = levelMap[level]
	}

	return groups
}

// getDefaultDependencyRules returns the default automatic dependency rules
func (dr *DependencyResolver) getDefaultDependencyRules() []DependencyRule {
	return []DependencyRule{
		{
			Name:        "DatabaseUserDependsOnCluster",
			Description: "Database users require clusters to exist",
			SourceKind:  types.KindDatabaseUser,
			TargetKind:  types.KindCluster,
			Priority:    100,
			IsRequired:  true,
			Condition: func(source, target interface{}) bool {
				// Database users depend on clusters in the same project
				return sameProjectCondition(source, target)
			},
		},
		{
			Name:        "ProjectDependency",
			Description: "All resources depend on their project",
			SourceKind:  types.KindCluster, // Apply to all non-project resources
			TargetKind:  types.KindProject,
			Priority:    200,
			IsRequired:  true,
			Condition: func(source, target interface{}) bool {
				// Resources depend on their project
				return sameProjectCondition(source, target)
			},
		},
		{
			Name:        "NetworkAccessAfterCluster",
			Description: "Network access is typically configured after clusters",
			SourceKind:  types.KindNetworkAccess,
			TargetKind:  types.KindCluster,
			Priority:    50,
			IsRequired:  false,
			Condition: func(source, target interface{}) bool {
				// Optional dependency for better ordering
				return sameProjectCondition(source, target)
			},
		},
	}
}

// sameProjectCondition checks if two resources belong to the same project (standalone function)
func sameProjectCondition(source, target interface{}) bool {
	sourceProject := extractProjectNameFromSpec(source)
	targetProject := extractProjectNameFromSpec(target)
	return sourceProject != "" && sourceProject == targetProject
}

// extractProjectNameFromSpec extracts project name from resource spec (standalone function)
func extractProjectNameFromSpec(spec interface{}) string {
	switch s := spec.(type) {
	case *types.ClusterManifest:
		return s.Spec.ProjectName
	case types.ClusterManifest:
		return s.Spec.ProjectName
	case types.ClusterSpec:
		return s.ProjectName
	case *types.DatabaseUserManifest:
		return s.Spec.ProjectName
	case types.DatabaseUserManifest:
		return s.Spec.ProjectName
	case types.DatabaseUserSpec:
		return s.ProjectName
	case *types.NetworkAccessManifest:
		return s.Spec.ProjectName
	case types.NetworkAccessManifest:
		return s.Spec.ProjectName
	case types.NetworkAccessSpec:
		return s.ProjectName
	default:
		return ""
	}
}

// extractExplicitDependencies extracts dependsOn field from resource specs
func (dr *DependencyResolver) extractExplicitDependencies(spec interface{}) []string {
	switch s := spec.(type) {
	case *types.ClusterManifest:
		return s.Metadata.DependsOn
	case types.ClusterManifest:
		return s.Metadata.DependsOn
	case *types.DatabaseUserManifest:
		return s.Metadata.DependsOn
	case types.DatabaseUserManifest:
		return s.Metadata.DependsOn
	case *types.NetworkAccessManifest:
		return s.Metadata.DependsOn
	case types.NetworkAccessManifest:
		return s.Metadata.DependsOn
	case *types.ProjectManifest:
		return s.Metadata.DependsOn
	case types.ProjectManifest:
		return s.Metadata.DependsOn
	default:
		return []string{}
	}
}

// getResourceKind determines the resource kind from spec
func (dr *DependencyResolver) getResourceKind(spec interface{}) types.ResourceKind {
	switch spec.(type) {
	case *types.ClusterManifest, types.ClusterManifest, types.ClusterSpec:
		return types.KindCluster
	case *types.DatabaseUserManifest, types.DatabaseUserManifest, types.DatabaseUserSpec:
		return types.KindDatabaseUser
	case *types.NetworkAccessManifest, types.NetworkAccessManifest, types.NetworkAccessSpec:
		return types.KindNetworkAccess
	case *types.ProjectManifest, types.ProjectManifest, types.ProjectConfig:
		return types.KindProject
	default:
		return types.KindCluster // Default fallback
	}
}

// sameProject checks if two resources belong to the same project
func (dr *DependencyResolver) sameProject(source, target interface{}) bool {
	sourceProject := dr.extractProjectName(source)
	targetProject := dr.extractProjectName(target)
	return sourceProject != "" && sourceProject == targetProject
}

// extractProjectName extracts project name from resource spec
func (dr *DependencyResolver) extractProjectName(spec interface{}) string {
	switch s := spec.(type) {
	case *types.ClusterManifest:
		return s.Spec.ProjectName
	case types.ClusterManifest:
		return s.Spec.ProjectName
	case types.ClusterSpec:
		return s.ProjectName
	case *types.DatabaseUserManifest:
		return s.Spec.ProjectName
	case types.DatabaseUserManifest:
		return s.Spec.ProjectName
	case types.DatabaseUserSpec:
		return s.ProjectName
	case *types.NetworkAccessManifest:
		return s.Spec.ProjectName
	case types.NetworkAccessManifest:
		return s.Spec.ProjectName
	case types.NetworkAccessSpec:
		return s.ProjectName
	default:
		return ""
	}
}

// isExplicitDependency checks if a dependency was explicitly defined
func (dr *DependencyResolver) isExplicitDependency(source, target string) bool {
	if deps, exists := dr.explicitDeps[source]; exists {
		for _, dep := range deps {
			if dep == target {
				return true
			}
		}
	}
	return false
}

// getAutomaticDependencyReason returns the reason for an automatic dependency
func (dr *DependencyResolver) getAutomaticDependencyReason(source, target string) string {
	sourceKind := dr.getResourceKind(dr.resourceSpecs[source])
	targetKind := dr.getResourceKind(dr.resourceSpecs[target])

	for _, rule := range dr.automaticRules {
		if rule.SourceKind == sourceKind && rule.TargetKind == targetKind {
			if rule.Condition(dr.resourceSpecs[source], dr.resourceSpecs[target]) {
				return rule.Description
			}
		}
	}

	return "Automatic dependency detection"
}

// extractUniqueResources extracts unique resource names from a path
func (dr *DependencyResolver) extractUniqueResources(path []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, resource := range path {
		if !seen[resource] {
			seen[resource] = true
			unique = append(unique, resource)
		}
	}

	return unique
}

// VisualizeDependencies creates a text-based visualization of dependencies
func (dr *DependencyResolver) VisualizeDependencies(analysis *DependencyAnalysis) string {
	var output strings.Builder

	output.WriteString("Dependency Analysis:\n")
	output.WriteString("===================\n\n")

	// Summary
	output.WriteString(fmt.Sprintf("Total Dependencies: %d\n", analysis.TotalDependencies))
	output.WriteString(fmt.Sprintf("Dependency Levels: %d\n", len(analysis.ParallelGroups)))
	output.WriteString(fmt.Sprintf("Critical Path Length: %d\n\n", len(analysis.CriticalPath)))

	// Cycles (if any)
	if len(analysis.Cycles) > 0 {
		output.WriteString("⚠️  Circular Dependencies Detected:\n")
		for i, cycle := range analysis.Cycles {
			output.WriteString(fmt.Sprintf("  %d. %s\n", i+1, strings.Join(cycle.Path, " → ")))
		}
		output.WriteString("\n")
	}

	// Critical Path
	if len(analysis.CriticalPath) > 0 {
		output.WriteString("🔥 Critical Path:\n")
		output.WriteString("  " + strings.Join(analysis.CriticalPath, " → ") + "\n\n")
	}

	// Parallel Groups
	output.WriteString("📊 Execution Stages:\n")
	for level, resources := range analysis.ParallelGroups {
		if len(resources) == 0 {
			continue
		}
		output.WriteString(fmt.Sprintf("  Stage %d: %s\n", level, strings.Join(resources, ", ")))
	}
	output.WriteString("\n")

	// Detailed Dependencies
	output.WriteString("🔗 Dependencies:\n")
	for _, dep := range analysis.Dependencies {
		typeSymbol := "🔧" // automatic
		if dep.Type == DependencyTypeExplicit {
			typeSymbol = "📝" // explicit
		}
		output.WriteString(fmt.Sprintf("  %s %s → %s (%s)\n", typeSymbol, dep.Source, dep.Target, dep.Reason))
	}

	return output.String()
}

// ValidateDependencies validates the dependency configuration
func (dr *DependencyResolver) ValidateDependencies() error {
	analysis, err := dr.ResolveDependencies()
	if err != nil {
		return err
	}

	if len(analysis.Cycles) > 0 {
		var cycleStrings []string
		for _, cycle := range analysis.Cycles {
			cycleStrings = append(cycleStrings, strings.Join(cycle.Path, " → "))
		}
		return fmt.Errorf("circular dependencies detected: %s", strings.Join(cycleStrings, "; "))
	}

	return nil
}

// GetExecutionOrder returns resources in dependency order for execution
func (dr *DependencyResolver) GetExecutionOrder() ([]string, error) {
	return dr.graph.TopologicalSort()
}

// AddDependencyRule adds a custom dependency rule
func (dr *DependencyResolver) AddDependencyRule(rule DependencyRule) {
	dr.automaticRules = append(dr.automaticRules, rule)
}

// RemoveDependencyRule removes a dependency rule by name
func (dr *DependencyResolver) RemoveDependencyRule(name string) {
	for i, rule := range dr.automaticRules {
		if rule.Name == name {
			dr.automaticRules = append(dr.automaticRules[:i], dr.automaticRules[i+1:]...)
			break
		}
	}
}
