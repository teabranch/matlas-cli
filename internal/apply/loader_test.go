package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teabranch/matlas-cli/internal/types"
)

func TestDefaultLoaderOptions(t *testing.T) {
	opts := DefaultLoaderOptions()

	if opts.StrictEnv {
		t.Error("Default StrictEnv should be false")
	}
	if opts.Debug {
		t.Error("Default Debug should be false")
	}
	if !opts.CacheEnabled {
		t.Error("Default CacheEnabled should be true")
	}
	if opts.CacheTTL != 5*time.Minute {
		t.Errorf("Default CacheTTL = %v, want %v", opts.CacheTTL, 5*time.Minute)
	}
	if !opts.AllowStdin {
		t.Error("Default AllowStdin should be true")
	}
	if opts.MaxFileSize != 10*1024*1024 {
		t.Errorf("Default MaxFileSize = %d, want %d", opts.MaxFileSize, 10*1024*1024)
	}
}

func TestNewConfigurationLoader(t *testing.T) {
	// Test with nil options
	cl := NewConfigurationLoader(nil)
	if cl.options.StrictEnv {
		t.Error("Should use default options when nil provided")
	}

	// Test with custom options
	opts := &LoaderOptions{
		StrictEnv: true,
		Debug:     true,
		Variables: map[string]string{"TEST": "value"},
	}
	cl = NewConfigurationLoader(opts)

	if !cl.processor.StrictMode {
		t.Error("Processor should inherit strict mode")
	}
	if !cl.processor.DebugMode {
		t.Error("Processor should inherit debug mode")
	}
	if cl.processor.Variables["TEST"] != "value" {
		t.Error("Processor should inherit variables")
	}
}

func TestConfigurationLoader_LoadAndProcess(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")

	content := `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: ${ENV}-project
spec:
  organizationId: ${ORG_ID}
  clusters:
    - metadata:
        name: ${ENV}-cluster
      region: ${REGION:-US_EAST_1}
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test with variables
	opts := DefaultLoaderOptions()
	opts.Variables = map[string]string{
		"ENV":    "test",
		"ORG_ID": "123456789012345678901234",
	}

	cl := NewConfigurationLoader(opts)
	result, err := cl.LoadAndProcess(tmpFile)

	if err != nil {
		t.Fatalf("LoadAndProcess failed: %v", err)
	}

	// Check substitutions
	if !strings.Contains(result.ProcessedContent, "test-project") {
		t.Error("ENV variable not substituted")
	}
	if !strings.Contains(result.ProcessedContent, "123456789012345678901234") {
		t.Error("ORG_ID variable not substituted")
	}
	if !strings.Contains(result.ProcessedContent, "US_EAST_1") {
		t.Error("Default value not applied")
	}

	// Check metadata
	if result.Source != tmpFile {
		t.Errorf("Source = %s, want %s", result.Source, tmpFile)
	}
	if len(result.Substitutions) == 0 {
		t.Error("Expected substitutions to be recorded")
	}
}

func TestConfigurationLoader_LoadApplyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")

	content := `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: ${ENV}-project
spec:
  name: test-project
  organizationId: ${ORG_ID}
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	opts := DefaultLoaderOptions()
	opts.Variables = map[string]string{
		"ENV":    "production",
		"ORG_ID": "123456789012345678901234",
	}

	cl := NewConfigurationLoader(opts)
	result, err := cl.LoadApplyConfig(tmpFile)

	if err != nil {
		t.Fatalf("LoadApplyConfig failed: %v", err)
	}

	config, ok := result.Config.(*types.ApplyConfig)
	if !ok {
		t.Fatal("Config is not ApplyConfig type")
	}

	if config.Spec.Name != "test-project" {
		t.Errorf("Config.Spec.Name = %s, want test-project", config.Spec.Name)
	}
	if config.Spec.OrganizationID != "123456789012345678901234" {
		t.Errorf("Config.Spec.OrganizationID = %s, want 123456789012345678901234", config.Spec.OrganizationID)
	}
}

