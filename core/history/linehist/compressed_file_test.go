package linehist

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/core/history"
)

func TestCompressedLineFileRoundTripReopenAndCompress(t *testing.T) {
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "compressed", CompressedLineFileOptions{Compression: compressionZstd})
	if err != nil {
		t.Fatal(err)
	}
	lines := make([]Line, 8000)
	for i := range lines {
		lines[i] = Line{
			Runs:    []Run{{Text: fmt.Sprintf("build step %05d repeated terminal payload repeated terminal payload repeated terminal payload", i)}},
			HardEnd: true,
		}
	}
	if err := file.AppendLines(lines); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(file.Path())
	if err != nil {
		t.Fatal(err)
	}
	rawSize := compactLinesEncodedSize(lines)
	if info.Size() >= int64(rawSize/4) {
		t.Fatalf("compressed file = %d bytes, compact raw = %d bytes", info.Size(), rawSize)
	}
	page, err := file.Lines(3997, 4003)
	if err != nil {
		t.Fatal(err)
	}
	if got := LineText(page[0]); !strings.Contains(got, "03997") {
		t.Fatalf("unexpected middle page %q", got)
	}
	var reverseIndexes []int
	if err := file.VisitLines(3997, 4003, true, func(index int, _ Line) bool {
		reverseIndexes = append(reverseIndexes, index)
		return len(reverseIndexes) < 3
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reverseIndexes, []int{4002, 4001, 4000}) {
		t.Fatalf("reverse visit indexes=%v", reverseIndexes)
	}
	path := file.Path()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".idx"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("compressed block storage must not create per-line sidecar: %v", err)
	}

	reopened, err := OpenCompressedLineFile(dir, "compressed", CompressedLineFileOptions{Compression: compressionZstd})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.LineCount() != len(lines) {
		t.Fatalf("recovered line count = %d, want %d", reopened.LineCount(), len(lines))
	}
	last, err := reopened.Lines(len(lines)-1, len(lines))
	if err != nil || len(last) != 1 || !strings.Contains(LineText(last[0]), "07999") {
		t.Fatalf("recovered last line = %#v, err=%v", last, err)
	}
}

func TestCompressedLineFileRoundTripPreservesLineMetadata(t *testing.T) {
	file, err := OpenCompressedLineFile(t.TempDir(), "metadata", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	want := []Line{
		{
			Runs: []Run{
				{Text: "plain "},
				{Text: "styled", Style: history.CellStyle{FG: "1", BG: "2", Bold: true, Italic: true, Underline: true, Blink: true, Reverse: true, Strikethrough: true}},
				{Text: "link", LinkURL: "https://example.com", LinkParams: "id=9"},
			},
			HardEnd:   true,
			UpdatedAt: time.Date(2026, 8, 1, 9, 30, 2, 123, time.UTC),
		},
		{HardEnd: true},
		{Runs: []Run{{Text: "chunk"}}, HardEnd: false},
		{Runs: []Run{{Text: "你好世界"}}, HardEnd: true},
	}
	if err := file.AppendLines(want); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	got, err := file.Lines(0, file.LineCount())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata round trip = %#v, want %#v", got, want)
	}
}

