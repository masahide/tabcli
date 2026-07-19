package tools

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestTabsListInactiveFilterAndSorts(t *testing.T) {
	createdOld := int64(1_000)
	createdNew := int64(3_000)
	lastOld := int64(1_700_000_000_000)
	lastNew := int64(1_700_086_300_000)
	bridge := fixtureBridge{snapshot: ChromeSnapshot{Tabs: []ChromeTab{
		{ID: 1, LastAccessed: &lastOld, Activity: &ActivityMetadata{CreatedAt: &createdNew}},
		{ID: 2, LastAccessed: &lastNew, Activity: &ActivityMetadata{CreatedAt: &createdOld}},
		{ID: 3, LastAccessed: nil, Activity: &ActivityMetadata{CreatedAt: nil}},
	}}}
	now := time.UnixMilli(1_700_086_400_000)

	inactive := int64(86_400)
	filtered, err := ListTabsAt(context.Background(), bridge, TabsListInput{InactiveForSeconds: &inactive}, now)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := tabIDs(filtered.Tabs), []int{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("inactive IDs = %v, want %v", got, want)
	}
	if filtered.Tabs[0].InactiveDurationSeconds == nil || *filtered.Tabs[0].InactiveDurationSeconds != 86_400 {
		t.Fatalf("inactive duration = %v", filtered.Tabs[0].InactiveDurationSeconds)
	}

	tests := []struct {
		sort  TabsSort
		order SortOrder
		want  []int
	}{
		{SortLastAccessed, SortAscending, []int{1, 2, 3}},
		{SortLastAccessed, SortDescending, []int{2, 1, 3}},
		{SortInactiveDuration, SortAscending, []int{2, 1, 3}},
		{SortCreatedAt, SortAscending, []int{2, 1, 3}},
		{SortCreatedAt, SortDescending, []int{1, 2, 3}},
	}
	for _, tt := range tests {
		t.Run(string(tt.sort)+"_"+string(tt.order), func(t *testing.T) {
			result, err := ListTabsAt(context.Background(), bridge, TabsListInput{SortBy: tt.sort, SortOrder: tt.order, IncludeActivity: true}, now)
			if err != nil {
				t.Fatal(err)
			}
			if got := tabIDs(result.Tabs); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("IDs = %v, want %v", got, tt.want)
			}
			if result.Tabs[0].Activity == nil {
				t.Fatal("includeActivity=true must include metadata")
			}
		})
	}
}

func TestTabsListUngroupedAndCreatedSortWithoutActivityOutput(t *testing.T) {
	createdOld, createdNew := int64(1_000), int64(2_000)
	bridge := fixtureBridge{snapshot: ChromeSnapshot{Tabs: []ChromeTab{
		{ID: 1, GroupID: 10, Activity: &ActivityMetadata{CreatedAt: &createdOld}},
		{ID: 2, GroupID: -1, Activity: &ActivityMetadata{CreatedAt: &createdNew}},
		{ID: 3, GroupID: -1, Activity: &ActivityMetadata{CreatedAt: &createdOld}},
	}}}
	result, err := ListTabs(context.Background(), bridge, TabsListInput{Ungrouped: true, SortBy: SortCreatedAt, SortOrder: SortAscending})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := tabIDs(result.Tabs), []int{3, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("IDs = %v, want %v", got, want)
	}
	if result.Tabs[0].Activity != nil {
		t.Fatal("includeActivity=false leaked activity metadata")
	}
}