func TestConfigurationLoader_LoadApplyDocument(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "document.yaml")

	content := `
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: ${ENV}-resources
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: ${ENV}-cluster
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	opts := DefaultLoaderOptions()
	opts.Variables = map[string]string{"ENV": "staging"}

	cl := NewConfigurationLoader(opts)
	result, err := cl.LoadApplyDocument(tmpFile)

	if err != nil {
		t.Fatalf("LoadApplyDocument failed: %v", err)
	}

	document, ok := result.Config.(*types.ApplyDocument)
	if !ok {
		t.Fatal("Config is not ApplyDocument type")
	}

	if document.Metadata.Name != "staging-resources" {
		t.Errorf("Document.Metadata.Name = %s, want staging-resources", document.Metadata.Name)
	}
	if len(document.Resources) != 1 {
		t.Errorf("Document.Resources length = %d, want 1", len(document.Resources))
	}
}

func TestConfigurationLoader_LoadMultiDocument(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "multi.yaml")

	content := `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: ${ENV}-project
---
apiVersion: matlas.mongodb.com/v1
kind: Cluster
metadata:
  name: ${ENV}-cluster
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	opts := DefaultLoaderOptions()
	opts.Variables = map[string]string{"ENV": "development"}

	cl := NewConfigurationLoader(opts)
	results, err := cl.LoadMultiDocument(tmpFile)

	if err != nil {
		t.Fatalf("LoadMultiDocument failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Results length = %d, want 2", len(results))
	}

	// Check that each document was processed
	for i, result := range results {
		if !strings.Contains(result.Source, "[") {
			t.Errorf("Result[%d].Source should contain document index", i)
		}
		if !strings.Contains(result.ProcessedContent, "development") {
			t.Errorf("Result[%d] should have substituted ENV variable", i)
		}
	}
}

func TestConfigurationLoader_LoadGlob(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test files
	files := []string{"config1.yaml", "config2.yaml", "other.txt"}
	for _, filename := range files {
		content := `name: ${ENV}-` + filename
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	opts := DefaultLoaderOptions()
	opts.Variables = map[string]string{"ENV": "test"}

	cl := NewConfigurationLoader(opts)

	// Test glob pattern
	pattern := filepath.Join(tmpDir, "*.yaml")
	results, err := cl.LoadGlob(pattern)

	if err != nil {
		t.Fatalf("LoadGlob failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Results length = %d, want 2", len(results))
	}

	// Verify all results have substitutions
	for _, result := range results {
		if !strings.Contains(result.ProcessedContent, "test-") {
			t.Error("Glob result should have substituted ENV variable")
		}
	}
}

func TestConfigurationLoader_LoadGlob_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewConfigurationLoader(nil)

	// Test with pattern that matches no files
	pattern := filepath.Join(tmpDir, "*.nonexistent")
	_, err := cl.LoadGlob(pattern)

	if err == nil {
		t.Error("Expected error for glob pattern with no matches")
	}
	if !strings.Contains(err.Error(), "no files match") {
		t.Errorf("Error should mention no matches, got: %v", err)
	}
}

func TestConfigurationLoader_ValidateOnly(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")

	// Create file with invalid template syntax
	content := `name: ${INVALID`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cl := NewConfigurationLoader(nil)
	result, err := cl.ValidateOnly(tmpFile)

	if err != nil {
		t.Fatalf("ValidateOnly failed: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for invalid template")
	}
}

func TestConfigurationLoader_ExtractVariables(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "vars.yaml")

	content := `
name: ${ENV}
database: ${DB_NAME:-default}
host: ${DB_HOST:?required}
port: ${DB_PORT:+5432}
`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cl := NewConfigurationLoader(nil)
	variables, err := cl.ExtractVariables(tmpFile)

	if err != nil {
		t.Fatalf("ExtractVariables failed: %v", err)
	}

	expected := []string{"ENV", "DB_NAME", "DB_HOST", "DB_PORT"}
	if len(variables) != len(expected) {
		t.Errorf("Variables length = %d, want %d", len(variables), len(expected))
	}

	// Convert to map for easier checking
	varMap := make(map[string]bool)
	for _, v := range variables {
		varMap[v] = true
	}

	for _, exp := range expected {
		if !varMap[exp] {
			t.Errorf("Expected variable %s not found", exp)
		}
	}
}

