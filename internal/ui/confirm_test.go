package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfirmationPrompt(t *testing.T) {
	tests := []struct {
		name           string
		skipPrompts    bool
		nonInteractive bool
	}{
		{"interactive with prompts", false, false},
		{"interactive skip prompts", true, false},
		{"non-interactive with prompts", false, true},
		{"non-interactive skip prompts", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := NewConfirmationPrompt(tt.skipPrompts, tt.nonInteractive)

			require.NotNil(t, prompt)
			assert.Equal(t, tt.skipPrompts, prompt.skipPrompts)
			assert.Equal(t, tt.nonInteractive, prompt.nonInteractive)
			assert.NotNil(t, prompt.input)
			assert.NotNil(t, prompt.output)
		})
	}
}

func TestNewConfirmationPromptWithIO(t *testing.T) {
	input := strings.NewReader("y\n")
	var output bytes.Buffer

	prompt := NewConfirmationPromptWithIO(input, &output, false, false)

	require.NotNil(t, prompt)
	assert.Equal(t, input, prompt.input)
	assert.Equal(t, &output, prompt.output)
	assert.False(t, prompt.skipPrompts)
	assert.False(t, prompt.nonInteractive)
}

func TestConfirmationPrompt_Confirm_SkipPrompts(t *testing.T) {
	var output bytes.Buffer
	prompt := NewConfirmationPromptWithIO(nil, &output, true, false)

	result, err := prompt.Confirm("Are you sure?")

	assert.NoError(t, err)
	assert.True(t, result)
	// Should not write anything to output when skipping
	assert.Empty(t, output.String())
}

func TestConfirmationPrompt_Confirm_NonInteractive(t *testing.T) {
	var output bytes.Buffer
	prompt := NewConfirmationPromptWithIO(nil, &output, false, true)

	result, err := prompt.Confirm("Are you sure?")

	assert.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "non-interactive mode")
	assert.Contains(t, err.Error(), "use --yes")
}

func TestConfirmationPrompt_Confirm_NonInteractiveWithSkip(t *testing.T) {
	var output bytes.Buffer
	prompt := NewConfirmationPromptWithIO(nil, &output, true, true)

	result, err := prompt.Confirm("Are you sure?")

	assert.NoError(t, err)
	assert.True(t, result)
	// Skip prompts takes precedence over non-interactive
}

func TestConfirmationPrompt_Confirm_YesResponses(t *testing.T) {
	yesInputs := []string{"y", "Y", "yes", "YES", "Yes"}

	for _, input := range yesInputs {
		t.Run("input: "+input, func(t *testing.T) {
			inputReader := strings.NewReader(input + "\n")
			var output bytes.Buffer
			prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

			result, err := prompt.Confirm("Are you sure?")

			assert.NoError(t, err)
			assert.True(t, result)
			assert.Contains(t, output.String(), "Are you sure?")
		})
	}
}

func TestConfirmationPrompt_Confirm_NoResponses(t *testing.T) {
	noInputs := []string{"n", "N", "no", "NO", "No", "false", "FALSE", "False", "true", "TRUE", "True"}

	for _, input := range noInputs {
		t.Run("input: "+input, func(t *testing.T) {
			inputReader := strings.NewReader(input + "\n")
			var output bytes.Buffer
			prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

			result, err := prompt.Confirm("Are you sure?")

			assert.NoError(t, err)
			assert.False(t, result)
			assert.Contains(t, output.String(), "Are you sure?")
		})
	}
}

func TestConfirmationPrompt_Confirm_InvalidResponses(t *testing.T) {
	invalidInputs := []string{"maybe", "sure", "1", "0", ""}

	for _, input := range invalidInputs {
		t.Run("input: "+input, func(t *testing.T) {
			// Invalid inputs are just treated as "no" - they don't prompt for retry
			inputReader := strings.NewReader(input + "\n")
			var output bytes.Buffer
			prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

			result, err := prompt.Confirm("Are you sure?")

			assert.NoError(t, err)
			assert.False(t, result) // Invalid input should be treated as "no"
			outputStr := output.String()
			assert.Contains(t, outputStr, "Are you sure?")
		})
	}
}

func TestConfirmationPrompt_Confirm_ReadError(t *testing.T) {
	// Use a reader that will return an error
	inputReader := &errorReader{}
	var output bytes.Buffer
	prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

	result, err := prompt.Confirm("Are you sure?")

	assert.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "failed to read user input")
}

func TestConfirmationPrompt_Confirm_WithCustomMessage(t *testing.T) {
	inputReader := strings.NewReader("yes\n")
	var output bytes.Buffer
	prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

	customMessage := "Do you want to delete all clusters?"
	result, err := prompt.Confirm(customMessage)

	assert.NoError(t, err)
	assert.True(t, result)
	assert.Contains(t, output.String(), customMessage)
	assert.Contains(t, output.String(), "[y/N]")
}

func TestConfirmationPrompt_ConfirmDeletion(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		resourceType   string
		resourceName   string
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "confirm deletion",
			input:          "y\n",
			resourceType:   "cluster",
			resourceName:   "my-cluster",
			expectedResult: true,
			expectError:    false,
		},
		{
			name:           "deny deletion",
			input:          "n\n",
			resourceType:   "project",
			resourceName:   "my-project",
			expectedResult: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputReader := strings.NewReader(tt.input)
			var output bytes.Buffer
			prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

			result, err := prompt.ConfirmDeletion(tt.resourceType, tt.resourceName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)

			outputStr := output.String()
			assert.Contains(t, outputStr, tt.resourceType)
			assert.Contains(t, outputStr, tt.resourceName)
			assert.Contains(t, outputStr, "cannot be undone")
		})
	}
}

func TestConfirmationPrompt_ConfirmDestructiveAction(t *testing.T) {
	inputReader := strings.NewReader("yes\n")
	var output bytes.Buffer
	prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

	result, err := prompt.ConfirmDestructiveAction("restart", "all clusters")

	assert.NoError(t, err)
	assert.True(t, result)

	outputStr := output.String()
	assert.Contains(t, outputStr, "restart")
	assert.Contains(t, outputStr, "all clusters")
}

// errorReader is a helper for testing read errors
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, assert.AnError
}

// Test edge cases and integration scenarios
func TestConfirmationPrompt_MultipleConfirmations(t *testing.T) {
	// Test each confirmation separately since they consume the reader
	t.Run("first confirmation", func(t *testing.T) {
		inputReader := strings.NewReader("y\n")
		var output bytes.Buffer
		prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

		result, err := prompt.Confirm("First question?")
		assert.NoError(t, err)
		assert.True(t, result)
		assert.Contains(t, output.String(), "First question?")
	})

	t.Run("second confirmation", func(t *testing.T) {
		inputReader := strings.NewReader("n\n")
		var output bytes.Buffer
		prompt := NewConfirmationPromptWithIO(inputReader, &output, false, false)

		result, err := prompt.Confirm("Second question?")
		assert.NoError(t, err)
		assert.False(t, result)
		assert.Contains(t, output.String(), "Second question?")
	})
}
