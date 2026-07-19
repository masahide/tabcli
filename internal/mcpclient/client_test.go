package mcpclient

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/masahide/tabcli/internal/discovery"
	"github.com/masahide/tabcli/internal/mcpserver"
	"github.com/masahide/tabcli/internal/tools"
)

type staticBridge struct{}

func (staticBridge) Snapshot(context.Context) (tools.ChromeSnapshot, error) {
	return tools.ChromeSnapshot{Tabs: []tools.ChromeTab{{ID: 42, Title: "Authenticated", Operable: true}}}, nil
}

func startTestServer(t *testing.T, path string) *mcpserver.RunningServer {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.1.0"}, nil)
	tools.RegisterListTools(server, staticBridge{})
	running, err := mcpserver.Start(mcpserver.Config{
		ListenAddress: "127.0.0.1:0", DiscoveryPath: path, ProfileID: "default",
		ProtocolVersion: tools.ProtocolVersion, MCPServer: server,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = running.Close(context.Background()) })
	return running
}

func TestClientResolvesDiscoveryAndAuthenticates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "discovery.json")
	startTestServer(t, path)
	client := New(path)

	var result tools.TabsListResult
	if err := client.Call(context.Background(), tools.ToolChromeTabsList, tools.TabsListInput{}, &result); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if len(result.Tabs) != 1 || result.Tabs[0].ID != 42 {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientResolveReportsProtocolMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "discovery.json")
	running := startTestServer(t, path)
	file := discovery.File{
		Endpoint: running.Endpoint, PID: 1, InstanceID: running.InstanceID, ProfileID: "default",
		ProtocolVersion: tools.ProtocolVersion + 1, Token: running.Token,
	}
	if err := discovery.Write(path, file); err != nil {
		t.Fatal(err)
	}
	_, err := New(path).Resolve()
	if !errors.Is(err, discovery.ErrProtocolVersionMismatch) {
		t.Fatalf("Resolve() error = %v, want protocol mismatch", err)
	}
}

func TestClientNormalizesBadAuthenticationAndUpstreamDisconnect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "discovery.json")
	running := startTestServer(t, path)
	file, err := discovery.Read(path, discovery.ReadOptions{ProtocolVersion: tools.ProtocolVersion, ProcessAlive: func(int) bool { return true }})
	if err != nil {
		t.Fatal(err)
	}

	file.Token = "wrong-token"
	if err := discovery.Write(path, file); err != nil {
		t.Fatal(err)
	}
	var result tools.TabsListResult
	err = New(path).Call(context.Background(), tools.ToolChromeTabsList, tools.TabsListInput{}, &result)
	var toolError *tools.Error
	if !errors.As(err, &toolError) || toolError.Code != tools.CodeBrowserDisconnected {
		t.Fatalf("bad auth error = %v, want BROWSER_DISCONNECTED", err)
	}

	file.Token = running.Token
	if err := running.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := discovery.Write(path, file); err != nil {
		t.Fatal(err)
	}
	err = New(path).Call(context.Background(), tools.ToolChromeTabsList, tools.TabsListInput{}, &result)
	if !errors.As(err, &toolError) || toolError.Code != tools.CodeBrowserDisconnected {
		t.Fatalf("disconnect error = %v, want BROWSER_DISCONNECTED", err)
	}
}
