package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PreviewEngine interface {
	Preview(context.Context, ClassificationPlan) (PreviewResult, error)
}

func RegisterPreviewTool(server *mcp.Server, engine PreviewEngine) {
	definition := catalogDefinition(ToolChromeTabGroupsPreview)
	mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input PreviewInput) (*mcp.CallToolResult, PreviewResult, error) {
		output, err := engine.Preview(ctx, input.Plan)
		if err != nil {
			return mcpErrorResult(err), PreviewResult{}, nil
		}
		return nil, output, nil
	})
}
