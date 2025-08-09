package ui

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProgressIndicator(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		quiet   bool
	}{
		{"default", false, false},
		{"verbose", true, false},
		{"quiet", false, true},
		{"verbose and quiet", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicator := NewProgressIndicator(tt.verbose, tt.quiet)

			require.NotNil(t, indicator)
			assert.Equal(t, tt.verbose, indicator.verbose)
			assert.Equal(t, tt.quiet, indicator.quiet)
			assert.NotNil(t, indicator.output)
			assert.Nil(t, indicator.spinner)
		})
	}
}

func TestNewProgressIndicatorWithWriter(t *testing.T) {
	var buf bytes.Buffer
	indicator := NewProgressIndicatorWithWriter(&buf, true, false)

	require.NotNil(t, indicator)
	assert.Equal(t, &buf, indicator.output)
	assert.True(t, indicator.verbose)
	assert.False(t, indicator.quiet)
}

func TestProgressIndicator_StartStopSpinner(t *testing.T) {
	var buf bytes.Buffer
	indicator := NewProgressIndicatorWithWriter(&buf, false, false)

	// Start spinner
	indicator.StartSpinner("Loading...")
	assert.NotNil(t, indicator.spinner)

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Stop spinner
	indicator.StopSpinner("Done!")
	assert.Nil(t, indicator.spinner)

	// Should have written something to buffer
	output := buf.String()
	assert.NotEmpty(t, output)
}

func TestProgressIndicator_StartStopSpinner_Quiet(t *testing.T) {
	var buf bytes.Buffer
	indicator := NewProgressIndicatorWithWriter(&buf, false, true)

	// Start spinner in quiet mode
	indicator.StartSpinner("Loading...")
	assert.Nil(t, indicator.spinner) // Should not create spinner

	// Stop spinner
	indicator.StopSpinner("Done!")
	assert.Nil(t, indicator.spinner)

	// Should not write anything in quiet mode
	assert.Empty(t, buf.String())
}

func TestProgressIndicator_StopSpinnerWithError(t *testing.T) {
	var buf bytes.Buffer
	indicator := NewProgressIndicatorWithWriter(&buf, false, false)

	// Start spinner
	indicator.StartSpinner("Loading...")
	require.NotNil(t, indicator.spinner)

	// Stop with error
	indicator.StopSpinnerWithError("Failed!")
	assert.Nil(t, indicator.spinner)

	output := buf.String()
	assert.NotEmpty(t, output)
}

func TestProgressIndicator_Print(t *testing.T) {
	var buf bytes.Buffer
	indicator := NewProgressIndicatorWithWriter(&buf, false, false)

	indicator.Print("Information message")

	output := buf.String()
	assert.Contains(t, output, "Information message")
}

func TestProgressIndicator_Print_Quiet(t *testing.T) {
	var buf bytes.Buffer
	indicator := NewProgressIndicatorWithWriter(&buf, false, true)

	indicator.Print("Information message")

	// Should not output in quiet mode
	assert.Empty(t, buf.String())
}

func TestProgressIndicator_PrintVerbose(t *testing.T) {
	var buf bytes.Buffer

	t.Run("verbose mode", func(t *testing.T) {
		buf.Reset()
		indicator := NewProgressIndicatorWithWriter(&buf, true, false)
		indicator.PrintVerbose("Verbose message")

		output := buf.String()
		assert.Contains(t, output, "Verbose message")
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		buf.Reset()
		indicator := NewProgressIndicatorWithWriter(&buf, false, false)
		indicator.PrintVerbose("Verbose message")

		// Should not output in non-verbose mode
		assert.Empty(t, buf.String())
	})

	t.Run("verbose but quiet mode", func(t *testing.T) {
		buf.Reset()
		indicator := NewProgressIndicatorWithWriter(&buf, true, true)
		indicator.PrintVerbose("Verbose message")

		// Should not output when quiet
		assert.Empty(t, buf.String())
	})
}

// Test Spinner functionality
func TestNewSpinner(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Loading...")

	require.NotNil(t, spinner)
	assert.Equal(t, &buf, spinner.output)
	assert.Equal(t, "Loading...", spinner.message)
	assert.False(t, spinner.active)
	assert.NotNil(t, spinner.frames)
	assert.NotEmpty(t, spinner.frames)
	assert.NotNil(t, spinner.stopChan)
}

func TestSpinner_StartStop(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Loading...")

	// Start spinner
	spinner.Start()
	assert.True(t, spinner.active)

	// Give it a moment to spin
	time.Sleep(50 * time.Millisecond)

	// Stop spinner
	spinner.Stop("Done!")
	assert.False(t, spinner.active)

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Done!")
}

func TestSpinner_StopWithError(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Loading...")

	spinner.Start()
	spinner.StopWithError("Failed!")
	assert.False(t, spinner.active)

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Failed!")
}

// Note: UpdateMessage method doesn't exist in the actual Spinner implementation
// These tests are removed as they test non-existent functionality

func TestSpinner_StopWithoutStart(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Loading...")

	// Stop without starting - should not panic
	spinner.Stop("Done!")
	assert.False(t, spinner.active)
}

func TestSpinner_StartTwice(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Loading...")

	// Start twice - should not cause issues
	spinner.Start()
	assert.True(t, spinner.active)

	spinner.Start() // Second start should be ignored
	assert.True(t, spinner.active)

	spinner.Stop("Done!")
	assert.False(t, spinner.active)
}

func TestSpinner_CustomFrames(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf, "Loading...")

	// Check that it has default frames
	assert.NotEmpty(t, spinner.frames)
	assert.Contains(t, spinner.frames, "⠋")
	assert.Contains(t, spinner.frames, "⠙")
}
