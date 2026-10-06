package main

import (
	"strconv"
	"strings"
	"unicode"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
	"github.com/mozillazg/go-pinyin"
)

// Picker search follows main: a case-insensitive subsequence match over the
// row's searchable metadata ("trpl" hits "term-pool"), not a substring
// contains. On top of that, CJK text is additionally searchable by pinyin so a
// Chinese machine/tag/terminal name is reachable from a Latin keyboard: both
// the full pinyin ("suoping" -> 锁屏) and the initials ("sp" -> 锁屏).

// pinyinSearchText is one searchable projection of a candidate string plus a
// mapping from each search rune back to the display rune it came from (one CJK
// rune expands to several pinyin runes). The mapping lets a pinyin match
// highlight the originating CJK character.
type pinyinSearchText struct {
	text   string
	runeAt []int
}

// pickerSearchHaystack returns the searchable projections of one candidate:
// the raw lowercased text, its full pinyin and its pinyin initials.
func pickerSearchHaystack(value string) []pinyinSearchText {
	lower := []rune(strings.ToLower(value))
	raw := pinyinSearchText{text: string(lower)}
	for i := range lower {
		raw.runeAt = append(raw.runeAt, i)
	}

	var full, initials strings.Builder
	fullAt := make([]int, 0, len(lower))
	initAt := make([]int, 0, len(lower))
	args := pinyin.Args{Style: pinyin.Normal, Separator: "", Fallback: func(r rune, _ pinyin.Args) []string {
		return []string{strings.ToLower(string(r))}
	}}
	for index, r := range lower {
		if !unicode.Is(unicode.Han, r) {
			full.WriteRune(r)
			initials.WriteRune(r)
			fullAt = append(fullAt, index)
			initAt = append(initAt, index)
			continue
		}
		pinyinRune := string(r)
		if pys := pinyin.Pinyin(string(r), args); len(pys) > 0 && len(pys[0]) > 0 && pys[0][0] != "" {
			pinyinRune = pys[0][0]
		}
		full.WriteString(pinyinRune)
		for range []rune(pinyinRune) {
			fullAt = append(fullAt, index)
		}
		if initial := []rune(pinyinRune); len(initial) > 0 {
			initials.WriteRune(initial[0])
		} else {
			initials.WriteRune(r)
		}
		initAt = append(initAt, index)
	}
	return []pinyinSearchText{
		raw,
		{text: full.String(), runeAt: fullAt},
		{text: initials.String(), runeAt: initAt},
	}
}

// subsequenceIndexes is main's TerminalPickerQueryMatchIndexes: a greedy,
// case-insensitive subsequence match returning the matched rune positions, or
// nil when the query is not a subsequence.
func subsequenceIndexes(text, query string) []int {
	query = strings.TrimSpace(query)
	if query == "" {
		return []int{}
	}
	textRunes := []rune(strings.ToLower(text))
	queryRunes := []rune(strings.ToLower(query))
	matches := make([]int, 0, len(queryRunes))
	textAt := 0
	for _, queryRune := range queryRunes {
		found := false
		for textAt < len(textRunes) {
			if textRunes[textAt] == queryRune {
				matches = append(matches, textAt)
				textAt++
				found = true
				break
			}
			textAt++
		}
		if !found {
			return nil
		}
	}
	return matches
}

// matchSearchValue returns the display-rune indexes to highlight when query
// matches value (raw subsequence, full pinyin or pinyin initials), or nil. The
// raw match wins so an ASCII query highlights literal characters.
func matchSearchValue(value, query string) []int {
	query = strings.TrimSpace(query)
	if query == "" {
		return []int{}
	}
	haystack := pickerSearchHaystack(value)
	if positions := subsequenceIndexes(haystack[0].text, query); positions != nil {
		return positions
	}
	for _, candidate := range haystack[1:] {
		positions := subsequenceIndexes(candidate.text, query)
		if positions == nil {
			continue
		}
		seen := map[int]bool{}
		indexes := make([]int, 0, len(positions))
		for _, position := range positions {
			if position < 0 || position >= len(candidate.runeAt) {
				continue
			}
			runeIndex := candidate.runeAt[position]
			if seen[runeIndex] {
				continue
			}
			seen[runeIndex] = true
			indexes = append(indexes, runeIndex)
		}
		return indexes
	}
	return nil
}

// pickerSourceSearchFields mirrors main's matchesTerminalPickerQuery metadata
// set: title, terminal id, state, tags, attachment count ("xN") and size.
func pickerSourceSearchFields(src *pb.Source) []string {
	title := strings.TrimSpace(src.GetTitle())
	if title == "" {
		title = strings.TrimSpace(src.GetTerminalId())
	}
	fields := []string{title, src.GetTerminalId()}
	if src.GetExited() {
		fields = append(fields, "exited")
	} else {
		fields = append(fields, "running")
	}
	if labels := publicTagLabels(src); len(labels) > 0 {
		fields = append(fields, strings.Join(labels, ", "))
	}
	attachment := "x0"
	if src.GetAttached() {
		attachment = "x1"
	}
	fields = append(fields, attachment)
	if src.GetCols() > 0 && src.GetRows() > 0 {
		fields = append(fields, strconv.Itoa(int(src.GetCols()))+"x"+strconv.Itoa(int(src.GetRows())))
	}
	return fields
}

func pickerSourceMatches(src *pb.Source, query string) bool {
	if strings.TrimSpace(query) == "" {
		return true
	}
	for _, field := range pickerSourceSearchFields(src) {
		if matchSearchValue(field, query) != nil {
			return true
		}
	}
	return false
}
