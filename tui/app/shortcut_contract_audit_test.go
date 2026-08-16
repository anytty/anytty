package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/shortcut"
	"github.com/anytty/anytty/tui/state"
)

// TestShortcutContractAuditClassifiesDefaultBindings 逐项证明默认 binding 能解析到唯一 spec、
// scene 合法且能进入 app handler；任一新增 binding 若落在未注册或未处理路径都会失败。
func TestShortcutContractAuditClassifiesDefaultBindings(t *testing.T) {
	entries := input.ShortcutEntriesForConfig(state.TUIShortcutConfig{})
	bindings := input.BindingCatalog()
	if len(entries) == 0 || len(bindings) == 0 || len(bindings) > len(entries) {
		t.Fatalf("default shortcut catalog and routed subset must be non-empty: entries=%d bindings=%d", len(entries), len(bindings))
	}

	seenKeys := map[string]string{}
	boundSpecs := map[string]bool{}
	entryInvocations := map[string]bool{}
	for _, entry := range entries {
		invocation, spec, err := actiondomain.ParseInvocation(entry.ActionID)
		if err != nil {
			t.Fatalf("default binding %s.%s action=%q has no canonical spec: %v", entry.Scene, entry.Key, entry.ActionID, err)
		}
		if !shortcut.AllowsScene(invocation.ID, entry.Scene) {
			t.Fatalf("default binding %s.%s action=%q is forbidden by spec %#v", entry.Scene, entry.Key, entry.ActionID, spec)
		}
		key := entry.Scene + "\x00" + entry.Key
		if previous, ok := seenKeys[key]; ok {
			t.Fatalf("default binding key %s.%s is duplicated by %q and %q", entry.Scene, entry.Key, previous, entry.ActionID)
		}
		seenKeys[key] = entry.ActionID
		boundSpecs[string(spec.ID)] = true
		entryInvocations[invocation.Signature()] = true

		intent, ok := shortcutIntentForInvocation(invocation, input.InputEvent{})
		if !ok {
			t.Fatalf("default binding %s.%s invocation=%#v has no canonical app handler", entry.Scene, entry.Key, invocation)
		}
		if intent.Kind == input.IntentNone || intent.Kind == input.IntentShortcutAction {
			t.Fatalf("default binding %s.%s invocation=%#v returned a non-concrete app intent %#v", entry.Scene, entry.Key, invocation, intent)
		}
	}

	for _, binding := range bindings {
		if !entryInvocations[binding.Invocation.Signature()] {
			t.Fatalf("routed binding %q invocation=%#v is absent from the default shortcut entries", binding.ID, binding.Invocation)
		}
		intent, ok := shortcutIntentForInvocation(binding.Invocation, input.InputEvent{})
		if !ok || intent.Kind == input.IntentNone || intent.Kind == input.IntentShortcutAction {
			t.Fatalf("routed binding %q invocation=%#v has no concrete app handler", binding.ID, binding.Invocation)
		}
	}

	seenSpecs := map[actiondomain.ID]bool{}
	for _, spec := range actiondomain.Specs() {
		if spec.ID == "" || seenSpecs[spec.ID] {
			t.Fatalf("canonical shortcut specs must be named and unique: %#v", spec)
		}
		seenSpecs[spec.ID] = true
	}
	for specID := range boundSpecs {
		if !seenSpecs[actiondomain.ID(specID)] {
			t.Fatalf("bound shortcut spec %q is not present in canonical spec inventory", specID)
		}
	}
}

// TestHitRegionProducersDeclareCanonicalInvocation 在 producer 层阻止 ActionID-only 命中区。
// append/layout 之后的测试无法区分 producer 正确与公共层 fallback，因此这里直接检查生产 AST。
func TestHitRegionProducersDeclareCanonicalInvocation(t *testing.T) {
	repoRoot := shortcutAuditRepoRoot(t)
	shortcutWalkProductionGo(t, repoRoot, func(path string, file *ast.File) {
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok || shortcutASTTypeName(literal.Type) != "HitRegion" {
				return true
			}
			hasActionID := false
			hasInvocation := false
			hasTargetMode := false
			hasRow := false
			hasRowPresence := false
			for _, element := range literal.Elts {
				field, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				name, _ := field.Key.(*ast.Ident)
				if name == nil {
					continue
				}
				switch name.Name {
				case "ActionID":
					hasActionID = true
				case "Invocation":
					hasInvocation = true
				case "TargetMode":
					hasTargetMode = true
				case "Row":
					hasRow = true
				case "HasRow":
					hasRowPresence = true
				}
			}
			if hasActionID && (!hasInvocation || !hasTargetMode) {
				t.Fatalf("%s#%s creates actionable HitRegion without producer-owned Invocation and TargetMode", path, shortcutEnclosingFunction(file, literal.Pos()))
			}
			if hasActionID && hasRow && !hasRowPresence {
				t.Fatalf("%s#%s creates row-target HitRegion without explicit HasRow", path, shortcutEnclosingFunction(file, literal.Pos()))
			}
			return true
		})
	})
}

// TestShortcutContractAuditClassifiesRenderProjections 证明 render 只保留有真实 surface 消费者的投影，
// 且每个投影都单向引用已注册 canonical action，不能重新拥有 dispatch 身份。
func TestShortcutContractAuditClassifiesRenderProjections(t *testing.T) {
	seen := map[render.ProjectionID]bool{}
	for _, spec := range render.ProjectionCatalog() {
		if spec.ID == "" || seen[spec.ID] {
			t.Fatalf("render action projections must be named and unique: %#v", spec)
		}
		seen[spec.ID] = true
		if len(spec.Surfaces) == 0 {
			t.Fatalf("render action projection %q has no classified surface", spec.ID)
		}
	}
	if len(seen) == 0 {
		t.Fatal("render action projection inventory must not be empty")
	}
}

func shortcutAuditRepoRoot(t *testing.T) string {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get shortcut audit working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(workingDir, "..", ".."))
}

func shortcutWalkProductionGo(t *testing.T, repoRoot string, visit func(path string, file *ast.File)) {
	t.Helper()
	err := filepath.WalkDir(filepath.Join(repoRoot, "tui"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		visit(filepath.ToSlash(relative), parsed)
		return nil
	})
	if err != nil {
		t.Fatalf("scan production TUI Go sources: %v", err)
	}
}

func shortcutASTTypeName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	default:
		return ""
	}
}

func shortcutEnclosingFunction(file *ast.File, position token.Pos) string {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Pos() <= position && position <= function.End() {
			return function.Name.Name
		}
	}
	return "<package>"
}
