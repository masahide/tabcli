package plan

import (
	"fmt"

	"github.com/masahide/tabcli/internal/tools"
)

type ClassificationPlan = tools.ClassificationPlan
type Assignment = tools.PlanAssignment
type Destination = tools.PlanDestination
type ContentReference = tools.ContentReference
type Policy = tools.PlanPolicy

const (
	PolicyUngroupedOnly      = tools.PlanPolicyUngroupedOnly
	PolicyPreserveExisting   = tools.PlanPolicyPreserveExisting
	PolicyExistingGroupsOnly = tools.PlanPolicyExistingGroupsOnly
	PolicyRebuildSelected    = tools.PlanPolicyRebuildSelected
)

var validColors = map[string]bool{
	"grey": true, "blue": true, "red": true, "yellow": true, "green": true,
	"pink": true, "purple": true, "cyan": true, "orange": true,
}

func ValidatePlan(plan ClassificationPlan, snapshot tools.ChromeSnapshot) error {
	if plan.Policy != PolicyUngroupedOnly && plan.Policy != PolicyPreserveExisting && plan.Policy != PolicyExistingGroupsOnly && plan.Policy != PolicyRebuildSelected {
		return tools.NewError(tools.CodePlanInvalid, "unsupported policy")
	}
	tabs := make(map[int]tools.ChromeTab, len(snapshot.Tabs))
	for _, tab := range snapshot.Tabs {
		tabs[tab.ID] = tab
	}
	groups := make(map[int]tools.ChromeGroup, len(snapshot.Groups))
	for _, group := range snapshot.Groups {
		groups[group.ID] = group
	}
	seen := make(map[int]bool)
	for _, assignment := range plan.Assignments {
		if seen[assignment.TabID] {
			return tools.NewError(tools.CodePlanInvalid, fmt.Sprintf("duplicate tabId %d", assignment.TabID))
		}
		seen[assignment.TabID] = true
		tab, exists := tabs[assignment.TabID]
		if !exists || tab.Incognito || !tab.Operable {
			return tools.NewError(tools.CodePlanInvalid, fmt.Sprintf("unknown or inoperable tabId %d", assignment.TabID))
		}
		if assignment.Index != nil && *assignment.Index < 0 {
			return tools.NewError(tools.CodePlanInvalid, "tab index must be non-negative")
		}
		destination := assignment.Destination
		choices := 0
		if destination.Ungroup {
			choices++
		}
		if destination.GroupID != nil {
			choices++
		} else if destination.Title != "" {
			choices++
		}
		if choices != 1 {
			return tools.NewError(tools.CodePlanInvalid, "destination must select exactly one of ungroup, groupId, or a new group title")
		}
		if destination.GroupID != nil {
			if _, exists := groups[*destination.GroupID]; !exists {
				return tools.NewError(tools.CodePlanInvalid, fmt.Sprintf("unknown groupId %d", *destination.GroupID))
			}
		}
		if plan.Policy == PolicyExistingGroupsOnly && destination.GroupID == nil && !destination.Ungroup {
			return tools.NewError(tools.CodePlanInvalid, "existing_groups_only forbids new groups")
		}
		if destination.Color != "" && !validColors[destination.Color] {
			return tools.NewError(tools.CodePlanInvalid, fmt.Sprintf("invalid group color %q", destination.Color))
		}
	}
	for _, reference := range plan.ContentRevisions {
		if !seen[reference.TabID] || reference.Revision == "" {
			return tools.NewError(tools.CodePlanInvalid, "contentRevision must refer to an assigned tab")
		}
	}
	return nil
}

func ValidateContentRevisions(plan ClassificationPlan, valid func(int, string) bool) error {
	for _, reference := range plan.ContentRevisions {
		if !valid(reference.TabID, reference.Revision) {
			return tools.NewError(tools.CodeContentStale, "page content revision is no longer valid")
		}
	}
	return nil
}
