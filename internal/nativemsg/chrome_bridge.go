package nativemsg

import (
	"context"

	"github.com/masahide/tabcli/internal/plan"
	"github.com/masahide/tabcli/internal/tools"
)

const OperationSnapshot = "snapshot"

const OperationContentGet = "content_get"

const OperationContentCompare = "content_compare"

const OperationContentDiff = "content_diff"

const OperationContentRevisionsValidate = "content_revisions_validate"

const OperationTabsClose = "tabs_close"

const (
	OperationApply    = "apply_operation"
	OperationRollback = "rollback"
	OperationUndo     = "undo_restore"
)

type ChromeBridge struct {
	Native *Bridge
}

func (bridge ChromeBridge) Snapshot(ctx context.Context) (tools.ChromeSnapshot, error) {
	var snapshot tools.ChromeSnapshot
	err := bridge.Native.Call(ctx, OperationSnapshot, struct{}{}, &snapshot)
	return snapshot, err
}

func (bridge ChromeBridge) Content(ctx context.Context, input tools.ContentGetInput) (tools.ContentGetResult, error) {
	var result tools.ContentGetResult
	err := bridge.Native.Call(ctx, OperationContentGet, input, &result)
	if err == nil {
		result.ProtocolVersion = tools.ProtocolVersion
	}
	return result, err
}

func (bridge ChromeBridge) CompareContent(ctx context.Context, input tools.ContentCompareInput) (tools.ContentCompareResult, error) {
	var result tools.ContentCompareResult
	err := bridge.Native.Call(ctx, OperationContentCompare, input, &result)
	if err == nil {
		result.ProtocolVersion = tools.ProtocolVersion
	}
	return result, err
}

func (bridge ChromeBridge) DiffContent(ctx context.Context, input tools.ContentDiffInput) (tools.ContentDiffResult, error) {
	var result tools.ContentDiffResult
	err := bridge.Native.Call(ctx, OperationContentDiff, input, &result)
	if err == nil {
		result.ProtocolVersion = tools.ProtocolVersion
	}
	return result, err
}

func (bridge ChromeBridge) CloseTabs(ctx context.Context, tabIDs []int) (tools.TabsCloseResult, error) {
	var result tools.TabsCloseResult
	err := bridge.Native.Call(ctx, OperationTabsClose, struct {
		TabIDs []int `json:"tabIds"`
	}{TabIDs: tabIDs}, &result)
	return result, err
}

func (bridge ChromeBridge) ValidateContentRevisions(ctx context.Context, references []tools.ContentReference) error {
	if len(references) == 0 {
		return nil
	}
	var response struct {
		Valid         bool  `json:"valid"`
		InvalidTabIDs []int `json:"invalidTabIds"`
	}
	if err := bridge.Native.Call(ctx, OperationContentRevisionsValidate, references, &response); err != nil {
		return err
	}
	if !response.Valid {
		return &tools.Error{
			Code: tools.CodeContentStale, Message: "page content revision is no longer valid",
			Details: map[string]any{"invalidTabIds": response.InvalidTabIDs},
		}
	}
	return nil
}

func (bridge ChromeBridge) ApplyOperation(ctx context.Context, operation plan.Operation) (plan.AppliedOperation, error) {
	var result plan.AppliedOperation
	err := bridge.Native.Call(ctx, OperationApply, operation, &result)
	return result, err
}

func (bridge ChromeBridge) Rollback(ctx context.Context, snapshot plan.UndoSnapshot, successful []plan.AppliedOperation) (plan.RollbackResult, error) {
	var result plan.RollbackResult
	err := bridge.Native.Call(ctx, OperationRollback, struct {
		Snapshot   plan.UndoSnapshot       `json:"snapshot"`
		Successful []plan.AppliedOperation `json:"successful"`
	}{snapshot, successful}, &result)
	return result, err
}

func (bridge ChromeBridge) Restore(ctx context.Context, snapshot plan.UndoSnapshot) (plan.UndoResult, error) {
	var result plan.UndoResult
	err := bridge.Native.Call(ctx, OperationUndo, struct {
		Snapshot plan.UndoSnapshot `json:"snapshot"`
	}{snapshot}, &result)
	return result, err
}
