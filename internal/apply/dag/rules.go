package dag

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/teabranch/matlas-cli/internal/types"
)

// Rule defines the interface for dependency rules
type Rule interface {
	// Name returns the unique name of the rule
	Name() string
	
	// Description returns a human-readable description
	Description() string
	
	// Priority returns the rule priority (higher = evaluated first)
	Priority() int
	
	// Evaluate evaluates the rule for a pair of operations
	// Returns the dependency edge if the rule applies, nil otherwise
	Evaluate(ctx context.Context, from, to *PlannedOperation) (*Edge, error)
}

// PlannedOperation represents an operation being planned
type PlannedOperation struct {
	ID           string
	Name         string
	ResourceType types.ResourceKind
	ResourceName string
	Spec         interface{} // The resource specification
	Properties   NodeProperties
	
	// For conditional evaluation
	Metadata map[string]interface{}
}

// RuleRegistry manages dependency rules
type RuleRegistry struct {
	mu    sync.RWMutex
	rules map[string]Rule
}

// NewRuleRegistry creates a new rule registry
func NewRuleRegistry() *RuleRegistry {
	return &RuleRegistry{
		rules: make(map[string]Rule),
	}
}

// Register registers a new rule
func (r *RuleRegistry) Register(rule Rule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}
	
	name := rule.Name()
	if name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}
	
	if _, exists := r.rules[name]; exists {
		return fmt.Errorf("rule %s is already registered", name)
	}
	
	r.rules[name] = rule
	return nil
}

// Unregister removes a rule from the registry
func (r *RuleRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.rules[name]; !exists {
		return fmt.Errorf("rule %s is not registered", name)
	}
	
	delete(r.rules, name)
	return nil
}

// GetRule retrieves a rule by name
func (r *RuleRegistry) GetRule(name string) (Rule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	rule, exists := r.rules[name]
	return rule, exists
}

// ListRules returns all registered rules sorted by priority
func (r *RuleRegistry) ListRules() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	rules := make([]Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		rules = append(rules, rule)
	}
	
	// Sort by priority (higher first)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority() > rules[j].Priority()
	})
	
	return rules
}

// RuleEvaluator evaluates rules to build a dependency graph
type RuleEvaluator struct {
	registry   *RuleRegistry
	operations []*PlannedOperation
}

// NewRuleEvaluator creates a new rule evaluator
func NewRuleEvaluator(registry *RuleRegistry) *RuleEvaluator {
	return &RuleEvaluator{
		registry:   registry,
		operations: make([]*PlannedOperation, 0),
	}
}

// AddOperation adds an operation to be evaluated
func (e *RuleEvaluator) AddOperation(op *PlannedOperation) {
	e.operations = append(e.operations, op)
}

// AddOperations adds multiple operations
func (e *RuleEvaluator) AddOperations(ops []*PlannedOperation) {
	e.operations = append(e.operations, ops...)
}

// Evaluate evaluates all rules and builds a dependency graph
func (e *RuleEvaluator) Evaluate(ctx context.Context) (*Graph, error) {
	graph := NewGraph(GraphMetadata{
		Name: "rule-evaluated-graph",
	})
	
	// Add all operations as nodes
	for _, op := range e.operations {
		node := &Node{
			ID:           op.ID,
			Name:         op.Name,
			ResourceType: op.ResourceType,
			Properties:   op.Properties,
			Labels:       map[string]string{"resource": op.ResourceName},
		}
		
		if err := graph.AddNode(node); err != nil {
			return nil, fmt.Errorf("failed to add node %s: %w", op.ID, err)
		}
	}
	
	// Get rules sorted by priority
	rules := e.registry.ListRules()
	
	// Evaluate each rule for all operation pairs
	for _, rule := range rules {
		for _, from := range e.operations {
			for _, to := range e.operations {
				if from.ID == to.ID {
					continue
				}
				
				// Evaluate rule
				edge, err := rule.Evaluate(ctx, from, to)
				if err != nil {
					return nil, fmt.Errorf("rule %s failed for %s -> %s: %w", 
						rule.Name(), from.ID, to.ID, err)
				}
				
				// Add edge if rule applies
				if edge != nil {
					// Set edge endpoints
					edge.From = from.ID
					edge.To = to.ID
					
					// Set reason if not provided
					if edge.Reason == "" {
						edge.Reason = rule.Description()
					}
					
					// Check if edge would create a cycle
					tempGraph := graph.Clone()
					if err := tempGraph.AddEdge(edge); err == nil {
						if hasCycle, _ := tempGraph.HasCycle(); !hasCycle {
							// Safe to add
							if err := graph.AddEdge(edge); err != nil {
								// Edge might already exist, that's ok
								continue
							}
						}
					}
				}
			}
		}
	}
	
	return graph, nil
}

// BaseRule provides common functionality for rules
type BaseRule struct {
	name        string
	description string
	priority    int
}

// NewBaseRule creates a new base rule
func NewBaseRule(name, description string, priority int) BaseRule {
	return BaseRule{
		name:        name,
		description: description,
		priority:    priority,
	}
}

func (r BaseRule) Name() string        { return r.name }
func (r BaseRule) Description() string { return r.description }
func (r BaseRule) Priority() int       { return r.priority }

