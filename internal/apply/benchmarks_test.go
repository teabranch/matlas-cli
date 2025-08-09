package apply

import (
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// BenchmarkInMemoryStateCache_Get benchmarks cache get operations
func BenchmarkInMemoryStateCache_Get(b *testing.B) {
	cache := NewInMemoryStateCache(1000, 1*time.Hour)
	defer cache.Stop()

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		projectID := generateProjectID(i)
		state := createBenchmarkProjectState(projectID)
		cache.Set(projectID, state, 0)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		projectID := generateProjectID(i % 100)
		cache.Get(projectID)
	}
}

// BenchmarkInMemoryStateCache_Set benchmarks cache set operations
func BenchmarkInMemoryStateCache_Set(b *testing.B) {
	cache := NewInMemoryStateCache(1000, 1*time.Hour)
	defer cache.Stop()

	// Create test states
	states := make([]*ProjectState, b.N)
	for i := 0; i < b.N; i++ {
		states[i] = createBenchmarkProjectState(generateProjectID(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		projectID := generateProjectID(i)
		cache.Set(projectID, states[i], 0)
	}
}

// BenchmarkInMemoryStateCache_Stats benchmarks cache statistics collection
func BenchmarkInMemoryStateCache_Stats(b *testing.B) {
	cache := NewInMemoryStateCache(100, 1*time.Hour)
	defer cache.Stop()

	// Pre-populate cache with some data
	for i := 0; i < 50; i++ {
		projectID := generateProjectID(i)
		state := createBenchmarkProjectState(projectID)
		cache.Set(projectID, state, 0)
	}

	// Perform some operations to generate stats
	for i := 0; i < 20; i++ {
		cache.Get(generateProjectID(i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Stats()
	}
}

// BenchmarkValidation_ValidateProvider benchmarks provider validation
func BenchmarkValidation_ValidateProvider(b *testing.B) {
	result := &ValidationResult{}
	providers := []string{"AWS", "GCP", "AZURE", "INVALID_PROVIDER"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		provider := providers[i%len(providers)]
		validateProvider(provider, "test.provider", result)
		// Clear result for next iteration
		result.Errors = result.Errors[:0]
		result.Warnings = result.Warnings[:0]
	}
}

// BenchmarkValidation_ValidateInstanceSize benchmarks instance size validation
func BenchmarkValidation_ValidateInstanceSize(b *testing.B) {
	result := &ValidationResult{}
	sizes := []string{"M10", "M20", "M40", "M80", "INVALID_SIZE"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		validateInstanceSize(size, "test.instanceSize", result)
		// Clear result for next iteration
		result.Errors = result.Errors[:0]
		result.Warnings = result.Warnings[:0]
	}
}

// BenchmarkValidation_ValidateResourceName benchmarks resource name validation
func BenchmarkValidation_ValidateResourceName(b *testing.B) {
	result := &ValidationResult{}
	opts := DefaultValidatorOptions()
	names := []string{
		"valid-name",
		"valid_name",
		"cluster123",
		"invalid@name",
		"INVALID-NAME",
		"invalid name",
		"",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		name := names[i%len(names)]
		validateResourceName(name, "test.name", result, opts)
		// Clear result for next iteration
		result.Errors = result.Errors[:0]
		result.Warnings = result.Warnings[:0]
	}
}

// BenchmarkStringContains benchmarks the contains helper function
func BenchmarkStringContains(b *testing.B) {
	testStrings := []string{
		"timeout",
		"rate limit reached",
		"connection refused",
		"internal server error",
		"invalid configuration",
		"not found",
		"unauthorized",
		"service unavailable",
	}

	searchTerms := []string{
		"timeout",
		"rate",
		"connection",
		"server",
		"invalid",
		"found",
		"auth",
		"service",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		str := testStrings[i%len(testStrings)]
		term := searchTerms[i%len(searchTerms)]
		contains(str, term)
	}
}

// Helper functions for benchmark tests

func createBenchmarkProjectState(projectName string) *ProjectState {
	return &ProjectState{
		Project: &types.ProjectManifest{
			Metadata: types.ResourceMetadata{Name: projectName},
		},
		Clusters:      []types.ClusterManifest{},
		DatabaseUsers: []types.DatabaseUserManifest{},
		NetworkAccess: []types.NetworkAccessManifest{},
		Fingerprint:   generateFingerprint(projectName),
		DiscoveredAt:  time.Now(),
	}
}

func generateProjectID(i int) string {
	return "project-" + padInt(i, 4)
}

func generateFingerprint(projectName string) string {
	return "fp-" + projectName + "-" + padInt(int(time.Now().Unix())%10000, 4)
}

func padInt(i, width int) string {
	str := ""
	for i > 0 {
		str = string(rune('0'+i%10)) + str
		i /= 10
	}
	if str == "" {
		str = "0"
	}
	for len(str) < width {
		str = "0" + str
	}
	return str
}
