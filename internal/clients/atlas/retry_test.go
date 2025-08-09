package atlas

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetry_SucceedsAfterTransientErrors(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	err := retry(ctx, 3, 10*time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("%w: simulated", ErrTransient)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_StopsOnPermanentError(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	errTest := errors.New("boom")
	err := retry(ctx, 5, 10*time.Millisecond, func() error {
		attempts++
		return errTest
	})
	if !errors.Is(err, errTest) {
		t.Fatalf("expected errTest, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
