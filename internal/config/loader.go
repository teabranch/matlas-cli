package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Load constructs a new *Config by merging (in increasing precedence order):
//  1. built-in defaults (see New())
//  2. YAML config file (default $HOME/.ATLAS/config.yaml, override via --config / ATLAS_CONFIG_FILE)
//  3. environment variables prefixed with ATLAS_
//  4. command-line flags bound on the provided *cobra.Command
//
// The resulting configuration is validated before being returned.
//
// Pass nil for cmd if you do not wish to bind flags (e.g., in tests).
func Load(cmd *cobra.Command, explicitPath string) (*Config, error) {
	cfg := New()

	v := viper.New()

	// ---------- 1. Defaults ----------
	v.SetDefault("output", cfg.Output)
	v.SetDefault("timeout", cfg.Timeout)

	// ---------- 2. Config file ----------
	// Resolve config file path
	if explicitPath == "" {
		if envPath := os.Getenv("ATLAS_CONFIG_FILE"); envPath != "" {
			explicitPath = envPath
		}
	}

	if explicitPath != "" {
		v.SetConfigFile(explicitPath)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home dir: %w", err)
		}
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(filepath.Join(homeDir, DefaultConfigDir))
	}

	if err := v.ReadInConfig(); err != nil {
		// If the file is missing we continue with env + defaults. Any other error is fatal.
		if _, isNotFound := err.(viper.ConfigFileNotFoundError); !isNotFound {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	// ---------- 3. Environment variables ----------
	v.SetEnvPrefix("ATLAS")
	// Convert camelCase keys to UPPER_SNAKE case automatically
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicitly bind environment variables that don't follow the automatic pattern
	_ = v.BindEnv("projectId", "ATLAS_PROJECT_ID")
	_ = v.BindEnv("orgId", "ATLAS_ORG_ID")
	_ = v.BindEnv("clusterName", "ATLAS_CLUSTER_NAME")
	_ = v.BindEnv("apiKey", "ATLAS_API_KEY")
	_ = v.BindEnv("publicKey", "ATLAS_PUB_KEY")

	// ---------- 4. Flags ----------
	if cmd != nil {
		// Bind both immediate flags and parent persistent flags.
		_ = v.BindPFlags(cmd.Flags())
		_ = v.BindPFlags(cmd.PersistentFlags())

		// Map dashed flag names to camelCase keys expected in struct tags.
		bind := func(key string, name string) {
			if f := cmd.Flags().Lookup(name); f != nil {
				_ = v.BindPFlag(key, f)
			}
		}
		bind("projectId", "project-id")
		bind("orgId", "org-id")
		bind("clusterName", "cluster-name")
		bind("apiKey", "api-key")
		bind("publicKey", "pub-key")
		// output and timeout flags use same spelling as struct tags already when no dashes.
	}

	// ---------- Unmarshal ----------
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
