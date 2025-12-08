package output

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"text/tabwriter"

	"github.com/teabranch/matlas-cli/internal/config"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"
)

// SearchIndexesFormatter provides formatting for Atlas Search indexes
type SearchIndexesFormatter struct {
	format config.OutputFormat
	writer io.Writer
}

// CreateSearchIndexesFormatter creates a new search indexes formatter
func CreateSearchIndexesFormatter() *SearchIndexesFormatter {
	return &SearchIndexesFormatter{
		format: config.OutputTable,
		writer: os.Stdout,
	}
}

// NewSearchIndexesFormatter creates a new formatter with specified format and writer
func NewSearchIndexesFormatter(format config.OutputFormat, writer io.Writer) *SearchIndexesFormatter {
	return &SearchIndexesFormatter{
		format: format,
		writer: writer,
	}
}

// FormatSearchIndexes formats a list of search indexes
func (f *SearchIndexesFormatter) FormatSearchIndexes(indexes []admin.SearchIndexResponse, outputFormat string) error {
	// Determine output format
	format := config.OutputFormat(outputFormat)
	if format == "" {
		format = f.format
	}

	switch format {
	case config.OutputJSON:
		return f.formatJSON(indexes)
	case config.OutputYAML:
		return f.formatYAML(indexes)
	case config.OutputTable, config.OutputText, "":
		return f.formatSearchIndexesTable(indexes)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// formatSearchIndexesTable formats search indexes as a table
func (f *SearchIndexesFormatter) formatSearchIndexesTable(indexes []admin.SearchIndexResponse) error {
	if len(indexes) == 0 {
		_, err := fmt.Fprintln(f.writer, "No search indexes found.")
		return err
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	// Write headers
	_, err := fmt.Fprintln(w, "NAME\tTYPE\tDATABASE\tCOLLECTION\tSTATUS\tQUERYABLE")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, "----\t----\t--------\t----------\t------\t---------")
	if err != nil {
		return err
	}

	// Write index data
	for _, index := range indexes {
		name := getSearchIndexStringField(index, "Name")
		indexType := getSearchIndexType(index)
		database := getSearchIndexStringField(index, "Database")
		collection := getSearchIndexStringField(index, "CollectionName")
		status := getSearchIndexStringField(index, "Status")
		queryable := getSearchIndexBoolField(index, "Queryable")

		queryableStr := "false"
		if queryable {
			queryableStr = "true"
		}

		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			name, indexType, database, collection, status, queryableStr)
		if err != nil {
			return err
		}
	}

	return nil
}

// FormatSearchIndex formats a single search index
func (f *SearchIndexesFormatter) FormatSearchIndex(index admin.SearchIndexResponse, outputFormat string) error {
	// Determine output format
	format := config.OutputFormat(outputFormat)
	if format == "" {
		format = f.format
	}

	switch format {
	case config.OutputJSON:
		return f.formatJSON(index)
	case config.OutputYAML:
		return f.formatYAML(index)
	case config.OutputTable, config.OutputText, "":
		return f.formatSearchIndexDetails(index)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// formatSearchIndexDetails formats a single search index with full details
func (f *SearchIndexesFormatter) formatSearchIndexDetails(index admin.SearchIndexResponse) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	name := getSearchIndexStringField(index, "Name")
	indexType := getSearchIndexType(index)
	database := getSearchIndexStringField(index, "Database")
	collection := getSearchIndexStringField(index, "CollectionName")
	status := getSearchIndexStringField(index, "Status")
	queryable := getSearchIndexBoolField(index, "Queryable")

	if _, err := fmt.Fprintf(w, "Index Name:\t%s\n", name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Type:\t%s\n", indexType); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Database:\t%s\n", database); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Collection:\t%s\n", collection); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Status:\t%s\n", status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Queryable:\t%t\n", queryable); err != nil {
		return err
	}

	// Show index ID if available
	if indexID := getSearchIndexStringField(index, "IndexID"); indexID != "" {
		if _, err := fmt.Fprintf(w, "Index ID:\t%s\n", indexID); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions for extracting fields from search indexes

func getSearchIndexStringField(index admin.SearchIndexResponse, fieldName string) string {
	v := reflect.ValueOf(index)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ""
	}

	if field := v.FieldByName(fieldName); field.IsValid() {
		if field.Kind() == reflect.Ptr {
			if !field.IsNil() {
				return field.Elem().String()
			}
		} else if field.Kind() == reflect.String {
			return field.String()
		}
	}
	return ""
}

func getSearchIndexBoolField(index admin.SearchIndexResponse, fieldName string) bool {
	v := reflect.ValueOf(index)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	if field := v.FieldByName(fieldName); field.IsValid() {
		if field.Kind() == reflect.Ptr {
			if !field.IsNil() {
				return field.Elem().Bool()
			}
		} else if field.Kind() == reflect.Bool {
			return field.Bool()
		}
	}
	return false
}

func getSearchIndexType(index admin.SearchIndexResponse) string {
	// Try to determine if this is a vector search index or regular search
	v := reflect.ValueOf(index)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "search"
		}
		v = v.Elem()
	}

	// Check if there's a type field
	if typeField := v.FieldByName("Type"); typeField.IsValid() && typeField.Kind() == reflect.String {
		indexType := typeField.String()
		if indexType != "" {
			return indexType
		}
	}

	// Check definition for vector search indicators
	if defField := v.FieldByName("LatestDefinition"); defField.IsValid() {
		// This would require deeper inspection of the definition
		// For now, default to "search"
	}

	return "search"
}

// formatJSON and formatYAML use the same base formatter logic
func (f *SearchIndexesFormatter) formatJSON(data interface{}) error {
	formatter := NewFormatter(config.OutputJSON, f.writer)
	return formatter.Format(data)
}

func (f *SearchIndexesFormatter) formatYAML(data interface{}) error {
	formatter := NewFormatter(config.OutputYAML, f.writer)
	return formatter.Format(data)
}
