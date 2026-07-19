package skill

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type workflowFixture struct {
	Utterance                   string   `json:"utterance"`
	Tools                       []string `json:"tools"`
	Policy                      string   `json:"policy"`
	InactiveForSeconds          int      `json:"inactiveForSeconds"`
	RequiresApprovalBeforeApply bool     `json:"requiresApprovalBeforeApply"`
}

func TestRepresentativeWorkflowFixturesAreDocumented(t *testing.T) {
	var fixtures []workflowFixture
	readJSON(t, "testdata/workflows.json", &fixtures)
	if len(fixtures) < 4 {
		t.Fatalf("workflow fixture count = %d", len(fixtures))
	}
	skill := readSkill(t)
	for _, fixture := range fixtures {
		if fixture.Utterance == "" || len(fixture.Tools) == 0 || fixture.Policy == "" {
			t.Fatalf("incomplete fixture: %#v", fixture)
		}
		for _, tool := range fixture.Tools {
			if !strings.Contains(skill, tool) {
				t.Errorf("skill does not document %s for %q", tool, fixture.Utterance)
			}
		}
		if fixture.RequiresApprovalBeforeApply && !strings.Contains(skill, "承認前に`tabcli group apply`を実行しない") {
			t.Errorf("skill lacks the approval boundary")
		}
	}
	if fixtures[1].InactiveForSeconds != 7*24*60*60 || !strings.Contains(skill, "期間指定がなければ`7d`を使い") {
		t.Errorf("seven-day default is not fixed")
	}
	if !strings.Contains(skill, "`existing_groups_only`に固定") {
		t.Errorf("existing-group policy is not fixed")
	}
	for _, required := range []string{"正確なtab ID", "明示承認", "--confirm", "承認前、対象が変化した場合"} {
		if !strings.Contains(skill, required) {
			t.Errorf("close workflow lacks %q", required)
		}
	}
}

func TestSafetyResponseFixturesAreDocumented(t *testing.T) {
	var fixtures []struct {
		Scenario         string `json:"scenario"`
		RequiredGuidance string `json:"requiredGuidance"`
	}
	readJSON(t, "testdata/safety.json", &fixtures)
	skill := readSkill(t)
	for _, fixture := range fixtures {
		if fixture.Scenario == "" || fixture.RequiredGuidance == "" {
			t.Fatalf("incomplete safety fixture: %#v", fixture)
		}
		if !strings.Contains(skill, fixture.RequiredGuidance) {
			t.Errorf("skill lacks %s guidance: %q", fixture.Scenario, fixture.RequiredGuidance)
		}
	}
}

func readSkill(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../skills/tabcli/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}