func TestCompressedLineFileTruncatesPartialTailOnReopen(t *testing.T) {
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "partial-tail", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	if err := file.AppendLines([]Line{
		{Runs: []Run{{Text: "one"}}, HardEnd: true},
		{Runs: []Run{{Text: "two"}}, HardEnd: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := file.AppendBoundary(); err != nil {
		t.Fatal(err)
	}
	path := file.Path()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	validInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	partial := makeCompressedBlockHeader(compressedBlockKindLines, compressedBlockCodecRaw, 1, 32, 32, 0, time.Now().Unix())[:10]
	tail, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tail.Write(partial); err != nil {
		_ = tail.Close()
		t.Fatal(err)
	}
	if err := tail.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenCompressedLineFile(dir, "partial-tail", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.LineCount() != 2 {
		t.Fatalf("recovered line count = %d, want 2", reopened.LineCount())
	}
	recoveredInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if recoveredInfo.Size() != validInfo.Size() {
		t.Fatalf("recovered file size = %d, want %d", recoveredInfo.Size(), validInfo.Size())
	}
	if err := reopened.AppendLines([]Line{{Runs: []Run{{Text: "three"}}, HardEnd: true}}); err != nil {
		t.Fatal(err)
	}
	if err := reopened.Sync(); err != nil {
		t.Fatal(err)
	}
	last, err := reopened.Lines(2, 3)
	if err != nil || len(last) != 1 || LineText(last[0]) != "three" {
		t.Fatalf("append after recovery = %#v, err=%v", last, err)
	}
}

func TestCompressedLineFileDetectsCorruptBlockPayload(t *testing.T) {
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "corrupt", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	if err := file.AppendLines([]Line{{Runs: []Run{{Text: "checksum payload"}}, HardEnd: true}}); err != nil {
		t.Fatal(err)
	}
	path := file.Path()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	payload, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	var value [1]byte
	if _, err := payload.ReadAt(value[:], compressedBlockHeaderSize); err != nil {
		_ = payload.Close()
		t.Fatal(err)
	}
	value[0] ^= 0xff
	if _, err := payload.WriteAt(value[:], compressedBlockHeaderSize); err != nil {
		_ = payload.Close()
		t.Fatal(err)
	}
	if err := payload.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenCompressedLineFile(dir, "corrupt", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err := reopened.Lines(0, 1); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("corrupt block read error = %v", err)
	}
}

func TestCompressedLineFileSupportsAlgorithmsLevelsAndMixedBlocks(t *testing.T) {
	dir := t.TempDir()
	terminalID := "mixed-codecs"
	linesFor := func(prefix string) []Line {
		lines := make([]Line, 2500)
		for i := range lines {
			lines[i] = Line{Runs: []Run{{Text: fmt.Sprintf("%s-%05d repeated terminal payload repeated terminal payload", prefix, i)}}, HardEnd: true}
		}
		return lines
	}
	write := func(algorithm string, level string, prefix string) {
		t.Helper()
		file, err := OpenCompressedLineFile(dir, terminalID, CompressedLineFileOptions{
			Compression:      algorithm,
			CompressionLevel: level,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := file.AppendLines(linesFor(prefix)); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}

	write(compressionZstd, compressionLevelBest, "zstd")
	write(compressionS2, compressionLevelBalanced, "s2")
	reopened, err := OpenCompressedLineFile(dir, terminalID, CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.LineCount() != 5000 {
		t.Fatalf("mixed codec line count = %d", reopened.LineCount())
	}
	for _, check := range []struct {
		index int
		want  string
	}{{0, "zstd-00000"}, {4999, "s2-02499"}} {
		lines, err := reopened.Lines(check.index, check.index+1)
		if err != nil || len(lines) != 1 || !strings.Contains(LineText(lines[0]), check.want) {
			t.Fatalf("mixed codec line %d = %#v, err=%v", check.index, lines, err)
		}
	}
}

func TestCompressedLineFileCompressionProfilesRoundTrip(t *testing.T) {
	lines := make([]Line, 1500)
	for i := range lines {
		lines[i] = Line{Runs: []Run{{Text: fmt.Sprintf("profile-%05d repeated terminal payload repeated terminal payload", i)}}, HardEnd: true}
	}
	for _, item := range []struct {
		algorithm string
		level     string
		codec     uint8
	}{
		{compressionNone, compressionLevelFast, compressedBlockCodecRaw},
		{compressionS2, compressionLevelFast, compressedBlockCodecS2},
		{compressionS2, compressionLevelBalanced, compressedBlockCodecS2},
		{compressionS2, compressionLevelBest, compressedBlockCodecS2},
		{compressionZstd, compressionLevelFast, compressedBlockCodecZstd},
		{compressionZstd, compressionLevelBalanced, compressedBlockCodecZstd},
		{compressionZstd, compressionLevelBest, compressedBlockCodecZstd},
	} {
		t.Run(item.algorithm+"/"+item.level, func(t *testing.T) {
			file, err := OpenCompressedLineFile(t.TempDir(), "profile", CompressedLineFileOptions{
				Compression:      item.algorithm,
				CompressionLevel: item.level,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if err := file.AppendLines(lines); err != nil {
				t.Fatal(err)
			}
			if err := file.Sync(); err != nil {
				t.Fatal(err)
			}
			if len(file.blocks) == 0 || file.blocks[0].codec != item.codec {
				t.Fatalf("block codec = %#v, want %d", file.blocks, item.codec)
			}
			page, err := file.Lines(749, 751)
			if err != nil || len(page) != 2 || !strings.Contains(LineText(page[0]), "profile-00749") {
				t.Fatalf("profile round trip = %#v, err=%v", page, err)
			}
		})
	}
}

func TestCompressedLineFileRejectsUnknownCompressionOptions(t *testing.T) {
	if _, err := OpenCompressedLineFile(t.TempDir(), "bad-algorithm", CompressedLineFileOptions{Compression: "gzip"}); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("unknown algorithm error = %v", err)
	}
	if _, err := OpenCompressedLineFile(t.TempDir(), "bad-level", CompressedLineFileOptions{Compression: compressionZstd, CompressionLevel: "maximum"}); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("unknown level error = %v", err)
	}
}

var benchmarkCompressedBlock []byte

func BenchmarkHistoryCompressionProfiles(b *testing.B) {
	lines := make([]Line, 4000)
	for i := range lines {
		lines[i] = Line{Runs: []Run{{Text: fmt.Sprintf("build step %05d repeated terminal payload repeated terminal payload", i)}}, HardEnd: true}
	}
	raw := encodeCompactLines(lines)
	if path := strings.TrimSpace(os.Getenv("ANYTTY_HISTORY_BENCH_FILE")); path != "" {
		file, err := os.Open(path)
		if err != nil {
			b.Fatal(err)
		}
		defer file.Close()
		raw = make([]byte, compressedBlockTargetBytes)
		n, err := io.ReadFull(file, raw)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			b.Fatal(err)
		}
		raw = raw[:n]
	}
	for _, item := range []struct {
		algorithm string
		level     string
	}{
		{compressionNone, compressionLevelFast},
		{compressionS2, compressionLevelFast},
		{compressionS2, compressionLevelBalanced},
		{compressionS2, compressionLevelBest},
		{compressionZstd, compressionLevelFast},
		{compressionZstd, compressionLevelBalanced},
		{compressionZstd, compressionLevelBest},
	} {
		b.Run(item.algorithm+"/"+item.level, func(b *testing.B) {
			file, err := OpenCompressedLineFile(b.TempDir(), "benchmark", CompressedLineFileOptions{
				Compression:      item.algorithm,
				CompressionLevel: item.level,
			})
			if err != nil {
				b.Fatal(err)
			}
			defer file.Close()
			b.SetBytes(int64(len(raw)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, benchmarkCompressedBlock = file.compressBlock(raw)
			}
			b.StopTimer()
			_, sample := file.compressBlock(raw)
			b.ReportMetric(float64(len(sample))/float64(len(raw)), "ratio")
		})
	}
}

func TestCompressedLineFileEnforcesPhysicalLimitAndKeepsNewestLines(t *testing.T) {
	const maxBytes = int64(400 * 1024)
	file, err := OpenCompressedLineFile(t.TempDir(), "bounded", CompressedLineFileOptions{
		MaxBytes:    maxBytes,
		Compression: compressionNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	const total = 7000
	lines := make([]Line, total)
	for i := range lines {
		lines[i] = Line{
			Runs:    []Run{{Text: fmt.Sprintf("line-%05d-%s", i, strings.Repeat(string(rune('a'+i%26)), 220))}},
			HardEnd: true,
		}
	}
	if err := file.AppendLines(lines); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(file.Path())
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > maxBytes {
		t.Fatalf("bounded history size = %d, limit = %d", info.Size(), maxBytes)
	}
	if file.LineCount() <= 0 || file.LineCount() >= total {
		t.Fatalf("retained line count = %d, total = %d", file.LineCount(), total)
	}
	if file.RetentionEpoch() == 0 {
		t.Fatal("oldest-block eviction did not advance retention epoch")
	}
	last, err := file.Lines(file.LineCount()-1, file.LineCount())
	if err != nil || len(last) != 1 || !strings.Contains(LineText(last[0]), "line-06999-") {
		t.Fatalf("retention did not keep newest line: %#v, err=%v", last, err)
	}
}

func TestCompressedLineFilePrunesBlocksOlderThanMaxAge(t *testing.T) {
	now := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	file, err := OpenCompressedLineFile(t.TempDir(), "age-bounded", CompressedLineFileOptions{
		MaxAge:      48 * time.Hour,
		Compression: compressionNone,
		now:         func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	appendBatch := func(prefix string) {
		t.Helper()
		lines := make([]Line, 200)
		for i := range lines {
			lines[i] = Line{Runs: []Run{{Text: fmt.Sprintf("%s-%03d", prefix, i)}}, HardEnd: true}
		}
		if err := file.AppendLines(lines); err != nil {
			t.Fatal(err)
		}
		if err := file.Sync(); err != nil {
			t.Fatal(err)
		}
	}

	appendBatch("old")
	now = now.Add(24 * time.Hour)
	appendBatch("middle")
	now = now.Add(48 * time.Hour)
	appendBatch("new")

	if file.LineCount() != 400 {
		t.Fatalf("retained line count = %d, want 400", file.LineCount())
	}
	first, err := file.Lines(0, 1)
	if err != nil || len(first) != 1 || LineText(first[0]) != "middle-000" {
		t.Fatalf("oldest retained line = %#v, err=%v", first, err)
	}
	if file.RetentionEpoch() == 0 {
		t.Fatal("age eviction did not advance retention epoch")
	}
}

func writeNonCurrentHistoryFile(t *testing.T, dir string, terminalID string, payload string) string {
	t.Helper()
	path, err := lineFilePath(dir, terminalID)
	if err != nil {
		t.Fatal(err)
	}
	legacyHeader := []byte{0x4c, 0x4c, 0x58, 0x54, 0x01, 0x00}
	if err := os.WriteFile(path, append(legacyHeader, payload...), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCompressedLineFileDiscardsNonCurrentFormatWithoutMigration(t *testing.T) {
	dir := t.TempDir()
	path := writeNonCurrentHistoryFile(t, dir, "no-compat", "must-not-migrate")
	base := strings.TrimSuffix(path, ".logical-lines.bin")
	obsolete := []string{
		path + ".idx",
		base + ".history-lines.bin",
		base + ".screen-rows.bin",
		path + ".rows.24.idx",
	}
	for _, item := range obsolete[1:] {
		if err := os.WriteFile(item, []byte("obsolete"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	current, err := OpenCompressedLineFile(dir, "no-compat", CompressedLineFileOptions{Compression: compressionZstd})
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	if current.LineCount() != 0 {
		t.Fatalf("old format was migrated, line count = %d", current.LineCount())
	}
	for _, item := range obsolete {
		if _, err := os.Stat(item); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("obsolete history remains at %s: %v", filepath.Base(item), err)
		}
	}
}

func TestCompressedLineFileDiscardsOtherBlockVersionsWithoutCompatibility(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old-version.logical-lines.bin")
	header := makeCompressedBlockHeader(compressedBlockKindBoundary, compressedBlockCodecRaw, 0, 0, 0, 0, time.Now().Unix())
	binary.LittleEndian.PutUint16(header[4:6], compressedBlockVersion-1)
	if err := os.WriteFile(path, header, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := OpenCompressedLineFile(dir, "old-version", CompressedLineFileOptions{Compression: compressionZstd})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if file.LineCount() != 0 {
		t.Fatalf("old block version was retained: %d lines", file.LineCount())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatalf("old block version payload remains: %d bytes", info.Size())
	}
}

func TestPrepareDirectoryDiscardsInactiveOldHistoryAndBoundsCurrentFiles(t *testing.T) {
	dir := t.TempDir()
	legacyPath := writeNonCurrentHistoryFile(t, dir, "inactive-old", "old")
	orphan := filepath.Join(dir, "inactive-orphan.history-lines.bin")
	if err := os.WriteFile(orphan, []byte("old orphan"), 0o600); err != nil {
		t.Fatal(err)
	}

	current, err := OpenCompressedLineFile(dir, "inactive-current", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	lines := make([]Line, 5000)
	for i := range lines {
		lines[i] = Line{Runs: []Run{{Text: fmt.Sprintf("current-%05d-%s", i, strings.Repeat("z", 220))}}, HardEnd: true}
	}
	if err := current.AppendLines(lines); err != nil {
		t.Fatal(err)
	}
	if err := current.Close(); err != nil {
		t.Fatal(err)
	}

	const maxBytes = int64(400 * 1024)
	if err := PrepareDirectory(dir, CompressedLineFileOptions{MaxBytes: maxBytes, Compression: compressionNone}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inactive obsolete file remains: %v", err)
	}
	legacyInfo, err := os.Stat(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if legacyInfo.Size() != 0 {
		t.Fatalf("inactive legacy payload was retained: %d bytes", legacyInfo.Size())
	}
	currentInfo, err := os.Stat(filepath.Join(dir, "inactive-current.logical-lines.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if currentInfo.Size() > maxBytes {
		t.Fatalf("inactive current file = %d bytes, max = %d", currentInfo.Size(), maxBytes)
	}
}

func TestDeleteTerminalAndAllHistoryFiles(t *testing.T) {
	dir := t.TempDir()
	for _, terminalID := range []string{"one", "two"} {
		file, err := OpenCompressedLineFile(dir, terminalID, CompressedLineFileOptions{Compression: compressionZstd})
		if err != nil {
			t.Fatal(err)
		}
		if err := file.AppendLines([]Line{{Runs: []Run{{Text: terminalID}}, HardEnd: true}}); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := DeleteTerminalHistory(dir, "one")
	if err != nil || removed != 1 {
		t.Fatalf("delete one removed=%d err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "one.logical-lines.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("terminal history remains: %v", err)
	}
	removed, err = DeleteAllHistory(dir)
	if err != nil || removed != 1 {
		t.Fatalf("delete all removed=%d err=%v", removed, err)
	}
}

func TestDeleteObsoleteCompactHistoryPreservesUnknownFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"one.compact", "two.compact", "keep.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := DeleteObsoleteCompactHistory(dir)
	if err != nil || removed != 2 {
		t.Fatalf("obsolete compact cleanup removed=%d err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Fatalf("unknown file was not preserved: %v", err)
	}
}

func TestStoreInvalidatesFrozenTokenWhenRetentionEvicts(t *testing.T) {
	file, err := OpenCompressedLineFile(t.TempDir(), "freeze-retention", CompressedLineFileOptions{
		MaxBytes:    400 * 1024,
		Compression: compressionNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore("freeze-retention", NewEngine(file))
	defer store.Close()
	batch := make([]string, 3500)
	for i := range batch {
		batch[i] = fmt.Sprintf("before-%05d-%s", i, strings.Repeat("x", 220))
	}
	if err := store.AppendLifecycleLines(batch); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Freeze(history.FreezeHistoryRequest{TerminalID: "freeze-retention", Cols: 80, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	_, frozenView, err := store.viewForRequest(history.HistoryWindowRequest{Token: snapshot.Token})
	if err != nil {
		t.Fatal(err)
	}
	for i := range batch {
		batch[i] = fmt.Sprintf("after-%05d-%s", i, strings.Repeat("y", 220))
	}
	if err := store.AppendLifecycleLines(batch); err != nil {
		t.Fatal(err)
	}
	if _, err := store.coldLogicalRowsForLineRange(frozenView.coldBase, frozenView.coldCount, frozenView.retention, 0, 1); !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("retention race error = %v, want stale window", err)
	}
	_, err = store.LatestWindow(history.HistoryWindowRequest{TerminalID: "freeze-retention", Token: snapshot.Token, Cols: 80, Limit: 10})
	if !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("evicted frozen token error = %v", err)
	}
}
