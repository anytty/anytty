package linehist

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/anytty/anytty/shared/filepublish"
)

const (
	searchIndexMagic             uint32 = 0x58444953 // "SIDX"
	searchIndexVersion           uint16 = 4
	searchIndexHeaderSize               = 40
	searchIndexByteSetBytes             = 256 / 8
	searchIndexBigramBloomBytes         = 512
	searchIndexBloomBytes               = 8 * 1024
	searchIndexByteSetEnd               = searchIndexByteSetBytes
	searchIndexBigramBloomEnd           = searchIndexByteSetEnd + searchIndexBigramBloomBytes
	searchIndexTrigramBloomBytes        = searchIndexBloomBytes - searchIndexBigramBloomEnd
	searchIndexBigramBloomBits          = searchIndexBigramBloomBytes * 8
	searchIndexTrigramBloomBits         = searchIndexTrigramBloomBytes * 8
	searchIndexBigramHashCount   uint8  = 4
	searchIndexTrigramHashCount  uint8  = 7
	searchIndexMaxUniqueTrigrams        = 32 * 1024
	searchIndexSuffix                   = ".search-qgram-v4.idx"
	searchIndexObsoleteV1Suffix         = ".search-trigram-v1.idx"
	searchIndexObsoleteV2Suffix         = ".search-qgram-v2.idx"
	searchIndexObsoleteV3Suffix         = ".search-qgram-v3.idx"
)

// Each block payload combines an exact byte set with compact bigram and
// trigram filters. Exact line matching remains authoritative after a block
// passes the filters.

func searchIndexPath(historyPath string) string {
	return historyPath + searchIndexSuffix
}

type searchQueryGrams struct {
	byteValues []byte
	bigrams    []uint16
	trigrams   []uint32
}

var searchTrigramSeenPool = sync.Pool{
	New: func() any { return make([]byte, 1<<21) },
}

var searchBigramSeenPool = sync.Pool{
	New: func() any { return make([]byte, 1<<13) },
}

func querySearchGrams(query string) searchQueryGrams {
	payload := []byte(query)
	result := searchQueryGrams{byteValues: payload}
	if len(payload) >= 2 {
		result.bigrams = make([]uint16, 0, len(payload)-1)
		for index := 0; index+1 < len(payload); index++ {
			result.bigrams = append(result.bigrams, byteBigram(payload[index:index+2]))
		}
	}
	if len(payload) >= 3 {
		unique := make(map[uint32]struct{}, len(payload)-2)
		for index := 0; index+2 < len(payload); index++ {
			unique[byteTrigram(payload[index:index+3])] = struct{}{}
		}
		result.trigrams = make([]uint32, 0, len(unique))
		for trigram := range unique {
			result.trigrams = append(result.trigrams, trigram)
		}
		sort.Slice(result.trigrams, func(left int, right int) bool {
			return result.trigrams[left] < result.trigrams[right]
		})
	}
	return result
}

func buildBlockSearchBloom(lines []Line) []byte {
	bloom := make([]byte, searchIndexBloomBytes)
	byteSet := bloom[:searchIndexByteSetEnd]
	bigramBloom := bloom[searchIndexByteSetEnd:searchIndexBigramBloomEnd]
	bigramSeen := searchBigramSeenPool.Get().([]byte)
	bigrams := make([]uint16, 0, 512)
	trigramSeen := searchTrigramSeenPool.Get().([]byte)
	trigrams := make([]uint32, 0, 4096)
	defer func() {
		for _, bigram := range bigrams {
			bigramSeen[bigram>>3] &^= 1 << uint(bigram&7)
		}
		searchBigramSeenPool.Put(bigramSeen)
		for _, trigram := range trigrams {
			trigramSeen[trigram>>3] &^= 1 << uint(trigram&7)
		}
		searchTrigramSeenPool.Put(trigramSeen)
	}()
	trigramsSaturated := false
	for _, line := range lines {
		var previous byte
		var trigramWindow uint32
		byteCount := 0
		for _, run := range line.Runs {
			for index := 0; index < len(run.Text); index++ {
				value := run.Text[index]
				byteSet[value>>3] |= 1 << uint(value&7)
				if byteCount >= 1 {
					bigram := uint16(previous)<<8 | uint16(value)
					seenByte := bigram >> 3
					seenMask := byte(1 << uint(bigram&7))
					if bigramSeen[seenByte]&seenMask == 0 {
						bigramSeen[seenByte] |= seenMask
						bigrams = append(bigrams, bigram)
					}
				}
				trigramWindow = trigramWindow<<8 | uint32(value)
				if byteCount >= 2 && !trigramsSaturated {
					trigram := trigramWindow & 0x00ffffff
					seenByte := trigram >> 3
					seenMask := byte(1 << uint(trigram&7))
					if trigramSeen[seenByte]&seenMask == 0 {
						trigramSeen[seenByte] |= seenMask
						trigrams = append(trigrams, trigram)
						if len(trigrams) > searchIndexMaxUniqueTrigrams {
							trigramsSaturated = true
						}
					}
				}
				previous = value
				byteCount++
			}
		}
	}
	for _, bigram := range bigrams {
		addBigramToBloom(bigramBloom, bigram)
	}
	trigramBloom := bloom[searchIndexBigramBloomEnd:]
	if trigramsSaturated {
		for index := range trigramBloom {
			trigramBloom[index] = 0xff
		}
	} else {
		for _, trigram := range trigrams {
			addTrigramToBloom(trigramBloom, trigram)
		}
	}
	return bloom
}

