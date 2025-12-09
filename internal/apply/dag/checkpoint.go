package dag

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Checkpoint represents a snapshot of execution state
type Checkpoint struct {
	// Metadata
	CheckpointID  string    `json:"checkpointId"`
	ExecutionID   string    `json:"executionId"`
	PlanID        string    `json:"planId"`
	CreatedAt     time.Time `json:"createdAt"`
	
	// State snapshot
	State         *ExecutionState `json:"state"`
	Graph         *Graph          `json:"graph"`
	
	// Checkpoint context
	Stage         int             `json:"stage"`
	OperationID   string          `json:"operationId,omitempty"`
	Reason        string          `json:"reason,omitempty"`
	
	// Metadata
	FileSize      int64           `json:"fileSize,omitempty"`
	Compressed    bool            `json:"compressed"`
}

// CheckpointManager manages checkpoints
type CheckpointManager struct {
	checkpointDir string
	maxCheckpoints int
	compression   bool
	mu            sync.RWMutex
}

// CheckpointConfig contains configuration for checkpoint management
type CheckpointConfig struct {
	CheckpointDir  string `json:"checkpointDir"`
	MaxCheckpoints int    `json:"maxCheckpoints"` // Maximum number of checkpoints to keep
	Compression    bool   `json:"compression"`     // Enable gzip compression
	AutoPrune      bool   `json:"autoPrune"`       // Automatically prune old checkpoints
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(config CheckpointConfig) *CheckpointManager {
	if config.CheckpointDir == "" {
		homeDir, _ := os.UserHomeDir()
		config.CheckpointDir = filepath.Join(homeDir, ".matlas", "checkpoints")
	}
	
	if config.MaxCheckpoints == 0 {
		config.MaxCheckpoints = 10 // Default to keeping 10 checkpoints
	}
	
	return &CheckpointManager{
		checkpointDir:  config.CheckpointDir,
		maxCheckpoints: config.MaxCheckpoints,
		compression:    config.Compression,
	}
}

// CreateCheckpoint creates a checkpoint of the current execution state
func (cm *CheckpointManager) CreateCheckpoint(executionID, planID string, state *ExecutionState, graph *Graph, stage int, operationID, reason string) (*Checkpoint, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Ensure checkpoint directory exists
	if err := os.MkdirAll(cm.checkpointDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}
	
	// Generate checkpoint ID
	checkpointID := fmt.Sprintf("cp-%s-%d-%d", executionID, stage, time.Now().Unix())
	
	// Clone state to avoid concurrent modifications
	stateSnapshot := state.Clone()
	
	// Create checkpoint
	checkpoint := &Checkpoint{
		CheckpointID: checkpointID,
		ExecutionID:  executionID,
		PlanID:       planID,
		CreatedAt:    time.Now(),
		State:        stateSnapshot,
		Graph:        graph,
		Stage:        stage,
		OperationID:  operationID,
		Reason:       reason,
		Compressed:   cm.compression,
	}
	
	// Update state's last checkpoint reference
	state.mu.Lock()
	state.LastCheckpoint = &CheckpointInfo{
		CheckpointID: checkpointID,
		CreatedAt:    checkpoint.CreatedAt,
		Stage:        stage,
		OperationID:  operationID,
	}
	state.mu.Unlock()
	
	// Serialize checkpoint
	if err := cm.writeCheckpoint(checkpoint); err != nil {
		return nil, fmt.Errorf("failed to write checkpoint: %w", err)
	}
	
	// Auto-prune old checkpoints if enabled
	if err := cm.pruneOldCheckpoints(executionID); err != nil {
		// Log warning but don't fail the checkpoint creation
		fmt.Fprintf(os.Stderr, "Warning: failed to prune old checkpoints: %v\n", err)
	}
	
	return checkpoint, nil
}

// writeCheckpoint writes a checkpoint to disk
func (cm *CheckpointManager) writeCheckpoint(checkpoint *Checkpoint) error {
	checkpointPath := cm.getCheckpointPath(checkpoint.CheckpointID)
	
	// Create file
	file, err := os.Create(checkpointPath)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint file: %w", err)
	}
	defer file.Close()
	
	var writer io.Writer = file
	
	// Add compression if enabled
	if cm.compression {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}
	
	// Encode checkpoint
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(checkpoint); err != nil {
		return fmt.Errorf("failed to encode checkpoint: %w", err)
	}
	
	// Get file size
	info, err := file.Stat()
	if err == nil {
		checkpoint.FileSize = info.Size()
	}
	
	return nil
}

