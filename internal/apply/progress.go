package apply

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/teabranch/matlas-cli/internal/ui"
)

// ProgressTracker manages progress tracking for multi-operation execution
type ProgressTracker struct {
	updateInterval time.Duration
	progress       *ui.ProgressIndicator
	currentBar     *ui.ProgressBar

	// State tracking
	mu                sync.RWMutex
	isActive          bool
	stopChan          chan struct{}
	executionProgress *ExecutorProgress

	// Configuration
	output      io.Writer
	verboseMode bool
	quietMode   bool
}

// ProgressEvent represents different types of progress events
type ProgressEvent struct {
	Type      ProgressEventType `json:"type"`
	Message   string            `json:"message"`
	Operation string            `json:"operation,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Data      interface{}       `json:"data,omitempty"`
}

// ProgressEventType represents the type of progress event
type ProgressEventType string

const (
	ProgressEventStart     ProgressEventType = "start"
	ProgressEventProgress  ProgressEventType = "progress"
	ProgressEventOperation ProgressEventType = "operation"
	ProgressEventComplete  ProgressEventType = "complete"
	ProgressEventError     ProgressEventType = "error"
)

// OperationProgress tracks progress for individual operations
type OperationProgress struct {
	OperationID string                 `json:"operationId"`
	Name        string                 `json:"name"`
	Status      OperationStatus        `json:"status"`
	Progress    float64                `json:"progress"` // 0.0 to 1.0
	StartedAt   time.Time              `json:"startedAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Message     string                 `json:"message,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MultiOperationProgress tracks progress across multiple operations
type MultiOperationProgress struct {
	PlanID              string                        `json:"planId"`
	TotalOperations     int                           `json:"totalOperations"`
	CompletedOperations int                           `json:"completedOperations"`
	FailedOperations    int                           `json:"failedOperations"`
	ActiveOperations    int                           `json:"activeOperations"`
	OverallProgress     float64                       `json:"overallProgress"` // 0.0 to 1.0
	EstimatedTimeLeft   time.Duration                 `json:"estimatedTimeLeft"`
	ElapsedTime         time.Duration                 `json:"elapsedTime"`
	StartedAt           time.Time                     `json:"startedAt"`
	Operations          map[string]*OperationProgress `json:"operations"`
	StageProgress       map[int]*StageProgress        `json:"stageProgress"`
	CurrentStage        int                           `json:"currentStage"`
	TotalStages         int                           `json:"totalStages"`
}

// StageProgress tracks progress for execution stages
type StageProgress struct {
	Stage               int        `json:"stage"`
	TotalOperations     int        `json:"totalOperations"`
	CompletedOperations int        `json:"completedOperations"`
	FailedOperations    int        `json:"failedOperations"`
	Progress            float64    `json:"progress"` // 0.0 to 1.0
	StartedAt           time.Time  `json:"startedAt"`
	CompletedAt         *time.Time `json:"completedAt,omitempty"`
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(updateInterval time.Duration) *ProgressTracker {
	return &ProgressTracker{
		updateInterval: updateInterval,
		output:         os.Stderr,
		stopChan:       make(chan struct{}),
		verboseMode:    false,
		quietMode:      false,
	}
}

// NewProgressTrackerWithOptions creates a progress tracker with custom options
func NewProgressTrackerWithOptions(updateInterval time.Duration, output io.Writer, verbose, quiet bool) *ProgressTracker {
	return &ProgressTracker{
		updateInterval: updateInterval,
		output:         output,
		stopChan:       make(chan struct{}),
		verboseMode:    verbose,
		quietMode:      quiet,
	}
}

// Start begins progress tracking for the given execution
func (pt *ProgressTracker) Start(ctx context.Context, executionProgress *ExecutorProgress) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isActive {
		return
	}

	pt.isActive = true
	pt.executionProgress = executionProgress
	pt.progress = ui.NewProgressIndicatorWithWriter(pt.output, pt.verboseMode, pt.quietMode)

	// Start the progress monitoring goroutine
	go pt.monitorProgress(ctx)
}

// Stop ends progress tracking
func (pt *ProgressTracker) Stop() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.isActive {
		return
	}

	pt.isActive = false
	close(pt.stopChan)

	// Stop any active progress indicators
	if pt.progress != nil {
		pt.progress.StopSpinner("Execution completed")
	}
	if pt.currentBar != nil {
		pt.currentBar.Finish("Done")
	}
}

