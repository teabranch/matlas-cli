// Package fileutil provides secure file operation utilities
package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// SecureFileWriter handles secure file operations with appropriate permissions
type SecureFileWriter struct {
	fileMode os.FileMode
	dirMode  os.FileMode
}

// NewSecureFileWriter creates a writer for sensitive files with restrictive permissions
func NewSecureFileWriter() *SecureFileWriter {
	return &SecureFileWriter{
		fileMode: 0600, // rw------- (owner read/write only)
		dirMode:  0700, // rwx------ (owner full access only)
	}
}

// WriteFile securely writes data to a file with proper permissions
func (w *SecureFileWriter) WriteFile(path string, data []byte) error {
	// Create parent directory with secure permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, w.dirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with secure permissions
	if err := os.WriteFile(path, data, w.fileMode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Verify permissions (defense in depth)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to verify file permissions: %w", err)
	}

	if info.Mode().Perm() != w.fileMode {
		// Attempt to fix permissions
		if err := os.Chmod(path, w.fileMode); err != nil {
			return fmt.Errorf("failed to set secure permissions: %w", err)
		}
	}

	return nil
}

// WriteFileWithMode writes a file with custom permissions
func (w *SecureFileWriter) WriteFileWithMode(path string, data []byte, mode os.FileMode) error {
	// Create parent directory with secure permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, w.dirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with specified permissions
	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// EnsureSecurePermissions ensures existing file has secure permissions
func (w *SecureFileWriter) EnsureSecurePermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Mode().Perm() != w.fileMode {
		if err := os.Chmod(path, w.fileMode); err != nil {
			return fmt.Errorf("failed to set secure permissions: %w", err)
		}
	}

	return nil
}
