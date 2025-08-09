package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/teabranch/matlas-cli/internal/config"
)

func TestFormatter_NewAndBasicOperation(t *testing.T) {
	var buf bytes.Buffer

	// Test all format types can be created
	formats := []config.OutputFormat{
		config.OutputJSON,
		config.OutputYAML,
		config.OutputText,
		config.OutputTable,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			formatter := NewFormatter(format, &buf)
			assert.NotNil(t, formatter)
			assert.Equal(t, format, formatter.format)
			assert.Equal(t, &buf, formatter.writer)
		})
	}
}

func TestTableData_BasicOperations(t *testing.T) {
	// Test TableData creation and access
	td := TableData{
		Headers: []string{"ID", "Name", "Status"},
		Rows: [][]string{
			{"1", "Alice", "Active"},
			{"2", "Bob", "Inactive"},
			{"3", "Charlie", "Active"},
		},
	}

	// Test structure
	assert.Len(t, td.Headers, 3)
	assert.Len(t, td.Rows, 3)
	assert.Equal(t, "ID", td.Headers[0])
	assert.Equal(t, "Name", td.Headers[1])
	assert.Equal(t, "Status", td.Headers[2])

	// Test row data
	assert.Equal(t, "1", td.Rows[0][0])
	assert.Equal(t, "Alice", td.Rows[0][1])
	assert.Equal(t, "Active", td.Rows[0][2])
	assert.Equal(t, "Bob", td.Rows[1][1])
	assert.Equal(t, "Charlie", td.Rows[2][1])
}

func TestTableData_EmptyData(t *testing.T) {
	// Test empty TableData
	td := TableData{
		Headers: []string{"Col1", "Col2"},
		Rows:    [][]string{},
	}

	assert.Len(t, td.Headers, 2)
	assert.Len(t, td.Rows, 0)
	assert.Equal(t, "Col1", td.Headers[0])
	assert.Equal(t, "Col2", td.Headers[1])
}

func TestFormatter_SimpleDataTypes(t *testing.T) {
	tests := []struct {
		name   string
		format config.OutputFormat
		data   interface{}
	}{
		{
			name:   "JSON with string",
			format: config.OutputJSON,
			data:   "hello world",
		},
		{
			name:   "JSON with number",
			format: config.OutputJSON,
			data:   42,
		},
		{
			name:   "JSON with boolean",
			format: config.OutputJSON,
			data:   true,
		},
		{
			name:   "YAML with string",
			format: config.OutputYAML,
			data:   "test value",
		},
		{
			name:   "Text with string",
			format: config.OutputText,
			data:   "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(tt.format, &buf)

			err := formatter.Format(tt.data)
			assert.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output)
		})
	}
}

func TestFormatter_ErrorHandling(t *testing.T) {
	var buf bytes.Buffer

	// Test unsupported format
	formatter := NewFormatter(config.OutputFormat("invalid"), &buf)
	err := formatter.Format("test data")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format")
}

func TestFormatter_ComplexStructures(t *testing.T) {
	type TestItem struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Valid bool   `json:"valid"`
	}

	data := []TestItem{
		{ID: 1, Name: "First", Valid: true},
		{ID: 2, Name: "Second", Valid: false},
	}

	formats := []config.OutputFormat{
		config.OutputJSON,
		config.OutputYAML,
		config.OutputText,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(format, &buf)

			err := formatter.Format(data)
			assert.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output)
			assert.Contains(t, output, "First")
			assert.Contains(t, output, "Second")
		})
	}
}
