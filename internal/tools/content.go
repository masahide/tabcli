package tools

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ContentGetInput struct {
	TabID    int `json:"tabId" jsonschema:"ID of exactly one current Chrome tab"`
	MaxChars int `json:"maxChars,omitempty" jsonschema:"maximum returned characters from 1 to 50000"`
}

// PageText marks privacy-sensitive content at the transport boundary. It
// intentionally has no String method so structured logging cannot stringify it
// through fmt.Stringer by accident.
type PageText string

type ContentGetResult struct {
	ProtocolVersion        int       `json:"protocolVersion"`
	TabID                  int       `json:"tabId"`
	Title                  string    `json:"title"`
	URL                    string    `json:"url"`
	ContentType            string    `json:"contentType"`
	Text                   PageText  `json:"text"`
	ExtractedAt            time.Time `json:"extractedAt"`
	Truncated              bool      `json:"truncated"`
	OriginalCharacterCount int       `json:"originalCharCount"`
	ReturnedCharacterCount int       `json:"returnedCharCount"`
	UntrustedContent       bool      `json:"untrustedContent"`
	ContentRevision        string    `json:"contentRevision"`
	DataHandlingNotice     string    `json:"dataHandlingNotice"`
}

type ContentCompareInput struct {
	TabIDs []int `json:"tabIds" jsonschema:"exactly two distinct positive current Chrome tab IDs"`
}

type ContentHashTab struct {
	TabID          int    `json:"tabId"`
	Title          string `json:"title"`
	URL            string `json:"url"`
	SHA256         string `json:"sha256"`
	CharacterCount int    `json:"characterCount"`
}

type ContentCompareResult struct {
	ProtocolVersion    int              `json:"protocolVersion"`
	HashAlgorithm      string           `json:"hashAlgorithm"`
	Match              bool             `json:"match"`
	ComparedAt         time.Time        `json:"comparedAt"`
	Tabs               []ContentHashTab `json:"tabs"`
	DataHandlingNotice string           `json:"dataHandlingNotice"`
}

type ContentDiffInput struct {
	TabIDs       []int `json:"tabIds" jsonschema:"exactly two distinct positive current Chrome tab IDs"`
	MaxChars     int   `json:"maxChars,omitempty" jsonschema:"maximum transient source characters per tab from 1 to 50000"`
	MaxDiffChars int   `json:"maxDiffChars,omitempty" jsonschema:"maximum returned changed-text characters from 1 to 50000"`
}

type ContentDiffTab struct {
	ContentHashTab
	SourceTruncated        bool `json:"sourceTruncated"`
	ReturnedCharacterCount int  `json:"returnedCharacterCount"`
}

type ContentDiffChange struct {
	Kind          string   `json:"kind"`
	OldLine       *int     `json:"oldLine,omitempty"`
	NewLine       *int     `json:"newLine,omitempty"`
	Text          PageText `json:"text"`
	TextTruncated bool     `json:"textTruncated,omitempty"`
}

type ContentDiffResult struct {
	ProtocolVersion            int                 `json:"protocolVersion"`
	HashAlgorithm              string              `json:"hashAlgorithm"`
	DiffAlgorithm              string              `json:"diffAlgorithm"`
	Format                     string              `json:"format"`
	Match                      bool                `json:"match"`
	ComparedAt                 time.Time           `json:"comparedAt"`
	Tabs                       []ContentDiffTab    `json:"tabs"`
	Changes                    []ContentDiffChange `json:"changes"`
	UntrustedContent           bool                `json:"untrustedContent"`
	Minimal                    bool                `json:"minimal"`
	SourceTruncated            bool                `json:"sourceTruncated"`
	DiffTruncated              bool                `json:"diffTruncated"`
	OriginalChangeCount        int                 `json:"originalChangeCount"`
	ReturnedChangeCount        int                 `json:"returnedChangeCount"`
	OriginalDiffCharacterCount int                 `json:"originalDiffCharacterCount"`
	ReturnedDiffCharacterCount int                 `json:"returnedDiffCharacterCount"`
	DataHandlingNotice         string              `json:"dataHandlingNotice"`
}

