package tools

import (
	"encoding/json"
	"testing"
)

func TestCatalogHasThePublicTools(t *testing.T) {
	want := []string{
		ToolChromeTabsList,
		ToolChromeTabGroupsList,
		ToolChromeTabContentGet,
		ToolChromeTabContentCompare,
		ToolChromeTabContentDiff,
		ToolChromeTabGroupsPreview,
		ToolChromeTabGroupsApply,
		ToolChromeTabGroupsUndo,
		ToolChromeTabsClose,
	}
	if len(Catalog) != len(want) {
		t.Fatalf("catalog has %d tools, want %d", len(Catalog), len(want))
	}
	seen := make(map[string]bool)
	for i, definition := range Catalog {
		if definition.Name != want[i] {
			t.Fatalf("catalog[%d].Name = %q, want %q", i, definition.Name, want[i])
		}
		if definition.Description == "" || definition.CLI == "" || seen[definition.Name] {
			t.Fatalf("invalid or duplicate definition: %#v", definition)
		}
		seen[definition.Name] = true
	}
}

func TestCloseToolIsMarkedDestructive(t *testing.T) {
	tool := mcpTool(catalogDefinition(ToolChromeTabsClose))
	if tool.Annotations == nil || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
		t.Fatalf("annotations = %#v", tool.Annotations)
	}
}

func TestClassificationPlanSchemaIsValidDraft202012JSON(t *testing.T) {
	if !json.Valid([]byte(ClassificationPlanJSONSchema)) {
		t.Fatal("classification plan schema is not valid JSON")
	}
}