// LoadCheckpoint loads a checkpoint from disk
func (cm *CheckpointManager) LoadCheckpoint(checkpointID string) (*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	checkpointPath := cm.getCheckpointPath(checkpointID)
	
	// Open file
	file, err := os.Open(checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
		}
		return nil, fmt.Errorf("failed to open checkpoint file: %w", err)
	}
	defer file.Close()
	
	var reader io.Reader = file
	
	// Detect compression (check if file starts with gzip magic number)
	magic := make([]byte, 2)
	if _, err := file.Read(magic); err != nil {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}
	file.Seek(0, 0) // Reset to beginning
	
	// Check for gzip magic number (0x1f, 0x8b)
	if magic[0] == 0x1f && magic[1] == 0x8b {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}
	
	// Decode checkpoint
	var checkpoint Checkpoint
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&checkpoint); err != nil {
		return nil, fmt.Errorf("failed to decode checkpoint: %w", err)
	}
	
	return &checkpoint, nil
}

// ListCheckpoints lists all checkpoints for an execution
func (cm *CheckpointManager) ListCheckpoints(executionID string) ([]*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	files, err := os.ReadDir(cm.checkpointDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Checkpoint{}, nil
		}
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}
	
	checkpoints := make([]*Checkpoint, 0)
	prefix := fmt.Sprintf("cp-%s-", executionID)
	
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			checkpointID := file.Name()[:len(file.Name())-5] // Remove .json extension
			
			// Filter by execution ID
			if len(checkpointID) > len(prefix) && checkpointID[:len(prefix)] == prefix {
				// Load checkpoint metadata (we could optimize this to only load metadata)
				checkpoint, err := cm.LoadCheckpoint(checkpointID)
				if err != nil {
					continue // Skip corrupted checkpoints
				}
				checkpoints = append(checkpoints, checkpoint)
			}
		}
	}
	
	// Sort by creation time (newest first)
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].CreatedAt.After(checkpoints[j].CreatedAt)
	})
	
	return checkpoints, nil
}

// GetLatestCheckpoint returns the most recent checkpoint for an execution
func (cm *CheckpointManager) GetLatestCheckpoint(executionID string) (*Checkpoint, error) {
	checkpoints, err := cm.ListCheckpoints(executionID)
	if err != nil {
		return nil, err
	}
	
	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found for execution: %s", executionID)
	}
	
	return checkpoints[0], nil
}

// DeleteCheckpoint deletes a specific checkpoint
func (cm *CheckpointManager) DeleteCheckpoint(checkpointID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	checkpointPath := cm.getCheckpointPath(checkpointID)
	if err := os.Remove(checkpointPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	
	return nil
}

// DeleteAllCheckpoints deletes all checkpoints for an execution
func (cm *CheckpointManager) DeleteAllCheckpoints(executionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	files, err := os.ReadDir(cm.checkpointDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read checkpoint directory: %w", err)
	}
	
	prefix := fmt.Sprintf("cp-%s-", executionID)
	
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			checkpointID := file.Name()[:len(file.Name())-5]
			
			if len(checkpointID) > len(prefix) && checkpointID[:len(prefix)] == prefix {
				checkpointPath := filepath.Join(cm.checkpointDir, file.Name())
				if err := os.Remove(checkpointPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to delete checkpoint %s: %w", checkpointID, err)
				}
			}
		}
	}
	
	return nil
}

// pruneOldCheckpoints removes old checkpoints beyond the max limit
func (cm *CheckpointManager) pruneOldCheckpoints(executionID string) error {
	checkpoints, err := cm.ListCheckpoints(executionID)
	if err != nil {
		return err
	}
	
	// Keep only the most recent maxCheckpoints
	if len(checkpoints) > cm.maxCheckpoints {
		toDelete := checkpoints[cm.maxCheckpoints:]
		for _, checkpoint := range toDelete {
			if err := cm.DeleteCheckpoint(checkpoint.CheckpointID); err != nil {
				return fmt.Errorf("failed to delete old checkpoint: %w", err)
			}
		}
	}
	
	return nil
}