func blockSearchBloomMayContain(bloom []byte, grams searchQueryGrams) bool {
	if len(grams.byteValues) == 0 || len(bloom) != searchIndexBloomBytes {
		return true
	}
	byteSet := bloom[:searchIndexByteSetEnd]
	for _, value := range grams.byteValues {
		if byteSet[value>>3]&(1<<uint(value&7)) == 0 {
			return false
		}
	}
	if !bigramBloomMayContain(bloom[searchIndexByteSetEnd:searchIndexBigramBloomEnd], grams.bigrams) {
		return false
	}
	if !trigramBloomMayContain(bloom[searchIndexBigramBloomEnd:], grams.trigrams) {
		return false
	}
	return true
}

func blockSearchBloomMayContainAll(bloom []byte, filters []searchQueryGrams) bool {
	for _, grams := range filters {
		if !blockSearchBloomMayContain(bloom, grams) {
			return false
		}
	}
	return true
}

func bigramBloomMayContain(bloom []byte, bigrams []uint16) bool {
	for _, bigram := range bigrams {
		first, step := bigramBloomHashes(bigram)
		for hash := uint8(0); hash < searchIndexBigramHashCount; hash++ {
			bit := (first + uint64(hash)*step) % searchIndexBigramBloomBits
			if bloom[bit>>3]&(1<<uint(bit&7)) == 0 {
				return false
			}
		}
	}
	return true
}

func addBigramToBloom(bloom []byte, bigram uint16) {
	first, step := bigramBloomHashes(bigram)
	for hash := uint8(0); hash < searchIndexBigramHashCount; hash++ {
		bit := (first + uint64(hash)*step) % searchIndexBigramBloomBits
		bloom[bit>>3] |= 1 << uint(bit&7)
	}
}

func trigramBloomMayContain(bloom []byte, trigrams []uint32) bool {
	for _, trigram := range trigrams {
		first, step := trigramBloomHashes(trigram)
		for hash := uint8(0); hash < searchIndexTrigramHashCount; hash++ {
			bit := (first + uint64(hash)*step) % searchIndexTrigramBloomBits
			if bloom[bit>>3]&(1<<uint(bit&7)) == 0 {
				return false
			}
		}
	}
	return true
}

func addTrigramToBloom(bloom []byte, trigram uint32) {
	first, step := trigramBloomHashes(trigram)
	for hash := uint8(0); hash < searchIndexTrigramHashCount; hash++ {
		bit := (first + uint64(hash)*step) % searchIndexTrigramBloomBits
		bloom[bit>>3] |= 1 << uint(bit&7)
	}
}

func byteTrigram(payload []byte) uint32 {
	return uint32(payload[0])<<16 | uint32(payload[1])<<8 | uint32(payload[2])
}

func byteBigram(payload []byte) uint16 {
	return uint16(payload[0])<<8 | uint16(payload[1])
}

func trigramBloomHashes(trigram uint32) (uint64, uint64) {
	first := mixSearchIndexHash(uint64(trigram) + 0x9e3779b97f4a7c15)
	step := mixSearchIndexHash(uint64(trigram)+0xd1b54a32d192ed03) | 1
	return first, step
}

func bigramBloomHashes(bigram uint16) (uint64, uint64) {
	first := mixSearchIndexHash(uint64(bigram) + 0xa0761d6478bd642f)
	step := mixSearchIndexHash(uint64(bigram)+0xe7037ed1a0b428db) | 1
	return first, step
}