func TestConfigurationLoader_SetVariable(t *testing.T) {
	cl := NewConfigurationLoader(nil)

	cl.SetVariable("TEST_VAR", "test_value")

	if cl.processor.Variables["TEST_VAR"] != "test_value" {
		t.Error("Variable not set in processor")
	}
	if cl.options.Variables["TEST_VAR"] != "test_value" {
		t.Error("Variable not set in options")
	}
}

func TestConfigurationLoader_SetVariables(t *testing.T) {
	cl := NewConfigurationLoader(nil)

	vars := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
	}
	cl.SetVariables(vars)

	for name, expected := range vars {
		if cl.processor.Variables[name] != expected {
			t.Errorf("Variable %s not set correctly in processor", name)
		}
		if cl.options.Variables[name] != expected {
			t.Errorf("Variable %s not set correctly in options", name)
		}
	}
}

func TestConfigurationLoader_Cache(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "cached.yaml")

	content := `name: ${ENV}`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	opts := DefaultLoaderOptions()
	opts.Variables = map[string]string{"ENV": "cached"}
	opts.Debug = true // Enable debug to see cache messages

	cl := NewConfigurationLoader(opts)

	// First load should process template
	result1, err := cl.LoadAndProcess(tmpFile)
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	// Second load should use cache
	result2, err := cl.LoadAndProcess(tmpFile)
	if err != nil {
		t.Fatalf("Second load failed: %v", err)
	}

	// Results should be identical
	if result1.ProcessedContent != result2.ProcessedContent {
		t.Error("Cached result differs from original")
	}

	// Check cache stats
	stats := cl.GetCacheStats()
	if stats["entries"].(int) == 0 {
		t.Error("Cache should have entries")
	}
}

func TestConfigurationLoader_CacheClear(t *testing.T) {
	cl := NewConfigurationLoader(nil)

	// Set a variable to populate cache
	cl.SetVariable("TEST", "value")

	stats := cl.GetCacheStats()
	if stats["entries"].(int) != 0 {
		t.Error("Cache should be empty after variable change")
	}

	// Clear cache explicitly
	cl.ClearCache()

	stats = cl.GetCacheStats()
	if stats["entries"].(int) != 0 {
		t.Error("Cache should be empty after clear")
	}
}

func TestConfigurationLoader_StrictMode(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "strict.yaml")

	content := `name: ${UNDEFINED_VAR}`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	opts := DefaultLoaderOptions()
	opts.StrictEnv = true

	cl := NewConfigurationLoader(opts)

	// Should fail in strict mode with undefined variable
	_, err = cl.LoadAndProcess(tmpFile)
	if err == nil {
		t.Error("Expected error in strict mode with undefined variable")
	}
	if !strings.Contains(err.Error(), "template processing failed") {
		t.Errorf("Error should mention template processing, got: %v", err)
	}
}

func TestConfigurationLoader_MaxFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.yaml")

	// Create a large file
	largeContent := strings.Repeat("data: value\n", 1000)
	err := os.WriteFile(tmpFile, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	opts := DefaultLoaderOptions()
	opts.MaxFileSize = 100 // Very small limit

	cl := NewConfigurationLoader(opts)

	_, err = cl.LoadAndProcess(tmpFile)
	if err == nil {
		t.Error("Expected error for file exceeding max size")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("Error should mention file size, got: %v", err)
	}
}

func TestConfigurationLoader_InvalidFile(t *testing.T) {
	cl := NewConfigurationLoader(nil)

	// Try to load non-existent file
	_, err := cl.LoadAndProcess("/non/existent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("Error should mention file opening, got: %v", err)
	}
}

