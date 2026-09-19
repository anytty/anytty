package files

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolverNormalizesRelativeHomeAndDotSegments(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	resolver, err := NewResolver(ResolverConfig{Roots: []string{root, home}, BaseDir: root, Home: home, CaseInsensitive: boolPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		input string
		want  string
	}{
		{"sub/file.txt", filepath.Join(root, "sub", "file.txt")},
		{"./sub/../sub/file.txt", filepath.Join(root, "sub", "file.txt")},
		{"~/docs", filepath.Join(home, "docs")},
	}
	for _, testCase := range cases {
		got, err := resolver.Resolve(testCase.input)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", testCase.input, err)
		}
		want, err := resolveExistingPrefix(testCase.want)
		if err != nil {
			// 路径不存在时 resolveExistingPrefix 已回退到存在祖先并拼回；
			// 这里只接受成功结果。
			t.Fatalf("resolve want %q: %v", testCase.want, err)
		}
		if got != want {
			t.Fatalf("Resolve(%q) = %q, want %q", testCase.input, got, want)
		}
	}
	if _, err := resolver.Resolve(""); !errors.Is(err, ErrPathInvalid) {
		t.Fatalf("empty path err = %v, want ErrPathInvalid", err)
	}
}

func TestResolverRejectsEscapeByDotDotAndAbsolutePath(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	resolver, err := NewResolver(ResolverConfig{Roots: []string{root}, BaseDir: root, CaseInsensitive: boolPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve("../outside.txt"); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("dotdot escape err = %v, want ErrPathOutsideRoots", err)
	}
	if _, err := resolver.Resolve(filepath.Join(base, "other.txt")); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("absolute escape err = %v, want ErrPathOutsideRoots", err)
	}
	if _, err := resolver.Resolve("/etc/hostname"); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("system escape err = %v, want ErrPathOutsideRoots", err)
	}
}

func TestResolverRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{root, outside, filepath.Join(root, "inner")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "inner"), filepath.Join(root, "link-inner")); err != nil {
		t.Fatal(err)
	}
	resolver, err := NewResolver(ResolverConfig{Roots: []string{root}, CaseInsensitive: boolPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(filepath.Join(root, "escape", "secret.txt")); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("symlink escape err = %v, want ErrPathOutsideRoots", err)
	}
	if _, err := resolver.Resolve(filepath.Join(root, "escape", "new.txt")); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("symlink write escape err = %v, want ErrPathOutsideRoots", err)
	}
	if _, err := resolver.Resolve(filepath.Join(root, "link-inner", "ok.txt")); err != nil {
		t.Fatalf("symlink inside root err = %v, want nil", err)
	}
	// 断链目标不存在时也必须按 target 路径做逃逸判断：否则写入会跟随
	// 断链在 root 外创建文件。
	if err := os.Symlink(filepath.Join(outside, "not-yet-created.txt"), filepath.Join(root, "broken")); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(filepath.Join(root, "broken")); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("broken symlink escape err = %v, want ErrPathOutsideRoots", err)
	}
}

func TestResolverCaseSensitivityFollowsConfig(t *testing.T) {
	root := t.TempDir()
	insensitive, err := NewResolver(ResolverConfig{Roots: []string{filepath.Join(root, "Root")}, CaseInsensitive: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	sensitive, err := NewResolver(ResolverConfig{Roots: []string{filepath.Join(root, "Root")}, CaseInsensitive: boolPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	lowercase := filepath.Join(root, "root", "file.txt")
	if _, err := insensitive.Resolve(lowercase); err != nil {
		t.Fatalf("case-insensitive resolve err = %v", err)
	}
	if _, err := sensitive.Resolve(lowercase); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("case-sensitive resolve err = %v, want ErrPathOutsideRoots", err)
	}
}

func TestResolverWithoutRootsOnlyNormalizes(t *testing.T) {
	resolver, err := NewResolver(ResolverConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve("relative.txt"); !errors.Is(err, ErrPathInvalid) {
		t.Fatalf("relative without base err = %v, want ErrPathInvalid", err)
	}
	got, err := resolver.Resolve("/tmp/x/../y")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/y" {
		t.Fatalf("normalized = %q, want /tmp/y", got)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
