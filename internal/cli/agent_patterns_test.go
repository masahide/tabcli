package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/masahide/tabcli/internal/tools"
)

type capturingCaller struct {
	name  string
	input any
}

func (caller *capturingCaller) Call(_ context.Context, name string, input any, _ any) error {
	caller.name, caller.input = name, input
	return nil
}

func TestAgentJSONUnknownCommandIsStructured(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := (Command{}).Run(context.Background(), []string{"--json", "unknown"}, &stdout, &stderr)
	if exit != ExitInvalidArgument || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr.String())
	}
	var result struct {
		Error tools.Error `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Error.Code != tools.CodeInvalidArgument || result.Error.Retryable {
		t.Fatalf("error=%#v", result.Error)
	}
}

func TestAgentHelpReturnsSuccess(t *testing.T) {
	for _, arguments := range [][]string{
		{"tabs", "list", "--help"},
		{"tabs", "compare", "7", "9", "--help"},
		{"tabs", "diff", "7", "9", "--help"},
	} {
		var stdout, stderr bytes.Buffer
		if exit := Run(context.Background(), arguments, &capturingCaller{}, &stdout, &stderr); exit != ExitOK {
			t.Fatalf("args=%v exit=%d stderr=%q", arguments, exit, stderr.String())
		}
	}
}

func TestAgentCLIMapsLiteralFlagsToMCPInputs(t *testing.T) {
	t.Run("tabs list", func(t *testing.T) {
		caller := &capturingCaller{}
		var stdout, stderr bytes.Buffer
		exit := Run(context.Background(), []string{"--json", "tabs", "list", "--window", "2", "--ungrouped", "--inactive-for", "7d", "--sort", "created_at", "--sort-order", "desc", "--include-activity"}, caller, &stdout, &stderr)
		if exit != ExitOK {
			t.Fatalf("exit=%d stderr=%q", exit, stderr.String())
		}
		input := caller.input.(tools.TabsListInput)
		if caller.name != tools.ToolChromeTabsList || input.WindowID == nil || *input.WindowID != 2 || !input.Ungrouped || input.InactiveForSeconds == nil || *input.InactiveForSeconds != 604800 || input.SortBy != tools.SortCreatedAt || input.SortOrder != tools.SortDescending || !input.IncludeActivity {
			t.Fatalf("name=%q input=%#v", caller.name, input)
		}
	})

	t.Run("tabs content positional before flags", func(t *testing.T) {
		caller := &capturingCaller{}
		var stdout, stderr bytes.Buffer
		exit := Run(context.Background(), []string{"--json", "tabs", "content", "7", "--max-chars", "123"}, caller, &stdout, &stderr)
		if exit != ExitOK {
			t.Fatalf("exit=%d stderr=%q", exit, stderr.String())
		}
		input := caller.input.(tools.ContentGetInput)
		if caller.name != tools.ToolChromeTabContentGet || input.TabID != 7 || input.MaxChars != 123 {
			t.Fatalf("name=%q input=%#v", caller.name, input)
		}
	})

	t.Run("tabs compare exact IDs", func(t *testing.T) {
		caller := &capturingCaller{}
		var stdout, stderr bytes.Buffer
		exit := Run(context.Background(), []string{"--json", "tabs", "compare", "7", "9"}, caller, &stdout, &stderr)
		if exit != ExitOK {
			t.Fatalf("exit=%d stderr=%q", exit, stderr.String())
		}
		input := caller.input.(tools.ContentCompareInput)
		if caller.name != tools.ToolChromeTabContentCompare || len(input.TabIDs) != 2 || input.TabIDs[0] != 7 || input.TabIDs[1] != 9 {
			t.Fatalf("name=%q input=%#v", caller.name, input)
		}
	})

	t.Run("tabs diff exact IDs and bounds", func(t *testing.T) {
		caller := &capturingCaller{}
		var stdout, stderr bytes.Buffer
		exit := Run(context.Background(), []string{"--json", "tabs", "diff", "7", "9", "--max-chars", "321", "--max-diff-chars", "123"}, caller, &stdout, &stderr)
		if exit != ExitOK {
			t.Fatalf("exit=%d stderr=%q", exit, stderr.String())
		}
		input := caller.input.(tools.ContentDiffInput)
		if caller.name != tools.ToolChromeTabContentDiff || len(input.TabIDs) != 2 || input.TabIDs[0] != 7 || input.TabIDs[1] != 9 || input.MaxChars != 321 || input.MaxDiffChars != 123 {
			t.Fatalf("name=%q input=%#v", caller.name, input)
		}
	})

	t.Run("tabs close exact IDs and confirmation", func(t *testing.T) {
		caller := &capturingCaller{}
		var stdout, stderr bytes.Buffer
		exit := Run(context.Background(), []string{"--json", "tabs", "close", "--confirm", "7", "9"}, caller, &stdout, &stderr)
		if exit != ExitOK {
			t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
		}
		input := caller.input.(tools.TabsCloseInput)
		if caller.name != tools.ToolChromeTabsClose || !input.Confirmed || len(input.TabIDs) != 2 || input.TabIDs[0] != 7 || input.TabIDs[1] != 9 {
			t.Fatalf("name=%q input=%#v", caller.name, input)
		}
	})
}

func TestTabsCloseRequiresExplicitConfirmation(t *testing.T) {
	caller := &capturingCaller{}
	var stdout, stderr bytes.Buffer
	exit := Run(context.Background(), []string{"--json", "tabs", "close", "7"}, caller, &stdout, &stderr)
	if exit != ExitInvalidArgument || caller.name != "" || !bytes.Contains(stdout.Bytes(), []byte(tools.CodeConfirmationRequired)) {
		t.Fatalf("exit=%d called=%q stdout=%q stderr=%q", exit, caller.name, stdout.String(), stderr.String())
	}
}
