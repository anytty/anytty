package protocoladapter

import (
	"reflect"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
)

func TestLiveSurfaceSnapshotFromProtoPreservesSparseRows(t *testing.T) {
	snapshot := liveSurfaceSnapshotFromProto("term-1", &apipb.NativeScreenResult{
		LiveRevision: 10,
		BaseRevision: 7,
		Size:         &apipb.TerminalSize{Cols: 80, Rows: 24},
		RowReplacements: []*apipb.ScreenRowReplace{
			{RowIndex: 3, Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: "three", Width: 5}}}},
			{RowIndex: 9, Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: "nine", Width: 4}}}},
		},
	})

	if snapshot.TerminalID != "term-1" || snapshot.BaseRevision != 7 || snapshot.Revision != 10 || snapshot.FullReplace {
		t.Fatalf("sparse snapshot metadata lost: %#v", snapshot)
	}
	if !reflect.DeepEqual(snapshot.ChangedRows, []int{3, 9}) || len(snapshot.Screen) != 2 {
		t.Fatalf("sparse row projection lost: %#v", snapshot)
	}
	if snapshot.Screen[0][0].Text != "three" || snapshot.Screen[1][0].Text != "nine" {
		t.Fatalf("sparse row cells lost: %#v", snapshot.Screen)
	}
}
