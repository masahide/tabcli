package tools

import (
	"context"
	"fmt"
	"testing"
)

type contentBridgeFixture struct {
	calls        int
	compareCalls int
	diffCalls    int
	lastDiff     ContentDiffInput
}

func (bridge *contentBridgeFixture) Content(_ context.Context, input ContentGetInput) (ContentGetResult, error) {
	bridge.calls++
	return ContentGetResult{
		ProtocolVersion:  ProtocolVersion,
		TabID:            input.TabID,
		Text:             "untrusted page text",
		UntrustedContent: true,
		ContentRevision:  "opaque-revision",
	}, nil
}

func (bridge *contentBridgeFixture) CompareContent(_ context.Context, input ContentCompareInput) (ContentCompareResult, error) {
	bridge.compareCalls++
	return ContentCompareResult{
		ProtocolVersion: ProtocolVersion,
		HashAlgorithm:   "SHA-256",
		Match:           true,
		Tabs: []ContentHashTab{
			{TabID: input.TabIDs[0], SHA256: "same"},
			{TabID: input.TabIDs[1], SHA256: "same"},
		},
	}, nil
}

func (bridge *contentBridgeFixture) DiffContent(_ context.Context, input ContentDiffInput) (ContentDiffResult, error) {
	bridge.diffCalls++
	bridge.lastDiff = input
	return ContentDiffResult{
		ProtocolVersion:  ProtocolVersion,
		Format:           "line-changes",
		UntrustedContent: true,
		Changes: []ContentDiffChange{
			{Kind: "delete", Text: "old"},
			{Kind: "insert", Text: "new"},
		},
	}, nil
}

func TestContentGetRequiresOneTabAndReturnsUntrustedNonCachedResult(t *testing.T) {
	bridge := &contentBridgeFixture{}
	for i := 0; i < 2; i++ {
		result, err := GetContent(context.Background(), bridge, ContentGetInput{TabID: 7, MaxChars: 10_000})
		if err != nil {
			t.Fatalf("GetContent() error = %v", err)
		}
		if result.TabID != 7 || !result.UntrustedContent || result.ContentRevision == "" {
			t.Fatalf("result = %#v", result)
		}
	}
	if bridge.calls != 2 {
		t.Fatalf("bridge calls = %d, want 2 (no cache)", bridge.calls)
	}
	if _, err := GetContent(context.Background(), bridge, ContentGetInput{TabID: 0}); err == nil {
		t.Fatal("zero tab ID succeeded")
	}
}

func TestContentToolIsReadOnly(t *testing.T) {
	want := map[string]bool{
		ToolChromeTabContentGet:     false,
		ToolChromeTabContentCompare: false,
		ToolChromeTabContentDiff:    false,
	}
	for _, definition := range Catalog {
		if _, ok := want[definition.Name]; ok {
			if !definition.ReadOnly {
				t.Fatal("content tool must have read-only hint")
			}
			want[definition.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("content tool %s missing from catalog", name)
		}
	}
}

func TestContentCompareRequiresTwoDistinctTabsAndDoesNotCache(t *testing.T) {
	bridge := &contentBridgeFixture{}
	for i := 0; i < 2; i++ {
		result, err := CompareContent(context.Background(), bridge, ContentCompareInput{TabIDs: []int{7, 9}})
		if err != nil {
			t.Fatalf("CompareContent() error = %v", err)
		}
		if !result.Match || len(result.Tabs) != 2 {
			t.Fatalf("result = %#v", result)
		}
	}
	if bridge.compareCalls != 2 {
		t.Fatalf("compare calls = %d, want 2 (no cache)", bridge.compareCalls)
	}
	for _, tabIDs := range [][]int{{}, {7}, {7, 7}, {7, 9, 11}, {0, 9}} {
		if _, err := CompareContent(context.Background(), bridge, ContentCompareInput{TabIDs: tabIDs}); err == nil {
			t.Fatalf("invalid tab IDs succeeded: %v", tabIDs)
		}
	}
}

func TestContentDiffRequiresTwoTabsAndAppliesBounds(t *testing.T) {
	bridge := &contentBridgeFixture{}
	result, err := DiffContent(context.Background(), bridge, ContentDiffInput{TabIDs: []int{7, 9}})
	if err != nil {
		t.Fatalf("DiffContent() error = %v", err)
	}
	if result.Format != "line-changes" || !result.UntrustedContent || bridge.diffCalls != 1 {
		t.Fatalf("result = %#v calls = %d", result, bridge.diffCalls)
	}
	if bridge.lastDiff.MaxChars != 50_000 || bridge.lastDiff.MaxDiffChars != 20_000 {
		t.Fatalf("defaults = %#v", bridge.lastDiff)
	}
	for _, input := range []ContentDiffInput{
		{TabIDs: []int{7, 7}},
		{TabIDs: []int{7, 9}, MaxChars: 50_001},
		{TabIDs: []int{7, 9}, MaxDiffChars: 50_001},
	} {
		if _, err := DiffContent(context.Background(), bridge, input); err == nil {
			t.Fatalf("invalid input succeeded: %#v", input)
		}
	}
}

func TestPageTextHasNoLoggingStringRepresentation(t *testing.T) {
	var text any = PageText("sensitive")
	if _, implements := text.(fmt.Stringer); implements {
		t.Fatal("PageText must not implement fmt.Stringer")
	}
}
