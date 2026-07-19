package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/masahide/tabcli/internal/tools"
)

const (
	ExitOK                   = 0
	ExitFailure              = 1
	ExitUsage                = 2
	ExitBrowserDisconnected  = 3
	ExitInvalidArgument      = 4
	ExitPlanInvalid          = 5
	ExitPlanStale            = 6
	ExitContentPermission    = 7
	ExitContentNotAccessible = 8
	ExitContentStale         = 9
	ExitPreviewExpired       = 10
	ExitPreviewNotFound      = 11
	ExitApplyRolledBack      = 12
	ExitApplyPartial         = 13
	ExitUndoUnavailable      = 14
	ExitProtocolMismatch     = 15
	ExitCrossWindowGroup     = 16
)

type CommandInfo struct {
	Path        string
	Usage       string
	Description string
	ToolName    string
}

func CommandMetadata() []CommandInfo {
	commands := []CommandInfo{
		{Path: "install", Usage: "install", Description: "Install the current-user Native Messaging manifest"},
		{Path: "uninstall", Usage: "uninstall", Description: "Remove product-managed current-user files"},
		{Path: "status", Usage: "status", Description: "Show the active Chrome connection"},
		{Path: "doctor", Usage: "doctor", Description: "Diagnose installation and connection without changes"},
		{Path: "version", Usage: "version", Description: "Show product, Go, OS/CPU, and protocol versions"},
	}
	for _, tool := range tools.Catalog {
		commands = append(commands, CommandInfo{Path: tool.CLI, Usage: tool.CLIUsage, Description: tool.Description, ToolName: tool.Name})
	}
	commands = append(commands, CommandInfo{Path: "mcp serve", Usage: "mcp serve", Description: "Serve the static tool catalog over stdio"})
	return commands
}

func RenderHelp(writer io.Writer) {
	fmt.Fprintln(writer, "tabcli")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  tabcli [--json] <command> [options]")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Commands:")
	for _, command := range CommandMetadata() {
		fmt.Fprintf(writer, "  %s\n      %s\n", command.Usage, command.Description)
	}
}

type Caller interface {
	Call(context.Context, string, any, any) error
}

type Command struct {
	Caller    Caller
	Install   func() (any, error)
	Uninstall func() (any, error)
	Status    func() (any, error)
	Doctor    func() (any, error)
	Version   any
	JSON      bool
}

type StatusResult struct {
	Connected       bool      `json:"connected"`
	Endpoint        string    `json:"endpoint"`
	PID             int       `json:"pid"`
	InstanceID      string    `json:"instanceId"`
	ProfileID       string    `json:"profileId"`
	ProtocolVersion int       `json:"protocolVersion"`
	CreatedAt       time.Time `json:"createdAt"`
}

func Run(ctx context.Context, args []string, caller Caller, stdout, stderr io.Writer) int {
	return Command{Caller: caller}.Run(ctx, args, stdout, stderr)
}

