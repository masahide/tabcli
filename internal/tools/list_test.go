package tools

import (
	"context"
	"reflect"
	"testing"
)

type fixtureBridge struct {
	snapshot ChromeSnapshot
}

func (b fixtureBridge) Snapshot(context.Context) (ChromeSnapshot, error) {
	return b.snapshot, nil
}

func TestListTabsAndGroupsFromChromeSnapshot(t *testing.T) {
	bridge := fixtureBridge{snapshot: ChromeSnapshot{
		Tabs: []ChromeTab{
			{ID: 1, WindowID: 10, Index: 0, GroupID: 100, Title: "Docs", URL: "https://example.com/docs", Active: true, Operable: true},
			{ID: 2, WindowID: 10, Index: 1, GroupID: -1, Title: "Settings", URL: "chrome://settings", Operable: false},
			{ID: 3, WindowID: 20, Index: 0, GroupID: -1, Title: "Private", URL: "https://private.example", Incognito: true, Operable: true},
		},
		Groups: []ChromeGroup{
			{ID: 100, WindowID: 10, Title: "Work", Color: "blue", TabIDs: []int{1}},
			{ID: 200, WindowID: 20, Title: "Secret", Color: "red", Incognito: true, TabIDs: []int{3}},
		},
	}}

	tabs, err := ListTabs(context.Background(), bridge, TabsListInput{})
	if err != nil {
		t.Fatalf("ListTabs() error = %v", err)
	}
	if got, want := tabIDs(tabs.Tabs), []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tab IDs = %v, want %v", got, want)
	}
	if tabs.Tabs[1].Operable {
		t.Fatal("chrome:// tab must be marked inoperable")
	}

	groups, err := ListGroups(context.Background(), bridge, GroupsListInput{})
	if err != nil {
		t.Fatalf("ListGroups() error = %v", err)
	}
	if len(groups.Groups) != 1 || groups.Groups[0].ID != 100 {
		t.Fatalf("groups = %#v, want only group 100", groups.Groups)
	}
}

func tabIDs(tabs []Tab) []int {
	ids := make([]int, len(tabs))
	for i, tab := range tabs {
		ids[i] = tab.ID
	}
	return ids
}
