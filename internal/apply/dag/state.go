package dag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExecutionState tracks the state of a plan execution
type ExecutionState struct {
	// Metadata
	ExecutionID   string    `json:"executionId"`
	PlanID        string    `json:"planId"`
	ProjectID     string    `json:"projectId"`
	StartedAt     time.Time `json:"startedAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	
	// Status
	Status        ExecutionStatus        `json:"status"`
	CurrentStage  int                    `json:"currentStage"`
	TotalStages   int                    `json:"totalStages"`
	
	// Operation tracking
	Operations    map[string]*OperationState `json:"operations"`
	
	// Progress metrics
	TotalOps      int                    `json:"totalOps"`
	CompletedOps  int                    `json:"completedOps"`
	FailedOps     int                    `json:"failedOps"`
	SkippedOps    int                    `json:"skippedOps"`
	
	// Error tracking
	Errors        []ExecutionError       `json:"errors,omitempty"`
	LastError     string                 `json:"lastError,omitempty"`
	
	// Checkpoint info
	LastCheckpoint *CheckpointInfo       `json:"lastCheckpoint,omitempty"`
	
	// Concurrency control
	mu sync.RWMutex `json:"-"`
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	ExecutionStatusPaused    ExecutionStatus = "paused"
)

// OperationState tracks the state of a single operation
type OperationState struct {
	OperationID   string              `json:"operationId"`
	NodeID        string              `json:"nodeId"`
	Status        OperationStatus     `json:"status"`
	StartedAt     *time.Time          `json:"startedAt,omitempty"`
	CompletedAt   *time.Time          `json:"completedAt,omitempty"`
	Duration      time.Duration       `json:"duration,omitempty"`
	RetryCount    int                 `json:"retryCount"`
	Error         string              `json:"error,omitempty"`
	Result        interface{}         `json:"result,omitempty"`
	Checkpointed  bool                `json:"checkpointed"`
}

// OperationStatus represents the status of an operation
type OperationStatus string

const (
	OpStatusPending   OperationStatus = "pending"
	OpStatusRunning   OperationStatus = "running"
	OpStatusCompleted OperationStatus = "completed"
	OpStatusFailed    OperationStatus = "failed"
	OpStatusSkipped   OperationStatus = "skipped"
	OpStatusRetrying  OperationStatus = "retrying"
)

// ExecutionError represents an error during execution
type ExecutionError struct {
	OperationID string    `json:"operationId"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

// CheckpointInfo contains information about a checkpoint
type CheckpointInfo struct {
	CheckpointID string    `json:"checkpointId"`
	CreatedAt    time.Time `json:"createdAt"`
	Stage        int       `json:"stage"`
	OperationID  string    `json:"operationId,omitempty"`
}

// StateManager manages execution state
type StateManager struct {
	stateDir string
	mu       sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager(stateDir string) *StateManager {
	if stateDir == "" {
		homeDir, _ := os.UserHomeDir()
		stateDir = filepath.Join(homeDir, ".matlas", "state")
	}
	
	return &StateManager{
		stateDir: stateDir,
	}
}

// NewExecutionState creates a new execution state
func NewExecutionState(executionID, planID, projectID string, totalStages, totalOps int) *ExecutionState {
	return &ExecutionState{
		ExecutionID:  executionID,
		PlanID:       planID,
		ProjectID:    projectID,
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       ExecutionStatusPending,
		CurrentStage: 0,
		TotalStages:  totalStages,
		Operations:   make(map[string]*OperationState),
		TotalOps:     totalOps,
		CompletedOps: 0,
		FailedOps:    0,
		SkippedOps:   0,
		Errors:       make([]ExecutionError, 0),
	}
}

// SaveState persists the execution state to disk
func (sm *StateManager) SaveState(state *ExecutionState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Ensure state directory exists
	if err := os.MkdirAll(sm.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	
	// Update timestamp
	state.mu.Lock()
	state.UpdatedAt = time.Now()
	state.mu.Unlock()
	
	// Serialize state
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}
	
	// Write to file
	stateFile := filepath.Join(sm.stateDir, fmt.Sprintf("%s.json", state.ExecutionID))
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	
	return nil
}

// LoadState loads execution state from disk
func (sm *StateManager) LoadState(executionID string) (*ExecutionState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	stateFile := filepath.Join(sm.stateDir, fmt.Sprintf("%s.json", executionID))
	
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("state not found: %s", executionID)
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	
	var state ExecutionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to deserialize state: %w", err)
	}
	
	return &state, nil
}

// ListExecutions lists all execution states
func (sm *StateManager) ListExecutions() ([]string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	files, err := os.ReadDir(sm.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}
	
	executions := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			execID := file.Name()[:len(file.Name())-5] // Remove .json extension
			executions = append(executions, execID)
		}
	}
	
	return executions, nil
}

// DeleteState removes execution state from disk
func (sm *StateManager) DeleteState(executionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	stateFile := filepath.Join(sm.stateDir, fmt.Sprintf("%s.json", executionID))
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete state: %w", err)
	}
	
	return nil
}

