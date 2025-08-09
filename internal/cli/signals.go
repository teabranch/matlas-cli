package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/teabranch/matlas-cli/internal/logging"
)

// SignalHandler manages graceful shutdown for CLI commands
type SignalHandler struct {
	ctx            context.Context
	cancel         context.CancelFunc
	logger         *logging.Logger
	cleanupFuncs   []CleanupFunc
	timeoutSeconds int
	mu             sync.RWMutex
	shutdownOnce   sync.Once
	interrupted    bool
}

// CleanupFunc represents a function to call during shutdown
type CleanupFunc func(ctx context.Context) error

// CleanupConfig holds configuration for cleanup operations
type CleanupConfig struct {
	Name        string
	Timeout     time.Duration
	Critical    bool // If true, failure to cleanup will cause immediate exit
	Description string
}

// RegisteredCleanup wraps a cleanup function with its configuration
type RegisteredCleanup struct {
	Func   CleanupFunc
	Config CleanupConfig
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler(logger *logging.Logger, timeoutSeconds int) *SignalHandler {
	ctx, cancel := context.WithCancel(context.Background())

	if timeoutSeconds <= 0 {
		timeoutSeconds = 30 // Default 30 second timeout
	}

	return &SignalHandler{
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
		cleanupFuncs:   make([]CleanupFunc, 0),
		timeoutSeconds: timeoutSeconds,
	}
}

// Context returns the signal handler's context
func (sh *SignalHandler) Context() context.Context {
	return sh.ctx
}

// RegisterCleanup registers a cleanup function to be called on shutdown
func (sh *SignalHandler) RegisterCleanup(fn CleanupFunc) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.cleanupFuncs = append(sh.cleanupFuncs, fn)
}

// RegisterCleanupWithConfig registers a cleanup function with specific configuration
func (sh *SignalHandler) RegisterCleanupWithConfig(fn CleanupFunc, config CleanupConfig) {
	// For now, we'll just use the simple registration
	// In the future, we can extend this to handle the configuration
	sh.RegisterCleanup(fn)
}

// Start begins signal monitoring
func (sh *SignalHandler) Start() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		sh.handleSignal(sig)
	}()
}

// handleSignal processes received signals
func (sh *SignalHandler) handleSignal(sig os.Signal) {
	sh.mu.Lock()
	if sh.interrupted {
		sh.mu.Unlock()
		// Already handling shutdown, force exit
		sh.logger.Critical("Force shutdown - received second signal", "signal", sig.String())
		os.Exit(128 + int(sig.(syscall.Signal)))
	}
	sh.interrupted = true
	sh.mu.Unlock()

	sh.shutdownOnce.Do(func() {
		switch sig {
		case syscall.SIGINT:
			sh.logger.Warn("Interrupt received - initiating graceful shutdown...", "signal", "SIGINT")
			fmt.Fprintf(os.Stderr, "\n⚠️  Interrupt received - cleaning up... (press Ctrl+C again to force quit)\n")
		case syscall.SIGTERM:
			sh.logger.Warn("Termination received - initiating graceful shutdown...", "signal", "SIGTERM")
			fmt.Fprintf(os.Stderr, "\n⚠️  Termination received - cleaning up...\n")
		default:
			sh.logger.Warn("Signal received - initiating graceful shutdown...", "signal", sig.String())
			fmt.Fprintf(os.Stderr, "\n⚠️  Signal received - cleaning up...\n")
		}

		// Cancel the main context
		sh.cancel()

		// Perform cleanup
		sh.performCleanup()

		// Exit with appropriate code
		exitCode := 128 + int(sig.(syscall.Signal))
		sh.logger.Info("Graceful shutdown completed", "exit_code", exitCode)
		os.Exit(exitCode)
	})
}