// ResourceKindRule is a rule that matches based on resource kinds
type ResourceKindRule struct {
	BaseRule
	fromKind  types.ResourceKind
	toKind    types.ResourceKind
	depType   DependencyType
	condition func(*PlannedOperation, *PlannedOperation) bool
}

// NewResourceKindRule creates a new resource kind-based rule
func NewResourceKindRule(
	name, description string,
	priority int,
	fromKind, toKind types.ResourceKind,
	depType DependencyType,
	condition func(*PlannedOperation, *PlannedOperation) bool,
) *ResourceKindRule {
	return &ResourceKindRule{
		BaseRule:  NewBaseRule(name, description, priority),
		fromKind:  fromKind,
		toKind:    toKind,
		depType:   depType,
		condition: condition,
	}
}

// Evaluate implements the Rule interface
func (r *ResourceKindRule) Evaluate(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
	// Check resource kinds match
	if from.ResourceType != r.fromKind || to.ResourceType != r.toKind {
		return nil, nil
	}
	
	// Check condition if provided
	if r.condition != nil && !r.condition(from, to) {
		return nil, nil
	}
	
	// Create edge
	edge := &Edge{
		Type:   r.depType,
		Weight: 1.0,
		Reason: r.Description(),
	}
	
	return edge, nil
}

// PropertyBasedRule evaluates dependencies based on resource properties
type PropertyBasedRule struct {
	BaseRule
	condition func(context.Context, *PlannedOperation, *PlannedOperation) (*Edge, error)
}

// NewPropertyBasedRule creates a new property-based rule
func NewPropertyBasedRule(
	name, description string,
	priority int,
	condition func(context.Context, *PlannedOperation, *PlannedOperation) (*Edge, error),
) *PropertyBasedRule {
	return &PropertyBasedRule{
		BaseRule:  NewBaseRule(name, description, priority),
		condition: condition,
	}
}

// Evaluate implements the Rule interface
func (r *PropertyBasedRule) Evaluate(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
	return r.condition(ctx, from, to)
}

// ConditionalRule wraps a rule with an additional runtime condition
type ConditionalRule struct {
	wrapped   Rule
	condition func(context.Context) bool
}

// NewConditionalRule creates a conditional rule wrapper
func NewConditionalRule(rule Rule, condition func(context.Context) bool) *ConditionalRule {
	return &ConditionalRule{
		wrapped:   rule,
		condition: condition,
	}
}

func (r *ConditionalRule) Name() string        { return r.wrapped.Name() + "_conditional" }
func (r *ConditionalRule) Description() string { return r.wrapped.Description() + " (conditional)" }
func (r *ConditionalRule) Priority() int       { return r.wrapped.Priority() }

// Evaluate implements the Rule interface
func (r *ConditionalRule) Evaluate(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
	// Check runtime condition first
	if !r.condition(ctx) {
		return nil, nil
	}
	
	// Evaluate wrapped rule
	return r.wrapped.Evaluate(ctx, from, to)
}

// CompositeRule combines multiple rules with AND/OR logic
type CompositeRule struct {
	BaseRule
	rules []Rule
	logic CompositeLogic
}

// CompositeLogic defines how rules are combined
type CompositeLogic int

const (
	// LogicAND requires all rules to apply
	LogicAND CompositeLogic = iota
	
	// LogicOR requires at least one rule to apply
	LogicOR
)

// NewCompositeRule creates a new composite rule
func NewCompositeRule(
	name, description string,
	priority int,
	logic CompositeLogic,
	rules ...Rule,
) *CompositeRule {
	return &CompositeRule{
		BaseRule: NewBaseRule(name, description, priority),
		rules:    rules,
		logic:    logic,
	}
}

// Evaluate implements the Rule interface
func (r *CompositeRule) Evaluate(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
	if r.logic == LogicAND {
		// All rules must apply
		var resultEdge *Edge
		for _, rule := range r.rules {
			edge, err := rule.Evaluate(ctx, from, to)
			if err != nil {
				return nil, err
			}
			if edge == nil {
				return nil, nil // One rule doesn't apply
			}
			if resultEdge == nil {
				resultEdge = edge
			}
		}
		return resultEdge, nil
	}
	
	// OR logic: at least one rule must apply
	for _, rule := range r.rules {
		edge, err := rule.Evaluate(ctx, from, to)
		if err != nil {
			return nil, err
		}
		if edge != nil {
			return edge, nil
		}
	}
	
	return nil, nil
}

// MutualExclusionRule identifies operations that cannot run in parallel
type MutualExclusionRule struct {
	BaseRule
	detector func(*PlannedOperation, *PlannedOperation) bool
}

// NewMutualExclusionRule creates a mutual exclusion rule
func NewMutualExclusionRule(
	name, description string,
	priority int,
	detector func(*PlannedOperation, *PlannedOperation) bool,
) *MutualExclusionRule {
	return &MutualExclusionRule{
		BaseRule: NewBaseRule(name, description, priority),
		detector: detector,
	}
}

// Evaluate implements the Rule interface
func (r *MutualExclusionRule) Evaluate(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
	if r.detector(from, to) {
		return &Edge{
			Type:   DependencyTypeMutualExclusion,
			Weight: 10.0, // High weight for mutual exclusion
			Reason: r.Description(),
		}, nil
	}
	return nil, nil
}
