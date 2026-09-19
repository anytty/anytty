package keys

import (
	"strings"
	"unicode/utf8"
)

// Bracket paste markers (DEC mode 2004, PROTOCOL §6.8).
const (
	BracketStart = "\x1b[200~"
	BracketEnd   = "\x1b[201~"
)

// PasteChunk is one independently encoded paste block: the normalized text
// the program sees, the exact PTY bytes written for it and its own event id.
type PasteChunk struct {
	ID    string
	Text  string
	Bytes []byte
}

// NormalizeText folds CRLF and lone CR into LF. This is the canonical text
// carried by paste events; the PTY encoding turns LF back into CR.
func NormalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

// EncodePaste encodes one already-chunked paste text: newlines become
// carriage returns (the byte a terminal sends for Enter) and, when the
// terminal has bracketed paste on, the payload is wrapped in
// ESC[200~..ESC[201~ so multi-line text cannot execute as commands.
func EncodePaste(text string, bracket bool) []byte {
	payload := strings.ReplaceAll(text, "\n", "\r")
	if !bracket {
		return []byte(payload)
	}
	return []byte(BracketStart + payload + BracketEnd)
}

// ChunkPaste normalizes text, splits it on UTF-8 rune boundaries into chunks
// of at most maxBytes bytes and encodes each chunk exactly once. Every chunk
// is an independent event: ids come from nextID, order is preserved and
// chunks are never merged. maxBytes <= 0 means one chunk. Empty text yields
// no chunks.
func ChunkPaste(text string, maxBytes int, bracket bool, nextID func() string) []PasteChunk {
	text = NormalizeText(text)
	if text == "" {
		return nil
	}
	if maxBytes <= 0 || maxBytes > len(text) {
		maxBytes = len(text)
	}
	var chunks []PasteChunk
	for len(text) > 0 {
		n := maxBytes
		if n > len(text) {
			n = len(text)
		}
		for n > 0 && n < len(text) && !utf8.RuneStart(text[n]) {
			n--
		}
		if n == 0 {
			_, n = utf8.DecodeRuneInString(text)
		}
		chunk := text[:n]
		id := ""
		if nextID != nil {
			id = nextID()
		}
		chunks = append(chunks, PasteChunk{ID: id, Text: chunk, Bytes: EncodePaste(chunk, bracket)})
		text = text[n:]
	}
	return chunks
}