func (command Command) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	args, command.JSON = parseGlobalJSON(args, command.JSON)
	if len(args) == 0 {
		if command.JSON {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "a command is required"))
		}
		RenderHelp(stderr)
		return ExitUsage
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		RenderHelp(stdout)
		return ExitOK
	}
	switch args[0] {
	case "install":
		if len(args) != 1 {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "install does not accept arguments"))
		}
		if command.Install == nil {
			return command.writeError(stdout, stderr, errors.New("install is unavailable"))
		}
		result, err := command.Install()
		if err != nil {
			return command.writeError(stdout, stderr, err)
		}
		if command.JSON {
			return writeJSON(stdout, result, stderr)
		}
		encoded, _ := json.Marshal(result)
		var installed struct {
			ManifestPath string `json:"manifestPath"`
			RegistryKey  string `json:"registryKey"`
		}
		_ = json.Unmarshal(encoded, &installed)
		fmt.Fprintf(stdout, "Installed Native Messaging manifest: %s\n", installed.ManifestPath)
		if installed.RegistryKey != "" {
			fmt.Fprintf(stdout, "Registered Native Messaging host: %s\n", installed.RegistryKey)
		}
		return ExitOK
	case "uninstall":
		if len(args) != 1 {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "uninstall does not accept arguments"))
		}
		if command.Uninstall == nil {
			return command.writeError(stdout, stderr, errors.New("uninstall is unavailable"))
		}
		result, err := command.Uninstall()
		if err != nil {
			return command.writeError(stdout, stderr, err)
		}
		if command.JSON {
			return writeJSON(stdout, result, stderr)
		}
		fmt.Fprintln(stdout, "Removed product-managed Native Messaging registration and settings.")
		return ExitOK
	case "status":
		if len(args) != 1 {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "status does not accept arguments"))
		}
		if command.Status == nil {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeBrowserDisconnected, "Chrome is not connected"))
		}
		status, err := command.Status()
		if err != nil {
			return command.writeError(stdout, stderr, err)
		}
		if command.JSON {
			return writeJSON(stdout, status, stderr)
		}
		if value, ok := status.(StatusResult); ok {
			fmt.Fprintf(stdout, "Chrome connected: %s (PID %d, protocol %d)\n", value.Endpoint, value.PID, value.ProtocolVersion)
		} else {
			fmt.Fprintln(stdout, status)
		}
		return ExitOK
	case "doctor":
		if len(args) != 1 {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "doctor does not accept arguments"))
		}
		if command.Doctor == nil {
			return command.writeError(stdout, stderr, errors.New("doctor is unavailable"))
		}
		result, err := command.Doctor()
		if err != nil {
			return command.writeError(stdout, stderr, err)
		}
		if command.JSON {
			return writeJSON(stdout, result, stderr)
		}
		printDoctor(stdout, result)
		return ExitOK
	case "version", "--version":
		if len(args) != 1 {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "version does not accept arguments"))
		}
		if command.JSON {
			return writeJSON(stdout, command.Version, stderr)
		}
		fmt.Fprintln(stdout, formatVersion(command.Version))
		return ExitOK
	case "list":
		return command.listTabs(ctx, args[1:], stdout, stderr)
	case "content":
		return command.getContent(ctx, args[1:], stdout, stderr)
	case "compare":
		return command.compareContent(ctx, args[1:], stdout, stderr)
	case "diff":
		return command.diffContent(ctx, args[1:], stdout, stderr)
	case "close":
		return command.closeTabs(ctx, args[1:], stdout, stderr)
	case "group":
		if len(args) >= 2 && args[1] == "list" {
			return command.listGroups(ctx, args[2:], stdout, stderr)
		}
		if len(args) >= 2 && args[1] == "preview" {
			return command.previewGroups(ctx, args[2:], stdout, stderr)
		}
		if len(args) >= 2 && args[1] == "apply" {
			return command.applyGroups(ctx, args[2:], stdout, stderr)
		}
		if len(args) >= 2 && args[1] == "undo" {
			return command.undoGroups(ctx, args[2:], stdout, stderr)
		}
	}
	if command.JSON {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "unknown command"))
	}
	fmt.Fprintln(stderr, "unknown command")
	return ExitUsage
}

