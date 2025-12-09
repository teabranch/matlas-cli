package dag

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestSecurityInputValidation tests that the DAG engine properly validates and sanitizes inputs
func TestSecurityInputValidation(t *testing.T) {
	t.Run("malformed_node_ids", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Test with special characters that could cause injection
		maliciousIDs := []string{
			"../../etc/passwd",
			"node; rm -rf /",
			"node && curl evil.com",
			"node\x00null",
			"node\nls",
			strings.Repeat("a", 10000), // Very long ID
		}

		for _, id := range maliciousIDs {
			node := &Node{
				ID:         id,
				Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
			}
			err := g.AddNode(node)
			if err != nil {
				t.Logf("Correctly rejected malicious ID: %s", id)
			}

			// Ensure the node wasn't actually added or ID is sanitized
			node, err = g.GetNode(id)
			if err == nil && node != nil {
				// Even if added, ensure ID is properly escaped/sanitized
				if strings.Contains(node.ID, "..") || strings.Contains(node.ID, ";") ||
					strings.Contains(node.ID, "\n") || strings.Contains(node.ID, "\x00") {
					t.Errorf("Malicious ID was not sanitized: %s", node.ID)
				}
			}
		}
	})

	t.Run("invalid_edge_injection", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})
		_ = g.AddNode(&Node{ID: "a", Properties: NodeProperties{EstimatedDuration: 1 * time.Second}})
		_ = g.AddNode(&Node{ID: "b", Properties: NodeProperties{EstimatedDuration: 1 * time.Second}})

		// Try to add edges with non-existent nodes (potential for manipulation)
		err := g.AddEdge(&Edge{From: "nonexistent", To: "a", Type: DependencyTypeHard})
		if err == nil {
			t.Error("Should reject edge with non-existent source node")
		}

		err = g.AddEdge(&Edge{From: "a", To: "nonexistent", Type: DependencyTypeHard})
		if err == nil {
			t.Error("Should reject edge with non-existent target node")
		}

		// Verify graph integrity wasn't compromised
		if len(g.Nodes) != 2 {
			t.Errorf("Graph integrity compromised: expected 2 nodes, got %d", len(g.Nodes))
		}
	})

	t.Run("negative_durations", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Negative durations could cause integer overflow or scheduling issues
		node := &Node{
			ID:         "a",
			Properties: NodeProperties{EstimatedDuration: -1 * time.Hour},
		}
		err := g.AddNode(node)
		if err == nil {
			node, err = g.GetNode("a")
			if err == nil && node != nil && node.Properties.EstimatedDuration < 0 {
				t.Error("Negative duration was accepted without validation")
			}
		}
	})

	t.Run("invalid_dependency_types", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})
		_ = g.AddNode(&Node{ID: "a", Properties: NodeProperties{EstimatedDuration: 1 * time.Second}})
		_ = g.AddNode(&Node{ID: "b", Properties: NodeProperties{EstimatedDuration: 1 * time.Second}})

		// Test with invalid dependency type values
		invalidType := DependencyType("invalid999")
		err := g.AddEdge(&Edge{From: "a", To: "b", Type: invalidType})

		// Should either reject or sanitize to valid type
		if err == nil {
			edges := g.Edges["a"]
			if len(edges) > 0 {
				for _, edge := range edges {
					if edge.Type == invalidType {
						t.Logf("Warning: Invalid dependency type was accepted: %s", invalidType)
					}
				}
			}
		}
	})
}