// ValidateCheckpoint validates that a checkpoint is intact and readable
func (cm *CheckpointManager) ValidateCheckpoint(checkpointID string) error {
	checkpoint, err := cm.LoadCheckpoint(checkpointID)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}
	
	// Basic validation
	if checkpoint.State == nil {
		return fmt.Errorf("checkpoint has nil state")
	}
	
	if checkpoint.ExecutionID == "" {
		return fmt.Errorf("checkpoint has empty execution ID")
	}
	
	if checkpoint.PlanID == "" {
		return fmt.Errorf("checkpoint has empty plan ID")
	}
	
	// Validate state has operations
	if len(checkpoint.State.Operations) == 0 {
		return fmt.Errorf("checkpoint state has no operations")
	}
	
	return nil
}

// RestoreFromCheckpoint restores execution state from a checkpoint
func (cm *CheckpointManager) RestoreFromCheckpoint(checkpointID string) (*ExecutionState, *Graph, error) {
	checkpoint, err := cm.LoadCheckpoint(checkpointID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}
	
	// Validate checkpoint
	if err := cm.ValidateCheckpoint(checkpointID); err != nil {
		return nil, nil, fmt.Errorf("checkpoint validation failed: %w", err)
	}
	
	// Clone state to avoid modifications affecting the checkpoint
	restoredState := checkpoint.State.Clone()
	
	// Update status to indicate resumed execution
	restoredState.SetStatus(ExecutionStatusRunning)
	
	return restoredState, checkpoint.Graph, nil
}

// getCheckpointPath returns the file path for a checkpoint
func (cm *CheckpointManager) getCheckpointPath(checkpointID string) string {
	return filepath.Join(cm.checkpointDir, fmt.Sprintf("%s.json", checkpointID))
}

// GetCheckpointSize returns the size of a checkpoint in bytes
func (cm *CheckpointManager) GetCheckpointSize(checkpointID string) (int64, error) {
	checkpointPath := cm.getCheckpointPath(checkpointID)
	
	info, err := os.Stat(checkpointPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat checkpoint: %w", err)
	}
	
	return info.Size(), nil
}

// GetTotalCheckpointSize returns the total size of all checkpoints for an execution
func (cm *CheckpointManager) GetTotalCheckpointSize(executionID string) (int64, error) {
	checkpoints, err := cm.ListCheckpoints(executionID)
	if err != nil {
		return 0, err
	}
	
	var totalSize int64
	for _, checkpoint := range checkpoints {
		size, err := cm.GetCheckpointSize(checkpoint.CheckpointID)
		if err != nil {
			continue // Skip checkpoints that can't be stat'd
		}
		totalSize += size
	}
	
	return totalSize, nil
}

// ShouldCreateCheckpoint determines if a checkpoint should be created
// based on the current state and configuration
func ShouldCreateCheckpoint(state *ExecutionState, stageCompleted bool, highRiskOp bool) bool {
	// Always checkpoint at stage boundaries
	if stageCompleted {
		return true
	}
	
	// Checkpoint before high-risk operations
	if highRiskOp {
		return true
	}
	
	// Checkpoint periodically (every 10 completed operations)
	if state.CompletedOps > 0 && state.CompletedOps%10 == 0 {
		return true
	}
	
	return false
}

// CheckpointSummary provides a summary of checkpoints for an execution
type CheckpointSummary struct {
	ExecutionID      string    `json:"executionId"`
	TotalCheckpoints int       `json:"totalCheckpoints"`
	TotalSize        int64     `json:"totalSize"`
	OldestCheckpoint *time.Time `json:"oldestCheckpoint,omitempty"`
	NewestCheckpoint *time.Time `json:"newestCheckpoint,omitempty"`
}

// GetCheckpointSummary returns a summary of checkpoints for an execution
func (cm *CheckpointManager) GetCheckpointSummary(executionID string) (*CheckpointSummary, error) {
	checkpoints, err := cm.ListCheckpoints(executionID)
	if err != nil {
		return nil, err
	}
	
	summary := &CheckpointSummary{
		ExecutionID:      executionID,
		TotalCheckpoints: len(checkpoints),
	}
	
	if len(checkpoints) == 0 {
		return summary, nil
	}
	
	// Get total size
	totalSize, err := cm.GetTotalCheckpointSize(executionID)
	if err == nil {
		summary.TotalSize = totalSize
	}
	
	// Get newest and oldest timestamps
	summary.NewestCheckpoint = &checkpoints[0].CreatedAt
	summary.OldestCheckpoint = &checkpoints[len(checkpoints)-1].CreatedAt
	
	return summary, nil
}
