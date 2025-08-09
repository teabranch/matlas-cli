package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// PaginationFlags holds standard pagination flag values
type PaginationFlags struct {
	Page  int
	Limit int
	All   bool
}

// PaginationOptions represents resolved pagination parameters
type PaginationOptions struct {
	Page  int
	Limit int
}

const (
	// DefaultPageSize is the default number of items per page
	DefaultPageSize = 50
	// MaxPageSize is the maximum allowed page size
	MaxPageSize = 500
	// MinPageSize is the minimum allowed page size
	MinPageSize = 1
)

// AddPaginationFlags adds standard pagination flags to a command
func AddPaginationFlags(cmd *cobra.Command, flags *PaginationFlags) {
	cmd.Flags().IntVar(&flags.Page, "page", 1, "Page number (1-based)")
	cmd.Flags().IntVar(&flags.Limit, "limit", DefaultPageSize, fmt.Sprintf("Number of items per page (1-%d)", MaxPageSize))
	cmd.Flags().BoolVar(&flags.All, "all", false, "Retrieve all items (overrides pagination)")
}

// Validate validates pagination flags and returns resolved options
func (p *PaginationFlags) Validate() (*PaginationOptions, error) {
	if p.All {
		// When --all is specified, we use a large limit and reset page to 1
		return &PaginationOptions{
			Page:  1,
			Limit: MaxPageSize,
		}, nil
	}

	if p.Page < 1 {
		return nil, fmt.Errorf("page must be >= 1, got %d", p.Page)
	}

	if p.Limit < MinPageSize {
		return nil, fmt.Errorf("limit must be >= %d, got %d", MinPageSize, p.Limit)
	}

	if p.Limit > MaxPageSize {
		return nil, fmt.Errorf("limit must be <= %d, got %d", MaxPageSize, p.Limit)
	}

	return &PaginationOptions{
		Page:  p.Page,
		Limit: p.Limit,
	}, nil
}

// CalculateSkip calculates the number of items to skip based on page and limit
func (o *PaginationOptions) CalculateSkip() int {
	return (o.Page - 1) * o.Limit
}

// ShouldPaginate returns true if pagination should be applied
func (o *PaginationOptions) ShouldPaginate() bool {
	return o.Page > 1 || o.Limit < MaxPageSize
}

// PaginationInfo represents pagination metadata for display
type PaginationInfo struct {
	Page       int
	Limit      int
	Total      int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

// NewPaginationInfo creates pagination info from options and total count
func NewPaginationInfo(opts *PaginationOptions, total int) *PaginationInfo {
	totalPages := (total + opts.Limit - 1) / opts.Limit // Ceiling division
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginationInfo{
		Page:       opts.Page,
		Limit:      opts.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    opts.Page < totalPages,
		HasPrev:    opts.Page > 1,
	}
}

// DisplayInfo returns a human-readable pagination summary
func (p *PaginationInfo) DisplayInfo() string {
	if p.Total == 0 {
		return "No items found"
	}

	start := (p.Page-1)*p.Limit + 1
	end := start + p.Limit - 1
	if end > p.Total {
		end = p.Total
	}

	if p.TotalPages == 1 {
		return fmt.Sprintf("Showing %d item(s)", p.Total)
	}

	return fmt.Sprintf("Showing %d-%d of %d items (page %d of %d)",
		start, end, p.Total, p.Page, p.TotalPages)
}
