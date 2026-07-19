package plan

import (
	"fmt"
	"slices"
	"sort"

	"github.com/masahide/tabcli/internal/tools"
)

type Operation = tools.PlanOperation
type OperationKind = tools.PlanOperationKind

const (
	OperationCreateGroup = tools.PlanOperationCreateGroup
	OperationUpdateGroup = tools.PlanOperationUpdateGroup
	OperationMoveTab     = tools.PlanOperationMoveTab
	OperationUngroupTab  = tools.PlanOperationUngroupTab
)

func BuildOperations(plan ClassificationPlan, snapshot tools.ChromeSnapshot) ([]Operation, error) {
	if err := ValidatePlan(plan, snapshot); err != nil {
		return nil, err
	}
	tabs := make(map[int]tools.ChromeTab, len(snapshot.Tabs))
	for _, tab := range snapshot.Tabs {
		tabs[tab.ID] = tab
	}
	groups := make(map[int]tools.ChromeGroup, len(snapshot.Groups))
	for _, group := range snapshot.Groups {
		groups[group.ID] = group
	}
	assignments := slices.Clone(plan.Assignments)
	sort.Slice(assignments, func(i, j int) bool { return assignments[i].TabID < assignments[j].TabID })
	creates := make(map[string]Operation)
	updates := make(map[int]Operation)
	var tabOperations []Operation
	for _, assignment := range assignments {
		tab := tabs[assignment.TabID]
		if plan.Policy == PolicyUngroupedOnly && tab.GroupID >= 0 {
			continue
		}
		destination := assignment.Destination
		if destination.Ungroup {
			if tab.GroupID >= 0 {
				tabOperations = append(tabOperations, Operation{Kind: OperationUngroupTab, TabID: tab.ID, WindowID: tab.WindowID, Pinned: assignment.Pinned, Index: assignment.Index})
			} else if assignment.Pinned != nil || assignment.Index != nil {
				tabOperations = append(tabOperations, Operation{Kind: OperationMoveTab, TabID: tab.ID, WindowID: tab.WindowID, Pinned: assignment.Pinned, Index: assignment.Index})
			}
			continue
		}
		if destination.GroupID != nil {
			group := groups[*destination.GroupID]
			if group.WindowID != tab.WindowID {
				return nil, tools.NewError(tools.CodeCrossWindowGroup, "tab and target group are in different windows")
			}
			if destination.Title != "" || destination.Color != "" {
				update := Operation{Kind: OperationUpdateGroup, GroupID: group.ID, WindowID: group.WindowID}
				if destination.Title != "" && destination.Title != group.Title {
					update.Title = destination.Title
				}
				if destination.Color != "" && destination.Color != group.Color {
					update.Color = destination.Color
				}
				if update.Title != "" || update.Color != "" {
					if previous, exists := updates[group.ID]; exists && (previous.Title != update.Title || previous.Color != update.Color) {
						return nil, tools.NewError(tools.CodePlanInvalid, "conflicting updates for one group")
					}
					updates[group.ID] = update
				}
			}
			explicitStateChange := assignment.Pinned != nil && *assignment.Pinned != tab.Pinned || assignment.Index != nil && *assignment.Index != tab.Index
			if tab.GroupID != group.ID || explicitStateChange {
				tabOperations = append(tabOperations, Operation{Kind: OperationMoveTab, TabID: tab.ID, GroupID: group.ID, WindowID: tab.WindowID, Pinned: assignment.Pinned, Index: assignment.Index})
			}
			continue
		}
		color := destination.Color
		if color == "" {
			color = "grey"
		}
		key := fmt.Sprintf("%d:%s:%s", tab.WindowID, destination.Title, color)
		if _, exists := creates[key]; !exists {
			creates[key] = Operation{
				Kind: OperationCreateGroup, NewGroupKey: key, WindowID: tab.WindowID, TabID: tab.ID,
				Title: destination.Title, Color: color,
			}
		}
		tabOperations = append(tabOperations, Operation{
			Kind: OperationMoveTab, TabID: tab.ID, NewGroupKey: key, WindowID: tab.WindowID, Pinned: assignment.Pinned, Index: assignment.Index,
		})
	}
	var operations []Operation
	createKeys := make([]string, 0, len(creates))
	for key := range creates {
		createKeys = append(createKeys, key)
	}
	sort.Strings(createKeys)
	for _, key := range createKeys {
		operations = append(operations, creates[key])
	}
	groupIDs := make([]int, 0, len(updates))
	for groupID := range updates {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Ints(groupIDs)
	for _, groupID := range groupIDs {
		operations = append(operations, updates[groupID])
	}
	operations = append(operations, tabOperations...)
	return operations, nil
}
