package app

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/masahide/tabcli/internal/cli"
	"github.com/masahide/tabcli/internal/mcpclient"
	"github.com/masahide/tabcli/internal/mcpserver"
	"github.com/masahide/tabcli/internal/tools"
)

type contractBridge struct{}

func (contractBridge) Snapshot(context.Context) (tools.ChromeSnapshot, error) {
	return tools.ChromeSnapshot{Tabs: []tools.ChromeTab{{ID: 17, Title: "Same", URL: "https://example.com", Operable: true}}}, nil
}

func TestDirectProxyAndCLIContractsMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "discovery.json")
	server := mcp.NewServer(&mcp.Implementation{Name: "contract", Version: "0.1.0"}, nil)
	tools.RegisterListTools(server, contractBridge{})
	running, err := mcpserver.Start(mcpserver.Config{
		ListenAddress: "127.0.0.1:0", DiscoveryPath: path, ProfileID: "default",
		ProtocolVersion: tools.ProtocolVersion, MCPServer: server,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = running.Close(context.Background()) })
	direct := mcpclient.New(path)

	var directResult tools.TabsListResult
	if err := direct.Call(context.Background(), tools.ToolChromeTabsList, tools.TabsListInput{}, &directResult); err != nil {
		t.Fatal(err)
	}
	proxySession := connectProxy(t, direct)
	proxyCall, err := proxySession.CallTool(context.Background(), &mcp.CallToolParams{Name: tools.ToolChromeTabsList, Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	proxyJSON, _ := json.Marshal(proxyCall.StructuredContent)
	var proxyResult tools.TabsListResult
	if err := json.Unmarshal(proxyJSON, &proxyResult); err != nil {
		t.Fatal(err)
	}
	var cliOut, cliErr bytes.Buffer
	if code := cli.Run(context.Background(), []string{"tabs", "list", "--json"}, direct, &cliOut, &cliErr); code != cli.ExitOK {
		t.Fatalf("CLI exit = %d, stderr = %q", code, cliErr.String())
	}
	var cliResult tools.TabsListResult
	if err := json.Unmarshal(cliOut.Bytes(), &cliResult); err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(directResult)
	for name, result := range map[string]tools.TabsListResult{"proxy": proxyResult, "cli": cliResult} {
		got, _ := json.Marshal(result)
		if !bytes.Equal(got, want) {
			t.Fatalf("%s result = %s, direct = %s", name, got, want)
		}
	}

	disconnected := mcpclient.New(filepath.Join(t.TempDir(), "missing.json"))
	var unused tools.TabsListResult
	if err := disconnected.Call(context.Background(), tools.ToolChromeTabsList, tools.TabsListInput{}, &unused); err == nil || !strings.Contains(err.Error(), string(tools.CodeBrowserDisconnected)) {
		t.Fatalf("direct disconnected error = %v", err)
	}
	disconnectedProxy := connectProxy(t, disconnected)
	proxyError, err := disconnectedProxy.CallTool(context.Background(), &mcp.CallToolParams{Name: tools.ToolChromeTabsList, Arguments: map[string]any{}})
	proxyErrorJSON, _ := json.Marshal(proxyError)
	if err != nil || !proxyError.IsError || !strings.Contains(string(proxyErrorJSON), string(tools.CodeBrowserDisconnected)) {
		t.Fatalf("proxy disconnected = %#v, %v", proxyError, err)
	}
	cliOut.Reset()
	cliErr.Reset()
	if code := cli.Run(context.Background(), []string{"tabs", "list", "--json"}, disconnected, &cliOut, &cliErr); code != cli.ExitBrowserDisconnected || !strings.Contains(cliOut.String(), string(tools.CodeBrowserDisconnected)) || cliErr.Len() != 0 {
		t.Fatalf("CLI disconnected exit = %d, stdout = %q, stderr = %q", code, cliOut.String(), cliErr.String())
	}
}
