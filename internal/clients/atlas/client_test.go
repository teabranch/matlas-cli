package atlas

import (
	"context"
	"testing"
	"time"

	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

func TestClient_Do_RetriesTransient(t *testing.T) {
	c, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient err: %v", err)
	}
	attempts := 0
	err = c.Do(context.Background(), func(api *admin.APIClient) error {
		attempts++
		if attempts < 2 {
			return ErrTransient
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestClient_Do_ContextCancelled(t *testing.T) {
	c, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient err: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err = c.Do(ctx, func(api *admin.APIClient) error {
		time.Sleep(30 * time.Millisecond)
		return nil
	})
	if err == nil {
		t.Fatalf("expected context deadline exceeded, got nil")
	}
}
