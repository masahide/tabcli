package tools

import "time"

type PlanPolicy string

const (
	PlanPolicyUngroupedOnly      PlanPolicy = "ungrouped_only"
	PlanPolicyPreserveExisting   PlanPolicy = "preserve_existing"
	PlanPolicyExistingGroupsOnly PlanPolicy = "existing_groups_only"
	PlanPolicyRebuildSelected    PlanPolicy = "rebuild_selected"
)

type PlanDestination struct {
	Ungroup bool   `json:"ungroup,omitempty"`
	GroupID *int   `json:"groupId,omitempty"`
	Title   string `json:"title,omitempty"`
	Color   string `json:"color,omitempty"`
}

type PlanAssignment struct {
	TabID       int             `json:"tabId"`
	Destination PlanDestination `json:"destination"`
	Pinned      *bool           `json:"pinned,omitempty"`
	Index       *int            `json:"index,omitempty"`
}

type ContentReference struct {
	TabID    int    `json:"tabId"`
	Revision string `json:"revision"`
}

type ClassificationPlan struct {
	Policy           PlanPolicy         `json:"policy"`
	Assignments      []PlanAssignment   `json:"assignments"`
	ContentRevisions []ContentReference `json:"contentRevisions,omitempty"`
}

type PlanOperationKind string

const (
	PlanOperationCreateGroup PlanOperationKind = "create_group"
	PlanOperationUpdateGroup PlanOperationKind = "update_group"
	PlanOperationMoveTab     PlanOperationKind = "move_tab"
	PlanOperationUngroupTab  PlanOperationKind = "ungroup_tab"
)

type PlanOperation struct {
	Kind        PlanOperationKind `json:"kind"`
	TabID       int               `json:"tabId,omitempty"`
	GroupID     int               `json:"groupId,omitempty"`
	NewGroupKey string            `json:"newGroupKey,omitempty"`
	WindowID    int               `json:"windowId,omitempty"`
	Title       string            `json:"title,omitempty"`
	Color       string            `json:"color,omitempty"`
	Collapsed   *bool             `json:"collapsed,omitempty"`
	Pinned      *bool             `json:"pinned,omitempty"`
	Index       *int              `json:"index,omitempty"`
}

type PreviewInput struct {
	Plan ClassificationPlan `json:"plan"`
}

type PreviewResult struct {
	ProtocolVersion int                `json:"protocolVersion"`
	PreviewID       string             `json:"previewId"`
	Revision        string             `json:"revision"`
	ExpiresAt       time.Time          `json:"expiresAt"`
	NormalizedPlan  ClassificationPlan `json:"normalizedPlan"`
	Operations      []PlanOperation    `json:"operations"`
}

type ApplyStatus string

const (
	ApplyStatusSuccess    ApplyStatus = "success"
	ApplyStatusRolledBack ApplyStatus = "rolled_back"
	ApplyStatusPartial    ApplyStatus = "partial"
)

type UndoStatus string

const (
	UndoStatusSuccess UndoStatus = "success"
	UndoStatusPartial UndoStatus = "partial"
)

type AppliedOperation struct {
	Operation      PlanOperation `json:"operation"`
	CreatedGroupID int           `json:"createdGroupId,omitempty"`
}

type RollbackResult struct {
	Complete     bool     `json:"complete"`
	Unrestorable []string `json:"unrestorable,omitempty"`
}

type ApplyInput struct {
	PreviewID string `json:"previewId"`
}

type ApplyResult struct {
	ProtocolVersion   int                `json:"protocolVersion"`
	Status            ApplyStatus        `json:"status"`
	AppliedOperations []AppliedOperation `json:"appliedOperations"`
	Rollback          *RollbackResult    `json:"rollback,omitempty"`
	Recovery          string             `json:"recovery"`
}

type UndoInput struct{}

type UndoResult struct {
	ProtocolVersion  int        `json:"protocolVersion"`
	Status           UndoStatus `json:"status"`
	RestoredTabIDs   []int      `json:"restoredTabIds,omitempty"`
	RestoredGroupIDs []int      `json:"restoredGroupIds,omitempty"`
	Unrestorable     []string   `json:"unrestorable,omitempty"`
}

const ClassificationPlanJSONSchema = `{
  "$schema":"https://json-schema.org/draft/2020-12/schema",
  "type":"object",
  "additionalProperties":false,
  "required":["policy","assignments"],
  "properties":{
    "policy":{"enum":["ungrouped_only","preserve_existing","existing_groups_only","rebuild_selected"]},
    "assignments":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["tabId","destination"],"properties":{"tabId":{"type":"integer","minimum":1},"destination":{"type":"object"}}}},
    "contentRevisions":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["tabId","revision"],"properties":{"tabId":{"type":"integer","minimum":1},"revision":{"type":"string","minLength":1}}}}
  }
}`
