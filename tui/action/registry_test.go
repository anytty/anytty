package action

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistrySpecsAreCompleteAndCanonical(t *testing.T) {
	seen := map[ID]bool{}
	for _, spec := range Specs() {
		if spec.ID == "" || spec.DefaultLabel == "" {
			t.Fatalf("incomplete action spec: %#v", spec)
		}
		if seen[spec.ID] {
			t.Fatalf("duplicate canonical shortcut spec %q", spec.ID)
		}
		seen[spec.ID] = true
		if resolved, ok := SpecByID(spec.ID); !ok || resolved.ID != spec.ID {
			t.Fatalf("canonical action %q must be addressable without alias lookup: %#v ok=%v", spec.ID, resolved, ok)
		}
		for _, alias := range spec.Aliases {
			invocation, _, err := ParseInvocation(alias)
			if err != nil || invocation.ID != spec.ID {
				t.Fatalf("alias %q must resolve only to %q: invocation=%#v err=%v", alias, spec.ID, invocation, err)
			}
		}
	}
	for _, source := range []string{"panel.close", "pane.close", "tab.jump.3", "floating.summon.9", "help.close"} {
		if _, _, err := ParseInvocation(source); err != nil {
			t.Fatalf("parse %q: %v", source, err)
		}
	}
}

func TestRegistryCanonicalizesKnownAliases(t *testing.T) {
	cases := []struct {
		source string
		id     ID
	}{
		{source: "system.open_terminal_picker", id: "menu.terminal_picker"},
		{source: "system.open_workbench_tree", id: "menu.workbench_tree"},
		{source: "menu.pane", id: "menu.panel"},
	}
	for _, tc := range cases {
		invocation, _, err := ParseInvocation(tc.source)
		if err != nil || invocation.ID != tc.id {
			t.Fatalf("source=%q invocation=%#v err=%v", tc.source, invocation, err)
		}
	}
}

func TestDestructiveTabActionUsesKillLabel(t *testing.T) {
	spec, ok := SpecByID("tab.kill")
	if !ok || spec.DefaultLabel != "KILL" {
		t.Fatalf("tab.kill must be distinguished from non-destructive tab.close: %#v ok=%v", spec, ok)
	}
}

func TestParameterizedInvocationUsesBaseIDAndTypedParam(t *testing.T) {
	invocation, spec, err := ParseInvocation("tab.jump.3")
	if err != nil {
		t.Fatal(err)
	}
	index, ok := invocation.Param("index")
	if invocation.ID != "tab.jump" || invocation.SourceActionID != "tab.jump.3" || !ok || index != 3 || spec.ID != "tab.jump" {
		t.Fatalf("unexpected invocation=%#v spec=%#v", invocation, spec)
	}
	for _, source := range []string{"tab.jump.0", "tab.jump.10", "tab.jump.x", "floating.summon.-1"} {
		if _, _, err := ParseInvocation(source); err == nil {
			t.Fatalf("expected invalid parameter for %q", source)
		}
	}
}

func TestActionDomainDoesNotDependOnTUIConsumers(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imported := range parsed.Imports {
			if strings.Contains(imported.Path.Value, "/tui/") {
				t.Fatalf("neutral action domain must not depend on TUI consumer %s in %s", imported.Path.Value, file)
			}
		}
	}
}
