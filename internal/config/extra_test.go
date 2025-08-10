package config_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/config"
)

// -------------------- Config validation --------------------

func TestConfigValidate(t *testing.T) {
	cfg := config.New()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}

	cfg.Output = config.OutputFormat("invalid")
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for unsupported output format")
	}

	cfg.Output = config.OutputJSON
	cfg.Timeout = 0
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for zero timeout")
	}
}

// -------------------- Context helper --------------------

func TestNewContextWithTimeout(t *testing.T) {
	cfg := &config.Config{Timeout: 50 * time.Millisecond}
	ctx, cancel := config.NewContext(context.Background(), cfg)
	defer cancel()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Fatalf("unexpected ctx.Err(): %v", ctx.Err())
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("context did not time out as expected")
	}
}

func TestNewContextWithoutTimeout(t *testing.T) {
	cfg := &config.Config{Timeout: 0}
	ctx, cancel := config.NewContext(context.Background(), cfg)
	defer cancel()

	if deadline, ok := ctx.Deadline(); ok {
		t.Fatalf("expected no deadline, got %v", deadline)
	}
}

// -------------------- Loader edge cases --------------------

func TestLoad_NoConfigFile(t *testing.T) {
	// Ensure HOME points to empty dir
	emptyHome := t.TempDir()
	t.Setenv("HOME", emptyHome)

	cfg, err := config.Load(nil, "")
	if err != nil {
		t.Fatalf("Load without config file should succeed: %v", err)
	}
	if cfg.Output != config.OutputTable {
		t.Fatalf("expected default output, got %s", cfg.Output)
	}
}

func TestLoad_BadYAML(t *testing.T) {
	tmpDir := t.TempDir()
	badPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(badPath, []byte("foo: [unbalanced"), 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("write bad yaml: %v", err)
	}

	if _, err := config.Load(nil, badPath); err == nil {
		t.Fatalf("expected error for invalid YAML")
	}
}

// -------------------- Flag to field mapping --------------------

func TestLoad_FlagClusterNameMapping(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("cluster-name", "", "")
	// Simulate user passing flag via CLI
	if err := cmd.ParseFlags([]string{"--cluster-name", "mycluster"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	cfg, err := config.Load(cmd, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClusterName != "mycluster" {
		t.Fatalf("expected clusterName mapped from flag, got %s", cfg.ClusterName)
	}
}

// -------------------- Credential negative path --------------------

func TestResolveAPIKey_NotFound(t *testing.T) {
	// Clear env vars to ensure we test the error case
	t.Setenv("ATLAS_API_KEY", "")
	t.Setenv("MATLAS_API_KEY", "")

	cfg := &config.Config{}
	if _, err := cfg.ResolveAPIKey(); !errors.Is(err, config.ErrAPIKeyNotFound) {
		t.Fatalf("expected ErrAPIKeyNotFound, got %v", err)
	}
}

func TestResolvePublicKey_NotFound(t *testing.T) {
	// Clear env vars to ensure we test the error case
	t.Setenv("ATLAS_PUB_KEY", "")
	t.Setenv("MATLAS_PUB_KEY", "")

	cfg := &config.Config{}
	if _, err := cfg.ResolvePublicKey(); !errors.Is(err, config.ErrPublicKeyNotFound) {
		t.Fatalf("expected ErrPublicKeyNotFound, got %v", err)
	}
}

func TestCreateAtlasClient(t *testing.T) {
	// If credentials are present we can't exercise the "missing creds" path
	if os.Getenv("ATLAS_PUB_KEY") != "" || os.Getenv("ATLAS_API_KEY") != "" {
		t.Skip("Atlas credentials present in environment – skipping error-path test")
	}

	client, err := config.New().CreateAtlasClient()
	require.Error(t, err)
	require.Nil(t, client)

	// Should fail with API key not found error (first credential checked)
	assert.True(t, errors.Is(err, config.ErrAPIKeyNotFound), "expected ErrAPIKeyNotFound, got %v", err)
}

func TestCreateAtlasClient_MissingAPIKey(t *testing.T) {
	// Clear env vars
	t.Setenv("ATLAS_API_KEY", "")
	t.Setenv("MATLAS_API_KEY", "")

	cfg := &config.Config{
		PublicKey: "test-pub-key",
	}

	_, err := cfg.CreateAtlasClient()
	if !errors.Is(err, config.ErrAPIKeyNotFound) {
		t.Fatalf("expected ErrAPIKeyNotFound, got %v", err)
	}
}

func TestCreateAtlasClient_MissingPublicKey(t *testing.T) {
	// Clear env vars
	t.Setenv("ATLAS_PUB_KEY", "")
	t.Setenv("MATLAS_PUB_KEY", "")

	cfg := &config.Config{
		APIKey: "test-api-key",
	}

	_, err := cfg.CreateAtlasClient()
	if !errors.Is(err, config.ErrPublicKeyNotFound) {
		t.Fatalf("expected ErrPublicKeyNotFound, got %v", err)
	}
}

func TestATLAS_PrefixPriority(t *testing.T) {
	// Test that ATLAS_ variables take precedence over MATLAS_ variables

	// Set both ATLAS_ and MATLAS_ variables
	t.Setenv("ATLAS_API_KEY", "atlas_api_key")
	t.Setenv("MATLAS_API_KEY", "matlas_api_key")
	t.Setenv("ATLAS_PUB_KEY", "atlas_pub_key")
	t.Setenv("MATLAS_PUB_KEY", "matlas_pub_key")

	cfg := &config.Config{}

	// Test API key resolution
	apiKey, err := cfg.ResolveAPIKey()
	if err != nil {
		t.Fatalf("Error resolving API key: %v", err)
	}
	if apiKey != "atlas_api_key" { //nolint:gosec // test credential comparison
		t.Errorf("Expected ATLAS_API_KEY to take precedence, got %s", apiKey)
	}

	// Test public key resolution
	pubKey, err := cfg.ResolvePublicKey()
	if err != nil {
		t.Fatalf("Error resolving public key: %v", err)
	}
	if pubKey != "atlas_pub_key" {
		t.Errorf("Expected ATLAS_PUB_KEY to take precedence, got %s", pubKey)
	}
}

func TestMATLAS_FallbackWorks(t *testing.T) {
	// Test that MATLAS_ variables work when ATLAS_ variables are not set

	// Clear ATLAS_ variables and set only MATLAS_ variables
	t.Setenv("ATLAS_API_KEY", "")
	t.Setenv("MATLAS_API_KEY", "matlas_api_key")
	t.Setenv("ATLAS_PUB_KEY", "")
	t.Setenv("MATLAS_PUB_KEY", "matlas_pub_key")

	cfg := &config.Config{}

	// Test API key resolution
	apiKey, err := cfg.ResolveAPIKey()
	if err != nil {
		t.Fatalf("Error resolving API key: %v", err)
	}
	if apiKey != "matlas_api_key" { //nolint:gosec // test credential comparison
		t.Errorf("Expected MATLAS_API_KEY fallback to work, got %s", apiKey)
	}

	// Test public key resolution
	pubKey, err := cfg.ResolvePublicKey()
	if err != nil {
		t.Fatalf("Error resolving public key: %v", err)
	}
	if pubKey != "matlas_pub_key" {
		t.Errorf("Expected MATLAS_PUB_KEY fallback to work, got %s", pubKey)
	}
}
