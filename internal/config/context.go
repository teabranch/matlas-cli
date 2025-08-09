package config

import (
	"context"
)

// NewContext derives a context based on the Timeout setting in the provided Config.
// If Timeout is zero or negative, it returns context.Background() (never times out).
// Callers are responsible for invoking the returned cancel function to avoid leaks.
func NewContext(parent context.Context, cfg *Config) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if cfg == nil || cfg.Timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, cfg.Timeout)
}
