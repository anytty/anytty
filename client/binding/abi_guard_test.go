package binding

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var expectedCSymbols = []string{
	"anytty_client_abi_version", "anytty_engine_create", "anytty_engine_open_session", "anytty_engine_execute",
	"anytty_engine_open_resource_stream", "anytty_engine_send_resource_stream_frame", "anytty_engine_close_resource_stream",
	"anytty_engine_command", "anytty_engine_next_event",
	"anytty_platform_next_request", "anytty_platform_complete", "anytty_engine_cancel", "anytty_engine_close_session", "anytty_engine_release",
	"anytty_engine_close", "anytty_buffer_free",
}

func TestBindingABIBaselinesStayGeneric(t *testing.T) {
	header, err := os.ReadFile("cabi/anytty_client.h")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(header), "ANYTTY_CLIENT_ABI_VERSION 3u") || ABIVersion != 3 {
		t.Fatalf("ABI version mismatch header=%q go=%d", header, ABIVersion)
	}
	re := regexp.MustCompile(`(?m)^anytty_status_v1 (anytty_[a-z0-9_]+)\(`)
	symbols := []string{"anytty_client_abi_version"}
	for _, match := range re.FindAllStringSubmatch(string(header), -1) {
		symbols = append(symbols, match[1])
	}
	if !slices.Equal(symbols, expectedCSymbols) {
		t.Fatalf("C symbols = %v, want %v", symbols, expectedCSymbols)
	}
	for _, forbidden := range []string{"terminal", "history", "file", "storage", "json", "base64"} {
		if strings.Contains(strings.ToLower(string(header)), forbidden) {
			t.Fatalf("binding ABI contains business/encoding term %q", forbidden)
		}
	}
}

func TestBindingCoreDoesNotImportPlatformOrDomainOwners(t *testing.T) {
	forbidden := []string{
		"C", "unsafe", "syscall/js", "encoding/json", "encoding/base64",
		"github.com/anytty/anytty/core", "github.com/anytty/anytty/tui", "github.com/anytty/anytty/cmd/anytty",
		"github.com/anytty/anytty/private", "github.com/anytty/anytty/internal/protocol",
		"github.com/anytty/anytty/remote", "github.com/pion/webrtc",
	}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && (entry.Name() == "cabi" || entry.Name() == "enginehost") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			for _, prefix := range forbidden {
				if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
					t.Errorf("binding core %s imports forbidden owner %s", path, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
