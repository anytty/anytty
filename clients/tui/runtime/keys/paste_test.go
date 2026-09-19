package keys

import (
	"strconv"
	"strings"
	"testing"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a\r\nb\r\n", "a\nb\n"},
		{"a\rb", "a\nb"},
		{"a\nb", "a\nb"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := NormalizeText(tc.in); got != tc.want {
			t.Fatalf("NormalizeText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEncodePaste(t *testing.T) {
	if got := EncodePaste("one\ntwo", false); string(got) != "one\rtwo" {
		t.Fatalf("plain paste = %q, want one\\rtwo", got)
	}
	if got := EncodePaste("one\ntwo", true); string(got) != "\x1b[200~one\rtwo\x1b[201~" {
		t.Fatalf("bracketed paste = %q", got)
	}
	if got := EncodePaste("", true); string(got) != "\x1b[200~\x1b[201~" {
		t.Fatalf("empty bracketed paste = %q", got)
	}
}

func chunkIDs() func() string {
	n := 0
	return func() string {
		n++
		return "ev-" + strconv.Itoa(n)
	}
}

func TestChunkPasteSplitsAndPreservesOrder(t *testing.T) {
	text := "abcdefghij"
	chunks := ChunkPaste(text, 4, false, chunkIDs())
	if len(chunks) != 3 {
		t.Fatalf("chunks = %d, want 3", len(chunks))
	}
	for i, want := range []struct {
		id   string
		text string
	}{
		{"ev-1", "abcd"},
		{"ev-2", "efgh"},
		{"ev-3", "ij"},
	} {
		if chunks[i].ID != want.id || chunks[i].Text != want.text {
			t.Fatalf("chunk %d = %+v, want id=%s text=%q", i, chunks[i], want.id, want.text)
		}
		wantBytes := EncodePaste(want.text, false)
		if string(chunks[i].Bytes) != string(wantBytes) {
			t.Fatalf("chunk %d bytes = %q, want %q", i, chunks[i].Bytes, wantBytes)
		}
	}
	if joined := chunks[0].Text + chunks[1].Text + chunks[2].Text; joined != text {
		t.Fatalf("chunks merged wrong: %q", joined)
	}
}

func TestChunkPasteKeepsRunesIntact(t *testing.T) {
	chunks := ChunkPaste("中文中", 4, true, chunkIDs())
	if len(chunks) != 3 {
		t.Fatalf("chunks = %d, want 3 (never split a rune)", len(chunks))
	}
	for i, want := range []string{"中", "文", "中"} {
		if chunks[i].Text != want {
			t.Fatalf("chunk %d = %q, want %q", i, chunks[i].Text, want)
		}
		if !strings.HasPrefix(string(chunks[i].Bytes), BracketStart) || !strings.HasSuffix(string(chunks[i].Bytes), BracketEnd) {
			t.Fatalf("chunk %d is not independently bracketed: %q", i, chunks[i].Bytes)
		}
	}
}

func TestChunkPasteCRLFAndSingleChunk(t *testing.T) {
	chunks := ChunkPaste("a\r\nb\rc", 0, false, chunkIDs())
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want 1 without a byte limit", len(chunks))
	}
	if chunks[0].Text != "a\nb\nc" {
		t.Fatalf("normalized text = %q, want a\\nb\\nc", chunks[0].Text)
	}
	if string(chunks[0].Bytes) != "a\rb\rc" {
		t.Fatalf("encoded bytes = %q, want a\\rb\\rc", chunks[0].Bytes)
	}

	if chunks := ChunkPaste("", 8, false, chunkIDs()); len(chunks) != 0 {
		t.Fatalf("empty paste produced %d chunks, want 0", len(chunks))
	}
}
