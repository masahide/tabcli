package tools

import (
	"context"
	"sort"
	"time"
)

type SnapshotBridge interface {
	Snapshot(context.Context) (ChromeSnapshot, error)
}

func ListTabs(ctx context.Context, bridge SnapshotBridge, input TabsListInput) (TabsListResult, error) {
	return ListTabsAt(ctx, bridge, input, time.Now())
}

func ListTabsAt(ctx context.Context, bridge SnapshotBridge, input TabsListInput, now time.Time) (TabsListResult, error) {
	if input.Ungrouped && input.GroupID != nil {
		return TabsListResult{}, NewError(CodeInvalidArgument, "groupId and ungrouped cannot be combined")
	}
	if input.InactiveForSeconds != nil && *input.InactiveForSeconds < 0 {
		return TabsListResult{}, NewError(CodeInvalidArgument, "inactiveForSeconds must be non-negative")
	}
	snapshot, err := bridge.Snapshot(ctx)
	if err != nil {
		return TabsListResult{}, err
	}
	result := TabsListResult{ProtocolVersion: ProtocolVersion, Tabs: make([]Tab, 0, len(snapshot.Tabs))}
	for _, source := range snapshot.Tabs {
		if source.Incognito || input.WindowID != nil && source.WindowID != *input.WindowID || input.GroupID != nil && source.GroupID != *input.GroupID || input.Ungrouped && source.GroupID >= 0 {
			continue
		}
		var inactiveSeconds *int64
		if source.LastAccessed != nil {
			seconds := (now.UnixMilli() - *source.LastAccessed) / 1000
			if seconds < 0 {
				seconds = 0
			}
			inactiveSeconds = &seconds
		}
		if input.InactiveForSeconds != nil && (inactiveSeconds == nil || *inactiveSeconds < *input.InactiveForSeconds) {
			continue
		}
		tab := Tab{
			ID: source.ID, Title: source.Title, URL: source.URL, WindowID: source.WindowID,
			Index: source.Index, GroupID: source.GroupID, Active: source.Active, Pinned: source.Pinned,
			LastAccessed: source.LastAccessed, Operable: source.Operable, InactiveDurationSeconds: inactiveSeconds,
		}
		tab.Activity = source.Activity
		result.Tabs = append(result.Tabs, tab)
	}
	if err := sortTabs(result.Tabs, input.SortBy, input.SortOrder); err != nil {
		return TabsListResult{}, err
	}
	if !input.IncludeActivity {
		for index := range result.Tabs {
			result.Tabs[index].Activity = nil
		}
	}
	return result, nil
}

func sortTabs(tabs []Tab, field TabsSort, order SortOrder) error {
	if field == "" {
		return nil
	}
	if field != SortPosition && field != SortLastAccessed && field != SortInactiveDuration && field != SortCreatedAt {
		return NewError(CodeInvalidArgument, "unsupported tab sort")
	}
	if order == "" {
		order = SortAscending
	}
	if order != SortAscending && order != SortDescending {
		return NewError(CodeInvalidArgument, "sortOrder must be asc or desc")
	}
	sort.SliceStable(tabs, func(i, j int) bool {
		if field == SortPosition {
			less := tabs[i].WindowID < tabs[j].WindowID || tabs[i].WindowID == tabs[j].WindowID && tabs[i].Index < tabs[j].Index
			if order == SortDescending {
				return !less && (tabs[i].WindowID != tabs[j].WindowID || tabs[i].Index != tabs[j].Index)
			}
			return less
		}
		left, right := tabSortValue(tabs[i], field), tabSortValue(tabs[j], field)
		if left == nil && right == nil {
			return tabs[i].ID < tabs[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if *left == *right {
			return tabs[i].ID < tabs[j].ID
		}
		descending := order == SortDescending
		if field == SortInactiveDuration {
			descending = !descending
		}
		if descending {
			return *left > *right
		}
		return *left < *right
	})
	return nil
}

func tabSortValue(tab Tab, field TabsSort) *int64 {
	if field == SortLastAccessed || field == SortInactiveDuration {
		return tab.LastAccessed
	}
	if tab.Activity == nil {
		return nil
	}
	return tab.Activity.CreatedAt
}

func ListGroups(ctx context.Context, bridge SnapshotBridge, input GroupsListInput) (GroupsListResult, error) {
	snapshot, err := bridge.Snapshot(ctx)
	if err != nil {
		return GroupsListResult{}, err
	}
	result := GroupsListResult{ProtocolVersion: ProtocolVersion, Groups: make([]Group, 0, len(snapshot.Groups))}
	for _, source := range snapshot.Groups {
		if source.Incognito || input.WindowID != nil && source.WindowID != *input.WindowID {
			continue
		}
		result.Groups = append(result.Groups, Group{
			ID: source.ID, Title: source.Title, Color: source.Color, Collapsed: source.Collapsed,
			WindowID: source.WindowID, TabIDs: append([]int(nil), source.TabIDs...),
		})
	}
	return result, nil
}