func mixSearchIndexHash(value uint64) uint64 {
	value ^= value >> 30
	value *= 0xbf58476d1ce4e5b9
	value ^= value >> 27
	value *= 0x94d049bb133111eb
	return value ^ (value >> 31)
}

func (f *CompressedLineFile) openSearchIndex() {
	if f == nil || f.path == "" {
		return
	}
	_ = os.Remove(f.path + searchIndexObsoleteV1Suffix)
	_ = os.Remove(f.path + searchIndexObsoleteV2Suffix)
	_ = os.Remove(f.path + searchIndexObsoleteV3Suffix)
	path := searchIndexPath(f.path)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return
	}
	f.searchIndexPath = path
	f.searchIndexFile = file
	dirty, err := f.recoverSearchIndex()
	if err != nil {
		_ = file.Truncate(0)
		_, _ = file.Seek(0, io.SeekStart)
		f.searchIndexSize = 0
		for index := range f.blocks {
			f.blocks[index].searchBloom = nil
			f.blocks[index].searchBloomReady = false
			f.blocks[index].searchBloomPersisted = false
		}
		return
	}
	if dirty {
		_ = f.rewriteSearchIndex()
	}
}

func (f *CompressedLineFile) recoverSearchIndex() (bool, error) {
	info, err := f.searchIndexFile.Stat()
	if err != nil {
		return false, err
	}
	size := info.Size()
	offset := int64(0)
	dirty := false
	for offset+searchIndexHeaderSize <= size {
		var header [searchIndexHeaderSize]byte
		if _, err := f.searchIndexFile.ReadAt(header[:], offset); err != nil {
			return false, err
		}
		record, err := parseSearchIndexHeader(header[:])
		if err != nil {
			dirty = true
			break
		}
		recordEnd := offset + searchIndexHeaderSize + int64(record.bloomLen)
		if recordEnd > size {
			dirty = true
			break
		}
		var bloom []byte
		if record.bloomLen > 0 {
			bloom = make([]byte, int(record.bloomLen))
			if _, err := f.searchIndexFile.ReadAt(bloom, offset+searchIndexHeaderSize); err != nil {
				return false, err
			}
			if crc32.ChecksumIEEE(bloom) != record.bloomChecksum {
				dirty = true
				break
			}
		}
		if block := f.searchIndexBlock(record.firstLine); block != nil && searchIndexRecordMatchesBlock(record, *block) {
			if block.searchBloomPersisted {
				dirty = true
			}
			block.searchBloom = bloom
			block.searchBloomReady = true
			block.searchBloomPersisted = true
		} else {
			dirty = true
		}
		offset = recordEnd
	}
	if offset != size {
		dirty = true
		if err := f.searchIndexFile.Truncate(offset); err != nil {
			return false, err
		}
	}
	if _, err := f.searchIndexFile.Seek(offset, io.SeekStart); err != nil {
		return false, err
	}
	f.searchIndexSize = offset
	return dirty, nil
}

func (f *CompressedLineFile) searchIndexBlock(firstLine int) *compressedBlock {
	index := sort.Search(len(f.blocks), func(index int) bool { return f.blocks[index].firstLine >= firstLine })
	if index >= len(f.blocks) || f.blocks[index].firstLine != firstLine {
		return nil
	}
	return &f.blocks[index]
}

func (f *CompressedLineFile) ensureBlockSearchBloom(block *compressedBlock, lines []Line) {
	if block == nil || block.searchBloomReady {
		return
	}
	block.searchBloom = buildBlockSearchBloom(lines)
	block.searchBloomReady = true
	f.appendSearchIndexRecord(block)
}

func (f *CompressedLineFile) appendSearchIndexRecord(block *compressedBlock) {
	if f == nil || block == nil || !block.searchBloomReady || block.searchBloomPersisted || f.searchIndexFile == nil {
		return
	}
	header := makeSearchIndexHeader(*block)
	recordBytes := int64(len(header) + len(block.searchBloom))
	if f.options.MaxBytes > 0 && f.writeOffset+f.searchIndexSize+recordBytes > f.options.MaxBytes {
		return
	}
	offset := f.searchIndexSize
	if err := writeAllAtEnd(f.searchIndexFile, header); err != nil {
		f.disableSearchIndex(offset)
		return
	}
	if len(block.searchBloom) > 0 {
		if err := writeAllAtEnd(f.searchIndexFile, block.searchBloom); err != nil {
			f.disableSearchIndex(offset)
			return
		}
	}
	f.searchIndexSize += recordBytes
	block.searchBloomPersisted = true
}

