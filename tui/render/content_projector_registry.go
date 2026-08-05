package render

import "github.com/anytty/anytty/tui/state"

type ContentProjector interface {
	ProjectContent(ContentProjectorContext) ContentVM
}

type ContentProjectorFunc func(ContentProjectorContext) ContentVM

func (fn ContentProjectorFunc) ProjectContent(ctx ContentProjectorContext) ContentVM {
	return fn(ctx)
}

type ContentProjectorContext struct {
	Root     state.Root
	Shell    state.ShellStore
	Pane     state.PaneState
	Binding  state.TerminalViewBinding
	Kind     ContentKind
	Active   bool
	Surface  state.TerminalSurfaceStore
	Session  state.TerminalSessionStore
	History  state.HistoryStore
	CopyMode state.CopyModeStore
}

type ContentProjectorRegistry struct {
	projectors map[ContentKind]ContentProjector
}

func DefaultContentProjectorRegistry() ContentProjectorRegistry {
	registry := ContentProjectorRegistry{projectors: map[ContentKind]ContentProjector{}}
	registry.Register(ContentTerminalLive, ContentProjectorFunc(projectTerminalLiveContent))
	registry.Register(ContentCopyHistory, ContentProjectorFunc(projectCopyHistoryContent))
	registry.Register(ContentEmptyPane, ContentProjectorFunc(projectEmptyPaneContent))
	registry.Register(ContentTerminalPicker, ContentProjectorFunc(projectTerminalPickerContent))
	registry.Register(ContentTerminalPool, ContentProjectorFunc(projectTerminalPoolContent))
	registry.Register(ContentConnections, ContentProjectorFunc(projectConnectionsContent))
	registry.Register(ContentWorkbenchTree, ContentProjectorFunc(projectWorkbenchTreeContent))
	registry.Register(ContentClipboardHistory, ContentProjectorFunc(projectClipboardHistoryContent))
	registry.Register(ContentFloatingOverview, ContentProjectorFunc(projectFloatingOverviewContent))
	registry.Register(ContentPrompt, ContentProjectorFunc(projectPromptContent))
	registry.Register(ContentHelp, ContentProjectorFunc(projectHelpContent))
	registry.Register(ContentPlaceholder, ContentProjectorFunc(projectPlaceholderContent))
	return registry
}

func (registry ContentProjectorRegistry) Register(kind ContentKind, projector ContentProjector) {
	if registry.projectors == nil || kind == "" || projector == nil {
		return
	}
	registry.projectors[kind] = projector
}

func (registry ContentProjectorRegistry) Projector(kind ContentKind) (ContentProjector, bool) {
	projector, ok := registry.projectors[kind]
	return projector, ok
}

func (registry ContentProjectorRegistry) Project(ctx ContentProjectorContext) ContentVM {
	ctx.Shell = contentProjectorShell(ctx)
	if projector, ok := registry.Projector(ctx.Kind); ok {
		return projector.ProjectContent(ctx)
	}
	return ContentVM{Kind: ContentPlaceholder, Lines: []Line{NewLine(activePaneTitle(ctx.Pane) + " inactive")}, Pending: true}
}

func contentProjectorShell(ctx ContentProjectorContext) state.ShellStore {
	if ctx.Shell.Workspace.ID != "" || len(ctx.Shell.Workspace.Tabs) > 0 || ctx.Shell.Overlay.Open {
		return ctx.Shell.ReadonlyDefaults()
	}
	return ctx.Root.Shell.ReadonlyDefaults()
}

func projectTerminalLiveContent(ctx ContentProjectorContext) ContentVM {
	surface := ctx.Surface
	session := ctx.Session
	if !ctx.Active && liveSurfaceIsPending(surface, session) {
		return placeholderContentForPane(ctx.Pane)
	}
	selectedIndex := 0
	if ctx.Active {
		selectedIndex = ctx.Shell.ReadonlyDefaults().ExitedPaneCTA.SelectedIndex
	}
	return buildLiveContentVMWithSelection(surface, session, ctx.Binding, endpointForBinding(ctx.Root, ctx.Binding), selectedIndex)
}

func endpointForBinding(root state.Root, binding state.TerminalViewBinding) state.EndpointItem {
	if binding.TerminalID == "" {
		return state.EndpointItem{}
	}
	endpoint, _ := root.Endpoints.DisplayEndpoint(binding.EndpointID)
	return endpoint
}

func liveSurfaceIsPending(surface state.TerminalSurfaceStore, session state.TerminalSessionStore) bool {
	return !surface.Ready &&
		surface.Err == "" &&
		session.LastError == "" &&
		surface.State != state.TerminalLiveExited &&
		session.State != state.TerminalLiveExited &&
		len(surface.Lines) == 0 &&
		len(surface.Screen) == 0
}

func projectCopyHistoryContent(ctx ContentProjectorContext) ContentVM {
	copyMode := ctx.CopyMode
	history := ctx.History
	if !copyMode.Active && !copyMode.Entering {
		history, copyMode = ctx.Root.History, ctx.Root.CopyMode
	}
	if !ctx.Active && !copyMode.HistoryRenderable() {
		return placeholderContentForPane(ctx.Pane)
	}
	return buildCopyHistoryContentVM(ctx.Root, history, copyMode)
}

func projectEmptyPaneContent(ctx ContentProjectorContext) ContentVM {
	return buildEmptyPaneContent(ctx.Pane)
}

func projectTerminalPickerContent(ctx ContentProjectorContext) ContentVM {
	return buildTerminalPickerContent(ctx.Root, ctx.Shell)
}

func projectTerminalPoolContent(ctx ContentProjectorContext) ContentVM {
	return buildTerminalPoolContent(ctx.Root, ctx.Shell)
}

func projectConnectionsContent(ctx ContentProjectorContext) ContentVM {
	return buildConnectionsContent(ctx.Root, ctx.Shell)
}

func projectWorkbenchTreeContent(ctx ContentProjectorContext) ContentVM {
	return buildWorkbenchTreeContent(ctx.Root, ctx.Shell)
}

func projectClipboardHistoryContent(ctx ContentProjectorContext) ContentVM {
	return buildClipboardHistoryContent(ctx.Root, ctx.Shell)
}

func projectFloatingOverviewContent(ctx ContentProjectorContext) ContentVM {
	return buildFloatingOverviewContent(ctx.Root, ctx.Shell)
}

func projectPromptContent(ctx ContentProjectorContext) ContentVM {
	return buildPromptContent(ctx.Shell)
}

func projectHelpContent(ctx ContentProjectorContext) ContentVM {
	return buildHelpContent(ctx.Root)
}

func projectPlaceholderContent(ctx ContentProjectorContext) ContentVM {
	return placeholderContentForPane(ctx.Pane)
}
