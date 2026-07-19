package tools

const ProtocolVersion = 3

type ChromeSnapshot struct {
	SessionID string        `json:"sessionId"`
	Tabs      []ChromeTab   `json:"tabs"`
	Groups    []ChromeGroup `json:"groups"`
}

type ChromeTab struct {
	ID           int               `json:"id"`
	Title        string            `json:"title"`
	URL          string            `json:"url"`
	WindowID     int               `json:"windowId"`
	Index        int               `json:"index"`
	GroupID      int               `json:"groupId"`
	Active       bool              `json:"active"`
	Pinned       bool              `json:"pinned"`
	LastAccessed *int64            `json:"lastAccessed,omitempty"`
	Incognito    bool              `json:"incognito"`
	Operable     bool              `json:"operable"`
	Activity     *ActivityMetadata `json:"activity,omitempty"`
}

type ActivityMetadata struct {
	CreatedAt                *int64 `json:"createdAt"`
	FirstObservedAt          int64  `json:"firstObservedAt"`
	ActivationCount          int    `json:"activationCount"`
	LastMovedAt              *int64 `json:"lastMovedAt"`
	LastGroupChangedAt       *int64 `json:"lastGroupChangedAt"`
	TrackingSince            int64  `json:"trackingSince"`
	ActivityDataCompleteness string `json:"activityDataCompleteness"`
}

type ChromeGroup struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Color     string `json:"color"`
	Collapsed bool   `json:"collapsed"`
	WindowID  int    `json:"windowId"`
	TabIDs    []int  `json:"tabIds"`
	Incognito bool   `json:"incognito"`
}

type Tab struct {
	ID                      int               `json:"id"`
	Title                   string            `json:"title"`
	URL                     string            `json:"url"`
	WindowID                int               `json:"windowId"`
	Index                   int               `json:"index"`
	GroupID                 int               `json:"groupId"`
	Active                  bool              `json:"active"`
	Pinned                  bool              `json:"pinned"`
	LastAccessed            *int64            `json:"lastAccessed,omitempty"`
	Operable                bool              `json:"operable"`
	InactiveDurationSeconds *int64            `json:"inactiveDurationSeconds"`
	Activity                *ActivityMetadata `json:"activity,omitempty"`
}

type Group struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Color     string `json:"color"`
	Collapsed bool   `json:"collapsed"`
	WindowID  int    `json:"windowId"`
	TabIDs    []int  `json:"tabIds"`
}

type TabsListInput struct {
	WindowID           *int      `json:"windowId,omitempty"`
	GroupID            *int      `json:"groupId,omitempty"`
	Ungrouped          bool      `json:"ungrouped,omitempty"`
	InactiveForSeconds *int64    `json:"inactiveForSeconds,omitempty"`
	SortBy             TabsSort  `json:"sortBy,omitempty"`
	SortOrder          SortOrder `json:"sortOrder,omitempty"`
	IncludeActivity    bool      `json:"includeActivity,omitempty"`
}

type TabsSort string

const (
	SortPosition         TabsSort = "position"
	SortLastAccessed     TabsSort = "last_accessed"
	SortInactiveDuration TabsSort = "inactive_duration"
	SortCreatedAt        TabsSort = "created_at"
)

type SortOrder string

const (
	SortAscending  SortOrder = "asc"
	SortDescending SortOrder = "desc"
)

type TabsListResult struct {
	ProtocolVersion int   `json:"protocolVersion"`
	Tabs            []Tab `json:"tabs"`
}

type GroupsListInput struct {
	WindowID *int `json:"windowId,omitempty"`
}

type GroupsListResult struct {
	ProtocolVersion int     `json:"protocolVersion"`
	Groups          []Group `json:"groups"`
}
