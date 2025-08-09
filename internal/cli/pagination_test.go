package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginationFlags_Validate(t *testing.T) {
	tests := []struct {
		name        string
		flags       PaginationFlags
		expectError bool
		expected    *PaginationOptions
	}{
		{
			name: "default values",
			flags: PaginationFlags{
				Page:  1,
				Limit: DefaultPageSize,
				All:   false,
			},
			expectError: false,
			expected: &PaginationOptions{
				Page:  1,
				Limit: DefaultPageSize,
			},
		},
		{
			name: "all flag set",
			flags: PaginationFlags{
				Page:  5,
				Limit: 10,
				All:   true,
			},
			expectError: false,
			expected: &PaginationOptions{
				Page:  1,
				Limit: MaxPageSize,
			},
		},
		{
			name: "valid custom values",
			flags: PaginationFlags{
				Page:  3,
				Limit: 25,
				All:   false,
			},
			expectError: false,
			expected: &PaginationOptions{
				Page:  3,
				Limit: 25,
			},
		},
		{
			name: "invalid page zero",
			flags: PaginationFlags{
				Page:  0,
				Limit: DefaultPageSize,
				All:   false,
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "invalid page negative",
			flags: PaginationFlags{
				Page:  -1,
				Limit: DefaultPageSize,
				All:   false,
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "limit too small",
			flags: PaginationFlags{
				Page:  1,
				Limit: 0,
				All:   false,
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "limit too large",
			flags: PaginationFlags{
				Page:  1,
				Limit: MaxPageSize + 1,
				All:   false,
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "minimum valid limit",
			flags: PaginationFlags{
				Page:  1,
				Limit: MinPageSize,
				All:   false,
			},
			expectError: false,
			expected: &PaginationOptions{
				Page:  1,
				Limit: MinPageSize,
			},
		},
		{
			name: "maximum valid limit",
			flags: PaginationFlags{
				Page:  1,
				Limit: MaxPageSize,
				All:   false,
			},
			expectError: false,
			expected: &PaginationOptions{
				Page:  1,
				Limit: MaxPageSize,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.flags.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPaginationOptions_CalculateSkip(t *testing.T) {
	tests := []struct {
		name     string
		opts     PaginationOptions
		expected int
	}{
		{
			name: "first page",
			opts: PaginationOptions{
				Page:  1,
				Limit: 10,
			},
			expected: 0,
		},
		{
			name: "second page",
			opts: PaginationOptions{
				Page:  2,
				Limit: 10,
			},
			expected: 10,
		},
		{
			name: "fifth page with large limit",
			opts: PaginationOptions{
				Page:  5,
				Limit: 50,
			},
			expected: 200,
		},
		{
			name: "page 10 with small limit",
			opts: PaginationOptions{
				Page:  10,
				Limit: 5,
			},
			expected: 45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.CalculateSkip()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPaginationOptions_ShouldPaginate(t *testing.T) {
	tests := []struct {
		name     string
		opts     PaginationOptions
		expected bool
	}{
		{
			name: "first page, max limit",
			opts: PaginationOptions{
				Page:  1,
				Limit: MaxPageSize,
			},
			expected: false,
		},
		{
			name: "first page, small limit",
			opts: PaginationOptions{
				Page:  1,
				Limit: 10,
			},
			expected: true,
		},
		{
			name: "second page, max limit",
			opts: PaginationOptions{
				Page:  2,
				Limit: MaxPageSize,
			},
			expected: true,
		},
		{
			name: "second page, small limit",
			opts: PaginationOptions{
				Page:  2,
				Limit: 10,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.ShouldPaginate()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewPaginationInfo(t *testing.T) {
	tests := []struct {
		name     string
		opts     *PaginationOptions
		total    int
		expected *PaginationInfo
	}{
		{
			name: "zero total",
			opts: &PaginationOptions{
				Page:  1,
				Limit: 10,
			},
			total: 0,
			expected: &PaginationInfo{
				Page:       1,
				Limit:      10,
				Total:      0,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
		},
		{
			name: "single page",
			opts: &PaginationOptions{
				Page:  1,
				Limit: 10,
			},
			total: 5,
			expected: &PaginationInfo{
				Page:       1,
				Limit:      10,
				Total:      5,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
		},
		{
			name: "multiple pages, first page",
			opts: &PaginationOptions{
				Page:  1,
				Limit: 10,
			},
			total: 25,
			expected: &PaginationInfo{
				Page:       1,
				Limit:      10,
				Total:      25,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    false,
			},
		},
		{
			name: "multiple pages, middle page",
			opts: &PaginationOptions{
				Page:  2,
				Limit: 10,
			},
			total: 25,
			expected: &PaginationInfo{
				Page:       2,
				Limit:      10,
				Total:      25,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    true,
			},
		},
		{
			name: "multiple pages, last page",
			opts: &PaginationOptions{
				Page:  3,
				Limit: 10,
			},
			total: 25,
			expected: &PaginationInfo{
				Page:       3,
				Limit:      10,
				Total:      25,
				TotalPages: 3,
				HasNext:    false,
				HasPrev:    true,
			},
		},
		{
			name: "exact page boundary",
			opts: &PaginationOptions{
				Page:  1,
				Limit: 10,
			},
			total: 20,
			expected: &PaginationInfo{
				Page:       1,
				Limit:      10,
				Total:      20,
				TotalPages: 2,
				HasNext:    true,
				HasPrev:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewPaginationInfo(tt.opts, tt.total)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPaginationInfo_DisplayInfo(t *testing.T) {
	tests := []struct {
		name     string
		info     *PaginationInfo
		expected string
	}{
		{
			name: "no items",
			info: &PaginationInfo{
				Page:       1,
				Limit:      10,
				Total:      0,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
			expected: "No items found",
		},
		{
			name: "single page with few items",
			info: &PaginationInfo{
				Page:       1,
				Limit:      10,
				Total:      5,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
			expected: "Showing 5 item(s)",
		},
		{
			name: "single page with exact limit",
			info: &PaginationInfo{
				Page:       1,
				Limit:      10,
				Total:      10,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
			expected: "Showing 10 item(s)",
		},
		{
			name: "first page of multiple",
			info: &PaginationInfo{
				Page:       1,
				Limit:      10,
				Total:      25,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    false,
			},
			expected: "Showing 1-10 of 25 items (page 1 of 3)",
		},
		{
			name: "middle page",
			info: &PaginationInfo{
				Page:       2,
				Limit:      10,
				Total:      25,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    true,
			},
			expected: "Showing 11-20 of 25 items (page 2 of 3)",
		},
		{
			name: "last page with partial items",
			info: &PaginationInfo{
				Page:       3,
				Limit:      10,
				Total:      25,
				TotalPages: 3,
				HasNext:    false,
				HasPrev:    true,
			},
			expected: "Showing 21-25 of 25 items (page 3 of 3)",
		},
		{
			name: "large numbers",
			info: &PaginationInfo{
				Page:       10,
				Limit:      50,
				Total:      1000,
				TotalPages: 20,
				HasNext:    true,
				HasPrev:    true,
			},
			expected: "Showing 451-500 of 1000 items (page 10 of 20)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.DisplayInfo()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddPaginationFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}
	flags := &PaginationFlags{}

	AddPaginationFlags(cmd, flags)

	// Check that flags were added
	pageFlag := cmd.Flags().Lookup("page")
	require.NotNil(t, pageFlag)
	assert.Equal(t, "Page number (1-based)", pageFlag.Usage)

	limitFlag := cmd.Flags().Lookup("limit")
	require.NotNil(t, limitFlag)
	assert.Contains(t, limitFlag.Usage, "Number of items per page")

	allFlag := cmd.Flags().Lookup("all")
	require.NotNil(t, allFlag)
	assert.Equal(t, "Retrieve all items (overrides pagination)", allFlag.Usage)
}

func TestPaginationConstants(t *testing.T) {
	assert.Equal(t, 50, DefaultPageSize)
	assert.Equal(t, 500, MaxPageSize)
	assert.Equal(t, 1, MinPageSize)

	// Ensure logical relationships
	assert.True(t, MinPageSize < DefaultPageSize)
	assert.True(t, DefaultPageSize < MaxPageSize)
}
