package linehist

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

func TestBlockSearchBloomSupportsMixedUTF8AndRunBoundaries(t *testing.T) {
	lines := []Line{
		{Runs: []Run{{Text: "deploy /api/用户/详情 UUID=550e8400-e29b-41d4-a716-446655440000"}}, HardEnd: true},
		{Runs: []Run{{Text: "中文部署完成：数据库连接"}, {Text: "正常 ✅ 🚀"}}, HardEnd: true},
		{Runs: []Run{{Text: "日本語ログ العربية e\u0301"}}, HardEnd: true},
		{Runs: []Run{{Text: "跨"}, {Text: "样式匹配"}}, HardEnd: true},
	}
	bloom := buildBlockSearchBloom(lines)
	if len(bloom) != searchIndexBloomBytes {
		t.Fatalf("bloom bytes=%d, want %d", len(bloom), searchIndexBloomBytes)
	}
	for _, query := range []string{
		"中",
		"中文部署",
		"用户/详情",
		"550e8400-e29b",
		"✅",
		"🚀",
		"日本語",
		"العربية",
		"e\u0301",
		"跨样式",
	} {
		t.Run(query, func(t *testing.T) {
			grams := querySearchGrams(query)
			if len(grams.trigrams) == 0 {
				t.Fatalf("UTF-8 query %q unexpectedly has no byte trigrams", query)
			}
			if !blockSearchBloomMayContain(bloom, grams) {
				t.Fatalf("Bloom filter produced a false negative for %q", query)
			}
		})
	}
	if blockSearchBloomMayContain(bloom, querySearchGrams("完全不存在的关键字🧭")) {
		t.Fatal("chosen absent mixed UTF-8 query should be rejected by the block filter")
	}
	if blockSearchBloomMayContain(bloom, querySearchGrams("x")) {
		t.Fatal("the exact byte set should reject a missing one-byte query")
	}
	if blockSearchBloomMayContain(bloom, querySearchGrams("zz")) {
		t.Fatal("the exact bigram set should reject a missing two-byte query")
	}
	if !blockSearchBloomMayContain(bloom, querySearchGrams("/")) {
		t.Fatal("the exact byte set produced a false negative")
	}
}

func TestPersistedLineTextMatchesExactSearchProjection(t *testing.T) {
	lines := []Line{
		{Runs: []Run{{Text: "ASCII"}, {Text: " 中文"}, {Text: " e\u0301 🚀"}}, HardEnd: true},
		{Runs: []Run{{Text: "العربية"}, {Text: " 日本語"}}, HardEnd: true},
		{Runs: []Run{{Text: " "}, {Text: "\t"}, {Text: "界"}}, HardEnd: true},
	}
	for index, line := range lines {
		projected := historySearchText(cellsFromRuns(line.Runs))
		if plain := LineText(line); plain != projected {
			t.Fatalf("line %d plain=%q projected=%q", index, plain, projected)
		}
	}
}

func TestBlockSearchBloomBoundsHighEntropyInput(t *testing.T) {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_"
	var payload strings.Builder
	for value := 0; value <= searchIndexMaxUniqueTrigrams; value++ {
		payload.WriteByte(alphabet[(value>>12)&63])
		payload.WriteByte(alphabet[(value>>6)&63])
		payload.WriteByte(alphabet[value&63])
	}
	bloom := buildBlockSearchBloom([]Line{{Runs: []Run{{Text: payload.String()}}, HardEnd: true}})
	if len(bloom) != searchIndexBloomBytes {
		t.Fatalf("high-entropy block filter bytes=%d, want %d", len(bloom), searchIndexBloomBytes)
	}
	for index, value := range bloom[searchIndexBigramBloomEnd:] {
		if value != 0xff {
			t.Fatalf("saturated trigram byte %d=%02x, want ff", index, value)
		}
	}
	query := payload.String()[300:340]
	if !blockSearchBloomMayContain(bloom, querySearchGrams(query)) {
		t.Fatalf("saturated filter produced a false negative for %q", query)
	}
}

