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

func TestAndroidManagedRuntimeDoesNotRestoreLegacyHubProtocol(t *testing.T) {
	mirrorRoot := filepath.Join("..", "clients", "mobile", "native", "android")
	if _, err := os.Stat(mirrorRoot); !os.IsNotExist(err) {
		t.Fatalf("Android source mirror must stay deleted: %s", mirrorRoot)
	}
	root := filepath.Join("..", "clients", "mobile", "android", "app", "src", "main", "java", "com", "anytty", "app")
	forbidden := []string{"sessionToken", "session_token", "/api/v1/sessions", "Authorization\" to \"Bearer", "connectHub("}
	legacyFiles := []string{
		filepath.Join(root, "connectors", "HubConnector.kt"),
		filepath.Join(root, "connectors", "LocalConnector.kt"),
		filepath.Join(root, "connectors", "RaceConnector.kt"),
	}
	for _, path := range legacyFiles {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("legacy Android connector must stay deleted: %s", path)
		}
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (filepath.Ext(path) != ".kt" && filepath.Ext(path) != ".java") {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, fragment := range forbidden {
			if strings.Contains(string(payload), fragment) {
				t.Fatalf("Android managed runtime %s restored legacy protocol fragment %q", path, fragment)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan Android managed runtime: %v", err)
	}
}
