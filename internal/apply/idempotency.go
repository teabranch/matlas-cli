package apply

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// IdempotencyManager manages operation state and resource fingerprinting for idempotent operations
type IdempotencyManager struct {
	mu                   sync.RWMutex
	operationStates      map[string]*OperationState
	resourceFingerprints map[string]string
	resourceOwnership    map[string]*ResourceOwnership
	checkpoints          map[string]*Checkpoint
	config               IdempotencyConfig
}

// IdempotencyConfig contains configuration for idempotency management
type IdempotencyConfig struct {
	// State tracking settings
	EnableStateTracking bool          `json:"enableStateTracking"`
	StateTTL            time.Duration `json:"stateTTL"`
	CheckpointInterval  time.Duration `json:"checkpointInterval"`

	// Fingerprinting settings
	EnableFingerprinting bool     `json:"enableFingerprinting"`
	FingerprintFields    []string `json:"fingerprintFields"`
	IgnoreMetadataFields []string `json:"ignoreMetadataFields"`

	// Deduplication settings
	EnableDeduplication bool          `json:"enableDeduplication"`
	DeduplicationWindow time.Duration `json:"deduplicationWindow"`

	// Resource ownership settings
	EnableOwnershipTracking bool          `json:"enableOwnershipTracking"`
	OwnershipTTL            time.Duration `json:"ownershipTTL"`
}

// OperationState represents the current state of an operation
type OperationState struct {
	ID           string             `json:"id"`
	PlanID       string             `json:"planId"`
	Status       OperationStatus    `json:"status"`
	ResourceID   string             `json:"resourceId"`
	ResourceKind types.ResourceKind `json:"resourceKind"`

	// State tracking
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Idempotency tracking
	Fingerprint   string          `json:"fingerprint"`
	PreviousState *OperationState `json:"previousState,omitempty"`
	RetryCount    int             `json:"retryCount"`
	IsRetry       bool            `json:"isRetry"`

	// Checkpointing
	LastCheckpoint *time.Time             `json:"lastCheckpoint,omitempty"`
	CheckpointData map[string]interface{} `json:"checkpointData,omitempty"`

	// Error tracking
	LastError    string           `json:"lastError,omitempty"`
	ErrorHistory []OperationError `json:"errorHistory,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceOwnership tracks which plan/operation owns a resource
type ResourceOwnership struct {
	ResourceID   string                 `json:"resourceId"`
	ResourceKind types.ResourceKind     `json:"resourceKind"`
	OwnerPlanID  string                 `json:"ownerPlanId"`
	OwnerOpID    string                 `json:"ownerOpId"`
	AcquiredAt   time.Time              `json:"acquiredAt"`
	ExpiresAt    time.Time              `json:"expiresAt"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Checkpoint represents a recoverable state during operation execution
type Checkpoint struct {
	ID            string                 `json:"id"`
	OperationID   string                 `json:"operationId"`
	PlanID        string                 `json:"planId"`
	CreatedAt     time.Time              `json:"createdAt"`
	Stage         string                 `json:"stage"`
	Data          map[string]interface{} `json:"data"`
	ResourceState interface{}            `json:"resourceState,omitempty"`
}

// OperationError represents a recoverable error from an operation
type OperationError struct {
	Timestamp   time.Time `json:"timestamp"`
	Error       string    `json:"error"`
	ErrorType   string    `json:"errorType"`
	Recoverable bool      `json:"recoverable"`
	RetryCount  int       `json:"retryCount"`
}

// ResourceFingerprint represents a content-based fingerprint of a resource
type ResourceFingerprint struct {
	ResourceID   string                 `json:"resourceId"`
	ResourceKind types.ResourceKind     `json:"resourceKind"`
	Fingerprint  string                 `json:"fingerprint"`
	ComputedAt   time.Time              `json:"computedAt"`
	Fields       map[string]interface{} `json:"fields"`
}

// NewIdempotencyManager creates a new idempotency manager
func NewIdempotencyManager(config IdempotencyConfig) *IdempotencyManager {
	return &IdempotencyManager{
		operationStates:      make(map[string]*OperationState),
		resourceFingerprints: make(map[string]string),
		resourceOwnership:    make(map[string]*ResourceOwnership),
		checkpoints:          make(map[string]*Checkpoint),
		config:               config,
	}
}

// DefaultIdempotencyConfig returns a default configuration for idempotency management
func DefaultIdempotencyConfig() IdempotencyConfig {
	return IdempotencyConfig{
		EnableStateTracking:     true,
		StateTTL:                24 * time.Hour,
		CheckpointInterval:      5 * time.Minute,
		EnableFingerprinting:    true,
		FingerprintFields:       []string{}, // Empty means all fields
		IgnoreMetadataFields:    []string{"createdAt", "updatedAt", "lastModified", "etag"},
		EnableDeduplication:     true,
		DeduplicationWindow:     1 * time.Hour,
		EnableOwnershipTracking: true,
		OwnershipTTL:            2 * time.Hour,
	}
}

// GetOperationState retrieves the current state of an operation
func (im *IdempotencyManager) GetOperationState(operationID string) (*OperationState, bool) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	state, exists := im.operationStates[operationID]
	if !exists {
		return nil, false
	}

	// Check if state has expired
	if im.config.StateTTL > 0 && time.Since(state.UpdatedAt) > im.config.StateTTL {
		return nil, false
	}

	return state, true
}

