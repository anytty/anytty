package core

import "testing"

func TestModuleName(t *testing.T) {
	if ModuleName != "core" {
		t.Fatalf("unexpected module name %q", ModuleName)
	}
}