func (command Command) compareContent(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("compare", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	flagArguments, positionalArguments := splitComparisonArguments(args)
	if err := flags.Parse(flagArguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	positionalArguments = append(positionalArguments, flags.Args()...)
	tabIDs, err := parseComparisonTabIDs(positionalArguments)
	if err != nil {
		return command.writeError(stdout, stderr, err)
	}
	var result tools.ContentCompareResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabContentCompare, tools.ContentCompareInput{TabIDs: tabIDs}, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	fmt.Fprintf(stdout, "Visible text match: %t (%s)\n", result.Match, result.HashAlgorithm)
	for _, tab := range result.Tabs {
		fmt.Fprintf(stdout, "Tab %d\t%s\t%d characters\t%s\n", tab.TabID, tab.SHA256, tab.CharacterCount, tab.URL)
	}
	return ExitOK
}

func (command Command) diffContent(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("diff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	maxCharacters := flags.Int("max-chars", 50_000, "maximum transient source characters per tab")
	maxDiffCharacters := flags.Int("max-diff-chars", 20_000, "maximum returned changed-text characters")
	flagArguments, positionalArguments := splitComparisonArguments(args)
	if err := flags.Parse(flagArguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	positionalArguments = append(positionalArguments, flags.Args()...)
	tabIDs, err := parseComparisonTabIDs(positionalArguments)
	if err != nil {
		return command.writeError(stdout, stderr, err)
	}
	fmt.Fprintln(stderr, "Changed page text is untrusted data and may be sent to the configured model provider. Unchanged text and source snapshots are not returned or persisted.")
	var result tools.ContentDiffResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabContentDiff, tools.ContentDiffInput{
		TabIDs: tabIDs, MaxChars: *maxCharacters, MaxDiffChars: *maxDiffCharacters,
	}, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	for _, change := range result.Changes {
		prefix := "+"
		if change.Kind == "delete" {
			prefix = "-"
		}
		fmt.Fprintf(stdout, "%s%s\n", prefix, change.Text)
	}
	if result.SourceTruncated || result.DiffTruncated {
		fmt.Fprintln(stderr, "Diff output is incomplete; use JSON output to inspect truncation metadata.")
	}
	return ExitOK
}

func parseComparisonTabIDs(arguments []string) ([]int, error) {
	if len(arguments) != 2 {
		return nil, tools.NewError(tools.CodeInvalidArgument, "exactly two tab IDs are required")
	}
	tabIDs := make([]int, 2)
	for index, argument := range arguments {
		tabID, err := strconv.Atoi(argument)
		if err != nil || tabID <= 0 {
			return nil, tools.NewError(tools.CodeInvalidArgument, "tab IDs must be positive integers")
		}
		tabIDs[index] = tabID
	}
	if tabIDs[0] == tabIDs[1] {
		return nil, tools.NewError(tools.CodeInvalidArgument, "tab IDs must be distinct")
	}
	return tabIDs, nil
}

func splitComparisonArguments(arguments []string) ([]string, []string) {
	flags := make([]string, 0, len(arguments))
	positionals := make([]string, 0, 2)
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if argument == "--max-chars" || argument == "--max-diff-chars" {
			flags = append(flags, argument)
			if index+1 < len(arguments) {
				index++
				flags = append(flags, arguments[index])
			}
			continue
		}
		if strings.HasPrefix(argument, "-") {
			flags = append(flags, argument)
			continue
		}
		positionals = append(positionals, argument)
	}
	return flags, positionals
}

func (command Command) closeTabs(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("close", flag.ContinueOnError)
	flags.SetOutput(stderr)
	confirmed := flags.Bool("confirm", false, "confirm closing the exact tab IDs")
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	if !*confirmed {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeConfirmationRequired, "--confirm is required to close tabs"))
	}
	if flags.NArg() == 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "usage: tabcli close --confirm TAB_ID [TAB_ID ...]"))
	}
	tabIDs := make([]int, 0, flags.NArg())
	for _, argument := range flags.Args() {
		tabID, err := strconv.Atoi(argument)
		if err != nil || tabID <= 0 {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "tab IDs must be positive integers"))
		}
		tabIDs = append(tabIDs, tabID)
	}
	var result tools.TabsCloseResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabsClose, tools.TabsCloseInput{TabIDs: tabIDs, Confirmed: true}, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	fmt.Fprintf(stdout, "Closed tabs: %d\n", len(result.ClosedTabIDs))
	return ExitOK
}

