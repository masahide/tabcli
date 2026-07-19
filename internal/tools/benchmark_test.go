package tools

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type benchmarkSnapshotBridge struct{ snapshot ChromeSnapshot }

func (bridge benchmarkSnapshotBridge) Snapshot(context.Context) (ChromeSnapshot, error) {
	return bridge.snapshot, nil
}

func BenchmarkTabsList500(b *testing.B) {
	now := time.Unix(1_700_000_000, 0)
	snapshot := ChromeSnapshot{SessionID: "benchmark", Tabs: make([]ChromeTab, 500)}
	for index := range snapshot.Tabs {
		lastAccessed := now.Add(-time.Duration(index) * time.Minute).UnixMilli()
		snapshot.Tabs[index] = ChromeTab{ID: index + 1, WindowID: index/100 + 1, Index: index % 100, GroupID: -1, Title: fmt.Sprintf("Tab %03d", index), URL: fmt.Sprintf("https://example.com/%d", index), LastAccessed: &lastAccessed, Operable: true}
	}
	bridge := benchmarkSnapshotBridge{snapshot: snapshot}
	b.ReportAllocs()
	for range b.N {
		result, err := ListTabsAt(context.Background(), bridge, TabsListInput{SortBy: SortLastAccessed, SortOrder: SortAscending, IncludeActivity: true}, now)
		if err != nil || len(result.Tabs) != 500 {
			b.Fatalf("result=%d err=%v", len(result.Tabs), err)
		}
	}
}
