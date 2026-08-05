package state

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	WorkbenchStorageAppID       = "tui"
	WorkbenchStorageScopePublic = "public"
	WorkbenchStorageKeyRoot     = "workbench/root"
	WorkbenchStorageSchema      = "anytty.tui.v3.workbench"
	WorkbenchStorageSchemaV1    = 1
	WorkbenchStorageSchemaV2    = 2

	legacyPaneCopyHistory PaneKind = "copy-history"
	legacyPaneExited      PaneKind = "exited"
)

var ErrInvalidWorkbenchSnapshot = errors.New("invalid workbench snapshot")

type WorkbenchStorageRef struct {
	AppID   string
	Scope   string
	OwnerID string
	Key     string
	Version uint64
}

type WorkbenchStorageSnapshot struct {
	Schema            string                `json:"schema"`
	SchemaVersion     int                   `json:"schemaVersion"`
	Workspace         WorkspaceState        `json:"workspace"`
	Workspaces        []WorkspaceState      `json:"workspaces"`
	PanelPresentation PanelPresentation     `json:"panelPresentation"`
	ActivePaneID      string                `json:"activePaneId"`
	ZoomedPaneID      string                `json:"zoomedPaneId,omitempty"`
	HeaderVisible     bool                  `json:"headerVisible"`
	FooterVisible     bool                  `json:"footerVisible"`
	TerminalViews     []TerminalViewBinding `json:"terminalViews,omitempty"`
}

func DefaultWorkbenchStorageRef(workspaceID string) WorkbenchStorageRef {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = DefaultWorkspaceID
	}
	return WorkbenchStorageRef{
		AppID:   WorkbenchStorageAppID,
		Scope:   WorkbenchStorageScopePublic,
		OwnerID: workspaceID,
		Key:     WorkbenchStorageKeyRoot,
	}
}

func (ref WorkbenchStorageRef) WithVersion(version uint64) WorkbenchStorageRef {
	ref.Version = version
	return ref
}

func (ref WorkbenchStorageRef) KeyPrefix() string {
	key := strings.TrimSpace(ref.Key)
	if key == "" {
		return "workbench/"
	}
	if strings.HasSuffix(key, "/") {
		return key
	}
	if slash := strings.LastIndex(key, "/"); slash >= 0 {
		return key[:slash+1]
	}
	return key
}

func SnapshotWorkbenchForStorage(shell ShellStore) WorkbenchStorageSnapshot {
	return SnapshotRootWorkbenchForStorage(Root{Shell: shell})
}

func SnapshotRootWorkbenchForStorage(root Root) WorkbenchStorageSnapshot {
	shell := root.Shell.EnsureDefaults()
	return WorkbenchStorageSnapshot{
		Schema:            WorkbenchStorageSchema,
		SchemaVersion:     WorkbenchStorageSchemaV2,
		Workspace:         workbenchStorageWorkspace(shell.Workspace),
		Workspaces:        workbenchStorageWorkspaces(shell.Workspaces),
		PanelPresentation: shell.PanelPresentation,
		ActivePaneID:      shell.ActivePaneID,
		ZoomedPaneID:      shell.ZoomedPaneID,
		HeaderVisible:     shell.HeaderVisible,
		FooterVisible:     shell.FooterVisible,
		TerminalViews:     terminalViewBindingsForWorkbenchStorage(root.TerminalViews.Bindings()),
	}
}

func workbenchStorageWorkspaces(workspaces []WorkspaceState) []WorkspaceState {
	if len(workspaces) == 0 {
		return nil
	}
	out := make([]WorkspaceState, len(workspaces))
	for index, workspace := range workspaces {
		out[index] = workbenchStorageWorkspace(workspace)
	}
	return out
}