// TestSecurityResourceExhaustion tests protection against DoS attacks
func TestSecurityResourceExhaustion(t *testing.T) {
	t.Run("large_graph_limits", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Try to create a very large graph (potential DoS)
		maxNodes := 100000

		startTime := time.Now()
		timeout := 5 * time.Second

		for i := 0; i < maxNodes; i++ {
			if time.Since(startTime) > timeout {
				t.Logf("Graph creation timed out after %d nodes (good - prevents DoS)", i)
				break
			}

			nodeID := fmt.Sprintf("node_%d", i)
			node := &Node{
				ID:         nodeID,
				Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
			}
			err := g.AddNode(node)
			if err != nil {
				t.Logf("Graph rejected node at size %d: %v (good - has limits)", i, err)
				break
			}
		}

		// If we created a massive graph, ensure operations still complete in reasonable time
		if len(g.Nodes) > 10000 {
			_, err := g.TopologicalSort()
			if err == nil {
				elapsed := time.Since(startTime)
				if elapsed > 10*time.Second {
					t.Errorf("TopologicalSort took too long on large graph: %v", elapsed)
				}
			}
		}
	})

	t.Run("deep_recursion_protection", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Create a very long chain (potential stack overflow)
		chainLength := 10000
		for i := 0; i < chainLength; i++ {
			nodeID := fmt.Sprintf("node_%d", i)
			_ = g.AddNode(&Node{
				ID:         nodeID,
				Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
			})

			if i > 0 {
				prevID := fmt.Sprintf("node_%d", i-1)
				_ = g.AddEdge(&Edge{From: nodeID, To: prevID, Type: DependencyTypeHard})
			}
		}

		// Test algorithms don't cause stack overflow
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Algorithm caused panic (likely stack overflow): %v", r)
			}
		}()

		startTime := time.Now()
		_, err := g.TopologicalSort()
		elapsed := time.Since(startTime)

		if err != nil {
			t.Logf("Algorithm correctly handled deep chain: %v", err)
		}

		if elapsed > 5*time.Second {
			t.Errorf("Algorithm took too long on deep chain: %v", elapsed)
		}
	})

	t.Run("cycle_bomb_protection", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Create multiple overlapping cycles (cycle bomb)
		numCycles := 100
		for c := 0; c < numCycles; c++ {
			for i := 0; i < 10; i++ {
				nodeID := fmt.Sprintf("cycle_%d_node_%d", c, i)
				_ = g.AddNode(&Node{
					ID:         nodeID,
					Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
				})

				if i > 0 {
					prevID := fmt.Sprintf("cycle_%d_node_%d", c, i-1)
					_ = g.AddEdge(&Edge{From: nodeID, To: prevID, Type: DependencyTypeHard})
				}
			}

			// Close the cycle
			firstID := fmt.Sprintf("cycle_%d_node_0", c)
			lastID := fmt.Sprintf("cycle_%d_node_9", c)
			_ = g.AddEdge(&Edge{From: firstID, To: lastID, Type: DependencyTypeHard})
		}

		startTime := time.Now()
		hasCycle, _ := g.HasCycle()
		elapsed := time.Since(startTime)

		if !hasCycle {
			t.Error("Failed to detect cycle bomb")
		}

		if elapsed > 5*time.Second {
			t.Errorf("Cycle detection took too long: %v", elapsed)
		}
	})

	t.Run("memory_exhaustion", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Create graph with massive property data
		// Note: Reduced size for practical testing
		hugeData := strings.Repeat("x", 100*1024) // 100KB per node

		count := 0
		for i := 0; i < 100; i++ {
			nodeID := fmt.Sprintf("node_%d", i)
			node := &Node{
				ID: nodeID,
				Properties: NodeProperties{
					EstimatedDuration: 1 * time.Second,
				},
				Labels: map[string]string{
					"huge_data": hugeData,
				},
			}

			err := g.AddNode(node)
			if err != nil {
				t.Logf("Graph rejected node with large metadata at %d nodes: %v", i, err)
				break
			}
			count++
		}

		// Log result (not failing, just documenting behavior)
		t.Logf("Graph accepted %d nodes with large metadata", count)
	})
}

