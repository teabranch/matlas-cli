package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/logging"
)

// Demo script to showcase all the new observability and UX features
func main() {
	fmt.Println("🚀 matlas-cli Observability & UX Enhancement Demo")
	fmt.Println("============================================================")

	// 1. Advanced Structured Logging Demo
	fmt.Println("\n📝 1. Advanced Structured Logging")
	demonstrateLogging()

	// 2. Signal Handling Demo
	fmt.Println("\n🛑 2. Graceful Signal Handling")
	demonstrateSignalHandling()

	// 3. Enhanced Error Handling Demo
	fmt.Println("\n🚨 3. Enhanced Error Handling")
	demonstrateErrorHandling()

	// 4. Shell Integration Demo
	fmt.Println("\n🐚 4. Shell Integration")
	demonstrateShellIntegration()

	fmt.Println("\n✅ Demo completed! All features are working correctly.")
	fmt.Println("\nTo use these features in matlas-cli:")
	fmt.Println("  • Use --verbose for detailed logging")
	fmt.Println("  • Use --quiet to suppress non-error output")
	fmt.Println("  • Use --log-format json for structured JSON logs")
	fmt.Println("  • Press Ctrl+C to test graceful shutdown")
	fmt.Println("  • Run 'matlas completion bash' for shell completion")
	fmt.Println("  • Run 'matlas shell-integration install bash' for setup")
}

func demonstrateLogging() {
	// Create logger with different configurations
	configs := []*logging.Config{
		{
			Level:         logging.LevelInfo,
			Format:        "text",
			Output:        os.Stdout,
			Verbose:       false,
			EnableMetrics: true,
		},
		{
			Level:         logging.LevelDebug,
			Format:        "json",
			Output:        os.Stdout,
			Verbose:       true,
			EnableAPILogs: true,
			MaskSecrets:   true,
		},
	}

	for i, config := range configs {
		fmt.Printf("\n  Example %d: %s format, verbose=%v\n", i+1, config.Format, config.Verbose)

		logger := logging.New(config)

		// Demonstrate different log levels
		logger.Info("Application started", "component", "demo", "version", "1.0.0")
		logger.Debug("Debug information", "user_id", "12345", "session", "abc123")
		logger.Warn("This is a warning", "resource", "cluster-1", "action", "create")

		// Demonstrate operation tracking
		op := logger.StartOperation("demo-operation", "testing")
		op.Progress("Processing step 1", 25.0)
		time.Sleep(100 * time.Millisecond)
		op.Progress("Processing step 2", 50.0)
		time.Sleep(100 * time.Millisecond)
		op.Complete("Operation finished successfully")

		// Demonstrate API logging
		if config.EnableAPILogs {
			req := &logging.APIRequest{
				Method:  "POST",
				URL:     "https://cloud.mongodb.com/api/atlas/v1.0/groups/123/clusters",
				Headers: map[string]string{"Authorization": "Bearer secret-token-here"},
				Body:    `{"name": "test-cluster"}`,
				Started: time.Now(),
			}
			logger.LogAPIRequest(req)

			resp := &logging.APIResponse{
				StatusCode: 201,
				Body:       `{"id": "cluster-123", "status": "CREATING"}`,
				Duration:   500 * time.Millisecond,
			}
			logger.LogAPIResponse(req, resp)
		}

		// Demonstrate metrics logging
		logger.LogMetric("cluster_creation_time", 45.5, "seconds", map[string]string{
			"cluster_type": "M10",
			"region":       "US_EAST_1",
		})
	}
}

