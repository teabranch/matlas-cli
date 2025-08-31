package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/teabranch/matlas-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// AdvancedSearchFormatter handles formatting for advanced search features
type AdvancedSearchFormatter struct {
	format config.OutputFormat
	writer io.Writer
}

// CreateAdvancedSearchFormatter creates a new AdvancedSearchFormatter
func CreateAdvancedSearchFormatter() *AdvancedSearchFormatter {
	return &AdvancedSearchFormatter{
		format: config.OutputTable,
		writer: os.Stdout,
	}
}

// FormatAnalyzers formats analyzer information
func (f *AdvancedSearchFormatter) FormatAnalyzers(analyzers []map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatAnalyzersJSON(analyzers)
	case "yaml":
		return f.formatAnalyzersYAML(analyzers)
	default:
		return f.formatAnalyzersTable(analyzers)
	}
}

// FormatFacets formats facet information
func (f *AdvancedSearchFormatter) FormatFacets(facets []map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatFacetsJSON(facets)
	case "yaml":
		return f.formatFacetsYAML(facets)
	default:
		return f.formatFacetsTable(facets)
	}
}

// FormatAutocomplete formats autocomplete information
func (f *AdvancedSearchFormatter) FormatAutocomplete(autocompletes []map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatAutocompleteJSON(autocompletes)
	case "yaml":
		return f.formatAutocompleteYAML(autocompletes)
	default:
		return f.formatAutocompleteTable(autocompletes)
	}
}

// FormatHighlighting formats highlighting information
func (f *AdvancedSearchFormatter) FormatHighlighting(highlighting []map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatHighlightingJSON(highlighting)
	case "yaml":
		return f.formatHighlightingYAML(highlighting)
	default:
		return f.formatHighlightingTable(highlighting)
	}
}

// FormatSynonyms formats synonyms information
func (f *AdvancedSearchFormatter) FormatSynonyms(synonyms []map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatSynonymsJSON(synonyms)
	case "yaml":
		return f.formatSynonymsYAML(synonyms)
	default:
		return f.formatSynonymsTable(synonyms)
	}
}

// FormatFuzzy formats fuzzy search information
func (f *AdvancedSearchFormatter) FormatFuzzy(fuzzy []map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatFuzzyJSON(fuzzy)
	case "yaml":
		return f.formatFuzzyYAML(fuzzy)
	default:
		return f.formatFuzzyTable(fuzzy)
	}
}

// FormatMetrics formats search metrics information
func (f *AdvancedSearchFormatter) FormatMetrics(metrics map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatMetricsJSON(metrics)
	case "yaml":
		return f.formatMetricsYAML(metrics)
	default:
		return f.formatMetricsTable(metrics)
	}
}

// FormatOptimizationReport formats optimization analysis report
func (f *AdvancedSearchFormatter) FormatOptimizationReport(report map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatOptimizationJSON(report)
	case "yaml":
		return f.formatOptimizationYAML(report)
	default:
		return f.formatOptimizationTable(report)
	}
}

// FormatValidationResult formats validation results
func (f *AdvancedSearchFormatter) FormatValidationResult(result map[string]interface{}, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return f.formatValidationJSON(result)
	case "yaml":
		return f.formatValidationYAML(result)
	default:
		return f.formatValidationTable(result)
	}
}

