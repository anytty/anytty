package render

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

func TestRenderVMBuilderDelegatesShellProjection(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Cols: 80, Rows: 24, Valid: true},
		Shell:    state.DefaultShell().OpenHelp("most-used"),
	}
	projected := NewShellProjector().Project(root)
	built := NewRenderVMBuilder().Build(root)

	if !reflect.DeepEqual(projected, built.Shell) {
		t.Fatalf("RenderVMBuilder must delegate shell projection projected=%#v built=%#v", projected, built.Shell)
	}
}

func TestProjectionFilesDoNotDependOnRendererRuntime(t *testing.T) {
	projectionFiles := []string{
		"projection.go",
		"vm.go",
		"content_projector_registry.go",
		"product_content.go",
		"copy_history.go",
		"style.go",
	}
	for _, file := range projectionFiles {
		data, err := os.ReadFile(filepath.Join(".", file))
		if err != nil {
			t.Fatalf("read projection file %s: %v", file, err)
		}
		text := string(data)
		for _, forbidden := range []string{
			"MeasureLayout(",
			"renderFramework(",
			"newCanvas(",
			"drawStyled",
			"overlayTextStyled",
			"writeTextStyled",
			"FrameSink",
			"Effect",
			"services.",
			"terminalhost",
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("projection file %s must not depend on renderer runtime token %q", file, forbidden)
			}
		}
	}
}
