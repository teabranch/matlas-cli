package config

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
)

// ErrAPIKeyNotFound is returned when ResolveAPIKey fails to find a key in any source.
var (
	ErrAPIKeyNotFound    = errors.New("atlas api key not found in flags/env/yaml/keychain")
	ErrPublicKeyNotFound = errors.New("atlas public key not found in flags/env/yaml/keychain")
)

// ResolveAPIKey returns a non-empty Atlas API key string following the secure resolution chain.
// Resolution order (first found wins):
//  1. Flag/YAML value stored in Config.APIKey (populated by Load())
//  2. Environment variable ATLAS_API_KEY (Atlas tooling standard)
//  3. Environment variable MATLAS_API_KEY (legacy matlas-cli compatibility)
//  4. Output of security find-generic-password command from keychain (macOS only)
//  5. If nothing found, returns ErrAPIKeyNotFound
func (c *Config) ResolveAPIKey() (string, error) {
	if c != nil && c.APIKey != "" {
		return c.APIKey, nil
	}

	if env := os.Getenv("ATLAS_API_KEY"); env != "" {
		return env, nil
	}

	if env := os.Getenv("MATLAS_API_KEY"); env != "" {
		return env, nil
	}

	// Fallback: keychain lookup (macOS)
	// security find-generic-password -a matlas -s api-key -w
	cmd := exec.Command("security", "find-generic-password", "-a", "matlas", "-s", "api-key", "-w")
	out, err := cmd.Output()
	if err == nil {
		apiKey := strings.TrimSpace(string(out))
		if apiKey != "" {
			return apiKey, nil
		}
	}

	// TODO: add support for other platforms (e.g., Windows Credential Manager, Linux secret-service)

	return "", ErrAPIKeyNotFound
}

// ResolvePublicKey returns a non-empty Atlas public key string following the secure resolution chain.
// Resolution order (first found wins):
//  1. Flag/YAML value stored in Config.PublicKey (populated by Load())
//  2. Environment variable ATLAS_PUB_KEY (Atlas tooling standard)
//  3. Environment variable MATLAS_PUB_KEY (legacy matlas-cli compatibility)
//  4. Output of security find-generic-password command from keychain (macOS only)
//  5. If nothing found, returns ErrPublicKeyNotFound
func (c *Config) ResolvePublicKey() (string, error) {
	if c != nil && c.PublicKey != "" {
		return c.PublicKey, nil
	}

	if env := os.Getenv("ATLAS_PUB_KEY"); env != "" {
		return env, nil
	}

	if env := os.Getenv("MATLAS_PUB_KEY"); env != "" {
		return env, nil
	}

	// Fallback: keychain lookup (macOS)
	// security find-generic-password -a matlas -s pub-key -w
	cmd := exec.Command("security", "find-generic-password", "-a", "matlas", "-s", "pub-key", "-w")
	out, err := cmd.Output()
	if err == nil {
		pubKey := strings.TrimSpace(string(out))
		if pubKey != "" {
			return pubKey, nil
		}
	}

	// TODO: add support for other platforms (e.g., Windows Credential Manager, Linux secret-service)

	return "", ErrPublicKeyNotFound
}

// CreateAtlasClient creates an Atlas client using resolved API key and public key
func (c *Config) CreateAtlasClient() (*atlasclient.Client, error) {
	apiKey, err := c.ResolveAPIKey()
	if err != nil {
		return nil, err
	}

	publicKey, err := c.ResolvePublicKey()
	if err != nil {
		return nil, err
	}

	return atlasclient.NewClient(atlasclient.Config{
		PrivateKey: apiKey,
		PublicKey:  publicKey,
	})
}
