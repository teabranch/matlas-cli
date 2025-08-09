package atlas

import (
	"context"
	"time"
)

// retry executes fn up to maxAttempts. It waits backoff duration (multiplied by 2 each attempt)
// between retries for errors that satisfy IsTransient. If the context is cancelled, it bails early.
func retry(ctx context.Context, maxAttempts int, backoff time.Duration, fn func() error) error {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if backoff <= 0 {
		backoff = 250 * time.Millisecond
	}

	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = fn()
		if err == nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		}
		// Break if error is not transient or we have no more attempts.
		if !IsTransient(err) || attempt == maxAttempts {
			return err
		}
		select {
		case <-time.After(backoff):
			backoff *= 2
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
