package render

import (
	"reflect"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

func TestContentProjectorRegistryCoversProductContentKinds(t *testing.T) {
	registry := DefaultContentProjectorRegistry()
	for _, kind := range []ContentKind{
		ContentTerminalLive,
		ContentCopyHistory,
		ContentEmptyPane,
		ContentTerminalPicker,
		ContentTerminalPool,
		ContentWorkbenchTree,
		ContentClipboardHistory,
		ContentPrompt,
		ContentHelp,
		ContentPlaceholder,
	} {
		if _, ok := registry.Projector(kind); !ok {
			t.Fatalf("missing content projector for %s", kind)
		}
	}
}

func TestContentProjectorRegistryKeepsCopyHistoryAuthoritativeOnly(t *testing.T) {
	registry := DefaultContentProjectorRegistry()
	root := state.Root{
		CopyMode: state.CopyModeStore{
			Active:     true,
			TerminalID: "term-1",
			BoundCols:  80,
			BoundToken: "missing",
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Ready:      true,
			Lines:      []string{"live fallback must not render"},
		},
	}
	content := registry.Project(ContentProjectorContext{Root: root, Kind: ContentCopyHistory})
	if content.Kind != ContentCopyHistory || !content.Pending || len(content.Lines) == 0 {
		t.Fatalf("copy projector should return pending authoritative state, got %#v", content)
	}
	if content.Lines[0].String() == "live fallback must not render" {
		t.Fatalf("copy projector must not fallback to live surface")
	}
}

func TestContentProjectorRegistryUsesPlaceholderForInactiveCopyProjectionWithoutCopyMode(t *testing.T) {
	registry := DefaultContentProjectorRegistry()
	content := registry.Project(ContentProjectorContext{
		Kind: ContentCopyHistory,
		Pane: state.PaneState{ID: "copy-pane", Title: "history", Kind: state.PaneTerminalLive},
	})
	if content.Kind != ContentPlaceholder || !content.Pending || content.Lines[0].PlainString() != "history inactive" {
		t.Fatalf("expected inactive copy projection placeholder, got %#v", content)
	}
}

func TestContentProjectorRegistryUsesPlaceholderForInactivePendingLivePane(t *testing.T) {
	registry := DefaultContentProjectorRegistry()
	content := registry.Project(ContentProjectorContext{
		Kind:   ContentTerminalLive,
		Pane:   state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"},
		Active: false,
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-logs",
			State:      state.TerminalLivePending,
		},
	})
	if content.Kind != ContentPlaceholder || !content.Pending || content.Lines[0].PlainString() != "logs inactive" {
		t.Fatalf("expected inactive pending live pane placeholder, got %#v", content)
	}
}

func TestContentProjectorRegistryKeepsInactiveReadyLivePaneContent(t *testing.T) {
	registry := DefaultContentProjectorRegistry()
	content := registry.Project(ContentProjectorContext{
		Kind:   ContentTerminalLive,
		Pane:   state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"},
		Active: false,
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-logs",
			Ready:      true,
			State:      state.TerminalLiveAttached,
			Lines:      []string{"logs ready"},
		},
	})
	if content.Kind != ContentTerminalLive || content.Pending || content.Lines[0].PlainString() != "logs ready" {
		t.Fatalf("expected inactive ready live pane content, got %#v", content)
	}
}

func TestContentProjectorRegistryProjectsOverlayContent(t *testing.T) {
	registry := DefaultContentProjectorRegistry()
	shell := state.DefaultShell().OpenHelp("most-used")
	content := registry.Project(ContentProjectorContext{Root: state.Root{Shell: shell}, Shell: shell, Kind: ContentHelp})
	if content.Kind != ContentHelp || len(content.Lines) == 0 || len(content.HitRegions) == 0 {
		t.Fatalf("expected help content projection, got %#v", content)
	}
}

func TestShellProjectorUsesInjectedContentRegistry(t *testing.T) {
	registry := ContentProjectorRegistry{projectors: map[ContentKind]ContentProjector{}}
	registry.Register(ContentTerminalLive, ContentProjectorFunc(func(ctx ContentProjectorContext) ContentVM {
		return ContentVM{Kind: ContentTerminalLive, Lines: []Line{NewLine("custom live")}}
	}))
	registry.Register(ContentHelp, ContentProjectorFunc(func(ctx ContentProjectorContext) ContentVM {
		return ContentVM{Kind: ContentHelp, Lines: []Line{NewLine("custom help")}}
	}))

	shell := state.DefaultShell().OpenHelp("most-used")
	vm := ShellProjector{Content: registry}.Project(state.Root{Shell: shell})
	if got := activeContent(vm).Lines[0].PlainString(); got != "custom live" {
		t.Fatalf("expected active content from injected registry, got %q", got)
	}
	if got := vm.Overlay.Content.Lines[0].PlainString(); got != "custom help" {
		t.Fatalf("expected overlay content from injected registry, got %q", got)
	}
}

func TestContentRendererBoundaryConsumesOnlyContentVMAndRect(t *testing.T) {
	method, ok := reflect.TypeOf((*ContentRenderer)(nil)).Elem().MethodByName("RenderContent")
	if !ok {
		t.Fatalf("content renderer must expose RenderContent")
	}
	if method.Type.NumIn() != 1 {
		t.Fatalf("content renderer must accept one request argument, got %d", method.Type.NumIn())
	}
	requestType := method.Type.In(0)
	if requestType != reflect.TypeOf(ContentRenderRequest{}) {
		t.Fatalf("content renderer must accept ContentRenderRequest, got %s", requestType)
	}
	request := reflect.TypeOf(ContentRenderRequest{})
	if _, ok := request.FieldByName("Content"); !ok {
		t.Fatalf("content renderer request must carry ContentVM")
	}
	if _, ok := request.FieldByName("Rect"); !ok {
		t.Fatalf("content renderer request must carry Rect")
	}
	content := reflect.TypeOf(ContentVM{})
	if _, ok := content.FieldByName("Extent"); !ok {
		t.Fatalf("ContentVM must carry terminal extent for viewport projection")
	}
	result := reflect.TypeOf(ContentRenderResult{})
	if _, ok := result.FieldByName("Overflow"); !ok {
		t.Fatalf("content render result must carry overflow hints for pane chrome")
	}
	for i := 0; i < request.NumField(); i++ {
		if request.Field(i).Type == reflect.TypeOf(state.Root{}) {
			t.Fatalf("content renderer request must not carry state.Root")
		}
	}
}