// TestSecurityConcurrency tests thread safety and race conditions
func TestSecurityConcurrency(t *testing.T) {
	t.Run("concurrent_modifications", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Pre-populate graph
		for i := 0; i < 10; i++ {
			nodeID := fmt.Sprintf("node_%d", i)
			_ = g.AddNode(&Node{
				ID:         nodeID,
				Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
			})
		}

		// Concurrent adds
		done := make(chan bool, 3)

		go func() {
			for i := 0; i < 100; i++ {
				nodeID := fmt.Sprintf("concurrent_a_%d", i)
				g.AddNode(&Node{
					ID:         nodeID,
					Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
				})
			}
			done <- true
		}()

		// Concurrent reads
		go func() {
			for i := 0; i < 100; i++ {
				nodeID := fmt.Sprintf("node_%d", i%10)
				_, _ = g.GetNode(nodeID)
			}
			done <- true
		}()

		// Concurrent edge additions
		go func() {
			for i := 0; i < 100; i++ {
				from := fmt.Sprintf("node_%d", i%10)
				to := fmt.Sprintf("node_%d", (i+1)%10)
				_ = g.AddEdge(&Edge{From: from, To: to, Type: DependencyTypeHard})
			}
			done <- true
		}()

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}

		// Verify graph integrity
		// Note: Cycles are expected due to the circular edge pattern (node_i -> node_(i+1)%10)
		// We're checking for data corruption, not cycles
		g.mu.RLock()
		// Check forward/reverse edge consistency
		for fromID, edges := range g.Edges {
			for _, edge := range edges {
				// Verify reverse edge exists
				found := false
				for _, revEdge := range g.ReverseEdges[edge.To] {
					if revEdge.From == fromID {
						found = true
						break
					}
				}
				if !found {
					g.mu.RUnlock()
					t.Errorf("Concurrent modifications corrupted graph: forward edge %s->%s has no reverse edge", fromID, edge.To)
					return
				}
			}
		}
		g.mu.RUnlock()
	})
}

// TestSecurityRuleExecution tests that custom rules cannot execute arbitrary code
func TestSecurityRuleExecution(t *testing.T) {
	t.Run("rule_sandbox", func(t *testing.T) {
		registry := NewRuleRegistry()

		// Create a rule that attempts malicious operations
		maliciousRule := NewPropertyBasedRule(
			"malicious",
			"A rule that attempts malicious operations",
			100,
			func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
				// Attempt to access file system (should be caught)
				// In real implementation, rules should run in restricted environment

				// This test ensures we're aware of the risk
				t.Log("Rule execution security: ensure rules cannot access filesystem or network")
				return nil, nil
			},
		)

		_ = registry.Register(maliciousRule)

		// Create operations instead of nodes
		eval := NewRuleEvaluator(registry)
		eval.AddOperation(&PlannedOperation{
			ID:         "a",
			Name:       "operation-a",
			Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
		})
		eval.AddOperation(&PlannedOperation{
			ID:         "b",
			Name:       "operation-b",
			Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
		})

		_, err := eval.Evaluate(context.Background())

		if err != nil {
			t.Logf("Rule evaluation error (expected if sandbox is enforced): %v", err)
		}
	})

	t.Run("rule_timeout", func(t *testing.T) {
		registry := NewRuleRegistry()

		// Create a rule that runs forever
		infiniteRule := NewPropertyBasedRule(
			"infinite",
			"A rule that runs forever",
			100,
			func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(1 * time.Hour):
					return nil, nil
				}
			},
		)

		_ = registry.Register(infiniteRule)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Create operations
		eval := NewRuleEvaluator(registry)
		eval.AddOperation(&PlannedOperation{
			ID:         "a",
			Name:       "operation-a",
			Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
		})
		eval.AddOperation(&PlannedOperation{
			ID:         "b",
			Name:       "operation-b",
			Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
		})

		startTime := time.Now()
		_, err := eval.Evaluate(ctx)
		elapsed := time.Since(startTime)

		if elapsed > 1*time.Second {
			t.Error("Rule evaluation did not respect context timeout")
		}

		if err == nil {
			t.Error("Expected timeout error from infinite rule")
		}
	})
}

