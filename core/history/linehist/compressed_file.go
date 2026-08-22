package linehist

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anytty/anytty/shared/filepublish"
	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zstd"
)

const (
	compressedBlockMagic           uint32 = 0x424C5854 // "TXLB"
	compressedBlockVersion         uint16 = 3
	compressedBlockHeaderSize             = 32
	compressedBlockTargetBytes            = 256 * 1024
	compressedBlockMaxRawBytes            = 2 * 1024 * 1024
	compressedRetentionNumerator          = 3
	compressedRetentionDenominator        = 4

	compressedBlockKindLines    uint8 = 1
	compressedBlockKindBoundary uint8 = 2
	compressedBlockKindGap      uint8 = 3
	compressedBlockCodecRaw     uint8 = 0
	compressedBlockCodecZstd    uint8 = 1
	compressedBlockCodecS2      uint8 = 2

	compressionNone = "none"
	compressionZstd = "zstd"
	compressionS2   = "s2"

	compressionLevelFast     = "fast"
	compressionLevelBalanced = "balanced"
	compressionLevelBest     = "best"
)

// CompressedLineFileOptions 控制单 terminal history 的物理保留上限、期限和编码。
// MaxBytes=0、MaxAge=0 分别表示不设对应限制；Compression=none 仍使用紧凑块格式。
type CompressedLineFileOptions struct {
	MaxBytes         int64
	MaxAge           time.Duration
	Compression      string
	CompressionLevel string
	now              func() time.Time
}

type compressedBlock struct {
	offset               int64
	firstLine            int
	lineCount            int
	rawLen               uint32
	storedLen            uint32
	codec                uint8
	checksum             uint32
	createdAt            int64
	searchBloom          []byte
	searchBloomReady     bool
	searchBloomPersisted bool
}

// CompressedLineFile 以独立压缩块保存 logical lines。块头本身就是可重建
// 位置索引；搜索另用固定上限的块级 q-gram sidecar，不建立逐行 postings。
type CompressedLineFile struct {
	path            string
	file            *os.File
	options         CompressedLineFileOptions
	encoder         *zstd.Encoder
	decoder         *zstd.Decoder
	blocks          []compressedBlock
	persistedLines  int
	pending         []Line
	pendingBytes    int
	writeOffset     int64
	retentionEpoch  uint64
	gapOffsets      []int
	searchIndexPath string
	searchIndexFile *os.File
	searchIndexSize int64
}

// OpenCompressedLineFile 打开生产块存储。旧格式不参与恢复：检测到非当前
// block magic/version 时直接丢弃旧 history，从空文件开始。
func OpenCompressedLineFile(dir string, terminalID string, options CompressedLineFileOptions) (*CompressedLineFile, error) {
	path, err := lineFilePath(dir, terminalID)
	if err != nil {
		return nil, err
	}
	options, err = normalizeCompressedLineFileOptions(options)
	if err != nil {
		return nil, os.ErrInvalid
	}
	if err := discardObsoleteLineFile(path); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	result, err := newCompressedLineFile(path, file, options)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := result.recover(); err != nil {
		_ = result.closeCodecs()
		_ = file.Close()
		return nil, err
	}
	result.openSearchIndex()
	if err := result.enforceLimit(); err != nil {
		_ = result.Close()
		return nil, err
	}
	if err := removeObsoleteTerminalHistory(path); err != nil {
		_ = result.Close()
		return nil, err
	}
	return result, nil
}

func lineFilePath(dir string, terminalID string) (string, error) {
	terminalID = strings.TrimSpace(terminalID)
	if dir == "" || terminalID == "" {
		return "", os.ErrInvalid
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, url.PathEscape(terminalID)+".logical-lines.bin"), nil
}