func TestConfigurationLoader_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")

	// Create file with invalid YAML
	content := `
name: test
  invalid: yaml: structure
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cl := NewConfigurationLoader(nil)

	_, err = cl.LoadApplyConfig(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse YAML") {
		t.Errorf("Error should mention YAML parsing, got: %v", err)
	}
}

func TestTemplateCache_Methods(t *testing.T) {
	cache := &templateCache{
		entries: make(map[string]*templateCacheEntry),
		ttl:     1 * time.Minute,
	}

	source := "test.yaml"
	content := "name: ${TEST}"
	result := &SubstitutionResult{
		Content:   "name: value",
		Variables: map[string]string{"TEST": "value"},
	}

	// Test set
	cache.set(source, content, result)

	// Test get with same content
	cached := cache.get(source, content)
	if cached == nil {
		t.Error("Should return cached result")
	}
	if cached.Content != result.Content {
		t.Error("Cached content differs")
	}

	// Test get with different content
	differentContent := "name: ${OTHER}"
	cached = cache.get(source, differentContent)
	if cached != nil {
		t.Error("Should not return cached result for different content")
	}

	// Test stats
	stats := cache.stats()
	if stats["entries"].(int) != 1 {
		t.Errorf("Stats entries = %d, want 1", stats["entries"].(int))
	}

	// Test clear
	cache.clear()
	stats = cache.stats()
	if stats["entries"].(int) != 0 {
		t.Error("Cache should be empty after clear")
	}
}

func TestTemplateCache_Expiration(t *testing.T) {
	cache := &templateCache{
		entries: make(map[string]*templateCacheEntry),
		ttl:     10 * time.Millisecond, // Very short TTL
	}

	source := "test.yaml"
	content := "name: test"
	result := &SubstitutionResult{Content: content}

	// Set entry
	cache.set(source, content, result)

	// Should be available immediately
	cached := cache.get(source, content)
	if cached == nil {
		t.Error("Should return cached result immediately")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should be expired
	cached = cache.get(source, content)
	if cached != nil {
		t.Error("Should not return expired cached result")
	}
}

func TestConfigurationLoader_CacheCleanup(t *testing.T) {
	opts := DefaultLoaderOptions()
	opts.CacheTTL = 10 * time.Millisecond

	cl := NewConfigurationLoader(opts)

	// Add an entry that will expire quickly
	cl.cache.set("test", "content", &SubstitutionResult{Content: "result"})

	// Start cleanup with short interval
	stop := cl.StartCacheCleanup(15 * time.Millisecond)
	defer close(stop)

	// Wait for cleanup to run
	time.Sleep(30 * time.Millisecond)

	// Entry should be cleaned up
	stats := cl.GetCacheStats()
	if stats["entries"].(int) != 0 {
		t.Error("Expired entries should be cleaned up")
	}
}

func TestLoadResult_Fields(t *testing.T) {
	result := &LoadResult{
		RawContent:       "original",
		ProcessedContent: "processed",
		Substitutions:    map[string]string{"VAR": "value"},
		Source:           "test.yaml",
		LoadedAt:         time.Now(),
	}

	if result.RawContent != "original" {
		t.Error("RawContent not set correctly")
	}
	if result.ProcessedContent != "processed" {
		t.Error("ProcessedContent not set correctly")
	}
	if result.Substitutions["VAR"] != "value" {
		t.Error("Substitutions not set correctly")
	}
	if result.Source != "test.yaml" {
		t.Error("Source not set correctly")
	}
	if result.LoadedAt.IsZero() {
		t.Error("LoadedAt should be set")
	}
}

func TestConfigurationLoader_LoadProjectKind(t *testing.T) {
	loader := NewConfigurationLoader(DefaultLoaderOptions())

	// Create a temporary file with Project kind YAML
	projectYAML := `apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-project
spec:
  name: "TestProject"
  organizationId: "507f1f77bcf86cd799439011"
  databaseUsers:
    - metadata:
        name: test-user
      username: testuser
      password: testpass
      roles:
        - roleName: readWrite
          databaseName: admin
      authDatabase: admin`

	tmpfile, err := os.CreateTemp("", "project-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(projectYAML))
	require.NoError(t, err)
	tmpfile.Close()

	// Load the configuration
	result, err := loader.LoadApplyConfig(tmpfile.Name())
	require.NoError(t, err)
	require.NotNil(t, result.Config)

	// Should be converted to ApplyConfig
	applyConfig, ok := result.Config.(*types.ApplyConfig)
	require.True(t, ok, "Expected ApplyConfig, got %T", result.Config)

	// Verify it's recognized as a Project kind
	assert.Equal(t, "Project", applyConfig.Kind)
	assert.Equal(t, "TestProject", applyConfig.Spec.Name)
	assert.Equal(t, "507f1f77bcf86cd799439011", applyConfig.Spec.OrganizationID)
	assert.Len(t, applyConfig.Spec.DatabaseUsers, 1)
	assert.Equal(t, "testuser", applyConfig.Spec.DatabaseUsers[0].Username)
}