// UpdateOperationState updates the state of an operation
func (im *IdempotencyManager) UpdateOperationState(state *OperationState) error {
	if state == nil {
		return fmt.Errorf("operation state cannot be nil")
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	state.UpdatedAt = time.Now()
	im.operationStates[state.ID] = state

	return nil
}

// CreateOperationState creates a new operation state
func (im *IdempotencyManager) CreateOperationState(operation *PlannedOperation, fingerprint string) *OperationState {
	now := time.Now()

	state := &OperationState{
		ID:             operation.ID,
		PlanID:         "", // Will be set by caller if available
		Status:         OperationStatusPending,
		ResourceID:     operation.ResourceName,
		ResourceKind:   operation.ResourceType,
		CreatedAt:      now,
		UpdatedAt:      now,
		Fingerprint:    fingerprint,
		RetryCount:     0,
		IsRetry:        false,
		CheckpointData: make(map[string]interface{}),
		Metadata:       make(map[string]interface{}),
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	im.operationStates[state.ID] = state

	return state
}

// ComputeResourceFingerprint computes a fingerprint for a resource
func (im *IdempotencyManager) ComputeResourceFingerprint(resource interface{}, resourceKind types.ResourceKind) (string, error) {
	if !im.config.EnableFingerprinting {
		return "", nil
	}

	// Convert resource to map for field filtering
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resource for fingerprinting: %w", err)
	}

	var resourceMap map[string]interface{}
	if err := json.Unmarshal(resourceBytes, &resourceMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal resource for fingerprinting: %w", err)
	}

	// Filter out ignored fields
	filteredResource := im.filterFingerprintFields(resourceMap)

	// Compute stable hash
	return im.computeStableHash(filteredResource)
}

// filterFingerprintFields filters out fields that should not be included in fingerprinting
func (im *IdempotencyManager) filterFingerprintFields(resource map[string]interface{}) map[string]interface{} {
	filtered := make(map[string]interface{})

	// If specific fields are configured, include only those
	if len(im.config.FingerprintFields) > 0 {
		for _, field := range im.config.FingerprintFields {
			if value, exists := resource[field]; exists {
				filtered[field] = value
			}
		}
		return filtered
	}

	// Otherwise, include all fields except ignored ones
	for key, value := range resource {
		shouldIgnore := false
		for _, ignoredField := range im.config.IgnoreMetadataFields {
			if key == ignoredField {
				shouldIgnore = true
				break
			}
		}

		if !shouldIgnore {
			filtered[key] = value
		}
	}

	return filtered
}

// computeStableHash computes a stable hash of the given data
func (im *IdempotencyManager) computeStableHash(data interface{}) (string, error) {
	// Use deterministic JSON marshaling for stable hashing
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data for hashing: %w", err)
	}

	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}

// IsOperationIdempotent checks if an operation is idempotent (no actual changes needed)
func (im *IdempotencyManager) IsOperationIdempotent(operationID string, currentFingerprint string) (bool, error) {
	if !im.config.EnableFingerprinting {
		return false, nil
	}

	state, exists := im.GetOperationState(operationID)
	if !exists {
		return false, nil
	}

	// If the fingerprints match and the operation was successful, it's idempotent
	return state.Fingerprint == currentFingerprint && state.Status == OperationStatusCompleted, nil
}

// IsDuplicateOperation checks if this operation is a duplicate within the deduplication window
func (im *IdempotencyManager) IsDuplicateOperation(operation *PlannedOperation) (bool, *OperationState, error) {
	if !im.config.EnableDeduplication {
		return false, nil, nil
	}

	im.mu.RLock()
	defer im.mu.RUnlock()

	cutoff := time.Now().Add(-im.config.DeduplicationWindow)

	for _, state := range im.operationStates {
		// Skip if outside deduplication window
		if state.UpdatedAt.Before(cutoff) {
			continue
		}

		// Check if this is a duplicate based on resource and operation type
		if state.ResourceID == operation.ResourceName &&
			state.ResourceKind == operation.ResourceType &&
			state.Status != OperationStatusFailed {
			return true, state, nil
		}
	}

	return false, nil, nil
}

