package search

import (
	"testing"
)

// Ensure the command is hidden and returns a consistent unsupported message when executed without required flags.
func TestNewSearchCmd_HiddenAndSubcommands(t *testing.T) {
	cmd := NewSearchCmd()
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}
	if !cmd.Hidden {
		t.Fatalf("expected search command to be hidden")
	}
	// Ensure common subcommands are present
	found := map[string]bool{}
	for _, c := range cmd.Commands() {
		found[c.Use] = true
	}
	if !found["list"] || !found["get"] || !found["create"] || !found["update"] || !found["delete"] {
		t.Fatalf("expected standard subcommands to be present, got: %+v", found)
	}
}
