package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/teabranch/matlas-cli/internal/config"
)

func TestFormatter_FormatJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected string
	}{
		{
			name: "simple object",
			data: map[string]interface{}{
				"name": "test",
				"age":  30,
			},
			expected: `{
  "age": 30,
  "name": "test"
}`,
		},
		{
			name: "array of objects",
			data: []map[string]interface{}{
				{"id": 1, "name": "first"},
				{"id": 2, "name": "second"},
			},
			expected: `[
  {
    "id": 1,
    "name": "first"
  },
  {
    "id": 2,
    "name": "second"
  }
]`,
		},
		{
			name:     "nil data",
			data:     nil,
			expected: "null\n",
		},
		{
			name:     "empty string",
			data:     "",
			expected: `""` + "\n",
		},
		{
			name:     "number",
			data:     42,
			expected: "42\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputJSON, &buf)

			err := formatter.formatJSON(tt.data)
			require.NoError(t, err)

			// Normalize JSON for comparison
			var expected, actual interface{}
			err = json.Unmarshal([]byte(tt.expected), &expected)
			require.NoError(t, err)
			err = json.Unmarshal(buf.Bytes(), &actual)
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestFormatter_FormatYAML(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
		want string
	}{
		{
			name: "simple object",
			data: map[string]interface{}{
				"name": "test",
				"age":  30,
			},
			want: "age: 30\nname: test\n",
		},
		{
			name: "nested object",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "john",
					"id":   123,
				},
				"active": true,
			},
			want: "active: true\nuser:\n  id: 123\n  name: john\n",
		},
		{
			name: "array",
			data: []interface{}{"item1", "item2", "item3"},
			want: "- item1\n- item2\n- item3\n",
		},
		{
			name: "nil data",
			data: nil,
			want: "null\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputYAML, &buf)

			err := formatter.formatYAML(tt.data)
			require.NoError(t, err)

			// Normalize YAML for comparison by parsing both
			var expected, actual interface{}
			err = yaml.Unmarshal([]byte(tt.want), &expected)
			require.NoError(t, err)
			err = yaml.Unmarshal(buf.Bytes(), &actual)
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestFormatter_FormatTableData(t *testing.T) {
	tests := []struct {
		name string
		data TableData
		want []string // Expected strings to be present in output
	}{
		{
			name: "simple table",
			data: TableData{
				Headers: []string{"Name", "Age", "City"},
				Rows: [][]string{
					{"Alice", "30", "New York"},
					{"Bob", "25", "San Francisco"},
				},
			},
			want: []string{"Name", "Age", "City", "Alice", "30", "New York", "Bob", "25", "San Francisco"},
		},
		{
			name: "empty table",
			data: TableData{
				Headers: []string{"Col1", "Col2"},
				Rows:    [][]string{},
			},
			want: []string{"No data found"},
		},
		{
			name: "single row",
			data: TableData{
				Headers: []string{"ID", "Status"},
				Rows: [][]string{
					{"123", "Active"},
				},
			},
			want: []string{"ID", "Status", "123", "Active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputText, &buf)

			err := formatter.formatTableData(tt.data)
			require.NoError(t, err)

			output := buf.String()
			for _, expected := range tt.want {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatter_FormatTable(t *testing.T) {
	tests := []struct {
		name string
		data []interface{}
		want []string
	}{
		{
			name: "empty slice",
			data: []interface{}{},
			want: []string{"No data found"},
		},
		{
			name: "simple objects",
			data: []interface{}{
				map[string]interface{}{"name": "Alice", "age": 30},
				map[string]interface{}{"name": "Bob", "age": 25},
			},
			want: []string{"Alice", "30", "Bob", "25"},
		},
		{
			name: "mixed types",
			data: []interface{}{
				"simple string",
				42,
				map[string]interface{}{"key": "value"},
			},
			want: []string{"simple string", "42", "key", "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputText, &buf)

			err := formatter.formatTable(tt.data)
			require.NoError(t, err)

			output := buf.String()
			for _, expected := range tt.want {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatter_FormatSingleObject(t *testing.T) {
	type TestStruct struct {
		Name   string
		Age    int
		Active bool
	}

	tests := []struct {
		name string
		data interface{}
		want []string
	}{
		{
			name: "struct object",
			data: TestStruct{
				Name:   "Alice",
				Age:    30,
				Active: true,
			},
			want: []string{"Name", "Alice", "Age", "30", "Active", "true"},
		},
		{
			name: "pointer to struct",
			data: &TestStruct{
				Name:   "Bob",
				Age:    25,
				Active: false,
			},
			want: []string{"Name", "Bob", "Age", "25", "Active", "false"},
		},
		{
			name: "non-struct type",
			data: "simple string",
			want: []string{"simple string"},
		},
		{
			name: "number",
			data: 42,
			want: []string{"42"},
		},
		{
			name: "nil data",
			data: nil,
			want: []string{}, // Should return without error, no output
		},
		{
			name: "map",
			data: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			want: []string{"key1", "value1", "key2", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputText, &buf)

			err := formatter.formatSingleObject(tt.data)
			require.NoError(t, err)

			output := buf.String()
			for _, expected := range tt.want {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatter_FormatText_Comprehensive(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
		want []string
	}{
		{
			name: "nil data",
			data: nil,
			want: []string{}, // Should handle gracefully
		},
		{
			name: "slice triggers formatTable",
			data: []interface{}{
				map[string]interface{}{"name": "test"},
			},
			want: []string{"name", "test"},
		},
		{
			name: "TableData triggers formatTableData",
			data: TableData{
				Headers: []string{"Col1"},
				Rows:    [][]string{{"Value1"}},
			},
			want: []string{"Col1", "Value1"},
		},
		{
			name: "single object triggers formatSingleObject",
			data: map[string]interface{}{"key": "value"},
			want: []string{"key", "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputText, &buf)

			err := formatter.formatText(tt.data)
			require.NoError(t, err)

			output := buf.String()
			for _, expected := range tt.want {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatter_Format_AllFormats(t *testing.T) {
	testData := map[string]interface{}{
		"name": "test",
		"age":  30,
	}

	tests := []struct {
		format config.OutputFormat
		checks []string
	}{
		{
			format: config.OutputJSON,
			checks: []string{"\"name\"", "\"test\"", "\"age\"", "30"},
		},
		{
			format: config.OutputYAML,
			checks: []string{"name: test", "age: 30"},
		},
		{
			format: config.OutputText,
			checks: []string{"name", "test", "age", "30"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(tt.format, &buf)

			err := formatter.Format(testData)
			require.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output)

			for _, check := range tt.checks {
				assert.Contains(t, output, check)
			}
		})
	}
}

func TestMaskConnectionString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "srv with creds",
			in:   "mongodb+srv://user:pass@cluster.mongodb.net/db?retryWrites=true&w=majority",
			want: "mongodb+srv://user:***@cluster.mongodb.net/db?retryWrites=true&w=majority",
		},
		{
			name: "standard with encoded password",
			in:   "mongodb://alice:p%40ss@localhost:27017/?ssl=true",
			want: "mongodb://alice:***@localhost:27017/?ssl=true",
		},
		{
			name: "no creds",
			in:   "mongodb+srv://cluster.mongodb.net/db",
			want: "mongodb+srv://cluster.mongodb.net/db",
		},
		{
			name: "username only",
			in:   "mongodb://user@localhost:27017/db",
			want: "mongodb://user@localhost:27017/db",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := maskConnectionString(tt.in)
			if got != tt.want {
				t.Fatalf("maskConnectionString() = %q, want %q", got, tt.want)
			}
			// Ensure masking occurs only when userinfo contains a password
			parts := strings.SplitN(tt.in, "://", 2)
			if len(parts) == 2 {
				rest := parts[1]
				if at := strings.Index(rest, "@"); at != -1 {
					userinfo := rest[:at]
					if strings.Contains(userinfo, ":") {
						if !strings.Contains(got, ":***@") {
							t.Fatalf("expected masked credentials in output: %q", got)
						}
					}
				}
			}
		})
	}
}

func TestFormatter_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter(config.OutputFormat("unsupported"), &buf)

	err := formatter.Format(map[string]interface{}{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format: unsupported")
}

func TestNewFormatter_ValidFormats(t *testing.T) {
	var buf bytes.Buffer

	formats := []config.OutputFormat{config.OutputJSON, config.OutputYAML, config.OutputText, config.OutputTable}
	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			formatter := NewFormatter(format, &buf)
			assert.NotNil(t, formatter)
			assert.Equal(t, format, formatter.format)
			assert.Equal(t, &buf, formatter.writer)
		})
	}
}

func TestFormatter_FormatValue_Extended(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{
			name:  "string value",
			value: "test string",
			want:  "test string",
		},
		{
			name:  "integer value",
			value: 42,
			want:  "42",
		},
		{
			name:  "boolean true",
			value: true,
			want:  "true",
		},
		{
			name:  "boolean false",
			value: false,
			want:  "false",
		},
		{
			name:  "nil value",
			value: nil,
			want:  "", // formatSingleObject returns early for nil without output
		},
		{
			name:  "float value",
			value: 3.14,
			want:  "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(config.OutputText, &buf)

			// Call formatValue if it's exported, otherwise test through formatSingleObject
			err := formatter.formatSingleObject(tt.value)
			require.NoError(t, err)

			output := buf.String()
			if tt.want == "" {
				// For nil case, just check no error occurred
				assert.NotNil(t, output) // Can be empty string
			} else {
				assert.Contains(t, output, tt.want)
			}
		})
	}
}

func TestTableData_Structure(t *testing.T) {
	// Test TableData struct creation and access
	td := TableData{
		Headers: []string{"Name", "Age"},
		Rows: [][]string{
			{"Alice", "30"},
			{"Bob", "25"},
		},
	}

	assert.Equal(t, []string{"Name", "Age"}, td.Headers)
	assert.Len(t, td.Rows, 2)
	assert.Equal(t, []string{"Alice", "30"}, td.Rows[0])
	assert.Equal(t, []string{"Bob", "25"}, td.Rows[1])
}

func TestFormatter_ComplexDataStructures(t *testing.T) {
	type User struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		IsActive bool   `json:"is_active"`
	}

	complexData := []User{
		{ID: 1, Name: "Alice Johnson", Email: "alice@example.com", IsActive: true},
		{ID: 2, Name: "Bob Smith", Email: "bob@example.com", IsActive: false},
	}

	formats := []config.OutputFormat{config.OutputJSON, config.OutputYAML, config.OutputText}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			var buf bytes.Buffer
			formatter := NewFormatter(format, &buf)

			err := formatter.Format(complexData)
			require.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output)

			// Check that user data appears in output
			assert.Contains(t, output, "Alice Johnson")
			assert.Contains(t, output, "Bob Smith")
			assert.Contains(t, output, "alice@example.com")
			assert.Contains(t, output, "bob@example.com")
		})
	}
}
