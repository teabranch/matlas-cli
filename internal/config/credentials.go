package config

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
)

// ErrAPIKeyNotFound is returned when ResolveAPIKey fails to find a key in any source.
var (
	ErrAPIKeyNotFound    = errors.New("atlas api key not found in flags/env/yaml or platform credential store")
	ErrPublicKeyNotFound = errors.New("atlas public key not found in flags/env/yaml or platform credential store")
)

// ResolveAPIKey returns a non-empty Atlas API key string following the secure resolution chain.
// Resolution order (first found wins):
//  1. Flag/YAML value stored in Config.APIKey (populated by Load())
//  2. Environment variable ATLAS_API_KEY (Atlas tooling standard)
//  3. Environment variable MATLAS_API_KEY (legacy matlas-cli compatibility)
//  4. Platform-specific credential storage:
//     - macOS: Keychain (security command)
//     - Windows: Credential Manager (PowerShell Get-StoredCredential)
//     - Linux: secret-service (secret-tool or GNOME Keyring)
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

	// Fallback: platform-specific credential storage
	if apiKey := getCredentialFromPlatformStore("api-key"); apiKey != "" {
		return apiKey, nil
	}

	return "", ErrAPIKeyNotFound
}

// ResolvePublicKey returns a non-empty Atlas public key string following the secure resolution chain.
// Resolution order (first found wins):
//  1. Flag/YAML value stored in Config.PublicKey (populated by Load())
//  2. Environment variable ATLAS_PUB_KEY (Atlas tooling standard)
//  3. Environment variable MATLAS_PUB_KEY (legacy matlas-cli compatibility)
//  4. Platform-specific credential storage:
//     - macOS: Keychain (security command)
//     - Windows: Credential Manager (PowerShell Get-StoredCredential)
//     - Linux: secret-service (secret-tool or GNOME Keyring)
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

	// Fallback: platform-specific credential storage
	if pubKey := getCredentialFromPlatformStore("pub-key"); pubKey != "" {
		return pubKey, nil
	}

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

// getCredentialFromPlatformStore retrieves credentials from platform-specific secure storage
func getCredentialFromPlatformStore(service string) string {
	switch runtime.GOOS {
	case "darwin":
		return getCredentialFromMacOSKeychain(service)
	case "windows":
		return getCredentialFromWindowsCredentialManager(service)
	case "linux":
		return getCredentialFromLinuxSecretService(service)
	default:
		return ""
	}
}

// getCredentialFromMacOSKeychain retrieves credentials from macOS Keychain
func getCredentialFromMacOSKeychain(service string) string {
	// security find-generic-password -a matlas -s <service> -w
	cmd := exec.Command("security", "find-generic-password", "-a", "matlas", "-s", service, "-w")
	out, err := cmd.Output()
	if err == nil {
		credential := strings.TrimSpace(string(out))
		if credential != "" {
			return credential
		}
	}
	return ""
}

// getCredentialFromWindowsCredentialManager retrieves credentials from Windows Credential Manager
func getCredentialFromWindowsCredentialManager(service string) string {
	// Use PowerShell to access Windows Credential Manager
	// Get-StoredCredential -Target "matlas:<service>" | Select-Object -ExpandProperty Password
	target := "matlas:" + service
	cmd := exec.Command("powershell", "-Command", 
		"try { $cred = Get-StoredCredential -Target '"+target+"' -ErrorAction Stop; "+
		"[Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($cred.Password)) "+
		"} catch { exit 1 }")
	
	out, err := cmd.Output()
	if err == nil {
		credential := strings.TrimSpace(string(out))
		if credential != "" {
			return credential
		}
	}
	
	// Fallback: try cmdkey (older method)
	// This doesn't retrieve passwords, but we can check if the credential exists
	// For security reasons, Windows doesn't easily expose stored passwords via cmdkey
	return ""
}

// getCredentialFromLinuxSecretService retrieves credentials from Linux secret-service (libsecret)
func getCredentialFromLinuxSecretService(service string) string {
	// Use secret-tool from libsecret to retrieve stored credentials
	// secret-tool lookup application matlas service <service>
	cmd := exec.Command("secret-tool", "lookup", "application", "matlas", "service", service)
	out, err := cmd.Output()
	if err == nil {
		credential := strings.TrimSpace(string(out))
		if credential != "" {
			return credential
		}
	}
	
	// Fallback: try GNOME Keyring directly (older systems)
	cmd = exec.Command("gnome-keyring", "get", "matlas-"+service)
	out, err = cmd.Output()
	if err == nil {
		credential := strings.TrimSpace(string(out))
		if credential != "" {
			return credential
		}
	}
	
	return ""
}
