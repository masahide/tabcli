package plan

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/masahide/tabcli/internal/tools"
)

type blockingMutationBridge struct {
	snapshot tools.ChromeSnapshot
	entered  chan struct{}
	release  chan struct{}
	once     sync.Once
}

func (bridge *blockingMutationBridge) Snapshot(context.Context) (tools.ChromeSnapshot, error) {
	return bridge.snapshot, nil
}
func (bridge *blockingMutationBridge) ApplyOperation(_ context.Context, operation Operation) (AppliedOperation, error) {
	bridge.entered <- struct{}{}
	bridge.once.Do(func() { <-bridge.release })
	return AppliedOperation{Operation: operation}, nil
}
func (*blockingMutationBridge) Rollback(context.Context, UndoSnapshot, []AppliedOperation) (RollbackResult, error) {
	return RollbackResult{Complete: true}, nil
}
func (*blockingMutationBridge) Restore(context.Context, UndoSnapshot) (UndoResult, error) {
	return UndoResult{Status: UndoStatusSuccess}, nil
}

func TestStressConcurrentApplyIsSerialized(t *testing.T) {
	bridge := &blockingMutationBridge{
		snapshot: tools.ChromeSnapshot{SessionID: "stress", Tabs: []tools.ChromeTab{{ID: 1, WindowID: 1, GroupID: -1, Operable: true}, {ID: 2, WindowID: 1, Index: 1, GroupID: -1, Operable: true}}, Groups: []tools.ChromeGroup{{ID: 100, WindowID: 1, Title: "Target", Color: "blue"}}},
		entered:  make(chan struct{}, 2), release: make(chan struct{}),
	}
	ids := []string{"first", "second"}
	previews := NewPreviewManager(bridge, PreviewOptions{RandomID: func() string { id := ids[0]; ids = ids[1:]; return id }, ContentRevisionValid: func(int, string) bool { return true }})
	for _, tabID := range []int{1, 2} {
		if _, err := previews.Preview(context.Background(), ClassificationPlan{Policy: PolicyExistingGroupsOnly, Assignments: []Assignment{{TabID: tabID, Destination: Destination{GroupID: intPointer(100)}}}}); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewApplyManager(previews, bridge)
	errors := make(chan error, 2)
	go func() { _, err := manager.Apply(context.Background(), "first"); errors <- err }()
	<-bridge.entered
	go func() { _, err := manager.Apply(context.Background(), "second"); errors <- err }()
	select {
	case <-bridge.entered:
		t.Fatal("concurrent apply operations overlapped")
	case <-time.After(20 * time.Millisecond):
	}
	close(bridge.release)
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
}

func TestStressApplyOneThousandOperations(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	bridge := &mutationBridgeFixture{snapshot: tools.ChromeSnapshot{SessionID: "stress", Groups: []tools.ChromeGroup{{ID: 100, WindowID: 1, Title: "Target", Color: "blue"}}}, rollbackResult: RollbackResult{Complete: true}}
	assignments := make([]Assignment, 1000)
	for index := range assignments {
		bridge.snapshot.Tabs = append(bridge.snapshot.Tabs, tools.ChromeTab{ID: index + 1, WindowID: 1, Index: index, GroupID: -1, Operable: true})
		assignments[index] = Assignment{TabID: index + 1, Destination: Destination{GroupID: intPointer(100)}}
	}
	previews := NewPreviewManager(bridge, PreviewOptions{Now: func() time.Time { return now }, RandomID: func() string { return "large" }, ContentRevisionValid: func(int, string) bool { return true }})
	if _, err := previews.Preview(context.Background(), ClassificationPlan{Policy: PolicyExistingGroupsOnly, Assignments: assignments}); err != nil {
		t.Fatal(err)
	}
	manager := NewApplyManager(previews, bridge)
	if _, err := manager.Apply(context.Background(), "large"); err != nil {
		t.Fatal(err)
	}
	if len(bridge.applied) != 1000 {
		t.Fatalf("applied=%d", len(bridge.applied))
	}
}
