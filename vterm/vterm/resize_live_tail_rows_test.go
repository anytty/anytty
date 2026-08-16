package vterm

import (
	"reflect"
	"strings"
	"testing"
)

func TestVTermResizeWithDamageExportsResizeLiveTailRows(t *testing.T) {
	vt := New(5, 2, 0, nil)
	vt.DisableEmulatorScrollback()
	if _, err := vt.Write([]byte("abcdefghij")); err != nil {
		t.Fatalf("write wrapped line: %v", err)
	}

	damage := vt.ResizeWithDamage(4, 2)
	if !damage.RequiresFullReplace || damage.FullReplaceReason != "resize" {
		t.Fatalf("expected resize full-replace damage, got %#v", damage)
	}
	if got := resizeLiveTailRowsFromDamageForTest(t, damage); got != 0 {
		t.Fatalf("expected displaced wrapped prefix without open tail to stay persisted-history-owned, got ResizeLiveTailRows=%d", got)
	}

	gotRows := append(damageRowsText(damage.ScrollbackAppend), screenRowsText(vt.ScreenContent().Cells)...)
	if strings.Contains(strings.Join(gotRows, "|"), "ij|ij") {
		t.Fatalf("visible suffix duplicated in resize damage plus screen, got %#v", gotRows)
	}
	if gotDamageRows := damageRowsText(damage.ScrollbackAppend); !reflect.DeepEqual(gotDamageRows, []string{"abcd"}) {
		t.Fatalf("expected only displaced rows in damage, got %#v screen=%#v", gotDamageRows, trimmedScreenRowsText(vt.ScreenContent().Cells))
	}
}

func TestVTermResizeWithDamageMarksOnlyOpenTailRowsLiveTailOwned(t *testing.T) {
	vt := New(4, 1, 0, nil)
	vt.DisableEmulatorScrollback()
	if _, err := vt.Write([]byte("abcd")); err != nil {
		t.Fatalf("write first segment: %v", err)
	}
	if _, err := vt.Write([]byte("efgh")); err != nil {
		t.Fatalf("write second segment: %v", err)
	}
	if _, err := vt.Write([]byte("ij")); err != nil {
		t.Fatalf("write tail segment: %v", err)
	}

	damage := vt.ResizeWithDamage(1, 1)
	if !damage.RequiresFullReplace || damage.FullReplaceReason != "resize" {
		t.Fatalf("expected resize full-replace damage, got %#v", damage)
	}
	if got := resizeLiveTailRowsFromDamageForTest(t, damage); got != 1 {
		t.Fatalf("expected only the displaced row from the visible open tail to stay live-tail-owned, got %d", got)
	}
	if gotDamageRows := damageRowsText(damage.ScrollbackAppend); !reflect.DeepEqual(gotDamageRows, []string{"i"}) {
		t.Fatalf("expected resize damage to include only the displaced visible tail row, got %#v", gotDamageRows)
	}
}

func resizeLiveTailRowsFromDamageForTest(t *testing.T, damage WriteDamage) int {
	t.Helper()
	value := reflect.ValueOf(damage)
	field := value.FieldByName("ResizeLiveTailRows")
	if !field.IsValid() {
		t.Fatal("expected WriteDamage.ResizeLiveTailRows field")
	}
	return int(field.Int())
}
