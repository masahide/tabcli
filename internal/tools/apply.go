package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ApplyEngine interface {
	Apply(context.Context, string) (ApplyResult, error)
	Undo(context.Context) (UndoResult, error)
}

func RegisterApplyTools(server *mcp.Server, engine ApplyEngine) {
	applyDefinition := catalogDefinition(ToolChromeTabGroupsApply)
	mcp.AddTool(server, mcpTool(applyDefinition), func(ctx context.Context, _ *mcp.CallToolRequest, input ApplyInput) (*mcp.CallToolResult, ApplyResult, error) {
		output, err := engine.Apply(ctx, input.PreviewID)
		if err != nil {
			return mcpErrorResult(err), output, nil
		}
		return nil, output, nil
	})
	undoDefinition := catalogDefinition(ToolChromeTabGroupsUndo)
	mcp.AddTool(server, mcpTool(undoDefinition), func(ctx context.Context, _ *mcp.CallToolRequest, _ UndoInput) (*mcp.CallToolResult, UndoResult, error) {
		output, err := engine.Undo(ctx)
		if err != nil {
			return mcpErrorResult(err), output, nil
		}
		return nil, output, nil
	})
}