// UpdateOperationState updates the state of a specific operation
func (state *ExecutionState) UpdateOperationState(opID string, status OperationStatus, err error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	
	opState, exists := state.Operations[opID]
	if !exists {
		opState = &OperationState{
			OperationID: opID,
			Status:      OpStatusPending,
		}
		state.Operations[opID] = opState
	}
	
	now := time.Now()
	prevStatus := opState.Status
	opState.Status = status
	
	// Update timestamps based on status
	switch status {
	case OpStatusRunning:
		if opState.StartedAt == nil {
			opState.StartedAt = &now
		}
	case OpStatusCompleted:
		opState.CompletedAt = &now
		if opState.StartedAt != nil {
			opState.Duration = now.Sub(*opState.StartedAt)
		}
		// Update counters
		if prevStatus != OpStatusCompleted {
			state.CompletedOps++
		}
	case OpStatusFailed:
		opState.CompletedAt = &now
		if err != nil {
			opState.Error = err.Error()
			state.Errors = append(state.Errors, ExecutionError{
				OperationID: opID,
				Message:     err.Error(),
				Timestamp:   now,
				Recoverable: true, // Can be determined by error type
			})
			state.LastError = err.Error()
		}
		// Update counters
		if prevStatus != OpStatusFailed {
			state.FailedOps++
		}
	case OpStatusSkipped:
		// Update counters
		if prevStatus != OpStatusSkipped {
			state.SkippedOps++
		}
	case OpStatusRetrying:
		opState.RetryCount++
	}
	
	state.UpdatedAt = now
}

// SetStage updates the current stage
func (state *ExecutionState) SetStage(stage int) {
	state.mu.Lock()
	defer state.mu.Unlock()
	
	state.CurrentStage = stage
	state.UpdatedAt = time.Now()
}

// SetStatus updates the execution status
func (state *ExecutionState) SetStatus(status ExecutionStatus) {
	state.mu.Lock()
	defer state.mu.Unlock()
	
	state.Status = status
	state.UpdatedAt = time.Now()
	
	if status == ExecutionStatusCompleted || status == ExecutionStatusFailed || status == ExecutionStatusCancelled {
		now := time.Now()
		state.CompletedAt = &now
	}
}

// GetProgress returns the current progress percentage
func (state *ExecutionState) GetProgress() float64 {
	state.mu.RLock()
	defer state.mu.RUnlock()
	
	if state.TotalOps == 0 {
		return 0
	}
	
	return float64(state.CompletedOps) / float64(state.TotalOps) * 100
}

// CanResume checks if execution can be resumed
func (state *ExecutionState) CanResume() bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	
	// Can resume if execution failed or was cancelled with some completed operations
	return (state.Status == ExecutionStatusFailed || state.Status == ExecutionStatusCancelled) &&
		state.CompletedOps > 0 &&
		state.CompletedOps < state.TotalOps
}

// GetPendingOperations returns all operations that haven't been completed
func (state *ExecutionState) GetPendingOperations() []string {
	state.mu.RLock()
	defer state.mu.RUnlock()
	
	pending := make([]string, 0)
	for opID, opState := range state.Operations {
		if opState.Status == OpStatusPending || opState.Status == OpStatusFailed {
			pending = append(pending, opID)
		}
	}
	
	return pending
}

// GetCompletedOperations returns all operations that have been completed
func (state *ExecutionState) GetCompletedOperations() []string {
	state.mu.RLock()
	defer state.mu.RUnlock()
	
	completed := make([]string, 0)
	for opID, opState := range state.Operations {
		if opState.Status == OpStatusCompleted {
			completed = append(completed, opID)
		}
	}
	
	return completed
}

// Clone creates a deep copy of the execution state
func (state *ExecutionState) Clone() *ExecutionState {
	state.mu.RLock()
	defer state.mu.RUnlock()
	
	clone := &ExecutionState{
		ExecutionID:    state.ExecutionID,
		PlanID:         state.PlanID,
		ProjectID:      state.ProjectID,
		StartedAt:      state.StartedAt,
		UpdatedAt:      state.UpdatedAt,
		CompletedAt:    state.CompletedAt,
		Status:         state.Status,
		CurrentStage:   state.CurrentStage,
		TotalStages:    state.TotalStages,
		Operations:     make(map[string]*OperationState),
		TotalOps:       state.TotalOps,
		CompletedOps:   state.CompletedOps,
		FailedOps:      state.FailedOps,
		SkippedOps:     state.SkippedOps,
		Errors:         make([]ExecutionError, len(state.Errors)),
		LastError:      state.LastError,
		LastCheckpoint: state.LastCheckpoint,
	}
	
	// Deep copy operations
	for opID, opState := range state.Operations {
		clone.Operations[opID] = &OperationState{
			OperationID:  opState.OperationID,
			NodeID:       opState.NodeID,
			Status:       opState.Status,
			StartedAt:    opState.StartedAt,
			CompletedAt:  opState.CompletedAt,
			Duration:     opState.Duration,
			RetryCount:   opState.RetryCount,
			Error:        opState.Error,
			Result:       opState.Result,
			Checkpointed: opState.Checkpointed,
		}
	}
	
	// Deep copy errors
	copy(clone.Errors, state.Errors)
	
	return clone
}

// Summary returns a human-readable summary of the execution state
func (state *ExecutionState) Summary() string {
	state.mu.RLock()
	defer state.mu.RUnlock()
	
	return fmt.Sprintf(
		"Execution %s: Status=%s, Progress=%.1f%% (%d/%d ops), Failed=%d, Stage=%d/%d",
		state.ExecutionID,
		state.Status,
		state.GetProgress(),
		state.CompletedOps,
		state.TotalOps,
		state.FailedOps,
		state.CurrentStage,
		state.TotalStages,
	)
}