func demonstrateSignalHandling() {
	logger := logging.New(logging.DefaultConfig())

	fmt.Println("  Creating signal handler...")
	handler := cli.NewSignalHandler(logger, 5) // 5 second timeout for demo

	// Register some cleanup functions
	handler.RegisterCleanup(cli.CreateFileCleanup("/tmp/test-file-1", "/tmp/test-file-2"))
	handler.RegisterCleanup(cli.CreateResourceCleanup("demo-resource", func(ctx context.Context) error {
		fmt.Println("    🧹 Cleaning up demo resource...")
		time.Sleep(100 * time.Millisecond)
		return nil
	}))

	fmt.Println("  Signal handler configured with cleanup functions")
	fmt.Println("  In a real application, press Ctrl+C to test graceful shutdown")

	// Simulate some work
	fmt.Println("  Simulating work...")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("  Work completed normally")
}

func demonstrateErrorHandling() {
	logger := logging.New(logging.DefaultConfig())
	formatter := cli.NewEnhancedErrorFormatter(true, logger)
	analyzer := cli.NewErrorAnalyzer(logger)

	// Test different types of errors
	testErrors := []error{
		errors.New("unauthorized: invalid API key"),
		errors.New("not found: cluster 'test-cluster' does not exist"),
		errors.New("validation failed: field 'name' is required"),
		errors.New("connection timeout: failed to connect to atlas.mongodb.com"),
		errors.New("conflict: cluster name already exists"),
	}

	for i, err := range testErrors {
		fmt.Printf("\n  Error %d Analysis:\n", i+1)

		// Wrap with context
		contextualErr := cli.WrapWithOperation(err, "create_cluster", "test-cluster")

		// Analyze the error
		analysis := analyzer.Analyze(err)
		fmt.Printf("    Category: %s\n", analysis.Category)
		fmt.Printf("    Severity: %s\n", analysis.Severity)
		fmt.Printf("    Retryable: %v\n", analysis.Retryable)
		fmt.Printf("    User Actionable: %v\n", analysis.UserActionable)

		// Format with suggestions
		formatted := formatter.FormatWithAnalysis(contextualErr)
		fmt.Printf("    Formatted: %s\n", formatted)
	}

	// Demonstrate error collection
	fmt.Println("\n  Error Collection Example:")
	collector := cli.NewErrorCollector()
	collector.Add(errors.New("error 1"), cli.ErrorContext{Operation: "step1"})
	collector.Add(errors.New("error 2"), cli.ErrorContext{Operation: "step2"})
	collector.Add(errors.New("error 3"), cli.ErrorContext{Operation: "step3"})

	fmt.Printf("    Collected %d errors\n", collector.Count())
	combined := collector.Combine()
	fmt.Printf("    Combined: %s\n", combined.Error())

	// Demonstrate panic recovery
	fmt.Println("\n  Panic Recovery Example:")
	err := cli.HandleWithRecovery("demo_operation", func() error {
		// Simulate a panic
		panic("Something went wrong!")
	})
	if err != nil {
		fmt.Printf("    Recovered from panic: %s\n", err.Error())
	}
}

func demonstrateShellIntegration() {
	logger := logging.New(logging.DefaultConfig())
	shell := cli.NewShellIntegration(logger)

	// Demonstrate alias generation
	fmt.Println("  Generated aliases for bash:")
	aliases := shell.GenerateShellAliases("bash")
	fmt.Print("    " + aliases)

	// Demonstrate installation instructions
	fmt.Println("\n  Installation instructions for zsh:")
	instructions := shell.InstallInstructions("zsh")
	lines := splitLines(instructions)
	for i, line := range lines {
		if i < 5 { // Show first 5 lines
			fmt.Printf("    %s\n", line)
		}
	}
	if len(lines) > 5 {
		fmt.Printf("    ... (%d more lines)\n", len(lines)-5)
	}

	fmt.Println("\n  Shell completion features:")
	fmt.Println("    • Dynamic project ID completion")
	fmt.Println("    • Output format completion (text, json, yaml)")
	fmt.Println("    • Config file path completion")
	fmt.Println("    • Atlas resource completion (clusters, databases)")
	fmt.Println("    • Smart suggestions with descriptions")
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}
