package main

import (
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// main's picker search is a case-insensitive subsequence, not a substring.
func TestSubsequenceSearchMatchesMain(t *testing.T) {
	if got := subsequenceIndexes("term-pool", "trpl"); len(got) != 4 ||
		got[0] != 0 || got[1] != 2 || got[2] != 5 || got[3] != 8 {
		t.Fatalf("subsequence term-pool/trpl = %v, want [0 2 5 8]", got)
	}
	if subsequenceIndexes("term-pool", "zzz") != nil {
		t.Fatal("non-subsequence must not match")
	}
	if subsequenceIndexes("TermPool", "tp") == nil {
		t.Fatal("subsequence must be case-insensitive")
	}
}

// A terminal titled with CJK is reachable from a Latin keyboard by full
// pinyin and by initials.
func TestPinyinSearch(t *testing.T) {
	if got := matchSearchValue("锁屏", "suoping"); len(got) == 0 {
		t.Fatalf("full pinyin suoping must match 锁屏, got %v", got)
	}
	if got := matchSearchValue("锁屏", "sp"); len(got) == 0 {
		t.Fatalf("pinyin initials sp must match 锁屏, got %v", got)
	}
	if matchSearchValue("锁屏", "ping") == nil {
		t.Fatal("partial pinyin must match")
	}
	if matchSearchValue("锁屏", "xyz") != nil {
		t.Fatal("unrelated query must not match")
	}
	// The pinyin match highlights the originating CJK rune(s).
	if got := matchSearchValue("锁屏", "sp"); len(got) != 2 {
		t.Fatalf("pinyin highlight indexes = %v, want the two CJK runes", got)
	}
}

func TestPickerSourceMatchesMetadataAndPinyin(t *testing.T) {
	src := &pb.Source{
		Id: "terminal:local:lock", Kind: "terminal", Title: "锁屏",
		Endpoint: "local", TerminalId: "lock", Attached: true,
		Cols: 211, Rows: 58, Tags: map[string]string{"tag1": "后端"},
	}
	for _, query := range []string{"suoping", "sp", "lock", "running", "x1", "211x58", "后端", "houduan"} {
		if !pickerSourceMatches(src, query) {
			t.Fatalf("query %q must match the source", query)
		}
	}
	if pickerSourceMatches(src, "zzzz") {
		t.Fatal("unrelated query must not match")
	}
}

// A pinyin match must highlight the originating CJK characters, not the pinyin
// letters (those are not displayed).
func TestPickerHighlightMarksCJKRunes(t *testing.T) {
	runs := pickerHighlightedRuns("锁屏", 8, "sp", stContent)
	var highlighted string
	for _, run := range runs {
		if run.style == stPickerMatch {
			highlighted += run.text
		}
	}
	if highlighted != "锁屏" {
		t.Fatalf("pinyin highlight = %q, want both CJK runes", highlighted)
	}
}
