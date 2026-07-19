package plan

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/masahide/tabcli/internal/tools"
)

type PreviewRecord struct {
	PreviewID      string             `json:"previewId"`
	Revision       string             `json:"revision"`
	ExpiresAt      time.Time          `json:"expiresAt"`
	NormalizedPlan ClassificationPlan `json:"normalizedPlan"`
	Operations     []Operation        `json:"operations"`
}

type TabUndoState struct {
	TabID    int  `json:"tabId"`
	WindowID int  `json:"windowId"`
	Index    int  `json:"index"`
	Pinned   bool `json:"pinned"`
	GroupID  int  `json:"groupId"`
}

type GroupUndoState struct {
	GroupID   int    `json:"groupId"`
	WindowID  int    `json:"windowId"`
	Title     string `json:"title"`
	Color     string `json:"color"`
	Collapsed bool   `json:"collapsed"`
}

type UndoSnapshot struct {
	SessionID string           `json:"sessionId"`
	Tabs      []TabUndoState   `json:"tabs"`
	Groups    []GroupUndoState `json:"groups"`
}

type PreviewOptions struct {
	Now                  func() time.Time
	RandomID             func() string
	ContentRevisionValid func(int, string) bool
}

type PreviewManager struct {
	bridge  tools.SnapshotBridge
	options PreviewOptions
	mu      sync.Mutex
	records map[string]PreviewRecord
}

func NewPreviewManager(bridge tools.SnapshotBridge, options PreviewOptions) *PreviewManager {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.RandomID == nil {
		options.RandomID = randomPreviewID
	}
	if options.ContentRevisionValid == nil {
		options.ContentRevisionValid = func(int, string) bool { return false }
	}
	return &PreviewManager{bridge: bridge, options: options, records: make(map[string]PreviewRecord)}
}

func (manager *PreviewManager) Preview(ctx context.Context, plan ClassificationPlan) (tools.PreviewResult, error) {
	snapshot, err := manager.bridge.Snapshot(ctx)
	if err != nil {
		return tools.PreviewResult{}, err
	}
	if err := ValidatePlan(plan, snapshot); err != nil {
		return tools.PreviewResult{}, err
	}
	if err := ValidateContentRevisions(plan, manager.options.ContentRevisionValid); err != nil {
		return tools.PreviewResult{}, err
	}
	operations, err := BuildOperations(plan, snapshot)
	if err != nil {
		return tools.PreviewResult{}, err
	}
	revision, err := Revision(snapshot)
	if err != nil {
		return tools.PreviewResult{}, err
	}
	now := manager.options.Now()
	record := PreviewRecord{
		PreviewID: manager.options.RandomID(), Revision: revision, ExpiresAt: now.Add(5 * time.Minute),
		NormalizedPlan: plan, Operations: operations,
	}
	manager.mu.Lock()
	manager.records[record.PreviewID] = record
	manager.mu.Unlock()
	return tools.PreviewResult{
		ProtocolVersion: tools.ProtocolVersion, PreviewID: record.PreviewID, Revision: record.Revision,
		ExpiresAt: record.ExpiresAt, NormalizedPlan: record.NormalizedPlan, Operations: record.Operations,
	}, nil
}

func (manager *PreviewManager) Lookup(previewID string) (PreviewRecord, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	record, exists := manager.records[previewID]
	if !exists {
		return PreviewRecord{}, tools.NewError(tools.CodePreviewNotFound, "previewId is unknown or already consumed")
	}
	if !manager.options.Now().Before(record.ExpiresAt) {
		delete(manager.records, previewID)
		return PreviewRecord{}, tools.NewError(tools.CodePreviewExpired, "previewId has expired")
	}
	return record, nil
}

func (manager *PreviewManager) Consume(previewID string) (PreviewRecord, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	record, exists := manager.records[previewID]
	if !exists {
		return PreviewRecord{}, tools.NewError(tools.CodePreviewNotFound, "previewId is unknown or already consumed")
	}
	delete(manager.records, previewID)
	if !manager.options.Now().Before(record.ExpiresAt) {
		return PreviewRecord{}, tools.NewError(tools.CodePreviewExpired, "previewId has expired")
	}
	return record, nil
}

func Revision(snapshot tools.ChromeSnapshot) (string, error) {
	type revisionTab struct {
		ID       int  `json:"id"`
		WindowID int  `json:"windowId"`
		Index    int  `json:"index"`
		GroupID  int  `json:"groupId"`
		Pinned   bool `json:"pinned"`
	}
	type revisionGroup struct {
		ID        int    `json:"id"`
		WindowID  int    `json:"windowId"`
		Title     string `json:"title"`
		Color     string `json:"color"`
		Collapsed bool   `json:"collapsed"`
		TabIDs    []int  `json:"tabIds"`
	}
	tabs := make([]revisionTab, 0, len(snapshot.Tabs))
	for _, tab := range snapshot.Tabs {
		if tab.Incognito {
			continue
		}
		tabs = append(tabs, revisionTab{ID: tab.ID, WindowID: tab.WindowID, Index: tab.Index, GroupID: tab.GroupID, Pinned: tab.Pinned})
	}
	sort.Slice(tabs, func(i, j int) bool { return tabs[i].ID < tabs[j].ID })
	groups := make([]revisionGroup, 0, len(snapshot.Groups))
	for _, group := range snapshot.Groups {
		if group.Incognito {
			continue
		}
		tabIDs := append([]int(nil), group.TabIDs...)
		sort.Ints(tabIDs)
		groups = append(groups, revisionGroup{ID: group.ID, WindowID: group.WindowID, Title: group.Title, Color: group.Color, Collapsed: group.Collapsed, TabIDs: tabIDs})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
	encoded, err := json.Marshal(struct {
		SessionID string          `json:"sessionId"`
		Tabs      []revisionTab   `json:"tabs"`
		Groups    []revisionGroup `json:"groups"`
	}{snapshot.SessionID, tabs, groups})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func randomPreviewID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}
