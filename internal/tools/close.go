package tools

import (
	"context"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TabsCloseInput struct {
	TabIDs    []int `json:"tabIds" jsonschema:"IDs of the current Chrome tabs to close"`
	Confirmed bool  `json:"confirmed" jsonschema:"true only after the user explicitly approved closing these exact tab IDs"`
}

type TabsCloseResult struct {
	ProtocolVersion int   `json:"protocolVersion"`
	ClosedTabIDs    []int `json:"closedTabIds"`
}

type TabsCloseBridge interface {
	SnapshotBridge
	CloseTabs(context.Context, []int) (TabsCloseResult, error)
}

func CloseTabs(ctx context.Context, bridge TabsCloseBridge, input TabsCloseInput) (TabsCloseResult, error) {
	if !input.Confirmed {
		return TabsCloseResult{}, NewError(CodeConfirmationRequired, "closing tabs requires explicit user confirmation")
	}
	if len(input.TabIDs) == 0 {
		return TabsCloseResult{}, NewError(CodeInvalidArgument, "at least one tabId is required")
	}
	seen := make(map[int]bool, len(input.TabIDs))
	for _, tabID := range input.TabIDs {
		if tabID <= 0 {
			return TabsCloseResult{}, NewError(CodeInvalidArgument, "tabIds must be positive integers")
		}
		if seen[tabID] {
			return TabsCloseResult{}, NewError(CodeInvalidArgument, "tabIds must not contain duplicates")
		}
		seen[tabID] = true
	}

	snapshot, err := bridge.Snapshot(ctx)
	if err != nil {
		return TabsCloseResult{}, err
	}
	available := make(map[int]bool, len(snapshot.Tabs))
	for _, tab := range snapshot.Tabs {
		if !tab.Incognito {
			available[tab.ID] = true
		}
	}
	missing := make([]int, 0)
	for _, tabID := range input.TabIDs {
		if !available[tabID] {
			missing = append(missing, tabID)
		}
	}
	if len(missing) != 0 {
		sort.Ints(missing)
		toolError := NewError(CodeTabNotFound, "one or more selected tabs are no longer available")
		toolError.Details = map[string]any{"tabIds": missing}
		return TabsCloseResult{}, toolError
	}

	result, err := bridge.CloseTabs(ctx, append([]int(nil), input.TabIDs...))
	if err != nil {
		return TabsCloseResult{}, err
	}
	result.ProtocolVersion = ProtocolVersion
	return result, nil
}

func RegisterCloseTool(server *mcp.Server, bridge TabsCloseBridge) {
	definition := catalogDefinition(ToolChromeTabsClose)
	mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input TabsCloseInput) (*mcp.CallToolResult, TabsCloseResult, error) {
		output, err := CloseTabs(ctx, bridge, input)
		if err != nil {
			return mcpErrorResult(err), TabsCloseResult{}, nil
		}
		return nil, output, nil
	})
}
