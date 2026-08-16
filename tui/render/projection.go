package render

import "github.com/anytty/anytty/tui/state"

func (RenderVMBuilder) Build(root state.Root) RenderVM {
	shell := NewShellProjector().Project(root)
	return RenderVM{Shell: shell, Theme: ThemeFromHostThemeConfig(root.HostTheme, root.Config)}
}

type ShellProjector struct {
	Content ContentProjectorRegistry
}

func NewShellProjector() ShellProjector {
	return ShellProjector{Content: DefaultContentProjectorRegistry()}
}

func (projector ShellProjector) Project(root state.Root) ShellVM {
	if projector.Content.projectors == nil {
		projector.Content = DefaultContentProjectorRegistry()
	}
	shellState := root.Shell.ReadonlyDefaults()
	root.Shell = shellState
	activeContent := projector.buildActiveContentVM(root, shellState)
	return ShellVM{
		Header:  buildHeaderVM(shellState, root),
		Footer:  buildFooterVM(root, shellState, activeContent),
		Layout:  projector.buildLayoutVM(shellState, activeContent, root),
		Overlay: projector.buildOverlayVM(root, shellState),
		Toasts:  buildToastVMs(shellState),
		Cursor:  activeContent.Cursor,
	}
}
