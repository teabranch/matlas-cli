package infra

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teabranch/matlas-cli/internal/apply"
)

func TestValidateDiffOptions(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		output      string
		showContext int
		expectErr   bool
	}{
		{name: "missing files", files: []string{}, output: "table", showContext: 3, expectErr: true},
		{name: "invalid output", files: []string{"config.yaml"}, output: "xml", showContext: 3, expectErr: true},
		{name: "negative context", files: []string{"config.yaml"}, output: "table", showContext: -1, expectErr: true},
		{name: "valid table", files: []string{"config.yaml"}, output: "table", showContext: 3, expectErr: false},
		{name: "valid unified", files: []string{"config.yaml"}, output: "unified", showContext: 0, expectErr: false},
		{name: "valid json", files: []string{"config.yaml"}, output: "json", showContext: 1, expectErr: false},
		{name: "valid yaml", files: []string{"config.yaml"}, output: "yaml", showContext: 2, expectErr: false},
		{name: "valid summary", files: []string{"config.yaml"}, output: "summary", showContext: 3, expectErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &DiffOptions{Files: tt.files, OutputFormat: tt.output, ShowContext: tt.showContext, Timeout: 1 * time.Minute}
			err := validateDiffOptions(opts)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDisplayDifferences_NoOperationsHonorsFormats(t *testing.T) {
	diff := &apply.Diff{
		Operations: []apply.Operation{},
		Summary:    apply.DiffSummary{TotalOperations: 0},
	}

	// Just ensure no errors are returned per format when no operations
	for _, format := range []string{"json", "yaml", "unified", "table", "summary"} {
		t.Run("format_"+format, func(t *testing.T) {
			err := displayDifferences(diff, &DiffOptions{OutputFormat: format, Timeout: 1 * time.Minute})
			assert.NoError(t, err)
		})
	}
}
