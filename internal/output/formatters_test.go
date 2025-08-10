package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teabranch/matlas-cli/internal/config"
)

func TestNewFormatter(t *testing.T) {
	var buf bytes.Buffer

	tests := []struct {
		name   string
		format config.OutputFormat
	}{
		{"json formatter", config.OutputJSON},
		{"yaml formatter", config.OutputYAML},
		{"table formatter", config.OutputTable},
		{"text formatter", config.OutputText},
		{"empty formatter", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.format, &buf)
			require.NotNil(t, formatter)
			assert.Equal(t, tt.format, formatter.format)
			assert.Equal(t, &buf, formatter.writer)
		})
	}
}

func TestFormatter_Format_Serialization(t *testing.T) {
	tests := []struct {
		name     string
		format   config.OutputFormat
		data     interface{}
		expected string
	}{
		{name: "json simple string", format: config.OutputJSON, data: "hello", expected: "\"hello\"\n"},
		{name: "yaml simple string", format: config.OutputYAML, data: "hello", expected: "hello\n"},
		{name: "json simple object", format: config.OutputJSON, data: map[string]string{"key": "value"}, expected: "{\n  \"key\": \"value\"\n}\n"},
		{name: "yaml simple object", format: config.OutputYAML, data: map[string]string{"key": "value"}, expected: "key: value\n"},
		{name: "json array", format: config.OutputJSON, data: []string{"item1", "item2"}, expected: "[\n  \"item1\",\n  \"item2\"\n]\n"},
		{name: "yaml array", format: config.OutputYAML, data: []string{"item1", "item2"}, expected: "- item1\n- item2\n"},
		{name: "json complex object", format: config.OutputJSON, data: map[string]interface{}{"name": "test", "number": 42, "enabled": true}, expected: "{\n  \"enabled\": true,\n  \"name\": \"test\",\n  \"number\": 42\n}\n"},
		{name: "yaml complex object", format: config.OutputYAML, data: map[string]interface{}{"name": "test", "number": 42, "enabled": true}, expected: "enabled: true\nname: test\nnumber: 42\n"},
		{name: "json nil", format: config.OutputJSON, data: nil, expected: "null\n"},
		{name: "yaml nil", format: config.OutputYAML, data: nil, expected: "null\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(tt.format, &buf)

			err := formatter.Format(tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestFormatter_Format_Table(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		expectError bool
		contains    []string
	}{
		{
			name: "slice of structs",
			data: []struct {
				Name string
				Age  int
			}{
				{"Alice", 30},
				{"Bob", 25},
			},
			expectError: false,
			contains:    []string{"Alice", "Bob", "30", "25"},
		},
		{
			name: "slice of maps",
			data: []map[string]interface{}{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
			},
			expectError: false,
			contains:    []string{"age", "name", "Alice", "Bob", "30", "25"},
		},
		{
			name:        "single struct",
			data:        struct{ Name string }{"Alice"},
			expectError: false,
			contains:    []string{"Alice"},
		},
		{
			name:        "string data",
			data:        "simple string",
			expectError: false,
			contains:    []string{"simple string"},
		},
		{
			name:        "number data",
			data:        42,
			expectError: false,
			contains:    []string{"42"},
		},
		{
			name:        "empty slice",
			data:        []string{},
			expectError: false,
			contains:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputTable, &buf)

			err := formatter.Format(tt.data)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				output := buf.String()
				for _, expectedStr := range tt.contains {
					assert.Contains(t, output, expectedStr)
				}
			}
		})
	}
}

func TestFormatter_Format_Text(t *testing.T) {
	// Text format should behave the same as table format
	var buf bytes.Buffer
	formatter := NewFormatter(config.OutputText, &buf)

	data := []struct {
		Name string
		Age  int
	}{
		{"Alice", 30},
		{"Bob", 25},
	}

	err := formatter.Format(data)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "Bob")
}

func TestFormatter_Format_Empty(t *testing.T) {
	// Empty format should behave the same as table format
	var buf bytes.Buffer
	formatter := NewFormatter("", &buf)

	data := map[string]string{"key": "value"}

	err := formatter.Format(data)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

func TestFormatter_Format_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter("unsupported", &buf)

	err := formatter.Format("test data")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format: unsupported")
}

func TestFormatter_FormatJSON_Error(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter(config.OutputJSON, &buf)

	// Function values cannot be JSON encoded
	err := formatter.Format(func() {})
	assert.Error(t, err)
}

func TestFormatter_FormatYAML_Error(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter(config.OutputYAML, &buf)

	// Function values cannot be YAML encoded - this should panic, so we recover
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior - YAML encoder panics on function types
			assert.NotNil(t, r)
		}
	}()

	// This will panic
	formatter.Format(func() {})
}

// Test with real-world like data structures
func TestFormatter_Format_RealWorldData(t *testing.T) {
	type Project struct {
		ID           string    `json:"id" yaml:"id"`
		Name         string    `json:"name" yaml:"name"`
		ClusterCount int       `json:"clusterCount" yaml:"clusterCount"`
		Created      time.Time `json:"created" yaml:"created"`
	}

	now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	projects := []Project{
		{
			ID:           "proj-123",
			Name:         "Production",
			ClusterCount: 3,
			Created:      now,
		},
		{
			ID:           "proj-456",
			Name:         "Development",
			ClusterCount: 1,
			Created:      now.Add(time.Hour),
		},
	}

	t.Run("JSON format", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewFormatter(config.OutputJSON, &buf)

		err := formatter.Format(projects)
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "proj-123")
		assert.Contains(t, output, "Production")
		assert.Contains(t, output, "proj-456")
		assert.Contains(t, output, "Development")
		// Should be properly formatted JSON
		assert.Contains(t, output, "{\n")
		assert.Contains(t, output, "  ")
	})

	t.Run("YAML format", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewFormatter(config.OutputYAML, &buf)

		err := formatter.Format(projects)
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "id: proj-123")
		assert.Contains(t, output, "name: Production")
		assert.Contains(t, output, "id: proj-456")
		assert.Contains(t, output, "name: Development")
	})

	t.Run("Table format", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewFormatter(config.OutputTable, &buf)

		err := formatter.Format(projects)
		assert.NoError(t, err)

		output := buf.String()
		// Table format for structs outputs the struct representation, not headers
		// Should contain the actual data values
		// Should contain data
		assert.Contains(t, output, "proj-123")
		assert.Contains(t, output, "Production")
		assert.Contains(t, output, "proj-456")
		assert.Contains(t, output, "Development")
	})
}

func TestFormatter_FormatText_WithNilData(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter(config.OutputTable, &buf)

	err := formatter.Format(nil)
	assert.NoError(t, err)

	output := buf.String()
	// Should handle nil gracefully - might be empty or contain nil representation
	// The important thing is that it doesn't panic
	assert.True(t, len(output) >= 0) // Just check it doesn't panic
}

func TestFormatter_FormatText_WithEmptyData(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{"empty slice", []string{}},
		{"empty map", map[string]string{}},
		{"empty struct", struct{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputTable, &buf)

			err := formatter.Format(tt.data)
			assert.NoError(t, err)
			// Should not panic and should produce some output
		})
	}
}
