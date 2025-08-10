package apply

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/teabranch/matlas-cli/internal/types"
)

// LoaderOptions provides configuration for the configuration loader
type LoaderOptions struct {
	StrictEnv    bool              // Fail on undefined environment variables
	Debug        bool              // Enable debug output
	CacheEnabled bool              // Enable template caching
	CacheTTL     time.Duration     // Cache time-to-live
	Variables    map[string]string // Custom variables
	AllowStdin   bool              // Allow reading from stdin
	MaxFileSize  int64             // Maximum file size in bytes
}

// DefaultLoaderOptions returns sensible defaults for the loader
func DefaultLoaderOptions() *LoaderOptions {
	return &LoaderOptions{
		StrictEnv:    false,
		Debug:        false,
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		Variables:    make(map[string]string),
		AllowStdin:   true,
		MaxFileSize:  10 * 1024 * 1024, // 10MB
	}
}

// ConfigurationLoader handles loading and processing configuration files
type ConfigurationLoader struct {
	options   *LoaderOptions
	processor *TemplateProcessor
	cache     *templateCache
	mu        sync.RWMutex
}

// templateCacheEntry represents a cached template result
type templateCacheEntry struct {
	Content     string
	Hash        string
	ProcessedAt time.Time
	Result      *SubstitutionResult
}

// templateCache provides thread-safe caching of template processing results
type templateCache struct {
	entries map[string]*templateCacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
}

// LoadResult contains the result of loading a configuration
type LoadResult struct {
	Config           interface{}         `json:"config,omitempty"`        // Parsed configuration
	RawContent       string              `json:"rawContent,omitempty"`    // Original file content
	ProcessedContent string              `json:"processedContent"`        // After template processing
	Substitutions    map[string]string   `json:"substitutions,omitempty"` // Variables that were substituted
	Errors           []SubstitutionError `json:"errors,omitempty"`        // Template processing errors
	Warnings         []SubstitutionError `json:"warnings,omitempty"`      // Template processing warnings
	Source           string              `json:"source"`                  // File path or "stdin"
	LoadedAt         time.Time           `json:"loadedAt"`                // When the file was loaded
}

// NewConfigurationLoader creates a new configuration loader
func NewConfigurationLoader(opts *LoaderOptions) *ConfigurationLoader {
	if opts == nil {
		opts = DefaultLoaderOptions()
	}

	processor := NewTemplateProcessor().
		WithStrictMode(opts.StrictEnv).
		WithDebugMode(opts.Debug).
		WithVariables(opts.Variables)

	cache := &templateCache{
		entries: make(map[string]*templateCacheEntry),
		ttl:     opts.CacheTTL,
	}

	return &ConfigurationLoader{
		options:   opts,
		processor: processor,
		cache:     cache,
	}
}