func parseGlobalJSON(args []string, initial bool) ([]string, bool) {
	filtered := make([]string, 0, len(args))
	jsonOutput := initial
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, jsonOutput
}

func printDoctor(writer io.Writer, value any) {
	encoded, _ := json.Marshal(value)
	var report struct {
		Checks []struct {
			Name    string `json:"name"`
			OK      bool   `json:"ok"`
			Message string `json:"message"`
		} `json:"checks"`
		UpdateInstructions string `json:"updateInstructions"`
	}
	if json.Unmarshal(encoded, &report) != nil {
		fmt.Fprintln(writer, string(encoded))
		return
	}
	for _, check := range report.Checks {
		status := "FAIL"
		if check.OK {
			status = "OK"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\n", status, check.Name, check.Message)
	}
	if report.UpdateInstructions != "" {
		fmt.Fprintf(writer, "Update: %s\n", report.UpdateInstructions)
	}
}

func formatVersion(value any) string {
	if value == nil {
		return "version unavailable"
	}
	encoded, _ := json.Marshal(value)
	var version struct {
		Version         string `json:"version"`
		Commit          string `json:"commit"`
		GoVersion       string `json:"goVersion"`
		OS              string `json:"os"`
		Arch            string `json:"arch"`
		ProtocolVersion int    `json:"protocolVersion"`
	}
	if json.Unmarshal(encoded, &version) == nil && version.Version != "" {
		return fmt.Sprintf("tabcli %s (commit %s, %s, %s/%s, protocol %d)", version.Version, version.Commit, version.GoVersion, version.OS, version.Arch, version.ProtocolVersion)
	}
	return strings.Trim(string(encoded), `"`)
}

func (command Command) applyGroups(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("group apply", flag.ContinueOnError)
	flags.SetOutput(stderr)
	previewID := flags.String("preview-id", "", "approved preview ID")
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON && !errors.Is(err, flag.ErrHelp) {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	if flags.NArg() != 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "group apply does not accept positional arguments"))
	}
	if *previewID == "" {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "--preview-id is required"))
	}
	var result tools.ApplyResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabGroupsApply, tools.ApplyInput{PreviewID: *previewID}, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	fmt.Fprintf(stdout, "Apply status: %s\nApplied operations: %d\nRecovery: %s\n", result.Status, len(result.AppliedOperations), result.Recovery)
	return ExitOK
}

func (command Command) undoGroups(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("group undo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON && !errors.Is(err, flag.ErrHelp) {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	if flags.NArg() != 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "group undo does not accept arguments"))
	}
	var result tools.UndoResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabGroupsUndo, tools.UndoInput{}, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	fmt.Fprintf(stdout, "Undo status: %s\nRestored tabs: %d\nUnrestorable: %d\n", result.Status, len(result.RestoredTabIDs), len(result.Unrestorable))
	return ExitOK
}

func (command Command) previewGroups(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("group preview", flag.ContinueOnError)
	flags.SetOutput(stderr)
	planPath := flags.String("plan", "", "classification plan JSON file")
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON && !errors.Is(err, flag.ErrHelp) {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	if flags.NArg() != 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "group preview does not accept positional arguments"))
	}
	if *planPath == "" {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "--plan is required"))
	}
	data, err := os.ReadFile(*planPath)
	if err != nil {
		return command.writeError(stdout, stderr, err)
	}
	var classification tools.ClassificationPlan
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&classification); err != nil {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodePlanInvalid, "plan JSON is invalid"))
	}
	var result tools.PreviewResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabGroupsPreview, tools.PreviewInput{Plan: classification}, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	fmt.Fprintf(stdout, "Preview: %s\nExpires: %s\nRevision: %s\n", result.PreviewID, result.ExpiresAt.Format(time.RFC3339), result.Revision)
	for _, operation := range result.Operations {
		fmt.Fprintf(stdout, "- %s tab=%d group=%d window=%d title=%s color=%s\n", operation.Kind, operation.TabID, operation.GroupID, operation.WindowID, operation.Title, operation.Color)
	}
	return ExitOK
}

