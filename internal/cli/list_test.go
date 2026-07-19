package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/masahide/tabcli/internal/tools"
)

type fakeCaller struct {
	err error
}

func (c fakeCaller) Call(_ context.Context, name string, _ any, output any) error {
	if c.err != nil {
		return c.err
	}
	switch name {
	case tools.ToolChromeTabsList:
		result := output.(*tools.TabsListResult)
		*result = tools.TabsListResult{ProtocolVersion: tools.ProtocolVersion, Tabs: []tools.Tab{{ID: 7, Title: "Example", URL: "https://example.com", WindowID: 1}}}
	case tools.ToolChromeTabGroupsList:
		result := output.(*tools.GroupsListResult)
		*result = tools.GroupsListResult{ProtocolVersion: tools.ProtocolVersion, Groups: []tools.Group{{ID: 3, Title: "Work", Color: "blue", WindowID: 1}}}
	}
	return nil
}

func TestTabsListTableOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"tabs", "list"}, fakeCaller{}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	for _, want := range []string{"ID", "TITLE", "URL", "7", "Example", "https://example.com"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("table output %q does not contain %q", stdout.String(), want)
		}
	}
}

func TestGroupsListJSONOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(context.Background(), []string{"groups", "list", "--json"}, fakeCaller{}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	var got tools.GroupsListResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("JSON output = %q: %v", stdout.String(), err)
	}
	if len(got.Groups) != 1 || got.Groups[0].ID != 3 {
		t.Fatalf("JSON result = %#v", got)
	}
}

func TestListBrowserDisconnectedExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run(
		context.Background(),
		[]string{"tabs", "list"},
		fakeCaller{err: tools.NewError(tools.CodeBrowserDisconnected, "Chrome is not connected")},
		&stdout,
		&stderr,
	)
	if exitCode != ExitBrowserDisconnected {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitBrowserDisconnected)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), string(tools.CodeBrowserDisconnected)) {
		t.Fatalf("stderr = %q, want structured code", stderr.String())
	}
}
