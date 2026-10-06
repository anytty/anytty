package runtime

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TestMethodRegistryMatchesProtocol keeps the executable registry, the
// HELLO example and the method table in PROTOCOL.zh-CN.md from drifting apart.
// The runtime registry is authoritative for execution; the document is the
// public contract and must enumerate exactly the same method names.
func TestMethodRegistryMatchesProtocol(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller unavailable")
	}
	docPath := filepath.Join(filepath.Dir(file), "..", "docs", "PROTOCOL.zh-CN.md")
	doc, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read protocol document: %v", err)
	}
	text := string(doc)
	start := strings.Index(text, "## 4.")
	if start < 0 {
		t.Fatalf("protocol method section not found")
	}
	end := strings.Index(text[start+len("## 4."):], "\n## ")
	if end < 0 {
		t.Fatalf("protocol method section has no terminating heading")
	}
	methodSection := text[start : start+len("## 4.")+end]
	rowRE := regexp.MustCompile(`\| \x60([a-zA-Z0-9_.]+)\x60 \|`)
	seen := map[string]bool{}
	for _, match := range rowRE.FindAllStringSubmatch(methodSection, -1) {
		seen[match[1]] = true
	}
	got := MethodNames()
	want := make([]string, 0, len(seen))
	for _, name := range got {
		if !seen[name] {
			t.Errorf("method %q is in runtime registry but missing from protocol table", name)
		}
		want = append(want, name)
	}
	for name := range seen {
		if _, ok := LookupMethod(name); !ok {
			t.Errorf("method %q is in protocol table but missing from runtime registry", name)
		}
	}
	// Keep the HELLO example useful to readers: it must contain every method
	// accepted by a fresh Session, even though the example is JSON-shaped prose.
	for _, name := range got {
		if !strings.Contains(text, `"`+name+`"`) {
			t.Errorf("method %q is missing from HELLO example", name)
		}
	}
	sort.Strings(want)
	if len(want) != len(got) {
		t.Fatalf("protocol table contains %d executable rows, registry has %d", len(want), len(got))
	}
}
