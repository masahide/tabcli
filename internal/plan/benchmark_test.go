package plan

import (
	"context"
	"testing"
	"time"

	"github.com/masahide/tabcli/internal/tools"
)

func BenchmarkApply200Tabs(b *testing.B) {
	snapshot := tools.ChromeSnapshot{SessionID: "benchmark", Groups: []tools.ChromeGroup{{ID: 100, WindowID: 1, Title: "Target", Color: "blue"}}}
	assignments := make([]Assignment, 200)
	for index := range assignments {
		snapshot.Tabs = append(snapshot.Tabs, tools.ChromeTab{ID: index + 1, WindowID: 1, Index: index, GroupID: -1, Operable: true})
		assignments[index] = Assignment{TabID: index + 1, Destination: Destination{GroupID: intPointer(100)}}
	}
	classification := ClassificationPlan{Policy: PolicyExistingGroupsOnly, Assignments: assignments}
	b.ReportAllocs()
	for range b.N {
		now := time.Unix(1_700_000_000, 0)
		bridge := &mutationBridgeFixture{snapshot: snapshot, rollbackResult: RollbackResult{Complete: true}}
		previews := NewPreviewManager(bridge, PreviewOptions{Now: func() time.Time { return now }, RandomID: func() string { return "benchmark" }, ContentRevisionValid: func(int, string) bool { return true }})
		if _, err := previews.Preview(context.Background(), classification); err != nil {
			b.Fatal(err)
		}
		manager := NewApplyManager(previews, bridge)
		if _, err := manager.Apply(context.Background(), "benchmark"); err != nil {
			b.Fatal(err)
		}
		if len(bridge.applied) != 200 {
			b.Fatalf("applied=%d", len(bridge.applied))
		}
	}
}