// UpdateOperationProgress updates progress for a specific operation
func (pt *ProgressTracker) UpdateOperationProgress(operationID string, status OperationStatus, _ float64, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.isActive || pt.executionProgress == nil {
		return
	}

	// Update operation status in execution progress
	pt.executionProgress.OperationStatuses[operationID] = status

	// Log verbose updates
	if pt.verboseMode && pt.progress != nil {
		statusMsg := fmt.Sprintf("Operation %s: %s", operationID, status)
		if message != "" {
			statusMsg += fmt.Sprintf(" - %s", message)
		}
		pt.progress.PrintVerbose(statusMsg)
	}
}

// UpdateStageProgress updates progress for a specific stage
func (pt *ProgressTracker) UpdateStageProgress(stage int, _ float64, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.isActive || pt.executionProgress == nil {
		return
	}

	pt.executionProgress.CurrentStage = stage

	// Update current operation message
	if message != "" {
		pt.executionProgress.CurrentOperation = message
	}

	// Show stage progress in non-quiet mode
	if !pt.quietMode && pt.progress != nil {
		stageMsg := fmt.Sprintf("Stage %d/%d", stage+1, pt.executionProgress.TotalStages)
		if message != "" {
			stageMsg += fmt.Sprintf(": %s", message)
		}
		pt.progress.Print(stageMsg)
	}
}

// ShowOverallProgress displays the overall execution progress
func (pt *ProgressTracker) ShowOverallProgress() {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if !pt.isActive || pt.executionProgress == nil || pt.quietMode {
		return
	}

	// Calculate completed operations
	completed := 0
	for _, status := range pt.executionProgress.OperationStatuses {
		if status == OperationStatusCompleted {
			completed++
		}
	}

	total := pt.executionProgress.TotalOperations
	if total == 0 {
		return
	}

	// Update or create progress bar
	if pt.currentBar == nil {
		pt.currentBar = ui.NewProgressBar(pt.output, total, "Overall Progress")
	}

	pt.currentBar.Update(completed)

	// Show percentage and ETA if available
	percentage := float64(completed) / float64(total) * 100
	if pt.verboseMode && pt.progress != nil {
		msg := fmt.Sprintf("Progress: %.1f%% (%d/%d operations)", percentage, completed, total)
		if pt.executionProgress.EstimatedTimeLeft > 0 {
			msg += fmt.Sprintf(", ETA: %v", pt.executionProgress.EstimatedTimeLeft.Round(time.Second))
		}
		pt.progress.PrintVerbose(msg)
	}
}

// GetMultiOperationProgress returns detailed progress information
func (pt *ProgressTracker) GetMultiOperationProgress() *MultiOperationProgress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.executionProgress == nil {
		return nil
	}

	// Count operations by status
	completed := 0
	failed := 0
	active := 0

	operations := make(map[string]*OperationProgress)
	for opID, status := range pt.executionProgress.OperationStatuses {
		switch status {
		case OperationStatusCompleted:
			completed++
		case OperationStatusFailed:
			failed++
		case OperationStatusRunning, OperationStatusRetrying:
			active++
		}

		operations[opID] = &OperationProgress{
			OperationID: opID,
			Status:      status,
			UpdatedAt:   time.Now(),
			Progress:    pt.getOperationProgress(status),
		}
	}

	// Calculate overall progress
	total := pt.executionProgress.TotalOperations
	overallProgress := 0.0
	if total > 0 {
		overallProgress = float64(completed) / float64(total)
	}

	// Calculate elapsed time
	elapsedTime := time.Since(pt.executionProgress.StartedAt)

	return &MultiOperationProgress{
		PlanID:              pt.executionProgress.PlanID,
		TotalOperations:     total,
		CompletedOperations: completed,
		FailedOperations:    failed,
		ActiveOperations:    active,
		OverallProgress:     overallProgress,
		ElapsedTime:         elapsedTime,
		EstimatedTimeLeft:   pt.executionProgress.EstimatedTimeLeft,
		StartedAt:           pt.executionProgress.StartedAt,
		Operations:          operations,
		CurrentStage:        pt.executionProgress.CurrentStage,
		TotalStages:         pt.executionProgress.TotalStages,
	}
}