func newCompressedLineFile(path string, file *os.File, options CompressedLineFileOptions) (*CompressedLineFile, error) {
	result := &CompressedLineFile{path: path, file: file, options: options}
	if options.Compression == compressionZstd {
		encoder, err := zstd.NewWriter(nil,
			zstd.WithEncoderConcurrency(1),
			zstd.WithEncoderLevel(zstdLevel(options.CompressionLevel)),
			zstd.WithWindowSize(compressedBlockTargetBytes),
			zstd.WithLowerEncoderMem(true),
		)
		if err != nil {
			return nil, err
		}
		result.encoder = encoder
	}
	return result, nil
}

func normalizeCompressedLineFileOptions(options CompressedLineFileOptions) (CompressedLineFileOptions, error) {
	if options.MaxBytes < 0 || options.MaxAge < 0 {
		return options, os.ErrInvalid
	}
	options.Compression = strings.ToLower(strings.TrimSpace(options.Compression))
	if options.Compression == "" {
		options.Compression = compressionNone
	}
	if options.Compression != compressionNone && options.Compression != compressionZstd && options.Compression != compressionS2 {
		return options, os.ErrInvalid
	}
	options.CompressionLevel = strings.ToLower(strings.TrimSpace(options.CompressionLevel))
	if options.CompressionLevel == "" {
		options.CompressionLevel = compressionLevelFast
	}
	if options.CompressionLevel != compressionLevelFast && options.CompressionLevel != compressionLevelBalanced && options.CompressionLevel != compressionLevelBest {
		return options, os.ErrInvalid
	}
	return options, nil
}

func zstdLevel(level string) zstd.EncoderLevel {
	switch level {
	case compressionLevelBalanced:
		return zstd.SpeedDefault
	case compressionLevelBest:
		return zstd.SpeedBestCompression
	default:
		return zstd.SpeedFastest
	}
}

