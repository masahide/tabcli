package plan

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/masahide/tabcli/internal/tools"
)

type ApplyStatus = tools.ApplyStatus
type UndoStatus = tools.UndoStatus
type AppliedOperation = tools.AppliedOperation
type RollbackResult = tools.RollbackResult
type ApplyResult = tools.ApplyResult
type UndoResult = tools.UndoResult

const (
	ApplyStatusSuccess    = tools.ApplyStatusSuccess
	ApplyStatusRolledBack = tools.ApplyStatusRolledBack
	ApplyStatusPartial    = tools.ApplyStatusPartial
	UndoStatusSuccess     = tools.UndoStatusSuccess
	UndoStatusPartial     = tools.UndoStatusPartial
)

type MutationBridge interface {
	tools.SnapshotBridge
	ApplyOperation(context.Context, Operation) (AppliedOperation, error)
	Rollback(context.Context, UndoSnapshot, []AppliedOperation) (RollbackResult, error)
	Restore(context.Context, UndoSnapshot) (UndoResult, error)
}

type ApplyManager struct {
	previews     *PreviewManager
	bridge       MutationBridge
	operationMu  sync.Mutex
	mu           sync.Mutex
	undoSnapshot *UndoSnapshot
}

func NewApplyManager(previews *PreviewManager, bridge MutationBridge) *ApplyManager {
	return &ApplyManager{previews: previews, bridge: bridge}
}

func (manager *ApplyManager) Apply(ctx context.Context, previewID string) (ApplyResult, error) {
	manager.operationMu.Lock()
	defer manager.operationMu.Unlock()
	record, err := manager.previews.Consume(previewID)
	if err != nil {
		return ApplyResult{}, err
	}
	current, err := manager.bridge.Snapshot(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	revision, err := Revision(current)
	if err != nil {
		return ApplyResult{}, err
	}
	if revision != record.Revision {
		return ApplyResult{}, tools.NewError(tools.CodePlanStale, "Chrome state changed after preview")
	}
	undo := captureUndoSnapshot(current, record.Operations)
	createdGroups := make(map[string]int)
	applied := make([]AppliedOperation, 0, len(record.Operations))
	for _, original := range record.Operations {
		operation := original
		if operation.Kind == OperationMoveTab && operation.NewGroupKey != "" {
			groupID, exists := createdGroups[operation.NewGroupKey]
			if !exists {
				return manager.rollbackFailure(ctx, undo, applied, errors.New("created group mapping is missing"))
			}
			operation.GroupID = groupID
		}
		result, applyErr := manager.bridge.ApplyOperation(ctx, operation)
		if applyErr != nil {
			return manager.rollbackFailure(ctx, undo, applied, applyErr)
		}
		if result.Operation.Kind == "" {
			result.Operation = operation
		}
		if operation.Kind == OperationCreateGroup {
			createdGroups[operation.NewGroupKey] = result.CreatedGroupID
		}
		applied = append(applied, result)
	}
	manager.mu.Lock()
	manager.undoSnapshot = &undo
	manager.mu.Unlock()
	return ApplyResult{ProtocolVersion: tools.ProtocolVersion, Status: ApplyStatusSuccess, AppliedOperations: applied, Recovery: "Call chrome_tab_groups_undo to restore this apply."}, nil
}

func (manager *ApplyManager) rollbackFailure(ctx context.Context, undo UndoSnapshot, applied []AppliedOperation, cause error) (ApplyResult, error) {
	reversed := slices.Clone(applied)
	slices.Reverse(reversed)
	rollback, rollbackErr := manager.bridge.Rollback(ctx, undo, reversed)
	result := ApplyResult{
		ProtocolVersion: tools.ProtocolVersion, AppliedOperations: applied, Rollback: &rollback,
	}
	if rollbackErr == nil && rollback.Complete {
		result.Status = ApplyStatusRolledBack
		result.Recovery = "No recovery action is required because the changes were rolled back."
		return result, &tools.Error{
			Code: tools.CodeApplyFailedRolledBack, Message: "apply failed and changes were rolled back",
			Details: map[string]any{"causeType": "operation", "appliedOperationCount": len(applied)},
		}
	}
	result.Status = ApplyStatusPartial
	result.Recovery = "Review rollback.unrestorable and restore the listed tabs or groups manually."
	return result, &tools.Error{
		Code: tools.CodeApplyPartial, Message: "apply failed and could not be fully rolled back",
		Details: map[string]any{"causeType": "operation", "appliedOperationCount": len(applied), "rollbackError": rollbackErr != nil, "cause": cause != nil},
	}
}

func (manager *ApplyManager) Undo(ctx context.Context) (UndoResult, error) {
	manager.operationMu.Lock()
	defer manager.operationMu.Unlock()
	manager.mu.Lock()
	if manager.undoSnapshot == nil {
		manager.mu.Unlock()
		return UndoResult{}, tools.NewError(tools.CodeUndoUnavailable, "no successful apply is available to undo")
	}
	snapshot := *manager.undoSnapshot
	manager.mu.Unlock()
	result, err := manager.bridge.Restore(ctx, snapshot)
	if err != nil {
		return UndoResult{}, err
	}
	result.ProtocolVersion = tools.ProtocolVersion
	manager.mu.Lock()
	manager.undoSnapshot = nil
	manager.mu.Unlock()
	return result, nil
}

func captureUndoSnapshot(snapshot tools.ChromeSnapshot, operations []Operation) UndoSnapshot {
	undo := UndoSnapshot{SessionID: snapshot.SessionID}
	tabIDs := make(map[int]bool)
	groupIDs := make(map[int]bool)
	for _, operation := range operations {
		if operation.TabID > 0 {
			tabIDs[operation.TabID] = true
		}
		if operation.GroupID >= 0 && (operation.Kind == OperationUpdateGroup || operation.Kind == OperationMoveTab && operation.NewGroupKey == "") {
			groupIDs[operation.GroupID] = true
		}
	}
	for _, tab := range snapshot.Tabs {
		if !tab.Incognito && tabIDs[tab.ID] {
			undo.Tabs = append(undo.Tabs, TabUndoState{TabID: tab.ID, WindowID: tab.WindowID, Index: tab.Index, Pinned: tab.Pinned, GroupID: tab.GroupID})
			if tab.GroupID >= 0 {
				groupIDs[tab.GroupID] = true
			}
		}
	}
	for _, group := range snapshot.Groups {
		if !group.Incognito && groupIDs[group.ID] {
			undo.Groups = append(undo.Groups, GroupUndoState{GroupID: group.ID, WindowID: group.WindowID, Title: group.Title, Color: group.Color, Collapsed: group.Collapsed})
		}
	}
	return undo
}
