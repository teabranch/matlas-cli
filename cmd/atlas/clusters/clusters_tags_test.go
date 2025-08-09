package clusters

import (
	"reflect"
	"testing"
)

func TestParseTagsFromStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "single tag",
			input:    []string{"environment=production"},
			expected: map[string]string{"environment": "production"},
			wantErr:  false,
		},
		{
			name:     "multiple tags",
			input:    []string{"environment=production", "team=backend", "cost-center=engineering"},
			expected: map[string]string{"environment": "production", "team": "backend", "cost-center": "engineering"},
			wantErr:  false,
		},
		{
			name:     "tag with spaces in value",
			input:    []string{"description=my production cluster"},
			expected: map[string]string{"description": "my production cluster"},
			wantErr:  false,
		},
		{
			name:     "tag with equals in value",
			input:    []string{"config=key=value"},
			expected: map[string]string{"config": "key=value"},
			wantErr:  false,
		},
		{
			name:     "invalid format - no equals",
			input:    []string{"environment"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid format - empty key",
			input:    []string{"=production"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "duplicate keys",
			input:    []string{"environment=production", "environment=staging"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "empty value is allowed",
			input:    []string{"optional="},
			expected: map[string]string{"optional": ""},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTagsFromStrings(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTagsFromStrings() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseTagsFromStrings() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseTagsFromStrings() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertTagsToAtlasFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected int // number of tags expected
	}{
		{
			name:     "empty tags",
			input:    nil,
			expected: 0,
		},
		{
			name:     "single tag",
			input:    map[string]string{"environment": "production"},
			expected: 1,
		},
		{
			name:     "multiple tags",
			input:    map[string]string{"environment": "production", "team": "backend"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTagsToAtlasFormat(tt.input)

			if tt.expected == 0 {
				if result != nil {
					t.Errorf("convertTagsToAtlasFormat() expected nil but got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("convertTagsToAtlasFormat() expected non-nil result")
				return
			}

			if len(*result) != tt.expected {
				t.Errorf("convertTagsToAtlasFormat() expected %d tags but got %d", tt.expected, len(*result))
			}

			// Verify the content
			for _, tag := range *result {
				if expectedValue, exists := tt.input[tag.Key]; exists {
					if tag.Value != expectedValue {
						t.Errorf("convertTagsToAtlasFormat() tag %s has value %s, want %s", tag.Key, tag.Value, expectedValue)
					}
				} else {
					t.Errorf("convertTagsToAtlasFormat() unexpected tag key: %s", tag.Key)
				}
			}
		})
	}
}