// LoadApplyConfig loads and parses an ApplyConfig from a file or stdin
// Also supports DiscoveredProject format, which will be converted to ApplyDocument
func (cl *ConfigurationLoader) LoadApplyConfig(source string) (*LoadResult, error) {
	// Load and process the raw content
	result, err := cl.LoadAndProcess(source)
	if err != nil {
		return result, err
	}

	// Check for empty content
	trimmed := strings.TrimSpace(result.ProcessedContent)
	if trimmed == "" {
		return result, fmt.Errorf("configuration file is empty")
	}

	// First, try to detect the format by parsing as generic YAML
	var genericDoc interface{}
	if err := yaml.Unmarshal([]byte(result.ProcessedContent), &genericDoc); err != nil {
		return result, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Check if it's a DiscoveredProject format
	if docMap, ok := genericDoc.(map[string]interface{}); ok {
		if kind, exists := docMap["kind"].(string); exists {
			switch kind {
			case "DiscoveredProject":
				// Convert DiscoveredProject to ApplyDocument
				converter := NewDiscoveredProjectConverter()
				applyDoc, err := converter.ConvertToApplyDocument(genericDoc)
				if err != nil {
					return result, fmt.Errorf("failed to convert DiscoveredProject format: %w", err)
				}
				result.Config = applyDoc
				return result, nil
			case "Project":
				// Parse as standalone Project manifest and convert to ApplyConfig
				var projectManifest types.ProjectManifest
				if err := yaml.Unmarshal([]byte(result.ProcessedContent), &projectManifest); err != nil {
					return result, fmt.Errorf("failed to parse Project manifest: %w", err)
				}

				// Convert Project manifest to ApplyConfig format
				applyConfig := &types.ApplyConfig{
					APIVersion: string(projectManifest.APIVersion),
					Kind:       "Project", // Keep as Project kind for compatibility
					Metadata: types.MetadataConfig{
						Name:        projectManifest.Metadata.Name,
						Labels:      projectManifest.Metadata.Labels,
						Annotations: projectManifest.Metadata.Annotations,
					},
					Spec: projectManifest.Spec,
				}
				result.Config = applyConfig
				return result, nil
			case "ApplyDocument":
				// Parse as ApplyDocument
				var document types.ApplyDocument
				if err := yaml.Unmarshal([]byte(result.ProcessedContent), &document); err != nil {
					return result, fmt.Errorf("failed to parse ApplyDocument: %w", err)
				}
				result.Config = &document
				return result, nil
			}
		}
	}

	// Parse as ApplyConfig (original behavior)
	var config types.ApplyConfig
	if err := yaml.Unmarshal([]byte(result.ProcessedContent), &config); err != nil {
		return result, fmt.Errorf("failed to parse YAML: %w", err)
	}

	result.Config = &config
	return result, nil
}

// LoadApplyDocument loads and parses an ApplyDocument from a file or stdin
func (cl *ConfigurationLoader) LoadApplyDocument(source string) (*LoadResult, error) {
	// Load and process the raw content
	result, err := cl.LoadAndProcess(source)
	if err != nil {
		return result, err
	}

	// Parse as ApplyDocument
	var document types.ApplyDocument
	if err := yaml.Unmarshal([]byte(result.ProcessedContent), &document); err != nil {
		return result, fmt.Errorf("failed to parse YAML: %w", err)
	}

	result.Config = &document
	return result, nil
}

// LoadMultiDocument loads and parses multiple YAML documents from a file
func (cl *ConfigurationLoader) LoadMultiDocument(source string) ([]*LoadResult, error) {
	// Load raw content
	content, err := cl.loadRawContent(source)
	if err != nil {
		return nil, err
	}

	// Split by document separator
	documents := strings.Split(content, "\n---\n")
	var results []*LoadResult

	for i, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		// Process template for each document
		substitutionResult := cl.processor.SubstituteEnvVars(doc)

		result := &LoadResult{
			RawContent:       doc,
			ProcessedContent: substitutionResult.Content,
			Substitutions:    substitutionResult.Variables,
			Errors:           substitutionResult.Errors,
			Warnings:         substitutionResult.Warnings,
			Source:           fmt.Sprintf("%s[%d]", source, i),
			LoadedAt:         time.Now(),
		}

		// Stop processing if there are errors in strict mode
		if cl.options.StrictEnv && len(result.Errors) > 0 {
			return nil, fmt.Errorf("template processing failed for document %d: %d errors", i, len(result.Errors))
		}

		// Try to parse as generic YAML to detect structure
		var yamlDoc interface{}
		if err := yaml.Unmarshal([]byte(result.ProcessedContent), &yamlDoc); err != nil {
			return nil, fmt.Errorf("failed to parse YAML document %d: %w", i, err)
		}

		result.Config = yamlDoc
		results = append(results, result)
	}

	return results, nil
}

// LoadAndProcess loads a file and processes templates without parsing YAML
func (cl *ConfigurationLoader) LoadAndProcess(source string) (*LoadResult, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// Load raw content
	content, err := cl.loadRawContent(source)
	if err != nil {
		return nil, err
	}

	// Check cache if enabled
	var substitutionResult *SubstitutionResult
	if cl.options.CacheEnabled {
		if cachedResult := cl.cache.get(source, content); cachedResult != nil {
			if cl.options.Debug {
				fmt.Printf("Using cached template result for %s\n", source)
			}
			substitutionResult = cachedResult
		}
	}

	// Process template if not cached
	if substitutionResult == nil {
		substitutionResult = cl.processor.SubstituteEnvVars(content)

		// Cache the result if enabled
		if cl.options.CacheEnabled {
			cl.cache.set(source, content, substitutionResult)
		}

		if cl.options.Debug {
			fmt.Printf("Processed template for %s with %d substitutions\n", source, len(substitutionResult.Variables))
		}
	}

	result := &LoadResult{
		RawContent:       content,
		ProcessedContent: substitutionResult.Content,
		Substitutions:    substitutionResult.Variables,
		Errors:           substitutionResult.Errors,
		Warnings:         substitutionResult.Warnings,
		Source:           source,
		LoadedAt:         time.Now(),
	}

	// Check for errors in strict mode
	if cl.options.StrictEnv && len(result.Errors) > 0 {
		return result, fmt.Errorf("template processing failed: %d errors", len(result.Errors))
	}

	return result, nil
}

