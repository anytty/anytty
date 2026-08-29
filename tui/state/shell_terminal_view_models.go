package state

import (
	"time"
)

type TerminalPickerStatusFilter string

const (
	TerminalPickerStatusAll     TerminalPickerStatusFilter = "all"
	TerminalPickerStatusRunning TerminalPickerStatusFilter = "running"
	TerminalPickerStatusExited  TerminalPickerStatusFilter = "exited"
)

type TerminalPickerView string

const (
	TerminalPickerViewList      TerminalPickerView = "list"
	TerminalPickerViewEndpoints TerminalPickerView = "endpoints"
	TerminalPickerViewTags      TerminalPickerView = "tags"
)

type TerminalPickerEndpointTab struct {
	EndpointID  EndpointID
	Label       string
	Count       int
	Selected    bool
	Transport   EndpointTransportKind
	ConnectMode EndpointConnectMode
	Status      EndpointStatusKind
	LastError   string
	ErrorKind   EndpointErrorKind
}

type TerminalPickerStatusOption struct {
	Status   TerminalPickerStatusFilter
	Label    string
	Count    int
	Selected bool
}

type TerminalPickerTagOption struct {
	Label    string
	Count    int
	Selected bool
}

// TerminalPickerItem 是 terminal picker 的只读行投影。
// EndpointID + TerminalID 是后续 attach/create/reconnect action 的目标身份；Endpoint* 字段来自 reducer-owned EndpointStore，renderer 不读取 registry 或 protocol client。
// EndpointSearchText 仅服务 create 行的跨 endpoint 搜索，展示仍必须使用 EndpointLabel，避免把搜索聚合文本当成可见 server 名称。
type TerminalPickerItem struct {
	EndpointID          EndpointID
	EndpointLabel       string
	EndpointSearchText  string
	EndpointTransport   EndpointTransportKind
	EndpointConnectMode EndpointConnectMode
	EndpointStatus      EndpointStatusKind
	EndpointLastError   string
	EndpointErrorKind   EndpointErrorKind
	PaneID              string
	Title               string
	Kind                PaneKind
	TerminalID          string
	Location            string
	Active              bool
	Selected            bool
	FromPool            bool
	PoolState           string
	Tags                map[string]string
	Cols                int
	Rows                int
	AttachmentCount     int
	LastOutputAt        time.Time
	CreateNew           bool
}

// TerminalPoolPageItem 是 Terminal Manager 页面使用的只读投影；真值来自 reducer-owned TerminalPoolStore，
// renderer 只能消费该投影绘制列表和详情，并通过 action 消息回到 app 层执行 attach/kill/edit/delete。
type TerminalPoolPageItem struct {
	EndpointID          EndpointID
	EndpointLabel       string
	EndpointTransport   EndpointTransportKind
	EndpointConnectMode EndpointConnectMode
	EndpointStatus      EndpointStatusKind
	EndpointLastError   string
	EndpointErrorKind   EndpointErrorKind
	TerminalID          string
	Title               string
	State               string
	CWD                 string
	Command             []string
	Tags                map[string]string
	ExitCode            *int
	ExitedAt            time.Time
	LastOutputAt        time.Time
	Cols                int
	Rows                int
	AttachmentCount     int
	Resources           TerminalResourceUsage
	ResourceHistory     []TerminalResourceUsage
	Attached            bool
	Selected            bool
}

// EndpointPickerGroup 是 terminal picker 的 endpoint 分组 view-model。
// Header 字段来自 EndpointStore；TerminalRows 只包含该 endpoint 下通过当前搜索过滤的 terminal 行。
type EndpointPickerGroup struct {
	EndpointID           EndpointID
	Label                string
	Transport            EndpointTransportKind
	ObservedPath         string
	RouteSelectionReason string
	ConnectionPhase      EndpointConnectionPhase
	ConnectMode          EndpointConnectMode
	Status               EndpointStatusKind
	LastError            string
	ErrorKind            EndpointErrorKind
	Configured           bool
	TerminalCount        int
	VisibleTerminalRows  []TerminalPickerItem
}

// TerminalPoolPageGroup 是 Terminal Manager 列表的 endpoint 分组 view-model。
// 它让 renderer 能展示 endpoint header，同时 action/selection 仍以 TerminalPoolPageItem 的 flat index 回到 app 层。
type TerminalPoolPageGroup struct {
	EndpointID           EndpointID
	Label                string
	Transport            EndpointTransportKind
	ObservedPath         string
	RouteSelectionReason string
	ConnectionPhase      EndpointConnectionPhase
	ConnectMode          EndpointConnectMode
	Status               EndpointStatusKind
	LastError            string
	ErrorKind            EndpointErrorKind
	Configured           bool
	TerminalCount        int
	VisibleTerminalRows  []TerminalPoolPageItem
}
