package main

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNormalizeViewerAssetManifest(t *testing.T) {
	root := t.TempDir()
	manifest := `{
  "schemaVersion": 1,
  "generatedBy": "test",
  "rendererIds": ["pdf"],
  "copiedAt": "2026-08-24T01:00:00Z",
  "assets": [{"rendererId":"pdf","id":"worker","to":` + strconv.Quote(filepath.Join(root, "vendor", "pdf.worker.mjs")) + `,"copied":true}]
}`

	normalized, err := normalizeAsset(root, "flyfish-viewer-assets.json", []byte(manifest))
	if err != nil {
		t.Fatal(err)
	}
	result := string(normalized)
	if strings.Contains(result, root) {
		t.Fatalf("normalized manifest leaks build root: %s", result)
	}
	if !strings.Contains(result, `"copiedAt": "1970-01-01T00:00:00Z"`) {
		t.Fatalf("normalized manifest has unstable timestamp: %s", result)
	}
	if !strings.Contains(result, `"to": "/vendor/pdf.worker.mjs"`) {
		t.Fatalf("normalized manifest has unexpected asset path: %s", result)
	}
}

func TestNormalizeViewerAssetManifestRejectsOutsidePath(t *testing.T) {
	root := t.TempDir()
	manifest := `{"assets":[{"to":` + strconv.Quote(filepath.Join(filepath.Dir(root), "outside")) + `}]}`
	if _, err := normalizeAsset(root, "flyfish-viewer-assets.json", []byte(manifest)); err == nil {
		t.Fatal("expected path outside the web root to be rejected")
	}
}
