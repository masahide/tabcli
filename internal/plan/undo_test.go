package plan

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func applyBeforeUndo(t *testing.T, restore UndoResult) (*mutationBridgeFixture, *ApplyManager) {
	t.Helper()
	now := time.Unix(1_700_000_000, 0)
	previews, bridge, manager := newApplyFixture(t, &now)
	bridge.restoreResult = restore
	createApplicablePreview(t, previews)
	if _, err := manager.Apply(context.Background(), "preview-apply"); err != nil {
		t.Fatal(err)
	}
	return bridge, manager
}

func TestUndoRestoresTabAndGroupSnapshot(t *testing.T) {
	bridge, manager := applyBeforeUndo(t, UndoResult{
		Status: UndoStatusSuccess, RestoredTabIDs: []int{1, 2}, RestoredGroupIDs: []int{100},
	})
	result, err := manager.Undo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != UndoStatusSuccess {
		t.Fatalf("undo result = %#v", result)
	}
	if len(bridge.restoreSnapshot.Groups) != 1 {
		t.Fatalf("restore snapshot = %#v", bridge.restoreSnapshot)
	}
	group := bridge.restoreSnapshot.Groups[0]
	if group.Title != "Work" || group.Color != "blue" || group.Collapsed {
		t.Fatalf("group restore state = %#v", group)
	}
}

func TestUndoReturnsClosedTabsAndDeletedGroupsAsUnrestorable(t *testing.T) {
	want := []string{"tab:2:closed", "group:100:deleted"}
	_, manager := applyBeforeUndo(t, UndoResult{
		Status: UndoStatusPartial, RestoredTabIDs: []int{1}, Unrestorable: want,
	})
	result, err := manager.Undo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != UndoStatusPartial || !reflect.DeepEqual(result.Unrestorable, want) {
		t.Fatalf("undo result = %#v, want unrestorable %v", result, want)
	}
}