// Analyzer formatting implementations
func (f *AdvancedSearchFormatter) formatAnalyzersJSON(analyzers []map[string]interface{}) error {
	output := map[string]interface{}{
		"analyzers": analyzers,
		"count":     len(analyzers),
	}
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatAnalyzersYAML(analyzers []map[string]interface{}) error {
	output := map[string]interface{}{
		"analyzers": analyzers,
		"count":     len(analyzers),
	}
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatAnalyzersTable(analyzers []map[string]interface{}) error {
	if len(analyzers) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No analyzers found.")
		return nil
	}

	headers := []string{"NAME", "TYPE", "STATUS", "DESCRIPTION"}
	var rows [][]string

	for _, analyzer := range analyzers {
		name := f.getStringValue(analyzer, "name")
		analyzerType := f.getStringValue(analyzer, "type")
		status := f.getStringValue(analyzer, "status")
		description := f.getStringValue(analyzer, "description")

		rows = append(rows, []string{name, analyzerType, status, description})
	}

	return f.renderTable(headers, rows)
}

// Facet formatting implementations
func (f *AdvancedSearchFormatter) formatFacetsJSON(facets []map[string]interface{}) error {
	output := map[string]interface{}{
		"facets": facets,
		"count":  len(facets),
	}
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatFacetsYAML(facets []map[string]interface{}) error {
	output := map[string]interface{}{
		"facets": facets,
		"count":  len(facets),
	}
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatFacetsTable(facets []map[string]interface{}) error {
	if len(facets) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No facet configurations found.")
		return nil
	}

	headers := []string{"FIELD", "TYPE", "STATUS", "DESCRIPTION"}
	var rows [][]string

	for _, facet := range facets {
		field := f.getStringValue(facet, "field")
		facetType := f.getStringValue(facet, "type")
		status := f.getStringValue(facet, "status")
		description := f.getStringValue(facet, "description")

		rows = append(rows, []string{field, facetType, status, description})
	}

	return f.renderTable(headers, rows)
}

// Autocomplete formatting implementations
func (f *AdvancedSearchFormatter) formatAutocompleteJSON(autocompletes []map[string]interface{}) error {
	output := map[string]interface{}{
		"autocomplete": autocompletes,
		"count":        len(autocompletes),
	}
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatAutocompleteYAML(autocompletes []map[string]interface{}) error {
	output := map[string]interface{}{
		"autocomplete": autocompletes,
		"count":        len(autocompletes),
	}
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatAutocompleteTable(autocompletes []map[string]interface{}) error {
	if len(autocompletes) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No autocomplete configurations found.")
		return nil
	}

	headers := []string{"FIELD", "MAX_EDITS", "PREFIX_LENGTH", "STATUS"}
	var rows [][]string

	for _, autocomplete := range autocompletes {
		field := f.getStringValue(autocomplete, "field")
		maxEdits := f.getStringValue(autocomplete, "maxEdits")
		prefixLength := f.getStringValue(autocomplete, "prefixLength")
		status := f.getStringValue(autocomplete, "status")

		rows = append(rows, []string{field, maxEdits, prefixLength, status})
	}

	return f.renderTable(headers, rows)
}

// Highlighting formatting implementations
func (f *AdvancedSearchFormatter) formatHighlightingJSON(highlighting []map[string]interface{}) error {
	output := map[string]interface{}{
		"highlighting": highlighting,
		"count":        len(highlighting),
	}
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatHighlightingYAML(highlighting []map[string]interface{}) error {
	output := map[string]interface{}{
		"highlighting": highlighting,
		"count":        len(highlighting),
	}
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatHighlightingTable(highlighting []map[string]interface{}) error {
	if len(highlighting) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No highlighting configurations found.")
		return nil
	}

	headers := []string{"FIELD", "MAX_CHARS", "MAX_PASSAGES", "STATUS"}
	var rows [][]string

	for _, highlight := range highlighting {
		field := f.getStringValue(highlight, "field")
		maxChars := f.getStringValue(highlight, "maxCharsToExamine")
		maxPassages := f.getStringValue(highlight, "maxNumPassages")
		status := f.getStringValue(highlight, "status")

		rows = append(rows, []string{field, maxChars, maxPassages, status})
	}

	return f.renderTable(headers, rows)
}

// Synonyms formatting implementations
func (f *AdvancedSearchFormatter) formatSynonymsJSON(synonyms []map[string]interface{}) error {
	output := map[string]interface{}{
		"synonyms": synonyms,
		"count":    len(synonyms),
	}
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatSynonymsYAML(synonyms []map[string]interface{}) error {
	output := map[string]interface{}{
		"synonyms": synonyms,
		"count":    len(synonyms),
	}
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatSynonymsTable(synonyms []map[string]interface{}) error {
	if len(synonyms) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No synonym configurations found.")
		return nil
	}

	headers := []string{"NAME", "INPUT_COUNT", "OUTPUT", "STATUS"}
	var rows [][]string

	for _, synonym := range synonyms {
		name := f.getStringValue(synonym, "name")
		inputCount := f.getStringValue(synonym, "inputCount")
		output := f.getStringValue(synonym, "output")
		status := f.getStringValue(synonym, "status")

		rows = append(rows, []string{name, inputCount, output, status})
	}

	return f.renderTable(headers, rows)
}

// Fuzzy formatting implementations
func (f *AdvancedSearchFormatter) formatFuzzyJSON(fuzzy []map[string]interface{}) error {
	output := map[string]interface{}{
		"fuzzy": fuzzy,
		"count": len(fuzzy),
	}
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatFuzzyYAML(fuzzy []map[string]interface{}) error {
	output := map[string]interface{}{
		"fuzzy": fuzzy,
		"count": len(fuzzy),
	}
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(output)
}

func (f *AdvancedSearchFormatter) formatFuzzyTable(fuzzy []map[string]interface{}) error {
	if len(fuzzy) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No fuzzy search configurations found.")
		return nil
	}

	headers := []string{"FIELD", "MAX_EDITS", "PREFIX_LENGTH", "MAX_EXPANSIONS", "STATUS"}
	var rows [][]string

	for _, fuzz := range fuzzy {
		field := f.getStringValue(fuzz, "field")
		maxEdits := f.getStringValue(fuzz, "maxEdits")
		prefixLength := f.getStringValue(fuzz, "prefixLength")
		maxExpansions := f.getStringValue(fuzz, "maxExpansions")
		status := f.getStringValue(fuzz, "status")

		rows = append(rows, []string{field, maxEdits, prefixLength, maxExpansions, status})
	}

	return f.renderTable(headers, rows)
}

// Metrics formatting implementations
func (f *AdvancedSearchFormatter) formatMetricsJSON(metrics map[string]interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(metrics)
}

func (f *AdvancedSearchFormatter) formatMetricsYAML(metrics map[string]interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(metrics)
}

func (f *AdvancedSearchFormatter) formatMetricsTable(metrics map[string]interface{}) error {
	_, _ = fmt.Fprintln(f.writer, "Search Index Metrics")
	_, _ = fmt.Fprintln(f.writer, strings.Repeat("=", 50))

	// Show placeholder warning prominently if present
	if warning, ok := metrics["_warning"].(string); ok {
		_, _ = fmt.Fprintln(f.writer, "⚠️  WARNING: PLACEHOLDER DATA")
		_, _ = fmt.Fprintf(f.writer, "   %s\n", warning)
		if note, ok := metrics["_note"].(string); ok {
			_, _ = fmt.Fprintf(f.writer, "   %s\n", note)
		}
		_, _ = fmt.Fprintln(f.writer, strings.Repeat("-", 50))
	}

	// Basic metrics
	if indexName, ok := metrics["indexName"]; ok {
		_, _ = fmt.Fprintf(f.writer, "Index Name: %v\n", indexName)
	}
	if queryCount, ok := metrics["queryCount"]; ok {
		_, _ = fmt.Fprintf(f.writer, "Total Queries: %v\n", queryCount)
	}
	if avgQueryTime, ok := metrics["avgQueryTime"]; ok {
		_, _ = fmt.Fprintf(f.writer, "Average Query Time: %v ms\n", avgQueryTime)
	}
	if indexSize, ok := metrics["indexSize"]; ok {
		_, _ = fmt.Fprintf(f.writer, "Index Size: %v\n", indexSize)
	}

	// Performance metrics
	if perf, ok := metrics["performance"]; ok {
		if perfMap, ok := perf.(map[string]interface{}); ok {
			_, _ = fmt.Fprintln(f.writer, "\nPerformance Metrics:")
			for key, value := range perfMap {
				if key != "placeholder" { // Skip placeholder markers in display
					_, _ = fmt.Fprintf(f.writer, "  %s: %v\n", key, value)
				}
			}
		}
	}

	return nil
}

// Optimization formatting implementations
func (f *AdvancedSearchFormatter) formatOptimizationJSON(report map[string]interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func (f *AdvancedSearchFormatter) formatOptimizationYAML(report map[string]interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(report)
}

func (f *AdvancedSearchFormatter) formatOptimizationTable(report map[string]interface{}) error {
	_, _ = fmt.Fprintln(f.writer, "Search Index Optimization Report")
	_, _ = fmt.Fprintln(f.writer, strings.Repeat("=", 50))

	if indexName, ok := report["indexName"]; ok {
		_, _ = fmt.Fprintf(f.writer, "Index Name: %v\n", indexName)
	}

	if score, ok := report["optimizationScore"]; ok {
		_, _ = fmt.Fprintf(f.writer, "Optimization Score: %v/100\n", score)
	}

	if recommendations, ok := report["recommendations"]; ok {
		if recList, ok := recommendations.([]interface{}); ok {
			_, _ = fmt.Fprintln(f.writer, "\nRecommendations:")
			for i, rec := range recList {
				if recMap, ok := rec.(map[string]interface{}); ok {
					_, _ = fmt.Fprintf(f.writer, "%d. %v\n", i+1, recMap["title"])
					if desc, ok := recMap["description"]; ok {
						_, _ = fmt.Fprintf(f.writer, "   %v\n", desc)
					}
				}
			}
		}
	}

	return nil
}

// Validation formatting implementations
func (f *AdvancedSearchFormatter) formatValidationJSON(result map[string]interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func (f *AdvancedSearchFormatter) formatValidationYAML(result map[string]interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override the main return value
			_ = err
		}
	}()
	return encoder.Encode(result)
}

func (f *AdvancedSearchFormatter) formatValidationTable(result map[string]interface{}) error {
	_, _ = fmt.Fprintln(f.writer, "Validation Results")
	_, _ = fmt.Fprintln(f.writer, strings.Repeat("=", 30))

	if valid, ok := result["valid"]; ok {
		status := "FAIL"
		if validBool, ok := valid.(bool); ok && validBool {
			status = "PASS"
		}
		_, _ = fmt.Fprintf(f.writer, "Status: %s\n", status)
	}

	if errors, ok := result["errors"]; ok {
		if errorList, ok := errors.([]interface{}); ok && len(errorList) > 0 {
			_, _ = fmt.Fprintln(f.writer, "\nErrors:")
			for _, err := range errorList {
				_, _ = fmt.Fprintf(f.writer, "  • %v\n", err)
			}
		}
	}

	if warnings, ok := result["warnings"]; ok {
		if warnList, ok := warnings.([]interface{}); ok && len(warnList) > 0 {
			_, _ = fmt.Fprintln(f.writer, "\nWarnings:")
			for _, warn := range warnList {
				_, _ = fmt.Fprintf(f.writer, "  • %v\n", warn)
			}
		}
	}

	return nil
}

// renderTable renders data in a table format
func (f *AdvancedSearchFormatter) renderTable(headers []string, rows [][]string) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(f.writer, "No data found")
		return err
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	// Print headers
	if len(headers) > 0 {
		if _, err := fmt.Fprintln(w, strings.Join(headers, "\t")); err != nil {
			return err
		}
		// Print separator line
		separators := make([]string, len(headers))
		for i := range separators {
			separators[i] = "----"
		}
		if _, err := fmt.Fprintln(w, strings.Join(separators, "\t")); err != nil {
			return err
		}
	}

	// Print rows
	for _, row := range rows {
		if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
			return err
		}
	}

	return nil
}

// Helper method to safely get string values from maps
func (f *AdvancedSearchFormatter) getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", value)
	}
	return ""
}