func discardObsoleteLineFile(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var prefix [6]byte
	_, readErr := io.ReadFull(file, prefix[:])
	if err := file.Close(); err != nil {
		return err
	}
	if readErr == nil &&
		binary.LittleEndian.Uint32(prefix[0:4]) == compressedBlockMagic &&
		binary.LittleEndian.Uint16(prefix[4:6]) == compressedBlockVersion {
		return nil
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return readErr
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return removeObsoleteTerminalHistory(path)
}

func (f *CompressedLineFile) Path() string {
	if f == nil {
		return ""
	}
	return f.path
}

func (f *CompressedLineFile) AppendLines(lines []Line) error {
	if f == nil || len(lines) == 0 {
		return nil
	}
	if f.file == nil {
		return os.ErrInvalid
	}
	for _, line := range lines {
		size := compactLineEncodedSize(line)
		if len(f.pending) > 0 && f.pendingBytes+size > compressedBlockTargetBytes {
			if err := f.flushPending(); err != nil {
				return err
			}
		}
		f.pending = append(f.pending, cloneLine(line))
		f.pendingBytes += size
		if f.pendingBytes >= compressedBlockTargetBytes {
			if err := f.flushPending(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *CompressedLineFile) AppendBoundary() error {
	if f == nil {
		return nil
	}
	if err := f.flushPending(); err != nil {
		return err
	}
	header := makeCompressedBlockHeader(compressedBlockKindBoundary, compressedBlockCodecRaw, 0, 0, 0, 0, f.nowTime().Unix())
	if err := writeAllAtEnd(f.file, header); err != nil {
		f.rollbackPartialRecord(f.writeOffset)
		return err
	}
	f.writeOffset += int64(len(header))
	return f.enforceLimit()
}

func (f *CompressedLineFile) AppendGap() error {
	if f == nil {
		return nil
	}
	if err := f.flushPending(); err != nil {
		return err
	}
	offset := f.LineCount()
	if len(f.gapOffsets) > 0 && f.gapOffsets[len(f.gapOffsets)-1] == offset {
		return nil
	}
	header := makeCompressedBlockHeader(compressedBlockKindGap, compressedBlockCodecRaw, offset, 0, 0, 0, f.nowTime().Unix())
	if err := writeAllAtEnd(f.file, header); err != nil {
		f.rollbackPartialRecord(f.writeOffset)
		return err
	}
	f.writeOffset += int64(len(header))
	f.gapOffsets = append(f.gapOffsets, offset)
	return f.enforceLimit()
}

func (f *CompressedLineFile) GapOffsets() []int {
	if f == nil {
		return nil
	}
	return append([]int(nil), f.gapOffsets...)
}

func (f *CompressedLineFile) LineCount() int {
	if f == nil {
		return 0
	}
	return f.persistedLines + len(f.pending)
}

func (f *CompressedLineFile) Base() int { return 0 }

func (f *CompressedLineFile) RetentionEpoch() uint64 {
	if f == nil {
		return 0
	}
	return f.retentionEpoch
}

// PruneRetention 立即应用当前文件的期限和容量策略。
func (f *CompressedLineFile) PruneRetention() error {
	if f == nil || f.file == nil {
		return nil
	}
	if err := f.flushPending(); err != nil {
		return err
	}
	return f.enforceLimit()
}

func (f *CompressedLineFile) Lines(start int, end int) ([]Line, error) {
	var result []Line
	err := f.VisitLines(start, end, false, func(_ int, line Line) bool {
		result = append(result, cloneLine(line))
		return true
	})
	return result, err
}

func (f *CompressedLineFile) VisitLines(start int, end int, reverse bool, visit func(index int, line Line) bool) error {
	if f == nil || f.file == nil {
		return nil
	}
	total := f.LineCount()
	start = clampLineIndex(start, 0, total)
	end = clampLineIndex(end, start, total)
	if start == end || visit == nil {
		return nil
	}
	if reverse {
		if end > f.persistedLines {
			from := maxInt(start, f.persistedLines) - f.persistedLines
			to := end - f.persistedLines
			for index := to - 1; index >= from; index-- {
				if !visit(f.persistedLines+index, f.pending[index]) {
					return nil
				}
			}
		}
		persistedEnd := minInt(end, f.persistedLines)
		if start >= persistedEnd {
			return nil
		}
		blockIndex := sort.Search(len(f.blocks), func(index int) bool {
			return f.blocks[index].firstLine >= persistedEnd
		}) - 1
		for blockIndex >= 0 {
			block := f.blocks[blockIndex]
			if block.firstLine+block.lineCount <= start {
				break
			}
			lines, err := f.readBlock(block)
			if err != nil {
				return err
			}
			from := maxInt(start, block.firstLine) - block.firstLine
			to := minInt(persistedEnd, block.firstLine+block.lineCount) - block.firstLine
			for index := to - 1; index >= from; index-- {
				if !visit(block.firstLine+index, lines[index]) {
					return nil
				}
			}
			blockIndex--
		}
		return nil
	}
	persistedEnd := minInt(end, f.persistedLines)
	if start < persistedEnd {
		blockIndex := sort.Search(len(f.blocks), func(i int) bool {
			return f.blocks[i].firstLine+f.blocks[i].lineCount > start
		})
		for blockIndex < len(f.blocks) {
			block := f.blocks[blockIndex]
			if block.firstLine >= persistedEnd {
				break
			}
			lines, err := f.readBlock(block)
			if err != nil {
				return err
			}
			from := maxInt(start, block.firstLine) - block.firstLine
			to := minInt(persistedEnd, block.firstLine+block.lineCount) - block.firstLine
			for index := from; index < to; index++ {
				if !visit(block.firstLine+index, lines[index]) {
					return nil
				}
			}
			blockIndex++
		}
	}
	if end > f.persistedLines {
		from := maxInt(start, f.persistedLines) - f.persistedLines
		to := end - f.persistedLines
		for index := from; index < to; index++ {
			if !visit(f.persistedLines+index, f.pending[index]) {
				return nil
			}
		}
	}
	return nil
}

func (f *CompressedLineFile) Sync() error {
	if f == nil || f.file == nil {
		return nil
	}
	if err := f.flushPending(); err != nil {
		return err
	}
	if err := f.file.Sync(); err != nil {
		return err
	}
	f.syncSearchIndex()
	return nil
}

func (f *CompressedLineFile) Close() error {
	if f == nil {
		return nil
	}
	var result error
	if f.file != nil {
		result = errors.Join(result, f.flushPending())
		result = errors.Join(result, f.file.Sync())
		result = errors.Join(result, f.file.Close())
		f.file = nil
	}
	f.closeSearchIndex()
	return errors.Join(result, f.closeCodecs())
}

func (f *CompressedLineFile) closeCodecs() error {
	if f == nil {
		return nil
	}
	if f.encoder != nil {
		f.encoder.Close()
		f.encoder = nil
	}
	if f.decoder != nil {
		f.decoder.Close()
		f.decoder = nil
	}
	return nil
}

func (f *CompressedLineFile) flushPending() error {
	if f == nil || len(f.pending) == 0 {
		return nil
	}
	raw := encodeCompactLines(f.pending)
	codec, stored := f.compressBlock(raw)
	checksum := crc32.ChecksumIEEE(raw)
	createdAt := f.nowTime().Unix()
	header := makeCompressedBlockHeader(compressedBlockKindLines, codec, len(f.pending), len(raw), len(stored), checksum, createdAt)
	offset := f.writeOffset
	if err := writeAllAtEnd(f.file, header); err != nil {
		f.rollbackPartialRecord(offset)
		return err
	}
	if err := writeAllAtEnd(f.file, stored); err != nil {
		f.rollbackPartialRecord(offset)
		return err
	}
	block := compressedBlock{
		offset:    offset,
		firstLine: f.persistedLines,
		lineCount: len(f.pending),
		rawLen:    uint32(len(raw)),
		storedLen: uint32(len(stored)),
		codec:     codec,
		checksum:  checksum,
		createdAt: createdAt,
	}
	block.searchBloom = buildBlockSearchBloom(f.pending)
	block.searchBloomReady = true
	f.blocks = append(f.blocks, block)
	f.appendSearchIndexRecord(&f.blocks[len(f.blocks)-1])
	f.persistedLines += len(f.pending)
	f.writeOffset += int64(len(header) + len(stored))
	f.pending = nil
	f.pendingBytes = 0
	return f.enforceLimit()
}

func (f *CompressedLineFile) compressBlock(raw []byte) (uint8, []byte) {
	var codec uint8
	var compressed []byte
	switch f.options.Compression {
	case compressionZstd:
		codec = compressedBlockCodecZstd
		compressed = f.encoder.EncodeAll(raw, nil)
	case compressionS2:
		codec = compressedBlockCodecS2
		switch f.options.CompressionLevel {
		case compressionLevelBalanced:
			compressed = s2.EncodeBetter(nil, raw)
		case compressionLevelBest:
			compressed = s2.EncodeBest(nil, raw)
		default:
			compressed = s2.Encode(nil, raw)
		}
	default:
		return compressedBlockCodecRaw, raw
	}
	if len(compressed) >= len(raw) {
		return compressedBlockCodecRaw, raw
	}
	return codec, compressed
}

func (f *CompressedLineFile) recover() error {
	info, err := f.file.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	offset := int64(0)
	for offset+compressedBlockHeaderSize <= size {
		var header [compressedBlockHeaderSize]byte
		if _, err := f.file.ReadAt(header[:], offset); err != nil {
			return err
		}
		kind, codec, lineCount, rawLen, storedLen, checksum, createdAt, err := parseCompressedBlockHeader(header[:])
		if err != nil {
			return err
		}
		recordEnd := offset + compressedBlockHeaderSize + int64(storedLen)
		if recordEnd > size {
			break
		}
		switch kind {
		case compressedBlockKindLines:
			if lineCount == 0 || rawLen == 0 || storedLen == 0 {
				return errors.New("invalid empty compressed history block")
			}
			f.blocks = append(f.blocks, compressedBlock{
				offset: offset, firstLine: f.persistedLines, lineCount: int(lineCount),
				rawLen: rawLen, storedLen: storedLen, codec: codec, checksum: checksum, createdAt: createdAt,
			})
			f.persistedLines += int(lineCount)
		case compressedBlockKindBoundary:
			if lineCount != 0 || rawLen != 0 || storedLen != 0 || codec != compressedBlockCodecRaw {
				return errors.New("invalid history boundary block")
			}
		case compressedBlockKindGap:
			if rawLen != 0 || storedLen != 0 || codec != compressedBlockCodecRaw || int(lineCount) > f.persistedLines {
				return errors.New("invalid history output gap block")
			}
			gap := int(lineCount)
			if len(f.gapOffsets) == 0 || f.gapOffsets[len(f.gapOffsets)-1] != gap {
				f.gapOffsets = append(f.gapOffsets, gap)
			}
		default:
			return errors.New("unsupported compressed history block kind")
		}
		offset = recordEnd
	}
	if offset != size {
		if err := f.file.Truncate(offset); err != nil {
			return err
		}
	}
	if _, err := f.file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	f.writeOffset = offset
	return nil
}

func (f *CompressedLineFile) readBlock(block compressedBlock) ([]Line, error) {
	stored := make([]byte, int(block.storedLen))
	if _, err := f.file.ReadAt(stored, block.offset+compressedBlockHeaderSize); err != nil {
		return nil, err
	}
	var raw []byte
	switch block.codec {
	case compressedBlockCodecRaw:
		raw = stored
	case compressedBlockCodecZstd:
		if err := f.ensureZstdDecoder(); err != nil {
			return nil, err
		}
		decoded, err := f.decoder.DecodeAll(stored, make([]byte, 0, int(block.rawLen)))
		if err != nil {
			return nil, err
		}
		raw = decoded
	case compressedBlockCodecS2:
		decodedLen, err := s2.DecodedLen(stored)
		if err != nil || decodedLen != int(block.rawLen) {
			return nil, errors.New("invalid s2 history block size")
		}
		decoded, err := s2.Decode(make([]byte, 0, int(block.rawLen)), stored)
		if err != nil {
			return nil, err
		}
		raw = decoded
	default:
		return nil, errors.New("unsupported compressed history codec")
	}
	if len(raw) != int(block.rawLen) || crc32.ChecksumIEEE(raw) != block.checksum {
		return nil, errors.New("compressed history block checksum mismatch")
	}
	lines, err := decodeCompactLines(raw)
	if err != nil {
		return nil, err
	}
	if len(lines) != block.lineCount {
		return nil, errors.New("compressed history block line count mismatch")
	}
	return lines, nil
}

func (f *CompressedLineFile) ensureZstdDecoder() error {
	if f.decoder != nil {
		return nil
	}
	decoder, err := zstd.NewReader(nil,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderLowmem(true),
		zstd.WithDecoderMaxWindow(compressedBlockMaxRawBytes),
	)
	if err != nil {
		return err
	}
	f.decoder = decoder
	return nil
}

func (f *CompressedLineFile) enforceLimit() error {
	if f == nil || f.file == nil {
		return nil
	}
	start := 0
	needsRewrite := false
	if f.options.MaxAge > 0 {
		cutoff := f.nowTime().Add(-f.options.MaxAge).Unix()
		for start < len(f.blocks) && f.blocks[start].createdAt < cutoff {
			start++
		}
		needsRewrite = start > 0
	}
	if f.options.MaxBytes > 0 && f.writeOffset+f.searchIndexSize > f.options.MaxBytes {
		needsRewrite = true
		target := f.options.MaxBytes * compressedRetentionNumerator / compressedRetentionDenominator
		used := int64(0)
		sizeStart := len(f.blocks)
		gapIndex := len(f.gapOffsets) - 1
		for i := len(f.blocks) - 1; i >= 0; i-- {
			size := int64(compressedBlockHeaderSize) + int64(f.blocks[i].storedLen) + searchIndexStoredBytes(f.blocks[i])
			for gapIndex >= 0 && f.gapOffsets[gapIndex] > f.blocks[i].firstLine {
				size += compressedBlockHeaderSize
				gapIndex--
			}
			if used+size > target {
				break
			}
			used += size
			sizeStart = i
		}
		if sizeStart > start {
			start = sizeStart
		}
	}
	if !needsRewrite {
		return nil
	}
	return f.rewriteBlocks(start)
}

func (f *CompressedLineFile) rewriteBlocks(start int) error {
	start = clampLineIndex(start, 0, len(f.blocks))
	droppedLines := 0
	if start < len(f.blocks) {
		droppedLines = f.blocks[start].firstLine
	} else {
		droppedLines = f.persistedLines
	}
	temporary, err := os.CreateTemp(filepath.Dir(f.path), ".anytty-history-compact-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	published := false
	defer func() {
		_ = temporary.Close()
		if !published {
			_ = os.Remove(temporaryPath)
		}
	}()
	newBlocks := make([]compressedBlock, 0, len(f.blocks)-start)
	oldGapOffsets := append([]int(nil), f.gapOffsets...)
	newOffset := int64(0)
	newFirst := 0
	for _, block := range f.blocks[start:] {
		size := int64(compressedBlockHeaderSize) + int64(block.storedLen)
		if _, err := io.CopyN(temporary, io.NewSectionReader(f.file, block.offset, size), size); err != nil {
			return err
		}
		block.offset = newOffset
		block.firstLine = newFirst
		newBlocks = append(newBlocks, block)
		newOffset += size
		newFirst += block.lineCount
	}
	newGapOffsets := make([]int, 0, len(oldGapOffsets))
	for _, gap := range oldGapOffsets {
		if gap <= droppedLines {
			continue
		}
		gap -= droppedLines
		header := makeCompressedBlockHeader(compressedBlockKindGap, compressedBlockCodecRaw, gap, 0, 0, 0, f.nowTime().Unix())
		if _, err := temporary.Write(header); err != nil {
			return err
		}
		newOffset += int64(len(header))
		newGapOffsets = append(newGapOffsets, gap)
	}
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := f.file.Close(); err != nil {
		return err
	}
	f.file = nil
	if err := filepublish.Rename(temporaryPath, f.path); err != nil {
		if reopened, reopenErr := os.OpenFile(f.path, os.O_RDWR, 0o600); reopenErr == nil {
			_, _ = reopened.Seek(f.writeOffset, io.SeekStart)
			f.file = reopened
		}
		return err
	}
	published = true
	file, err := os.OpenFile(f.path, os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Seek(newOffset, io.SeekStart); err != nil {
		_ = file.Close()
		return err
	}
	f.file = file
	f.blocks = newBlocks
	f.persistedLines = newFirst
	f.writeOffset = newOffset
	f.gapOffsets = newGapOffsets
	if err := f.rewriteSearchIndex(); err != nil {
		f.closeSearchIndex()
	}
	if err := filepublish.SyncDirectory(filepath.Dir(f.path)); err != nil {
		return err
	}
	if droppedLines > 0 {
		f.retentionEpoch++
	}
	return nil
}

func makeCompressedBlockHeader(kind uint8, codec uint8, lineCount int, rawLen int, storedLen int, checksum uint32, createdAt int64) []byte {
	header := make([]byte, compressedBlockHeaderSize)
	binary.LittleEndian.PutUint32(header[0:4], compressedBlockMagic)
	binary.LittleEndian.PutUint16(header[4:6], compressedBlockVersion)
	header[6] = kind
	header[7] = codec
	binary.LittleEndian.PutUint32(header[8:12], uint32(lineCount))
	binary.LittleEndian.PutUint32(header[12:16], uint32(rawLen))
	binary.LittleEndian.PutUint32(header[16:20], uint32(storedLen))
	binary.LittleEndian.PutUint32(header[20:24], checksum)
	binary.LittleEndian.PutUint64(header[24:32], uint64(createdAt))
	return header
}

func parseCompressedBlockHeader(header []byte) (uint8, uint8, uint32, uint32, uint32, uint32, int64, error) {
	if len(header) != compressedBlockHeaderSize || binary.LittleEndian.Uint32(header[0:4]) != compressedBlockMagic {
		return 0, 0, 0, 0, 0, 0, 0, errors.New("invalid compressed history block magic")
	}
	if binary.LittleEndian.Uint16(header[4:6]) != compressedBlockVersion {
		return 0, 0, 0, 0, 0, 0, 0, errors.New("unsupported compressed history block version")
	}
	kind := header[6]
	codec := header[7]
	lineCount := binary.LittleEndian.Uint32(header[8:12])
	rawLen := binary.LittleEndian.Uint32(header[12:16])
	storedLen := binary.LittleEndian.Uint32(header[16:20])
	checksum := binary.LittleEndian.Uint32(header[20:24])
	createdAt := int64(binary.LittleEndian.Uint64(header[24:32]))
	if rawLen > compressedBlockMaxRawBytes || storedLen > compressedBlockMaxRawBytes {
		return 0, 0, 0, 0, 0, 0, 0, errors.New("compressed history block exceeds size limit")
	}
	if codec != compressedBlockCodecRaw && codec != compressedBlockCodecZstd && codec != compressedBlockCodecS2 {
		return 0, 0, 0, 0, 0, 0, 0, errors.New("unsupported compressed history codec")
	}
	return kind, codec, lineCount, rawLen, storedLen, checksum, createdAt, nil
}

func (f *CompressedLineFile) nowTime() time.Time {
	if f != nil && f.options.now != nil {
		return f.options.now()
	}
	return time.Now()
}

func (f *CompressedLineFile) rollbackPartialRecord(offset int64) {
	if f == nil || f.file == nil {
		return
	}
	_ = f.file.Truncate(offset)
	_, _ = f.file.Seek(offset, io.SeekStart)
}

func writeAllAtEnd(file *os.File, payload []byte) error {
	return writeAllWithRetry(file, payload, time.Sleep)
}

const historyWriteMaxAttempts = 3

// Retry only within one file record. Callers still roll the entire record back
// when a permanent error or the bounded retry limit is reached.
func writeAllWithRetry(writer io.Writer, payload []byte, sleep func(time.Duration)) error {
	failures := 0
	for len(payload) > 0 {
		n, err := writer.Write(payload)
		if n < 0 || n > len(payload) {
			return io.ErrShortWrite
		}
		if n > 0 {
			payload = payload[n:]
		}
		if err != nil {
			failures++
			if len(payload) == 0 || failures >= historyWriteMaxAttempts || !retryableHistoryWriteError(err) {
				return err
			}
			if sleep != nil {
				sleep(time.Duration(1<<(failures-1)) * time.Millisecond)
			}
			continue
		}
		if n == 0 {
			failures++
			if failures >= historyWriteMaxAttempts {
				return io.ErrShortWrite
			}
			if sleep != nil {
				sleep(time.Duration(1<<(failures-1)) * time.Millisecond)
			}
		}
	}
	return nil
}

func retryableHistoryWriteError(err error) bool {
	type temporaryError interface {
		Temporary() bool
	}
	var temporary temporaryError
	return errors.As(err, &temporary) && temporary.Temporary()
}

func removeObsoleteTerminalHistory(currentPath string) error {
	base := strings.TrimSuffix(currentPath, ".logical-lines.bin")
	var result error
	for _, path := range []string{
		currentPath + ".idx",
		base + ".history-lines.bin",
		base + ".screen-rows.bin",
	} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}
	matches, _ := filepath.Glob(currentPath + ".rows.*.idx")
	for _, path := range matches {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}
	return result
}

func cloneLine(line Line) Line {
	result := line
	result.Runs = append([]Run(nil), line.Runs...)
	return result
}

func clampLineIndex(value int, low int, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
