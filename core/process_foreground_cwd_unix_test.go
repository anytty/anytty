//go:build !windows

package core

import "testing"

func TestParseForegroundWorkingDirectoriesAcceptsOnlyAbsoluteCWDRecords(t *testing.T) {
	output := []byte("p220\nfcwd\nn/workspaces/anytty\np330\nfcwd\nnrelative/path\npbad\nn/ignored\n")
	got := parseForegroundWorkingDirectories(output)
	if got[220] != "/workspaces/anytty" {
		t.Fatalf("working directories = %#v", got)
	}
	if _, exists := got[330]; exists {
		t.Fatalf("relative cwd must be rejected: %#v", got)
	}
}
