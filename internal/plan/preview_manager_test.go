package plan

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/masahide/tabcli/internal/tools"
)

type previewBridgeFixture struct {
	snapshotCalls int
	mutationCalls int
}

func (bridge *previewBridgeFixture) Snapshot(context.Context) (tools.ChromeSnapshot, error) {
	bridge.snapshotCalls++
	return previewSnapshot(), nil
}

func (bridge *previewBridgeFixture) MutateForTest() {
	bridge.mutationCalls++
}

func TestPreviewNeverMutatesChrome(t *testing.T) {
	bridge := &previewBridgeFixture{}
	manager := NewPreviewManager(bridge, PreviewOptions{
		Now:                  func() time.Time { return time.Unix(1_700_000_000, 0) },
		RandomID:             func() string { return "preview-1" },
		ContentRevisionValid: func(int, string) bool { return true },
	})
	result, err := manager.Preview(context.Background(), ClassificationPlan{
		Policy:      PolicyExistingGroupsOnly,
		Assignments: []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(100)}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bridge.snapshotCalls != 1 || bridge.mutationCalls != 0 {
		t.Fatalf("snapshot calls = %d, mutation calls = %d", bridge.snapshotCalls, bridge.mutationCalls)
	}
	if result.PreviewID != "preview-1" || len(result.Operations) != 1 {
		t.Fatalf("preview = %#v", result)
	}
}

func TestPreviewExpiryAndDeterministicRevision(t *testing.T) {
	snapshot := previewSnapshot()
	reordered := snapshot
	reordered.Tabs = slices.Clone(snapshot.Tabs)
	slices.Reverse(reordered.Tabs)
	reordered.Groups = slices.Clone(snapshot.Groups)
	slices.Reverse(reordered.Groups)
	first, err := Revision(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Revision(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("revision is order dependent: %q != %q", first, second)
	}
	changed := snapshot
	changed.Groups = slices.Clone(snapshot.Groups)
	changed.Groups[0].Color = "red"
	changedRevision, err := Revision(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedRevision == first {
		t.Fatal("relevant group change did not change revision")
	}

	now := time.Unix(1_700_000_000, 0)
	manager := NewPreviewManager(&previewBridgeFixture{}, PreviewOptions{
		Now:                  func() time.Time { return now },
		RandomID:             func() string { return "preview-expiring" },
		ContentRevisionValid: func(int, string) bool { return true },
	})
	_, err = manager.Preview(context.Background(), ClassificationPlan{
		Policy:      PolicyExistingGroupsOnly,
		Assignments: []Assignment{{TabID: 2, Destination: Destination{GroupID: intPointer(100)}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(5 * time.Minute)
	_, err = manager.Lookup("preview-expiring")
	var toolError *tools.Error
	if !errors.As(err, &toolError) || toolError.Code != tools.CodePreviewExpired {
		t.Fatalf("expired lookup error = %v", err)
	}
}