func TestCompactSearchFilterFallsBackForFrequentRepeatedTrigram(t *testing.T) {
	line := strings.Repeat("000x", 20_000)
	bloom := buildBlockSearchBloom([]Line{{Runs: []Run{{Text: line}}, HardEnd: true}})
	grams := querySearchGrams("000000000")
	if !trigramBloomMayContain(bloom[searchIndexBigramBloomEnd:], grams.trigrams) {
		t.Fatal("test setup requires the common trigram layer to be positive")
	}
	if !blockSearchBloomMayContain(bloom, grams) {
		t.Fatal("the compact filter must fall back when all query trigrams are common")
	}
	if _, _, found := findHistoryTextMatch(line, "000000000", false, 0, -1); found {
		t.Fatal("exact fallback unexpectedly found nine consecutive zeroes")
	}

	present := buildBlockSearchBloom([]Line{{Runs: []Run{{Text: line + " 000000000"}}, HardEnd: true}})
	if !blockSearchBloomMayContain(present, grams) {
		t.Fatal("the compact filter produced a false negative for a repeated query")
	}
}

func TestCompressedLineFilePersistsSearchIndexAndFindsMixedUTF8(t *testing.T) {
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "search-index", CompressedLineFileOptions{Compression: compressionZstd})
	if err != nil {
		t.Fatal(err)
	}
	lines := mixedSearchIndexLines(12_000)
	if err := file.AppendLines(lines); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	if len(file.blocks) < 3 {
		t.Fatalf("test needs multiple physical blocks, got %d", len(file.blocks))
	}
	for index, block := range file.blocks {
		if !block.searchBloomReady || !block.searchBloomPersisted || len(block.searchBloom) != searchIndexBloomBytes {
			t.Fatalf("block %d search index state=%#v", index, block)
		}
	}
	indexInfo, err := os.Stat(searchIndexPath(file.Path()))
	if err != nil {
		t.Fatal(err)
	}
	wantIndexBytes := int64(len(file.blocks) * (searchIndexHeaderSize + searchIndexBloomBytes))
	if indexInfo.Size() != wantIndexBytes {
		t.Fatalf("search index bytes=%d, want %d", indexInfo.Size(), wantIndexBytes)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenCompressedLineFile(dir, "search-index", CompressedLineFileOptions{Compression: compressionZstd})
	if err != nil {
		t.Fatal(err)
	}
	for index, block := range reopened.blocks {
		if !block.searchBloomReady || !block.searchBloomPersisted {
			t.Fatalf("reopened block %d did not recover its filter", index)
		}
	}
	store := NewStore("search-index", NewEngine(reopened))
	t.Cleanup(func() { _ = store.Close() })
	frozen, err := store.Freeze(history.FreezeHistoryRequest{Cols: 160, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"中文标记-07321-🚀", "用户/7321/详情", "✅", "日"} {
		result, err := store.Search(context.Background(), history.HistorySearchRequest{
			Token: frozen.Token, Cols: 160, Limit: 100, Query: query, Direction: history.HistorySearchForward,
		})
		if err != nil {
			t.Fatalf("search %q: %v", query, err)
		}
		if !result.Found {
			t.Fatalf("mixed UTF-8 query %q was not found", query)
		}
	}
	missing, err := store.Search(context.Background(), history.HistorySearchRequest{
		Token: frozen.Token, Cols: 160, Limit: 100,
		Query: "完全不存在-🧭-not-present", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Found {
		t.Fatalf("missing query unexpectedly found: %#v", missing)
	}
}

func TestCompressedLineFileCorruptSearchIndexFallsBackAndRebuilds(t *testing.T) {
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "search-index-corrupt", CompressedLineFileOptions{Compression: compressionZstd})
	if err != nil {
		t.Fatal(err)
	}
	if err := file.AppendLines(mixedSearchIndexLines(4000)); err != nil {
		t.Fatal(err)
	}
	path := file.Path()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	index, err := os.OpenFile(searchIndexPath(path), os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	var value [1]byte
	if _, err := index.ReadAt(value[:], searchIndexHeaderSize); err != nil {
		_ = index.Close()
		t.Fatal(err)
	}
	value[0] ^= 0xff
	if _, err := index.WriteAt(value[:], searchIndexHeaderSize); err != nil {
		_ = index.Close()
		t.Fatal(err)
	}
	if err := index.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenCompressedLineFile(dir, "search-index-corrupt", CompressedLineFileOptions{Compression: compressionZstd})
	if err != nil {
		t.Fatalf("a corrupt optional index must not block history: %v", err)
	}
	if reopened.blocks[0].searchBloomReady {
		t.Fatal("corrupt filter should have been discarded")
	}
	store := NewStore("search-index-corrupt", NewEngine(reopened))
	t.Cleanup(func() { _ = store.Close() })
	frozen, err := store.Freeze(history.FreezeHistoryRequest{Cols: 160, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Search(context.Background(), history.HistorySearchRequest{
		Token: frozen.Token, Cols: 160, Limit: 100,
		Query: "中文标记-01234-🚀", Direction: history.HistorySearchForward,
	})
	if err != nil || !result.Found {
		t.Fatalf("fallback search result=%#v err=%v", result, err)
	}
	rebuilt := false
	for _, block := range reopened.blocks {
		rebuilt = rebuilt || block.searchBloomReady && block.searchBloomPersisted
	}
	if !rebuilt {
		t.Fatal("fallback scan did not rebuild and persist the missing filter")
	}
}

func TestCompressedLineFileRetentionBoundsMainAndSearchIndex(t *testing.T) {
	const maxBytes = int64(500 * 1024)
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "search-index-retention", CompressedLineFileOptions{
		MaxBytes: maxBytes, Compression: compressionNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := make([]Line, 6000)
	for index := range lines {
		lines[index] = Line{Runs: []Run{{Text: fmt.Sprintf("保留测试-%05d-%s", index, strings.Repeat("内容", 80))}}, HardEnd: true}
	}
	if err := file.AppendLines(lines); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	mainInfo, err := os.Stat(file.Path())
	if err != nil {
		t.Fatal(err)
	}
	indexInfo, err := os.Stat(searchIndexPath(file.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if mainInfo.Size()+indexInfo.Size() > maxBytes {
		t.Fatalf("history main+index=%d exceeds max=%d", mainInfo.Size()+indexInfo.Size(), maxBytes)
	}
	if file.LineCount() >= len(lines) {
		t.Fatal("test setup did not trigger retention")
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenCompressedLineFile(dir, "search-index-retention", CompressedLineFileOptions{
		MaxBytes: maxBytes, Compression: compressionNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	for index, block := range reopened.blocks {
		if !block.searchBloomPersisted {
			t.Fatalf("retained block %d lost its search filter", index)
		}
	}
	last, err := reopened.Lines(reopened.LineCount()-1, reopened.LineCount())
	if err != nil || len(last) != 1 || !strings.Contains(LineText(last[0]), "05999") {
		t.Fatalf("retention lost newest line: %#v err=%v", last, err)
	}
}

func TestLazySearchIndexBuildRespectsPhysicalLimit(t *testing.T) {
	const maxBytes = int64(420 * 1024)
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "search-index-lazy-limit", CompressedLineFileOptions{
		MaxBytes: maxBytes, Compression: compressionNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := make([]Line, 5000)
	for index := range lines {
		lines[index] = Line{Runs: []Run{{Text: fmt.Sprintf("懒索引-%05d-%s", index, strings.Repeat("数据", 70))}}, HardEnd: true}
	}
	if err := file.AppendLines(lines); err != nil {
		t.Fatal(err)
	}
	path := file.Path()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(searchIndexPath(path)); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenCompressedLineFile(dir, "search-index-lazy-limit", CompressedLineFileOptions{
		MaxBytes: maxBytes, Compression: compressionNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore("search-index-lazy-limit", NewEngine(reopened))
	t.Cleanup(func() { _ = store.Close() })
	frozen, err := store.Freeze(history.FreezeHistoryRequest{Cols: 120, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Search(context.Background(), history.HistorySearchRequest{
		Token: frozen.Token, Cols: 120, Limit: 20,
		Query: "不存在的懒索引查询🧭", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatal(err)
	}
	mainInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	indexInfo, err := os.Stat(searchIndexPath(path))
	if err != nil {
		t.Fatal(err)
	}
	if mainInfo.Size()+indexInfo.Size() > maxBytes {
		t.Fatalf("lazy index pushed physical bytes to %d, max=%d", mainInfo.Size()+indexInfo.Size(), maxBytes)
	}
}

func mixedSearchIndexLines(count int) []Line {
	lines := make([]Line, count)
	for index := range lines {
		lines[index] = Line{
			Runs: []Run{
				{Text: fmt.Sprintf("2026-08-21T08:00:00Z INFO seq=%05d 中文标记-%05d-", index, index)},
				{Text: fmt.Sprintf("🚀 用户/%d/详情 UUID=%08x-%04x ✅ 日本語 العربية", index, index*2654435761, index%65536)},
			},
			HardEnd: true,
		}
	}
	return lines
}

var benchmarkSearchBloom []byte

func BenchmarkBuildBlockSearchBloom(b *testing.B) {
	lines := mixedSearchIndexLines(2_000)
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		benchmarkSearchBloom = buildBlockSearchBloom(lines)
	}
}