// monitorProgress runs the progress monitoring loop
func (pt *ProgressTracker) monitorProgress(ctx context.Context) {
	ticker := time.NewTicker(pt.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pt.stopChan:
			return
		case <-ticker.C:
			pt.ShowOverallProgress()
		}
	}
}

// getOperationProgress returns progress percentage for operation status
func (pt *ProgressTracker) getOperationProgress(status OperationStatus) float64 {
	switch status {
	case OperationStatusPending:
		return 0.0
	case OperationStatusRunning:
		return 0.5
	case OperationStatusRetrying:
		return 0.3
	case OperationStatusCompleted:
		return 1.0
	case OperationStatusFailed, OperationStatusSkipped:
		return 1.0 // Consider failed/skipped as "done"
	default:
		return 0.0
	}
}

// RealTimeProgressReporter provides real-time progress updates
type RealTimeProgressReporter struct {
	tracker    *ProgressTracker
	output     io.Writer
	verbose    bool
	quiet      bool
	logChannel chan ProgressEvent
	mu         sync.RWMutex
}

// NewRealTimeProgressReporter creates a new real-time progress reporter
func NewRealTimeProgressReporter(tracker *ProgressTracker, output io.Writer, verbose, quiet bool) *RealTimeProgressReporter {
	return &RealTimeProgressReporter{
		tracker:    tracker,
		output:     output,
		verbose:    verbose,
		quiet:      quiet,
		logChannel: make(chan ProgressEvent, 100), // Buffered channel for events
	}
}

// LogOperation logs an operation event
func (rpr *RealTimeProgressReporter) LogOperation(operationID, message string, eventType ProgressEventType) {
	if rpr.quiet {
		return
	}

	event := ProgressEvent{
		Type:      eventType,
		Message:   message,
		Operation: operationID,
		Timestamp: time.Now(),
	}

	select {
	case rpr.logChannel <- event:
	default:
		// Channel full, skip this event
	}

	// Immediate output for important events
	if eventType == ProgressEventError || (rpr.verbose && eventType == ProgressEventOperation) {
		timestamp := event.Timestamp.Format("15:04:05")
		_, _ = fmt.Fprintf(rpr.output, "[%s] %s: %s\n", timestamp, eventType, message)
	}
}

// Start begins real-time progress reporting
func (rpr *RealTimeProgressReporter) Start(ctx context.Context) {
	go rpr.processEvents(ctx)
}

// processEvents processes progress events in real-time
func (rpr *RealTimeProgressReporter) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-rpr.logChannel:
			rpr.handleEvent(event)
		}
	}
}

// handleEvent handles individual progress events
func (rpr *RealTimeProgressReporter) handleEvent(event ProgressEvent) {
	if rpr.quiet {
		return
	}

	timestamp := event.Timestamp.Format("15:04:05")

	switch event.Type {
	case ProgressEventStart:
		_, _ = fmt.Fprintf(rpr.output, "[%s] Starting execution: %s\n", timestamp, event.Message)
	case ProgressEventComplete:
		_, _ = fmt.Fprintf(rpr.output, "[%s] Execution completed: %s\n", timestamp, event.Message)
	case ProgressEventError:
		_, _ = fmt.Fprintf(rpr.output, "[%s] ERROR: %s\n", timestamp, event.Message)
	case ProgressEventOperation:
		if rpr.verbose {
			opMsg := event.Message
			if event.Operation != "" {
				opMsg = fmt.Sprintf("[%s] %s", event.Operation, event.Message)
			}
			_, _ = fmt.Fprintf(rpr.output, "[%s] %s\n", timestamp, opMsg)
		}
	case ProgressEventProgress:
		if rpr.verbose {
			_, _ = fmt.Fprintf(rpr.output, "[%s] Progress: %s\n", timestamp, event.Message)
		}
	}
}
