package linehist

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/anytty/anytty/core/history"
)

const compactLineFlagHardEnd uint8 = 1 << 0

func encodeCompactLines(lines []Line) []byte {
	payload := make([]byte, 0, compactLinesEncodedSize(lines))
	payload = binary.AppendUvarint(payload, uint64(len(lines)))
	for _, line := range lines {
		var flags byte
		if line.HardEnd {
			flags = compactLineFlagHardEnd
		}
		payload = append(payload, flags)
		payload = binary.AppendVarint(payload, unixNanoOrZero(line.UpdatedAt))
		payload = binary.AppendUvarint(payload, uint64(len(line.Runs)))
		for _, run := range line.Runs {
			payload = appendCompactString(payload, run.Text)
			payload = appendCompactString(payload, run.Style.FG)
			payload = appendCompactString(payload, run.Style.BG)
			payload = append(payload, compactStyleFlags(run.Style))
			payload = appendCompactString(payload, run.LinkURL)
			payload = appendCompactString(payload, run.LinkParams)
		}
	}
	return payload
}

func decodeCompactLines(payload []byte) ([]Line, error) {
	lineCount, size := binary.Uvarint(payload)
	if size <= 0 {
		return nil, errors.New("invalid compact history line count")
	}
	payload = payload[size:]
	if lineCount > uint64(len(payload)) {
		return nil, errors.New("compact history line count exceeds block")
	}
	lines := make([]Line, 0, int(lineCount))
	for i := uint64(0); i < lineCount; i++ {
		if len(payload) == 0 {
			return nil, errors.New("truncated compact history line")
		}
		line := Line{HardEnd: payload[0]&compactLineFlagHardEnd != 0}
		payload = payload[1:]
		updatedAt, size := binary.Varint(payload)
		if size <= 0 {
			return nil, errors.New("invalid compact history timestamp")
		}
		payload = payload[size:]
		if updatedAt != 0 {
			line.UpdatedAt = time.Unix(0, updatedAt).UTC()
		}
		runCount, size := binary.Uvarint(payload)
		if size <= 0 {
			return nil, errors.New("invalid compact history run count")
		}
		payload = payload[size:]
		if runCount > uint64(len(payload)) {
			return nil, errors.New("compact history run count exceeds block")
		}
		line.Runs = make([]Run, 0, int(runCount))
		for runIndex := uint64(0); runIndex < runCount; runIndex++ {
			var run Run
			var err error
			if run.Text, payload, err = takeCompactString(payload); err != nil {
				return nil, err
			}
			if run.Style.FG, payload, err = takeCompactString(payload); err != nil {
				return nil, err
			}
			if run.Style.BG, payload, err = takeCompactString(payload); err != nil {
				return nil, err
			}
			if len(payload) == 0 {
				return nil, errors.New("truncated compact history style")
			}
			applyCompactStyleFlags(&run.Style, payload[0])
			payload = payload[1:]
			if run.LinkURL, payload, err = takeCompactString(payload); err != nil {
				return nil, err
			}
			if run.LinkParams, payload, err = takeCompactString(payload); err != nil {
				return nil, err
			}
			line.Runs = append(line.Runs, run)
		}
		lines = append(lines, line)
	}
	if len(payload) != 0 {
		return nil, errors.New("compact history block has trailing bytes")
	}
	return lines, nil
}

func appendCompactString(payload []byte, value string) []byte {
	payload = binary.AppendUvarint(payload, uint64(len(value)))
	return append(payload, value...)
}

func takeCompactString(payload []byte) (string, []byte, error) {
	length, size := binary.Uvarint(payload)
	if size <= 0 {
		return "", nil, errors.New("invalid compact history string length")
	}
	payload = payload[size:]
	if length > uint64(len(payload)) {
		return "", nil, errors.New("compact history string exceeds block")
	}
	return string(payload[:int(length)]), payload[int(length):], nil
}

func compactLinesEncodedSize(lines []Line) int {
	size := uvarintSize(uint64(len(lines)))
	for _, line := range lines {
		size += compactLineEncodedSize(line)
	}
	return size
}

func compactLineEncodedSize(line Line) int {
	size := 1 + varintSize(unixNanoOrZero(line.UpdatedAt)) + uvarintSize(uint64(len(line.Runs)))
	for _, run := range line.Runs {
		for _, value := range []string{run.Text, run.Style.FG, run.Style.BG, run.LinkURL, run.LinkParams} {
			size += uvarintSize(uint64(len(value))) + len(value)
		}
		size++
	}
	return size
}

func unixNanoOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixNano()
}

func varintSize(value int64) int {
	var buf [binary.MaxVarintLen64]byte
	return binary.PutVarint(buf[:], value)
}

func uvarintSize(value uint64) int {
	size := 1
	for value >= 0x80 {
		value >>= 7
		size++
	}
	return size
}

func compactStyleFlags(style history.CellStyle) uint8 {
	var flags uint8
	if style.Bold {
		flags |= 1 << 0
	}
	if style.Italic {
		flags |= 1 << 1
	}
	if style.Underline {
		flags |= 1 << 2
	}
	if style.Blink {
		flags |= 1 << 3
	}
	if style.Reverse {
		flags |= 1 << 4
	}
	if style.Strikethrough {
		flags |= 1 << 5
	}
	return flags
}

func applyCompactStyleFlags(style *history.CellStyle, flags uint8) {
	style.Bold = flags&(1<<0) != 0
	style.Italic = flags&(1<<1) != 0
	style.Underline = flags&(1<<2) != 0
	style.Blink = flags&(1<<3) != 0
	style.Reverse = flags&(1<<4) != 0
	style.Strikethrough = flags&(1<<5) != 0
}
