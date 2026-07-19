package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/masahide/tabcli/internal/tools"
)

func TestGeneratedHelpGoldenAndCommandMetadata(t *testing.T) {
	want, err := os.ReadFile("testdata/help.golden")
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	RenderHelp(&got)
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("help:\n%s\nwant:\n%s", got.Bytes(), want)
	}
	commands := CommandMetadata()
	if len(commands) != 15 {
		t.Fatalf("command metadata has %d commands, want 15", len(commands))
	}
	for _, command := range commands {
		if command.Path == "" || command.Usage == "" || command.Description == "" {
			t.Fatalf("invalid command metadata: %#v", command)
		}
	}
}

func TestTopLevelJSONProducesStableSuccessAndErrorShapes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		command := Command{Doctor: func() (any, error) {
			return map[string]any{"checks": []any{}}, nil
		}}
		if exit := command.Run(context.Background(), []string{"--json", "doctor"}, &stdout, &stderr); exit != ExitOK {
			t.Fatalf("exit=%d stderr=%q", exit, stderr.String())
		}
		var result map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if stderr.Len() != 0 {
			t.Fatalf("JSON diagnostics leaked to stderr: %q", stderr.String())
		}
	})
	t.Run("error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		command := Command{Caller: fakeCaller{err: tools.NewError(tools.CodePlanStale, "changed")}}
		exit := command.Run(context.Background(), []string{"--json", "groups", "list"}, &stdout, &stderr)
		if exit != ExitPlanStale || stderr.Len() != 0 {
			t.Fatalf("exit=%d stderr=%q", exit, stderr.String())
		}
		if !strings.Contains(stdout.String(), `"code":"PLAN_STALE"`) {
			t.Fatalf("JSON error = %q", stdout.String())
		}
	})
}

func TestStructuredErrorsHaveUniqueExitCodes(t *testing.T) {
	codes := []tools.ErrorCode{
		tools.CodeBrowserDisconnected,
		tools.CodeInvalidArgument,
		tools.CodePlanInvalid,
		tools.CodePlanStale,
		tools.CodeContentPermissionRequired,
		tools.CodeContentNotAccessible,
		tools.CodeContentStale,
		tools.CodePreviewExpired,
		tools.CodePreviewNotFound,
		tools.CodeApplyFailedRolledBack,
		tools.CodeApplyPartial,
		tools.CodeUndoUnavailable,
	}
	seen := map[int]tools.ErrorCode{}
	for _, code := range codes {
		exit := ExitCodeForError(tools.NewError(code, "test"))
		if exit == ExitOK || exit == ExitFailure || exit == ExitUsage {
			t.Fatalf("%s uses non-specific exit code %d", code, exit)
		}
		if previous, duplicate := seen[exit]; duplicate {
			t.Fatalf("%s and %s share exit code %d", previous, code, exit)
		}
		seen[exit] = code
	}
}

func TestMCPToolCatalogAndCLIHelpDoNotDrift(t *testing.T) {
	commands := CommandMetadata()
	byPath := make(map[string]CommandInfo, len(commands))
	for _, command := range commands {
		byPath[command.Path] = command
	}
	for _, tool := range tools.Catalog {
		command, ok := byPath[tool.CLI]
		if !ok {
			t.Errorf("MCP tool %s has no CLI command %q", tool.Name, tool.CLI)
			continue
		}
		if command.ToolName != tool.Name || command.Description != tool.Description || command.Usage != tool.CLIUsage {
			t.Errorf("catalog drift for %s: command=%#v tool=%#v", tool.Name, command, tool)
		}
	}
}
