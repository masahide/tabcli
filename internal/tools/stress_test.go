package tools

import (
	"context"
	"testing"
	"time"
)

func TestStressListTenThousandTabs(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snapshot := ChromeSnapshot{SessionID: "large", Tabs: make([]ChromeTab, 10_000)}
	for index := range snapshot.Tabs {
		last := now.Add(-time.Duration(index) * time.Second).UnixMilli()
		snapshot.Tabs[index] = ChromeTab{ID: index + 1, WindowID: index/500 + 1, Index: index % 500, GroupID: -1, LastAccessed: &last, Operable: true}
	}
	result, err := ListTabsAt(context.Background(), benchmarkSnapshotBridge{snapshot: snapshot}, TabsListInput{SortBy: SortLastAccessed, SortOrder: SortAscending}, now)
	if err != nil || len(result.Tabs) != len(snapshot.Tabs) {
		t.Fatalf("tabs=%d err=%v", len(result.Tabs), err)
	}
}