func workbenchStorageWorkspace(workspace WorkspaceState) WorkspaceState {
	workspace = cloneWorkspace(workspace)
	for tabIndex := range workspace.Tabs {
		for paneIndex := range workspace.Tabs[tabIndex].Panes {
			workspace.Tabs[tabIndex].Panes[paneIndex] = workbenchStoragePane(workspace.Tabs[tabIndex].Panes[paneIndex])
		}
		for floatingIndex := range workspace.Tabs[tabIndex].Floatings {
			workspace.Tabs[tabIndex].Floatings[floatingIndex] = workbenchStorageFloating(workspace.Tabs[tabIndex].Floatings[floatingIndex])
		}
	}
	return workspace
}

func workbenchStoragePane(pane PaneState) PaneState {
	switch pane.Kind {
	case legacyPaneExited, legacyPaneCopyHistory:
		// 中文说明：旧 storage 可能保存过展示态；当前契约只保存连接槽位。
		if pane.TerminalID != "" {
			pane.Kind = PaneTerminalLive
		} else {
			pane.Kind = PaneEmpty
		}
	}
	return pane
}

func workbenchStorageFloating(floating FloatingPaneState) FloatingPaneState {
	pane := workbenchStoragePane(floating.Pane)
	// 中文说明：floating 是共享 slot；坐标、大小、层级、隐藏/折叠和 fit 都是每个 TUI 的本地显示态。
	return FloatingPaneState{
		ID:    floating.ID,
		Title: floating.Title,
		Pane:  pane,
	}
}

func (snapshot WorkbenchStorageSnapshot) ToShellStore() (ShellStore, error) {
	if err := snapshot.Validate(); err != nil {
		return ShellStore{}, err
	}
	shell := ShellStore{
		Workspace:         workbenchStorageWorkspace(snapshot.Workspace),
		Workspaces:        workbenchStorageWorkspaces(snapshot.Workspaces),
		PanelPresentation: snapshot.PanelPresentation,
		ActivePaneID:      snapshot.ActivePaneID,
		ZoomedPaneID:      snapshot.ZoomedPaneID,
		HeaderVisible:     snapshot.HeaderVisible,
		FooterVisible:     snapshot.FooterVisible,
		initialized:       true,
	}
	return shell.EnsureDefaults(), nil
}

func (snapshot WorkbenchStorageSnapshot) ToTerminalViewStore() (TerminalViewStore, error) {
	if err := snapshot.Validate(); err != nil {
		return TerminalViewStore{}, err
	}
	store := TerminalViewStore{}
	for _, binding := range snapshot.TerminalViews {
		binding = binding.ForWorkbenchRestore()
		if binding.FloatingID != "" {
			store = store.BindFloating(binding)
			continue
		}
		store = store.BindPane(binding)
	}
	store = store.withLegacyWorkbenchPaneBindings(snapshot.Workspace, snapshot.Workspaces)
	return store, nil
}

func (store TerminalViewStore) withLegacyWorkbenchPaneBindings(active WorkspaceState, workspaces []WorkspaceState) TerminalViewStore {
	store = store.withLegacyWorkspacePaneBindings(active)
	for _, workspace := range workspaces {
		store = store.withLegacyWorkspacePaneBindings(workspace)
	}
	return store
}

func (store TerminalViewStore) withLegacyWorkspacePaneBindings(workspace WorkspaceState) TerminalViewStore {
	for _, tab := range workspace.Tabs {
		for _, pane := range tab.Panes {
			if pane.ID == "" || pane.TerminalID == "" {
				continue
			}
			if _, ok := store.PaneBinding(pane.ID); ok {
				continue
			}
			// 中文说明：V2 早期 snapshot 只有 pane.TerminalID；恢复时迁移成 view binding，
			// 之后输入和 lifecycle 都只通过 TerminalViewStore 连接意图投影。
			store = store.BindPane(NewPaneTerminalView(pane.ID, pane.TerminalID, 0, 0, 0, TerminalResizeRoleFollower, "", TerminalPaneViewID(pane.ID), false))
		}
		for _, floating := range tab.Floatings {
			if floating.ID == "" || floating.Pane.TerminalID == "" {
				continue
			}
			if _, ok := store.FloatingBinding(floating.ID); ok {
				continue
			}
			store = store.BindFloating(NewFloatingTerminalView(floating.ID, floating.Pane.ID, floating.Pane.TerminalID, 0, 0, 0, TerminalResizeRoleFollower, "", TerminalFloatingViewID(floating.ID), false))
		}
	}
	return store
}