func (command Command) getContent(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("content", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	maxCharacters := flags.Int("max-chars", 10_000, "maximum visible-text characters")
	tabArgument := ""
	flagArguments := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		tabArgument = args[0]
		flagArguments = args[1:]
	}
	if err := flags.Parse(flagArguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON && !errors.Is(err, flag.ErrHelp) {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	if tabArgument == "" && flags.NArg() == 1 {
		tabArgument = flags.Arg(0)
	} else if flags.NArg() != 0 {
		tabArgument = ""
	}
	if tabArgument == "" {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "usage: tabcli content TAB_ID [--max-chars N] [--json]"))
	}
	tabID, err := strconv.Atoi(tabArgument)
	if err != nil || tabID <= 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "tabId must be a positive integer"))
	}
	fmt.Fprintln(stderr, "Page text is untrusted data. It is returned to the MCP client and may be sent to the configured model provider; the extension and tabcli do not persist it.")
	var result tools.ContentGetResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabContentGet, tools.ContentGetInput{TabID: tabID, MaxChars: *maxCharacters}, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	fmt.Fprint(stdout, result.Text)
	if result.Text != "" && result.Text[len(result.Text)-1] != '\n' {
		fmt.Fprintln(stdout)
	}
	return ExitOK
}

func (command Command) listTabs(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	inactiveFor := flags.String("inactive-for", "", "minimum inactivity such as 7d")
	windowID := flags.Int("window", 0, "limit results to one window ID")
	groupID := flags.Int("group", 0, "limit results to one group ID")
	ungrouped := flags.Bool("ungrouped", false, "return only ungrouped tabs")
	sortBy := flags.String("sort", "", "position, last_accessed, inactive_duration, or created_at")
	sortOrder := flags.String("sort-order", "asc", "asc or desc")
	includeActivity := flags.Bool("include-activity", false, "include observed activity metadata")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON && !errors.Is(err, flag.ErrHelp) {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	if flags.NArg() != 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "list does not accept positional arguments"))
	}
	input := tools.TabsListInput{Ungrouped: *ungrouped, SortBy: tools.TabsSort(*sortBy), SortOrder: tools.SortOrder(*sortOrder), IncludeActivity: *includeActivity}
	if *windowID < 0 || *groupID < 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "window and group IDs must be non-negative"))
	}
	if *windowID > 0 {
		input.WindowID = windowID
	}
	if *groupID > 0 {
		input.GroupID = groupID
	}
	if *inactiveFor != "" {
		duration, err := ParseDuration(*inactiveFor)
		if err != nil {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidDuration, err.Error()))
		}
		seconds := int64(duration / time.Second)
		input.InactiveForSeconds = &seconds
	}
	var result tools.TabsListResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabsList, input, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	table := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tWINDOW\tGROUP\tTITLE\tURL")
	for _, tab := range result.Tabs {
		fmt.Fprintf(table, "%d\t%d\t%d\t%s\t%s\n", tab.ID, tab.WindowID, tab.GroupID, tab.Title, tab.URL)
	}
	_ = table.Flush()
	return ExitOK
}

var durationPattern = regexp.MustCompile(`^([1-9][0-9]*)(m|h|d)$`)

func ParseDuration(value string) (time.Duration, error) {
	match := durationPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, fmt.Errorf("duration must be a positive integer followed by m, h, or d")
	}
	amount, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %w", err)
	}
	unit := time.Minute
	if match[2] == "h" {
		unit = time.Hour
	} else if match[2] == "d" {
		unit = 24 * time.Hour
	}
	if amount > int64((1<<63-1)/unit) {
		return 0, fmt.Errorf("duration is too large")
	}
	return time.Duration(amount) * unit, nil
}

