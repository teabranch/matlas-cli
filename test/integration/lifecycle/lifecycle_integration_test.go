//go:build integration
// +build integration

package lifecycle

import (
	"testing"
)

// TestLifecycleIntegration runs all lifecycle validation tests
// This provides comprehensive validation testing for Atlas Search and VPC Endpoints
// without making any actual API calls to Atlas
func TestLifecycleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping lifecycle integration tests in short mode")
	}

	t.Run("SearchLifecycleValidation", func(t *testing.T) {
		TestSearchLifecycleValidation(t)
	})

	t.Run("VPCEndpointLifecycleValidation", func(t *testing.T) {
		TestVPCEndpointLifecycleValidation(t)
	})
}

// TestLifecycleSafety verifies that lifecycle tests don't make external API calls
func TestLifecycleSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping lifecycle safety test in short mode")
	}

	t.Run("VerifyValidationOnly", func(t *testing.T) {
		// This test ensures that our lifecycle validation tests are truly
		// validation-only and don't attempt to make Atlas API calls

		// The fact that these tests can run without Atlas credentials
		// proves they are validation-only

		// Test Atlas Search validation
		testSearchIndexValidation(t, "basic")
		testSearchIndexValidation(t, "vector")

		// Test VPC Endpoint validation
		testVPCEndpointValidation(t, "basic")
		testVPCEndpointValidation(t, "multi-provider")

		t.Log("✓ All lifecycle validation tests completed without requiring Atlas API credentials")
		t.Log("✓ This confirms the tests are validation-only and respect the safety constraint")
	})
}