type ContentGetBridge interface {
	Content(context.Context, ContentGetInput) (ContentGetResult, error)
}

type ContentCompareBridge interface {
	CompareContent(context.Context, ContentCompareInput) (ContentCompareResult, error)
}

type ContentDiffBridge interface {
	DiffContent(context.Context, ContentDiffInput) (ContentDiffResult, error)
}

type ContentToolsBridge interface {
	ContentGetBridge
	ContentCompareBridge
	ContentDiffBridge
}

func GetContent(ctx context.Context, bridge ContentGetBridge, input ContentGetInput) (ContentGetResult, error) {
	if input.TabID <= 0 {
		return ContentGetResult{}, NewError(CodeInvalidArgument, "tabId must identify exactly one tab")
	}
	if input.MaxChars == 0 {
		input.MaxChars = 10_000
	}
	if input.MaxChars < 1 || input.MaxChars > 50_000 {
		return ContentGetResult{}, NewError(CodeInvalidArgument, "maxChars must be between 1 and 50000")
	}
	return bridge.Content(ctx, input)
}

func CompareContent(ctx context.Context, bridge ContentCompareBridge, input ContentCompareInput) (ContentCompareResult, error) {
	if err := validateComparisonTabIDs(input.TabIDs); err != nil {
		return ContentCompareResult{}, err
	}
	return bridge.CompareContent(ctx, input)
}

func DiffContent(ctx context.Context, bridge ContentDiffBridge, input ContentDiffInput) (ContentDiffResult, error) {
	if err := validateComparisonTabIDs(input.TabIDs); err != nil {
		return ContentDiffResult{}, err
	}
	if input.MaxChars == 0 {
		input.MaxChars = 50_000
	}
	if input.MaxDiffChars == 0 {
		input.MaxDiffChars = 20_000
	}
	if input.MaxChars < 1 || input.MaxChars > 50_000 {
		return ContentDiffResult{}, NewError(CodeInvalidArgument, "maxChars must be between 1 and 50000")
	}
	if input.MaxDiffChars < 1 || input.MaxDiffChars > 50_000 {
		return ContentDiffResult{}, NewError(CodeInvalidArgument, "maxDiffChars must be between 1 and 50000")
	}
	return bridge.DiffContent(ctx, input)
}

func validateComparisonTabIDs(tabIDs []int) error {
	if len(tabIDs) != 2 || tabIDs[0] <= 0 || tabIDs[1] <= 0 || tabIDs[0] == tabIDs[1] {
		return NewError(CodeInvalidArgument, "tabIds must contain exactly two distinct positive tab IDs")
	}
	return nil
}

func RegisterContentTools(server *mcp.Server, bridge ContentToolsBridge) {
	definition := catalogDefinition(ToolChromeTabContentGet)
	mcp.AddTool(server, mcpTool(definition), func(ctx context.Context, _ *mcp.CallToolRequest, input ContentGetInput) (*mcp.CallToolResult, ContentGetResult, error) {
		output, err := GetContent(ctx, bridge, input)
		if err != nil {
			return mcpErrorResult(err), ContentGetResult{}, nil
		}
		return nil, output, nil
	})

	compareDefinition := catalogDefinition(ToolChromeTabContentCompare)
	mcp.AddTool(server, mcpTool(compareDefinition), func(ctx context.Context, _ *mcp.CallToolRequest, input ContentCompareInput) (*mcp.CallToolResult, ContentCompareResult, error) {
		output, err := CompareContent(ctx, bridge, input)
		if err != nil {
			return mcpErrorResult(err), ContentCompareResult{}, nil
		}
		return nil, output, nil
	})

	diffDefinition := catalogDefinition(ToolChromeTabContentDiff)
	mcp.AddTool(server, mcpTool(diffDefinition), func(ctx context.Context, _ *mcp.CallToolRequest, input ContentDiffInput) (*mcp.CallToolResult, ContentDiffResult, error) {
		output, err := DiffContent(ctx, bridge, input)
		if err != nil {
			return mcpErrorResult(err), ContentDiffResult{}, nil
		}
		return nil, output, nil
	})
}
