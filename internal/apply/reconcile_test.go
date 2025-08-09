package apply

import (
	"context"
	"testing"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestReconciliation_AutoFix_MVP(t *testing.T) {
	rm := &ReconciliationManager{
		idempotencyManager: NewIdempotencyManager(DefaultIdempotencyConfig()),
		config:             ReconciliationConfig{EnableAutoReconciliation: true, SafeOperationsOnly: true},
	}

	drift := ResourceDrift{
		ResourceID:   "cluster-1",
		ResourceKind: types.KindCluster,
		DriftType:    DriftTypeMetadata,
		Differences: []FieldDrift{
			{Path: "metadata.labels.env", DesiredValue: "prod", ActualValue: "staging"},
		},
		Reconcilable: true,
		AutoFix:      true,
		Fingerprint:  "abc123",
	}

	rec := &ResourceReconciliation{}
	if err := rm.executeAutoFix(context.Background(), drift, rec); err != nil {
		t.Fatalf("executeAutoFix returned error: %v", err)
	}

	if len(rec.Changes) != 1 {
		t.Fatalf("expected 1 change to be recorded, got %d", len(rec.Changes))
	}
	change := rec.Changes[0]
	if change.Path != "metadata.labels.env" || change.Type != ChangeTypeModify {
		t.Fatalf("unexpected change recorded: %+v", change)
	}
	if change.OldValue != "staging" || change.NewValue != "prod" {
		t.Fatalf("unexpected change values: old=%v new=%v", change.OldValue, change.NewValue)
	}
}
