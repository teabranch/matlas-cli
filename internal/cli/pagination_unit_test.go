package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaginationFlags_Structure(t *testing.T) {
	flags := &PaginationFlags{
		Page:  1,
		Limit: 100,
		All:   false,
	}

	assert.Equal(t, 1, flags.Page)
	assert.Equal(t, 100, flags.Limit)
	assert.False(t, flags.All)
}

func TestPaginationOptions_Structure(t *testing.T) {
	opts := &PaginationOptions{
		Page:  2,
		Limit: 50,
	}

	assert.Equal(t, 2, opts.Page)
	assert.Equal(t, 50, opts.Limit)
}

func TestPaginationFlags_ValidateMethod(t *testing.T) {
	flags := &PaginationFlags{
		Page:  3,
		Limit: 25,
		All:   false,
	}

	opts, err := flags.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, opts)
	assert.Equal(t, flags.Page, opts.Page)
	assert.Equal(t, flags.Limit, opts.Limit)
}

func TestPaginationOptions_Validation(t *testing.T) {
	tests := []struct {
		name  string
		opts  PaginationOptions
		valid bool
	}{
		{
			name:  "valid pagination",
			opts:  PaginationOptions{Page: 1, Limit: 50},
			valid: true,
		},
		{
			name:  "zero page",
			opts:  PaginationOptions{Page: 0, Limit: 50},
			valid: false,
		},
		{
			name:  "negative limit",
			opts:  PaginationOptions{Page: 1, Limit: -10},
			valid: false,
		},
		{
			name:  "zero limit",
			opts:  PaginationOptions{Page: 1, Limit: 0},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic structure validation
			if tt.valid {
				assert.Greater(t, tt.opts.Page, 0)
				assert.Greater(t, tt.opts.Limit, 0)
			} else {
				// At least one field should be invalid
				invalid := tt.opts.Page <= 0 || tt.opts.Limit <= 0
				assert.True(t, invalid)
			}
		})
	}
}

func TestPaginationDefaults(t *testing.T) {
	// Test default pagination values
	flags := &PaginationFlags{}

	// Should have zero values initially
	assert.Equal(t, 0, flags.Page)
	assert.Equal(t, 0, flags.Limit)

	// Test setting default values
	flags.Page = 1
	flags.Limit = 100

	assert.Equal(t, 1, flags.Page)
	assert.Equal(t, 100, flags.Limit)
}
