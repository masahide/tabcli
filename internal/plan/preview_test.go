package plan

import (
	"errors"
	"reflect"
	"testing"

	"github.com/masahide/tabcli/internal/tools"
)

func previewSnapshot() tools.ChromeSnapshot {
	return tools.ChromeSnapshot{
		SessionID: "session-1",
		Tabs: []tools.ChromeTab{
			{ID: 1, WindowID: 10, Index: 0, GroupID: 100, Pinned: true, Operable: true},
			{ID: 2, WindowID: 10, Index: 1, GroupID: -1, Operable: true},
		},
		Groups: []tools.ChromeGroup{{ID: 100, WindowID: 10, Title: "Work", Color: "blue", TabIDs: []int{1}}},
	}
}

func TestNewLogicalGroupsSplitByWindowAndExistingGroupsRejectCrossWindow(t *testing.T) {
	snapshot := previewSnapshot()
	snapshot.Tabs = append(snapshot.Tabs, tools.ChromeTab{ID: 3, WindowID: 20, Index: 0, GroupID: -1, Operable: true})
	operations, err := BuildOperations(ClassificationPlan{
		Policy: PolicyRebuildSelected,
		Assignments: []Assignment{
			{TabID: 2, Destination: Destination{Title: "Research", Color: "cyan"}},
			{TabID: 3, Destination: Destination{Title: "Research", Color: "cyan"}},
		},
	}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	windows := []int{}
	for _, operation := range operations {
		if operation.Kind == OperationCreateGroup {
			windows = append(windows, operation.WindowID)
		}
	}
	if !reflect.DeepEqual(windows, []int{10, 20}) {
		t.Fatalf("created group windows = %v, want [10 20]", windows)
	}

	_, err = BuildOperations(ClassificationPlan{
		Policy:      PolicyExistingGroupsOnly,
		Assignments: []Assignment{{TabID: 3, Destination: Destination{GroupID: intPointer(100)}}},
	}, snapshot)
	var toolError *tools.Error
	if !errors.As(err, &toolError) || toolError.Code != tools.CodeCrossWindowGroup {
		t.Fatalf("cross-window error = %v", err)
	}
}

func TestBuildOperationsDiffs(t *testing.T) {
	tests := []struct {
		name string
		plan ClassificationPlan
		want []OperationKind
	}{
		{name: "no change", plan: ClassificationPlan{Policy: PolicyPreserveExisting, Assignments: []Assignment{{TabID: 1, Destination: Destination{GroupID: intPointer(100)}}}}, want: []OperationKind{}},
		{name: "create and move", plan: ClassificationPlan{Policy: PolicyRebuildSelected, Assignments: []Assignment{{TabID: 2, Destination: Destination{Title: "New", Color: "green"}}}}, want: []OperationKind{OperationCreateGroup, OperationMoveTab}},
		{name: "update group", plan: ClassificationPlan{Policy: PolicyPreserveExisting, Assignments: []Assignment{{TabID: 1, Destination: Destination{GroupID: intPointer(100), Title: "Renamed", Color: "red"}}}}, want: []OperationKind{OperationUpdateGroup}},
		{name: "move existing", plan: ClassificationPlan{Policy: PolicyExistingGroupsOnly, Assignments: []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(100)}}}}, want: []OperationKind{OperationMoveTab}},
		{name: "ungroup", plan: ClassificationPlan{Policy: PolicyRebuildSelected, Assignments: []Assignment{{TabID: 1, Destination: Destination{Ungroup: true}}}}, want: []OperationKind{OperationUngroupTab}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operations, err := BuildOperations(tt.plan, previewSnapshot())
			if err != nil {
				t.Fatal(err)
			}
			got := make([]OperationKind, len(operations))
			for i, operation := range operations {
				got[i] = operation.Kind
				if operation.Pinned != nil {
					t.Fatal("preview must not change pinned state unless explicitly requested")
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("operation kinds = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreviewPolicySemantics(t *testing.T) {
	_, err := BuildOperations(ClassificationPlan{
		Policy:      PolicyExistingGroupsOnly,
		Assignments: []Assignment{{TabID: 2, Destination: Destination{Title: "Forbidden", Color: "blue"}}},
	}, previewSnapshot())
	if err == nil {
		t.Fatal("existing_groups_only accepted a new group")
	}

	operations, err := BuildOperations(ClassificationPlan{
		Policy:      PolicyPreserveExisting,
		Assignments: []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(100)}}},
	}, previewSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || operations[0].TabID != 2 {
		t.Fatalf("preserve_existing altered unmentioned tab: %#v", operations)
	}
}

func TestBuildOperationsMergesDuplicateGroupCreationAndNoOps(t *testing.T) {
	operations, err := BuildOperations(ClassificationPlan{
		Policy: PolicyRebuildSelected,
		Assignments: []Assignment{
			{TabID: 1, Destination: Destination{GroupID: intPointer(100)}},
			{TabID: 2, Destination: Destination{Title: "Shared", Color: "green"}},
		},
	}, previewSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	createCount := 0
	for _, operation := range operations {
		if operation.Kind == OperationCreateGroup {
			createCount++
		}
		if operation.TabID == 1 {
			t.Fatalf("no-op for tab 1 was retained: %#v", operation)
		}
	}
	if createCount != 1 {
		t.Fatalf("create operations = %d, want 1", createCount)
	}
}

func TestPinnedPositionIsPreservedUnlessExplicitlyRequested(t *testing.T) {
	operations, err := BuildOperations(ClassificationPlan{
		Policy:      PolicyPreserveExisting,
		Assignments: []Assignment{{TabID: 1, Destination: Destination{GroupID: intPointer(100)}}},
	}, previewSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 0 {
		t.Fatalf("implicit plan changed pinned tab: %#v", operations)
	}

	pinned := false
	index := 2
	operations, err = BuildOperations(ClassificationPlan{
		Policy: PolicyPreserveExisting,
		Assignments: []Assignment{{
			TabID: 1, Destination: Destination{GroupID: intPointer(100)}, Pinned: &pinned, Index: &index,
		}},
	}, previewSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || operations[0].Pinned == nil || *operations[0].Pinned || operations[0].Index == nil || *operations[0].Index != 2 {
		t.Fatalf("explicit pinned/index operation = %#v", operations)
	}
}
