package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/teabranch/matlas-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// Formatter handles output formatting for different formats
type Formatter struct {
	format config.OutputFormat
	writer io.Writer
}

// NewFormatter creates a new formatter for the specified format
func NewFormatter(format config.OutputFormat, writer io.Writer) *Formatter {
	return &Formatter{
		format: format,
		writer: writer,
	}
}

// Format outputs data in the configured format
func (f *Formatter) Format(data interface{}) error {
	switch f.format {
	case config.OutputJSON:
		return f.formatJSON(data)
	case config.OutputYAML:
		return f.formatYAML(data)
	case config.OutputTable, config.OutputText, "":
		return f.formatText(data)
	default:
		return fmt.Errorf("unsupported output format: %s", f.format)
	}
}

// formatJSON outputs data as pretty-printed JSON
func (f *Formatter) formatJSON(data interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// formatYAML outputs data as YAML
func (f *Formatter) formatYAML(data interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

// formatText outputs data as human-readable text/tables
func (f *Formatter) formatText(data interface{}) error {
	if data == nil {
		return nil
	}

	// Handle different data types
	switch v := data.(type) {
	case []interface{}:
		return f.formatTable(v)
	case TableData:
		return f.formatTableData(v)
	default:
		// For single objects, try to format as a simple table
		return f.formatSingleObject(v)
	}
}

// TableData represents structured table data with headers and rows
type TableData struct {
	Headers []string
	Rows    [][]string
}

// formatTableData formats TableData as a text table
func (f *Formatter) formatTableData(data TableData) error {
	if len(data.Rows) == 0 {
		_, err := fmt.Fprintln(f.writer, "No data found")
		return err
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Print headers
	if len(data.Headers) > 0 {
		fmt.Fprintln(w, strings.Join(data.Headers, "\t"))
		// Print separator line
		separators := make([]string, len(data.Headers))
		for i := range separators {
			separators[i] = strings.Repeat("-", len(data.Headers[i]))
		}
		fmt.Fprintln(w, strings.Join(separators, "\t"))
	}

	// Print rows
	for _, row := range data.Rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	return nil
}

// formatTable formats a slice of data as a table by introspection
func (f *Formatter) formatTable(data []interface{}) error {
	if len(data) == 0 {
		_, err := fmt.Fprintln(f.writer, "No data found")
		return err
	}

	// For now, just print each item on a line
	// This can be enhanced based on specific struct types
	for _, item := range data {
		if err := f.formatSingleObject(item); err != nil {
			return err
		}
		fmt.Fprintln(f.writer)
	}

	return nil
}

// formatSingleObject formats a single object as key-value pairs
func (f *Formatter) formatSingleObject(data interface{}) error {
	if data == nil {
		return nil
	}

	// Use reflection to extract fields
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		// For non-struct types, just print the value
		_, err := fmt.Fprintf(f.writer, "%v\n", data)
		return err
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name (prefer json tag)
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			if parts := strings.Split(jsonTag, ","); parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Format the value properly, handling pointers and nil values
		formattedValue := formatValue(value)
		fmt.Fprintf(w, "%s:\t%s\n", fieldName, formattedValue)
	}

	return nil
}

// formatValue formats a reflect.Value into a readable string, handling pointers and nil values
func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return "<invalid>"
	}

	// Handle nil pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "<nil>"
		}
		// Dereference the pointer and format the underlying value
		return formatValue(v.Elem())
	}

	// Handle slices
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			return "[]"
		}
		if v.Len() == 0 {
			return "[]"
		}
		// For non-empty slices, show the count and first few elements
		elements := make([]string, 0, min(3, v.Len()))
		for i := 0; i < min(3, v.Len()); i++ {
			elements = append(elements, formatValue(v.Index(i)))
		}
		if v.Len() > 3 {
			return fmt.Sprintf("[%s... (%d items)]", strings.Join(elements, ", "), v.Len())
		}
		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	}

	// Handle maps
	if v.Kind() == reflect.Map {
		if v.IsNil() {
			return "{}"
		}
		if v.Len() == 0 {
			return "{}"
		}
		return fmt.Sprintf("{%d items}", v.Len())
	}

	// Handle structs - show a summary instead of full details
	if v.Kind() == reflect.Struct {
		typeName := v.Type().Name()
		if typeName == "" {
			typeName = "struct"
		}

		// For time.Time, format it nicely
		if v.Type().String() == "time.Time" {
			if timeVal, ok := v.Interface().(time.Time); ok {
				return timeVal.Format("2006-01-02 15:04:05 UTC")
			}
		}

		return fmt.Sprintf("<%s>", typeName)
	}

	// Handle basic types
	switch v.Kind() {
	case reflect.String:
		str := v.String()
		if str == "" {
			return `""`
		}
		// If string is very long, truncate it
		if len(str) > 100 {
			return fmt.Sprintf(`"%.97s..."`, str)
		}
		return fmt.Sprintf(`"%s"`, str)
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", v.Float())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FormatList is a helper for formatting lists with proper table headers
func FormatList(formatter *Formatter, items interface{}, headers []string, rowFunc func(interface{}) []string) error {
	switch formatter.format {
	case config.OutputJSON, config.OutputYAML:
		return formatter.Format(items)
	case config.OutputTable, config.OutputText, "":
		// Convert to TableData
		itemsValue := reflect.ValueOf(items)
		if itemsValue.Kind() != reflect.Slice {
			return fmt.Errorf("expected slice for list formatting")
		}

		var rows [][]string
		for i := 0; i < itemsValue.Len(); i++ {
			item := itemsValue.Index(i).Interface()
			row := rowFunc(item)
			rows = append(rows, row)
		}

		tableData := TableData{
			Headers: headers,
			Rows:    rows,
		}
		return formatter.formatTableData(tableData)
	default:
		return fmt.Errorf("unsupported output format: %s", formatter.format)
	}
}