// performCleanup executes all registered cleanup functions
func (sh *SignalHandler) performCleanup() {
	if len(sh.cleanupFuncs) == 0 {
		sh.logger.Debug("No cleanup functions registered")
		return
	}

	sh.logger.Info("Starting cleanup process", "cleanup_functions", len(sh.cleanupFuncs))
	fmt.Fprintf(os.Stderr, "🧹 Running cleanup... (%d operations)\n", len(sh.cleanupFuncs))

	// Create cleanup context with timeout
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Duration(sh.timeoutSeconds)*time.Second)
	defer cleanupCancel()

	var wg sync.WaitGroup
	errorChan := make(chan error, len(sh.cleanupFuncs))

	// Execute cleanup functions
	sh.mu.RLock()
	for i, fn := range sh.cleanupFuncs {
		wg.Add(1)
		go func(index int, cleanupFn CleanupFunc) {
			defer wg.Done()

			sh.logger.Debug("Executing cleanup function", "index", index)
			if err := cleanupFn(cleanupCtx); err != nil {
				sh.logger.Error("Cleanup function failed",
					"index", index,
					"error", err.Error())
				errorChan <- fmt.Errorf("cleanup[%d]: %w", index, err)
			} else {
				sh.logger.Debug("Cleanup function completed", "index", index)
			}
		}(i, fn)
	}
	sh.mu.RUnlock()

	// Wait for all cleanup functions to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		sh.logger.Info("All cleanup functions completed successfully")
		fmt.Fprintf(os.Stderr, "✅ Cleanup completed successfully\n")
	case <-cleanupCtx.Done():
		sh.logger.Warn("Cleanup timeout reached", "timeout_seconds", sh.timeoutSeconds)
		fmt.Fprintf(os.Stderr, "⏰ Cleanup timeout after %d seconds\n", sh.timeoutSeconds)
	}

	// Check for errors
	close(errorChan)
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		sh.logger.Error("Some cleanup functions failed", "error_count", len(errors))
		fmt.Fprintf(os.Stderr, "⚠️  %d cleanup errors occurred\n", len(errors))
		for _, err := range errors {
			sh.logger.Error("Cleanup error", "error", err.Error())
		}
	}
}

// Wait blocks until a signal is received or context is cancelled
func (sh *SignalHandler) Wait() {
	<-sh.ctx.Done()
}

// IsInterrupted returns true if a signal has been received
func (sh *SignalHandler) IsInterrupted() bool {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.interrupted
}

// Shutdown triggers graceful shutdown manually
func (sh *SignalHandler) Shutdown() {
	sh.cancel()
}

// WithSignalHandler creates a signal handler and runs the provided function
func WithSignalHandler(logger *logging.Logger, timeoutSeconds int, fn func(*SignalHandler) error) error {
	handler := NewSignalHandler(logger, timeoutSeconds)
	handler.Start()

	// Run the main function
	err := fn(handler)

	// If we get here without interruption, perform normal cleanup
	if !handler.IsInterrupted() {
		handler.performCleanup()
	}

	return err
}

// Common cleanup functions

// CreateFileCleanup creates a cleanup function that removes files
func CreateFileCleanup(filePaths ...string) CleanupFunc {
	return func(ctx context.Context) error {
		for _, path := range filePaths {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove file %s: %w", path, err)
			}
		}
		return nil
	}
}

// CreateDirectoryCleanup creates a cleanup function that removes directories
func CreateDirectoryCleanup(dirPaths ...string) CleanupFunc {
	return func(ctx context.Context) error {
		for _, path := range dirPaths {
			if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove directory %s: %w", path, err)
			}
		}
		return nil
	}
}

// CreateResourceCleanup creates a cleanup function for custom resources
func CreateResourceCleanup(name string, cleanupFn func(context.Context) error) CleanupFunc {
	return func(ctx context.Context) error {
		if err := cleanupFn(ctx); err != nil {
			return fmt.Errorf("failed to cleanup %s: %w", name, err)
		}
		return nil
	}
}

// ChainCleanup chains multiple cleanup functions into one
func ChainCleanup(cleanups ...CleanupFunc) CleanupFunc {
	return func(ctx context.Context) error {
		var errors []error
		for i, cleanup := range cleanups {
			if err := cleanup(ctx); err != nil {
				errors = append(errors, fmt.Errorf("cleanup[%d]: %w", i, err))
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf("multiple cleanup errors: %v", errors)
		}
		return nil
	}
}