// AcquireResourceOwnership attempts to acquire exclusive ownership of a resource
func (im *IdempotencyManager) AcquireResourceOwnership(resourceID string, resourceKind types.ResourceKind, planID, operationID string) (*ResourceOwnership, error) {
	if !im.config.EnableOwnershipTracking {
		return nil, nil
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	ownershipKey := fmt.Sprintf("%s/%s", resourceKind, resourceID)

	// Check if resource is already owned
	if existing, exists := im.resourceOwnership[ownershipKey]; exists {
		// Check if ownership has expired
		if time.Now().Before(existing.ExpiresAt) {
			if existing.OwnerPlanID != planID {
				return nil, fmt.Errorf("resource %s is owned by plan %s", resourceID, existing.OwnerPlanID)
			}
			// Extend ownership if same plan
			existing.ExpiresAt = time.Now().Add(im.config.OwnershipTTL)
			return existing, nil
		}
	}

	// Acquire new ownership
	ownership := &ResourceOwnership{
		ResourceID:   resourceID,
		ResourceKind: resourceKind,
		OwnerPlanID:  planID,
		OwnerOpID:    operationID,
		AcquiredAt:   time.Now(),
		ExpiresAt:    time.Now().Add(im.config.OwnershipTTL),
		Metadata:     make(map[string]interface{}),
	}

	im.resourceOwnership[ownershipKey] = ownership
	return ownership, nil
}

// ReleaseResourceOwnership releases ownership of a resource
func (im *IdempotencyManager) ReleaseResourceOwnership(resourceID string, resourceKind types.ResourceKind, planID string) error {
	if !im.config.EnableOwnershipTracking {
		return nil
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	ownershipKey := fmt.Sprintf("%s/%s", resourceKind, resourceID)

	if existing, exists := im.resourceOwnership[ownershipKey]; exists {
		if existing.OwnerPlanID == planID {
			delete(im.resourceOwnership, ownershipKey)
		} else {
			return fmt.Errorf("cannot release ownership: resource owned by different plan")
		}
	}

	return nil
}

// CreateCheckpoint creates a checkpoint for an operation
func (im *IdempotencyManager) CreateCheckpoint(operationID, planID, stage string, data map[string]interface{}, resourceState interface{}) (*Checkpoint, error) {
	checkpoint := &Checkpoint{
		ID:            fmt.Sprintf("%s-%s-%d", operationID, stage, time.Now().Unix()),
		OperationID:   operationID,
		PlanID:        planID,
		CreatedAt:     time.Now(),
		Stage:         stage,
		Data:          data,
		ResourceState: resourceState,
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	im.checkpoints[checkpoint.ID] = checkpoint

	// Update operation state with checkpoint
	if state, exists := im.operationStates[operationID]; exists {
		now := time.Now()
		state.LastCheckpoint = &now
		state.CheckpointData = data
	}

	return checkpoint, nil
}

// GetLatestCheckpoint retrieves the latest checkpoint for an operation
func (im *IdempotencyManager) GetLatestCheckpoint(operationID string) (*Checkpoint, bool) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	var latest *Checkpoint
	for _, checkpoint := range im.checkpoints {
		if checkpoint.OperationID == operationID {
			if latest == nil || checkpoint.CreatedAt.After(latest.CreatedAt) {
				latest = checkpoint
			}
		}
	}

	return latest, latest != nil
}

// CleanupExpiredState removes expired operation states and ownership records
func (im *IdempotencyManager) CleanupExpiredState() {
	im.mu.Lock()
	defer im.mu.Unlock()

	now := time.Now()

	// Clean up expired operation states
	if im.config.StateTTL > 0 {
		for id, state := range im.operationStates {
			if now.Sub(state.UpdatedAt) > im.config.StateTTL {
				delete(im.operationStates, id)
			}
		}
	}

	// Clean up expired ownership records
	for key, ownership := range im.resourceOwnership {
		if now.After(ownership.ExpiresAt) {
			delete(im.resourceOwnership, key)
		}
	}

	// Clean up old checkpoints (keep only the last 10 per operation)
	operationCheckpoints := make(map[string][]*Checkpoint)
	for _, checkpoint := range im.checkpoints {
		operationCheckpoints[checkpoint.OperationID] = append(operationCheckpoints[checkpoint.OperationID], checkpoint)
	}

	for _, checkpoints := range operationCheckpoints {
		if len(checkpoints) > 10 {
			// Sort by creation time and keep only the latest 10
			for i := 0; i < len(checkpoints)-10; i++ {
				delete(im.checkpoints, checkpoints[i].ID)
			}
		}
	}
}

// StartCleanupWorker starts a background worker to periodically clean up expired state
func (im *IdempotencyManager) StartCleanupWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			im.CleanupExpiredState()
		}
	}
}

// ListOperationStates returns a snapshot list of all tracked operation states
func (im *IdempotencyManager) ListOperationStates() []*OperationState {
	im.mu.RLock()
	defer im.mu.RUnlock()

	states := make([]*OperationState, 0, len(im.operationStates))
	for _, s := range im.operationStates {
		states = append(states, s)
	}
	return states
}

// ListOperationStatesByProject returns operation states that match the given project ID
// This uses OperationState.Metadata["projectID"] when available
func (im *IdempotencyManager) ListOperationStatesByProject(projectID string) []*OperationState {
	if projectID == "" {
		return nil
	}
	im.mu.RLock()
	defer im.mu.RUnlock()

	var states []*OperationState
	for _, s := range im.operationStates {
		if s != nil && s.Metadata != nil {
			if pid, ok := s.Metadata["projectID"].(string); ok && pid == projectID {
				states = append(states, s)
			}
		}
	}
	return states
}
