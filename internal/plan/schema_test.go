package plan

import (
	"errors"
	"testing"

	"github.com/masahide/tabcli/internal/tools"
)

func schemaSnapshot() tools.ChromeSnapshot {
	return tools.ChromeSnapshot{
		SessionID: "session-1",
		Tabs: []tools.ChromeTab{
			{ID: 1, WindowID: 10, GroupID: 100, Operable: true},
			{ID: 2, WindowID: 10, GroupID: -1, Operable: true},
		},
		Groups: []tools.ChromeGroup{{ID: 100, WindowID: 10, Title: "Existing", Color: "blue", TabIDs: []int{1}}},
	}
}

func TestValidateContentRevisions(t *testing.T) {
	plan := ClassificationPlan{
		Policy:           PolicyPreserveExisting,
		Assignments:      []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(100)}}},
		ContentRevisions: []ContentReference{{TabID: 2, Revision: "revision-1"}},
	}
	if err := ValidateContentRevisions(plan, func(tabID int, revision string) bool {
		return tabID == 2 && revision == "revision-1"
	}); err != nil {
		t.Fatalf("valid revision rejected: %v", err)
	}
	err := ValidateContentRevisions(plan, func(int, string) bool { return false })
	var toolError *tools.Error
	if !errors.As(err, &toolError) || toolError.Code != tools.CodeContentStale {
		t.Fatalf("invalid revision error = %v, want CONTENT_STALE", err)
	}
}

func TestValidatePlanSupportsAllPoliciesAndDestinations(t *testing.T) {
	tests := []ClassificationPlan{
		{Policy: PolicyUngroupedOnly, Assignments: []Assignment{{TabID: 2, Destination: Destination{Title: "New", Color: "green"}}}},
		{Policy: PolicyPreserveExisting, Assignments: []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(100)}}}},
		{Policy: PolicyExistingGroupsOnly, Assignments: []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(100)}}}},
		{Policy: PolicyRebuildSelected, Assignments: []Assignment{{TabID: 1, Destination: Destination{Ungroup: true}}}},
	}
	for _, plan := range tests {
		t.Run(string(plan.Policy), func(t *testing.T) {
			if err := ValidatePlan(plan, schemaSnapshot()); err != nil {
				t.Fatalf("ValidatePlan() error = %v", err)
			}
		})
	}
}

func TestValidatePlanRejectsInvalidSchemaAndReferences(t *testing.T) {
	tests := []struct {
		name string
		plan ClassificationPlan
	}{
		{name: "duplicate tab ID", plan: ClassificationPlan{Policy: PolicyRebuildSelected, Assignments: []Assignment{{TabID: 1, Destination: Destination{Ungroup: true}}, {TabID: 1, Destination: Destination{Ungroup: true}}}}},
		{name: "unknown tab ID", plan: ClassificationPlan{Policy: PolicyRebuildSelected, Assignments: []Assignment{{TabID: 99, Destination: Destination{Ungroup: true}}}}},
		{name: "unknown group ID", plan: ClassificationPlan{Policy: PolicyExistingGroupsOnly, Assignments: []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(999)}}}}},
		{name: "invalid color", plan: ClassificationPlan{Policy: PolicyRebuildSelected, Assignments: []Assignment{{TabID: 2, Destination: Destination{Title: "New", Color: "chartreuse"}}}}},
		{name: "ambiguous destination", plan: ClassificationPlan{Policy: PolicyRebuildSelected, Assignments: []Assignment{{TabID: 2, Destination: Destination{Ungroup: true, GroupID: intPointer(100)}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePlan(tt.plan, schemaSnapshot()); err == nil {
				t.Fatalf("ValidatePlan(%#v) succeeded", tt.plan)
			}
		})
	}
}

func intPointer(value int) *int { return &value }
