// Package config defines the runtime configuration model and helpers.
package config

import (
	"fmt"
	"time"
)

// OutputFormat represents the supported output serialization formats.
// Text: human-friendly table/list; JSON: machine-readable; YAML: verbose configuration-like.
// Extend with more as needed (e.g., CSV).

type OutputFormat string

const (
	OutputTable OutputFormat = "table"
	OutputText  OutputFormat = "text"
	OutputJSON  OutputFormat = "json"
	OutputYAML  OutputFormat = "yaml"
)

// DefaultTimeout is the fallback duration applied when the user does not
// specify `--timeout`, `ATLAS_TIMEOUT`, or `timeout` YAML key.
const DefaultTimeout = 30 * time.Second

// DefaultConfigDir is the default directory under the user's home for matlas config files.
const DefaultConfigDir = ".matlas"

// Config is the fully-resolved, immutable runtime configuration for a single command invocation.
//
// All fields should have zero-value semantics that mean "not set" so the precedence resolver
// can determine whether a value originated from a lower tier (e.g., YAML) or was supplied by
// a higher priority source (flag/env).
//
// Use `mapstructure` tags so Viper can unmarshal seamlessly regardless of source.
// CamelCase YAML keys are the preferred canonical spelling in config files.
// Env variables use the ATLAS_ prefix and UPPER_SNAKE_CASE conversion handled externally.

type Config struct {
	// Atlas / Cloud settings
	ProjectID   string `mapstructure:"projectId" yaml:"projectId"`
	OrgID       string `mapstructure:"orgId" yaml:"orgId"`
	ClusterName string `mapstructure:"clusterName" yaml:"clusterName"`

	// Generic CLI behaviour
	Output  OutputFormat  `mapstructure:"output" yaml:"output"`
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout"`

	// Credentials (avoid printing/logging!)
	APIKey    string `mapstructure:"apiKey" yaml:"apiKey"`
	PublicKey string `mapstructure:"publicKey" yaml:"publicKey"`
}

// New returns a Config populated with builtin defaults.
// Callers should subsequently merge flag/env/YAML values on top.
func New() *Config {
	return &Config{
		Output:  OutputTable,
		Timeout: DefaultTimeout,
	}
}

// ResolveProjectID resolves the project ID from flag value or configuration
// If flagValue is provided, it takes precedence. Otherwise, uses config.ProjectID
func (c *Config) ResolveProjectID(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return c.ProjectID
}

// ResolveOrgID resolves the organization ID from flag value or configuration
// If flagValue is provided, it takes precedence. Otherwise, uses config.OrgID
func (c *Config) ResolveOrgID(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return c.OrgID
}

// Validate performs sanity checks after the full precedence merge.
// Only inexpensive validation belongs here; cross-service validation should live closer to service layers.
func (c *Config) Validate() error {
	switch c.Output {
	case OutputTable, OutputText, OutputJSON, OutputYAML, "", "summary", "detailed", "unified":
		// ok (empty means caller forgot to merge; treat as default)
	default:
		return fmt.Errorf("unsupported output format: %s", c.Output)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}
