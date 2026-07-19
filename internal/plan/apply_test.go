package plan

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/masahide/tabcli/internal/tools"
)

type mutationBridgeFixture struct {
	snapshot         tools.ChromeSnapshot
	applied          []Operation
	failAt           int
	rollbackResult   RollbackResult
	rollbackReceived []AppliedOperation
	restoreResult    UndoResult
	restoreSnapshot  UndoSnapshot
}

func (bridge *mutationBridgeFixture) Snapshot(context.Context) (tools.ChromeSnapshot, error) {
	return bridge.snapshot, nil
}

func (bridge *mutationBridgeFixture) ApplyOperation(_ context.Context, operation Operation) (AppliedOperation, error) {
	if bridge.failAt > 0 && len(bridge.applied)+1 == bridge.failAt {
		return AppliedOperation{}, errors.New("injected operation failure")
	}
	bridge.applied = append(bridge.applied, operation)
	return AppliedOperation{Operation: operation, CreatedGroupID: 500 + len(bridge.applied)}, nil
}

func (bridge *mutationBridgeFixture) Rollback(_ context.Context, _ UndoSnapshot, successful []AppliedOperation) (RollbackResult, error) {
	bridge.rollbackReceived = append([]AppliedOperation(nil), successful...)
	return bridge.rollbackResult, nil
}

func (bridge *mutationBridgeFixture) Restore(_ context.Context, snapshot UndoSnapshot) (UndoResult, error) {
	bridge.restoreSnapshot = snapshot
	return bridge.restoreResult, nil
}

