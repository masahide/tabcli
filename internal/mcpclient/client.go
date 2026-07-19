package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/masahide/tabcli/internal/buildinfo"
	"github.com/masahide/tabcli/internal/discovery"
	"github.com/masahide/tabcli/internal/tools"
)

type Client struct {
	DiscoveryPath string
}

func New(discoveryPath string) *Client {
	return &Client{DiscoveryPath: discoveryPath}
}

func (client *Client) Resolve() (discovery.File, error) {
	return discovery.Read(client.DiscoveryPath, discovery.ReadOptions{ProtocolVersion: tools.ProtocolVersion})
}

func (client *Client) Call(ctx context.Context, name string, input any, output any) error {
	file, err := client.Resolve()
	if err != nil {
		return tools.NewError(tools.CodeBrowserDisconnected, "Chrome is not connected. Start Chrome with the extension enabled, then retry.")
	}
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: bearerTransport{token: file.Token, base: http.DefaultTransport},
	}
	transport := &mcp.StreamableClientTransport{
		Endpoint: file.Endpoint, HTTPClient: httpClient, MaxRetries: -1, DisableStandaloneSSE: true,
	}
	sdkClient := mcp.NewClient(&mcp.Implementation{Name: "tabcli", Version: buildinfo.Version}, nil)
	session, err := sdkClient.Connect(ctx, transport, nil)
	if err != nil {
		return tools.NewError(tools.CodeBrowserDisconnected, "Chrome is not connected. Start Chrome with the extension enabled, then retry.")
	}
	defer session.Close()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: input})
	if err != nil {
		return tools.NewError(tools.CodeBrowserDisconnected, "Chrome connection failed. Ensure Chrome and the extension are running, then retry.")
	}
	if result.IsError {
		return callToolError(result)
	}
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, output)
}

func callToolError(result *mcp.CallToolResult) error {
	for _, content := range result.Content {
		text, ok := content.(*mcp.TextContent)
		if !ok || strings.TrimSpace(text.Text) == "" {
			continue
		}
		message := strings.TrimSpace(text.Text)
		var structured tools.Error
		if json.Unmarshal([]byte(message), &structured) == nil && structured.Code != "" && structured.Message != "" {
			return &structured
		}
		for _, code := range []tools.ErrorCode{
			tools.CodeBrowserDisconnected, tools.CodeDiscoveryNotFound, tools.CodeDiscoveryStale,
			tools.CodeProtocolVersionMismatch, tools.CodeUpstreamUnavailable, tools.CodeAuthenticationFailed,
			tools.CodeInvalidDuration, tools.CodeTabNotFound, tools.CodeGroupNotFound, tools.CodeTabNotOperable, tools.CodePlanInvalid,
			tools.CodePlanStale, tools.CodeContentPermissionRequired, tools.CodeContentNotAccessible,
			tools.CodeContentExtractionFailed, tools.CodeContentStale, tools.CodeUndoUnavailable, tools.CodeApplyFailedRolledBack,
			tools.CodeApplyPartial, tools.CodeInvalidArgument, tools.CodeCrossWindowGroup,
			tools.CodePreviewExpired, tools.CodePreviewNotFound,
		} {
			prefix := string(code) + ":"
			if strings.HasPrefix(message, prefix) {
				return tools.NewError(code, strings.TrimSpace(strings.TrimPrefix(message, prefix)))
			}
		}
		return errors.New(message)
	}
	return errors.New("MCP tool failed without an error message")
}

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (transport bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	copy := request.Clone(request.Context())
	copy.Header.Set("Authorization", "Bearer "+transport.token)
	return transport.base.RoundTrip(copy)
}
