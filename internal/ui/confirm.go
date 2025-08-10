package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ConfirmationPrompt handles interactive confirmation prompts
type ConfirmationPrompt struct {
	input  io.Reader
	output io.Writer
	// skipPrompts indicates if all prompts should be skipped (--yes flag)
	skipPrompts bool
	// nonInteractive indicates if we're in non-interactive mode
	nonInteractive bool
}

// NewConfirmationPrompt creates a new confirmation prompt handler
func NewConfirmationPrompt(skipPrompts, nonInteractive bool) *ConfirmationPrompt {
	return &ConfirmationPrompt{
		input:          os.Stdin,
		output:         os.Stdout,
		skipPrompts:    skipPrompts,
		nonInteractive: nonInteractive,
	}
}

// NewConfirmationPromptWithIO creates a confirmation prompt with custom I/O (useful for testing)
func NewConfirmationPromptWithIO(input io.Reader, output io.Writer, skipPrompts, nonInteractive bool) *ConfirmationPrompt {
	return &ConfirmationPrompt{
		input:          input,
		output:         output,
		skipPrompts:    skipPrompts,
		nonInteractive: nonInteractive,
	}
}

// Confirm prompts the user for confirmation and returns true if they confirm
func (c *ConfirmationPrompt) Confirm(message string) (bool, error) {
	// If --yes flag is set, automatically confirm
	if c.skipPrompts {
		return true, nil
	}

	// If in non-interactive mode without --yes, deny by default for safety
	if c.nonInteractive {
		return false, fmt.Errorf("operation requires confirmation but running in non-interactive mode (use --yes to confirm)")
	}

	// Display the prompt
	_, _ = fmt.Fprintf(c.output, "%s [y/N]: ", message)

	// Read user input
	scanner := bufio.NewScanner(c.input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("failed to read user input: %w", err)
		}
		// EOF or no input - treat as "no"
		return false, nil
	}

	input := strings.TrimSpace(strings.ToLower(scanner.Text()))

	// Only accept explicit "y" or "yes" as confirmation
	switch input {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// ConfirmDeletion provides a specialized confirmation for deletion operations
func (c *ConfirmationPrompt) ConfirmDeletion(resourceType, resourceName string) (bool, error) {
	message := fmt.Sprintf("Are you sure you want to delete %s '%s'? This action cannot be undone.",
		resourceType, resourceName)
	return c.Confirm(message)
}

// ConfirmDestructiveAction provides confirmation for potentially destructive actions
func (c *ConfirmationPrompt) ConfirmDestructiveAction(action, target string) (bool, error) {
	message := fmt.Sprintf("Are you sure you want to %s %s?", action, target)
	return c.Confirm(message)
}

// ConfirmWithDetails shows detailed information before asking for confirmation
func (c *ConfirmationPrompt) ConfirmWithDetails(action string, details []string) (bool, error) {
	// If --yes flag is set, automatically confirm
	if c.skipPrompts {
		return true, nil
	}

	// If in non-interactive mode without --yes, deny by default for safety
	if c.nonInteractive {
		return false, fmt.Errorf("operation requires confirmation but running in non-interactive mode (use --yes to confirm)")
	}

	// Display the action and details
	_, _ = fmt.Fprintf(c.output, "%s\n\n", action)

	if len(details) > 0 {
		_, _ = fmt.Fprintf(c.output, "Details:\n")
		for _, detail := range details {
			_, _ = fmt.Fprintf(c.output, "  - %s\n", detail)
		}
		_, _ = fmt.Fprintf(c.output, "\n")
	}

	return c.Confirm("Do you want to proceed?")
}

// WarnAndConfirm displays a warning message before asking for confirmation
func (c *ConfirmationPrompt) WarnAndConfirm(warning, action string) (bool, error) {
	// If --yes flag is set, automatically confirm
	if c.skipPrompts {
		return true, nil
	}

	// If in non-interactive mode without --yes, deny by default for safety
	if c.nonInteractive {
		return false, fmt.Errorf("operation requires confirmation but running in non-interactive mode (use --yes to confirm)")
	}

	// Display warning
	_, _ = fmt.Fprintf(c.output, "⚠️  WARNING: %s\n\n", warning)

	return c.Confirm(action)
}