func TestApplyRollsBackEveryFailurePositionAndReportsPartialRollback(t *testing.T) {
	for failAt := 1; failAt <= 2; failAt++ {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			now := time.Unix(1_700_000_000, 0)
			previews, bridge, manager := newApplyFixture(t, &now)
			bridge.failAt = failAt
			_, err := previews.Preview(context.Background(), ClassificationPlan{
				Policy:      PolicyRebuildSelected,
				Assignments: []Assignment{{TabID: 2, Destination: Destination{Title: "New", Color: "green"}}},
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = manager.Apply(context.Background(), "preview-apply")
			var toolError *tools.Error
			if !errors.As(err, &toolError) || toolError.Code != tools.CodeApplyFailedRolledBack {
				t.Fatalf("failure at %d error = %v", failAt, err)
			}
			if len(bridge.rollbackReceived) != failAt-1 {
				t.Fatalf("rollback received %d operations, want %d", len(bridge.rollbackReceived), failAt-1)
			}
		})
	}

	now := time.Unix(1_700_000_000, 0)
	previews, bridge, manager := newApplyFixture(t, &now)
	bridge.failAt = 2
	bridge.rollbackResult = RollbackResult{Complete: false, Unrestorable: []string{"tab:2"}}
	_, err := previews.Preview(context.Background(), ClassificationPlan{
		Policy:      PolicyRebuildSelected,
		Assignments: []Assignment{{TabID: 2, Destination: Destination{Title: "New", Color: "green"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Apply(context.Background(), "preview-apply")
	var toolError *tools.Error
	if !errors.As(err, &toolError) || toolError.Code != tools.CodeApplyPartial {
		t.Fatalf("partial rollback error = %v", err)
	}
}

func newApplyFixture(t *testing.T, now *time.Time) (*PreviewManager, *mutationBridgeFixture, *ApplyManager) {
	t.Helper()
	bridge := &mutationBridgeFixture{snapshot: previewSnapshot(), rollbackResult: RollbackResult{Complete: true}}
	previews := NewPreviewManager(bridge, PreviewOptions{
		Now: func() time.Time { return *now }, RandomID: func() string { return "preview-apply" },
		ContentRevisionValid: func(int, string) bool { return true },
	})
	manager := NewApplyManager(previews, bridge)
	return previews, bridge, manager
}

func createApplicablePreview(t *testing.T, previews *PreviewManager) {
	t.Helper()
	_, err := previews.Preview(context.Background(), ClassificationPlan{
		Policy:      PolicyExistingGroupsOnly,
		Assignments: []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(100)}}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestApplyValidPreviewAndRejectsDoubleApply(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	previews, bridge, manager := newApplyFixture(t, &now)
	createApplicablePreview(t, previews)
	result, err := manager.Apply(context.Background(), "preview-apply")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != ApplyStatusSuccess || len(bridge.applied) != 1 {
		t.Fatalf("result = %#v, applied = %#v", result, bridge.applied)
	}
	if manager.undoSnapshot == nil || len(manager.undoSnapshot.Tabs) != 1 || manager.undoSnapshot.Tabs[0].TabID != 2 || len(manager.undoSnapshot.Groups) != 1 || manager.undoSnapshot.Groups[0].GroupID != 100 {
		t.Fatalf("undo snapshot includes unrelated state: %#v", manager.undoSnapshot)
	}
	_, err = manager.Apply(context.Background(), "preview-apply")
	var toolError *tools.Error
	if !errors.As(err, &toolError) || toolError.Code != tools.CodePreviewNotFound {
		t.Fatalf("double apply error = %v", err)
	}
}

func TestApplyRejectsExpiredUnknownAndStalePreviewWithoutMutation(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		now := time.Unix(1_700_000_000, 0)
		previews, bridge, manager := newApplyFixture(t, &now)
		createApplicablePreview(t, previews)
		now = now.Add(5 * time.Minute)
		_, err := manager.Apply(context.Background(), "preview-apply")
		if err == nil || len(bridge.applied) != 0 {
			t.Fatalf("expired apply = %v, applied = %#v", err, bridge.applied)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		now := time.Unix(1_700_000_000, 0)
		_, bridge, manager := newApplyFixture(t, &now)
		_, err := manager.Apply(context.Background(), "unknown")
		if err == nil || len(bridge.applied) != 0 {
			t.Fatalf("unknown apply = %v, applied = %#v", err, bridge.applied)
		}
	})
	t.Run("stale", func(t *testing.T) {
		now := time.Unix(1_700_000_000, 0)
		previews, bridge, manager := newApplyFixture(t, &now)
		createApplicablePreview(t, previews)
		bridge.snapshot.Groups[0].Color = "red"
		_, err := manager.Apply(context.Background(), "preview-apply")
		var toolError *tools.Error
		if !errors.As(err, &toolError) || toolError.Code != tools.CodePlanStale || len(bridge.applied) != 0 {
			t.Fatalf("stale apply = %v, applied = %#v", err, bridge.applied)
		}
	})
}

func TestUndoSnapshotChangesOnlyAfterSuccessfulApply(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	previews, bridge, manager := newApplyFixture(t, &now)
	createApplicablePreview(t, previews)
	if manager.undoSnapshot != nil {
		t.Fatal("preview alone created undo state")
	}
	if _, err := manager.Apply(context.Background(), "preview-apply"); err != nil {
		t.Fatal(err)
	}
	first := *manager.undoSnapshot

	bridge.snapshot.Tabs[1].Index = 5
	bridge.failAt = len(bridge.applied) + 1
	createApplicablePreview(t, previews)
	if _, err := manager.Apply(context.Background(), "preview-apply"); err == nil {
		t.Fatal("injected failure succeeded")
	}
	if manager.undoSnapshot == nil || !reflect.DeepEqual(*manager.undoSnapshot, first) {
		t.Fatalf("failed apply replaced undo state: %#v, want %#v", manager.undoSnapshot, first)
	}

	bridge.failAt = 0
	createApplicablePreview(t, previews)
	if _, err := manager.Apply(context.Background(), "preview-apply"); err != nil {
		t.Fatal(err)
	}
	if manager.undoSnapshot == nil || reflect.DeepEqual(*manager.undoSnapshot, first) {
		t.Fatal("second successful apply did not replace undo state")
	}
}
