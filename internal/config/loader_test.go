package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/config"
)

func TestLoad_Precedence(t *testing.T) {
	// 1. YAML file with baseline values
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := []byte("projectId: yamlProj\nclusterName: yamlCluster\n")
	if err := os.WriteFile(yamlPath, yamlContent, 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("write yaml: %v", err)
	}

	// 2. Environment variable that should be overridden by flag
	t.Setenv("ATLAS_PROJECT_ID", "envProj")

	// 3. Cobra command with flag override
	cmd := &cobra.Command{}
	cmd.Flags().String("project-id", "", "")
	if err := cmd.Flags().Set("project-id", "flagProj"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	// Parse the flag set so cobra/viper see values as "Changed".
	if err := cmd.ParseFlags([]string{"--project-id", "flagProj"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	cfg, err := config.Load(cmd, yamlPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := cfg.ProjectID, "flagProj"; got != want {
		t.Errorf("ProjectID precedence mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.ClusterName, "yamlCluster"; got != want {
		t.Errorf("ClusterName from YAML: got %s want %s", got, want)
	}
}

func TestResolveAPIKey(t *testing.T) {
	// Test Config field takes precedence
	cfg := &config.Config{APIKey: "cfgKey"}
	if key, _ := cfg.ResolveAPIKey(); key != "cfgKey" {
		t.Errorf("expected cfgKey, got %s", key)
	}

	// Test env var takes precedence over empty config
	cfg2 := &config.Config{}
	t.Setenv("ATLAS_API_KEY", "envKey")
	if key, _ := cfg2.ResolveAPIKey(); key != "envKey" {
		t.Errorf("expected envKey, got %s", key)
	}

	// Test error when no key found
	cfg3 := &config.Config{}
	t.Setenv("ATLAS_API_KEY", "")
	if _, err := cfg3.ResolveAPIKey(); err == nil {
		t.Error("expected error when no API key found")
	}
}

func TestResolvePublicKey(t *testing.T) {
	// Test Config field takes precedence
	cfg := &config.Config{PublicKey: "cfgPubKey"}
	if key, _ := cfg.ResolvePublicKey(); key != "cfgPubKey" {
		t.Errorf("expected cfgPubKey, got %s", key)
	}

	// Test env var takes precedence over empty config
	cfg2 := &config.Config{}
	t.Setenv("ATLAS_PUB_KEY", "envPubKey")
	if key, _ := cfg2.ResolvePublicKey(); key != "envPubKey" {
		t.Errorf("expected envPubKey, got %s", key)
	}

	// Test MATLAS_PUB_KEY fallback
	cfg3 := &config.Config{}
	t.Setenv("ATLAS_PUB_KEY", "")
	t.Setenv("MATLAS_PUB_KEY", "matlasKey")
	if key, _ := cfg3.ResolvePublicKey(); key != "matlasKey" {
		t.Errorf("expected matlasKey, got %s", key)
	}

	// Test error when no key found
	cfg4 := &config.Config{}
	t.Setenv("ATLAS_PUB_KEY", "")
	t.Setenv("MATLAS_PUB_KEY", "")
	if _, err := cfg4.ResolvePublicKey(); err == nil {
		t.Error("expected error when no public key found")
	}
}
