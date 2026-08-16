package core

import (
	"reflect"
	"testing"
)

func TestNativeScreenDeltaRowsCarriesScrollAndReplacement(t *testing.T) {
	base := &nativeScreenBaseline{rowHashes: []uint64{10, 20, 30}}
	copies, replacements, ok := nativeScreenDeltaRows(base, []uint64{20, 30, 40})
	if !ok {
		t.Fatal("matching screen height should produce a delta")
	}
	wantCopies := []NativeScreenRowCopy{{SourceRow: 1, DestinationRow: 0, Count: 2}}
	if !reflect.DeepEqual(copies, wantCopies) || !reflect.DeepEqual(replacements, []int{2}) {
		t.Fatalf("delta rows copies=%#v replacements=%#v", copies, replacements)
	}
}

func TestNativeScreenDeltaRowsRejectsDifferentHeight(t *testing.T) {
	base := &nativeScreenBaseline{rowHashes: []uint64{10, 20}}
	if _, _, ok := nativeScreenDeltaRows(base, []uint64{10, 20, 30}); ok {
		t.Fatal("different screen height must require a full frame")
	}
}
