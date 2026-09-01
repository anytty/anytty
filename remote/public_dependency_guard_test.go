package remote_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicRemoteRuntimeDoesNotDependOnPrivateServices(t *testing.T) {
	forbidden := []string{
		"github.com/anytty/anytty/legacy-hub",
		"github.com/anytty/anytty/web-control",
		"github.com/anytty/anytty/private/",
		"session_token",
		"/api/v1/sessions",
	}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, fragment := range forbidden {
			if strings.Contains(string(data), fragment) {
				t.Fatalf("public remote runtime %s contains forbidden dependency or legacy field %q", path, fragment)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan public remote runtime: %v", err)
	}
	goMod, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, fragment := range forbidden[:3] {
		if strings.Contains(string(goMod), fragment) {
			t.Fatalf("public root go.mod contains forbidden dependency %q", fragment)
		}
	}
}

func TestFlutterManagedRuntimeDoesNotRestoreLegacyHubProtocol(t *testing.T) {
	legacyRoot := filepath.Join("..", "clients", "mobile")
	if _, err := os.Stat(legacyRoot); !os.IsNotExist(err) {
		t.Fatalf("legacy Capacitor client must stay deleted: %s", legacyRoot)
	}
	forbidden := []string{"sessionToken", "session_token", "/api/v1/sessions", "Authorization\" to \"Bearer", "connectHub("}
	roots := []string{
		filepath.Join("..", "clients", "flutter", "lib"),
		filepath.Join("..", "clients", "flutter", "android", "app", "src", "main", "kotlin"),
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			extension := filepath.Ext(path)
			if entry.IsDir() || (extension != ".dart" && extension != ".kt" && extension != ".java") {
				return nil
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, fragment := range forbidden {
				if strings.Contains(string(payload), fragment) {
					t.Fatalf("Flutter managed runtime %s restored legacy protocol fragment %q", path, fragment)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan Flutter managed runtime: %v", err)
		}
	}
}