func (f *CompressedLineFile) disableSearchIndex(validSize int64) {
	if f.searchIndexFile != nil {
		_ = f.searchIndexFile.Truncate(validSize)
		_ = f.searchIndexFile.Close()
		f.searchIndexFile = nil
	}
	f.searchIndexSize = validSize
}

func (f *CompressedLineFile) syncSearchIndex() {
	if f != nil && f.searchIndexFile != nil {
		_ = f.searchIndexFile.Sync()
	}
}

func (f *CompressedLineFile) closeSearchIndex() {
	if f == nil || f.searchIndexFile == nil {
		return
	}
	_ = f.searchIndexFile.Sync()
	_ = f.searchIndexFile.Close()
	f.searchIndexFile = nil
}

func (f *CompressedLineFile) rewriteSearchIndex() error {
	if f == nil || f.searchIndexPath == "" {
		return nil
	}
	temporary, err := os.CreateTemp(filepath.Dir(f.searchIndexPath), ".anytty-history-search-index-*")
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
	newSize := int64(0)
	persisted := make([]bool, len(f.blocks))
	for index := len(f.blocks) - 1; index >= 0; index-- {
		block := &f.blocks[index]
		if !block.searchBloomReady {
			continue
		}
		header := makeSearchIndexHeader(*block)
		recordBytes := int64(len(header) + len(block.searchBloom))
		if f.options.MaxBytes > 0 && f.writeOffset+newSize+recordBytes > f.options.MaxBytes {
			continue
		}
		if err := writeAllAtEnd(temporary, header); err != nil {
			return err
		}
		if len(block.searchBloom) > 0 {
			if err := writeAllAtEnd(temporary, block.searchBloom); err != nil {
				return err
			}
		}
		persisted[index] = true
		newSize += recordBytes
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
	if f.searchIndexFile != nil {
		_ = f.searchIndexFile.Close()
		f.searchIndexFile = nil
	}
	if err := filepublish.Rename(temporaryPath, f.searchIndexPath); err != nil {
		return err
	}
	published = true
	file, err := os.OpenFile(f.searchIndexPath, os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Seek(newSize, io.SeekStart); err != nil {
		_ = file.Close()
		return err
	}
	f.searchIndexFile = file
	f.searchIndexSize = newSize
	for index := range f.blocks {
		f.blocks[index].searchBloomPersisted = persisted[index]
	}
	return filepublish.SyncDirectory(filepath.Dir(f.searchIndexPath))
}

type searchIndexRecord struct {
	flags         uint8
	firstLine     int
	lineCount     int
	rawLen        uint32
	blockChecksum uint32
	bloomLen      uint32
	bloomChecksum uint32
}

func makeSearchIndexHeader(block compressedBlock) []byte {
	header := make([]byte, searchIndexHeaderSize)
	binary.LittleEndian.PutUint32(header[0:4], searchIndexMagic)
	binary.LittleEndian.PutUint16(header[4:6], searchIndexVersion)
	header[6] = searchIndexTrigramHashCount
	binary.LittleEndian.PutUint64(header[8:16], uint64(block.firstLine))
	binary.LittleEndian.PutUint32(header[16:20], uint32(block.lineCount))
	binary.LittleEndian.PutUint32(header[20:24], block.rawLen)
	binary.LittleEndian.PutUint32(header[24:28], block.checksum)
	binary.LittleEndian.PutUint32(header[28:32], uint32(len(block.searchBloom)))
	binary.LittleEndian.PutUint32(header[32:36], crc32.ChecksumIEEE(block.searchBloom))
	binary.LittleEndian.PutUint32(header[36:40], crc32.ChecksumIEEE(header[:36]))
	return header
}

func parseSearchIndexHeader(header []byte) (searchIndexRecord, error) {
	if len(header) != searchIndexHeaderSize || binary.LittleEndian.Uint32(header[0:4]) != searchIndexMagic {
		return searchIndexRecord{}, os.ErrInvalid
	}
	if binary.LittleEndian.Uint16(header[4:6]) != searchIndexVersion || header[6] != searchIndexTrigramHashCount {
		return searchIndexRecord{}, os.ErrInvalid
	}
	if binary.LittleEndian.Uint32(header[36:40]) != crc32.ChecksumIEEE(header[:36]) {
		return searchIndexRecord{}, os.ErrInvalid
	}
	firstLine := binary.LittleEndian.Uint64(header[8:16])
	lineCount := binary.LittleEndian.Uint32(header[16:20])
	if firstLine > uint64(^uint(0)>>1) || uint64(lineCount) > uint64(^uint(0)>>1) || lineCount == 0 {
		return searchIndexRecord{}, os.ErrInvalid
	}
	record := searchIndexRecord{
		flags:         header[7],
		firstLine:     int(firstLine),
		lineCount:     int(lineCount),
		rawLen:        binary.LittleEndian.Uint32(header[20:24]),
		blockChecksum: binary.LittleEndian.Uint32(header[24:28]),
		bloomLen:      binary.LittleEndian.Uint32(header[28:32]),
		bloomChecksum: binary.LittleEndian.Uint32(header[32:36]),
	}
	if record.flags != 0 || record.bloomLen != searchIndexBloomBytes {
		return searchIndexRecord{}, os.ErrInvalid
	}
	return record, nil
}

func searchIndexRecordMatchesBlock(record searchIndexRecord, block compressedBlock) bool {
	return record.firstLine == block.firstLine &&
		record.lineCount == block.lineCount &&
		record.rawLen == block.rawLen &&
		record.blockChecksum == block.checksum
}

func searchIndexStoredBytes(block compressedBlock) int64 {
	if !block.searchBloomReady || !block.searchBloomPersisted {
		return 0
	}
	return int64(searchIndexHeaderSize + len(block.searchBloom))
}

// ReadSearchBatch returns at most one compressed block (or the pending tail).
// An empty lines slice with an advanced cursor means the Bloom filter proved
// that the block cannot contain the query.
func (f *CompressedLineFile) readSearchBatch(start int, end int, cursor int, reverse bool, filters []searchQueryGrams) (int, []Line, int, error) {
	total := f.LineCount()
	start = clampLineIndex(start, 0, total)
	end = clampLineIndex(end, start, total)
	if reverse {
		cursor = clampLineIndex(cursor, start, end)
		if cursor <= start {
			return start, nil, start, nil
		}
		if cursor > f.persistedLines {
			batchStart := maxInt(start, f.persistedLines)
			batchEnd := cursor
			lines := cloneLines(f.pending[batchStart-f.persistedLines : batchEnd-f.persistedLines])
			return batchStart, lines, batchStart, nil
		}
		blockIndex := sort.Search(len(f.blocks), func(index int) bool {
			return f.blocks[index].firstLine+f.blocks[index].lineCount >= cursor
		})
		if blockIndex >= len(f.blocks) {
			return start, nil, start, os.ErrInvalid
		}
		return f.readPersistedSearchBlock(blockIndex, start, cursor, true, filters)
	}

	cursor = clampLineIndex(cursor, start, end)
	if cursor >= end {
		return end, nil, end, nil
	}
	if cursor >= f.persistedLines {
		batchStart := cursor
		batchEnd := end
		lines := cloneLines(f.pending[batchStart-f.persistedLines : batchEnd-f.persistedLines])
		return batchStart, lines, batchEnd, nil
	}
	blockIndex := sort.Search(len(f.blocks), func(index int) bool {
		return f.blocks[index].firstLine+f.blocks[index].lineCount > cursor
	})
	if blockIndex >= len(f.blocks) {
		return end, nil, end, os.ErrInvalid
	}
	return f.readPersistedSearchBlock(blockIndex, cursor, end, false, filters)
}

func (f *CompressedLineFile) readPersistedSearchBlock(blockIndex int, rangeStart int, rangeEnd int, reverse bool, filters []searchQueryGrams) (int, []Line, int, error) {
	block := &f.blocks[blockIndex]
	batchStart := maxInt(rangeStart, block.firstLine)
	batchEnd := minInt(rangeEnd, block.firstLine+block.lineCount)
	nextCursor := batchEnd
	if reverse {
		nextCursor = batchStart
	}
	if block.searchBloomReady && !blockSearchBloomMayContainAll(block.searchBloom, filters) {
		return batchStart, nil, nextCursor, nil
	}
	lines, err := f.readBlock(*block)
	if err != nil {
		return batchStart, nil, nextCursor, err
	}
	f.ensureBlockSearchBloom(block, lines)
	if !blockSearchBloomMayContainAll(block.searchBloom, filters) {
		return batchStart, nil, nextCursor, nil
	}
	from := batchStart - block.firstLine
	to := batchEnd - block.firstLine
	return batchStart, lines[from:to], nextCursor, nil
}

func cloneLines(lines []Line) []Line {
	result := make([]Line, len(lines))
	for index, line := range lines {
		result[index] = cloneLine(line)
	}
	return result
}
