package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"tui2-shell", []string{"tui2-shell"}},
		{"python3 tui2/examples/python-shell/shell.py", []string{"python3", "tui2/examples/python-shell/shell.py"}},
		{"python3 -u shell.py --flag", []string{"python3", "-u", "shell.py", "--flag"}},
		{"'sh' \"a b\"", []string{"sh", "a b"}},
		{`sh a\ b`, []string{"sh", "a b"}},
		{"   spaced   out  ", []string{"spaced", "out"}},
		{`python3 "quoted arg"`, []string{"python3", "quoted arg"}},
		{`python3 'single quote'`, []string{"python3", "single quote"}},
	}
	for _, tc := range cases {
		got, err := splitCommand(tc.in)
		if err != nil {
			t.Fatalf("splitCommand(%q): %v", tc.in, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("splitCommand(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
	for _, bad := range []string{`python3 "unbalanced`, `sh trailing\`} {
		if _, err := splitCommand(bad); err == nil {
			t.Errorf("splitCommand(%q) must fail", bad)
		}
	}
}

func TestResolveShellCommandWithArgs(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "fakeshell")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	// command on PATH plus arguments
	got, err := resolveShell("fakeshell --model demo")
	if err != nil {
		t.Fatalf("resolveShell command: %v", err)
	}
	want := []string{"fakeshell", "--model", "demo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveShell = %#v, want %#v", got, want)
	}

	// explicit path keeps working (single token)
	got, err = resolveShell(fake)
	if err != nil {
		t.Fatalf("resolveShell path: %v", err)
	}
	if !reflect.DeepEqual(got, []string{fake}) {
		t.Fatalf("resolveShell(path) = %#v", got)
	}

	// a bare positional path also works as command + args
	got, err = resolveShell(fake + " --flag")
	if err != nil {
		t.Fatalf("resolveShell(path+args): %v", err)
	}
	if !reflect.DeepEqual(got, []string{fake, "--flag"}) {
		t.Fatalf("resolveShell(path+args) = %#v", got)
	}

	// unknown command names the command in the error
	if _, err := resolveShell("nosuchcommand --x"); err == nil {
		t.Fatal("unknown command must fail")
	} else if got := err.Error(); !strings.Contains(got, "nosuchcommand") || !strings.Contains(got, "not found") {
		t.Fatalf("unclear error for missing command: %v", err)
	}

	// quoting keeps a path with spaces a single token
	spaced := filepath.Join(dir, "with space")
	if err := os.WriteFile(spaced, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = resolveShell("'" + spaced + "'")
	if err != nil {
		t.Fatalf("resolveShell(quoted path): %v", err)
	}
	if !reflect.DeepEqual(got, []string{spaced}) {
		t.Fatalf("resolveShell(quoted path) = %#v", got)
	}
}
