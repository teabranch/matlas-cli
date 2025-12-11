// Package security provides security utilities for masking sensitive information
package security

import (
	"net/url"
	"strings"
)

// MaskConnectionString safely masks credentials in MongoDB connection strings
// while preserving the username and structure for debugging purposes.
//
// Examples:
//   - mongodb://user:pass@host/db -> mongodb://user:***@host/db
//   - mongodb+srv://admin:P@ssw0rd!@cluster.mongodb.net/test -> mongodb+srv://admin:***@cluster.mongodb.net/test
//   - mongodb://host/db (no credentials) -> mongodb://host/db (unchanged)
func MaskConnectionString(uri string) string {
	if uri == "" {
		return ""
	}

	// Fast path: if no '@', there are no credentials
	if !strings.Contains(uri, "@") {
		return uri
	}

	// Parse as URL to safely extract components
	parsedURL, err := url.Parse(uri)
	if err != nil {
		// Fallback to simple masking if parsing fails
		return "***MASKED_URI***"
	}

	// If there's user info, mask the password
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		if username != "" {
			// Keep username visible but mask password completely
			parsedURL.User = url.User(username)
		} else {
			// No username, clear user info entirely
			parsedURL.User = nil
		}
	}

	return parsedURL.String()
}

// MaskCredentialInString masks any credential-like strings in text
// by showing only the first and last 4 characters.
//
// This is useful for API keys, tokens, and other credential types.
//
// Examples:
//   - "abcdefghijklmnopqrst" -> "abcd************qrst"
//   - "short" -> "***"
func MaskCredentialInString(text string) string {
	if text == "" {
		return ""
	}

	// For very short strings, mask completely
	if len(text) <= 8 {
		return "***"
	}

	// Show first 4 and last 4 characters
	return text[:4] + strings.Repeat("*", len(text)-8) + text[len(text)-4:]
}

// MaskValue provides intelligent masking based on value type
func MaskValue(value interface{}) interface{} {
	str, ok := value.(string)
	if !ok {
		return "***"
	}

	// Check if it looks like a connection string
	if strings.HasPrefix(str, "mongodb://") || strings.HasPrefix(str, "mongodb+srv://") {
		return MaskConnectionString(str)
	}

	// Default to credential masking
	return MaskCredentialInString(str)
}
