package tools

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ToolChromeTabsList          = "chrome_tabs_list"
	ToolChromeTabGroupsList     = "chrome_tab_groups_list"
	ToolChromeTabContentGet     = "chrome_tab_content_get"
	ToolChromeTabContentCompare = "chrome_tab_content_compare"
	ToolChromeTabContentDiff    = "chrome_tab_content_diff"
	ToolChromeTabGroupsPreview  = "chrome_tab_groups_preview"
	ToolChromeTabGroupsApply    = "chrome_tab_groups_apply"
	ToolChromeTabGroupsUndo     = "chrome_tab_groups_undo"
	ToolChromeTabsClose         = "chrome_tabs_close"
)

type Definition struct {
	Name        string
	Description string
	ReadOnly    bool
	CLI         string
	CLIUsage    string
}

type Caller interface {
	Call(context.Context, string, any, any) error
}

var Catalog = []Definition{
	{Name: ToolChromeTabsList, Description: "List current non-incognito Chrome tabs without changing Chrome state.", ReadOnly: true, CLI: "tabs list", CLIUsage: "tabs list [--window ID] [--group ID] [--ungrouped] [--inactive-for DURATION] [--sort FIELD] [--sort-order ORDER] [--include-activity]"},
	{Name: ToolChromeTabGroupsList, Description: "List current non-incognito Chrome tab groups without changing Chrome state.", ReadOnly: true, CLI: "groups list", CLIUsage: "groups list [--window ID]"},
	{Name: ToolChromeTabContentGet, Description: "Get bounded visible text from one explicitly selected tab as privacy-sensitive untrusted data that may be sent to the configured model provider and is not persisted.", ReadOnly: true, CLI: "tabs content", CLIUsage: "tabs content TAB_ID [--max-chars N]"},
	{Name: ToolChromeTabContentCompare, Description: "Compare the full visible text of exactly two selected tabs by SHA-256 without returning or persisting page text.", ReadOnly: true, CLI: "tabs compare", CLIUsage: "tabs compare TAB_ID_A TAB_ID_B"},
	{Name: ToolChromeTabContentDiff, Description: "Return only bounded changed visible-text lines for exactly two selected tabs; unchanged text and source snapshots are not returned or persisted.", ReadOnly: true, CLI: "tabs diff", CLIUsage: "tabs diff TAB_ID_A TAB_ID_B [--max-chars N] [--max-diff-chars N]"},
	{Name: ToolChromeTabGroupsPreview, Description: "Validate a classification plan and preview its changes without modifying Chrome.", ReadOnly: true, CLI: "groups preview", CLIUsage: "groups preview --plan FILE"},
	{Name: ToolChromeTabGroupsApply, Description: "Apply one valid, user-approved preview.", CLI: "groups apply", CLIUsage: "groups apply --preview-id ID"},
	{Name: ToolChromeTabGroupsUndo, Description: "Undo the most recent successful bulk apply where possible.", CLI: "groups undo", CLIUsage: "groups undo"},
	{Name: ToolChromeTabsClose, Description: "Close explicitly selected current Chrome tabs after user confirmation.", CLI: "tabs close", CLIUsage: "tabs close --confirm TAB_ID [TAB_ID ...]"},
}

func catalogDefinition(name string) Definition {
	for _, definition := range Catalog {
		if definition.Name == name {
			return definition
		}
	}
	panic("missing tool catalog definition: " + name)
}

func RegisterListTools(server *mcp.Server, bridge SnapshotBridge) {
	mcp.AddTool(server, &mcp.Tool{
		Name: ToolChromeTabsList, Description: catalogDefinition(ToolChromeTabsList).Description,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input TabsListInput) (*mcp.CallToolResult, TabsListResult, error) {
		output, err := ListTabs(ctx, bridge, input)
		if err != nil {
			return mcpErrorResult(err), TabsListResult{}, nil
		}
		return nil, output, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: ToolChromeTabGroupsList, Description: catalogDefinition(ToolChromeTabGroupsList).Description,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input GroupsListInput) (*mcp.CallToolResult, GroupsListResult, error) {
		output, err := ListGroups(ctx, bridge, input)
		if err != nil {
			return mcpErrorResult(err), GroupsListResult{}, nil
		}
		return nil, output, nil
	})
}

func RegisterProxyTools(server *mcp.Server, caller Caller) {
	for _, definition := range Catalog {
		definition := definition
		switch definition.Name {
		case ToolChromeTabsList:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input TabsListInput) (*mcp.CallToolResult, TabsListResult, error) {
				var output TabsListResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), TabsListResult{}, nil
				}
				return nil, output, nil
			})
		case ToolChromeTabGroupsList:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input GroupsListInput) (*mcp.CallToolResult, GroupsListResult, error) {
				var output GroupsListResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), GroupsListResult{}, nil
				}
				return nil, output, nil
			})
		case ToolChromeTabContentGet:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input ContentGetInput) (*mcp.CallToolResult, ContentGetResult, error) {
				var output ContentGetResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), ContentGetResult{}, nil
				}
				return nil, output, nil
			})
		case ToolChromeTabContentCompare:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input ContentCompareInput) (*mcp.CallToolResult, ContentCompareResult, error) {
				var output ContentCompareResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), ContentCompareResult{}, nil
				}
				return nil, output, nil
			})
		case ToolChromeTabContentDiff:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input ContentDiffInput) (*mcp.CallToolResult, ContentDiffResult, error) {
				var output ContentDiffResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), ContentDiffResult{}, nil
				}
				return nil, output, nil
			})
		case ToolChromeTabGroupsPreview:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input PreviewInput) (*mcp.CallToolResult, PreviewResult, error) {
				var output PreviewResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), PreviewResult{}, nil
				}
				return nil, output, nil
			})
		case ToolChromeTabGroupsApply:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input ApplyInput) (*mcp.CallToolResult, ApplyResult, error) {
				var output ApplyResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), ApplyResult{}, nil
				}
				return nil, output, nil
			})
		case ToolChromeTabGroupsUndo:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input UndoInput) (*mcp.CallToolResult, UndoResult, error) {
				var output UndoResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), UndoResult{}, nil
				}
				return nil, output, nil
			})
		case ToolChromeTabsClose:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input TabsCloseInput) (*mcp.CallToolResult, TabsCloseResult, error) {
				var output TabsCloseResult
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), TabsCloseResult{}, nil
				}
				return nil, output, nil
			})
		default:
			mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, map[string]any, error) {
				output := make(map[string]any)
				err := caller.Call(ctx, definition.Name, input, &output)
				if err != nil {
					return mcpErrorResult(err), map[string]any{}, nil
				}
				return nil, output, nil
			})
		}
	}
}

func mcpErrorResult(err error) *mcp.CallToolResult {
	var toolError *Error
	if !errors.As(err, &toolError) {
		toolError = NewError(CodeUpstreamUnavailable, "The upstream Chrome connection failed")
	}
	encoded, _ := json.Marshal(toolError)
	result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(encoded)}}}
	result.SetError(toolError)
	return result
}

func mcpTool(definition Definition) *mcp.Tool {
	destructive := definition.Name == ToolChromeTabsClose
	return &mcp.Tool{
		Name: definition.Name, Description: definition.Description,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: definition.ReadOnly, DestructiveHint: &destructive},
	}
}