func (command Command) listGroups(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("group list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	windowID := flags.Int("window", 0, "limit results to one window ID")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		if command.JSON && !errors.Is(err, flag.ErrHelp) {
			return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, err.Error()))
		}
		return ExitUsage
	}
	if flags.NArg() != 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "group list does not accept positional arguments"))
	}
	input := tools.GroupsListInput{}
	if *windowID < 0 {
		return command.writeError(stdout, stderr, tools.NewError(tools.CodeInvalidArgument, "window ID must be non-negative"))
	}
	if *windowID > 0 {
		input.WindowID = windowID
	}
	var result tools.GroupsListResult
	if err := command.Caller.Call(ctx, tools.ToolChromeTabGroupsList, input, &result); err != nil {
		return command.writeError(stdout, stderr, err)
	}
	if *jsonOutput || command.JSON {
		return writeJSON(stdout, result, stderr)
	}
	table := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tWINDOW\tCOLOR\tTITLE\tTABS")
	for _, group := range result.Groups {
		fmt.Fprintf(table, "%d\t%d\t%s\t%s\t%d\n", group.ID, group.WindowID, group.Color, group.Title, len(group.TabIDs))
	}
	_ = table.Flush()
	return ExitOK
}

func writeJSON(writer io.Writer, value any, stderr io.Writer) int {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitFailure
	}
	return ExitOK
}

func ExitCodeForError(err error) int {
	var toolError *tools.Error
	if errors.As(err, &toolError) {
		codes := map[tools.ErrorCode]int{
			tools.CodeBrowserDisconnected:       ExitBrowserDisconnected,
			tools.CodeDiscoveryNotFound:         ExitBrowserDisconnected,
			tools.CodeDiscoveryStale:            ExitBrowserDisconnected,
			tools.CodeUpstreamUnavailable:       ExitBrowserDisconnected,
			tools.CodeInvalidArgument:           ExitInvalidArgument,
			tools.CodeInvalidDuration:           ExitInvalidArgument,
			tools.CodeTabNotFound:               ExitInvalidArgument,
			tools.CodeGroupNotFound:             ExitInvalidArgument,
			tools.CodeTabNotOperable:            ExitInvalidArgument,
			tools.CodePlanInvalid:               ExitPlanInvalid,
			tools.CodePlanStale:                 ExitPlanStale,
			tools.CodeContentPermissionRequired: ExitContentPermission,
			tools.CodeContentNotAccessible:      ExitContentNotAccessible,
			tools.CodeContentExtractionFailed:   ExitContentNotAccessible,
			tools.CodeContentStale:              ExitContentStale,
			tools.CodePreviewExpired:            ExitPreviewExpired,
			tools.CodePreviewNotFound:           ExitPreviewNotFound,
			tools.CodeApplyFailedRolledBack:     ExitApplyRolledBack,
			tools.CodeApplyPartial:              ExitApplyPartial,
			tools.CodeUndoUnavailable:           ExitUndoUnavailable,
			tools.CodeProtocolVersionMismatch:   ExitProtocolMismatch,
			tools.CodeCrossWindowGroup:          ExitCrossWindowGroup,
			tools.CodeConfirmationRequired:      ExitInvalidArgument,
			tools.CodeTabCloseFailed:            ExitFailure,
		}
		if code, ok := codes[toolError.Code]; ok {
			return code
		}
	}
	return ExitFailure
}

func (command Command) writeError(stdout, stderr io.Writer, err error) int {
	if command.JSON {
		var toolError *tools.Error
		if errors.As(err, &toolError) {
			_ = json.NewEncoder(stdout).Encode(map[string]any{"error": toolError})
		} else {
			_ = json.NewEncoder(stdout).Encode(map[string]any{"error": map[string]any{"code": "INTERNAL_ERROR", "message": err.Error(), "retryable": false}})
		}
	} else {
		fmt.Fprintln(stderr, err)
	}
	return ExitCodeForError(err)
}
