package cli

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teabranch/matlas-cli/internal/logging"
)

func TestNewSignalHandler(t *testing.T) {
	logger := logging.New(nil)

	tests := []struct {
		name            string
		timeoutSeconds  int
		expectedTimeout int
	}{
		{
			name:            "default timeout",
			timeoutSeconds:  0,
			expectedTimeout: 30,
		},
		{
			name:            "negative timeout",
			timeoutSeconds:  -5,
			expectedTimeout: 30,
		},
		{
			name:            "custom timeout",
			timeoutSeconds:  60,
			expectedTimeout: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewSignalHandler(logger, tt.timeoutSeconds)

			require.NotNil(t, handler)
			assert.Equal(t, tt.expectedTimeout, handler.timeoutSeconds)
			assert.NotNil(t, handler.ctx)
			assert.NotNil(t, handler.cancel)
			assert.NotNil(t, handler.logger)
			assert.NotNil(t, handler.cleanupFuncs)
			assert.Equal(t, 0, len(handler.cleanupFuncs))
			assert.False(t, handler.interrupted)
		})
	}
}

func TestSignalHandler_Context(t *testing.T) {
	logger := logging.New(nil)
	handler := NewSignalHandler(logger, 30)

	ctx := handler.Context()
	assert.NotNil(t, ctx)

	// Context should not be cancelled initially
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
		// Expected behavior
	}
}

func TestSignalHandler_RegisterCleanup(t *testing.T) {
	logger := logging.New(nil)
	handler := NewSignalHandler(logger, 30)

	var called int32
	cleanupFunc := func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	// Register cleanup function
	handler.RegisterCleanup(cleanupFunc)

	// Check it was registered
	handler.mu.RLock()
	assert.Equal(t, 1, len(handler.cleanupFuncs))
	handler.mu.RUnlock()

	// Register another one
	handler.RegisterCleanup(cleanupFunc)

	handler.mu.RLock()
	assert.Equal(t, 2, len(handler.cleanupFuncs))
	handler.mu.RUnlock()
}

func TestSignalHandler_RegisterCleanupWithConfig(t *testing.T) {
	logger := logging.New(nil)
	handler := NewSignalHandler(logger, 30)

	cleanupFunc := func(ctx context.Context) error {
		return nil
	}

	config := CleanupConfig{
		Name:        "test-cleanup",
		Timeout:     5 * time.Second,
		Critical:    true,
		Description: "Test cleanup function",
	}

	// Register with config (currently just delegates to simple registration)
	handler.RegisterCleanupWithConfig(cleanupFunc, config)

	handler.mu.RLock()
	assert.Equal(t, 1, len(handler.cleanupFuncs))
	handler.mu.RUnlock()
}

func TestCleanupConfig(t *testing.T) {
	config := CleanupConfig{
		Name:        "test-cleanup",
		Timeout:     5 * time.Second,
		Critical:    true,
		Description: "A test cleanup function",
	}

	assert.Equal(t, "test-cleanup", config.Name)
	assert.Equal(t, 5*time.Second, config.Timeout)
	assert.True(t, config.Critical)
	assert.Equal(t, "A test cleanup function", config.Description)
}

func TestRegisteredCleanup(t *testing.T) {
	cleanupFunc := func(ctx context.Context) error {
		return nil
	}

	config := CleanupConfig{
		Name:        "test",
		Timeout:     time.Second,
		Critical:    false,
		Description: "Test cleanup",
	}

	registered := RegisteredCleanup{
		Func:   cleanupFunc,
		Config: config,
	}

	assert.NotNil(t, registered.Func)
	assert.Equal(t, config, registered.Config)
}

func TestCleanupFunc(t *testing.T) {
	var called bool
	var receivedCtx context.Context

	cleanupFunc := CleanupFunc(func(ctx context.Context) error {
		called = true
		receivedCtx = ctx
		return nil
	})

	ctx := context.Background()
	err := cleanupFunc(ctx)

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, ctx, receivedCtx)
}

func TestCleanupFunc_WithError(t *testing.T) {
	expectedErr := errors.New("cleanup failed")

	cleanupFunc := CleanupFunc(func(ctx context.Context) error {
		return expectedErr
	})

	err := cleanupFunc(context.Background())
	assert.Equal(t, expectedErr, err)
}

// TestSignalHandler_IsInterrupted tests the interrupted flag functionality
func TestSignalHandler_IsInterrupted(t *testing.T) {
	logger := logging.New(nil)
	handler := NewSignalHandler(logger, 30)

	// Initially should not be interrupted
	assert.False(t, handler.interrupted)

	// The actual signal handling is complex to test in unit tests
	// since it involves OS signals. The interrupted flag would be
	// set by the signal handling goroutine in real usage.
}
