package appcodegen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestGeneratedFilesAreUpToDate 是 codegen 幂等护栏：重新生成的结果必须与
// checked-in 文件逐字节一致，否则 CI 必须失败而不是静默漂移。
func TestGeneratedFilesAreUpToDate(t *testing.T) {
	root := "../.."
	files, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		current, err := os.ReadFile(filepath.Join(root, file.Path))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(current, file.Content) {
			t.Errorf("%s is out of date; run go run ./scripts/generate_application_api.go", file.Path)
		}
	}
}
