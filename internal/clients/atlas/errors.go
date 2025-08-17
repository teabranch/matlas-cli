package atlas

import (
	"errors"
	"fmt"

	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// Typed errors used by higher layers to reason about Atlas failures.
var (
	ErrNotFound     = errors.New("atlas: not found")
	ErrConflict     = errors.New("atlas: conflict")
	ErrUnauthorized = errors.New("atlas: unauthorized")
	ErrTransient    = errors.New("atlas: transient")
)

// convertError inspects an error returned by the Atlas SDK and wraps it in a typed error where possible.
// Callers should prefer using the helper predicates (IsNotFound, etc.) rather than comparing errors directly.
func convertError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case admin.IsErrorCode(err, "NOT_FOUND") || admin.IsErrorCode(err, "CLUSTER_NOT_FOUND") || admin.IsErrorCode(err, "DATABASE_USER_NOT_FOUND") || admin.IsErrorCode(err, "NETWORK_ACCESS_NOT_FOUND") || admin.IsErrorCode(err, "PROJECT_NOT_FOUND"):
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	case admin.IsErrorCode(err, "CONFLICT"):
		return fmt.Errorf("%w: %v", ErrConflict, err)
	case admin.IsErrorCode(err, "UNAUTHORIZED") || admin.IsErrorCode(err, "FORBIDDEN"):
		return fmt.Errorf("%w: %v", ErrUnauthorized, err)
	case admin.IsErrorCode(err, "TOO_MANY_REQUESTS") || admin.IsErrorCode(err, "INTERNAL") || admin.IsErrorCode(err, "UNEXPECTED_ERROR"):
		// These are usually safe to retry.
		return fmt.Errorf("%w: %v", ErrTransient, err)
	default:
		return err
	}
}

// IsNotFound reports whether err represents a not-found condition.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsConflict reports whether err represents a conflict condition.
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }

// IsUnauthorized reports whether err represents an authentication/authorization error.
func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }

// IsTransient reports whether err represents a transient/retryable failure.
func IsTransient(err error) bool { return errors.Is(err, ErrTransient) }