func terminalViewBindingsForWorkbenchStorage(bindings []TerminalViewBinding) []TerminalViewBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]TerminalViewBinding, 0, len(bindings))
	for _, binding := range bindings {
		out = append(out, binding.ForWorkbenchStorage())
	}
	return out
}

// ForWorkbenchStorage 只保留 pane/floating 到 terminal 的连接意图。
// Channel、CanResize、SizeLocked、OwnerViewID 等字段属于 core 当前 truth，重进 TUI 后必须重新 attach 获取。
func (binding TerminalViewBinding) ForWorkbenchStorage() TerminalViewBinding {
	binding.SurfaceID = ""
	binding.Channel = 0
	binding.Attached = false
	binding.CanResize = false
	binding.SizeLocked = false
	binding.ControlReason = ""
	binding.OwnerSurfaceID = ""
	binding.OwnerViewID = ""
	binding.ResizeEpoch = 0
	binding.ResizePending = false
	binding.AttachPending = false
	binding.Unresolved = false
	binding.UnresolvedReason = ""
	binding.LastError = ""
	binding.ResizeRole = TerminalResizeRoleFollower
	return binding
}

// ForWorkbenchRestore 兼容旧 snapshot 中已经写入的 runtime-only 字段。
func (binding TerminalViewBinding) ForWorkbenchRestore() TerminalViewBinding {
	return binding.ForWorkbenchStorage()
}

func (snapshot WorkbenchStorageSnapshot) Validate() error {
	if snapshot.Schema != WorkbenchStorageSchema || snapshot.SchemaVersion != WorkbenchStorageSchemaV2 {
		return ErrInvalidWorkbenchSnapshot
	}
	if snapshot.Workspace.ID == "" {
		return ErrInvalidWorkbenchSnapshot
	}
	if len(snapshot.Workspaces) == 0 {
		return ErrInvalidWorkbenchSnapshot
	}
	if snapshot.PanelPresentation != PanelPresentationCard && snapshot.PanelPresentation != PanelPresentationSplitLine {
		return ErrInvalidWorkbenchSnapshot
	}
	if !snapshot.validTerminalViews() {
		return ErrInvalidWorkbenchSnapshot
	}
	return nil
}

func (snapshot WorkbenchStorageSnapshot) validTerminalViews() bool {
	seen := map[string]struct{}{}
	for _, binding := range snapshot.TerminalViews {
		if binding.ViewID == "" || binding.TerminalID == "" || binding.PaneID == "" && binding.FloatingID == "" {
			return false
		}
		if _, ok := seen[binding.ViewID]; ok {
			return false
		}
		seen[binding.ViewID] = struct{}{}
	}
	return true
}

func EncodeWorkbenchStorageSnapshot(shell ShellStore) ([]byte, error) {
	return json.Marshal(SnapshotWorkbenchForStorage(shell))
}

func EncodeWorkbenchStorageSnapshotValue(snapshot WorkbenchStorageSnapshot) ([]byte, error) {
	if err := snapshot.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(snapshot)
}

func DecodeWorkbenchStorageSnapshot(data []byte) (WorkbenchStorageSnapshot, error) {
	var snapshot WorkbenchStorageSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return WorkbenchStorageSnapshot{}, err
	}
	if err := snapshot.Validate(); err != nil {
		return WorkbenchStorageSnapshot{}, err
	}
	return snapshot, nil
}
