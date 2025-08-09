package atlas

import (
	"errors"
	"fmt"
	"testing"
)

func TestConvertError(t *testing.T) {
	tests := []struct {
		name        string
		inputErr    error
		expectedErr error
		description string
	}{
		{
			name:        "nil error",
			inputErr:    nil,
			expectedErr: nil,
			description: "nil input should return nil",
		},
		{
			name:        "non-atlas error",
			inputErr:    errors.New("generic error"),
			expectedErr: nil,
			description: "non-Atlas errors should pass through unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertError(tt.inputErr)

			if tt.expectedErr == nil && result != tt.inputErr {
				t.Errorf("expected passthrough of original error, got %v", result)
			} else if tt.expectedErr != nil {
				if !errors.Is(result, tt.expectedErr) {
					t.Errorf("expected error to wrap %v, but errors.Is failed. Got: %v", tt.expectedErr, result)
				}
				if !errors.Is(result, tt.inputErr) {
					t.Errorf("expected wrapped error to contain original error, but errors.Is failed")
				}
			}
		})
	}
}

func TestErrorPredicates(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		isNotFound     bool
		isConflict     bool
		isUnauthorized bool
		isTransient    bool
	}{
		{
			name:       "nil error",
			err:        nil,
			isNotFound: false, isConflict: false, isUnauthorized: false, isTransient: false,
		},
		{
			name:       "wrapped not found",
			err:        fmt.Errorf("%w: resource missing", ErrNotFound),
			isNotFound: true, isConflict: false, isUnauthorized: false, isTransient: false,
		},
		{
			name:       "wrapped conflict",
			err:        fmt.Errorf("%w: duplicate resource", ErrConflict),
			isNotFound: false, isConflict: true, isUnauthorized: false, isTransient: false,
		},
		{
			name:       "wrapped unauthorized",
			err:        fmt.Errorf("%w: access denied", ErrUnauthorized),
			isNotFound: false, isConflict: false, isUnauthorized: true, isTransient: false,
		},
		{
			name:       "wrapped transient",
			err:        fmt.Errorf("%w: temporary failure", ErrTransient),
			isNotFound: false, isConflict: false, isUnauthorized: false, isTransient: true,
		},
		{
			name:       "direct error types",
			err:        ErrNotFound,
			isNotFound: true, isConflict: false, isUnauthorized: false, isTransient: false,
		},
		{
			name:       "unrelated error",
			err:        errors.New("something else"),
			isNotFound: false, isConflict: false, isUnauthorized: false, isTransient: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsNotFound(tt.err) != tt.isNotFound {
				t.Errorf("IsNotFound() = %v, want %v", IsNotFound(tt.err), tt.isNotFound)
			}
			if IsConflict(tt.err) != tt.isConflict {
				t.Errorf("IsConflict() = %v, want %v", IsConflict(tt.err), tt.isConflict)
			}
			if IsUnauthorized(tt.err) != tt.isUnauthorized {
				t.Errorf("IsUnauthorized() = %v, want %v", IsUnauthorized(tt.err), tt.isUnauthorized)
			}
			if IsTransient(tt.err) != tt.isTransient {
				t.Errorf("IsTransient() = %v, want %v", IsTransient(tt.err), tt.isTransient)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Test error chaining with known error types
	wrappedErr := fmt.Errorf("%w: context info", ErrNotFound)

	// Test that wrapped errors are detectable
	if !IsNotFound(wrappedErr) {
		t.Error("wrapped ErrNotFound should be detectable")
	}

	// Test double wrapping
	doubleWrapped := fmt.Errorf("operation failed: %w", wrappedErr)
	if !IsNotFound(doubleWrapped) {
		t.Error("double-wrapped error should still be detectable as not found")
	}
}
