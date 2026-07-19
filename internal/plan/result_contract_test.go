package plan

import (
	"encoding/json"
	"testing"
)

func TestApplyAndUndoResultsAreMachineReadable(t *testing.T) {
	applyJSON, err := json.Marshal(ApplyResult{
		Status:            ApplyStatusPartial,
		AppliedOperations: []AppliedOperation{{Operation: Operation{Kind: OperationMoveTab, TabID: 2}}},
		Rollback:          &RollbackResult{Complete: false, Unrestorable: []string{"tab:2"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var apply map[string]any
	if err := json.Unmarshal(applyJSON, &apply); err != nil {
		t.Fatal(err)
	}
	if apply["status"] != string(ApplyStatusPartial) || apply["appliedOperations"] == nil || apply["rollback"] == nil {
		t.Fatalf("apply contract = %s", applyJSON)
	}

	undoJSON, err := json.Marshal(UndoResult{
		Status: UndoStatusPartial, RestoredTabIDs: []int{1}, Unrestorable: []string{"tab:2:closed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var undo map[string]any
	if err := json.Unmarshal(undoJSON, &undo); err != nil {
		t.Fatal(err)
	}
	if undo["status"] != string(UndoStatusPartial) || undo["unrestorable"] == nil {
		t.Fatalf("undo contract = %s", undoJSON)
	}
}