// LoadGlob loads multiple files matching a glob pattern
func (cl *ConfigurationLoader) LoadGlob(pattern string) ([]*LoadResult, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no files match pattern: %s", pattern)
	}

	var results []*LoadResult
	for _, match := range matches {
		result, err := cl.LoadAndProcess(match)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", match, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// ValidateOnly validates a template without processing substitutions
func (cl *ConfigurationLoader) ValidateOnly(source string) (*SubstitutionResult, error) {
	content, err := cl.loadRawContent(source)
	if err != nil {
		return nil, err
	}

	return cl.processor.ValidateTemplate(content), nil
}

// ExtractVariables extracts all variable references from a file
func (cl *ConfigurationLoader) ExtractVariables(source string) ([]string, error) {
	content, err := cl.loadRawContent(source)
	if err != nil {
		return nil, err
	}

	return cl.processor.ExtractVariables(content), nil
}

// SetVariable sets a custom variable that overrides environment variables
func (cl *ConfigurationLoader) SetVariable(name, value string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.processor.Variables[name] = value
	cl.options.Variables[name] = value

	// Invalidate cache since variables changed
	if cl.options.CacheEnabled {
		cl.cache.clear()
	}
}

// SetVariables sets multiple custom variables
func (cl *ConfigurationLoader) SetVariables(vars map[string]string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for name, value := range vars {
		cl.processor.Variables[name] = value
		cl.options.Variables[name] = value
	}

	// Invalidate cache since variables changed
	if cl.options.CacheEnabled {
		cl.cache.clear()
	}
}

// ClearCache clears the template processing cache
func (cl *ConfigurationLoader) ClearCache() {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.cache.clear()
}

// GetCacheStats returns cache statistics
func (cl *ConfigurationLoader) GetCacheStats() map[string]interface{} {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return cl.cache.stats()
}

// loadRawContent loads content from a file or stdin
func (cl *ConfigurationLoader) loadRawContent(source string) (string, error) {
	var reader io.Reader
	var size int64

	if source == "-" || source == "stdin" {
		if !cl.options.AllowStdin {
			return "", fmt.Errorf("stdin input not allowed")
		}
		reader = os.Stdin
		size = cl.options.MaxFileSize // Use max size for stdin (used to configure limit reader)
	} else {
		file, err := os.Open(source) //nolint:gosec // user-provided path is expected for CLI tool
		if err != nil {
			return "", fmt.Errorf("failed to open file %s: %w", source, err)
		}
		defer func() { _ = file.Close() }()

		// Check file size
		info, err := file.Stat()
		if err != nil {
			return "", fmt.Errorf("failed to stat file %s: %w", source, err)
		}
		size = info.Size()

		if size > cl.options.MaxFileSize {
			return "", fmt.Errorf("file %s is too large (%d bytes, max %d)", source, size, cl.options.MaxFileSize)
		}

		reader = file
	}

	// Read content with size limit
	limited := io.LimitReader(reader, size)
	content, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("failed to read content from %s: %w", source, err)
	}

	if cl.options.Debug {
		fmt.Printf("Loaded %d bytes from %s\n", len(content), source)
	}

	return string(content), nil
}

// Template cache methods

func (tc *templateCache) get(source, content string) *SubstitutionResult {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	hash := tc.computeHash(content)
	entry, exists := tc.entries[source]
	if !exists {
		return nil
	}

	// Check if entry is expired
	if time.Since(entry.ProcessedAt) > tc.ttl {
		return nil
	}

	// Check if content changed
	if entry.Hash != hash {
		return nil
	}

	return entry.Result
}

func (tc *templateCache) set(source, content string, result *SubstitutionResult) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	hash := tc.computeHash(content)
	tc.entries[source] = &templateCacheEntry{
		Content:     content,
		Hash:        hash,
		ProcessedAt: time.Now(),
		Result:      result,
	}
}

func (tc *templateCache) clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.entries = make(map[string]*templateCacheEntry)
}

func (tc *templateCache) stats() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	stats := map[string]interface{}{
		"entries": len(tc.entries),
		"ttl":     tc.ttl.String(),
	}

	// Count expired entries
	expired := 0
	now := time.Now()
	for _, entry := range tc.entries {
		if now.Sub(entry.ProcessedAt) > tc.ttl {
			expired++
		}
	}
	stats["expired"] = expired

	return stats
}

func (tc *templateCache) computeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// cleanup removes expired entries from the cache
func (tc *templateCache) cleanup() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now()
	for source, entry := range tc.entries {
		if now.Sub(entry.ProcessedAt) > tc.ttl {
			delete(tc.entries, source)
		}
	}
}

// StartCacheCleanup starts a background goroutine to clean up expired cache entries
func (cl *ConfigurationLoader) StartCacheCleanup(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cl.cache.cleanup()
			case <-stop:
				return
			}
		}
	}()

	return stop
}
