package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/masahide/tabcli/internal/tools"
)

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

type proxyCaller struct {
	called string
	err    error
}

func TestProxyListsToolsWhenBrowserDisconnectedAndFailsOnlyExecution(t *testing.T) {
	caller := &proxyCaller{err: tools.NewError(tools.CodeBrowserDisconnected, "Chrome is not connected")}
	session := connectProxy(t, caller)

	listed, err := session.ListTools(context.Background(), nil)
	if err != nil || len(listed.Tools) != len(tools.Catalog) {
		t.Fatalf("ListTools() = %#v, %v", listed, err)
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: tools.ToolChromeTabsList, Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool() protocol error = %v", err)
	}
	encoded, _ := json.Marshal(result)
	if !result.IsError || !strings.Contains(string(encoded), string(tools.CodeBrowserDisconnected)) {
		t.Fatalf("CallTool() result = %#v, want BROWSER_DISCONNECTED tool error", result)
	}
	var structuredError tools.Error
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			_ = json.Unmarshal([]byte(text.Text), &structuredError)
		}
	}
	if structuredError.Code != tools.CodeBrowserDisconnected || !structuredError.Retryable {
		t.Fatalf("structured error = %#v", structuredError)
	}
}

func TestContentComparisonToolsExposeTypedReadOnlySchemas(t *testing.T) {
	session := connectProxy(t, &proxyCaller{})
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		tools.ToolChromeTabContentCompare: {"tabIds", "sha256", "readOnlyHint"},
		tools.ToolChromeTabContentDiff:    {"tabIds", "maxChars", "maxDiffChars", "changes", "sha256", "sourceTruncated", "untrustedContent", "readOnlyHint"},
	}
	for _, tool := range listed.Tools {
		required, ok := want[tool.Name]
		if !ok {
			continue
		}
		encoded, _ := json.Marshal(tool)
		for _, fragment := range required {
			if !bytes.Contains(encoded, []byte(fragment)) {
				t.Errorf("tool %s schema lacks %q: %s", tool.Name, fragment, encoded)
			}
		}
		delete(want, tool.Name)
	}
	if len(want) != 0 {
		t.Fatalf("comparison tools missing: %v", want)
	}
	for name, arguments := range map[string]map[string]any{
		tools.ToolChromeTabContentCompare: {"tabIds": []int{7, 9}},
		tools.ToolChromeTabContentDiff:    {"tabIds": []int{7, 9}},
	} {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
		if err != nil || result.IsError {
			t.Fatalf("CallTool(%s) = %#v, %v", name, result, err)
		}
	}
}

func (caller *proxyCaller) Call(_ context.Context, name string, _ any, output any) error {
	caller.called = name
	if caller.err != nil {
		return caller.err
	}
	var value any = tools.TabsListResult{ProtocolVersion: tools.ProtocolVersion, Tabs: []tools.Tab{{ID: 9, Title: "Forwarded"}}}
	switch name {
	case tools.ToolChromeTabContentCompare:
		value = tools.ContentCompareResult{
			ProtocolVersion: tools.ProtocolVersion, HashAlgorithm: "SHA-256", Match: true,
			ComparedAt: time.Unix(0, 0).UTC(),
			Tabs: []tools.ContentHashTab{
				{TabID: 7, SHA256: "same", CharacterCount: 4},
				{TabID: 9, SHA256: "same", CharacterCount: 4},
			},
		}
	case tools.ToolChromeTabContentDiff:
		value = tools.ContentDiffResult{
			ProtocolVersion: tools.ProtocolVersion, HashAlgorithm: "SHA-256",
			DiffAlgorithm: "line-lcs-or-bounded-replacement", Format: "line-changes",
			ComparedAt: time.Unix(0, 0).UTC(), Tabs: []tools.ContentDiffTab{
				{ContentHashTab: tools.ContentHashTab{TabID: 7, SHA256: "old", CharacterCount: 3}},
				{ContentHashTab: tools.ContentHashTab{TabID: 9, SHA256: "new", CharacterCount: 3}},
			},
			Changes:          []tools.ContentDiffChange{{Kind: "delete", Text: "old"}, {Kind: "insert", Text: "new"}},
			UntrustedContent: true, Minimal: true,
		}
	}
	encoded, _ := json.Marshal(value)
	return json.Unmarshal(encoded, output)
}

func connectProxy(t *testing.T, caller ProxyCaller) *mcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := NewProxyServer(caller)
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	return clientSession
}

func TestProxyInitializesListsToolsAndForwardsCall(t *testing.T) {
	caller := &proxyCaller{}
	session := connectProxy(t, caller)
	instructions := session.InitializeResult().Instructions
	for _, required := range []string{"List tabs", "preview", "explicit approval", "apply", "content compare", "content diff", "Never close"} {
		if !strings.Contains(instructions, required) {
			t.Fatalf("instructions %q do not contain %q", instructions, required)
		}
	}

	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(listed.Tools) != len(tools.Catalog) {
		t.Fatalf("tools/list returned %d tools, want %d", len(listed.Tools), len(tools.Catalog))
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: tools.ToolChromeTabsList, Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() tool error = %v", result.GetError())
	}
	if caller.called != tools.ToolChromeTabsList {
		t.Fatalf("forwarded tool = %q", caller.called)
	}
	structured := result.StructuredContent.(map[string]any)
	if structured["protocolVersion"] != float64(tools.ProtocolVersion) {
		t.Fatalf("structured result = %#v", structured)
	}
}

func TestRunStdioProxyExitsOnEOFAndWritesOnlyJSONRPC(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	}, "\n") + "\n"
	var stdout, stderr bytes.Buffer
	err := RunStdioProxy(
		context.Background(),
		io.NopCloser(strings.NewReader(input)),
		nopWriteCloser{&stdout},
		&stderr,
		&proxyCaller{},
	)
	if err != nil {
		t.Fatalf("RunStdioProxy() error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	trimmed := strings.TrimSpace(stdout.String())
	if trimmed == "" {
		return
	}
	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		var message map[string]any
		if err := json.Unmarshal([]byte(line), &message); err != nil || message["jsonrpc"] != "2.0" {
			t.Fatalf("stdout line is not JSON-RPC: %q (%v)", line, err)
		}
	}
}
