// Package atlas provides a thin wrapper over the Atlas Go SDK with retries and helpers.
package atlas

import (
	"context"
	"os"
	"time"

	"github.com/teabranch/matlas-cli/internal/logging"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// Config defines optional settings for initializing the Atlas client wrapper.
// Zero values are replaced by sensible defaults or environment variables.
type Config struct {
	PublicKey  string          // Atlas public API key; fallback: ATLAS_PUB_KEY env var
	PrivateKey string          // Atlas private API key; fallback: ATLAS_API_KEY env var
	BaseURL    string          // Override Atlas API base URL (for testing)
	RetryMax   int             // Maximum retry attempts for transient failures (default 3)
	RetryDelay time.Duration   // Initial back-off between retries (default 250ms doubled each attempt)
	Logger     *logging.Logger // Optional structured logger (default logging.Default())
}

// Client wraps the Atlas SDK admin API client, adding logging and (soon) retry middleware.
type Client struct {
	Atlas        *admin.APIClient
	logger       *logging.Logger
	retryMax     int
	retryBackoff time.Duration
}

// NewClient constructs a new Client using the supplied configuration. Credentials are resolved
// from the Config first and fall back to environment variables. The resulting Client is safe
// for concurrent use across goroutines.
func NewClient(cfg Config) (*Client, error) {
	// Resolve credentials.
	if cfg.PublicKey == "" {
		cfg.PublicKey = os.Getenv("ATLAS_PUB_KEY")
	}
	if cfg.PrivateKey == "" {
		cfg.PrivateKey = os.Getenv("ATLAS_API_KEY")
	}

	// Provide sensible defaults.
	if cfg.RetryMax == 0 {
		cfg.RetryMax = 3
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 250 * time.Millisecond
	}
	if cfg.Logger == nil {
		cfg.Logger = logging.Default()
	}

	modifiers := []admin.ClientModifier{
		admin.UseDigestAuth(cfg.PublicKey, cfg.PrivateKey),
	}
	// If BaseURL override provided (e.g., testing), we supply a modifier.
	if cfg.BaseURL != "" {
		// The SDK exposes UseBaseURL in internal/core; simplest is to set after creation.
		modifiers = append(modifiers, admin.UseBaseURL(cfg.BaseURL))
	}

	atlasClient, err := admin.NewClient(modifiers...)
	if err != nil {
		return nil, err
	}

	return &Client{
		Atlas:        atlasClient,
		logger:       cfg.Logger.WithFields(map[string]any{"component": "atlas.Client"}),
		retryMax:     cfg.RetryMax,
		retryBackoff: cfg.RetryDelay,
	}, nil
}

// New is a convenience function that creates a client with just an API key.
// It uses the API key as the private key and looks up the public key from ATLAS_PUB_KEY env var.
func New(apiKey string) (*Client, error) {
	return NewClient(Config{
		PrivateKey: apiKey,
	})
}

// WithContext returns the underlying SDK client while allowing the caller to pass a context.
// Prefer this helper when you need to specify timeouts or cancellation.
func (c *Client) WithContext(ctx context.Context) *admin.APIClient {
	// Currently the SDK methods accept context.Context on API calls.
	_ = ctx // reserved for future middleware (e.g., per-call logging)
	return c.Atlas
}

// Do executes fn with automatic retry for transient errors and respects the provided context.
// The callback receives the underlying Atlas API client to perform SDK operations.
func (c *Client) Do(ctx context.Context, fn func(api *admin.APIClient) error) error {
	return retry(ctx, c.retryMax, c.retryBackoff, func() error {
		return convertError(fn(c.Atlas))
	})
}