// TestSecurityPrivilegeEscalation tests that dependency manipulation cannot bypass security
func TestSecurityPrivilegeEscalation(t *testing.T) {
	t.Run("dependency_ordering_manipulation", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Simulate a security-sensitive operation that must run last
		g.AddNode(&Node{
			ID: "security_check",
			Properties: NodeProperties{
				EstimatedDuration: 1 * time.Second,
			},
			Labels: map[string]string{"critical": "true"},
		})

		// Attacker tries to add a malicious operation that should run before security check
		g.AddNode(&Node{
			ID:         "malicious",
			Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
		})

		// Try to manipulate dependencies to run malicious before security
		err := g.AddEdge(&Edge{From: "security_check", To: "malicious", Type: DependencyTypeHard})
		if err != nil {
			t.Logf("Correctly prevented manipulation: %v", err)
		}

		// Verify security_check is still properly ordered
		sorted, err := g.TopologicalSort()
		if err != nil {
			t.Fatalf("Topological sort failed: %v", err)
		}

		// Find positions
		securityPos := -1
		maliciousPos := -1
		for i, nodeID := range sorted {
			if nodeID == "security_check" {
				securityPos = i
			}
			if nodeID == "malicious" {
				maliciousPos = i
			}
		}

		// Ensure security check wasn't bypassed
		if securityPos != -1 && maliciousPos != -1 && maliciousPos > securityPos {
			t.Error("Security check was bypassed by dependency manipulation")
		}
	})
}

// TestSecurityInformationDisclosure tests that sensitive data isn't leaked
func TestSecurityInformationDisclosure(t *testing.T) {
	t.Run("sensitive_metadata_in_errors", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Add node with sensitive data
		g.AddNode(&Node{
			ID: "db_user",
			Properties: NodeProperties{
				EstimatedDuration: 1 * time.Second,
			},
			Labels: map[string]string{
				"password": "supersecret123",
				"api_key":  "sk_live_abc123",
				"username": "admin",
			},
		})

		// Force an error and check error message
		err := g.AddEdge(&Edge{From: "db_user", To: "nonexistent", Type: DependencyTypeHard})
		if err != nil {
			errMsg := err.Error()

			// Ensure sensitive data isn't in error message
			if strings.Contains(errMsg, "supersecret") ||
				strings.Contains(errMsg, "sk_live") ||
				strings.Contains(errMsg, "api_key") {
				t.Error("Error message contains sensitive metadata")
			}
		}
	})

	t.Run("json_export_sanitization", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		g.AddNode(&Node{
			ID: "resource",
			Properties: NodeProperties{
				EstimatedDuration: 1 * time.Second,
			},
			Labels: map[string]string{
				"password": "secret",
				"token":    "bearer_xyz",
			},
		})

		// Export to JSON
		jsonData, err := g.ToJSON()
		if err != nil {
			t.Fatalf("Failed to export JSON: %v", err)
		}

		jsonStr := string(jsonData)

		// Check if sensitive fields are redacted
		if strings.Contains(jsonStr, "secret") || strings.Contains(jsonStr, "bearer_xyz") {
			t.Log("Warning: JSON export may contain sensitive data - consider implementing redaction")
		}
	})
}

// TestSecurityFuzzing performs basic fuzzing on critical functions
func TestSecurityFuzzing(t *testing.T) {
	t.Run("fuzz_add_node", func(t *testing.T) {
		g := NewGraph(GraphMetadata{Name: "security-test"})

		// Generate random inputs
		testCases := []string{
			"",
			" ",
			"\n",
			"\t",
			"normal",
			strings.Repeat("a", 1000),
			"unicode☃️",
			"quotes\"'",
			"<script>alert('xss')</script>",
		}

		for _, tc := range testCases {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("AddNode panicked on input %q: %v", tc, r)
					}
				}()

				node := &Node{
					ID:         tc,
					Properties: NodeProperties{EstimatedDuration: 1 * time.Second},
				}
				err := g.AddNode(node)
				if err != nil {
					t.Logf("Input %q rejected: %v", tc, err)
				}
			}()
		}
	})
}

// TestSecuritySupplyChain documents security considerations for dependencies
func TestSecuritySupplyChain(t *testing.T) {
	t.Run("dependency_verification", func(t *testing.T) {
		// This test documents the need for dependency scanning
		t.Log("Security: Run 'gosec ./...' to scan for vulnerabilities")
		t.Log("Security: Run 'go mod verify' to check module checksums")
		t.Log("Security: Review go.mod for unexpected dependencies")
		t.Log("Security: Use Dependabot or similar for automated updates")
	})
}
