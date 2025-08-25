package search

import (
	"testing"
)

// Ensure the command is visible and has all expected subcommands available.
func TestNewSearchCmd_VisibleAndSubcommands(t *testing.T) {
	cmd := NewSearchCmd()
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}
	if cmd.Hidden {
		t.Fatalf("expected search command to be visible, but it was hidden")
	}
	
	// Verify command properties
	if cmd.Use != "search" {
		t.Errorf("expected Use to be 'search', got '%s'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be non-empty")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be non-empty")
	}
	
	// Verify aliases are present
	expectedAliases := []string{"search-index", "search-indexes"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}
	
	// Ensure all expected subcommands are present
	found := map[string]bool{}
	for _, c := range cmd.Commands() {
		found[c.Use] = true
	}
	expectedCommands := []string{"list", "get", "create", "update", "delete"}
	for _, expectedCmd := range expectedCommands {
		if !found[expectedCmd] {
			t.Errorf("expected subcommand '%s' to be present", expectedCmd)
		}
	}
	
	if len(found) != len(expectedCommands) {
		t.Errorf("expected exactly %d subcommands, got %d: %+v", len(expectedCommands), len(found), found)
	}
}
